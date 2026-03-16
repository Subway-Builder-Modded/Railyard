import { create } from 'zustand';

import { IsGameRunning, LaunchGame, StopGame } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';

export interface LogEntry {
  stream: 'stdout' | 'stderr';
  line: string;
  timestamp: number;
}

interface GameState {
  running: boolean;
  logs: LogEntry[];
  maxLogs: number;
  serverPort: number | null;

  initialize: () => void;
  launch: () => Promise<void>;
  stop: () => Promise<void>;
  clearLogs: () => void;
}

export const useGameStore = create<GameState>((set, get) => ({
  running: false,
  logs: [],
  maxLogs: 5000,
  serverPort: null,

  initialize: () => {
    // Check initial state
    IsGameRunning().then((running) => set({ running }));

    // Listen for events from backend
    EventsOn('game:status', (status: string) => {
      set({ running: status === 'running' });
      if (status === 'stopped') {
        set({ serverPort: null });
      }
    });

    EventsOn('server:port', (port: number) => {
      set({ serverPort: port });
    });

    EventsOn(
      'game:log',
      (data: { stream: 'stdout' | 'stderr'; line: string }) => {
        const entry: LogEntry = {
          stream: data.stream,
          line: data.line,
          timestamp: Date.now(),
        };
        const { logs, maxLogs } = get();
        const next = [...logs, entry];
        set({ logs: next.length > maxLogs ? next.slice(-maxLogs) : next });
      },
    );

    EventsOn('game:exit', (exitCode: number) => {
      const entry: LogEntry = {
        stream: 'stderr',
        line:
          exitCode === 0
            ? '--- Game exited normally ---'
            : `--- Game exited with code ${exitCode} ---`,
        timestamp: Date.now(),
      };
      set((s) => ({ logs: [...s.logs, entry] }));
    });
  },

  launch: async () => {
    await LaunchGame();
  },

  stop: async () => {
    await StopGame();
  },

  clearLogs: () => set({ logs: [] }),
}));
