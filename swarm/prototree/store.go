package prototree

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"

	"redwood.dev/log"
	"redwood.dev/process"
	"redwood.dev/state"
	"redwood.dev/types"
	"redwood.dev/utils"
)

//go:generate mockery --name Store --output ./mocks/ --case=underscore
type Store interface {
	SubscribedStateURIs() utils.StringSet
	AddSubscribedStateURI(stateURI string) error
	RemoveSubscribedStateURI(stateURI string) error
	OnNewSubscribedStateURI(handler func(stateURI string)) (unsubscribe func())

	MaxPeersPerSubscription() uint64
	SetMaxPeersPerSubscription(max uint64) error

	TxSeenByPeer(deviceSpecificID, stateURI string, txID types.ID) bool
	MarkTxSeenByPeer(deviceSpecificID, stateURI string, txID types.ID) error
	PruneTxSeenRecordsOlderThan(threshold time.Duration) error

	EncryptedTx(stateURI string, txID types.ID) (EncryptedTx, error)
	SaveEncryptedTx(stateURI string, txID types.ID, etx EncryptedTx) error
}

type store struct {
	process.Process
	log.Logger

	db *state.DBTree

	data   storeData
	dataMu sync.RWMutex

	subscribedStateURIListeners   map[*subscribedStateURIListener]struct{}
	subscribedStateURIListenersMu sync.RWMutex
}

type subscribedStateURIListener struct {
	handler func(stateURI string)
}

type storeData struct {
	SubscribedStateURIs     utils.StringSet
	MaxPeersPerSubscription uint64
	TxsSeenByPeers          map[string]map[string]map[types.ID]uint64
	EncryptedTxs            map[string]map[types.ID]EncryptedTx
}

type storeDataCodec struct {
	SubscribedStateURIs     map[string]bool                         `tree:"subscribedStateURIs"`
	MaxPeersPerSubscription uint64                                  `tree:"maxPeersPerSubscription"`
	TxsSeenByPeers          map[string]map[string]map[string]uint64 `tree:"txsSeenByPeers"`
	EncryptedTxs            map[string]map[string]EncryptedTx       `tree:"encryptedTxs"`
}

var storeRootKeypath = state.Keypath("prototree")

func NewStore(db *state.DBTree) (*store, error) {
	s := &store{
		Process:                     *process.New("prototree store"),
		Logger:                      log.NewLogger("prototree store"),
		db:                          db,
		subscribedStateURIListeners: make(map[*subscribedStateURIListener]struct{}),
	}
	s.Infof(0, "opening prototree store")
	err := s.loadData()
	return s, err
}

func (s *store) Start() error {
	err := s.Process.Start()
	if err != nil {
		return err
	}

	pruneTxsTask := NewPruneTxsTask(5*time.Minute, s)
	err = s.Process.SpawnChild(nil, pruneTxsTask)
	if err != nil {
		return err
	}
	return nil
}

func (s *store) loadData() error {
	node := s.db.State(false)
	defer node.Close()

	var codec storeDataCodec
	err := node.NodeAt(storeRootKeypath, nil).Scan(&codec)
	if errors.Cause(err) == types.Err404 {
		// do nothing
	} else if err != nil {
		return err
	}

	txsSeenByPeers := make(map[string]map[string]map[types.ID]uint64)
	for deviceSpecificID, x := range codec.TxsSeenByPeers {
		txsSeenByPeers[deviceSpecificID] = make(map[string]map[types.ID]uint64)
		for stateURI, y := range x {
			txsSeenByPeers[deviceSpecificID][stateURI] = make(map[types.ID]uint64)
			for txIDStr, whenSeen := range y {
				txID, err := types.IDFromHex(txIDStr)
				if err != nil {
					s.Errorf("while unmarshaling tx ID: %v", err)
					continue
				}
				txsSeenByPeers[deviceSpecificID][stateURI][txID] = whenSeen
			}
		}
	}

	var subscribedStateURIs []string
	for stateURI := range codec.SubscribedStateURIs {
		subscribedStateURIs = append(subscribedStateURIs, stateURI)
	}

	encryptedTxs := make(map[string]map[types.ID]EncryptedTx)
	for stateURI := range codec.EncryptedTxs {
		encryptedTxs[stateURI] = make(map[types.ID]EncryptedTx)
		for txIDStr, etx := range codec.EncryptedTxs[stateURI] {
			txID, err := types.IDFromHex(txIDStr)
			if err != nil {
				return err
			}
			encryptedTxs[stateURI][txID] = etx
		}
	}

	s.dataMu.Lock()
	defer s.dataMu.Unlock()
	s.data = storeData{
		SubscribedStateURIs:     utils.NewStringSet(subscribedStateURIs),
		MaxPeersPerSubscription: codec.MaxPeersPerSubscription,
		TxsSeenByPeers:          txsSeenByPeers,
		EncryptedTxs:            encryptedTxs,
	}
	return nil
}

// func (s *store) saveData() error {
// 	txsSeenByPeers := make(map[string]map[string]map[string]uint64)
// 	for deviceSpecificID, y := range s.data.TxsSeenByPeers {
// 		txsSeenByPeers[deviceSpecificID] = make(map[string]map[string]uint64)
// 		for stateURI, z := range y {
// 			txsSeenByPeers[deviceSpecificID][stateURI] = make(map[string]uint64)
// 			for txID, whenSeen := range z {
// 				txsSeenByPeers[deviceSpecificID][stateURI][txID.Hex()] = whenSeen
// 			}
// 		}
// 	}

// 	codec := storeDataCodec{
// 		SubscribedStateURIs:     s.data.SubscribedStateURIs.Slice(),
// 		MaxPeersPerSubscription: s.data.MaxPeersPerSubscription,
// 		TxsSeenByPeers:          txsSeenByPeers,
// 	}

// 	node := s.db.State(true)
// 	defer node.Close()

// 	err := node.Set(state.Keypath("prototree"), nil, codec)
// 	if err != nil {
// 		return err
// 	}
// 	return node.Save()
// }

func (s *store) SubscribedStateURIs() utils.StringSet {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()
	return s.data.SubscribedStateURIs.Copy()
}

func (s *store) AddSubscribedStateURI(stateURI string) error {
	func() {
		s.subscribedStateURIListenersMu.RLock()
		defer s.subscribedStateURIListenersMu.RUnlock()
		for listener := range s.subscribedStateURIListeners {
			listener.handler(stateURI)
		}
	}()

	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	s.data.SubscribedStateURIs.Add(stateURI)

	node := s.db.State(true)
	defer node.Close()

	err := node.Set(s.keypathForSubscribedStateURI(stateURI), nil, true)
	if err != nil {
		return err
	}
	return node.Save()
}

func (s *store) RemoveSubscribedStateURI(stateURI string) error {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	s.data.SubscribedStateURIs.Remove(stateURI)

	node := s.db.State(true)
	defer node.Close()

	err := node.Delete(s.keypathForSubscribedStateURI(stateURI), nil)
	if err != nil {
		return err
	}
	return node.Save()
}

func (s *store) keypathForSubscribedStateURI(stateURI string) state.Keypath {
	return storeRootKeypath.Pushs("subscribedStateURIs").Pushs(stateURI)
}

func (s *store) OnNewSubscribedStateURI(handler func(stateURI string)) (unsubscribe func()) {
	s.subscribedStateURIListenersMu.Lock()
	defer s.subscribedStateURIListenersMu.Unlock()

	listener := &subscribedStateURIListener{handler: handler}
	s.subscribedStateURIListeners[listener] = struct{}{}

	return func() {
		s.subscribedStateURIListenersMu.Lock()
		defer s.subscribedStateURIListenersMu.Unlock()
		delete(s.subscribedStateURIListeners, listener)
	}
}

func (s *store) MaxPeersPerSubscription() uint64 {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()
	return s.data.MaxPeersPerSubscription
}

func (s *store) SetMaxPeersPerSubscription(max uint64) error {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	s.data.MaxPeersPerSubscription = max

	node := s.db.State(true)
	defer node.Close()

	err := node.Set(s.keypathForMaxPeersPerSubscription(), nil, max)
	if err != nil {
		return err
	}
	return node.Save()
}

func (s *store) keypathForMaxPeersPerSubscription() state.Keypath {
	return storeRootKeypath.Pushs("maxPeersPerSubscription")
}

func (s *store) TxSeenByPeer(deviceSpecificID, stateURI string, txID types.ID) bool {
	if len(deviceSpecificID) == 0 {
		return false
	}

	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	if _, exists := s.data.TxsSeenByPeers[deviceSpecificID]; !exists {
		return false
	}
	if _, exists := s.data.TxsSeenByPeers[deviceSpecificID][stateURI]; !exists {
		return false
	}
	_, exists := s.data.TxsSeenByPeers[deviceSpecificID][stateURI][txID]
	return exists
}

func (s *store) MarkTxSeenByPeer(deviceSpecificID, stateURI string, txID types.ID) error {
	if len(deviceSpecificID) == 0 {
		return nil
	}

	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	now := uint64(time.Now().UTC().Unix())

	if _, exists := s.data.TxsSeenByPeers[deviceSpecificID]; !exists {
		s.data.TxsSeenByPeers[deviceSpecificID] = make(map[string]map[types.ID]uint64)
	}
	if _, exists := s.data.TxsSeenByPeers[deviceSpecificID][stateURI]; !exists {
		s.data.TxsSeenByPeers[deviceSpecificID][stateURI] = make(map[types.ID]uint64)
	}
	s.data.TxsSeenByPeers[deviceSpecificID][stateURI][txID] = now

	node := s.db.State(true)
	defer node.Close()

	err := node.Set(s.keypathForTxSeenByPeer(deviceSpecificID, stateURI, txID), nil, now)
	if err != nil {
		return err
	}

	return node.Save()
}

func (s *store) keypathForTxSeenByPeer(deviceSpecificID, stateURI string, txID types.ID) state.Keypath {
	return storeRootKeypath.Pushs("txsSeenByPeers").Pushs(deviceSpecificID).Pushs(stateURI).Pushs(txID.Hex())
}

type pruneTxsTask struct {
	process.PeriodicTask
	log.Logger

	interval time.Duration
	store    Store
}

func NewPruneTxsTask(
	interval time.Duration,
	store Store,
) *pruneTxsTask {
	t := &pruneTxsTask{
		Logger: log.NewLogger("prototree store"),
		store:  store,
	}
	t.PeriodicTask = *process.NewPeriodicTask("PruneTxsTask", interval, t.pruneTxs)
	return t
}

func (t *pruneTxsTask) pruneTxs(ctx context.Context) {
	err := t.store.PruneTxSeenRecordsOlderThan(t.interval)
	if err != nil {
		t.Errorf("while pruning prototree store txs: %v", err)
	}
}

func (s *store) PruneTxSeenRecordsOlderThan(threshold time.Duration) error {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	node := s.db.State(true)
	defer node.Close()

	s.Debugf("pruning prototree store txs")

	for deviceSpecificID, y := range s.data.TxsSeenByPeers {
		for stateURI, z := range y {
			for txID, whenSeen := range z {
				t := time.Unix(int64(whenSeen), 0)
				if t.Add(threshold).Before(time.Now().UTC()) {
					delete(s.data.TxsSeenByPeers[deviceSpecificID][stateURI], txID)

					err := node.Delete(s.keypathForTxSeenByPeer(deviceSpecificID, stateURI, txID), nil)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return node.Save()
}

func (s *store) EncryptedTx(stateURI string, txID types.ID) (EncryptedTx, error) {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	_, exists := s.data.EncryptedTxs[stateURI]
	if !exists {
		return EncryptedTx{}, types.Err404
	}
	etx, exists := s.data.EncryptedTxs[stateURI][txID]
	if !exists {
		return EncryptedTx{}, types.Err404
	}
	return etx, nil
}

func (s *store) SaveEncryptedTx(stateURI string, txID types.ID, etx EncryptedTx) error {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	_, exists := s.data.EncryptedTxs[stateURI]
	if !exists {
		s.data.EncryptedTxs[stateURI] = make(map[types.ID]EncryptedTx)
	}
	s.data.EncryptedTxs[stateURI][txID] = etx

	node := s.db.State(true)
	defer node.Close()

	err := node.Set(s.keypathForEncryptedTx(stateURI, txID), nil, etx)
	if err != nil {
		return err
	}
	return node.Save()
}

func (s *store) keypathForEncryptedTx(stateURI string, txID types.ID) state.Keypath {
	return storeRootKeypath.Pushs("encryptedTxs").Pushs(stateURI).Pushs(txID.Hex())
}
