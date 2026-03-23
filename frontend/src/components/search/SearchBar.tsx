import { Search, X } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { SEARCH_BAR_PLACEHOLDER } from '@/lib/search';

interface SearchBarProps {
  query: string;
  onQueryChange: (query: string) => void;
}

export function SearchBar({ query, onQueryChange }: SearchBarProps) {
  return (
    <div className="relative group">
      <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none transition-colors group-focus-within:text-primary" />
      <input
        placeholder={SEARCH_BAR_PLACEHOLDER}
        value={query}
        onChange={(e) => onQueryChange(e.target.value)}
        aria-label="Search mods and maps"
        className="h-11 w-full rounded-xl border border-border bg-card pl-11 pr-11 text-sm text-foreground shadow-xs placeholder:text-muted-foreground transition-[border-color,box-shadow] outline-none focus:border-ring focus:ring-[3px] focus:ring-ring/25 dark:bg-input/30"
      />
      {query && (
        <Button
          variant="ghost"
          size="icon"
          className="absolute right-1.5 top-1/2 -translate-y-1/2 h-8 w-8 text-muted-foreground hover:text-foreground"
          onClick={() => onQueryChange('')}
          aria-label="Clear search"
        >
          <X className="h-3.5 w-3.5" />
        </Button>
      )}
    </div>
  );
}
