import { create } from 'zustand';

import { IsGameRunning, LaunchGame, StopGame } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';

export interface LogEntry {
  stream: 'stdout' | 'stderr';
  line: string;
  timestamp: number;
}

export interface GameLogSession {
  id: string;
  startedAt: number;
  endedAt: number | null;
  logs: LogEntry[];
}

function createSessionId(): string {
  if (
    typeof crypto !== 'undefined' &&
    typeof crypto.randomUUID === 'function'
  ) {
    return crypto.randomUUID();
  }

  return `session-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function createNewSessionPatch(
  sessions: GameLogSession[],
  startedAt: number,
  logs: LogEntry[] = [],
): { sessionId: string; sessions: GameLogSession[] } {
  const sessionId = createSessionId();
  return {
    sessionId,
    sessions: [
      ...sessions,
      {
        id: sessionId,
        startedAt,
        endedAt: null,
        logs,
      },
    ],
  };
}

interface GameState {
  running: boolean;
  sessions: GameLogSession[];
  selectedSessionId: string | null;
  maxLogs: number;
  serverPort: number | null;

  initialize: () => void;
  launch: () => Promise<void>;
  stop: () => Promise<void>;
  selectSession: (id: string) => void;
  clearLogs: () => void;
}

export const useGameStore = create<GameState>((set) => ({
  running: false,
  sessions: [],
  selectedSessionId: null,
  maxLogs: 5000,
  serverPort: null,

  initialize: () => {
    const appendLogToSession = (stream: 'stdout' | 'stderr', line: string) => {
      const timestamp = Date.now();
      set((state) => {
        const activeIndex = state.sessions.findIndex(
          (session) => session.endedAt === null,
        );

        if (activeIndex === -1) {
          const entry: LogEntry = { stream, line, timestamp };
          const { sessionId, sessions } = createNewSessionPatch(
            state.sessions,
            timestamp,
            [entry],
          );

          return {
            selectedSessionId: sessionId,
            sessions,
          };
        }

        const nextSessions = [...state.sessions];
        const target = nextSessions[activeIndex];
        const nextLogs = [...target.logs, { stream, line, timestamp }];

        nextSessions[activeIndex] = {
          ...target,
          logs:
            nextLogs.length > state.maxLogs
              ? nextLogs.slice(-state.maxLogs)
              : nextLogs,
        };

        return {
          sessions: nextSessions,
          selectedSessionId:
            state.selectedSessionId === null ||
            state.selectedSessionId === target.id
              ? target.id
              : state.selectedSessionId,
        };
      });
    };

    // Check initial state
    IsGameRunning().then((response) => {
      if (response.status !== 'success' || !response.running) {
        set({ running: false });
        return;
      }

      const now = Date.now();
      set((state) => {
        const hasActiveSession = state.sessions.some(
          (session) => session.endedAt === null,
        );
        if (hasActiveSession) {
          return { running: true };
        }

        const { sessionId, sessions } = createNewSessionPatch(
          state.sessions,
          now,
        );
        return {
          running: true,
          selectedSessionId: sessionId,
          sessions,
        };
      });
    });

    // Listen for events from backend
    EventsOn('game:status', (status: string) => {
      if (status === 'running') {
        const now = Date.now();
        set((state) => {
          const hasActiveSession = state.sessions.some(
            (session) => session.endedAt === null,
          );
          if (hasActiveSession) {
            return { running: true };
          }

          const { sessionId, sessions } = createNewSessionPatch(
            state.sessions,
            now,
          );
          return {
            running: true,
            selectedSessionId: sessionId,
            sessions,
          };
        });
        return;
      }

      set((state) => {
        const activeIndex = state.sessions.findIndex(
          (session) => session.endedAt === null,
        );
        if (activeIndex === -1) {
          return { running: false, serverPort: null };
        }

        const nextSessions = [...state.sessions];
        nextSessions[activeIndex] = {
          ...nextSessions[activeIndex],
          endedAt: Date.now(),
        };

        return {
          running: false,
          serverPort: null,
          sessions: nextSessions,
        };
      });
    });

    EventsOn('server:port', (port: number) => {
      set({ serverPort: port });
    });

    EventsOn(
      'game:log',
      (data: { stream: 'stdout' | 'stderr'; line: string }) => {
        appendLogToSession(data.stream, data.line);
      },
    );

    EventsOn('game:exit', (exitCode: number) => {
      appendLogToSession(
        'stderr',
        exitCode === 0
          ? '--- Game exited normally ---'
          : `--- Game exited with code ${exitCode} ---`,
      );
    });
  },

  launch: async () => {
    const response = await LaunchGame();
    if (response.status === 'error') {
      throw new Error(response.message || 'Failed to launch game');
    }
  },

  stop: async () => {
    const response = await StopGame();
    if (response.status === 'error') {
      throw new Error(response.message || 'Failed to stop game');
    }
  },

  selectSession: (id: string) => set({ selectedSessionId: id }),

  clearLogs: () =>
    set((state) => {
      if (!state.selectedSessionId) return {};

      // Never delete the currently active session
      const selected = state.sessions.find(
        (s) => s.id === state.selectedSessionId,
      );
      if (selected?.endedAt === null) return {};

      const nextSessions = state.sessions.filter(
        (session) => session.id !== state.selectedSessionId,
      );

      return {
        sessions: nextSessions,
        selectedSessionId:
          nextSessions.length > 0
            ? nextSessions[nextSessions.length - 1].id
            : null,
      };
    }),

}));
