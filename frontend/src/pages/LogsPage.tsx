import {
  ArrowDownToLine,
  Calendar,
  ExternalLink,
  Play,
  Square,
  Terminal,
  Trash2,
} from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';

import { AppDialog } from '@/components/dialogs/AppDialog';
import { EmptyState } from '@/components/shared/EmptyState';
import { PageHeading } from '@/components/shared/PageHeading';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';
import { useConfigStore } from '@/stores/config-store';
import type { LogEntry } from '@/stores/game-store';
import { useGameStore } from '@/stores/game-store';

import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';

function LogLine({ entry }: { entry: LogEntry }) {
  const time = new Date(entry.timestamp).toLocaleTimeString();
  return (
    <div
      className={cn(
        'flex gap-3 rounded-sm px-4 py-0.5 text-xs font-mono hover:bg-muted/50',
        entry.stream === 'stderr' && 'text-[var(--uninstall-primary)]',
      )}
    >
      <span className="shrink-0 select-none tabular-nums text-muted-foreground/60">
        {time}
      </span>
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
    launch,
    stop,
    clearLogs,
    serverPort,
  } = useGameStore();

  const canLaunch = useConfigStore((s) => s.validation?.executablePathValid);
  const [deleteSessionOpen, setDeleteSessionOpen] = useState(false);

  const selectedSession =
    sessions.find((session) => session.id === selectedSessionId) ?? null;
  const latestSessionId =
    sessions.length > 0 ? sessions[sessions.length - 1].id : null;
  const logs = selectedSession?.logs ?? [];
  const isSelectedSessionActive = selectedSession?.endedAt === null;
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

  const handleLaunch = async () => {
    try {
      await launch();
    } catch (err) {
      toast.error(String(err) || 'Failed to launch game.');
    }
  };

  const handleStop = async () => {
    try {
      await stop();
    } catch (err) {
      toast.error(String(err) || 'Failed to stop game.');
    }
  };

  const handleDeleteSessionClick = () => {
    if (isSelectedSessionActive) {
      toast.error(
        'Cannot delete the active session while the game is running.',
      );
      return;
    }
    setDeleteSessionOpen(true);
  };

  const handleDeleteSessionConfirm = () => {
    clearLogs();
    toast.success('Session deleted.');
  };

  return (
    <div className="relative isolate flex flex-col h-[calc(100vh-theme(spacing.14)-theme(spacing.12))]">
      <div className="relative z-10">
        <PageHeading
          icon={Terminal}
          title="Game Logs"
          description="Inspect game output, troubleshoot issues, and export diagnostics."
        />
      </div>

      <div className="relative z-10 mb-4 flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Badge variant={running ? 'success' : 'secondary'} size="sm">
            {running ? 'Running' : 'Stopped'}
          </Badge>
          {logs.length > 0 && (
            <span className="tabular-nums text-xs text-muted-foreground">
              {logs.length.toLocaleString()} lines
            </span>
          )}
          <div className="h-4 w-px shrink-0 bg-border" />
          {running ? (
            <Button
              variant="ghost"
              size="sm"
              onClick={handleStop}
              className="h-7 gap-1.5 px-2.5 text-xs font-semibold bg-[color-mix(in_srgb,var(--install-primary)_18%,transparent)] text-[var(--install-primary)] hover:!bg-[color-mix(in_srgb,var(--uninstall-primary)_22%,transparent)] hover:!text-[var(--uninstall-primary)]"
            >
              <Square className="h-3 w-3" />
              Stop
            </Button>
          ) : (
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleLaunch}
                    disabled={!canLaunch}
                    className="h-7 gap-1.5 px-2.5 text-xs font-semibold hover:bg-accent/45 hover:text-primary disabled:opacity-50"
                  >
                    <Play className="h-3 w-3" />
                    Launch
                  </Button>
                </span>
              </TooltipTrigger>
              {!canLaunch && (
                <TooltipContent>
                  Configure game executable in Settings first
                </TooltipContent>
              )}
            </Tooltip>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {sessions.length > 0 && selectedSessionId && (
            <div className="flex items-center overflow-hidden rounded-xl border border-border/70 bg-background/90 shadow-sm backdrop-blur-md">
              <Select value={selectedSessionId} onValueChange={selectSession}>
                <SelectTrigger
                  size="sm"
                  className={cn(
                    'border-0 bg-transparent shadow-none',
                    'dark:bg-transparent dark:hover:bg-transparent',
                    'h-8 min-w-[13rem] gap-2 px-3',
                    'text-xs font-semibold text-muted-foreground',
                    'hover:bg-accent/45 hover:text-primary dark:hover:bg-accent/45',
                    'data-[state=open]:bg-accent/45 data-[state=open]:text-primary',
                    '[&_svg]:!text-current',
                    'rounded-none border-r border-border/60',
                  )}
                >
                  <span className="flex min-w-0 items-center gap-2">
                    <Calendar
                      className="h-3.5 w-3.5 shrink-0 text-current"
                      aria-hidden
                    />
                    <SelectValue placeholder="Select session" />
                  </span>
                </SelectTrigger>
                <SelectContent
                  side="bottom"
                  sideOffset={4}
                  position="popper"
                  align="end"
                  avoidCollisions={false}
                  className="rounded-xl border border-border/70 bg-background/95 p-1 shadow-lg backdrop-blur-md"
                >
                  {[...sessions].reverse().map((session) => (
                    <SelectItem
                      key={session.id}
                      value={session.id}
                      className={cn(
                        'rounded-lg text-sm',
                        'data-[highlighted]:bg-accent/45 data-[highlighted]:text-primary',
                        'data-[state=checked]:bg-accent/35 data-[state=checked]:text-primary',
                      )}
                    >
                      <span className="flex w-full items-center justify-between gap-3">
                        <span>
                          {new Date(session.startedAt).toLocaleString()}
                        </span>
                        {session.id === latestSessionId && (
                          <Badge variant="success" size="sm">
                            {session.endedAt === null ? 'Active' : 'Latest'}
                          </Badge>
                        )}
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <button
                type="button"
                onClick={handleDeleteSessionClick}
                aria-label="Delete this session"
                className={cn(
                  'flex h-8 w-8 items-center justify-center transition-colors',
                  'text-[var(--uninstall-primary)]',
                  'hover:bg-[color-mix(in_srgb,var(--uninstall-primary)_20%,transparent)]',
                )}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          )}
          {sessions.length > 0 && (
            <div className="self-stretch flex items-center py-1.5">
              <div className="h-full w-[2px] rounded-full bg-foreground/20" />
            </div>
          )}

          <Button
            variant="outline"
            size="sm"
            className="border-border/70"
            disabled={!serverPort}
            onClick={() =>
              serverPort &&
              BrowserOpenURL(`http://127.0.0.1:${serverPort}/debug/thumbnails`)
            }
          >
            <ExternalLink className="h-4 w-4 mr-1.5" />
            Debug Thumbnails
          </Button>

          <Button
            variant="outline"
            size="sm"
            className="border-border/70"
            onClick={() =>
              bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
            }
            disabled={logs.length === 0}
          >
            <ArrowDownToLine className="h-4 w-4 mr-1.5" />
            Scroll to Bottom
          </Button>
        </div>
      </div>

      {logs.length === 0 ? (
        <div className="relative z-10 flex-1">
          <EmptyState
            icon={Terminal}
            title={
              sessions.length === 0 ? 'No game logs' : 'No logs in this session'
            }
            description={
              sessions.length === 0
                ? running
                  ? 'Waiting for output...'
                  : 'Your game logs will appear here.'
                : 'Select another session or wait for output.'
            }
          />
        </div>
      ) : (
        <div
          ref={containerRef}
          className="relative z-10 flex-1 overflow-y-auto rounded-xl border border-border/70 bg-background/40 shadow-sm backdrop-blur-sm"
        >
          <div className="py-2">
            {logs.map((entry, i) => (
              <LogLine key={i} entry={entry} />
            ))}
            <div ref={bottomRef} />
          </div>
        </div>
      )}

      <AppDialog
        open={deleteSessionOpen}
        onOpenChange={setDeleteSessionOpen}
        title="Delete Session"
        description="This will permanently remove the selected session log."
        icon={Trash2}
        tone="uninstall"
        confirm={{
          label: 'Delete',
          onConfirm: () => {
            handleDeleteSessionConfirm();
            setDeleteSessionOpen(false);
          },
        }}
      />
    </div>
  );
}
