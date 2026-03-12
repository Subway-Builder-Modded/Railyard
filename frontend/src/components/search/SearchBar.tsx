import { Input } from "@/components/ui/input";
import { Search, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { SEARCH_BAR_PLACEHOLDER } from "@/lib/search";

interface SearchBarProps {
  query: string;
  onQueryChange: (query: string) => void;
  placeholder?: string;
}

export function SearchBar({ query, onQueryChange, placeholder = SEARCH_BAR_PLACEHOLDER }: SearchBarProps) {
  return (
    <div className="relative">
      <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
      <Input
        placeholder={placeholder}
        value={query}
        onChange={(e) => onQueryChange(e.target.value)}
        className="pl-10 pr-10 h-10 bg-card border-border text-foreground placeholder:text-muted-foreground focus-visible:ring-ring"
        aria-label="Search mods and maps"
      />
      {query && (
        <Button
          variant="ghost"
          size="icon"
          className="absolute right-1 top-1/2 -translate-y-1/2 h-8 w-8 text-muted-foreground hover:text-foreground"
          onClick={() => onQueryChange("")}
          aria-label="Clear search"
        >
          <X className="h-3.5 w-3.5" />
        </Button>
      )}
    </div>
  );
}
