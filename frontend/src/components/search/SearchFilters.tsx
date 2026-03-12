import { Input } from "@/components/ui/input";
import { Search } from "lucide-react";
import { TypeToggle } from "./TypeToggle";
import { TagsFilter } from "./TagsFilter";
import { SortSelect } from "./SortSelect";
import type { SortState } from "@/lib/constants";
import type { AssetType } from "@/lib/asset-types";

interface SearchFiltersProps {
  query: string;
  onQueryChange: (query: string) => void;
  type: AssetType;
  onTypeChange: (type: AssetType) => void;
  availableTags: string[];
  selectedTags: string[];
  onTagsChange: (tags: string[]) => void;
  sort: SortState;
  onSortChange: (sort: SortState) => void;
}

export function SearchFilters({
  query,
  onQueryChange,
  type,
  onTypeChange,
  availableTags,
  selectedTags,
  onTagsChange,
  sort,
  onSortChange,
}: SearchFiltersProps) {
  return (
    <div className="space-y-4">
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input
          placeholder="Search mods and maps..."
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
          className="pl-10"
        />
      </div>
      <div className="flex flex-wrap items-center gap-3">
        <TypeToggle value={type} onChange={onTypeChange} />
        <div className="h-6 w-px bg-border" />
        <TagsFilter
          availableTags={availableTags}
          selectedTags={selectedTags}
          onChange={onTagsChange}
        />
        <div className="flex-1" />
        <SortSelect value={sort} onChange={onSortChange} tab={type} />
      </div>
    </div>
  );
}
