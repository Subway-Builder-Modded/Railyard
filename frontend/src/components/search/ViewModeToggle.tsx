import { Rows3, SquareMenu, TableProperties } from 'lucide-react';

import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import { isSearchViewMode, type SearchViewMode } from '@/lib/search-view-mode';

interface ViewModeToggleProps {
  value: SearchViewMode;
  onChange: (value: SearchViewMode) => void;
}

export function ViewModeToggle({ value, onChange }: ViewModeToggleProps) {
  return (
    <ToggleGroup
      type="single"
      value={value}
      variant="outline"
      size="sm"
      onValueChange={(v) => {
        if (isSearchViewMode(v)) onChange(v);
      }}
      aria-label="Browse view mode"
    >
      <ToggleGroupItem value="full" aria-label="Full view">
        <SquareMenu className="h-4 w-4 mr-1.5" />
        Full
      </ToggleGroupItem>
      <ToggleGroupItem value="compact" aria-label="Compact view">
        <TableProperties className="h-4 w-4 mr-1.5" />
        Compact
      </ToggleGroupItem>
      <ToggleGroupItem value="list" aria-label="List view">
        <Rows3 className="h-4 w-4 mr-1.5" />
        List
      </ToggleGroupItem>
    </ToggleGroup>
  );
}
