import { Loader2, type LucideIcon } from 'lucide-react';
import { type ReactNode, useMemo } from 'react';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  getLocalAccentClasses,
  type LocalAccentTone,
} from '@/lib/local-accent';

import type { types } from '../../../wailsjs/go/models';

const ENTRIES_PREVIEW_LIMIT = 10;

export interface UpdateConfirmEntry {
  key: string;
  name: string;
  currentVersion: string;
  latestVersion: string;
}

interface AssetActionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  icon: LucideIcon;
  iconClassName?: string;
  confirmLabel: string;
  confirmClassName?: string;
  confirmVariant?: 'default' | 'destructive';
  tone?: LocalAccentTone;
  loading: boolean;
  onConfirm: () => void;
  conflict?: types.MapCodeConflict;
  entries?: UpdateConfirmEntry[];
  children?: ReactNode;
}

function conflictSourceLabel(conflict: types.MapCodeConflict): string {
  if (conflict.existingAssetId?.startsWith('vanilla:')) {
    return 'Vanilla';
  }
  return conflict.existingIsLocal ? 'Local' : 'Registry';
}

export function AssetActionDialog({
  open,
  onOpenChange,
  title,
  description,
  icon: Icon,
  iconClassName,
  confirmLabel,
  confirmClassName,
  confirmVariant = 'default',
  tone,
  loading,
  onConfirm,
  conflict,
  entries,
  children,
}: AssetActionDialogProps) {
  const toneStyles = tone ? getLocalAccentClasses(tone) : null;

  const sortedEntries = useMemo(
    () =>
      entries
        ? [...entries].sort((a, b) => a.name.localeCompare(b.name))
        : [],
    [entries],
  );
  const previewEntries = sortedEntries.slice(0, ENTRIES_PREVIEW_LIMIT);
  const remainingCount = Math.max(0, sortedEntries.length - previewEntries.length);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Icon className={iconClassName ?? 'h-5 w-5'} />
            {title}
          </DialogTitle>
          <DialogDescription>{description}</DialogDescription>
          {conflict ? (
            <div
              className={`mt-1 rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground ${toneStyles?.dialogPanel ?? 'border-border'}`}
            >
              <p className="font-medium text-foreground">
                Conflicting City Code: {conflict.cityCode}
              </p>
              <p className="mt-1">
                Existing Asset: {conflict.existingAssetId} (
                {conflictSourceLabel(conflict)})
              </p>
              {conflict.existingVersion ? (
                <p className="mt-1">
                  Existing Version: {conflict.existingVersion}
                </p>
              ) : null}
            </div>
          ) : null}
          {children}
        </DialogHeader>

        {previewEntries.length > 0 && (
          <div className="max-h-48 overflow-y-auto rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
            <ul className="space-y-1">
              {previewEntries.map((entry) => (
                <li key={entry.key} className="flex gap-2">
                  <span className="min-w-0 flex-1 truncate">{entry.name}</span>
                  <span className="font-mono tabular-nums text-foreground">
                    {entry.currentVersion} &rarr; {entry.latestVersion}
                  </span>
                </li>
              ))}
              {remainingCount > 0 && (
                <li className="pt-1 text-right font-medium text-muted-foreground">
                  +{remainingCount} more
                </li>
              )}
            </ul>
          </div>
        )}

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={loading}
            className={
              toneStyles?.dialogCancel ??
              'hover:bg-muted/70 hover:text-foreground'
            }
          >
            Cancel
          </Button>
          <Button
            variant={confirmVariant}
            onClick={onConfirm}
            disabled={loading}
            className={confirmClassName}
          >
            {loading && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
