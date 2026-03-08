package downloader

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

func newTestDownloader() *Downloader {
	return &Downloader{}
}

func runInParallel(total int, fn func(index int)) {
	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			fn(index)
		}(i)
	}
	wg.Wait()
}

func successfulOperation(message string, delay time.Duration, onRun func()) func() operationResult {
	return func() operationResult {
		if onRun != nil {
			onRun()
		}
		if delay > 0 {
			time.Sleep(delay)
		}
		return operationResult{
			genericResponse: types.GenericResponse{
				Status:  types.ResponseSuccess,
				Message: message,
			},
		}
	}
}

func updateAtomicMax(max *int32, current int32) {
	for {
		existing := atomic.LoadInt32(max)
		if current <= existing {
			return
		}
		if atomic.CompareAndSwapInt32(max, existing, current) {
			return
		}
	}
}

func TestEnqueueOperationDeduplicatesByKey(t *testing.T) {
	d := newTestDownloader()
	requestKey := d.operationKey(operationActionInstall, types.AssetTypeMap, "map-a", "1.0.0")
	const callers = 6

	var runCount int32
	var dedupedCount int32
	results := make([]operationResult, callers)

	// Simulate multiple concurrent calls with the same operation key
	runInParallel(callers, func(index int) {
		result, deduped := d.enqueueOperation(requestKey, successfulOperation("ok", 30*time.Millisecond, func() {
			atomic.AddInt32(&runCount, 1)
		}))
		if deduped {
			atomic.AddInt32(&dedupedCount, 1)
			return
		}
		results[index] = result
	})

	// Only one caller should have executed the operation, while the others should have been deduped
	require.Equal(t, int32(1), atomic.LoadInt32(&runCount))
	require.Equal(t, int32(callers-1), atomic.LoadInt32(&dedupedCount))

	successCount := 0
	for _, result := range results {
		if result.genericResponse.Status == types.ResponseSuccess {
			successCount++
			require.Equal(t, "ok", result.genericResponse.Message)
		}
	}
	require.Equal(t, 1, successCount)
}

func TestEnqueueOperationRunsSequentially(t *testing.T) {
	d := newTestDownloader()
	const jobs = 5

	var runCount int32
	var activeCount int32
	var maxConcurrent int32

	// Enqueue multiple operations with the same key to test sequential execution
	runInParallel(jobs, func(i int) {
		key := d.operationKey(operationActionInstall, types.AssetTypeMod, fmt.Sprintf("mod-%d", i), "1.0.0")
		_, _ = d.enqueueOperation(key, successfulOperation("done", 20*time.Millisecond, func() {
			atomic.AddInt32(&runCount, 1)
			current := atomic.AddInt32(&activeCount, 1)
			updateAtomicMax(&maxConcurrent, current)
			defer atomic.AddInt32(&activeCount, -1)
		}))
	})

	// All operations should have run, but never more than one at a time
	require.Equal(t, int32(jobs), atomic.LoadInt32(&runCount))
	require.Equal(t, int32(1), atomic.LoadInt32(&maxConcurrent))
}

func TestIsValidOperationAction(t *testing.T) {
	require.True(t, isValidOperationAction(operationActionInstall))
	require.True(t, isValidOperationAction(operationActionUninstall))
	require.False(t, isValidOperationAction(operationAction("invalid")))
}

func TestOperationKeyPanicsOnInvalidAction(t *testing.T) {
	d := &Downloader{}
	require.Panics(t, func() {
		_ = d.operationKey(operationAction("invalid"), types.AssetTypeMap, "map-a", "1.0.0")
	})
}
