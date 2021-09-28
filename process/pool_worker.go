package process

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/multierr"
)

type PoolWorker interface {
	Interface
	Add(item PoolWorkerItem)
	ForceRetry(uniqueID PoolUniqueID)
}

type PoolWorkerItem interface {
	DedupeActivePoolUniqueIDer
	BlacklistPoolUniqueIDer
	RetryPoolUniqueIDer
	Work(ctx context.Context) (retry bool)
}

type PoolWorkerScheduler interface {
	CheckForRetriesInterval() time.Duration
	RetryWhen(item PoolWorkerItem) time.Time
}

type poolWorker struct {
	Process
	concurrency uint64
	pool        *workPool
	scheduler   PoolWorkerScheduler
}

func NewPoolWorker(name string, concurrency uint64, scheduler PoolWorkerScheduler) *poolWorker {
	return &poolWorker{
		Process:     *New(name),
		concurrency: concurrency,
		pool:        NewWorkPool(name, concurrency, scheduler.CheckForRetriesInterval()),
		scheduler:   scheduler,
	}
}

func (w *poolWorker) Start() error {
	err := w.Process.Start()
	if err != nil {
		return err
	}

	err = w.Process.SpawnChild(nil, w.pool)
	if err != nil {
		return err
	}

	for i := uint64(0); i < w.concurrency; i++ {
		w.Process.Go(nil, fmt.Sprintf("worker %v", i), func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				x, err := w.pool.Get(ctx)
				if err != nil {
					return
				}
				item := x.(PoolWorkerItem)

				fmt.Println("WORK", item.(ID).ID())
				retry := item.Work(ctx)
				fmt.Println("  -", item.(ID).ID(), retry, w.scheduler.RetryWhen(item))
				if retry {
					w.pool.RetryLater(item, w.scheduler.RetryWhen(item))
				} else {
					w.pool.Blacklist(item.BlacklistUniqueID())
				}
				w.pool.Return(item)
			}
		})
	}
	return nil
}

func (w *poolWorker) Close() error {
	return multierr.Append(
		w.Process.Close(),
		w.pool.Close(),
	)
}

func (w *poolWorker) Add(item PoolWorkerItem) {
	w.pool.Add(item)
}

func (w *poolWorker) ForceRetry(id PoolUniqueID) {
	w.pool.ForceRetry(id)
}

type StaticScheduler struct {
	checkForRetriesInterval time.Duration
	retryAfter              time.Duration
}

func NewStaticScheduler(checkForRetriesInterval time.Duration, retryAfter time.Duration) StaticScheduler {
	return StaticScheduler{checkForRetriesInterval, retryAfter}
}

func (s StaticScheduler) CheckForRetriesInterval() time.Duration { return s.checkForRetriesInterval }
func (s StaticScheduler) RetryWhen(item PoolWorkerItem) time.Time {
	return time.Now().Add(s.retryAfter)
}
