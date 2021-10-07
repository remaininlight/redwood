package protohush

import (
	"context"
	"sync"
)

type (
	BaseHushTransport struct {
		muIncomingDHPubkeyAttestationsCallbacks      sync.RWMutex
		muIncomingIndividualSessionProposalCallbacks sync.RWMutex
		muIncomingIndividualSessionApprovalCallbacks sync.RWMutex
		muIncomingIndividualMessageCallbacks         sync.RWMutex
		muIncomingGroupMessageCallbacks              sync.RWMutex

		incomingDHPubkeyExchangeCallbacks          []IncomingDHPubkeyAttestationsCallback
		incomingIndividualSessionProposalCallbacks []IncomingIndividualSessionProposalCallback
		incomingIndividualSessionApprovalCallbacks []IncomingIndividualSessionApprovalCallback
		incomingIndividualMessageCallbacks         []IncomingIndividualMessageCallback
		incomingGroupMessageCallbacks              []IncomingGroupMessageCallback
	}

	IncomingDHPubkeyAttestationsCallback      func(ctx context.Context, attestations []DHPubkeyAttestation, peer HushPeerConn)
	IncomingIndividualSessionProposalCallback func(ctx context.Context, encryptedProposal []byte, alice HushPeerConn)
	IncomingIndividualSessionApprovalCallback func(ctx context.Context, approval IndividualSessionApproval, bob HushPeerConn)
	IncomingIndividualMessageCallback         func(ctx context.Context, msg IndividualMessage, peer HushPeerConn)
	IncomingGroupMessageCallback              func(ctx context.Context, msg GroupMessage, peer HushPeerConn)
)

func (t *BaseHushTransport) OnIncomingDHPubkeyAttestations(handler IncomingDHPubkeyAttestationsCallback) {
	t.muIncomingDHPubkeyAttestationsCallbacks.Lock()
	defer t.muIncomingDHPubkeyAttestationsCallbacks.Unlock()
	t.incomingDHPubkeyExchangeCallbacks = append(t.incomingDHPubkeyExchangeCallbacks, handler)
}

func (t *BaseHushTransport) HandleIncomingDHPubkeyAttestations(ctx context.Context, attestations []DHPubkeyAttestation, peerConn HushPeerConn) {
	t.muIncomingDHPubkeyAttestationsCallbacks.RLock()
	defer t.muIncomingDHPubkeyAttestationsCallbacks.RUnlock()

	var wg sync.WaitGroup
	wg.Add(len(t.incomingDHPubkeyExchangeCallbacks))
	for _, handler := range t.incomingDHPubkeyExchangeCallbacks {
		handler := handler
		go func() {
			defer wg.Done()
			handler(ctx, attestations, peerConn)
		}()
	}
	wg.Wait()
}

func (t *BaseHushTransport) OnIncomingIndividualSessionProposal(handler IncomingIndividualSessionProposalCallback) {
	t.muIncomingIndividualSessionProposalCallbacks.Lock()
	defer t.muIncomingIndividualSessionProposalCallbacks.Unlock()
	t.incomingIndividualSessionProposalCallbacks = append(t.incomingIndividualSessionProposalCallbacks, handler)
}

func (t *BaseHushTransport) HandleIncomingIndividualSessionProposal(ctx context.Context, encryptedProposal []byte, alice HushPeerConn) {
	t.muIncomingIndividualSessionProposalCallbacks.RLock()
	defer t.muIncomingIndividualSessionProposalCallbacks.RUnlock()

	var wg sync.WaitGroup
	wg.Add(len(t.incomingIndividualSessionProposalCallbacks))
	for _, handler := range t.incomingIndividualSessionProposalCallbacks {
		handler := handler
		go func() {
			defer wg.Done()
			handler(ctx, encryptedProposal, alice)
		}()
	}
	wg.Wait()
}

func (t *BaseHushTransport) OnIncomingIndividualSessionApproval(handler IncomingIndividualSessionApprovalCallback) {
	t.muIncomingIndividualSessionApprovalCallbacks.Lock()
	defer t.muIncomingIndividualSessionApprovalCallbacks.Unlock()
	t.incomingIndividualSessionApprovalCallbacks = append(t.incomingIndividualSessionApprovalCallbacks, handler)
}

func (t *BaseHushTransport) HandleIncomingIndividualSessionApproval(ctx context.Context, approval IndividualSessionApproval, bob HushPeerConn) {
	t.muIncomingIndividualSessionApprovalCallbacks.RLock()
	defer t.muIncomingIndividualSessionApprovalCallbacks.RUnlock()

	var wg sync.WaitGroup
	wg.Add(len(t.incomingIndividualSessionApprovalCallbacks))
	for _, handler := range t.incomingIndividualSessionApprovalCallbacks {
		handler := handler
		go func() {
			defer wg.Done()
			handler(ctx, approval, bob)
		}()
	}
	wg.Wait()
}

func (t *BaseHushTransport) OnIncomingIndividualMessage(handler IncomingIndividualMessageCallback) {
	t.muIncomingIndividualMessageCallbacks.Lock()
	defer t.muIncomingIndividualMessageCallbacks.Unlock()
	t.incomingIndividualMessageCallbacks = append(t.incomingIndividualMessageCallbacks, handler)
}

func (t *BaseHushTransport) HandleIncomingIndividualMessage(ctx context.Context, msg IndividualMessage, peerConn HushPeerConn) {
	t.muIncomingIndividualMessageCallbacks.RLock()
	defer t.muIncomingIndividualMessageCallbacks.RUnlock()

	var wg sync.WaitGroup
	wg.Add(len(t.incomingIndividualMessageCallbacks))
	for _, handler := range t.incomingIndividualMessageCallbacks {
		handler := handler
		go func() {
			defer wg.Done()
			handler(ctx, msg, peerConn)
		}()
	}
	wg.Wait()
}

func (t *BaseHushTransport) OnIncomingGroupMessage(handler IncomingGroupMessageCallback) {
	t.muIncomingGroupMessageCallbacks.Lock()
	defer t.muIncomingGroupMessageCallbacks.Unlock()
	t.incomingGroupMessageCallbacks = append(t.incomingGroupMessageCallbacks, handler)
}

func (t *BaseHushTransport) HandleIncomingGroupMessage(ctx context.Context, msg GroupMessage, peerConn HushPeerConn) {
	t.muIncomingGroupMessageCallbacks.RLock()
	defer t.muIncomingGroupMessageCallbacks.RUnlock()

	var wg sync.WaitGroup
	wg.Add(len(t.incomingGroupMessageCallbacks))
	for _, handler := range t.incomingGroupMessageCallbacks {
		handler := handler
		go func() {
			defer wg.Done()
			handler(ctx, msg, peerConn)
		}()
	}
	wg.Wait()
}
