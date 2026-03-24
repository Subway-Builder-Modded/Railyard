import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { useDownloadQueueStore } from './download-queue-store';

describe('useDownloadQueueStore', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    useDownloadQueueStore.setState({ total: 0, completed: 0 });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('enqueue increments total', () => {
    useDownloadQueueStore.getState().enqueue();
    useDownloadQueueStore.getState().enqueue();
    expect(useDownloadQueueStore.getState().total).toBe(2);
    expect(useDownloadQueueStore.getState().completed).toBe(0);
  });

  it('complete increments completed', () => {
    useDownloadQueueStore.getState().enqueue();
    useDownloadQueueStore.getState().enqueue();
    useDownloadQueueStore.getState().complete();
    expect(useDownloadQueueStore.getState().completed).toBe(1);
  });

  it('resets both counters to zero after batch completes and timeout elapses', () => {
    useDownloadQueueStore.getState().enqueue();
    useDownloadQueueStore.getState().enqueue();
    useDownloadQueueStore.getState().complete();
    useDownloadQueueStore.getState().complete();

    // Counters not reset yet — timer is pending
    expect(useDownloadQueueStore.getState().total).toBe(2);
    expect(useDownloadQueueStore.getState().completed).toBe(2);

    vi.advanceTimersByTime(5000);

    expect(useDownloadQueueStore.getState().total).toBe(0);
    expect(useDownloadQueueStore.getState().completed).toBe(0);
  });

  it('does not reset when more items were enqueued after the last complete', () => {
    useDownloadQueueStore.getState().enqueue();
    useDownloadQueueStore.getState().enqueue();
    useDownloadQueueStore.getState().complete();
    // Second item still in flight — enqueue a third before completing
    useDownloadQueueStore.getState().enqueue();
    useDownloadQueueStore.getState().complete(); // completed = 2, total = 3

    vi.advanceTimersByTime(5000);

    // Should NOT reset because completed (2) < total (3)
    expect(useDownloadQueueStore.getState().total).toBe(3);
  });
});
