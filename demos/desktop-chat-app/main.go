// +build !headless

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/brynbellomy/klog"
	"github.com/markbates/pkger"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"github.com/webview/webview"

	rw "redwood.dev"
	"redwood.dev/ctx"
	"redwood.dev/tree"
)

var app = struct {
	ctx.Context
}{}

func main() {
	cliApp := cli.NewApp()
	// cliApp.Version = env.AppVersion

	configPath, err := rw.DefaultConfigPath("redwood-webview")
	if err != nil {
		app.Error(err)
		os.Exit(1)
	}

	cliApp.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config",
			Value: configPath,
			Usage: "location of config file",
		},
		cli.UintFlag{
			Name:  "port",
			Value: 54231,
			Usage: "port on which to serve UI assets",
		},
		cli.BoolFlag{
			Name:  "pprof",
			Usage: "enable pprof",
		},
		cli.BoolFlag{
			Name:  "dev",
			Usage: "enable dev mode",
		},
	}

	cliApp.Action = func(c *cli.Context) error {
		configPath := c.String("config")
		enablePprof := c.Bool("pprof")
		dev := c.Bool("dev")
		port := c.Uint("port")
		return run(configPath, enablePprof, dev, port)
	}

	err = cliApp.Run(os.Args)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}

func run(configPath string, enablePprof bool, dev bool, port uint) error {
	if enablePprof {
		go func() {
			http.ListenAndServe(":6060", nil)
		}()
		runtime.SetBlockProfileRate(int(time.Millisecond.Nanoseconds()) * 100)
	}

	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	klog.InitFlags(flagset)
	flagset.Set("logtostderr", "true")
	flagset.Set("v", "2")
	klog.SetFormatter(&klog.FmtConstWidth{
		FileNameCharWidth: 24,
		UseColor:          true,
	})

	klog.Flush()
	defer klog.Flush()

	config, err := rw.ReadConfigAtPath("redwood-webview", configPath)
	if os.IsNotExist(err) {
		err := os.MkdirAll(filepath.Dir(configPath), 0777|os.ModeDir)
		if err != nil {
			return err
		}
		config, err = rw.ReadConfigAtPath("redwood-webview", configPath)
		if err != nil {
			return err
		}

	} else if err != nil {
		return err
	}

	if dev {
		config.Node.DevMode = true
	}

	err = ensureDataDirs(config)
	if err != nil {
		return err
	}

	signingKeypair, err := rw.SigningKeypairFromHDMnemonic(config.Node.HDMnemonicPhrase, rw.DefaultHDDerivationPath)
	if err != nil {
		return err
	}

	encryptingKeypair, err := rw.GenerateEncryptingKeypair()
	if err != nil {
		return err
	}

	var (
		txStore       = rw.NewBadgerTxStore(config.TxDBRoot())
		refStore      = rw.NewRefStore(config.RefDataRoot())
		peerStore     = rw.NewPeerStore()
		controllerHub = rw.NewControllerHub(config.StateDBRoot(), txStore, refStore)
	)

	err = refStore.Start()
	if err != nil {
		return err
	}
	app.CtxAddChild(refStore.Ctx(), nil)

	err = txStore.Start()
	if err != nil {
		return err
	}
	app.CtxAddChild(txStore.Ctx(), nil)

	var transports []rw.Transport

	if config.P2PTransport.Enabled {
		libp2pTransport, err := rw.NewLibp2pTransport(
			signingKeypair.Address(),
			config.P2PTransport.ListenPort,
			config.P2PTransport.ReachableAt,
			config.P2PTransport.KeyFile,
			encryptingKeypair,
			controllerHub,
			refStore,
			peerStore,
		)
		if err != nil {
			return err
		}
		transports = append(transports, libp2pTransport)
	}

	if config.HTTPTransport.Enabled {
		tlsCertFilename := filepath.Join(config.Node.DataRoot, "..", "server.crt")
		tlsKeyFilename := filepath.Join(config.Node.DataRoot, "..", "server.key")

		var cookieSecret [32]byte
		copy(cookieSecret[:], []byte(config.HTTPTransport.CookieSecret))

		httpTransport, err := rw.NewHTTPTransport(
			config.HTTPTransport.ListenHost,
			config.HTTPTransport.ReachableAt,
			config.HTTPTransport.DefaultStateURI,
			controllerHub,
			refStore,
			peerStore,
			signingKeypair,
			encryptingKeypair,
			cookieSecret,
			tlsCertFilename,
			tlsKeyFilename,
			config.Node.DevMode,
		)
		if err != nil {
			return err
		}
		transports = append(transports, httpTransport)
	}

	host, err := rw.NewHost(signingKeypair, encryptingKeypair, transports, controllerHub, refStore, peerStore, config)
	if err != nil {
		return err
	}

	err = host.Start()
	if err != nil {
		panic(err)
	}

	if config.HTTPRPC.Enabled {
		rwRPC := rw.NewHTTPRPCServer(signingKeypair.Address(), host)
		httpRPC := &HTTPRPCServer{rwRPC, signingKeypair}

		err = rw.StartHTTPRPC(httpRPC, config.HTTPRPC)
		if err != nil {
			return err
		}
		app.CtxAddChild(httpRPC.Ctx(), nil)
	}

	for _, bootstrapPeer := range config.Node.BootstrapPeers {
		bootstrapPeer := bootstrapPeer
		go func() {
			app.Infof(0, "connecting to bootstrap peer: %v %v", bootstrapPeer.Transport, bootstrapPeer.DialAddresses)
			for _, dialAddr := range bootstrapPeer.DialAddresses {
				host.AddPeer(rw.PeerDialInfo{TransportName: bootstrapPeer.Transport, DialAddr: dialAddr})
			}
		}()
	}

	go func() {
		time.Sleep(5 * time.Second)
		for stateURI := range config.Node.SubscribedStateURIs {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			sub, err := host.Subscribe(ctx, stateURI, 0, nil, nil)
			if err != nil {
				app.Errorf("error subscribing to %v: %v", stateURI, err)
				continue
			}
			sub.Close()
			app.Successf("subscribed to %v", stateURI)
		}
	}()

	app.CtxAddChild(host.Ctx(), nil)
	app.CtxStart(
		func() error { return nil },
		nil,
		nil,
		nil,
	)
	defer app.CtxStop("shutdown", nil)

	klog.Info(rw.PrettyJSON(config))

	initializeLocalState(host)

	go startGUI(port)
	go startAPI(host, port)
	go inputLoop(host)
	app.AttachInterruptHandler()
	app.CtxWait()

	return nil
}

func initializeLocalState(host rw.Host) {
	// pkger.Include("./sync9.js")
	f, err := pkger.Open("/sync9.js")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_, _, err = host.AddRef(f)
	if err != nil {
		panic(err)
	}

	state, err := host.StateAtVersion("chat.local/servers", nil)
	if err != nil && errors.Cause(err) != rw.ErrNoController {
		panic(err)
	} else if err == nil {
		defer state.Close()

		exists, err := state.Exists(tree.Keypath("value"))
		if err != nil {
			panic(err)
		}
		if exists {
			return
		}
		state.Close()
	}

	type M = map[string]interface{}

	err = host.SendTx(context.Background(), rw.Tx{
		StateURI: "chat.local/servers",
		ID:       rw.GenesisTxID,
		Patches: []rw.Patch{{
			Val: M{
				"Merge-Type": M{
					"Content-Type": "resolver/dumb",
					"value":        M{},
				},
				"Validator": M{
					"Content-Type": "validator/permissions",
					"value": M{
						host.Address().Hex(): M{
							"^.*$": M{
								"write": true,
							},
						},
					},
				},
				"value": []interface{}{},
			},
		}},
	})
	if err != nil {
		panic(err)
	}
}

func startGUI(port uint) {
	debug := true
	w := webview.New(debug)
	defer w.Destroy()
	w.SetTitle("Minimal webview example")
	w.SetSize(800, 600, webview.HintNone)
	w.Navigate(fmt.Sprintf("http://localhost:%v/index.html", port))
	w.Run()
}

func ensureDataDirs(config *rw.Config) error {
	err := os.MkdirAll(config.RefDataRoot(), 0777|os.ModeDir)
	if err != nil {
		return err
	}

	err = os.MkdirAll(config.TxDBRoot(), 0777|os.ModeDir)
	if err != nil {
		return err
	}

	err = os.MkdirAll(config.StateDBRoot(), 0777|os.ModeDir)
	if err != nil {
		return err
	}
	return nil
}

func inputLoop(host rw.Host) {
	fmt.Println("Type \"help\" for a list of commands.")
	fmt.Println()

	var longestCommandLength int
	for cmd := range replCommands {
		if len(cmd) > longestCommandLength {
			longestCommandLength = len(cmd)
		}
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("> ")

		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		parts := strings.Split(line, " ")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}

		if len(parts) < 1 {
			app.Error("enter a command")
			continue
		} else if parts[0] == "help" {
			fmt.Println("___ Commands _________")
			fmt.Println()
			for cmd, info := range replCommands {
				difference := longestCommandLength - len(cmd)
				space := strings.Repeat(" ", difference+4)
				fmt.Printf("%v%v- %v\n", cmd, space, info.HelpText)
			}
			continue
		}

		cmd, exists := replCommands[parts[0]]
		if !exists {
			app.Error("unknown command")
			continue
		}

		err := cmd.Handler(app.Ctx(), parts[1:], host)
		if err != nil {
			app.Error(err)
		}
	}
}

var replCommands = map[string]struct {
	HelpText string
	Handler  func(ctx context.Context, args []string, host rw.Host) error
}{
	"stateuris": {
		"list all known state URIs",
		func(ctx context.Context, args []string, host rw.Host) error {
			stateURIs, err := host.Controllers().KnownStateURIs()
			if err != nil {
				return err
			}
			if len(stateURIs) == 0 {
				fmt.Println("no known state URIs")
			} else {
				for _, stateURI := range stateURIs {
					fmt.Println("- ", stateURI)
				}
			}
			return nil
		},
	},
	"state": {
		"print the current state tree",
		func(ctx context.Context, args []string, host rw.Host) error {
			if len(args) < 1 {
				return errors.New("missing argument: state URI")
			}

			stateURI := args[0]
			state, err := host.Controllers().StateAtVersion(stateURI, nil)
			if err != nil {
				return err
			}
			var keypath tree.Keypath
			var rng *tree.Range
			if len(args) > 1 {
				_, keypath, rng, err = rw.ParsePatchPath([]byte(args[1]))
				if err != nil {
					return err
				}
			}
			app.Debugf("stateURI: %v / keypath: %v / range: %v", stateURI, keypath, rng)
			state = state.NodeAt(keypath, rng)
			state.DebugPrint(app.Debugf, false, 0)
			fmt.Println(rw.PrettyJSON(state))
			return nil
		},
	},
	"peers": {
		"list all known peers",
		func(ctx context.Context, args []string, host rw.Host) error {
			for _, peer := range host.Peers() {
				fmt.Println("- ", peer.Address(), peer.DialInfo(), peer.LastContact())
			}
			return nil
		},
	},
}
