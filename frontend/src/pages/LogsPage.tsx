import { useEffect, useRef } from "react";
import { useGameStore, LogEntry } from "@/stores/game-store";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/shared/EmptyState";
import { Trash2, ArrowDownToLine, Terminal } from "lucide-react";
import { cn } from "@/lib/utils";

function LogLine({ entry }: { entry: LogEntry }) {
  const time = new Date(entry.timestamp).toLocaleTimeString();
  return (
    <div className={cn(
      "flex gap-2 px-3 py-0.5 text-xs font-mono hover:bg-muted/50",
      entry.stream === "stderr" && "text-destructive"
    )}>
      <span className="text-muted-foreground shrink-0 select-none">{time}</span>
      <span className="break-all whitespace-pre-wrap">{entry.line}</span>
    </div>
  );
}

export function LogsPage() {
  const { logs, running, clearLogs } = useGameStore();
  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new logs arrive, but only if already near bottom
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    const isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 100;
    if (isNearBottom) {
      bottomRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [logs.length]);

  return (
    <div className="flex flex-col h-[calc(100vh-theme(spacing.14)-theme(spacing.12))]">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-semibold tracking-tight">Game Logs</h1>
          <Badge variant={running ? "default" : "secondary"}>
            {running ? "Running" : "Stopped"}
          </Badge>
        </div>
        <div className="flex items-center gap-2">
          {logs.length > 0 && (
            <span className="text-xs text-muted-foreground mr-2">{logs.length} lines</span>
          )}
          <Button variant="outline" size="sm" onClick={clearLogs} disabled={logs.length === 0}>
            <Trash2 className="h-4 w-4 mr-1.5" />
            Clear
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => bottomRef.current?.scrollIntoView({ behavior: "smooth" })}
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
          title="No game logs yet"
          description={running ? "Waiting for output..." : "Launch the game to see logs here."}
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
