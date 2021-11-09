// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import (
	context "context"

	blob "redwood.dev/blob"

	crypto "redwood.dev/crypto"

	mock "github.com/stretchr/testify/mock"

	swarm "redwood.dev/swarm"

	time "time"

	types "redwood.dev/types"
)

// BlobPeerConn is an autogenerated mock type for the BlobPeerConn type
type BlobPeerConn struct {
	mock.Mock
}

// AddStateURI provides a mock function with given fields: stateURI
func (_m *BlobPeerConn) AddStateURI(stateURI string) error {
	ret := _m.Called(stateURI)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(stateURI)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Addresses provides a mock function with given fields:
func (_m *BlobPeerConn) Addresses() []types.Address {
	ret := _m.Called()

	var r0 []types.Address
	if rf, ok := ret.Get(0).(func() []types.Address); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.Address)
		}
	}

	return r0
}

// AnnouncePeers provides a mock function with given fields: ctx, peerDialInfos
func (_m *BlobPeerConn) AnnouncePeers(ctx context.Context, peerDialInfos []swarm.PeerDialInfo) error {
	ret := _m.Called(ctx, peerDialInfos)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, []swarm.PeerDialInfo) error); ok {
		r0 = rf(ctx, peerDialInfos)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Close provides a mock function with given fields:
func (_m *BlobPeerConn) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeviceUniqueID provides a mock function with given fields:
func (_m *BlobPeerConn) DeviceUniqueID() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// DialInfo provides a mock function with given fields:
func (_m *BlobPeerConn) DialInfo() swarm.PeerDialInfo {
	ret := _m.Called()

	var r0 swarm.PeerDialInfo
	if rf, ok := ret.Get(0).(func() swarm.PeerDialInfo); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(swarm.PeerDialInfo)
	}

	return r0
}

// Dialable provides a mock function with given fields:
func (_m *BlobPeerConn) Dialable() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Endpoint provides a mock function with given fields: dialInfo
func (_m *BlobPeerConn) Endpoint(dialInfo swarm.PeerDialInfo) (swarm.PeerEndpoint, bool) {
	ret := _m.Called(dialInfo)

	var r0 swarm.PeerEndpoint
	if rf, ok := ret.Get(0).(func(swarm.PeerDialInfo) swarm.PeerEndpoint); ok {
		r0 = rf(dialInfo)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(swarm.PeerEndpoint)
		}
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(swarm.PeerDialInfo) bool); ok {
		r1 = rf(dialInfo)
	} else {
		r1 = ret.Get(1).(bool)
	}

	return r0, r1
}

// Endpoints provides a mock function with given fields:
func (_m *BlobPeerConn) Endpoints() map[swarm.PeerDialInfo]swarm.PeerEndpoint {
	ret := _m.Called()

	var r0 map[swarm.PeerDialInfo]swarm.PeerEndpoint
	if rf, ok := ret.Get(0).(func() map[swarm.PeerDialInfo]swarm.PeerEndpoint); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[swarm.PeerDialInfo]swarm.PeerEndpoint)
		}
	}

	return r0
}

// EnsureConnected provides a mock function with given fields: ctx
func (_m *BlobPeerConn) EnsureConnected(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Failures provides a mock function with given fields:
func (_m *BlobPeerConn) Failures() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// FetchBlobChunk provides a mock function with given fields: sha3
func (_m *BlobPeerConn) FetchBlobChunk(sha3 types.Hash) ([]byte, error) {
	ret := _m.Called(sha3)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(types.Hash) []byte); ok {
		r0 = rf(sha3)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(types.Hash) error); ok {
		r1 = rf(sha3)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FetchBlobManifest provides a mock function with given fields: blobID
func (_m *BlobPeerConn) FetchBlobManifest(blobID blob.ID) (blob.Manifest, error) {
	ret := _m.Called(blobID)

	var r0 blob.Manifest
	if rf, ok := ret.Get(0).(func(blob.ID) blob.Manifest); ok {
		r0 = rf(blobID)
	} else {
		r0 = ret.Get(0).(blob.Manifest)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(blob.ID) error); ok {
		r1 = rf(blobID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LastContact provides a mock function with given fields:
func (_m *BlobPeerConn) LastContact() time.Time {
	ret := _m.Called()

	var r0 time.Time
	if rf, ok := ret.Get(0).(func() time.Time); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	return r0
}

// LastFailure provides a mock function with given fields:
func (_m *BlobPeerConn) LastFailure() time.Time {
	ret := _m.Called()

	var r0 time.Time
	if rf, ok := ret.Get(0).(func() time.Time); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	return r0
}

// PublicKeys provides a mock function with given fields: addr
func (_m *BlobPeerConn) PublicKeys(addr types.Address) (*crypto.SigningPublicKey, *crypto.AsymEncPubkey) {
	ret := _m.Called(addr)

	var r0 *crypto.SigningPublicKey
	if rf, ok := ret.Get(0).(func(types.Address) *crypto.SigningPublicKey); ok {
		r0 = rf(addr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*crypto.SigningPublicKey)
		}
	}

	var r1 *crypto.AsymEncPubkey
	if rf, ok := ret.Get(1).(func(types.Address) *crypto.AsymEncPubkey); ok {
		r1 = rf(addr)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*crypto.AsymEncPubkey)
		}
	}

	return r0, r1
}

// Ready provides a mock function with given fields:
func (_m *BlobPeerConn) Ready() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// RemainingBackoff provides a mock function with given fields:
func (_m *BlobPeerConn) RemainingBackoff() time.Duration {
	ret := _m.Called()

	var r0 time.Duration
	if rf, ok := ret.Get(0).(func() time.Duration); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(time.Duration)
	}

	return r0
}

// RemoveStateURI provides a mock function with given fields: stateURI
func (_m *BlobPeerConn) RemoveStateURI(stateURI string) error {
	ret := _m.Called(stateURI)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(stateURI)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SendBlobChunk provides a mock function with given fields: chunk, exists
func (_m *BlobPeerConn) SendBlobChunk(chunk []byte, exists bool) error {
	ret := _m.Called(chunk, exists)

	var r0 error
	if rf, ok := ret.Get(0).(func([]byte, bool) error); ok {
		r0 = rf(chunk, exists)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SendBlobManifest provides a mock function with given fields: m, exists
func (_m *BlobPeerConn) SendBlobManifest(m blob.Manifest, exists bool) error {
	ret := _m.Called(m, exists)

	var r0 error
	if rf, ok := ret.Get(0).(func(blob.Manifest, bool) error); ok {
		r0 = rf(m, exists)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetDeviceUniqueID provides a mock function with given fields: id
func (_m *BlobPeerConn) SetDeviceUniqueID(id string) error {
	ret := _m.Called(id)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StateURIs provides a mock function with given fields:
func (_m *BlobPeerConn) StateURIs() types.StringSet {
	ret := _m.Called()

	var r0 types.StringSet
	if rf, ok := ret.Get(0).(func() types.StringSet); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.StringSet)
		}
	}

	return r0
}

// Transport provides a mock function with given fields:
func (_m *BlobPeerConn) Transport() swarm.Transport {
	ret := _m.Called()

	var r0 swarm.Transport
	if rf, ok := ret.Get(0).(func() swarm.Transport); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(swarm.Transport)
		}
	}

	return r0
}

// UpdateConnStats provides a mock function with given fields: success
func (_m *BlobPeerConn) UpdateConnStats(success bool) {
	_m.Called(success)
}
