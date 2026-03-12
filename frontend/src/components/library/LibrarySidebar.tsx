import { cn } from "@/lib/utils";
import { MapPin, Package, LayoutGrid, Download } from "lucide-react";
import { Separator } from "@/components/ui/separator";
import {
  type LibraryFilterState,
  type LibraryTypeFilter,
} from "@/stores/library-store";
import { type Dispatch, type SetStateAction } from "react";

const FILTER_SECTION_TITLE_CLASS =
  "text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-2 px-1";

interface LibrarySidebarProps {
  filters: LibraryFilterState;
  onFiltersChange: Dispatch<SetStateAction<LibraryFilterState>>;
  modCount: number;
  mapCount: number;
  totalCount: number;
  updatesCount: number;
  onShowUpdatesOnly: () => void;
  showingUpdatesOnly: boolean;
}

const typeOptions: Array<{
  value: LibraryTypeFilter;
  label: string;
  icon: typeof MapPin;
}> = [
  { value: "all", label: "All", icon: LayoutGrid },
  { value: "mods", label: "Mods", icon: Package },
  { value: "maps", label: "Maps", icon: MapPin },
];

export function LibrarySidebar({
  filters,
  onFiltersChange,
  modCount,
  mapCount,
  totalCount,
  updatesCount,
  onShowUpdatesOnly,
  showingUpdatesOnly,
}: LibrarySidebarProps) {
  const counts: Record<LibraryTypeFilter, number> = {
    all: totalCount,
    mods: modCount,
    maps: mapCount,
  };

  return (
    <div className="space-y-5">
      {/* Type filter */}
      <div>
        <p className={FILTER_SECTION_TITLE_CLASS}>Type</p>
        <nav className="space-y-0.5" aria-label="Content type filter">
          {typeOptions.map(({ value, label, icon: Icon }) => (
            <button
              key={value}
              onClick={() => {
                onFiltersChange((prev) => ({ ...prev, type: value }));
              }}
              className={cn(
                "w-full flex items-center justify-between gap-2 px-3 py-2 rounded-md text-sm transition-colors",
                filters.type === value && !showingUpdatesOnly
                  ? "bg-accent text-accent-foreground font-medium"
                  : "text-muted-foreground hover:text-foreground hover:bg-accent/60",
              )}
              aria-current={
                filters.type === value && !showingUpdatesOnly
                  ? "true"
                  : undefined
              }
            >
              <span className="flex items-center gap-2">
                <Icon className="h-3.5 w-3.5 shrink-0" />
                {label}
              </span>
              <span
                className={cn(
                  "text-xs tabular-nums",
                  filters.type === value && !showingUpdatesOnly
                    ? "text-foreground"
                    : "text-muted-foreground",
                )}
              >
                {counts[value]}
              </span>
            </button>
          ))}
        </nav>
      </div>

      {/* Updates available filter */}
      {updatesCount > 0 && (
        <>
          <Separator />
          <div>
            <p className={FILTER_SECTION_TITLE_CLASS}>Updates</p>
            <button
              onClick={onShowUpdatesOnly}
              className={cn(
                "w-full flex items-center justify-between gap-2 px-3 py-2 rounded-md text-sm transition-colors",
                showingUpdatesOnly
                  ? "bg-accent text-accent-foreground font-medium"
                  : "text-muted-foreground hover:text-foreground hover:bg-accent/60",
              )}
            >
              <span className="flex items-center gap-2">
                <Download className="h-3.5 w-3.5 shrink-0" />
                Updates available
              </span>
              <span
                className={cn(
                  "text-xs tabular-nums",
                  showingUpdatesOnly
                    ? "text-foreground"
                    : "text-muted-foreground",
                )}
              >
                {updatesCount}
              </span>
            </button>
          </div>
        </>
      )}
    </div>
  );
}
