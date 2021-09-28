package protohush

import (
	"bytes"

	"redwood.dev/swarm/protohush/pb"
	"redwood.dev/types"
)

type SharedKey = pb.SharedKey
type IndividualSessionID = pb.IndividualSessionID
type IndividualSessionProposal = pb.IndividualSessionProposal
type IndividualSessionApproval = pb.IndividualSessionApproval
type DHPair = pb.DHPair
type DHPubkeyAttestation = pb.DHPubkeyAttestation
type IndividualMessage = pb.IndividualMessage
type IndividualMessageHeader = pb.IndividualMessage_Header

// type GroupSession = pb.GroupSession
// type GroupMessage = pb.GroupMessage

var GenerateSharedKey = pb.GenerateSharedKey
var IndividualMessageFromDoubleRatchetMessage = pb.IndividualMessageFromDoubleRatchetMessage

type EncryptedIndividualSessionProposal struct {
	AliceAddr         types.Address `tree:"aliceAddr"`
	EncryptedProposal []byte        `tree:"encryptedProposal"`
}

func (p EncryptedIndividualSessionProposal) Hash() types.Hash {
	return types.HashBytes(append(p.AliceAddr.Bytes(), p.EncryptedProposal...))
}

type IndividualMessageIntent struct {
	ID          types.ID      `tree:"id"`
	SessionType string        `tree:"sessionType"`
	Recipient   types.Address `tree:"recipient"`
	Plaintext   []byte        `tree:"plaintext"`
}

func addrsSorted(alice, bob types.Address) (types.Address, types.Address) {
	if bytes.Compare(alice.Bytes(), bob.Bytes()) < 0 {
		return alice, bob
	}
	return bob, alice
}
