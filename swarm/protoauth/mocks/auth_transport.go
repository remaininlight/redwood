// Code generated by mockery v2.8.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	process "redwood.dev/process"

	protoauth "redwood.dev/swarm/protoauth"

	swarm "redwood.dev/swarm"

	types "redwood.dev/types"
)

// AuthTransport is an autogenerated mock type for the AuthTransport type
type AuthTransport struct {
	mock.Mock
}

// Autoclose provides a mock function with given fields:
func (_m *AuthTransport) Autoclose() {
	_m.Called()
}

// AutocloseWithCleanup provides a mock function with given fields: closeFn
func (_m *AuthTransport) AutocloseWithCleanup(closeFn func()) {
	_m.Called(closeFn)
}

// Close provides a mock function with given fields:
func (_m *AuthTransport) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Ctx provides a mock function with given fields:
func (_m *AuthTransport) Ctx() context.Context {
	ret := _m.Called()

	var r0 context.Context
	if rf, ok := ret.Get(0).(func() context.Context); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(context.Context)
		}
	}

	return r0
}

// Done provides a mock function with given fields:
func (_m *AuthTransport) Done() <-chan struct{} {
	ret := _m.Called()

	var r0 <-chan struct{}
	if rf, ok := ret.Get(0).(func() <-chan struct{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan struct{})
		}
	}

	return r0
}

// Go provides a mock function with given fields: ctx, name, fn
func (_m *AuthTransport) Go(ctx context.Context, name string, fn func(context.Context)) <-chan struct{} {
	ret := _m.Called(ctx, name, fn)

	var r0 <-chan struct{}
	if rf, ok := ret.Get(0).(func(context.Context, string, func(context.Context)) <-chan struct{}); ok {
		r0 = rf(ctx, name, fn)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan struct{})
		}
	}

	return r0
}

// Name provides a mock function with given fields:
func (_m *AuthTransport) Name() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// NewChild provides a mock function with given fields: ctx, name
func (_m *AuthTransport) NewChild(ctx context.Context, name string) *process.Process {
	ret := _m.Called(ctx, name)

	var r0 *process.Process
	if rf, ok := ret.Get(0).(func(context.Context, string) *process.Process); ok {
		r0 = rf(ctx, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*process.Process)
		}
	}

	return r0
}

// NewPeerConn provides a mock function with given fields: ctx, dialAddr
func (_m *AuthTransport) NewPeerConn(ctx context.Context, dialAddr string) (swarm.PeerConn, error) {
	ret := _m.Called(ctx, dialAddr)

	var r0 swarm.PeerConn
	if rf, ok := ret.Get(0).(func(context.Context, string) swarm.PeerConn); ok {
		r0 = rf(ctx, dialAddr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(swarm.PeerConn)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, dialAddr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// OnChallengeIdentity provides a mock function with given fields: handler
func (_m *AuthTransport) OnChallengeIdentity(handler protoauth.ChallengeIdentityCallback) {
	_m.Called(handler)
}

// PeersClaimingAddress provides a mock function with given fields: ctx, address
func (_m *AuthTransport) PeersClaimingAddress(ctx context.Context, address types.Address) (<-chan protoauth.AuthPeerConn, error) {
	ret := _m.Called(ctx, address)

	var r0 <-chan protoauth.AuthPeerConn
	if rf, ok := ret.Get(0).(func(context.Context, types.Address) <-chan protoauth.AuthPeerConn); ok {
		r0 = rf(ctx, address)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan protoauth.AuthPeerConn)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, types.Address) error); ok {
		r1 = rf(ctx, address)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ProcessTree provides a mock function with given fields:
func (_m *AuthTransport) ProcessTree() map[string]interface{} {
	ret := _m.Called()

	var r0 map[string]interface{}
	if rf, ok := ret.Get(0).(func() map[string]interface{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]interface{})
		}
	}

	return r0
}

// SpawnChild provides a mock function with given fields: ctx, child
func (_m *AuthTransport) SpawnChild(ctx context.Context, child process.Spawnable) error {
	ret := _m.Called(ctx, child)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, process.Spawnable) error); ok {
		r0 = rf(ctx, child)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Start provides a mock function with given fields:
func (_m *AuthTransport) Start() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}