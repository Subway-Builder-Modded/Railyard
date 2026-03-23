import { Rows3, SquareMenu, TableProperties } from 'lucide-react';

import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import { isSearchViewMode, type SearchViewMode } from '@/lib/search-view-mode';
import { cn } from '@/lib/utils';

interface ViewModeToggleProps {
  value: SearchViewMode;
  onChange: (value: SearchViewMode) => void;
}

export function ViewModeToggle({ value, onChange }: ViewModeToggleProps) {
  return (
    <ToggleGroup
      type="single"
      value={value}
      variant="default"
      size="sm"
      spacing={1}
      onValueChange={(v) => {
        if (isSearchViewMode(v)) onChange(v);
      }}
      aria-label="Browse view mode"
      className={cn(
        'rounded-xl border border-border/70 bg-background/90 p-0.5 shadow-sm backdrop-blur-md',
      )}
    >
      <ToggleGroupItem
        value="full"
        aria-label="Full view"
        className={cn(
          'h-7 px-2.5 text-xs font-semibold text-muted-foreground',
          'hover:bg-accent/45 hover:text-primary data-[state=on]:bg-accent/45 data-[state=on]:text-primary',
          'rounded-lg',
        )}
      >
        <SquareMenu className="h-4 w-4" />
        <span className="hidden sm:inline">Full</span>
      </ToggleGroupItem>
      <ToggleGroupItem
        value="compact"
        aria-label="Compact view"
        className={cn(
          'h-7 px-2.5 text-xs font-semibold text-muted-foreground',
          'hover:bg-accent/45 hover:text-primary data-[state=on]:bg-accent/45 data-[state=on]:text-primary',
          'rounded-lg',
        )}
      >
        <TableProperties className="h-4 w-4" />
        <span className="hidden sm:inline">Compact</span>
      </ToggleGroupItem>
      <ToggleGroupItem
        value="list"
        aria-label="List view"
        className={cn(
          'h-7 px-2.5 text-xs font-semibold text-muted-foreground',
          'hover:bg-accent/45 hover:text-primary data-[state=on]:bg-accent/45 data-[state=on]:text-primary',
          'rounded-lg',
        )}
      >
        <Rows3 className="h-4 w-4" />
        <span className="hidden sm:inline">List</span>
      </ToggleGroupItem>
    </ToggleGroup>
  );
}
