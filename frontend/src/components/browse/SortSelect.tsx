import { useEffect } from 'react';

import { SortOptionIcon } from '@/components/browse/sort-icons';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select';
import type { AssetType } from '@/lib/asset-types';
import {
  DEFAULT_SORT_STATE,
  getSortOptionsForType,
  SortKey,
  sortKeyToState,
  type SortState,
  sortStateToOptionKey,
} from '@/lib/constants';
import { cn } from '@/lib/utils';

interface SortSelectProps {
  value: SortState;
  onChange: (value: SortState) => void;
  tab: AssetType;
}

export function SortSelect({ value, onChange, tab }: SortSelectProps) {
  const sortOptions = getSortOptionsForType(tab);
  const selectedOptionKey = sortStateToOptionKey(value, tab);
  const selectedOption =
    sortOptions.find((opt) => opt.value === selectedOptionKey) ?? sortOptions[0];

  // Reset to default if current value is not available in filtered options
  useEffect(() => {
    if (!sortOptions.some((opt) => opt.value === selectedOptionKey)) {
      const defaultKey = SortKey.fromState(DEFAULT_SORT_STATE);
      const defaultOption =
        sortOptions.find((opt) => SortKey.equals(opt.value, defaultKey)) ??
        sortOptions[0];
      if (defaultOption) {
        onChange(defaultOption.sort);
      }
    }
  }, [onChange, selectedOptionKey, sortOptions]);

  return (
    <Select
      value={selectedOptionKey}
      onValueChange={(v) => onChange(sortKeyToState(v))}
    >
      <SelectTrigger
        size="sm"
        className={cn(
          'h-8 min-w-[11.5rem] justify-between rounded-xl border border-border/70 bg-background/90 px-3 text-xs font-semibold text-muted-foreground shadow-sm backdrop-blur-md',
          'hover:bg-accent/45 hover:text-primary data-[state=open]:bg-accent/45 data-[state=open]:text-primary',
          // Ensure the chevron + option icon inherit the same active color as the label.
          '[&_svg]:text-current',
        )}
      >
        {selectedOption ? (
          <span className="flex min-w-0 items-center gap-2">
            <SortOptionIcon option={selectedOption} />
            <span className="min-w-0 flex-1 truncate">{selectedOption.label}</span>
          </span>
        ) : (
          <span className="text-muted-foreground">Sort</span>
        )}
      </SelectTrigger>
      {/* Make sure that the selected option is always visible and ensure the dropdown renders downwards */}
      <SelectContent
        side="bottom"
        sideOffset={4}
        position="popper"
        align="end"
        avoidCollisions={false}
        className="max-h-72 overflow-y-auto rounded-xl border border-border/70 bg-background/95 p-1 shadow-lg backdrop-blur-md"
      >
        {sortOptions.map((opt) => (
          <SelectItem
            key={opt.value}
            value={opt.value}
            className={cn(
              'rounded-lg text-sm',
              'data-[highlighted]:bg-accent/45 data-[highlighted]:text-primary',
              'data-[state=checked]:bg-accent/35 data-[state=checked]:text-primary',
            )}
          >
            <span className="flex items-center gap-2">
              <SortOptionIcon option={opt} />
              <span>{opt.label}</span>
            </span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
