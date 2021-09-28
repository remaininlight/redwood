package process_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"redwood.dev/internal/testutils"
	"redwood.dev/process"
)

func TestPoolWorker(t *testing.T) {

	makeItems := func() (item [5]*workItem) {
		item[0] = &workItem{id: "a", processed: testutils.NewAwaiter()}
		item[1] = &workItem{id: "b", processed: testutils.NewAwaiter()}
		item[2] = &workItem{id: "b2", processed: testutils.NewAwaiter()}
		item[3] = &workItem{id: "c", processed: testutils.NewAwaiter()}
		item[4] = &workItem{id: "d", processed: testutils.NewAwaiter()}
		return
	}

	// t.Run("doesn't process items that aren't ready", func(t *testing.T) {
	// 	t.Parallel()

	// 	var wg sync.WaitGroup
	// 	item := makeItems()

	// 	w := process.NewPoolWorker("", 2, 100*time.Millisecond)
	// 	err := w.Start()
	// 	require.NoError(t, err)
	// 	defer w.Close()

	// 	w.Add(item[0])
	// 	w.Add(item[1])
	// 	w.Add(item[2])
	// 	w.Add(item[3])
	// 	w.Add(item[4])

	// 	wg.Add(5)
	// 	go requireNeverHappened(t, item[0], &wg)
	// 	go requireNeverHappened(t, item[1], &wg)
	// 	go requireNeverHappened(t, item[2], &wg)
	// 	go requireNeverHappened(t, item[3], &wg)
	// 	go requireNeverHappened(t, item[4], &wg)
	// 	wg.Wait()
	// })

	// t.Run("retries items when requested", func(t *testing.T) {
	// 	t.Parallel()

	// 	var wg sync.WaitGroup
	// 	item := makeItems()
	// 	item[0].retry = true
	// 	item[0].retryIn = 1 * time.Second
	// 	// item[2].ready = true

	// 	w := process.NewPoolWorker("", 2, 100*time.Millisecond)
	// 	err := w.Start()
	// 	require.NoError(t, err)
	// 	defer w.Close()

	// 	w.Add(item[0])

	// 	wg.Add(1)
	// 	go requireHappened(t, item[0], 2, &wg)
	// 	wg.Wait()
	// })

	t.Run("disallows simultaneous processing of items with the same UniqueID", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup
		item := makeItems()
		item[1].block = make(chan struct{})
		item[1].retry = true
		item[1].retryIn = 2 * time.Second

		w := process.NewPoolWorker("", 2, 100*time.Millisecond)
		err := w.Start()
		require.NoError(t, err)
		defer w.Close()

		w.Add(item[1])
		w.Add(item[2])

		wg.Add(2)
		go requireHappened(t, item[1], 1, &wg)
		go requireNeverHappened(t, item[2], &wg)
		wg.Wait()

		fmt.Println("---- closing block")
		close(item[1].block)
		wg.Add(1)
		// go requireHappened(t, item[1], 1, &wg)
		go requireHappened(t, item[2], 1, &wg)
		wg.Wait()
	})

	// t.Run("disallows simultaneous processing of more than `concurrency` items", func(t *testing.T) {
	// 	t.Parallel()

	// 	var wg sync.WaitGroup
	// 	item := makeItems()
	// 	item[0].block = make(chan struct{})
	// 	item[1].block = make(chan struct{})

	// 	w := process.NewPoolWorker("", 2, 100*time.Millisecond)
	// 	err := w.Start()
	// 	require.NoError(t, err)
	// 	defer w.Close()

	// 	w.Add(item[0])
	// 	w.Add(item[1])
	// 	w.Add(item[3])

	// 	wg.Add(3)
	// 	go requireHappened(t, item[0], 1, &wg)
	// 	go requireHappened(t, item[1], 1, &wg)
	// 	go requireNeverHappened(t, item[3], &wg)
	// 	wg.Wait()

	// 	close(item[1].block)
	// 	wg.Add(3)
	// 	go requireNeverHappened(t, item[0], &wg)
	// 	go requireNeverHappened(t, item[1], &wg)
	// 	go requireHappened(t, item[3], 1, &wg)
	// 	wg.Wait()
	// 	close(item[0].block)
	// })

	// t.Run("will never retry an item with a UniqueID that previously returned `false` from Work()", func(t *testing.T) {
	// 	t.Parallel()

	// 	var wg sync.WaitGroup
	// 	item := makeItems()

	// 	w := process.NewPoolWorker("", 2, 100*time.Millisecond)
	// 	err := w.Start()
	// 	require.NoError(t, err)
	// 	defer w.Close()

	// 	w.Add(item[1])

	// 	wg.Add(1)
	// 	go requireHappened(t, item[1], 1, &wg)
	// 	wg.Wait()

	// 	w.Add(item[2])

	// 	wg.Add(1)
	// 	go requireNeverHappened(t, item[2], &wg)
	// 	wg.Wait()
	// })
}

type workItem struct {
	id        string
	retry     bool
	retryIn   time.Duration
	processed testutils.Awaiter
	block     chan struct{}
}

func (i workItem) ID() string                                 { return i.id }
func (i workItem) BlacklistUniqueID() process.PoolUniqueID    { return i.id[:1] }
func (i workItem) RetryUniqueID() process.PoolUniqueID        { return i.id[:1] }
func (i workItem) DedupeActiveUniqueID() process.PoolUniqueID { return i.id[:1] }

func (i *workItem) Work(ctx context.Context) (retry bool, when time.Time) {
	i.processed.ItHappened()
	if i.block != nil {
		select {
		case <-i.block:
			i.block = nil
		case <-ctx.Done():
			return false, time.Time{}
		}
	}
	retry = i.retry
	i.retry = false
	return retry, time.Now().Add(i.retryIn)
}

func requireHappened(t *testing.T, item *workItem, times int, wg *sync.WaitGroup) {
	t.Helper()
	defer wg.Done()
	for i := 0; i < times; i++ {
		item.processed.AwaitOrFail(t, 5*time.Second, item.id)
		fmt.Println("happened:", item.id)
	}
	item.processed.NeverHappenedOrFail(t, 5*time.Second, item.id)
}

func requireNeverHappened(t *testing.T, item *workItem, wg *sync.WaitGroup) {
	t.Helper()
	defer wg.Done()
	item.processed.NeverHappenedOrFail(t, 5*time.Second, item.id)
}
