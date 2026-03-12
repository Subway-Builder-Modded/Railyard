import { cn } from "@/lib/utils";
import {
  MapPin,
  Package,
  Download,
  Tag,
  GraduationCap,
  BadgeCheck,
  Layers3,
} from "lucide-react";
import { Separator } from "@/components/ui/separator";
import { Checkbox } from "@/components/ui/checkbox";
import {
  type LibraryFilterState,
  type LibraryTypeFilter,
} from "@/stores/library-store";
import { type Dispatch, type SetStateAction, type ComponentType } from "react";
import {
  formatSourceQuality,
  LEVEL_OF_DETAIL_VALUES,
  LOCATION_TAGS,
  SOURCE_QUALITY_VALUES,
} from "@/lib/map-filter-values";
import { SEARCH_FILTER_EMPTY_LABELS } from "@/lib/search";

const FILTER_SECTION_TITLE_CLASS =
  "text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-2 px-1";
const FILTER_SECTION_OPTION_CLASS =
  "w-full flex items-center gap-2 px-2 py-1.5 rounded-md text-left text-sm text-muted-foreground hover:text-foreground hover:bg-accent/60 transition-colors";
const FILTER_SECTION_CLEAR_CLASS =
  "mt-2 text-xs text-muted-foreground hover:text-foreground transition-colors";

interface LibrarySidebarProps {
  filters: LibraryFilterState;
  onFiltersChange: Dispatch<SetStateAction<LibraryFilterState>>;
  modCount: number;
  mapCount: number;
  updatesCount: number;
  onShowUpdatesOnly: () => void;
  showingUpdatesOnly: boolean;
  availableTags: string[];
  availableSpecialDemand: string[];
}

const typeOptions: Array<{
  value: LibraryTypeFilter;
  label: string;
  icon: typeof MapPin;
}> = [
  { value: "mods", label: "Mods", icon: Package },
  { value: "maps", label: "Maps", icon: MapPin },
];

export function LibrarySidebar({
  filters,
  onFiltersChange,
  modCount,
  mapCount,
  updatesCount,
  onShowUpdatesOnly,
  showingUpdatesOnly,
  availableTags,
  availableSpecialDemand,
}: LibrarySidebarProps) {
  const counts: Record<LibraryTypeFilter, number> = {
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
              onClick={() =>
                onFiltersChange((prev) => ({ ...prev, type: value }))
              }
              className={cn(
                "w-full flex items-center justify-between gap-2 px-3 py-2 rounded-md text-sm transition-colors",
                filters.type === value
                  ? "bg-accent text-accent-foreground font-medium"
                  : "text-muted-foreground hover:text-foreground hover:bg-accent/60",
              )}
              aria-current={filters.type === value ? "true" : undefined}
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

      {/* Mod tags filter */}
      {filters.type === "mods" && (
        <>
          <Separator />
          <ChecklistFilterSection
            title="Tag"
            icon={Tag}
            values={availableTags}
            selected={filters.mod.tags}
            onChange={(values) =>
              onFiltersChange((prev) => ({
                ...prev,
                mod: { ...prev.mod, tags: values },
              }))
            }
            emptyLabel={SEARCH_FILTER_EMPTY_LABELS.tags}
          />
        </>
      )}

      {filters.type === "maps" && (
        <>
          <Separator />
          <ChecklistFilterSection
            title="Location"
            icon={MapPin}
            values={LOCATION_TAGS}
            selected={filters.map.locations}
            onChange={(values) =>
              onFiltersChange((prev) => ({
                ...prev,
                map: { ...prev.map, locations: values },
              }))
            }
          />
          <ChecklistFilterSection
            title="Source Quality"
            icon={BadgeCheck}
            values={SOURCE_QUALITY_VALUES}
            selected={filters.map.sourceQuality}
            formatValue={formatSourceQuality}
            onChange={(values) =>
              onFiltersChange((prev) => ({
                ...prev,
                map: { ...prev.map, sourceQuality: values },
              }))
            }
          />
          <ChecklistFilterSection
            title="Level of Detail"
            icon={Layers3}
            values={LEVEL_OF_DETAIL_VALUES}
            selected={filters.map.levelOfDetail}
            onChange={(values) =>
              onFiltersChange((prev) => ({
                ...prev,
                map: { ...prev.map, levelOfDetail: values },
              }))
            }
          />
          <ChecklistFilterSection
            title="Special Demand"
            icon={GraduationCap}
            values={availableSpecialDemand}
            selected={filters.map.specialDemand}
            onChange={(values) =>
              onFiltersChange((prev) => ({
                ...prev,
                map: { ...prev.map, specialDemand: values },
              }))
            }
            emptyLabel={SEARCH_FILTER_EMPTY_LABELS.specialDemand}
          />
        </>
      )}

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

interface ChecklistFilterSectionProps {
  title: string;
  values: readonly string[];
  selected: string[];
  icon: ComponentType<{ className?: string }>;
  onChange: (values: string[]) => void;
  emptyLabel?: string;
  formatValue?: (value: string) => string;
}

function ChecklistFilterSection({
  title,
  icon: Icon,
  values,
  selected,
  onChange,
  emptyLabel = SEARCH_FILTER_EMPTY_LABELS.generic,
  formatValue = (value) => value,
}: ChecklistFilterSectionProps) {
  const toggle = (value: string) => {
    onChange(
      selected.includes(value)
        ? selected.filter((v) => v !== value)
        : [...selected, value],
    );
  };

  return (
    <div>
      <FilterSectionTitle title={title} icon={Icon} />
      {values.length === 0 ? (
        <p className="text-xs text-muted-foreground px-1">{emptyLabel}</p>
      ) : (
        <div className="space-y-1">
          {values.map((value) => (
            <button
              key={value}
              type="button"
              onClick={() => toggle(value)}
              className={FILTER_SECTION_OPTION_CLASS}
            >
              <Checkbox checked={selected.includes(value)} aria-hidden="true" />
              <span>{formatValue(value)}</span>
            </button>
          ))}
        </div>
      )}
      {selected.length > 0 && (
        <button
          type="button"
          onClick={() => onChange([])}
          className={FILTER_SECTION_CLEAR_CLASS}
        >
          Clear {title.toLowerCase()}
        </button>
      )}
    </div>
  );
}

interface FilterSectionTitleProps {
  title: string;
  icon?: ComponentType<{ className?: string }>;
}

function FilterSectionTitle({ title, icon: Icon }: FilterSectionTitleProps) {
  return (
    <p
      className={cn(
        FILTER_SECTION_TITLE_CLASS,
        Icon && "flex items-center gap-1.5",
      )}
    >
      {Icon && <Icon className="h-3.5 w-3.5" />}
      {title}
    </p>
  );
}
