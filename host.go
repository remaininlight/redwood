package redwood

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/brynbellomy/redwood/ctx"
	"github.com/brynbellomy/redwood/types"
)

type Host interface {
	ctx.Logger
	Ctx() *ctx.Context
	Start() error

	// Get(ctx context.Context, url string) (interface{}, error)
	Subscribe(ctx context.Context, stateURI string) error
	SendTx(ctx context.Context, tx Tx) error
	AddRef(reader io.ReadCloser) (types.Hash, types.Hash, error)
	FetchRef(ctx context.Context, ref types.RefID)
	AnnounceRefs(ctx context.Context, refIDs []types.RefID)
	AddPeer(ctx context.Context, transportName string, reachableAt StringSet) error
	Transport(name string) Transport
	Controllers() ControllerHub
	Address() types.Address
	ChallengePeerIdentity(ctx context.Context, peer Peer) (SigningPublicKey, EncryptingPublicKey, error)
	ForEachProviderOfStateURI(ctx context.Context, stateURI string) <-chan Peer

	HandleFetchHistoryRequest(stateURI string, parents []types.ID, toVersion types.ID, peer Peer) error
	HandleIncomingSubscription(stateURI string, peer Peer)
	HandleIncomingSubscriptionClosed(stateURI string, peer Peer)
	HandleTxReceived(tx Tx, peer Peer)
	HandlePrivateTxReceived(encryptedTx EncryptedTx, peer Peer)
	HandleAckReceived(txID types.ID, peer Peer)
	HandleChallengeIdentity(challengeMsg types.ChallengeMsg, peer Peer) error
	HandleFetchRefReceived(refID types.RefID, peer Peer)
}

type host struct {
	*ctx.Context

	config *Config

	ControllerHub
	transports        map[string]Transport
	signingKeypair    *SigningKeypair
	encryptingKeypair *EncryptingKeypair

	subscriptionsOut   map[string]*txMultiSub // map[stateURI]
	subscriptionsOutMu sync.RWMutex
	subscriptionsIn    map[string]map[Peer]struct{} // map[stateURI]
	subscriptionsInMu  sync.RWMutex
	peerSeenTxs        map[PeerDialInfo]map[types.ID]bool
	peerSeenTxsMu      sync.RWMutex

	peerStore PeerStore
	refStore  RefStore

	chRefsNeeded chan []types.RefID
}

var (
	ErrUnsignedTx = errors.New("unsigned tx")
	ErrProtocol   = errors.New("protocol error")
	ErrPeerIsSelf = errors.New("peer is self")
)

func NewHost(
	signingKeypair *SigningKeypair,
	encryptingKeypair *EncryptingKeypair,
	transports []Transport,
	controllerHub ControllerHub,
	refStore RefStore,
	peerStore PeerStore,
	config *Config,
) (Host, error) {
	transportsMap := make(map[string]Transport)
	for _, tpt := range transports {
		transportsMap[tpt.Name()] = tpt
	}
	h := &host{
		Context:           &ctx.Context{},
		transports:        transportsMap,
		ControllerHub:     controllerHub,
		signingKeypair:    signingKeypair,
		encryptingKeypair: encryptingKeypair,
		subscriptionsOut:  make(map[string]*txMultiSub),
		subscriptionsIn:   make(map[string]map[Peer]struct{}),
		peerSeenTxs:       make(map[PeerDialInfo]map[types.ID]bool),
		peerStore:         peerStore,
		refStore:          refStore,
		chRefsNeeded:      make(chan []types.RefID, 100),
		config:            config,
	}
	return h, nil
}

func (h *host) Ctx() *ctx.Context {
	return h.Context
}

func (h *host) Start() error {
	return h.CtxStart(
		// on startup
		func() error {
			h.SetLogLabel(h.Address().Pretty() + " host")

			// Set up the controller Hub
			h.ControllerHub.OnNewState(h.handleNewState)
			h.CtxAddChild(h.ControllerHub.Ctx(), nil)
			err := h.ControllerHub.Start()
			if err != nil {
				return err
			}

			// Set up the ref store
			h.refStore.OnRefsNeeded(h.handleRefsNeeded)

			// Set up the transports
			for _, transport := range h.transports {
				transport.SetHost(h)
				h.CtxAddChild(transport.Ctx(), nil)
				err := transport.Start()
				if err != nil {
					return err
				}
			}

			go h.periodicallyFetchMissingRefs()
			go h.periodicallyVerifyPeers()

			return nil
		},
		nil,
		nil,
		// on shutdown
		func() {},
	)
}

func (h *host) Transport(name string) Transport {
	return h.transports[name]
}

func (h *host) Controllers() ControllerHub {
	return h.ControllerHub
}

func (h *host) Address() types.Address {
	return h.signingKeypair.Address()
}

func (h *host) HandleTxReceived(tx Tx, peer Peer) {
	h.Infof(0, "tx %v received from %v peer %v", tx.ID.Pretty(), peer.Transport().Name(), peer.Address())
	h.markTxSeenByPeer(peer, tx.ID)

	have, err := h.ControllerHub.HaveTx(tx.StateURI, tx.ID)
	if err != nil {
		h.Errorf("error fetching tx %v from store: %v", tx.ID.Pretty(), err)
		// @@TODO: does it make sense to return here?
		return
	}

	if !have {
		err := h.ControllerHub.AddTx(&tx, false)
		if err != nil {
			h.Errorf("error adding tx to controllerHub: %v", err)
		}
	}

	err = peer.Ack(tx.ID)
	if err != nil {
		h.Errorf("error ACKing peer: %v", err)
	}
}

func (h *host) ForEachProviderOfStateURI(ctx context.Context, stateURI string) <-chan Peer {
	var wg sync.WaitGroup
	ch := make(chan Peer)
	for _, tpt := range h.transports {
		innerCh, err := tpt.ForEachProviderOfStateURI(ctx, stateURI)
		if err != nil {
			h.Warnf("transport %v could not fetch providers of state URI '%v'", tpt.Name(), stateURI)
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-h.Ctx().Done():
					return
				case <-ctx.Done():
					return
				case peer, open := <-innerCh:
					if !open {
						return
					}

					ch <- peer
				}
			}
		}()

	}

	go func() {
		defer close(ch)
		wg.Wait()
	}()

	return ch
}

func (h *host) ForEachProviderOfRef(ctx context.Context, refID types.RefID) <-chan Peer {
	var wg sync.WaitGroup
	ch := make(chan Peer)
	for _, tpt := range h.transports {
		innerCh, err := tpt.ForEachProviderOfRef(ctx, refID)
		if err != nil {
			h.Warnf("transport %v could not fetch providers of ref %v", tpt.Name(), refID)
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-h.Ctx().Done():
					return
				case <-ctx.Done():
					return
				case peer, open := <-innerCh:
					if !open {
						return
					}

					ch <- peer
				}
			}
		}()
	}

	go func() {
		defer close(ch)
		wg.Wait()
	}()

	return ch
}

func (h *host) HandlePrivateTxReceived(encryptedTx EncryptedTx, peer Peer) {
	h.Infof(0, "private tx %v received", encryptedTx.TxID.Pretty())
	h.markTxSeenByPeer(peer, encryptedTx.TxID)

	bs, err := h.encryptingKeypair.OpenMessageFrom(EncryptingPublicKeyFromBytes(encryptedTx.SenderPublicKey), encryptedTx.EncryptedPayload)
	if err != nil {
		h.Errorf("error decrypting tx: %v", err)
		return
	}

	var tx Tx
	err = json.Unmarshal(bs, &tx)
	if err != nil {
		h.Errorf("error decoding tx: %v", err)
		return
	}

	if encryptedTx.TxID != tx.ID {
		h.Errorf("private tx id does not match")
		return
	}

	have, err := h.ControllerHub.HaveTx(tx.StateURI, tx.ID)
	if err != nil {
		h.Errorf("error fetching tx %v from store: %v", tx.ID.Pretty(), err)
		return
	}

	if !have {
		// Add to controllerHub
		err := h.ControllerHub.AddTx(&tx, false)
		if err != nil {
			h.Errorf("error adding tx to controllerHub: %v", err)
		}
	}

	err = peer.Ack(tx.ID)
	if err != nil {
		h.Errorf("error ACKing peer: %v", err)
	}
}

func (h *host) HandleAckReceived(txID types.ID, peer Peer) {
	h.Infof(0, "ack received for %v", txID.Hex())
	h.markTxSeenByPeer(peer, txID)
}

func (h *host) markTxSeenByPeer(peer Peer, txID types.ID) {
	h.peerSeenTxsMu.Lock()
	defer h.peerSeenTxsMu.Unlock()

	for _, tuple := range peer.DialInfos() {
		if h.peerSeenTxs[tuple] == nil {
			h.peerSeenTxs[tuple] = make(map[types.ID]bool)
		}
		h.peerSeenTxs[tuple][txID] = true
	}
}

func (h *host) txSeenByPeer(peer Peer, txID types.ID) bool {
	if peer.Address() == (types.Address{}) {
		return false
	}

	// @@TODO: convert to LRU cache
	h.peerSeenTxsMu.Lock()
	defer h.peerSeenTxsMu.Unlock()

	for _, tuple := range peer.DialInfos() {
		if h.peerSeenTxs[tuple] == nil {
			continue
		}
		if h.peerSeenTxs[tuple][txID] {
			return true
		}
	}
	return false
}

func (h *host) AddPeer(ctx context.Context, transportName string, reachableAt StringSet) error {
	transport := h.Transport(transportName)
	if transport == nil {
		h.peerStore.AddReachableAddresses(transportName, reachableAt)
		return nil
	}

	peer, err := transport.GetPeerByConnStrings(ctx, reachableAt)
	if err != nil {
		return err
	}

	err = peer.EnsureConnected(ctx)
	if err != nil {
		return err
	}

	h.peerStore.AddReachableAddresses(transportName, reachableAt)

	sigpubkey, _, err := h.ChallengePeerIdentity(ctx, peer)
	if err != nil {
		return err
	}

	h.Infof(0, "added peer %v at %v %v", sigpubkey.Address(), transportName, reachableAt.Slice())
	return nil
}

func (h *host) periodicallyVerifyPeers() {
	for {
		maybePeers := h.peerStore.MaybePeers()
		var wg sync.WaitGroup
		wg.Add(len(maybePeers))
		for _, maybePeer := range maybePeers {
			maybePeer := maybePeer
			go func() {
				defer wg.Done()

				transport := h.Transport(maybePeer.TransportName())
				if transport == nil {
					// Unsupported transport
					return
				}

				peer, err := transport.GetPeerByConnStrings(context.TODO(), maybePeer.ReachableAt())
				if errors.Cause(err) == ErrPeerIsSelf {
					return
				} else if errors.Cause(err) == types.ErrConnection {
					return
				} else if err != nil {
					h.Warn("could not get peer at %v %v: ", maybePeer.TransportName(), maybePeer.ReachableAt())
					return
				}

				_, _, err = h.ChallengePeerIdentity(context.TODO(), peer)
				if err != nil {
					h.Errorf("error verifying peer identity: %v ", err)
					return
				}
			}()
		}
		wg.Wait()

		time.Sleep(3 * time.Second)
	}
}

func (h *host) HandleFetchHistoryRequest(stateURI string, parents []types.ID, toVersion types.ID, peer Peer) error {
	// @@TODO: respect the input params

	iter := h.ControllerHub.FetchTxs(stateURI)
	defer iter.Cancel()

	for {
		tx := iter.Next()
		if iter.Error() != nil {
			return iter.Error()
		} else if tx == nil {
			return nil
		}

		err := peer.Put(*tx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *host) HandleIncomingSubscription(stateURI string, peer Peer) {
	h.subscriptionsInMu.Lock()
	defer h.subscriptionsInMu.Unlock()

	if _, exists := h.subscriptionsIn[stateURI]; !exists {
		h.subscriptionsIn[stateURI] = make(map[Peer]struct{})
	}

	h.subscriptionsIn[stateURI][peer] = struct{}{}
}

func (h *host) HandleIncomingSubscriptionClosed(stateURI string, peer Peer) {
	h.subscriptionsInMu.Lock()
	defer h.subscriptionsInMu.Unlock()

	if _, exists := h.subscriptionsIn[stateURI]; exists {
		delete(h.subscriptionsIn[stateURI], peer)
	}
}

func (h *host) Subscribe(ctx context.Context, stateURI string) error {
	h.subscriptionsOutMu.Lock()
	defer h.subscriptionsOutMu.Unlock()

	h.config.Update(func() error {
		set := NewStringSet(h.config.Node.SubscribedStateURIs)
		set.Add(stateURI)
		h.config.Node.SubscribedStateURIs = set.Slice()
		return nil
	})

	if _, exists := h.subscriptionsOut[stateURI]; exists {
		return errors.New("already subscribed to that state URI")
	}

	multiSub := newTxMultiSub(stateURI, 4, h, h.peerStore)
	multiSub.Start()
	h.subscriptionsOut[stateURI] = multiSub

	go func() {
		select {
		case <-multiSub.chStop:
			h.subscriptionsOutMu.Lock()
			defer h.subscriptionsOutMu.Unlock()
			delete(h.subscriptionsOut, stateURI)
		}
	}()

	return nil
}

func (h *host) ChallengePeerIdentity(ctx context.Context, peer Peer) (_ SigningPublicKey, _ EncryptingPublicKey, err error) {
	defer withStack(&err)

	err = peer.EnsureConnected(ctx)
	if err != nil {
		return nil, nil, err
	}

	challengeMsg, err := types.GenerateChallengeMsg()
	if err != nil {
		return nil, nil, err
	}

	err = peer.ChallengeIdentity(types.ChallengeMsg(challengeMsg))
	if err != nil {
		return nil, nil, err
	}

	resp, err := peer.ReceiveChallengeIdentityResponse()
	if err != nil {
		return nil, nil, err
	}

	sigpubkey, err := RecoverSigningPubkey(types.HashBytes(challengeMsg), resp.Signature)
	if err != nil {
		return nil, nil, err
	}
	encpubkey := EncryptingPublicKeyFromBytes(resp.EncryptingPublicKey)

	h.peerStore.AddVerifiedCredentials(peer.Transport().Name(), peer.ReachableAt(), sigpubkey.Address(), sigpubkey, encpubkey)

	return sigpubkey, encpubkey, nil
}

func (h *host) HandleChallengeIdentity(challengeMsg types.ChallengeMsg, peer Peer) error {
	defer peer.CloseConn()

	sig, err := h.signingKeypair.SignHash(types.HashBytes(challengeMsg))
	if err != nil {
		return err
	}
	return peer.RespondChallengeIdentity(ChallengeIdentityResponse{
		Signature:           sig,
		EncryptingPublicKey: h.encryptingKeypair.EncryptingPublicKey.Bytes(),
	})
}

type peersWithAddressResult struct {
	Peer
	EncryptingPublicKey
}

func (h *host) peersWithAddress(ctx context.Context, address types.Address) (<-chan peersWithAddressResult, error) {
	if address == h.Address() {
		return nil, errors.WithStack(ErrPeerIsSelf)
	}

	ch := make(chan peersWithAddressResult)
	go func() {
		defer close(ch)

		var alreadySent sync.Map

		if storedPeers := h.peerStore.PeersWithAddress(address); len(storedPeers) > 0 {
			for _, storedPeer := range storedPeers {
				transport, exists := h.transports[storedPeer.TransportName()]
				if !exists {
					h.Warnf("transport '%v' for no longer exists", storedPeer.TransportName())
					continue
				}

				peer, err := transport.GetPeerByConnStrings(ctx, storedPeer.ReachableAt())
				if err != nil {
					h.Errorf("error calling transport.GetPeer: %v", err)
					continue
				}
				_, encpubkey := storedPeer.PublicKeypairs()
				ch <- peersWithAddressResult{peer, encpubkey}
				for _, tuple := range storedPeer.DialInfos() {
					alreadySent.Store(tuple, struct{}{})
				}
			}
		}

		var transportsWg sync.WaitGroup
		for _, transport := range h.transports {

			transportsWg.Add(1)
			transport := transport
			go func() {
				defer transportsWg.Done()

				ctx, cancel := context.WithCancel(ctx)
				defer cancel()
				chPeers, err := transport.PeersClaimingAddress(ctx, address)
				if err != nil {
					h.Errorf("error fetching peers with address %v from transport %v", address.Hex(), transport.Name())
					return
				}

				var peersWg sync.WaitGroup
			PeerLoop:
				for peer := range chPeers {
					for _, tuple := range peer.DialInfos() {
						if _, sent := alreadySent.Load(tuple); sent {
							continue PeerLoop
						}
					}

					peersWg.Add(1)
					peer := peer
					go func() {
						defer peersWg.Done()

						err = peer.EnsureConnected(context.TODO())
						if err != nil {
							h.Errorf("error ensuring peer is connected: %v", err)
							return
						}
						defer peer.CloseConn()

						signingPubkey, encryptingPubkey, err := h.ChallengePeerIdentity(ctx, peer)
						if err != nil {
							h.Errorf("error requesting peer credentials: %v", err)
							return
						} else if signingPubkey.Address() != address {
							h.Errorf("peer sent invalid signature")
							return
						}

						for _, tuple := range peer.DialInfos() {
							alreadySent.Store(tuple, struct{}{})
						}
						ch <- peersWithAddressResult{peer, encryptingPubkey}
					}()
				}
				peersWg.Wait()
			}()
		}

		transportsWg.Wait()
	}()
	return ch, nil
}

func (h *host) broadcastPrivateTxToRecipient(ctx context.Context, txID types.ID, marshalledTx []byte, recipientAddr types.Address) error {
	chPeers, err := h.peersWithAddress(ctx, recipientAddr)
	if err != nil {
		return err
	}

	var anySucceeded bool
	var wg sync.WaitGroup
	for p := range chPeers {
		wg.Add(1)

		p := p
		go func() {
			defer wg.Done()

			err = p.Peer.EnsureConnected(context.TODO())
			if err != nil {
				return
			}
			defer p.Peer.CloseConn()

			msgEncrypted, err := h.encryptingKeypair.SealMessageFor(p.EncryptingPublicKey, marshalledTx)
			if err != nil {
				return
			}

			err = p.Peer.PutPrivate(EncryptedTx{
				TxID:             txID,
				EncryptedPayload: msgEncrypted,
				SenderPublicKey:  h.encryptingKeypair.EncryptingPublicKey.Bytes(),
			})
			if err != nil {
				return
			}
			// @@TODO: wait for ack?
			anySucceeded = true
		}()
	}
	wg.Wait()

	if !anySucceeded {
		return errors.Errorf("could not reach recipient %v", recipientAddr.Hex())
	}
	return nil
}

func (h *host) broadcastTx(ctx context.Context, tx *Tx) error {
	if len(tx.Sig) == 0 {
		return errors.WithStack(ErrUnsignedTx)
	}

	if tx.IsPrivate() {
		marshalledTx, err := json.Marshal(tx)
		if err != nil {
			return errors.WithStack(err)
		}

		var wg sync.WaitGroup
		for _, recipientAddr := range tx.Recipients {
			if recipientAddr == h.Address() {
				continue
			}

			wg.Add(1)
			go func() {
				defer wg.Done()

				err := h.broadcastPrivateTxToRecipient(ctx, tx.ID, marshalledTx, recipientAddr)
				if err != nil {
					h.Errorf(err.Error())
				}
			}()
		}
		wg.Wait()

	} else {
		var wg sync.WaitGroup
		h.broadcastTxToSubscribedPeers(ctx, tx, &wg)
		h.broadcastTxToStateURIProviders(ctx, tx, &wg)
		wg.Wait()
	}
	return nil
}

func (h *host) broadcastTxToStateURIProviders(ctx context.Context, tx *Tx, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second) // @@TODO: make configurable
	defer cancel()

	ch := make(chan Peer)
	go func() {
		defer close(ch)

		chProviders := h.ForEachProviderOfStateURI(ctx, tx.StateURI)
		for {
			select {
			case peer, open := <-chProviders:
				if !open {
					chProviders = nil
					continue
				}
				ch <- peer
			case <-ctx.Done():
				return
			case <-h.Ctx().Done():
				return
			}
		}
	}()

	for peer := range ch {
		h.Debugf("rebroadcasting %v to %v", tx.ID.Pretty(), peer.ReachableAt().Slice())
		if h.txSeenByPeer(peer, tx.ID) || tx.From == peer.Address() { // @@TODO: do we always want to avoid broadcasting when `from == peer.address`?
			h.Debugf("tx already seen by peer %v %v", peer.Transport().Name(), peer.Address())
			continue
		}
		h.Debugf("tx %v NOT already seen by peer: %v %v %v", tx.ID.Pretty(), peer.Transport().Name(), peer.Address(), peer.ReachableAt().Slice())

		wg.Add(1)
		peer := peer
		go func() {
			defer wg.Done()
			defer peer.CloseConn()

			err := peer.EnsureConnected(ctx)
			if err != nil {
				h.Errorf("error connecting to peer: %v", err)
				return
			}

			err = peer.Put(*tx)
			if err != nil {
				h.Errorf("error writing tx to peer: %v", err)
				return
			}
		}()
	}
}

func (h *host) broadcastTxToSubscribedPeers(ctx context.Context, tx *Tx, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	h.subscriptionsInMu.RLock()
	defer h.subscriptionsInMu.RUnlock()

	for peer := range h.subscriptionsIn[tx.StateURI] {
		h.Debugf("rebroadcasting %v to %v", tx.ID.Pretty(), peer.ReachableAt().Slice())
		if h.txSeenByPeer(peer, tx.ID) {
			h.Debugf("tx already seen by peer %v %v", peer.Transport().Name(), peer.Address())
			continue
		}
		h.Debugf("tx %v NOT already seen by peer: %v %v %v", tx.ID.Pretty(), peer.Transport().Name(), peer.Address(), peer.ReachableAt().Slice())

		wg.Add(1)
		peer := peer
		go func() {
			defer wg.Done()

			err := peer.EnsureConnected(ctx)
			if err != nil {
				h.Errorf("error connecting to peer: %v", err)
				h.HandleIncomingSubscriptionClosed(tx.StateURI, peer)
				return
			}

			err = peer.Put(*tx)
			if err != nil {
				h.Errorf("error writing tx to peer: %v", err)
				h.HandleIncomingSubscriptionClosed(tx.StateURI, peer)
				return
			}
		}()
	}
}

func (h *host) SendTx(ctx context.Context, tx Tx) error {
	h.Info(0, "adding tx ", tx.ID.Pretty())

	if tx.From == (types.Address{}) {
		tx.From = h.signingKeypair.Address()
	}

	if len(tx.Parents) == 0 && tx.ID != GenesisTxID {
		parents, err := h.ControllerHub.Leaves(tx.StateURI)
		if err != nil {
			return err
		}

		tx.Parents = parents
	}

	if len(tx.Sig) == 0 {
		err := h.SignTx(&tx)
		if err != nil {
			return err
		}
	}

	err := h.ControllerHub.AddTx(&tx, false)
	if err != nil {
		return err
	}
	return nil
}

func (h *host) SignTx(tx *Tx) error {
	var err error
	tx.Sig, err = h.signingKeypair.SignHash(tx.Hash())
	return err
}

func (h *host) handleNewState(tx *Tx) {
	// @@TODO: broadcast state to state subscribers

	// @@TODO: don't do this, this is stupid.  store ungossiped txs in the DB and create a
	// PeerManager that gossips them on a SleeperTask-like trigger.
	go func() {
		// Broadcast tx to others
		ctx, cancel := context.WithTimeout(h.Ctx(), 10*time.Second)
		defer cancel()

		err := h.broadcastTx(ctx, tx)
		if err != nil {
			h.Errorf("error rebroadcasting tx: %v", err)
		}
	}()
}

func (h *host) AddRef(reader io.ReadCloser) (types.Hash, types.Hash, error) {
	return h.refStore.StoreObject(reader)
}

func (h *host) handleRefsNeeded(refs []types.RefID) {
	select {
	case <-h.Ctx().Done():
		return
	case h.chRefsNeeded <- refs:
	}
}

func (h *host) periodicallyFetchMissingRefs() {
	tick := time.NewTicker(10 * time.Second) // @@TODO: make configurable
	defer tick.Stop()

	for {
		select {
		case <-h.Ctx().Done():
			return

		case refs := <-h.chRefsNeeded:
			h.fetchMissingRefs(refs)

		case <-tick.C:
			refs, err := h.refStore.RefsNeeded()
			if err != nil {
				h.Errorf("error fetching list of needed refs: %v", err)
				continue
			}

			if len(refs) > 0 {
				h.fetchMissingRefs(refs)
			}
		}
	}
}

func (h *host) fetchMissingRefs(refs []types.RefID) {
	var wg sync.WaitGroup
	for _, refID := range refs {
		wg.Add(1)
		refID := refID
		go func() {
			defer wg.Done()
			h.FetchRef(h.Ctx(), refID)
		}()
	}
	wg.Wait()
}

func (h *host) FetchRef(ctx context.Context, refID types.RefID) {
	for peer := range h.ForEachProviderOfRef(ctx, refID) {
		err := peer.EnsureConnected(ctx)
		if err != nil {
			h.Errorf("error connecting to peer: %v", err)
			continue
		}

		err = peer.FetchRef(refID)
		if err != nil {
			h.Errorf("error writing to peer: %v", err)
			continue
		}

		// Not currently used
		_, err = peer.ReceiveRefHeader()
		if err != nil {
			h.Errorf("error reading from peer: %v", err)
			continue
		}

		pr, pw := io.Pipe()
		go func() {
			var err error
			defer func() { pw.CloseWithError(err) }()

			for {
				select {
				case <-ctx.Done():
					err = ctx.Err()
					return
				default:
				}

				pkt, err := peer.ReceiveRefPacket()
				if err != nil {
					h.Errorf("error receiving ref from peer: %v", err)
					return
				} else if pkt.End {
					return
				}

				var n int
				n, err = pw.Write(pkt.Data)
				if err != nil {
					h.Errorf("error receiving ref from peer: %v", err)
					return
				} else if n < len(pkt.Data) {
					err = io.ErrUnexpectedEOF
					return
				}
			}
		}()

		sha1Hash, sha3Hash, err := h.refStore.StoreObject(pr)
		if err != nil {
			h.Errorf("could not store ref: %v", err)
			continue
		}
		// @@TODO: check stored refHash against the one we requested

		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		h.AnnounceRefs(ctx, []types.RefID{
			{HashAlg: types.SHA1, Hash: sha1Hash},
			{HashAlg: types.SHA3, Hash: sha3Hash},
		})
		return
	}
}

func (h *host) AnnounceRefs(ctx context.Context, refIDs []types.RefID) {
	var wg sync.WaitGroup
	wg.Add(len(refIDs) * len(h.transports))

	for _, transport := range h.transports {
		for _, refID := range refIDs {
			transport := transport
			refID := refID

			go func() {
				defer wg.Done()

				err := transport.AnnounceRef(ctx, refID)
				if errors.Cause(err) == types.ErrUnimplemented {
					return
				} else if err != nil {
					h.Warnf("error announcing ref %v over transport %v: %v", refID, transport.Name(), err)
				}
			}()
		}
	}
	wg.Wait()
}

const (
	REF_CHUNK_SIZE = 1024 // @@TODO: tunable buffer size?
)

func (h *host) HandleFetchRefReceived(refID types.RefID, peer Peer) {
	defer peer.CloseConn()

	objectReader, _, err := h.refStore.Object(refID)
	// @@TODO: handle the case where we don't have the ref more gracefully
	if err != nil {
		panic(err)
	}

	err = peer.SendRefHeader()
	if err != nil {
		h.Errorf("[ref server] %+v", errors.WithStack(err))
		return
	}

	buf := make([]byte, REF_CHUNK_SIZE)
	for {
		n, err := io.ReadFull(objectReader, buf)
		if err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF {
			buf = buf[:n]
		} else if err != nil {
			h.Errorf("[ref server] %+v", err)
			return
		}

		err = peer.SendRefPacket(buf, false)
		if err != nil {
			h.Errorf("[ref server] %+v", errors.WithStack(err))
			return
		}
	}

	err = peer.SendRefPacket(nil, true)
	if err != nil {
		h.Errorf("[ref server] %+v", errors.WithStack(err))
		return
	}
}
