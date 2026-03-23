import { CircleFadingArrowUp, Download, RefreshCw } from 'lucide-react';
import { useMemo } from 'react';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { getLocalAccentClasses } from '@/lib/local-accent';

const UPDATE_ACCENT = getLocalAccentClasses('update');
const UPDATE_PREVIEW_LIMIT = 10;

export interface UpdateConfirmEntry {
  key: string;
  name: string;
  currentVersion: string;
  latestVersion: string;
}

interface UpdateConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  entries: UpdateConfirmEntry[];
  confirmLabel: string;
  confirming: boolean;
  onConfirm: () => void;
}

export function UpdateConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  entries,
  confirmLabel,
  confirming,
  onConfirm,
}: UpdateConfirmDialogProps) {
  const sortedEntries = useMemo(
    () => [...entries].sort((left, right) => left.name.localeCompare(right.name)),
    [entries],
  );
  const previewEntries = useMemo(
    () => sortedEntries.slice(0, UPDATE_PREVIEW_LIMIT),
    [sortedEntries],
  );
  const remainingPreviewCount = Math.max(
    0,
    sortedEntries.length - previewEntries.length,
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <CircleFadingArrowUp
              className="h-5 w-5 text-[var(--update-primary)]"
              aria-hidden
            />
            {title}
          </DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        {previewEntries.length > 0 && (
          <div className="max-h-48 overflow-y-auto rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
            <ul className="space-y-1">
              {previewEntries.map((entry) => (
                <li key={entry.key} className="flex gap-2">
                  <span className="min-w-0 flex-1 truncate">{entry.name}</span>
                  <span className="font-mono tabular-nums text-foreground">
                    {entry.currentVersion} -&gt; {entry.latestVersion}
                  </span>
                </li>
              ))}
              {remainingPreviewCount > 0 && (
                <li className="pt-1 text-right font-medium text-muted-foreground">
                  +{remainingPreviewCount} more
                </li>
              )}
            </ul>
          </div>
        )}

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            className="border-[color-mix(in_srgb,var(--update-primary)_55%,transparent)] bg-transparent text-[var(--update-primary)] hover:bg-[color-mix(in_srgb,var(--update-primary)_12%,transparent)] hover:text-[var(--update-primary)]"
          >
            Cancel
          </Button>
          <Button
            disabled={confirming}
            onClick={onConfirm}
            className={UPDATE_ACCENT.solidButton}
          >
            {confirming ? (
              <RefreshCw className="h-3 w-3 animate-spin" aria-hidden />
            ) : (
              <Download className="h-3 w-3" aria-hidden />
            )}
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
