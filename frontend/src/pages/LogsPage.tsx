import { ArrowDownToLine, ExternalLink, Terminal, Trash2 } from 'lucide-react';
import { useEffect, useRef } from 'react';

import { EmptyState } from '@/components/shared/EmptyState';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { cn } from '@/lib/utils';
import type { LogEntry } from '@/stores/game-store';
import { useGameStore } from '@/stores/game-store';

import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';

function LogLine({ entry }: { entry: LogEntry }) {
  const time = new Date(entry.timestamp).toLocaleTimeString();
  return (
    <div
      className={cn(
        'flex gap-2 px-3 py-0.5 text-xs font-mono hover:bg-muted/50',
        entry.stream === 'stderr' && 'text-destructive',
      )}
    >
      <span className="text-muted-foreground shrink-0 select-none">{time}</span>
      <span className="break-all whitespace-pre-wrap">{entry.line}</span>
    </div>
  );
}

export function LogsPage() {
  const {
    sessions,
    selectedSessionId,
    selectSession,
    running,
    clearLogs,
    serverPort,
  } = useGameStore();
  const selectedSession =
    sessions.find((session) => session.id === selectedSessionId) ?? null;
  const latestSessionId =
    sessions.length > 0 ? sessions[sessions.length - 1].id : null;
  const logs = selectedSession?.logs ?? [];
  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const previousSessionIdRef = useRef<string | null>(null);
  const skipNextAutoScrollRef = useRef(false);

  useEffect(() => {
    if (previousSessionIdRef.current !== selectedSessionId) {
      previousSessionIdRef.current = selectedSessionId;
      skipNextAutoScrollRef.current = true;
    }
  }, [selectedSessionId]);

  // Auto-scroll to bottom when new logs arrive, but only if already near bottom
  useEffect(() => {
    if (skipNextAutoScrollRef.current) {
      skipNextAutoScrollRef.current = false;
      return;
    }

    const container = containerRef.current;
    if (!container) return;
    const isNearBottom =
      container.scrollHeight - container.scrollTop - container.clientHeight <
      100;
    if (isNearBottom) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs.length]);

  return (
    <div className="flex flex-col h-[calc(100vh-theme(spacing.14)-theme(spacing.12))]">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-semibold tracking-tight">Game Logs</h1>
          <Badge variant={running ? 'default' : 'secondary'}>
            {running ? 'Running' : 'Stopped'}
          </Badge>
        </div>
        <div className="flex items-center gap-2">
          {sessions.length > 0 && selectedSessionId && (
            <Select value={selectedSessionId} onValueChange={selectSession}>
              <SelectTrigger className="w-64">
                <SelectValue placeholder="Select session" />
              </SelectTrigger>
              <SelectContent>
                {sessions.map((session) => (
                  <SelectItem key={session.id} value={session.id}>
                    <div className="flex w-full items-center justify-between gap-2">
                      <span>
                        {new Date(session.startedAt).toLocaleString()}
                      </span>
                      {session.id === latestSessionId && (
                        <Badge
                          variant="outline"
                          className="rounded-full border-emerald-500 text-emerald-500"
                        >
                          Latest
                        </Badge>
                      )}
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
          {logs.length > 0 && (
            <span className="text-xs text-muted-foreground mr-2">
              {logs.length} lines
            </span>
          )}
          {serverPort && (
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                BrowserOpenURL(
                  `http://127.0.0.1:${serverPort}/debug/thumbnails`,
                )
              }
            >
              <ExternalLink className="h-4 w-4 mr-1.5" />
              Debug Thumbnails
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={clearLogs}
            disabled={logs.length === 0}
          >
            <Trash2 className="h-4 w-4 mr-1.5" />
            Clear
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() =>
              bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
            }
            disabled={logs.length === 0}
          >
            <ArrowDownToLine className="h-4 w-4 mr-1.5" />
            Scroll to bottom
          </Button>
        </div>
      </div>

      {logs.length === 0 ? (
        <EmptyState
          icon={Terminal}
          title={
            sessions.length === 0
              ? 'No game logs yet'
              : 'No logs in this session'
          }
          description={
            sessions.length === 0
              ? running
                ? 'Waiting for output...'
                : 'Launch the game to see logs here.'
              : 'Select another session or wait for output.'
          }
        />
      ) : (
        <div
          ref={containerRef}
          className="flex-1 overflow-y-auto rounded-md border bg-muted/30"
        >
          <div className="py-2">
            {logs.map((entry, i) => (
              <LogLine key={i} entry={entry} />
            ))}
            <div ref={bottomRef} />
          </div>
        </div>
      )}
    </div>
  );
}
