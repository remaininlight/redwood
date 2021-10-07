package protohush_test

import (
	"testing"

	"github.com/status-im/doubleratchet"
	"github.com/stretchr/testify/require"

	"redwood.dev/internal/testutils"
	"redwood.dev/swarm/protohush"
)

func TestBadgerStore_RatchetSessionStore(t *testing.T) {
	dbA := testutils.SetupDBTree(t)
	defer dbA.Close()

	dbB := testutils.SetupDBTree(t)
	defer dbB.Close()

	storeA := protohush.NewStore(dbA)
	storeB := protohush.NewStore(dbB)

	var (
		sk = protohush.SharedKey{
			0xeb, 0x8, 0x10, 0x7c, 0x33, 0x54, 0x0, 0x20,
			0xe9, 0x4f, 0x6c, 0x84, 0xe4, 0x39, 0x50, 0x5a,
			0x2f, 0x60, 0xbe, 0x81, 0xa, 0x78, 0x8b, 0xeb,
			0x1e, 0x2c, 0x9, 0x8d, 0x4b, 0x4d, 0xc1, 0x40,
		}

		aliceAddr = testutils.RandomAddress(t)
		bobAddr   = testutils.RandomAddress(t)

		sessionID = protohush.IndividualSessionID{
			SessionType: "foo",
			AliceAddr:   aliceAddr,
			BobAddr:     bobAddr,
			Epoch:       11,
		}

		session = protohush.IndividualSessionProposal{
			SessionID: sessionID,
			SharedKey: sk,
		}

		aliceMessage1 = []byte("hi bob")
		bobMessage1   = []byte("hi alice")
		aliceMessage2 = []byte("cool")
		bobMessage2   = []byte("neat")
	)

	_, err := storeA.EnsureDHPair()
	require.NoError(t, err)
	bobDHPair, err := storeB.EnsureDHPair()
	require.NoError(t, err)

	err = storeA.SaveApprovedIndividualSession(session)
	require.NoError(t, err)
	err = storeB.SaveApprovedIndividualSession(session)
	require.NoError(t, err)

	alice, err := doubleratchet.NewWithRemoteKey(sessionID.Bytes(), sk[:], bobDHPair.PublicKey(), storeA.RatchetSessionStore(), doubleratchet.WithKeysStorage(storeA.RatchetKeyStore()))
	require.NoError(t, err)

	bob, err := doubleratchet.New(sessionID.Bytes(), sk[:], bobDHPair, storeB.RatchetSessionStore(), doubleratchet.WithKeysStorage(storeB.RatchetKeyStore()))
	require.NoError(t, err)

	msg, err := alice.RatchetEncrypt(aliceMessage1, nil)
	require.NoError(t, err)

	m, err := bob.RatchetDecrypt(msg, nil)
	require.NoError(t, err)
	require.Equal(t, aliceMessage1, m)

	msg, err = bob.RatchetEncrypt(bobMessage1, nil)
	require.NoError(t, err)

	m, err = alice.RatchetDecrypt(msg, nil)
	require.NoError(t, err)
	require.Equal(t, bobMessage1, m)

	alice2, err := doubleratchet.Load(sessionID.Bytes(), storeA.RatchetSessionStore())
	require.NoError(t, err)

	bob2, err := doubleratchet.Load(sessionID.Bytes(), storeB.RatchetSessionStore())
	require.NoError(t, err)

	msg, err = alice2.RatchetEncrypt(aliceMessage2, nil)
	require.NoError(t, err)

	m, err = bob2.RatchetDecrypt(msg, nil)
	require.NoError(t, err)
	require.Equal(t, aliceMessage2, m)

	msg, err = bob2.RatchetEncrypt(bobMessage2, nil)
	require.NoError(t, err)

	m, err = alice2.RatchetDecrypt(msg, nil)
	require.NoError(t, err)
	require.Equal(t, bobMessage2, m)
}

func TestBadgerStore_RatchetKeyStore(t *testing.T) {
	db := testutils.SetupDBTree(t)
	defer db.Close()

	store := protohush.NewStore(db)
	rks := store.RatchetKeyStore()

	var (
		sessionID = []byte("foo bar")
		pubKey    = doubleratchet.Key("asdfasdfasdffdsafdsafdsa")
		msgNum    = uint(12345)
		mk        = doubleratchet.Key("xyzzy zork xyzzy zork xyzzy zork xyzzy zork")
		keySeqNum = uint(4665)
	)

	err := rks.Put(sessionID, pubKey, msgNum, mk, keySeqNum)
	require.NoError(t, err)

	mk2, ok, err := rks.Get(pubKey, msgNum)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, mk, mk2)
}
