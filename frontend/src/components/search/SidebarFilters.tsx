import { cn } from "@/lib/utils";
import {
  MapPin,
  Package,
  Tag,
  GraduationCap,
  BadgeCheck,
  Layers3,
} from "lucide-react";
import { Separator } from "@/components/ui/separator";
import { Checkbox } from "@/components/ui/checkbox";
import { type ComponentType, type Dispatch, type SetStateAction } from "react";
import {
  formatSourceQuality,
  LEVEL_OF_DETAIL_VALUES,
  LOCATION_TAGS,
  SOURCE_QUALITY_VALUES,
} from "@/lib/map-filter-values";
import { SEARCH_FILTER_EMPTY_LABELS } from "@/lib/search";
import { type SearchFilterState } from "@/stores/search-store";

const FILTER_SECTION_TITLE_CLASS =
  "text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-2 px-1";
const FILTER_SECTION_OPTION_CLASS =
  "w-full flex items-center gap-2 px-2 py-1.5 rounded-md text-left text-sm text-muted-foreground hover:text-foreground hover:bg-accent/60 transition-colors";
const FILTER_SECTION_CLEAR_CLASS =
  "mt-2 text-xs text-muted-foreground hover:text-foreground transition-colors";

interface SidebarFiltersProps {
  filters: SearchFilterState;
  onFiltersChange: Dispatch<SetStateAction<SearchFilterState>>;
  availableTags: string[];
  availableSpecialDemand: string[];
  modCount: number;
  mapCount: number;
}

const typeOptions = [
  { value: "maps" as const, label: "Maps", icon: MapPin },
  { value: "mods" as const, label: "Mods", icon: Package },
];

export function SidebarFilters({
  filters,
  onFiltersChange,
  availableTags,
  availableSpecialDemand,
  modCount,
  mapCount,
}: SidebarFiltersProps) {
  const counts: Record<"mods" | "maps", number> = {
    mods: modCount,
    maps: mapCount,
  };

  return (
    <div className="space-y-5">
      {/* Type filter */}
      <div>
        <FilterSectionTitle title="Type" />
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
                  filters.type === value
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

      {filters.type !== "maps" && (
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

      {filters.type !== "mods" && (
        <>
          <Separator />
          <ChecklistFilterSection
            title="Location"
            icon={MapPin}
            values={[...LOCATION_TAGS]}
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
            values={[...SOURCE_QUALITY_VALUES]}
            formatValue={formatSourceQuality}
            selected={filters.map.sourceQuality}
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
            values={[...LEVEL_OF_DETAIL_VALUES]}
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
    </div>
  );
}

interface FilterSectionProperties {
  title: string;
  values: string[];
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
}: FilterSectionProperties) {
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

interface TitleProperties {
  title: string;
  icon?: ComponentType<{ className?: string }>;
}

function FilterSectionTitle({ title, icon: Icon }: TitleProperties) {
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
