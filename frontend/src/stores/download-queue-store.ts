import { create } from "zustand";

interface DownloadQueueState {
  /** Total items enqueued in the current batch. */
  total: number;
  /** Items that have completed (installed / errored) so far. */
  completed: number;

  /** Call when a new install is about to start. */
  enqueue: () => void;
  /** Call when an install finishes (success or failure). */
  complete: () => void;
}

export const useDownloadQueueStore = create<DownloadQueueState>((set, get) => ({
  total: 0,
  completed: 0,

  enqueue: () => set((s) => ({ total: s.total + 1 })),

  complete: () => {
    const newCompleted = get().completed + 1;
    set({ completed: newCompleted });

    // Reset counters once the entire batch finishes
    if (newCompleted >= get().total) {
      setTimeout(() => {
        const { completed, total } = get();
        if (completed >= total && total > 0) {
          set({ total: 0, completed: 0 });
        }
      }, 5000);
    }
  },
}));
