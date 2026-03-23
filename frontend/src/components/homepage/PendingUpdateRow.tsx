import { ArrowRight, MapPin, Package, RefreshCw } from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import type { AssetType } from '@/lib/asset-types';
import { cn } from '@/lib/utils';

interface PendingUpdateRowProps {
  name: string;
  type: AssetType;
  currentVersion: string;
  latestVersion: string;
  isUpdating: boolean;
  onUpdate: () => void;
  updateButtonClassName: string;
}

export function PendingUpdateRow({
  name,
  type,
  currentVersion,
  latestVersion,
  isUpdating,
  onUpdate,
  updateButtonClassName,
}: PendingUpdateRowProps) {
  return (
    <div className="group flex items-center gap-3 rounded-lg border border-border bg-background/60 px-3.5 py-2.5 transition-colors duration-150 hover:border-foreground/15">
      <div className="flex min-w-0 flex-1 items-center gap-2">
        <Badge
          size="sm"
          className={cn(
            'capitalize',
            'border-border/80 bg-muted/60 text-muted-foreground',
            'group-hover:border-[color-mix(in_oklab,var(--install-primary)_35%,transparent)]',
            'group-hover:bg-[color-mix(in_oklab,var(--install-primary)_14%,transparent)]',
            'group-hover:text-[var(--install-primary)]',
          )}
        >
          {type === 'map' ? (
            <MapPin className="h-2.5 w-2.5" aria-hidden />
          ) : (
            <Package className="h-2.5 w-2.5" aria-hidden />
          )}
          {type}
        </Badge>
        <span className="truncate text-sm font-medium text-foreground">
          {name}
        </span>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <span className="font-mono text-xs text-muted-foreground">
          {currentVersion}
        </span>
        <ArrowRight className="h-3 w-3 text-muted-foreground/40" aria-hidden />
        <span className="font-mono text-xs font-semibold text-[var(--update-primary)]">
          {latestVersion}
        </span>
        <Button
          size="sm"
          disabled={isUpdating}
          onClick={onUpdate}
          className={cn(
            'h-7 min-w-[4.5rem] px-3 text-xs',
            updateButtonClassName,
          )}
        >
          {isUpdating ? (
            <RefreshCw className="h-3 w-3 animate-spin" aria-hidden />
          ) : (
            'Update'
          )}
        </Button>
      </div>
    </div>
  );
}
