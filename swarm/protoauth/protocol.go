package protoauth

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"redwood.dev/crypto"
	"redwood.dev/identity"
	"redwood.dev/log"
	"redwood.dev/process"
	"redwood.dev/swarm"
	"redwood.dev/types"
	"redwood.dev/utils"
)

//go:generate mockery --name AuthProtocol --output ./mocks/ --case=underscore
type AuthProtocol interface {
	process.Interface
	ChallengePeerIdentity(ctx context.Context, peerConn AuthPeerConn) (err error)
}

//go:generate mockery --name AuthTransport --output ./mocks/ --case=underscore
type AuthTransport interface {
	swarm.Transport
	OnChallengeIdentity(handler ChallengeIdentityCallback)
}

//go:generate mockery --name AuthPeerConn --output ./mocks/ --case=underscore
type AuthPeerConn interface {
	swarm.PeerConn
	ChallengeIdentity(challengeMsg ChallengeMsg) error
	RespondChallengeIdentity(verifyAddressResponse []ChallengeIdentityResponse) error
	ReceiveChallengeIdentityResponse() ([]ChallengeIdentityResponse, error)
}

type authProtocol struct {
	process.Process
	log.Logger

	keyStore   identity.KeyStore
	peerStore  swarm.PeerStore
	transports map[string]AuthTransport
	poolWorker process.PoolWorker
}

func NewAuthProtocol(transports []swarm.Transport, keyStore identity.KeyStore, peerStore swarm.PeerStore) *authProtocol {
	transportsMap := make(map[string]AuthTransport)
	for _, tpt := range transports {
		if tpt, is := tpt.(AuthTransport); is {
			transportsMap[tpt.Name()] = tpt
		}
	}
	return &authProtocol{
		Process:    *process.New("AuthProtocol"),
		Logger:     log.NewLogger(ProtocolName),
		transports: transportsMap,
		keyStore:   keyStore,
		peerStore:  peerStore,
	}
}

const ProtocolName = "protoauth"

func (ap *authProtocol) Name() string {
	return ProtocolName
}

func (ap *authProtocol) Start() error {
	err := ap.Process.Start()
	if err != nil {
		return err
	}

	ap.poolWorker = process.NewPoolWorker("pool worker", 4, process.NewStaticScheduler(5*time.Second, 10*time.Second))
	err = ap.Process.SpawnChild(nil, ap.poolWorker)
	if err != nil {
		return err
	}
	for _, pd := range ap.peerStore.UnverifiedPeers() {
		ap.poolWorker.Add(verifyPeer{pd.DialInfo(), ap})
	}

	announcePeersTask := NewAnnouncePeersTask(10*time.Second, ap, ap.peerStore, ap.transports)
	ap.peerStore.OnNewUnverifiedPeer(func(dialInfo swarm.PeerDialInfo) {
		announcePeersTask.Enqueue()
		ap.poolWorker.Add(verifyPeer{dialInfo, ap})
	})
	err = ap.Process.SpawnChild(nil, announcePeersTask)
	if err != nil {
		return err
	}

	for _, tpt := range ap.transports {
		ap.Infof(0, "registering %v", tpt.Name())
		tpt.OnChallengeIdentity(ap.handleChallengeIdentity)
	}
	return nil
}

func (ap *authProtocol) ChallengePeerIdentity(ctx context.Context, peerConn AuthPeerConn) (err error) {
	defer utils.WithStack(&err)

	if !peerConn.Ready() || !peerConn.Dialable() {
		return errors.Wrapf(swarm.ErrUnreachable, "peer: %v", peerConn.DialInfo())
	}

	err = peerConn.EnsureConnected(ctx)
	if err != nil {
		return err
	}

	challengeMsg, err := GenerateChallengeMsg()
	if err != nil {
		return err
	}

	err = peerConn.ChallengeIdentity(challengeMsg)
	if err != nil {
		return err
	}

	resp, err := peerConn.ReceiveChallengeIdentityResponse()
	if err != nil {
		return err
	}

	for _, proof := range resp {
		sigpubkey, err := crypto.RecoverSigningPubkey(types.HashBytes(challengeMsg), proof.Signature)
		if err != nil {
			return err
		}
		encpubkey := crypto.AsymEncPubkeyFromBytes(proof.AsymEncPubkey)

		ap.peerStore.AddVerifiedCredentials(peerConn.DialInfo(), peerConn.DeviceSpecificID(), sigpubkey.Address(), sigpubkey, encpubkey)
	}

	return nil
}

func (ap *authProtocol) handleChallengeIdentity(challengeMsg ChallengeMsg, peerConn AuthPeerConn) error {
	defer peerConn.Close()

	publicIdentities, err := ap.keyStore.PublicIdentities()
	if err != nil {
		ap.Errorf("error fetching public identities from key store: %v", err)
		return err
	}

	var responses []ChallengeIdentityResponse
	for _, identity := range publicIdentities {
		sig, err := ap.keyStore.SignHash(identity.Address(), types.HashBytes(challengeMsg))
		if err != nil {
			ap.Errorf("error signing hash: %v", err)
			return err
		}
		responses = append(responses, ChallengeIdentityResponse{
			Signature:     sig,
			AsymEncPubkey: identity.AsymEncKeypair.AsymEncPubkey.Bytes(),
		})
	}
	return peerConn.RespondChallengeIdentity(responses)
}

type announcePeersTask struct {
	process.PeriodicTask
	log.Logger
	authProto  AuthProtocol
	peerStore  swarm.PeerStore
	transports map[string]AuthTransport
}

func NewAnnouncePeersTask(
	interval time.Duration,
	authProto AuthProtocol,
	peerStore swarm.PeerStore,
	transports map[string]AuthTransport,
) *announcePeersTask {
	t := &announcePeersTask{
		Logger:     log.NewLogger(ProtocolName),
		authProto:  authProto,
		peerStore:  peerStore,
		transports: transports,
	}
	t.PeriodicTask = *process.NewPeriodicTask("AnnouncePeersTask", interval, t.announcePeers)
	return t
}

func (t *announcePeersTask) announcePeers(ctx context.Context) {
	// Announce peers
	{
		allDialInfos := t.peerStore.AllDialInfos()

		for _, tpt := range t.transports {
			for _, peerDetails := range t.peerStore.PeersFromTransport(tpt.Name()) {
				if !peerDetails.Ready() || !peerDetails.Dialable() {
					continue
				} else if peerDetails.DialInfo().TransportName != tpt.Name() {
					continue
				}

				tpt := tpt
				peerDetails := peerDetails

				t.Process.Go(nil, "announce peers", func(ctx context.Context) {
					peerConn, err := tpt.NewPeerConn(ctx, peerDetails.DialInfo().DialAddr)
					if errors.Cause(err) == swarm.ErrPeerIsSelf {
						return
					} else if err != nil {
						t.Warnf("error creating new %v peerConn: %v", tpt.Name(), err)
						return
					}
					defer peerConn.Close()

					err = peerConn.EnsureConnected(ctx)
					if errors.Cause(err) == types.ErrConnection {
						return
					} else if err != nil {
						t.Warnf("error connecting to %v peerConn (%v): %v", tpt.Name(), peerDetails.DialInfo().DialAddr, err)
						return
					}

					peerConn.AnnouncePeers(ctx, allDialInfos)
					if err != nil {
						// t.Errorf("error writing to peerConn: %+v", err)
					}
				})
			}
		}
	}
}

type verifyPeer struct {
	dialInfo  swarm.PeerDialInfo
	authProto *authProtocol
}

var _ process.PoolWorkerItem = verifyPeer{}

func (t verifyPeer) BlacklistUniqueID() process.PoolUniqueID    { return t }
func (t verifyPeer) RetryUniqueID() process.PoolUniqueID        { return t }
func (t verifyPeer) DedupeActiveUniqueID() process.PoolUniqueID { return t }
func (t verifyPeer) ID() string                                 { return "" }

func (t verifyPeer) Work(ctx context.Context) (retry bool) {
	unverifiedPeer := t.authProto.peerStore.PeerWithDialInfo(t.dialInfo)
	if unverifiedPeer == nil {
		return true
	}

	if !unverifiedPeer.Ready() {
		return true
	} else if !unverifiedPeer.Dialable() {
		return false
	}

	transport, exists := t.authProto.transports[unverifiedPeer.DialInfo().TransportName]
	if !exists {
		// Unsupported transport
		return false
	}

	peerConn, err := transport.NewPeerConn(ctx, unverifiedPeer.DialInfo().DialAddr)
	if errors.Cause(err) == swarm.ErrPeerIsSelf {
		return false
	} else if errors.Cause(err) == types.ErrConnection {
		return true
	} else if err != nil {
		return true
	}

	authPeerConn, is := peerConn.(AuthPeerConn)
	if !is {
		return false
	}
	defer authPeerConn.Close()

	err = t.authProto.ChallengePeerIdentity(ctx, authPeerConn)
	if errors.Cause(err) == types.ErrConnection {
		// no-op
		return true
	} else if errors.Cause(err) == context.Canceled {
		// no-op
		return true
	} else if err != nil {
		t.authProto.Errorf("error verifying peerConn identity (%v): %v", authPeerConn.DialInfo(), err)
		return true
	}
	t.authProto.Successf("authenticated with %v (addresses=%v)", t.dialInfo, authPeerConn.Addresses())
	return false
}
