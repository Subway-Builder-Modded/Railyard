import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { SORT_OPTIONS, type SortOption } from "@/lib/constants";
import { useEffect } from "react";

interface SortSelectProps {
  value: SortOption;
  onChange: (value: SortOption) => void;
  tab: string
}

export function SortSelect({ value, onChange, tab }: SortSelectProps) {
  let sortOptions = [...SORT_OPTIONS];
  if(tab !== "maps") {
    sortOptions = sortOptions.filter(opt => opt.value !== "population-desc");
  }

  // Reset to default if current value is not available in filtered options
  useEffect(() => {
    if (!sortOptions.some(opt => opt.value === value)) {
      onChange(sortOptions[0].value as SortOption);
    }
  }, [tab, value, onChange, sortOptions]);

  return (
    <Select value={value} onValueChange={(v) => onChange(v as SortOption)}>
      <SelectTrigger className="w-36 h-8 text-xs">
        <SelectValue placeholder="Sort by..." />
      </SelectTrigger>
      <SelectContent>
        {sortOptions.map((opt) => (
          <SelectItem key={opt.value} value={opt.value}>
            {opt.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
