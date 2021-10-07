package process

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"redwood.dev/utils"
)

type ID interface {
	ID() string
}

type Pool interface {
	Add(item interface{})
	Get(ctx context.Context) (interface{}, error)
	Return(item interface{})
}

type PoolUniqueID interface{}

type pool struct {
	Process
	itemsAvailable *utils.Mailbox
	chItems        chan interface{}
}

func NewPool() *pool {
	return &pool{
		Process:        *New("pool"),
		itemsAvailable: utils.NewMailbox(0),
		chItems:        make(chan interface{}),
	}
}

func (p *pool) Start() error {
	err := p.Process.Start()
	if err != nil {
		return err
	}
	p.Process.Go(nil, "deliverAvailableItems", p.deliverAvailableItems)
	return nil
}

func (p *pool) deliverAvailableItems(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.itemsAvailable.Notify():
			for _, item := range p.itemsAvailable.RetrieveAll() {
				select {
				case <-ctx.Done():
					return
				case p.chItems <- item:
				}
			}
		}
	}
}

func (p *pool) Get(ctx context.Context) (interface{}, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case item := <-p.chItems:
			return item, nil
		}
	}
}

func (p *pool) Add(item interface{}) {
	p.itemsAvailable.Deliver(item)
}

func (p *pool) Return(item interface{}) {
	p.Add(item)
}

type dedupeActivePool struct {
	Pool
	activeItems   map[PoolUniqueID][]DedupeActivePoolUniqueIDer
	activeItemsMu sync.RWMutex
}

type DedupeActivePoolUniqueIDer interface {
	DedupeActiveUniqueID() PoolUniqueID
}

func NewDedupeActivePool(innerPool Pool) *dedupeActivePool {
	return &dedupeActivePool{
		Pool:        innerPool,
		activeItems: make(map[PoolUniqueID][]DedupeActivePoolUniqueIDer),
	}
}

func (p *dedupeActivePool) Get(ctx context.Context) (_ interface{}, err error) {
	for {
		x, err := p.Pool.Get(ctx)
		if err != nil {
			return nil, err
		}
		item := x.(DedupeActivePoolUniqueIDer)

		alreadyActive := p.setItemActive(item, true)
		if alreadyActive {
			continue
		}
		return item, nil
	}
}

func (p *dedupeActivePool) Add(item interface{}) {
	p.Pool.Add(item)
}

func (p *dedupeActivePool) Return(item interface{}) {
	p.setItemActive(item.(DedupeActivePoolUniqueIDer), false)
	p.Pool.Return(item)
}

func (p *dedupeActivePool) setItemActive(item DedupeActivePoolUniqueIDer, active bool) (alreadyActive bool) {
	p.activeItemsMu.Lock()
	defer p.activeItemsMu.Unlock()

	_, alreadyActive = p.activeItems[item.DedupeActiveUniqueID()]
	if alreadyActive {
		if active {
			p.activeItems[item.DedupeActiveUniqueID()] = append(p.activeItems[item.DedupeActiveUniqueID()], item)
		} else {
			items := p.activeItems[item.DedupeActiveUniqueID()]
			delete(p.activeItems, item.DedupeActiveUniqueID())
			for _, item := range items[1:] { // Don't re-process the initial item
				p.Pool.Return(item)
			}
		}
	} else {
		if active {
			p.activeItems[item.DedupeActiveUniqueID()] = append(p.activeItems[item.DedupeActiveUniqueID()], item)
		} else {
			panic("invariant violation")
		}
	}
	return alreadyActive
}

type blacklistPool struct {
	Pool
	blacklist   map[PoolUniqueID]struct{}
	blacklistMu sync.RWMutex
}

type BlacklistPoolUniqueIDer interface {
	BlacklistUniqueID() PoolUniqueID
}

func NewBlacklistPool(innerPool Pool) *blacklistPool {
	return &blacklistPool{
		Pool:      innerPool,
		blacklist: make(map[PoolUniqueID]struct{}),
	}
}

func (p *blacklistPool) Get(ctx context.Context) (_ interface{}, err error) {
	for {
		x, err := p.Pool.Get(ctx)
		if err != nil {
			return nil, err
		}
		item := x.(BlacklistPoolUniqueIDer)
		if !p.isBlacklisted(item.BlacklistUniqueID()) {
			return item, nil
		}
	}
}

func (p *blacklistPool) Add(item interface{}) {
	if !p.isBlacklisted(item.(BlacklistPoolUniqueIDer).BlacklistUniqueID()) {
		p.Pool.Add(item)
	}
}

func (p *blacklistPool) Return(item interface{}) {
	if !p.isBlacklisted(item.(BlacklistPoolUniqueIDer).BlacklistUniqueID()) {
		p.Pool.Return(item)
	}
}

func (p *blacklistPool) Blacklist(id PoolUniqueID) {
	p.blacklistMu.Lock()
	defer p.blacklistMu.Unlock()
	p.blacklist[id] = struct{}{}
}

func (p *blacklistPool) isBlacklisted(id PoolUniqueID) bool {
	p.blacklistMu.RLock()
	defer p.blacklistMu.RUnlock()
	_, exists := p.blacklist[id]
	return exists
}

type retryPool struct {
	Process
	Pool
	retryInterval        time.Duration
	itemsAwaitingRetry   map[PoolUniqueID]retryPoolEntry
	itemsAwaitingRetryMu sync.Mutex
	forceRetry           *utils.Mailbox
}

type RetryPoolUniqueIDer interface {
	RetryUniqueID() PoolUniqueID
}

type retryPoolEntry struct {
	item RetryPoolUniqueIDer
	when time.Time
}

func NewRetryPool(innerPool Pool, retryInterval time.Duration) *retryPool {
	return &retryPool{
		Process:            *New("retry pool"),
		Pool:               innerPool,
		retryInterval:      retryInterval,
		itemsAwaitingRetry: make(map[PoolUniqueID]retryPoolEntry),
		forceRetry:         utils.NewMailbox(0),
	}
}

func (p *retryPool) Start() error {
	err := p.Process.Start()
	if err != nil {
		return err
	}
	p.Process.Go(nil, "handleItemsAwaitingRetry", p.handleItemsAwaitingRetry)
	return nil
}

func (p *retryPool) Get(ctx context.Context) (_ interface{}, err error) {
	return p.Pool.Get(ctx)
}

func (p *retryPool) Add(item interface{}) {
	p.Pool.Add(item)
}

func (p *retryPool) Return(item interface{}) {
	p.itemsAwaitingRetryMu.Lock()
	defer p.itemsAwaitingRetryMu.Unlock()
	_, exists := p.itemsAwaitingRetry[item.(RetryPoolUniqueIDer).RetryUniqueID()]
	if !exists {
		p.Pool.Return(item)
	}
}

func (p *retryPool) RetryLater(item RetryPoolUniqueIDer, when time.Time) {
	p.itemsAwaitingRetryMu.Lock()
	defer p.itemsAwaitingRetryMu.Unlock()
	p.itemsAwaitingRetry[item.RetryUniqueID()] = retryPoolEntry{item, when}
}

func (p *retryPool) ForceRetry(uniqueID PoolUniqueID) {
	p.forceRetry.Deliver(uniqueID)
}

func (p *retryPool) handleItemsAwaitingRetry(ctx context.Context) {
	ticker := time.NewTicker(p.retryInterval)
	for {
		select {
		case <-p.Process.Done():
			return
		case <-ctx.Done():
			return

		case <-ticker.C:
			func() {
				p.itemsAwaitingRetryMu.Lock()
				defer p.itemsAwaitingRetryMu.Unlock()

				itemsAwaitingRetry := p.itemsAwaitingRetry
				p.itemsAwaitingRetry = make(map[PoolUniqueID]retryPoolEntry)

				now := time.Now()

				for _, entry := range itemsAwaitingRetry {
					if entry.when.Before(now) {
						p.Pool.Return(entry.item)
					} else {
						p.itemsAwaitingRetry[entry.item.RetryUniqueID()] = entry
					}
				}
			}()

		case <-p.forceRetry.Notify():
			for _, x := range p.forceRetry.RetrieveAll() {
				id := x.(PoolUniqueID)

				var entry retryPoolEntry
				var exists bool
				func() {
					p.itemsAwaitingRetryMu.Lock()
					defer p.itemsAwaitingRetryMu.Unlock()
					entry, exists = p.itemsAwaitingRetry[id]
					delete(p.itemsAwaitingRetry, id)
				}()
				if exists {
					p.Pool.Add(entry.item)
				}
			}
		}
	}
}

type semaphorePool struct {
	Pool
	sem *semaphore.Weighted
}

func NewSemaphorePool(innerPool Pool, concurrency uint64) *semaphorePool {
	return &semaphorePool{
		Pool: innerPool,
		sem:  semaphore.NewWeighted(int64(concurrency)),
	}
}

func (p *semaphorePool) Get(ctx context.Context) (_ interface{}, err error) {
	err = p.sem.Acquire(ctx, 1)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			p.sem.Release(1)
		}
	}()

	return p.Pool.Get(ctx)
}

func (p *semaphorePool) Add(item interface{}) {
	p.Pool.Add(item)
}

func (p *semaphorePool) Return(item interface{}) {
	p.sem.Release(1)
	p.Pool.Return(item)
}

type workPool struct {
	Process
	pool             *pool
	dedupeActivePool *dedupeActivePool
	blacklistPool    *blacklistPool
	retryPool        *retryPool
	semaphorePool    *semaphorePool
}

func NewWorkPool(name string, concurrency uint64, retryInterval time.Duration) *workPool {
	var (
		pool             = NewPool()
		retryPool        = NewRetryPool(pool, retryInterval)
		dedupeActivePool = NewDedupeActivePool(retryPool)
		blacklistPool    = NewBlacklistPool(dedupeActivePool)
		semaphorePool    = NewSemaphorePool(blacklistPool, concurrency)
	)
	return &workPool{
		Process:          *New(name),
		pool:             pool,
		dedupeActivePool: dedupeActivePool,
		blacklistPool:    blacklistPool,
		retryPool:        retryPool,
		semaphorePool:    semaphorePool,
	}
}

func (p *workPool) Start() error {
	err := p.Process.Start()
	if err != nil {
		return err
	}
	err = p.Process.SpawnChild(nil, p.pool)
	if err != nil {
		return err
	}
	return p.Process.SpawnChild(nil, p.retryPool)
}

func (p *workPool) Get(ctx context.Context) (interface{}, error) {
	return p.semaphorePool.Get(ctx)
}

func (p *workPool) Add(item interface{}) {
	p.semaphorePool.Add(item)
}

func (p *workPool) Return(item interface{}) {
	p.semaphorePool.Return(item)
}

func (p *workPool) Blacklist(id PoolUniqueID) {
	p.blacklistPool.Blacklist(id)
}

func (p *workPool) RetryLater(item RetryPoolUniqueIDer, when time.Time) {
	p.retryPool.RetryLater(item, when)
}

func (p *workPool) ForceRetry(id PoolUniqueID) {
	p.retryPool.ForceRetry(id)
}
