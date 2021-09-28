package protohush

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/status-im/doubleratchet"

	"redwood.dev/identity"
	"redwood.dev/log"
	"redwood.dev/process"
	"redwood.dev/swarm"
	"redwood.dev/types"
	"redwood.dev/utils"
)

//
// @@TODO:
//   - move store.EnsureDHPair() to hushProtocol and make it save the attestation
//   - handleIncomingIndividualSessionsTask#validateProposal: ensure that epoch > latest epoch. If not, return proposedSessionExists = true
//

//go:generate mockery --name HushProtocol --output ./mocks/ --case=underscore
type HushProtocol interface {
	process.Interface

	ProposeIndividualSession(ctx context.Context, sessionType string, recipient types.Address, epoch uint64) error
	ProposeNextIndividualSession(ctx context.Context, sessionType string, recipient types.Address) error

	EnqueueIndividualMessage(sessionType string, recipient types.Address, plaintext []byte) error
	// OnIndividualSessionProposed(sessionType string, handler func(proposal IndividualSessionProposal) bool)
	OnIndividualMessageDecrypted(sessionType string, handler IndividualMessageDecryptedCallback)

	// ProposeGroupSession(ctx context.Context, sessionType, name string, epoch uint64, recipients []types.Address) error
	// EnqueueGroupMessage(sessionID SessionID, plaintext []byte) error
	// OnGroupSessionProposed(session GroupSession) bool
	// OnGroupMessageEncrypted(sessionType string, handler func(sessionID SessionID, msg GroupMessage))
	// OnGroupMessageDecrypted(sessionType string, handler func(sessionID SessionID, msg GroupMessage))
}

//go:generate mockery --name HushTransport --output ./mocks/ --case=underscore
type HushTransport interface {
	swarm.Transport
	OnIncomingDHPubkeyAttestations(handler IncomingDHPubkeyAttestationsCallback)
	OnIncomingIndividualSessionProposal(handler IncomingIndividualSessionProposalCallback)
	OnIncomingIndividualSessionApproval(handler IncomingIndividualSessionApprovalCallback)
	OnIncomingIndividualMessage(handler IncomingIndividualMessageCallback)

	// OnIncomingGroupSession(handler func(ctx context.Context, encryptedProposal []byte, bob HushPeerConn))
	// OnIncomingGroupMessage(handler func(ctx context.Context, msg GroupMessage))
}

//go:generate mockery --name HushPeerConn --output ./mocks/ --case=underscore
type HushPeerConn interface {
	swarm.PeerConn
	SendDHPubkeyAttestations(ctx context.Context, attestations []DHPubkeyAttestation) error
	ProposeIndividualSession(ctx context.Context, encryptedProposal []byte) error
	ApproveIndividualSession(ctx context.Context, approval IndividualSessionApproval) error
	SendHushIndividualMessage(ctx context.Context, msg IndividualMessage) error
	// SendHushGroupMessage(ctx context.Context, msg GroupMessage) error
}

type hushProtocol struct {
	process.Process
	log.Logger

	store      Store
	keyStore   identity.KeyStore
	peerStore  swarm.PeerStore
	transports map[string]HushTransport

	individualMessageDecryptedListeners   map[string][]IndividualMessageDecryptedCallback
	individualMessageDecryptedListenersMu sync.RWMutex

	exchangeDHPubkeysTask *exchangeDHPubkeysTask
	// proposeIndividualSessionsTask         *proposeIndividualSessionsTask
	poolWorker process.PoolWorker
	// handleIncomingIndividualSessionsTask  *handleIncomingIndividualSessionsTask
	// approveIndividualSessionsTask         *approveIndividualSessionsTask
	// sendIndividualMessagesTask            *sendIndividualMessagesTask
	decryptIncomingIndividualMessagesTask *decryptIncomingIndividualMessagesTask
}

const ProtocolName = "protohush"

type IndividualMessageDecryptedCallback func(sessionID IndividualSessionID, sender types.Address, plaintext []byte)

func NewHushProtocol(
	transports []swarm.Transport,
	store Store,
	keyStore identity.KeyStore,
	peerStore swarm.PeerStore,
) *hushProtocol {
	transportsMap := make(map[string]HushTransport)
	for _, tpt := range transports {
		if tpt, is := tpt.(HushTransport); is {
			transportsMap[tpt.Name()] = tpt
		}
	}
	return &hushProtocol{
		Process:                             *process.New(ProtocolName),
		Logger:                              log.NewLogger(ProtocolName),
		store:                               store,
		keyStore:                            keyStore,
		peerStore:                           peerStore,
		transports:                          transportsMap,
		individualMessageDecryptedListeners: make(map[string][]IndividualMessageDecryptedCallback),
	}
}

func (hp *hushProtocol) Start() error {
	err := hp.Process.Start()
	if err != nil {
		return err
	}

	// Ensure we have a DH keypair and a signed attestation of that keypair in the DB
	{
		identity, err := hp.keyStore.DefaultPublicIdentity()
		if err != nil {
			return err
		}
		dhPair, err := hp.store.EnsureDHPair()
		if err != nil {
			return err
		}
		sig, err := hp.keyStore.SignHash(identity.Address(), dhPair.AttestationHash())
		if err != nil {
			return err
		}
		err = hp.store.SaveDHPubkeyAttestations([]DHPubkeyAttestation{{Pubkey: dhPair.Public, Epoch: dhPair.Epoch, Sig: sig}})
		if err != nil {
			return err
		}
	}

	hp.exchangeDHPubkeysTask = NewExchangeDHPubkeysTask(10*time.Second, hp.store, hp.peerStore, hp.keyStore, hp.transports)
	err = hp.Process.SpawnChild(nil, hp.exchangeDHPubkeysTask)
	if err != nil {
		return err
	}

	// hp.proposeIndividualSessionsTask = NewProposeIndividualSessionsTask(10*time.Second, hp.store, hp.peerStore, hp.keyStore, hp.transports)
	hp.poolWorker = process.NewPoolWorker("pool worker", 8, process.NewStaticScheduler(5*time.Second, 10*time.Second))
	err = hp.Process.SpawnChild(nil, hp.poolWorker)
	if err != nil {
		return err
	}

	// hp.handleIncomingIndividualSessionsTask = NewHandleIncomingIndividualSessionsTask(10*time.Second, hp)
	// err = hp.Process.SpawnChild(nil, hp.handleIncomingIndividualSessionsTask)
	// if err != nil {
	// 	return err
	// }

	// hp.approveIndividualSessionsTask = NewApproveIndividualSessionsTask(10*time.Second, hp)
	// err = hp.Process.SpawnChild(nil, hp.approveIndividualSessionsTask)
	// if err != nil {
	// 	return err
	// }

	// hp.sendIndividualMessagesTask = NewSendIndividualMessagesTask(10*time.Second, hp)
	// err = hp.Process.SpawnChild(nil, hp.sendIndividualMessagesTask)
	// if err != nil {
	// 	return err
	// }

	hp.decryptIncomingIndividualMessagesTask = NewDecryptIncomingIndividualMessagesTask(10*time.Second, hp)
	err = hp.Process.SpawnChild(nil, hp.decryptIncomingIndividualMessagesTask)
	if err != nil {
		return err
	}

	for _, tpt := range hp.transports {
		tpt.OnIncomingDHPubkeyAttestations(hp.handleIncomingDHPubkeyAttestations)
		tpt.OnIncomingIndividualSessionProposal(hp.handleIncomingIndividualSessionProposal)
		tpt.OnIncomingIndividualSessionApproval(hp.handleIncomingIndividualSessionApproval)
		tpt.OnIncomingIndividualMessage(hp.handleIncomingIndividualMessage)
	}

	return nil
}

func (hp *hushProtocol) EnqueueIndividualMessage(sessionType string, recipient types.Address, plaintext []byte) error {
	intent := IndividualMessageIntent{
		ID:          types.RandomID(),
		SessionType: sessionType,
		Recipient:   recipient,
		Plaintext:   plaintext,
	}
	err := hp.store.SaveOutgoingIndividualMessageIntent(intent)
	if err != nil {
		return errors.Wrapf(err, "while saving individual message intent to database")
	}
	hp.poolWorker.Add(sendIndividualMessage{sessionType, recipient, intent.ID, hp})
	return nil
}

// func (hp *hushProtocol) EnqueueGroupMessage(recipients []types.Address, plaintext []byte) error {
// 	encKey, err := crypto.NewSymEncKey()
// 	if err != nil {
// 		return GroupMessage{}, err
// 	}

// 	var encryptionKeys []Message
// 	for _, recipient := range recipients {
// 		msg, err := hp.Encrypt(recipient, encKey.Bytes())
// 		if err != nil {
// 			return GroupMessage{}, err
// 		}
// 		encryptionKeys = append(encryptionKeys, msg)
// 	}

// 	encEncKey, err := encKey.Encrypt(plaintext)
// 	if err != nil {
// 		return GroupMessage{}, err
// 	}

// 	ciphertext, err := encEncKey.MarshalBinary()
// 	if err != nil {
// 		return GroupMessage{}, err
// 	}

// 	return GroupMessage{
// 		EncryptionKeys: encryptionKeys,
// 		Ciphertext:     ciphertext,
// 	}, nil
// }

// func (hp *hushProtocol) Decrypt(sender types.Address, msg Message) ([]byte, error) {
// 	identity, err := hp.keyStore.DefaultPublicIdentity()
// 	if err != nil {
// 		hp.Errorf("while fetching default public identity: %v", err)
// 		return
// 	}
// 	aliceAddr, bobAddr := addrsSorted(identity.Address(), sender)

// 	sessionID, err := hp.store.LatestIndividualSessionIDWithUsers(aliceAddr, bobAddr)
// 	if err != nil {
// 		return Message{}, err
// 	}

// 	session, err := doubleratchet.Load(sessionID.Bytes(), hp.store.RatchetSessionStore())
// 	if err != nil {
// 		return nil, err
// 	}
// 	m := doubleratchet.Message{
// 		Header: doubleratchet.MessageHeader{
// 			DH: msg.Header.DhPubkey,
// 			N:  msg.Header.N,
// 			PN: msg.Header.Pn,
// 		},
// 		Ciphertext: msg.Ciphertext,
// 	}
// 	return session.RatchetDecrypt(m, nil)
// }

// func (hp *hushProtocol) DecryptFromGroup(msg GroupMessage) ([]byte, error) {
// 	groupSession, err := hp.store.LatestGroupSessionWithIDHash(msg.GroupSessionIDHash)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var symEncMsg crypto.SymEncMsg
// 	err = symEncMsg.UnmarshalBinary(msg.Ciphertext)
// 	if err != nil {
// 		return nil, err
// 	}

// 	for _, encEncKey := range msg.EncryptionKeys {
// 		for _, member := range groupSession.Members {
// 			symEncKeyBytes, err := hp.Decrypt(member, encEncKey)
// 			if err != nil {
// 				// return nil, err
// 				continue
// 			}
// 			symEncKey := crypto.SymEncKeyFromBytes(symEncKeyBytes)
// 			plaintext, err := symEncKey.Decrypt(symEncMsg)
// 			if err != nil {
// 				continue
// 			}
// 			return plaintext, nil
// 		}
// 	}
// 	return nil, errors.New("could not decrypt")
// }

func (hp *hushProtocol) ProposeNextIndividualSession(ctx context.Context, sessionType string, recipient types.Address) error {
	identity, err := hp.keyStore.DefaultPublicIdentity()
	if err != nil {
		return errors.Wrapf(err, "while fetching default public identity")
	}

	session, err := hp.store.LatestIndividualSessionWithUsers(sessionType, identity.Address(), recipient)
	if errors.Cause(err) == types.Err404 {
		return hp.ProposeIndividualSession(ctx, sessionType, recipient, 0)
	} else if err != nil {
		return errors.Wrapf(err, "while fetching latest individual session ID with %v", recipient)
	}
	return hp.ProposeIndividualSession(ctx, sessionType, recipient, session.SessionID.Epoch+1)
}

func (hp *hushProtocol) ProposeIndividualSession(ctx context.Context, sessionType string, recipient types.Address, epoch uint64) error {
	sk, err := GenerateSharedKey()
	if err != nil {
		return errors.Wrapf(err, "while generating shared key for session %v-%v-%v", sessionType, recipient.Hex(), epoch)
	}

	identity, err := hp.keyStore.DefaultPublicIdentity()
	if err != nil {
		return errors.Wrapf(err, "while fetching default public identity")
	}

	sessionID := IndividualSessionID{
		SessionType: sessionType,
		AliceAddr:   identity.Address(),
		BobAddr:     recipient,
		Epoch:       epoch,
	}

	proposal := IndividualSessionProposal{
		SessionID: sessionID,
		SharedKey: sk,
	}

	err = hp.store.SaveOutgoingIndividualSessionProposal(proposal)
	if err != nil {
		return err
	}

	proposalHash, err := proposal.Hash()
	if err != nil {
		return err
	}

	hp.poolWorker.Add(proposeIndividualSessions{proposalHash, hp})
	return nil
}

// func (hp *hushProtocol) ProposeGroupSession(ctx context.Context, sessionType, name string, epoch uint64, recipients []types.Address) error {
// 	var members [][]byte
// 	for _, recipient := range recipients {
// 		members = append(members, recipient.Bytes())
// 	}

// 	proposal := GroupSession{
// 		SessionID: GroupSessionID{
// 			SessionType: sessionType,
// 			Name:        name,
// 			Epoch:       epoch,
// 		},
// 		Members: members,
// 	}
// 	err = hp.store.SaveOutgoingIndividualSession(recipientAddr, proposal)
// 	if err != nil {
// 		return err
// 	}
// 	hp.proposeIndividualSessionsTask.Enqueue()
// 	return nil
// }

func (hp *hushProtocol) handleIncomingDHPubkeyAttestations(ctx context.Context, attestations []DHPubkeyAttestation, peer HushPeerConn) {
	var valid []DHPubkeyAttestation
	addrs := utils.NewAddressSet(nil)
	for _, a := range attestations {
		addr, err := a.Address()
		if err != nil {
			continue
		}
		addrs.Add(addr)
		valid = append(valid, a)
	}

	err := hp.store.SaveDHPubkeyAttestations(valid)
	if err != nil {
		hp.Errorf("while saving DH pubkey attestations: %v", err)
		return
	}

	// Kick off workers to propose outgoing indvidual sessions that depend on the DH pubkeys we received
	{
		identity, err := hp.keyStore.DefaultPublicIdentity()
		if err != nil {
			hp.Errorf("while fetching default public identity: %v", err)
			return
		}
		myAddr := identity.Address()

		for addr := range addrs {
			proposals, err := hp.store.OutgoingIndividualSessionProposalsForUsers(myAddr, addr)
			if err != nil {
				hp.Errorf("while fetching outgoing individual session proposals for %v and %v: %v", myAddr, addr, err)
				continue
			}
			for _, p := range proposals {
				proposalHash, err := p.Hash()
				if err != nil {
					hp.Errorf("while hashing outgoing individual session proposal for %v and %v: %v", myAddr, addr, err)
					continue
				}
				hp.Successf("retrying outgoing session proposal %v", proposalHash)
				// hp.poolWorker.Add(proposeIndividualSessions{proposalHash, hp})
				hp.poolWorker.ForceRetry(proposeIndividualSessions{proposalHash, hp})
			}

			approvals, err := hp.store.OutgoingIndividualSessionApprovalsForUser(addr)
			if err != nil {
				hp.Errorf("while fetching outgoing individual session approvals for %v and %v: %v", myAddr, addr, err)
				continue
			}
			for _, a := range approvals {
				hp.Successf("retrying outgoing session approval %v", a.ProposalHash)
				// hp.poolWorker.Add(proposeIndividualSessions{proposalHash, hp})
				hp.poolWorker.ForceRetry(approveIndividualSession{addr, a.ProposalHash, hp})
			}
		}
	}

	// hp.approveIndividualSessionsTask.Enqueue()
	// hp.sendIndividualMessagesTask.Enqueue()
}

func (hp *hushProtocol) handleIncomingIndividualSessionProposal(ctx context.Context, encryptedProposal []byte, alice HushPeerConn) {
	aliceAddrs := alice.Addresses()
	if len(aliceAddrs) == 0 {
		hp.Errorf("while handling incoming individual session proposal: not authenticated")
		return
	}
	aliceAddr := aliceAddrs[0]
	proposal := EncryptedIndividualSessionProposal{aliceAddr, encryptedProposal}

	err := hp.store.SaveIncomingIndividualSessionProposal(proposal)
	if err != nil {
		hp.Errorf("while saving incoming individual session proposal: %v", err)
		return
	}
	hp.poolWorker.Add(handleIncomingIndividualSessions{proposal.Hash(), hp})
}

func (hp *hushProtocol) handleIncomingIndividualSessionApproval(ctx context.Context, approval IndividualSessionApproval, bob HushPeerConn) {
	hp.Warnf("incoming individual session approval %v", utils.PrettyJSON(approval))

	proposal, err := hp.store.OutgoingIndividualSessionProposalByHash(approval.ProposalHash)
	if errors.Cause(err) == types.Err404 {
		hp.Errorf("proposal hash does not match any known proposal (hash: %v)", approval.ProposalHash)
		return
	} else if err != nil {
		hp.Errorf("while fetching outgoing shared key proposal: %v", err)
		return
	}

	// bobAddrs := bob.Addresses()
	// if len(bobAddrs) == 0 {
	// 	hp.Errorf("not authenticated with peer")
	// 	return
	// }

	// This saves the session to the DB
	_, err = doubleratchet.NewWithRemoteKey(
		proposal.SessionID.Bytes(),
		proposal.SharedKey[:],
		proposal.RemoteDHPubkey,
		hp.store.RatchetSessionStore(),
		doubleratchet.WithKeysStorage(hp.store.RatchetKeyStore()),
	)
	if err != nil {
		hp.Errorf("while initializing double ratchet session: %v", err)
		return
	}

	err = hp.store.SaveApprovedIndividualSession(proposal)
	if err != nil {
		hp.Errorf("while saving approved individual session: %v", err)
		return
	}

	err = hp.store.DeleteOutgoingIndividualSessionProposal(approval.ProposalHash)
	if err != nil {
		hp.Errorf("while deleting incoming individual session approval: %v", err)
		return
	}

	// Retry sending any pending messages
	{
		sessionID, err := hp.store.IndividualSessionIDBySessionHash(approval.ProposalHash)
		if err != nil {
			hp.Errorf("while fetching individual session ID from database: %v", err)
			return
		}
		hp.Debugf("retrying messages for session %v / %v %v", sessionID, proposal.SessionID.SessionType, proposal.SessionID.BobAddr)
		ids, err := hp.store.OutgoingIndividualMessageIntentIDsForTypeAndRecipient(proposal.SessionID.SessionType, proposal.SessionID.BobAddr)
		if err != nil {
			hp.Errorf("while fetching individual message IDs from database: %v", err)
			return
		}
		hp.Debugf("retrying messages for session %v (len %v)", sessionID, ids)
		for id := range ids {
			hp.Debugf("retrying message %v", id)
			hp.poolWorker.Add(sendIndividualMessage{sessionID.SessionType, proposal.SessionID.BobAddr, id, hp})
		}
	}
}

func (hp *hushProtocol) handleIncomingIndividualMessage(ctx context.Context, msg IndividualMessage, peerConn HushPeerConn) {
	peerAddrs := peerConn.Addresses()
	if len(peerAddrs) == 0 {
		panic("impossible") // @@TODO
	}
	sender := peerAddrs[0]

	hp.Debugf("received encrypted individual message from %v", sender)

	err := hp.store.SaveIncomingIndividualMessage(sender, msg)
	if err != nil {
		hp.Errorf("while saving incoming individual message: %v", err)
		return
	}
	hp.decryptIncomingIndividualMessagesTask.Enqueue()
}

func (hp *hushProtocol) OnIndividualMessageDecrypted(sessionType string, handler IndividualMessageDecryptedCallback) {
	hp.individualMessageDecryptedListenersMu.Lock()
	defer hp.individualMessageDecryptedListenersMu.Unlock()
	hp.individualMessageDecryptedListeners[sessionType] = append(hp.individualMessageDecryptedListeners[sessionType], handler)
}

func (hp *hushProtocol) notifyIndividualMessageDecryptedListeners(sessionType string, sessionID IndividualSessionID, sender types.Address, plaintext []byte) {
	hp.individualMessageDecryptedListenersMu.RLock()
	defer hp.individualMessageDecryptedListenersMu.RUnlock()

	child := hp.Process.NewChild(context.TODO(), "notify individual message decrypted listener")
	for _, listener := range hp.individualMessageDecryptedListeners[sessionType] {
		listener := listener
		child.Go(context.TODO(), "notify individual message decrypted listener", func(ctx context.Context) {
			listener(sessionID, sender, plaintext)
		})
	}
	child.Autoclose()
	<-child.Done()
}

type exchangeDHPubkeysTask struct {
	process.PeriodicTask
	log.Logger
	store      Store
	peerStore  swarm.PeerStore
	keyStore   identity.KeyStore
	transports map[string]HushTransport
}

func NewExchangeDHPubkeysTask(
	interval time.Duration,
	store Store,
	peerStore swarm.PeerStore,
	keyStore identity.KeyStore,
	transports map[string]HushTransport,
) *exchangeDHPubkeysTask {
	t := &exchangeDHPubkeysTask{
		Logger:     log.NewLogger(ProtocolName),
		store:      store,
		peerStore:  peerStore,
		keyStore:   keyStore,
		transports: transports,
	}
	t.PeriodicTask = *process.NewPeriodicTask("ExchangeDHPubkeysTask", interval, t.exchangeDHPubkeys)
	return t
}

func (t *exchangeDHPubkeysTask) exchangeDHPubkeys(ctx context.Context) {
	attestations, err := t.store.DHPubkeyAttestations()
	if err != nil {
		t.Errorf("while fetching DH pubkey attestations: %v", err)
		return
	}

	for _, peer := range t.peerStore.VerifiedPeers() {
		if !peer.Ready() || !peer.Dialable() || len(peer.Addresses()) == 0 {
			continue
		}
		tpt, ok := t.transports[peer.DialInfo().TransportName]
		if !ok {
			continue
		}
		peerConn, err := tpt.NewPeerConn(ctx, peer.DialInfo().DialAddr)
		if err != nil {
			continue
		}
		hushPeerConn, ok := peerConn.(HushPeerConn)
		if !ok {
			continue
		}

		peerAddrs := peer.Addresses()
		peerAddr := peerAddrs[0]

		name := fmt.Sprintf("exchange DH pubkey (addr=%v)", peerAddr)

		t.Process.Go(ctx, name, func(ctx context.Context) {
			err = hushPeerConn.EnsureConnected(ctx)
			if err != nil {
				t.Errorf("while exchanging DH pubkey: %v", err)
				return
			}
			defer hushPeerConn.Close()

			// t.Debugf("sending DH pubkey attestations:")
			// for _, a := range attestations {
			// 	addr, err := a.Address()
			// 	if err != nil {
			// 		t.Errorf("  - (bad signature) %v: %v", a.PubkeyHex(), err)
			// 		continue
			// 	}
			// 	t.Debugf("  - %v: %v (epoch=%v)", addr, a.PubkeyHex(), a.Epoch)
			// }

			err = hushPeerConn.SendDHPubkeyAttestations(ctx, attestations)
			if err != nil {
				t.Errorf("while exchanging DH pubkey: %v", err)
				return
			}
		})
	}
}

type proposeIndividualSessions struct {
	proposalHash types.Hash
	hushProto    *hushProtocol
}

var _ process.PoolWorkerItem = proposeIndividualSessions{}

func (t proposeIndividualSessions) BlacklistUniqueID() process.PoolUniqueID    { return t }
func (t proposeIndividualSessions) RetryUniqueID() process.PoolUniqueID        { return t }
func (t proposeIndividualSessions) DedupeActiveUniqueID() process.PoolUniqueID { return t }
func (t proposeIndividualSessions) ID() string                                 { return t.proposalHash.Hex() }

func (t proposeIndividualSessions) Work(ctx context.Context) (retry bool) {
	defer func() {
		if retry {
			t.hushProto.Warnf("Propose individual session %v: retrying later", t.proposalHash)
		} else {
			t.hushProto.Warnf("Propose individual session %v: done", t.proposalHash)
		}
	}()

	session, err := t.hushProto.store.OutgoingIndividualSessionProposalByHash(t.proposalHash)
	if errors.Cause(err) == types.Err404 {
		return false
	} else if err != nil {
		t.hushProto.Errorf("while fetching outgoing individual session proposal: %v", err)
		return true
	}

	if len(t.hushProto.peerStore.PeersWithAddress(session.SessionID.BobAddr)) == 0 {
		return true
	}

	identity, err := t.hushProto.keyStore.DefaultPublicIdentity()
	if err != nil {
		t.hushProto.Errorf("while fetching default public identity from keystore: %v", err)
		return true
	}

	// Assign Bob's latest DH pubkey
	remoteDHPubkey, err := t.hushProto.store.LatestDHPubkeyFor(session.SessionID.BobAddr)
	if errors.Cause(err) == types.Err404 {
		t.hushProto.Debugf("waiting to propose individual session: still awaiting DH pubkey for %v", session.SessionID.BobAddr)
		return true
	} else if err != nil {
		t.hushProto.Errorf("while fetching latest DH pubkey for %v: %v", session.SessionID.BobAddr, err)
		return true
	}
	session.RemoteDHPubkey = remoteDHPubkey.Pubkey

	// Sign the proposal
	hash, err := session.Hash()
	if err != nil {
		t.hushProto.Errorf("while hashing outgoing individual session proposal: %v", err)
		return true
	}
	sig, err := t.hushProto.keyStore.SignHash(identity.Address(), hash)
	if err != nil {
		t.hushProto.Errorf("while signing hash of outgoing individual session proposal: %v", err)
		return true
	}
	session.AliceSig = sig

	err = t.hushProto.store.SaveOutgoingIndividualSessionProposal(session)
	if err != nil {
		t.hushProto.Errorf("while updating outgoing individual session proposal: %v", err)
		return true
	}

	sessionBytes, err := session.Marshal()
	if err != nil {
		t.hushProto.Errorf("while marshaling outgoing individual session proposal: %v", err)
		return true
	}

	err = findPeerByAddress(ctx, t.hushProto.peerStore, t.hushProto.transports, session.SessionID.BobAddr, func(bob HushPeerConn) error {
		err = bob.EnsureConnected(ctx)
		if err != nil {
			return err
		}
		defer bob.Close()

		_, bobAsymEncPubkey := bob.PublicKeys(session.SessionID.BobAddr)

		encryptedProposalBytes, err := t.hushProto.keyStore.SealMessageFor(identity.Address(), bobAsymEncPubkey, sessionBytes)
		if err != nil {
			return err
		}

		err = bob.ProposeIndividualSession(ctx, encryptedProposalBytes)
		if err != nil {
			return err
		}
		t.hushProto.Infof(0, "proposed individual session %v", session.SessionID)
		return nil
	})
	if err != nil {
		t.hushProto.Errorf("while proposing individual session to peer: %v", err)
	}
	return true // Continue retrying until we get an approval
}

// type handleIncomingIndividualSessionsTask struct {
// 	process.PeriodicTask
// 	log.Logger
// 	hushProto *hushProtocol
// }

// func NewHandleIncomingIndividualSessionsTask(
// 	interval time.Duration,
// 	hushProto *hushProtocol,
// ) *handleIncomingIndividualSessionsTask {
// 	t := &handleIncomingIndividualSessionsTask{
// 		Logger:    log.NewLogger(ProtocolName),
// 		hushProto: hushProto,
// 	}
// 	t.PeriodicTask = *process.NewPeriodicTask("HandleIncomingIndividualSessionsTask", interval, t.handleIncomingIndividualSessions)
// 	return t
// }

// func (t *handleIncomingIndividualSessionsTask) handleIncomingIndividualSessions(ctx context.Context) {
// 	proposals, err := t.hushProto.store.IncomingIndividualSessionProposals()
// 	if err != nil {
// 		t.Errorf("while fetching incoming individual session proposals from database: %v", err)
// 		return
// 	}

// 	for _, proposal := range proposals {
// 		name := fmt.Sprintf("handle incoming individual session proposal (from=%v)", proposal.AliceAddr)
// 		proposal := proposal

// 		t.Process.Go(ctx, name, func(ctx context.Context) {
// 			t.handleIncomingIndividualSession(ctx, proposal)
// 		})
// 	}
// }

type handleIncomingIndividualSessions struct {
	proposalHash types.Hash
	hushProto    *hushProtocol
}

var _ process.PoolWorkerItem = handleIncomingIndividualSessions{}

func (t handleIncomingIndividualSessions) BlacklistUniqueID() process.PoolUniqueID    { return t }
func (t handleIncomingIndividualSessions) RetryUniqueID() process.PoolUniqueID        { return t }
func (t handleIncomingIndividualSessions) DedupeActiveUniqueID() process.PoolUniqueID { return t }
func (t handleIncomingIndividualSessions) ID() string                                 { return t.proposalHash.Hex() }

func (t handleIncomingIndividualSessions) Work(ctx context.Context) (retry bool) {
	defer func() {
		if retry {
			t.hushProto.Warnf("Handle incoming individual session %v: retrying later", t.proposalHash)
		} else {
			t.hushProto.Warnf("Handle incoming individual session %v: done", t.proposalHash)
		}
	}()

	proposal, err := t.hushProto.store.IncomingIndividualSessionProposal(t.proposalHash)
	if errors.Cause(err) == types.Err404 {
		// Done
		return false
	} else if err != nil {
		t.hushProto.Errorf("while fetching incoming individual session proposal from database: %v", err)
		return true
	}

	t.hushProto.Successf("INCOMING PROPOSAL %v", utils.PrettyJSON(proposal))

	identity, err := t.hushProto.keyStore.DefaultPublicIdentity()
	if err != nil {
		t.hushProto.Errorf("while fetching identity from database: %v", err)
		return true
	}
	// myAddr := identity.Address()

	proposedSession, err := t.decryptProposal(proposal)
	if err != nil {
		t.hushProto.Errorf("while decrypting incoming individual session proposal: %v", err)
		return true
	}
	t.hushProto.Debugf("incoming individual session proposal: %v", proposedSession.SessionID)

	err = t.validateProposal(proposal.AliceAddr, proposedSession)
	switch errors.Cause(err) {
	case ErrInvalidSignature:
		t.hushProto.Errorf("invalid signature on incoming individual session proposal")
		err := t.hushProto.store.DeleteIncomingIndividualSessionProposal(proposal)
		if err != nil {
			t.hushProto.Errorf("while deleting incoming individual session proposal: %v", err)
			return false
		}
		return false
	case ErrEpochTooLow:
		return false

	case ErrInvalidDHPubkey:
		t.hushProto.Debugf("ignoring session proposal: DH pubkey %v does not match our latest", proposedSession.RemoteDHPubkey)
		err := t.hushProto.store.DeleteIncomingIndividualSessionProposal(proposal)
		if err != nil {
			t.hushProto.Errorf("while deleting incoming individual session proposal: %v", err)
		}
		// t.proposeReplacementSession(myAddr, proposal, proposedSession)
		return false

	default:
		t.hushProto.Errorf("while validating incoming individual session proposal: %v", err)
		return true

	case nil:
		// Yay
	}

	// // If this is a replacement session proposal, delete our original proposal from the DB
	// if (proposedSession.ReplacesSessionWithHash != types.Hash{}) {
	// 	err := t.hushProto.store.DeleteOutgoingIndividualSessionProposal(proposedSession.ReplacesSessionWithHash)
	// 	if err != nil {
	// 		t.hushProto.Errorf("while deleting outgoing individual session proposal: %v", err)
	// 	}
	// }

	// Save the session to the DB
	dhPair, err := t.hushProto.store.EnsureDHPair()
	if err != nil {
		t.hushProto.Errorf("while fetching DH pair: %v", err)
		return true
	}

	t.hushProto.Infof(0, "opened incoming individual session %v", proposedSession.SessionID)

	_, err = doubleratchet.New(
		proposedSession.SessionID.Bytes(),
		proposedSession.SharedKey[:],
		dhPair,
		t.hushProto.store.RatchetSessionStore(),
		doubleratchet.WithKeysStorage(t.hushProto.store.RatchetKeyStore()),
	)
	if err != nil {
		t.hushProto.Errorf("while initializing double ratchet session: %v", err)
		return true
	}

	err = t.hushProto.store.SaveApprovedIndividualSession(proposedSession)
	if err != nil {
		t.hushProto.Errorf("while saving approved individual session: %v", err)
		return true
	}

	// Prepare the approval message for sending
	proposalHash, err := proposedSession.Hash()
	if err != nil {
		t.hushProto.Errorf("while hashing individual session proposal: %v", err)
		return true
	}
	sig, err := identity.SignHash(proposalHash)
	if err != nil {
		t.hushProto.Errorf("while signing individual session proposal hash: %v", err)
		return true
	}
	approval := IndividualSessionApproval{
		ProposalHash: proposalHash,
		BobSig:       sig,
	}
	err = t.hushProto.store.SaveOutgoingIndividualSessionApproval(proposedSession.SessionID.AliceAddr, approval)
	if err != nil {
		t.hushProto.Errorf("while saving outgoing individual session approval: %v", err)
		return true
	}

	// Delete the incoming proposal
	err = t.hushProto.store.DeleteIncomingIndividualSessionProposal(proposal)
	if err != nil {
		t.hushProto.Errorf("while deleting incoming individual session proposal: %v", err)
		return true
	}

	t.hushProto.poolWorker.Add(approveIndividualSession{proposedSession.SessionID.AliceAddr, proposalHash, t.hushProto})
	return false
}

func (t handleIncomingIndividualSessions) decryptProposal(proposal EncryptedIndividualSessionProposal) (IndividualSessionProposal, error) {
	identity, err := t.hushProto.keyStore.DefaultPublicIdentity()
	if err != nil {
		return IndividualSessionProposal{}, errors.Wrapf(err, "while fetching default public identity")
	}

	alices := t.hushProto.peerStore.PeersWithAddress(proposal.AliceAddr)
	if len(alices) == 0 {
		return IndividualSessionProposal{}, errors.Wrapf(err, "no known peer with address %v", proposal.AliceAddr)
	}
	alice := alices[0]

	_, aliceAsymEncPubkey := alice.PublicKeys(proposal.AliceAddr)

	sessionBytes, err := t.hushProto.keyStore.OpenMessageFrom(identity.Address(), aliceAsymEncPubkey, proposal.EncryptedProposal)
	if err != nil {
		return IndividualSessionProposal{}, errors.Wrapf(err, "while decrypting shared key proposal")
	}

	var proposedSession IndividualSessionProposal
	err = proposedSession.Unmarshal(sessionBytes)
	if err != nil {
		return IndividualSessionProposal{}, errors.Wrapf(err, "while unmarshaling shared key proposal")
	}
	return proposedSession, nil
}

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrInvalidDHPubkey  = errors.New("invalid DH pubkey")
	ErrEpochTooLow      = errors.New("epoch too low")
)

func (t handleIncomingIndividualSessions) validateProposal(sender types.Address, proposedSession IndividualSessionProposal) error {
	peers := t.hushProto.peerStore.PeersWithAddress(sender)
	if len(peers) == 0 {
		return errors.Errorf("no known peer with address %v", sender)
	}
	peer := peers[0]
	peerSigPubkey, _ := peer.PublicKeys(sender)

	// Check the signature
	sessionHash, err := proposedSession.Hash()
	if err != nil {
		return errors.Wrapf(err, "while hashing incoming individual session proposal")
	} else if !peerSigPubkey.VerifySignature(sessionHash, proposedSession.AliceSig) {
		return ErrInvalidSignature
	}

	// Check the DH pubkey against our latest
	dhPair, err := t.hushProto.store.EnsureDHPair()
	if err != nil {
		return errors.Wrapf(err, "while fetching DH keypair from database")
	}
	if !bytes.Equal(dhPair.Public, proposedSession.RemoteDHPubkey) {
		return ErrInvalidDHPubkey
	}

	// Check to see if this is a re-send of a known proposal (same sessionID and shared key)
	var sessionExists bool
	existingSession, err := t.hushProto.store.IndividualSessionByID(proposedSession.SessionID)
	if errors.Cause(err) == types.Err404 {
		// Do nothing
	} else if err != nil {
		return errors.Wrapf(err, "while checking database for session")
	} else {
		sessionExists = true
	}

	if sessionExists {
		if existingSession.SharedKey != proposedSession.SharedKey {
			return ErrEpochTooLow
		}

	} else {
		// Check to see if the epoch is high enough
		latestSession, err := t.hushProto.store.LatestIndividualSessionWithUsers(proposedSession.SessionID.SessionType, proposedSession.SessionID.AliceAddr, proposedSession.SessionID.BobAddr)
		if errors.Cause(err) == types.Err404 {
			// We have no sessions, no issue with epoch
			return nil
		} else if err != nil {
			return errors.Wrapf(err, "while checking database for latest session")
		}

		if proposedSession.SessionID.Epoch <= latestSession.SessionID.Epoch {
			return ErrEpochTooLow
		}
	}

	// } else {
	// 	haveAnySessions = true
	// }

	// var haveThisSession bool

	// if haveAnySessions

	// if session.SessionID.Epoch

	return nil
}

func (t handleIncomingIndividualSessions) proposeReplacementSession(myAddr types.Address, proposal EncryptedIndividualSessionProposal, proposedSession IndividualSessionProposal) {
	var replacementSessionID IndividualSessionID
	_, err := t.hushProto.store.LatestIndividualSessionWithUsers(proposedSession.SessionID.SessionType, myAddr, proposal.AliceAddr)
	if errors.Cause(err) == types.Err404 {
		replacementSessionID = IndividualSessionID{
			SessionType: proposedSession.SessionID.SessionType,
			AliceAddr:   myAddr,
			BobAddr:     proposedSession.SessionID.AliceAddr,
			Epoch:       0,
		}

	} else if err != nil {
		t.hushProto.Errorf("while checking database for latest session ID: %v", err)
		return

	} else {
		replacementSessionID = IndividualSessionID{
			SessionType: proposedSession.SessionID.SessionType,
			AliceAddr:   myAddr,
			BobAddr:     proposedSession.SessionID.AliceAddr,
			Epoch:       proposedSession.SessionID.Epoch + 1,
		}
	}

	bobDHPubkeyAttestation, err := t.hushProto.store.LatestDHPubkeyFor(proposal.AliceAddr)
	if err != nil {
		t.hushProto.Errorf("while fetching DH pubkey for %v from database: %v", proposal.AliceAddr, err)
		return
	}

	proposedSessionHash, err := proposedSession.Hash()
	if err != nil {
		t.hushProto.Errorf("while hashing proposed session: %v", err)
		return
	}

	replacementSession := IndividualSessionProposal{
		SessionID:               replacementSessionID,
		SharedKey:               proposedSession.SharedKey,
		AliceSig:                nil,
		ReplacesSessionWithHash: proposedSessionHash,
		RemoteDHPubkey:          bobDHPubkeyAttestation.Pubkey,
	}

	replacementSessionHash, err := replacementSession.Hash()
	if err != nil {
		t.hushProto.Errorf("while hashing individual session proposal: %v", err)
		return
	}

	aliceSig, err := t.hushProto.keyStore.SignHash(myAddr, replacementSessionHash)
	if err != nil {
		t.hushProto.Errorf("while signing hash of individual session proposal: %v", err)
		return
	}
	replacementSession.AliceSig = aliceSig

	err = t.hushProto.store.SaveOutgoingIndividualSessionProposal(replacementSession)
	if err != nil {
		t.hushProto.Errorf("while suggesting next session ID: %v", err)
		return
	}

	err = t.hushProto.store.DeleteIncomingIndividualSessionProposal(proposal)
	if err != nil {
		t.hushProto.Errorf("while deleting incoming individual session proposal: %v", err)
		return
	}
}

// type approveIndividualSessionsTask struct {
// 	process.PeriodicTask
// 	log.Logger
// 	hushProto *hushProtocol
// }

// func NewApproveIndividualSessionsTask(
// 	interval time.Duration,
// 	hushProto *hushProtocol,
// ) *approveIndividualSessionsTask {
// 	t := &approveIndividualSessionsTask{
// 		Logger:    log.NewLogger(ProtocolName),
// 		hushProto: hushProto,
// 	}
// 	t.PeriodicTask = *process.NewPeriodicTask("ApproveIndividualSessionsTask", interval, t.approveIndividualSessions)
// 	return t
// }

// func (t *approveIndividualSessionsTask) approveIndividualSessions(ctx context.Context) {
// 	approvalsByAddress, err := t.hushProto.store.OutgoingIndividualSessionApprovals()
// 	if err != nil {
// 		t.Errorf("while fetching outgoing individual session approvals from database: %v", err)
// 		return
// 	}

// 	for aliceAddr, approvals := range approvalsByAddress {
// 		for _, approval := range approvals {
// 			name := fmt.Sprintf("handle outgoing individual session approval (to=%v)", aliceAddr)
// 			approval := approval

// 			t.Process.Go(ctx, name, func(ctx context.Context) {
// 				t.approveIndividualSession(ctx, aliceAddr, approval)
// 			})
// 		}
// 	}
// }

type approveIndividualSession struct {
	aliceAddr    types.Address
	proposalHash types.Hash
	hushProto    *hushProtocol
}

var _ process.PoolWorkerItem = approveIndividualSession{}

func (t approveIndividualSession) BlacklistUniqueID() process.PoolUniqueID    { return t }
func (t approveIndividualSession) RetryUniqueID() process.PoolUniqueID        { return t }
func (t approveIndividualSession) DedupeActiveUniqueID() process.PoolUniqueID { return t }
func (t approveIndividualSession) ID() string                                 { return t.proposalHash.Hex() }

func (t approveIndividualSession) Work(ctx context.Context) (retry bool) {
	approval, err := t.hushProto.store.OutgoingIndividualSessionApproval(t.aliceAddr, t.proposalHash)
	if errors.Cause(err) == types.Err404 {
		// Done
		return false
	} else if err != nil {
		t.hushProto.Errorf("while fetching incoming individual session approval from database: %v", err)
		return true
	}

	t.hushProto.Warnf("approving individual incoming session with %v (hash: %v)", t.aliceAddr, t.proposalHash)
	defer func() {
		if retry {
			t.hushProto.Warnf("Approve incoming individual session %v: retrying later", t.proposalHash)
		} else {
			t.hushProto.Warnf("Approve incoming individual session %v: done", t.proposalHash)
		}
	}()

	err = findPeerByAddress(ctx, t.hushProto.peerStore, t.hushProto.transports, t.aliceAddr, func(alice HushPeerConn) error {
		err := alice.EnsureConnected(ctx)
		if err != nil {
			return err
		}
		defer alice.Close()

		err = alice.ApproveIndividualSession(ctx, approval)
		if err != nil {
			return err
		}
		t.hushProto.Debugf("approved individual session with %v", t.aliceAddr)

		err = t.hushProto.store.DeleteOutgoingIndividualSessionApproval(t.aliceAddr, approval.ProposalHash)
		if err != nil {
			return err
		}

		// t.hushProto.sendIndividualMessagesTask.Enqueue()

		return nil
	})
	if err != nil {
		t.hushProto.Errorf("while approving incoming individual session: %v", err)
		return true
	}

	// Retry sending any pending messages
	{
		sessionID, err := t.hushProto.store.IndividualSessionIDBySessionHash(t.proposalHash)
		if err != nil {
			t.hushProto.Errorf("while fetching individual session ID from database: %v", err)
			return true
		}
		ids, err := t.hushProto.store.OutgoingIndividualMessageIntentIDsForTypeAndRecipient(sessionID.SessionType, t.aliceAddr)
		if err != nil {
			t.hushProto.Errorf("while fetching individual message IDs from database: %v", err)
			return true
		}
		for id := range ids {
			t.hushProto.poolWorker.Add(sendIndividualMessage{sessionID.SessionType, t.aliceAddr, id, t.hushProto})
		}
	}
	return false
}

// type sendIndividualMessagesTask struct {
// 	process.PeriodicTask
// 	log.Logger
// 	hushProto *hushProtocol
// }

// func NewSendIndividualMessagesTask(
// 	interval time.Duration,
// 	hushProto *hushProtocol,
// ) *sendIndividualMessagesTask {
// 	t := &sendIndividualMessagesTask{
// 		Logger:    log.NewLogger(ProtocolName),
// 		hushProto: hushProto,
// 	}
// 	t.PeriodicTask = *process.NewPeriodicTask("SendIndividualMessagesTask", interval, t.sendIndividualMessages)
// 	return t
// }

// func (t *sendIndividualMessagesTask) sendIndividualMessages(ctx context.Context) {
// 	intents, err := t.hushProto.store.OutgoingIndividualMessageIntents()
// 	if err != nil {
// 		t.Errorf("while fetching outgoing messages from database: %v", err)
// 		return
// 	}

// 	for _, intent := range intents {
// 		name := fmt.Sprintf("send individual message (to=%v)", intent.Recipient)
// 		intent := intent

// 		t.Process.Go(ctx, name, func(ctx context.Context) {
// 			t.sendIndividualMessage(ctx, intent)
// 		})
// 	}
// }

type sendIndividualMessage struct {
	sessionType string
	recipient   types.Address
	id          types.ID
	hushProto   *hushProtocol
}

var _ process.PoolWorkerItem = sendIndividualMessage{}

func (t sendIndividualMessage) BlacklistUniqueID() process.PoolUniqueID    { return t }
func (t sendIndividualMessage) RetryUniqueID() process.PoolUniqueID        { return t }
func (t sendIndividualMessage) DedupeActiveUniqueID() process.PoolUniqueID { return t }
func (t sendIndividualMessage) ID() string                                 { return t.id.Hex() }

func (t sendIndividualMessage) Work(ctx context.Context) (retry bool) {
	intent, err := t.hushProto.store.OutgoingIndividualMessageIntent(t.sessionType, t.recipient, t.id)
	if errors.Cause(err) == types.Err404 {
		// Done
		return false
	} else if err != nil {
		t.hushProto.Errorf("while fetching message intent from database: %v", err)
		return true
	}

	t.hushProto.Debugf(`sending individual message to %v ("%v")`, intent.Recipient, utils.TrimStringToLen(string(intent.Plaintext), 10))
	defer func() {
		if retry {
			t.hushProto.Warnf("Send individual message %v: retrying later", t.id)
		} else {
			t.hushProto.Warnf("Send individual message %v: done", t.id)
		}
	}()

	identity, err := t.hushProto.keyStore.DefaultPublicIdentity()
	if err != nil {
		t.hushProto.Errorf("while fetching default public identity from database: %v", err)
		return true
	}

	session, err := t.hushProto.store.LatestIndividualSessionWithUsers(intent.SessionType, identity.Address(), intent.Recipient)
	if errors.Cause(err) == types.Err404 {
		// No established session, now check if we have a proposed session
		_, err = t.hushProto.store.OutgoingIndividualSessionProposalByUsersAndType(intent.SessionType, identity.Address(), intent.Recipient)
		if errors.Cause(err) == types.Err404 {
			t.hushProto.Debugf("no individual session exists with %v, proposing epoch 0", intent.Recipient)

			err := t.hushProto.ProposeNextIndividualSession(ctx, intent.SessionType, intent.Recipient)
			if err != nil {
				t.hushProto.Errorf("while proposing next individual session with %v: %v", intent.Recipient, err)
				return true
			}

		} else if err != nil {
			t.hushProto.Errorf("while looking up session with ID: %v", err)
			return true
		}
		// Retry later once we have an established session
		return true

	} else if err != nil {
		t.hushProto.Errorf("while looking up session with ID: %v", err)
		return true
	}

	sessionHash, err := session.Hash()
	if err != nil {
		t.hushProto.Errorf("while hashing latest session %v: %v", session.SessionID, err)
		return true
	}
	t.hushProto.Successf("sending message (sessionHash=%v)", sessionHash)

	drsession, err := doubleratchet.Load(session.SessionID.Bytes(), t.hushProto.store.RatchetSessionStore())
	if err != nil {
		t.hushProto.Errorf("while loading doubleratchet session %v: %v", session.SessionID, err)
		return true
	}

	msg, err := drsession.RatchetEncrypt(intent.Plaintext, nil)
	if err != nil {
		t.hushProto.Errorf("while encrypting outgoing individual message (sessionID=%v): %v", session.SessionID, err)
		return true
	}

	err = findPeerByAddress(ctx, t.hushProto.peerStore, t.hushProto.transports, intent.Recipient, func(recipient HushPeerConn) error {
		err := recipient.EnsureConnected(ctx)
		if err != nil {
			return err
		}
		defer recipient.Close()

		err = recipient.SendHushIndividualMessage(ctx, IndividualMessageFromDoubleRatchetMessage(msg, sessionHash))
		if err != nil {
			return err
		}
		t.hushProto.Debugf("sent individual message to %v", intent.Recipient)

		err = t.hushProto.store.DeleteOutgoingIndividualMessageIntent(intent)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.hushProto.Errorf("while sending outgoing individual message (sessionID=%v): %v", session.SessionID, err)
		return true
	}
	return false
}

type decryptIncomingIndividualMessagesTask struct {
	process.PeriodicTask
	log.Logger
	hushProto  *hushProtocol
	store      Store
	peerStore  swarm.PeerStore
	keyStore   identity.KeyStore
	transports map[string]HushTransport
}

func NewDecryptIncomingIndividualMessagesTask(
	interval time.Duration,
	hushProto *hushProtocol,
) *decryptIncomingIndividualMessagesTask {
	t := &decryptIncomingIndividualMessagesTask{
		Logger:    log.NewLogger(ProtocolName),
		hushProto: hushProto,
	}
	t.PeriodicTask = *process.NewPeriodicTask("DecryptIncomingIndividualMessagesTask", interval, t.decryptIncomingIndividualMessages)
	return t
}

func (t *decryptIncomingIndividualMessagesTask) decryptIncomingIndividualMessages(ctx context.Context) {
	messages, err := t.hushProto.store.IncomingIndividualMessages()
	if err != nil {
		t.Errorf("while fetching incoming messages from database: %v", err)
		return
	}

	for _, msg := range messages {
		name := fmt.Sprintf("decrypt incoming message")
		msg := msg

		t.Process.Go(ctx, name, func(ctx context.Context) {
			t.decryptIncomingIndividualMessage(ctx, msg)
		})
	}
}

func (t *decryptIncomingIndividualMessagesTask) decryptIncomingIndividualMessage(ctx context.Context, msg IndividualMessage) {
	sessionID, err := t.hushProto.store.IndividualSessionIDBySessionHash(msg.SessionHash)
	if err != nil {
		t.Errorf("while looking up session with ID: %v", err)
		return
	}

	t.Debugf("decrypting incoming individual message (sessionID=%v)", sessionID)

	session, err := doubleratchet.Load(sessionID.Bytes(), t.hushProto.store.RatchetSessionStore())
	if err != nil {
		t.Errorf("while loading doubleratchet session %v: %v", sessionID, err)
		return
	}
	plaintext, err := session.RatchetDecrypt(msg.ToDoubleRatchetMessage(), nil)
	if err != nil {
		t.Errorf("while decrypting incoming individual message (sessionID=%v): %v", sessionID, err)
		return
	}
	identity, err := t.hushProto.keyStore.DefaultPublicIdentity()
	if err != nil {
		t.Errorf("while fetching identity from key store: %v", err)
		return
	}

	var sender types.Address
	if sessionID.AliceAddr == identity.Address() {
		sender = sessionID.BobAddr
	} else {
		sender = sessionID.AliceAddr
	}

	t.Successf("decrypted incoming individual message (sessionID=%v): %v", sessionID, string(plaintext))

	t.hushProto.notifyIndividualMessageDecryptedListeners(sessionID.SessionType, sessionID, sender, plaintext)

	err = t.hushProto.store.DeleteIncomingIndividualMessage(msg)
	if err != nil {
		t.Errorf("while deleting incoming individual message (sessionID=%v): %v", sessionID, err)
		return
	}
}

func findPeerByAddress(
	ctx context.Context,
	peerStore swarm.PeerStore,
	transports map[string]HushTransport,
	peerAddr types.Address,
	fn func(hushPeerConn HushPeerConn) error,
) error {
	pds := peerStore.PeersWithAddress(peerAddr)
	if len(pds) == 0 {
		return errors.Errorf("not authenticated with %v", peerAddr)
	}

	for _, peer := range pds {
		if !peer.Dialable() || !peer.Ready() {
			continue
		}

		for _, tpt := range transports {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			peerConn, err := tpt.NewPeerConn(ctx, peer.DialInfo().DialAddr)
			if err != nil {
				continue
			}
			hushPeerConn, is := peerConn.(HushPeerConn)
			if !is {
				continue
			}

			err = fn(hushPeerConn)
			if err != nil {
				log.NewLogger("find peer by address").Errorf("while finding peer: %v", err) // @@TODO
				continue
			}
			return nil
		}
	}
	return errors.Errorf("protohush failed to find %v", peerAddr)
}
