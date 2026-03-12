import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  DEFAULT_SORT_STATE,
  getSortOptionsForType,
  SortKey,
  sortKeyToState,
  sortStateToOptionKey,
  type SortState,
} from "@/lib/constants";
import { useEffect } from "react";
import type { AssetType } from "@/lib/asset-types";

interface SortSelectProps {
  value: SortState;
  onChange: (value: SortState) => void;
  tab: AssetType;
}

export function SortSelect({ value, onChange, tab }: SortSelectProps) {
  const sortOptions = getSortOptionsForType(tab);
  const selectedOptionKey = sortStateToOptionKey(value, tab);

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
      <SelectTrigger className="w-36 h-8 text-xs">
        <SelectValue placeholder="Sort by..." />
      </SelectTrigger>
      <SelectContent
        side="bottom"
        sideOffset={4}
        position="popper"
        align="end"
        avoidCollisions={false}
        className="max-h-72 overflow-y-auto"
      >
        {sortOptions.map((opt) => (
          <SelectItem key={opt.value} value={opt.value}>
            {opt.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
