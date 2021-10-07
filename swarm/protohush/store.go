package protohush

import (
	"github.com/status-im/doubleratchet"

	"redwood.dev/types"
	"redwood.dev/utils"
)

type Store interface {
	// Diffie-Hellman keys
	EnsureDHPair() (DHPair, error)
	DHPubkeyAttestations() ([]DHPubkeyAttestation, error)
	LatestDHPubkeyFor(addr types.Address) (DHPubkeyAttestation, error)
	SaveDHPubkeyAttestations(attestations []DHPubkeyAttestation) error

	// Individual session handshake protocol
	// OutgoingIndividualSessionProposals() (map[types.Hash]IndividualSessionProposal, error)
	OutgoingIndividualSessionProposalHashes() (utils.HashSet, error)
	OutgoingIndividualSessionProposalByHash(proposalHash types.Hash) (IndividualSessionProposal, error)
	OutgoingIndividualSessionProposalsForUsers(aliceAddr, bobAddr types.Address) ([]IndividualSessionProposal, error)
	OutgoingIndividualSessionProposalByUsersAndType(sessionType string, aliceAddr, bobAddr types.Address) (IndividualSessionProposal, error)
	SaveOutgoingIndividualSessionProposal(proposal IndividualSessionProposal) error
	DeleteOutgoingIndividualSessionProposal(sessionHash types.Hash) error

	IncomingIndividualSessionProposals() (map[types.Hash]EncryptedIndividualSessionProposal, error)
	IncomingIndividualSessionProposal(hash types.Hash) (EncryptedIndividualSessionProposal, error)
	SaveIncomingIndividualSessionProposal(proposal EncryptedIndividualSessionProposal) error
	DeleteIncomingIndividualSessionProposal(proposal EncryptedIndividualSessionProposal) error

	OutgoingIndividualSessionApprovals() (map[types.Address]map[types.Hash]IndividualSessionApproval, error)
	OutgoingIndividualSessionApprovalsForUser(aliceAddr types.Address) ([]IndividualSessionApproval, error)
	OutgoingIndividualSessionApproval(aliceAddr types.Address, proposalHash types.Hash) (IndividualSessionApproval, error)
	SaveOutgoingIndividualSessionApproval(sender types.Address, approval IndividualSessionApproval) error
	DeleteOutgoingIndividualSessionApproval(aliceAddr types.Address, proposalHash types.Hash) error

	// Established individual sessions
	LatestIndividualSessionWithUsers(sessionType string, aliceAddr, bobAddr types.Address) (IndividualSessionProposal, error)
	IndividualSessionByID(id IndividualSessionID) (IndividualSessionProposal, error)
	IndividualSessionIDBySessionHash(hash types.Hash) (IndividualSessionID, error)
	SaveApprovedIndividualSession(session IndividualSessionProposal) error

	// Individual messages
	OutgoingIndividualMessageIntents() ([]IndividualMessageIntent, error)
	OutgoingIndividualMessageIntentIDsForTypeAndRecipient(sessionType string, recipient types.Address) (utils.IDSet, error)
	OutgoingIndividualMessageIntent(sessionType string, recipient types.Address, id types.ID) (IndividualMessageIntent, error)
	SaveOutgoingIndividualMessageIntent(intent IndividualMessageIntent) error
	DeleteOutgoingIndividualMessageIntent(intent IndividualMessageIntent) error

	IncomingIndividualMessages() ([]IndividualMessage, error)
	SaveIncomingIndividualMessage(sender types.Address, msg IndividualMessage) error
	DeleteIncomingIndividualMessage(msg IndividualMessage) error

	// Group messages
	OutgoingGroupMessageIntent(sessionType, id string) (GroupMessageIntent, error)
	SaveOutgoingGroupMessageIntent(intent GroupMessageIntent) error
	DeleteOutgoingGroupMessageIntent(sessionType, id string) error
	// SaveOutgoingGroupMessageForRecipients(id types.ID, msg GroupMessage, recipients []types.Address) error
	// OutgoingGroupMessageIDsForRecipient(recipient types.Address) (utils.IDSet, error)
	// OutgoingGroupMessageForRecipient(id types.ID, recipient types.Address) (GroupMessage, error)
	// DeleteOutgoingGroupMessageForRecipient(id types.ID, recipient types.Address) error

	IncomingGroupMessages() ([]GroupMessage, error)
	SaveIncomingGroupMessage(sender types.Address, msg GroupMessage) error
	DeleteIncomingGroupMessage(msg GroupMessage) error

	DebugPrint()

	//--------------------

	// LatestGroupSessionWithIDHash(msg.GroupSessionIDHash)

	// SessionExists(sessionID IndividualSessionID) (bool, error)
	// LatestIndividualSessionIDWithUsers(alice, bob types.Address) (IndividualSessionID, error)
	// LatestGroupSessionWithIDHash(sessionIDHash types.Hash) (GroupSession, error)

	// OutgoingIndividualSessionProposals() ([]IndividualSession, error)
	// OutgoingIndividualSessionProposal(proposalHash types.Hash) (IndividualSession, error)
	// SaveOutgoingIndividualSessionProposal(proposal IndividualSession) error
	// DeleteOutgoingIndividualSessionProposal(proposalHash types.Hash) error

	// OutgoingSessionApprovals() (map[types.Address][]SessionApproval, error)
	// SaveOutgoingSessionApproval(bobAddr types.Address, approval SessionApproval) error
	// DeleteOutgoingSessionApproval(bobAddr types.Address, proposalHash types.Hash) error

	// // SaveSharedKey(sessionID SessionID, sharedKey SharedKey) error

	// EnsureDHPair() (DHPair, error)
	// // MarkDHPubkeyAsNeeded(bobAddr types.Address) error
	// // DHPubkeysNeeded() ([]types.Address, error)
	// DHPubkeyAttestations() ([]DHPubkeyAttestation, error)
	// SaveDHPubkeyAttestations(attestations []DHPubkeyAttestation) error

	// EnqueueIndividualMessage(recipient types.Address, plaintext []byte) error

	// // DHPair(sessionID SessionID) (DHPair, error)
	// // SaveDHPair(sessionID SessionID, dhPair DHPair) error

	// // DHPubkey(sessionID SessionID) (doubleratchet.Key, error)

	RatchetSessionStore() RatchetSessionStore
	RatchetKeyStore() RatchetKeyStore
}

type RatchetSessionStore interface {
	doubleratchet.SessionStorage
}

type RatchetKeyStore interface {
	doubleratchet.KeysStorage
}
