import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  getSortOptionsForType,
  sortKeyToState,
  sortStateToOptionKey,
  type SortState,
  type ListingType,
} from "@/lib/constants";
import { useEffect } from "react";

interface SortSelectProps {
  value: SortState;
  onChange: (value: SortState) => void;
  tab: ListingType;
}

export function SortSelect({ value, onChange, tab }: SortSelectProps) {
  const sortOptions = getSortOptionsForType(tab);
  const selectedOptionKey = sortStateToOptionKey(value, tab);

  // Reset to default if current value is not available in filtered options
  useEffect(() => {
    if (!sortOptions.some((opt) => opt.value === selectedOptionKey)) {
      onChange(sortOptions[0].sort);
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
      {/* Make sure that the selected option is always visible and ensure the dropdown renders downwards */}
      <SelectContent
        side="bottom"
        sideOffset={4}
        position="popper"
        align="end"
        avoidCollisions={false}
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
