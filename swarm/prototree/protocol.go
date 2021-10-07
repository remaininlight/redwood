package prototree

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"redwood.dev/identity"
	"redwood.dev/log"
	"redwood.dev/process"
	"redwood.dev/state"
	"redwood.dev/swarm"
	"redwood.dev/swarm/protohush"
	"redwood.dev/tree"
	"redwood.dev/types"
)

//go:generate mockery --name TreeProtocol --output ./mocks/ --case=underscore
type TreeProtocol interface {
	process.Interface
	ProvidersOfStateURI(ctx context.Context, stateURI string) <-chan TreePeerConn
	Subscribe(ctx context.Context, stateURI string, subscriptionType SubscriptionType, keypath state.Keypath, fetchHistoryOpts *FetchHistoryOpts) (ReadableSubscription, error)
	Unsubscribe(stateURI string) error
	SubscribeStateURIs() (StateURISubscription, error)
	SendTx(ctx context.Context, tx tree.Tx) error
}

//go:generate mockery --name TreeTransport --output ./mocks/ --case=underscore
type TreeTransport interface {
	swarm.Transport
	ProvidersOfStateURI(ctx context.Context, stateURI string) (<-chan TreePeerConn, error)
	OnTxReceived(handler TxReceivedCallback)
	OnPrivateTxReceived(handler PrivateTxReceivedCallback)
	OnAckReceived(handler AckReceivedCallback)
	OnWritableSubscriptionOpened(handler WritableSubscriptionOpenedCallback)
	OnP2PStateURIReceived(handler P2PStateURIReceivedCallback)
}

//go:generate mockery --name TreePeerConn --output ./mocks/ --case=underscore
type TreePeerConn interface {
	swarm.PeerConn
	Subscribe(ctx context.Context, stateURI string) (ReadableSubscription, error)
	SendTx(ctx context.Context, tx tree.Tx) error
	SendPrivateTx(ctx context.Context, encryptedTx EncryptedTx) (err error)
	Ack(stateURI string, txID types.ID) error
	AnnounceP2PStateURI(ctx context.Context, stateURI string) error
}

type treeProtocol struct {
	process.Process
	log.Logger

	store Store

	transports    map[string]TreeTransport
	controllerHub tree.ControllerHub
	txStore       tree.TxStore
	keyStore      identity.KeyStore
	peerStore     swarm.PeerStore

	hushProto protohush.HushProtocol

	acl ACL

	readableSubscriptions   map[string]*multiReaderSubscription // map[stateURI]
	readableSubscriptionsMu sync.RWMutex
	writableSubscriptions   map[string]map[WritableSubscription]struct{} // map[stateURI]
	writableSubscriptionsMu sync.RWMutex

	// broadcastTxsToStateURIProvidersTask *broadcastTxsToStateURIProvidersTask
	announceP2PStateURIsTask *announceP2PStateURIsTask
}

var (
	_ TreeProtocol      = (*treeProtocol)(nil)
	_ process.Interface = (*treeProtocol)(nil)
)

func NewTreeProtocol(
	transports []swarm.Transport,
	hushProto protohush.HushProtocol,
	controllerHub tree.ControllerHub,
	txStore tree.TxStore,
	keyStore identity.KeyStore,
	peerStore swarm.PeerStore,
	store Store,
) *treeProtocol {
	transportsMap := make(map[string]TreeTransport)
	for _, tpt := range transports {
		if tpt, is := tpt.(TreeTransport); is {
			transportsMap[tpt.Name()] = tpt
		}
	}
	tp := &treeProtocol{
		Process:       *process.New(ProtocolName),
		Logger:        log.NewLogger(ProtocolName),
		hushProto:     hushProto,
		store:         store,
		transports:    transportsMap,
		controllerHub: controllerHub,
		txStore:       txStore,
		keyStore:      keyStore,
		peerStore:     peerStore,

		acl: DefaultACL{ControllerHub: controllerHub},

		readableSubscriptions: make(map[string]*multiReaderSubscription),
		writableSubscriptions: make(map[string]map[WritableSubscription]struct{}),
	}
	// tp.broadcastTxsToStateURIProvidersTask = NewBroadcastTxsToStateURIProvidersTask(10*time.Second, store, peerStore, transportsMap)
	tp.announceP2PStateURIsTask = NewAnnounceP2PStateURIsTask(10*time.Second, tp)
	return tp
}

const ProtocolName = "prototree"

func (tp *treeProtocol) Name() string {
	return ProtocolName
}

func (tp *treeProtocol) Start() error {
	err := tp.Process.Start()
	if err != nil {
		return err
	}

	tp.controllerHub.OnNewState(tp.handleNewState)
	tp.hushProto.OnGroupMessageEncrypted(ProtocolName, tp.handlePrivateTxEncrypted)
	tp.hushProto.OnGroupMessageDecrypted(ProtocolName, tp.handlePrivateTxDecrypted)

	for _, tpt := range tp.transports {
		tp.Infof(0, "registering %v", tpt.Name())
		tpt.OnTxReceived(tp.handleTxReceived)
		tpt.OnAckReceived(tp.handleAckReceived)
		tpt.OnWritableSubscriptionOpened(tp.handleWritableSubscriptionOpened)
		tpt.OnP2PStateURIReceived(tp.handleP2PStateURIReceived)
	}

	tp.Process.Go(nil, "initial subscribe", func(ctx context.Context) {
		for _, stateURI := range tp.store.SubscribedStateURIs().Slice() {
			tp.Infof(0, "subscribing to %v", stateURI)
			sub, err := tp.Subscribe(ctx, stateURI, SubscriptionType_Txs, nil, nil)
			if err != nil {
				tp.Errorf("error subscribing to %v: %v", stateURI, err)
				continue
			}
			sub.Close()
		}
	})

	// err = tp.Process.SpawnChild(nil, tp.broadcastTxsToStateURIProvidersTask)
	// if err != nil {
	// 	return err
	// }
	err = tp.Process.SpawnChild(nil, tp.announceP2PStateURIsTask)
	if err != nil {
		return err
	}
	return nil
}

func (tp *treeProtocol) SendTx(ctx context.Context, tx tree.Tx) (err error) {
	tp.Infof(0, "adding tx (%v) %v", tx.StateURI, tx.ID.Pretty())

	defer func() {
		if err != nil {
			return
		}
		// If we send a tx to a state URI that we're not subscribed to yet, auto-subscribe.
		if !tp.store.SubscribedStateURIs().Contains(tx.StateURI) {
			err := tp.store.AddSubscribedStateURI(tx.StateURI)
			if err != nil {
				tp.Errorf("error adding %v to config store SubscribedStateURIs: %v", tx.StateURI, err)
			}
		}
	}()

	if tx.From.IsZero() {
		publicIdentities, err := tp.keyStore.PublicIdentities()
		if err != nil {
			return err
		} else if len(publicIdentities) == 0 {
			return errors.New("keystore has no public identities")
		}
		tx.From = publicIdentities[0].Address()
	}

	if len(tx.Parents) == 0 && tx.ID != tree.GenesisTxID {
		var parents []types.ID
		parents, err = tp.controllerHub.Leaves(tx.StateURI)
		if err != nil {
			return err
		}
		tx.Parents = parents
	}

	if len(tx.Sig) == 0 {
		tx.Sig, err = tp.keyStore.SignHash(tx.From, tx.Hash())
		if err != nil {
			return err
		}
	}

	switch tp.acl.TypeOf(tx.StateURI) {
	case StateURIType_Invalid:
		return errors.Errorf("invalid state URI: %v", tx.StateURI)

	case StateURIType_Private:
		err = tp.controllerHub.AddTx(tx)
		if err != nil {
			return err
		}
		// Broadcasting happens in `tp.handleNewState` to ensure that we have
		// the state tree's `.Members` field

	case StateURIType_Public, StateURIType_DeviceLocal:
		err = tp.controllerHub.AddTx(tx)
		if err != nil {
			return err
		}
		tp.broadcastToWritableSubscribers(ctx, tx.StateURI, &tx, nil, nil, nil)
	}
	return nil
}

func (tp *treeProtocol) hushMessageIDForTx(tx tree.Tx) string {
	return tx.StateURI + ":" + tx.ID.Hex()
}

func (tp *treeProtocol) parseHushMessageID(id string) (string, types.ID, error) {
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		return "", types.ID{}, errors.Errorf("bad hush message ID for tx: %v", id)
	}
	txID, err := types.IDFromHex(parts[1])
	if err != nil {
		return "", types.ID{}, err
	}
	return parts[0], txID, nil
}

func (tp *treeProtocol) handleTxReceived(tx tree.Tx, peerConn TreePeerConn) {
	tp.Infof(0, "tx received: tx=%v peer=%v", tx.ID.Pretty(), peerConn.DialInfo())
	tp.store.MarkTxSeenByPeer(peerConn.DeviceSpecificID(), tx.StateURI, tx.ID)

	exists, err := tp.txStore.TxExists(tx.StateURI, tx.ID)
	if err != nil {
		tp.Errorf("error fetching tx %v from store: %v", tx.ID.Pretty(), err)
		// @@TODO: does it make sense to return here?
		return
	}

	if !exists {
		err := tp.controllerHub.AddTx(tx)
		if err != nil {
			tp.Errorf("error adding tx to controllerHub: %v", err)
		}
	}

	// The ACK happens in a separate stream
	peerConn2, err := peerConn.Transport().NewPeerConn(context.TODO(), peerConn.DialInfo().DialAddr)
	if err != nil {
		tp.Errorf("error ACKing peer: %v", err)
	}
	defer peerConn2.Close()
	err = peerConn2.(TreePeerConn).Ack(tx.StateURI, tx.ID)
	if err != nil {
		tp.Errorf("error ACKing peer: %v", err)
	}
}

func (tp *treeProtocol) handlePrivateTxReceived(encryptedTx EncryptedTx, peerConn TreePeerConn) {
	tp.Infof(0, "private tx received: tx=%v peer=%v", encryptedTx.ID, peerConn.DialInfo())

	err := tp.hushProto.DecryptGroupMessage(encryptedTx)
	if err != nil {
		tp.Errorf("while submitting private tx for decryption: %v", err)
		return
	}

	// @@TODO: ack?
}

func (tp *treeProtocol) handlePrivateTxEncrypted(encryptedTx protohush.GroupMessage) {
	tp.Infof(0, "broadcasting private tx %v", encryptedTx.ID)

	stateURI, txID, err := tp.parseHushMessageID(encryptedTx.ID)
	if err != nil {
		tp.Errorf("while parsing hush message ID: %v", err)
		return
	}

	tx, err := tp.txStore.FetchTx(stateURI, txID)
	if err != nil {
		tp.Errorf("while fetching tx %v %v from tx store: %v", stateURI, txID, err)
		return
	}

	err = tp.store.SaveEncryptedTx(tx.StateURI, tx.ID, encryptedTx)
	if err != nil {
		tp.Errorf("while saving encrypted tx to database: %v", err)
		return
	}

	tp.broadcastToWritableSubscribers(context.TODO(), stateURI, &tx, &encryptedTx, nil, nil)
	// @@TODO: send to Vault
}

func (tp *treeProtocol) handlePrivateTxDecrypted(sender types.Address, plaintext []byte, encryptedTx protohush.GroupMessage) {
	var tx tree.Tx
	err := tx.Unmarshal(plaintext)
	if err != nil {
		tp.Errorf("while unmarshaling tx protobuf: %v", err)
		return
	}

	tp.Infof(0, "private tx received (stateURI=%v id=%v sender=%v)", tx.StateURI, tx.ID, sender)

	err = tp.store.SaveEncryptedTx(tx.StateURI, tx.ID, encryptedTx)
	if err != nil {
		tp.Errorf("while saving encrypted tx to database: %v", err)
		return
	}

	err = tp.controllerHub.AddTx(tx)
	if err != nil {
		tp.Errorf("while adding private tx to controller: %v", err)
		return
	}

	// tp.store.MarkTxSeenByPeer(peerConn.DeviceSpecificID(), tx.StateURI, tx.ID)

	// // The ACK happens in a separate stream
	// peerConn2, err := peerConn.Transport().NewPeerConn(context.TODO(), peerConn.DialInfo().DialAddr)
	// if err != nil {
	//  tp.Errorf("error ACKing peer: %v", err)
	// }
	// defer peerConn2.Close()
	// err = peerConn2.(TreePeerConn).Ack(tx.StateURI, tx.ID)
	// if err != nil {
	//  tp.Errorf("error ACKing peer: %v", err)
	// }

	tp.broadcastToWritableSubscribers(context.TODO(), tx.StateURI, &tx, &encryptedTx, nil, nil)
	// @@TODO: send to Vault
}

func (tp *treeProtocol) handleAckReceived(stateURI string, txID types.ID, peerConn TreePeerConn) {
	tp.Infof(0, "ack received: tx=%v peer=%v", txID.Hex(), peerConn.DialInfo().DialAddr)
	tp.store.MarkTxSeenByPeer(peerConn.DeviceSpecificID(), stateURI, txID)
}

func (tp *treeProtocol) handleP2PStateURIReceived(stateURI string, peerConn TreePeerConn) {
	// tp.Infof(0, "p2p state URI received: stateURI=%v peer=%v", stateURI, peerConn.DialInfo().DialAddr)

	peerConn.AddStateURI(stateURI)

	err := tp.subscribe(context.TODO(), stateURI)
	if err != nil {
		tp.Errorf("while subscribing to p2p state URI %v: %v", stateURI, err)
	}
}

type FetchHistoryOpts struct {
	FromTxID types.ID
	ToTxID   types.ID
}

func (tp *treeProtocol) handleFetchHistoryRequest(stateURI string, opts FetchHistoryOpts, writeSub WritableSubscription) error {
	// @@TODO: respect the `opts.ToTxID` param
	// @@TODO: if .FromTxID == 0, set it to GenesisTxID

	allowed, err := tp.acl.HasReadAccess(stateURI, nil, writeSub.Addresses())
	if err != nil {
		return errors.Wrapf(err, "while querying ACL for read access (stateURI=%v)", stateURI)
	} else if !allowed {
		return types.Err403
	}

	isPrivate := tp.acl.TypeOf(stateURI) == StateURIType_Private

	iter := tp.controllerHub.FetchTxs(stateURI, opts.FromTxID)
	defer iter.Close()

	for {
		tx := iter.Next()
		if iter.Error() != nil {
			return iter.Error()
		} else if tx == nil {
			break
		}

		var encryptedTx *EncryptedTx
		if isPrivate {
			encryptedTx2, err := tp.store.EncryptedTx(stateURI, tx.ID)
			if err != nil {
				return err
			}
			encryptedTx = &encryptedTx2
		}

		leaves, err := tp.controllerHub.Leaves(stateURI)
		if err != nil {
			return err
		}

		if isPrivate {
			tx = nil
		} else {
			encryptedTx = nil
		}
		msg := SubscriptionMsg{
			StateURI:    stateURI,
			Tx:          tx,
			EncryptedTx: encryptedTx,
			State:       nil,
			Leaves:      leaves,
		}
		writeSub.EnqueueWrite(msg)
	}
	return nil
}

func (tp *treeProtocol) handleWritableSubscriptionOpened(
	req SubscriptionRequest,
	writeSubImplFactory WritableSubscriptionImplFactory,
) (<-chan struct{}, error) {
	allowed, err := tp.acl.HasReadAccess(req.StateURI, req.Keypath, req.Addresses)
	if err != nil {
		return nil, errors.Wrapf(err, "while querying ACL for read access (stateURI=%v)", req.StateURI)
	} else if !allowed {
		tp.Debugf("blocked incoming subscription from %v %+v", req.Addresses, errors.New(""))
		return nil, types.Err403
	}

	isPrivate := tp.acl.TypeOf(req.StateURI) == StateURIType_Private

	writeSubImpl, err := writeSubImplFactory()
	if err != nil {
		return nil, err
	}

	writeSub := newWritableSubscription(req.StateURI, req.Keypath, req.Type, isPrivate, req.Addresses, writeSubImpl)
	err = tp.Process.SpawnChild(nil, writeSub)
	if err != nil {
		tp.Errorf("while spawning writable subscription: %v", err)
		return nil, err
	}

	func() {
		tp.writableSubscriptionsMu.Lock()
		defer tp.writableSubscriptionsMu.Unlock()

		if _, exists := tp.writableSubscriptions[req.StateURI]; !exists {
			tp.writableSubscriptions[req.StateURI] = make(map[WritableSubscription]struct{})
		}
		tp.writableSubscriptions[req.StateURI][writeSub] = struct{}{}
	}()

	tp.Process.Go(nil, "await close "+writeSub.String(), func(ctx context.Context) {
		select {
		case <-writeSub.Done():
		case <-ctx.Done():
		}
		tp.handleWritableSubscriptionClosed(writeSub)
	})

	if req.Type.Includes(SubscriptionType_Txs) && req.FetchHistoryOpts != nil {
		tp.handleFetchHistoryRequest(req.StateURI, *req.FetchHistoryOpts, writeSub)
	}

	if req.Type.Includes(SubscriptionType_States) {
		// Normalize empty keypaths
		if req.Keypath.Equals(state.KeypathSeparator) {
			req.Keypath = nil
		}

		// Immediately write the current state to the subscriber
		node, err := tp.controllerHub.StateAtVersion(req.StateURI, nil)
		if err != nil && errors.Cause(err) != tree.ErrNoController {
			tp.Errorf("error writing initial state to peer: %v", err)
			writeSub.Close()
			return nil, err

		} else if err == nil {
			defer node.Close()

			leaves, err := tp.controllerHub.Leaves(req.StateURI)
			if err != nil {
				tp.Errorf("error writing initial state to peer (%v): %v", req.StateURI, err)
			} else {
				node, err := node.CopyToMemory(req.Keypath, nil)
				if err != nil && errors.Cause(err) == types.Err404 {
					// no-op
				} else if err != nil {
					tp.Errorf("error writing initial state to peer (%v): %v", req.StateURI, err)
				} else {
					writeSub.EnqueueWrite(SubscriptionMsg{
						StateURI:    req.StateURI,
						Tx:          nil,
						EncryptedTx: nil,
						State:       node,
						Leaves:      leaves,
					})
				}
			}
		}
	}
	return writeSub.Done(), nil
}

func (tp *treeProtocol) handleWritableSubscriptionClosed(sub WritableSubscription) {
	tp.writableSubscriptionsMu.Lock()
	defer tp.writableSubscriptionsMu.Unlock()
	delete(tp.writableSubscriptions[sub.StateURI()], sub)
}

func (tp *treeProtocol) subscribe(ctx context.Context, stateURI string) error {
	treeType := tp.acl.TypeOf(stateURI)
	if treeType == StateURIType_Invalid {
		return errors.Errorf("invalid state URI: %v", stateURI)
	}

	err := tp.store.AddSubscribedStateURI(stateURI)
	if err != nil {
		return errors.Wrap(err, "while updating config store")
	}

	_, err = tp.controllerHub.EnsureController(stateURI)
	if err != nil {
		return err
	}

	switch treeType {
	case StateURIType_Invalid:
	case StateURIType_DeviceLocal:
	case StateURIType_Private:
		tp.openReadableSubscription(stateURI)
	case StateURIType_Public:
		tp.openReadableSubscription(stateURI)
	}
	return nil
}

func (tp *treeProtocol) openReadableSubscription(stateURI string) {
	tp.readableSubscriptionsMu.Lock()
	defer tp.readableSubscriptionsMu.Unlock()

	if _, exists := tp.readableSubscriptions[stateURI]; !exists {
		tp.Debugf("opening subscription to %v", stateURI)
		multiSub := newMultiReaderSubscription(
			stateURI,
			tp.store.MaxPeersPerSubscription(),
			func(msg SubscriptionMsg, peerConn TreePeerConn) {
				if msg.EncryptedTx != nil {
					tp.handlePrivateTxReceived(*msg.EncryptedTx, peerConn)
				} else if msg.Tx != nil {
					tp.handleTxReceived(*msg.Tx, peerConn)
				} else {
					panic("wat")
				}
			},
			tp.ProvidersOfStateURI,
		)
		tp.Process.SpawnChild(nil, multiSub)
		tp.readableSubscriptions[stateURI] = multiSub
	}
}

func (tp *treeProtocol) Subscribe(
	ctx context.Context,
	stateURI string,
	subscriptionType SubscriptionType,
	keypath state.Keypath,
	fetchHistoryOpts *FetchHistoryOpts,
) (ReadableSubscription, error) {
	err := tp.subscribe(ctx, stateURI)
	if err != nil {
		return nil, err
	}

	// Open the subscription with the node's own credentials
	identities, err := tp.keyStore.Identities()
	if err != nil {
		return nil, err
	}
	var addrs []types.Address
	for _, identity := range identities {
		addrs = append(addrs, identity.Address())
	}

	sub := newInProcessSubscription(stateURI, keypath, subscriptionType, tp)
	req := SubscriptionRequest{
		StateURI:         stateURI,
		Keypath:          keypath,
		Type:             subscriptionType,
		FetchHistoryOpts: fetchHistoryOpts,
		Addresses:        addrs,
	}
	tp.handleWritableSubscriptionOpened(req, func() (WritableSubscriptionImpl, error) {
		return sub, nil
	})
	return sub, nil
}

func (tp *treeProtocol) Unsubscribe(stateURI string) error {
	// @@TODO: when we unsubscribe, we should close the subs of any peers reading from us
	func() {
		tp.readableSubscriptionsMu.Lock()
		defer tp.readableSubscriptionsMu.Unlock()

		if sub, exists := tp.readableSubscriptions[stateURI]; exists {
			sub.Close()
			delete(tp.readableSubscriptions, stateURI)
		}
	}()

	err := tp.store.RemoveSubscribedStateURI(stateURI)
	if err != nil {
		return errors.Wrap(err, "while updating config")
	}
	return nil
}

func (tp *treeProtocol) SubscribeStateURIs() (StateURISubscription, error) {
	sub := newStateURISubscription(tp.store)
	err := tp.SpawnChild(nil, sub)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// Returns peers discovered through any transport who claim to provide the
// stateURI in question.
func (tp *treeProtocol) ProvidersOfStateURI(ctx context.Context, stateURI string) <-chan TreePeerConn {
	var (
		ch          = make(chan TreePeerConn)
		alreadySent sync.Map
	)

	switch tp.acl.TypeOf(stateURI) {
	case StateURIType_Invalid, StateURIType_DeviceLocal:
		close(ch)
		return ch

	case StateURIType_Private:
		members, err := tp.acl.MembersOf(stateURI)
		if err != nil {
			tp.Errorf("while fetching members of state URI '%v': %v", stateURI, err)
			close(ch)
			return ch
		}
		pds := tp.peerStore.PeersServingStateURI(stateURI)
		for addr := range members {
			pds = append(pds, tp.peerStore.PeersWithAddress(addr)...)
		}
		tp.Process.Go(nil, "ProvidersOfStateURI "+stateURI, func(ctx context.Context) {
			peerConns := tp.peerDetailsToPeerConns(ctx, pds)
			for _, peerConn := range peerConns {
				if _, exists := alreadySent.LoadOrStore(peerConn.DialInfo(), struct{}{}); exists {
					continue
				}
				select {
				case <-ctx.Done():
					return
				case ch <- peerConn:
				}
			}
		})

	case StateURIType_Public:
		child := tp.Process.NewChild(ctx, "ProvidersOfStateURI "+stateURI)
		defer child.AutocloseWithCleanup(func() {
			close(ch)
		})

		child.Go(nil, "from PeerStore", func(ctx context.Context) {
			peerConns := tp.peerDetailsToPeerConns(ctx, tp.peerStore.PeersServingStateURI(stateURI))
			for _, peerConn := range peerConns {
				if _, exists := alreadySent.LoadOrStore(peerConn.DialInfo(), struct{}{}); exists {
					continue
				}
				select {
				case <-ctx.Done():
					return
				case ch <- peerConn:
				}
			}
		})

		for _, tpt := range tp.transports {
			innerCh, err := tpt.ProvidersOfStateURI(ctx, stateURI)
			if err != nil {
				tp.Warnf("error fetching providers of State-URI %v on transport %v: %v %+v", stateURI, tpt.Name(), err)
				continue
			}

			child.Go(nil, tpt.Name(), func(ctx context.Context) {
				for {
					select {
					case <-ctx.Done():
						return
					case peer, open := <-innerCh:
						if !open {
							return
						}
						peer.AddStateURI(stateURI)

						if _, exists := alreadySent.LoadOrStore(peer.DialInfo(), struct{}{}); exists {
							continue
						}

						select {
						case <-ctx.Done():
							return
						case ch <- peer:
						}
					}
				}
			})
		}
	}

	return ch
}

func (tp *treeProtocol) peerDetailsToPeerConns(ctx context.Context, pds []swarm.PeerDetails) []TreePeerConn {
	var conns []TreePeerConn
	for _, peerDetails := range pds {
		dialInfo := peerDetails.DialInfo()
		tpt, exists := tp.transports[dialInfo.TransportName]
		if !exists {
			continue
		}
		peerConn, err := tpt.NewPeerConn(ctx, dialInfo.DialAddr)
		if err != nil {
			continue
		}
		treePeerConn, is := peerConn.(TreePeerConn)
		if !is {
			continue
		}
		conns = append(conns, treePeerConn)
	}
	return conns
}

func (tp *treeProtocol) handleNewState(tx tree.Tx, node state.Node, leaves []types.ID) {
	node, err := node.CopyToMemory(nil, nil)
	if err != nil {
		tp.Errorf("handleNewState: couldn't copy state to memory: %v", err)
		node = state.NewMemoryNode() // give subscribers an empty state
	}

	// @@TODO: don't do this, this is stupid.  store ungossiped txs in the DB and create a
	// service that gossips them on a SleeperTask-like trigger.

	// Broadcast state and tx to others
	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	// child := tp.Process.NewChild(ctx, "handleNewState")
	// defer child.AutocloseWithCleanup(cancel)

	switch tp.acl.TypeOf(tx.StateURI) {
	case StateURIType_Invalid:
		panic("invariant violation")

	case StateURIType_Private:
		// If this is the genesis tx of a private state URI, ensure that we subscribe to that state URI
		// @@TODO: allow blacklisting of senders
		if tx.ID == tree.GenesisTxID && !tp.store.SubscribedStateURIs().Contains(tx.StateURI) {
			tp.Process.Go(nil, "auto-subscribe", func(ctx context.Context) {
				ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()

				sub, err := tp.Subscribe(ctx, tx.StateURI, 0, nil, nil)
				if err != nil {
					tp.Errorf("error subscribing to state URI %v: %v", tx.StateURI, err)
				}
				sub.Close() // We don't need the in-process subscription
			})
		}

		members, err := tp.acl.MembersOf(tx.StateURI)
		if err != nil {
			tp.Errorf("while fetching members of %v: %v", tx.StateURI, err)
			return
		}
		txBytes, err := tx.Marshal()
		if err != nil {
			tp.Errorf("while marshaling tx %v %v: %v", tx.StateURI, tx.ID, err)
			return
		}
		err = tp.hushProto.EncryptGroupMessage(ProtocolName, tp.hushMessageIDForTx(tx), members.Slice(), txBytes)
		if err != nil {
			tp.Errorf("while enqueuing hush tx %v %v: %v", tx.StateURI, tx.ID, err)
			return
		}

		// tp.broadcastToPrivateRecipients(tx)

	case StateURIType_Public, StateURIType_DeviceLocal:
		// @@TODO: handle "device local" separately
		// tp.broadcastTxsToStateURIProvidersTask.addTx(tx)
		// child.Go(nil, "broadcastToWritableSubscribers", func(ctx context.Context) {
		// 	tp.broadcastToWritableSubscribers(ctx, tx, nil, node, leaves, &alreadySentPeers, child)
		// })
	}
}

func (tp *treeProtocol) broadcastToWritableSubscribers(
	ctx context.Context,
	stateURI string,
	tx *tree.Tx,
	encryptedTx *EncryptedTx,
	state state.Node,
	leaves []types.ID,
) {
	tp.writableSubscriptionsMu.RLock()
	defer tp.writableSubscriptionsMu.RUnlock()

	isPrivate := tp.acl.TypeOf(tx.StateURI) == StateURIType_Private

	for writeSub := range tp.writableSubscriptions[tx.StateURI] {
		allowed, err := tp.acl.HasReadAccess(tx.StateURI, nil, writeSub.Addresses())
		if err != nil {
			tp.Errorf("while checking ACL of state URI %v", tx.StateURI)
			continue
		} else if !allowed {
			// @@TODO: close subscription
			continue
		}

		if peer, isPeer := writeSub.(TreePeerConn); isPeer {
			// If the subscriber wants us to send states, we never skip sending
			if tp.store.TxSeenByPeer(peer.DeviceSpecificID(), tx.StateURI, tx.ID) && !writeSub.Type().Includes(SubscriptionType_States) {
				continue
			}
		}

		if state != nil {
			// Drill down to the part of the state that the subscriber is interested in
			state = state.NodeAt(writeSub.Keypath(), nil)
		}

		if isPrivate {
			tx = nil
		} else {
			encryptedTx = nil
		}
		writeSub.EnqueueWrite(SubscriptionMsg{
			StateURI:    stateURI,
			Tx:          tx,
			EncryptedTx: encryptedTx,
			State:       state,
			Leaves:      leaves,
		})
	}
}

// type broadcastTxsToStateURIProvidersTask struct {
// 	process.PeriodicTask
// 	log.Logger
// 	treeStore                 Store
// 	peerStore                 swarm.PeerStore
// 	transports                map[string]TreeTransport
// 	txsForStateURIProviders   map[string][]tree.Tx
// 	txsForStateURIProvidersMu sync.Mutex
// }

// func NewBroadcastTxsToStateURIProvidersTask(
// 	interval time.Duration,
// 	treeStore Store,
// 	peerStore swarm.PeerStore,
// 	transports map[string]TreeTransport,
// ) *broadcastTxsToStateURIProvidersTask {
// 	t := &broadcastTxsToStateURIProvidersTask{
// 		Logger:                  log.NewLogger(ProtocolName),
// 		treeStore:               treeStore,
// 		peerStore:               peerStore,
// 		transports:              transports,
// 		txsForStateURIProviders: make(map[string][]tree.Tx),
// 	}
// 	t.PeriodicTask = *process.NewPeriodicTask("BroadcastTxsToStateURIProvidersTask", interval, t.broadcastTxsToStateURIProviders)
// 	return t
// }

// func (t *broadcastTxsToStateURIProvidersTask) addTx(tx tree.Tx) {
// 	t.txsForStateURIProvidersMu.Lock()
// 	defer t.txsForStateURIProvidersMu.Unlock()
// 	t.txsForStateURIProviders[tx.StateURI] = append(t.txsForStateURIProviders[tx.StateURI], tx.Copy())
// }

// func (t *broadcastTxsToStateURIProvidersTask) takeTxs() map[string][]tree.Tx {
// 	t.txsForStateURIProvidersMu.Lock()
// 	defer t.txsForStateURIProvidersMu.Unlock()
// 	txs := t.txsForStateURIProviders
// 	t.txsForStateURIProviders = make(map[string][]tree.Tx)
// 	return txs
// }

// func (t *broadcastTxsToStateURIProvidersTask) broadcastTxsToStateURIProviders(ctx context.Context) {
// 	txs := t.takeTxs()
// 	if len(txs) == 0 {
// 		return
// 	}

// 	t.Debugf("broadcasting txs to state URI providers")

// 	encryptedTxs := make(map[string]map[types.ID]protohush.GroupMessage)

// 	for stateURI, txs := range txs {
// 		encryptedTxs[stateURI] = make(map[types.ID]protohush.GroupMessage)

// 		for _, peerDetails := range t.peerStore.PeersServingStateURI(stateURI) {
// 			txs := txs
// 			peerDetails := peerDetails

// 			t.Process.Go(nil, peerDetails.DialInfo().String(), func(ctx context.Context) {
// 				tpt, exists := t.transports[peerDetails.DialInfo().TransportName]
// 				if !exists {
// 					return
// 				}
// 				peerConn, err := tpt.NewPeerConn(ctx, peerDetails.DialInfo().DialAddr)
// 				if err != nil {
// 					t.Errorf("while creating NewPeerConn: %v", err)
// 					return
// 				} else if !peerConn.Ready() || !peerConn.Dialable() {
// 					return
// 				}
// 				treePeer, is := peerConn.(TreePeerConn)
// 				if !is {
// 					t.Errorf("peer is not TreePeerConn, should be impossible")
// 					return
// 				}
// 				err = treePeer.EnsureConnected(ctx)
// 				if err != nil {
// 					return
// 				}
// 				defer treePeer.Close()

// 				for _, tx := range txs {
// 					if t.treeStore.TxSeenByPeer(treePeer.DeviceSpecificID(), stateURI, tx.ID) {
// 						continue
// 					}

// 					_, exists := encryptedTxs[stateURI][tx.ID]
// 					if !exists {
// 						encryptedTx, err := t.treeStore.EncryptedTx(stateURI, tx.ID)
// 						if err != nil {
// 							t.Errorf("while fetching encrypted tx from database: %v", err)
// 							continue
// 						}
// 						encryptedTxs[stateURI][tx.ID] = encryptedTx
// 					}

// 					err = treePeer.Put(ctx, SubscriptionMsg{
// 						StateURI: tx.StateURI,
// 						Tx:       &tx,
// 					})
// 					if err != nil {
// 						t.Errorf("while sending tx to state URI provider: %v", err)
// 						continue
// 					}
// 				}
// 			})
// 		}
// 	}
// }

type announceP2PStateURIsTask struct {
	process.PeriodicTask
	log.Logger
	treeProto *treeProtocol
	// txStore       tree.TxStore
	// peerStore     swarm.PeerStore
	// controllerHub tree.ControllerHub
	// transports    map[string]TreeTransport
}

func NewAnnounceP2PStateURIsTask(
	interval time.Duration,
	treeProto *treeProtocol,
) *announceP2PStateURIsTask {
	t := &announceP2PStateURIsTask{
		Logger:    log.NewLogger(ProtocolName),
		treeProto: treeProto,
	}
	t.PeriodicTask = *process.NewPeriodicTask("BroadcastTxsToStateURIProvidersTask", interval, t.announceP2PStateURIs)
	return t
}

func (t *announceP2PStateURIsTask) announceP2PStateURIs(ctx context.Context) {
	// t.Debugf("announcing p2p state URIs")

	stateURIs, err := t.treeProto.txStore.KnownStateURIs()
	if err != nil {
		t.Errorf("while fetching state URIs from tx store: %v", err)
		return
	}
	for _, stateURI := range stateURIs {
		if t.treeProto.acl.TypeOf(stateURI) != StateURIType_Private {
			continue
		}

		members, err := t.treeProto.acl.MembersOf(stateURI)
		if err != nil {
			t.Errorf("while fetching members of state URI %v: %v", stateURI, err)
			continue
		}

		for peerAddress := range members {
			for _, peerDetails := range t.treeProto.peerStore.PeersWithAddress(peerAddress) {
				tpt, exists := t.treeProto.transports[peerDetails.DialInfo().TransportName]
				if !exists {
					continue
				}
				peerConn, err := tpt.NewPeerConn(ctx, peerDetails.DialInfo().DialAddr)
				if errors.Cause(err) == swarm.ErrPeerIsSelf {
					continue
				} else if err != nil {
					t.Errorf("while creating NewPeerConn: %v", err)
					continue
				} else if !peerConn.Ready() || !peerConn.Dialable() {
					continue
				}
				treePeer, is := peerConn.(TreePeerConn)
				if !is {
					t.Errorf("peer is not TreePeerConn, should be impossible")
					continue
				}

				t.treeProto.Process.Go(nil, peerDetails.DialInfo().String(), func(ctx context.Context) {
					err := treePeer.EnsureConnected(ctx)
					if err != nil {
						return
					}
					defer treePeer.Close()

					err = treePeer.AnnounceP2PStateURI(ctx, stateURI)
					if err != nil {
						return
					}
				})
			}
		}
	}
}
