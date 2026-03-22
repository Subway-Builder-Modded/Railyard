import {
  BadgeCheck,
  GraduationCap,
  Layers3,
  MapPin,
  Package,
  Tag,
} from 'lucide-react';
import { type ComponentType, type Dispatch, type SetStateAction } from 'react';

import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Separator } from '@/components/ui/separator';
import type { AssetType } from '@/lib/asset-types';
import { filterVisibleListingValues } from '@/lib/listing-counts';
import {
  formatSourceQuality,
  LEVEL_OF_DETAIL_VALUES,
  LOCATION_TAGS,
  SOURCE_QUALITY_VALUES,
} from '@/lib/map-filter-values';
import { SEARCH_FILTER_EMPTY_LABELS } from '@/lib/search';
import { cn } from '@/lib/utils';
import type { SearchFilterState } from '@/stores/search-store';

const FILTER_SECTION_TITLE_CLASS =
  'text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-2 px-1';
const FILTER_SECTION_CLEAR_CLASS = 'mt-2';

interface LibrarySidebarProps {
  filters: SearchFilterState;
  onFiltersChange: Dispatch<SetStateAction<SearchFilterState>>;
  onTypeChange: (type: AssetType) => void;
  modCount: number;
  mapCount: number;
  availableTags: string[];
  availableSpecialDemand: string[];
  modTagCounts: Record<string, number>;
  mapLocationCounts: Record<string, number>;
  mapSourceQualityCounts: Record<string, number>;
  mapLevelOfDetailCounts: Record<string, number>;
  mapSpecialDemandCounts: Record<string, number>;
}

const typeOptions: Array<{
  value: AssetType;
  label: string;
  icon: typeof MapPin;
}> = [
  { value: 'map', label: 'Maps', icon: MapPin },
  { value: 'mod', label: 'Mods', icon: Package },
];

export function LibrarySidebar({
  filters,
  onFiltersChange,
  onTypeChange,
  modCount,
  mapCount,
  availableTags,
  availableSpecialDemand,
  modTagCounts,
  mapLocationCounts,
  mapSourceQualityCounts,
  mapLevelOfDetailCounts,
  mapSpecialDemandCounts,
}: LibrarySidebarProps) {
  const counts: Record<AssetType, number> = {
    mod: modCount,
    map: mapCount,
  };

  return (
    <div className="space-y-5">
      <div>
        <p className={FILTER_SECTION_TITLE_CLASS}>Type</p>
        <nav className="space-y-0.5" aria-label="Content type filter">
          {typeOptions.map(({ value, label, icon: Icon }) => (
            <Button
              key={value}
              onClick={() => onTypeChange(value)}
              intent="plain"
              size="sm"
              className={cn(
                'w-full justify-between font-medium',
                filters.type === value
                  ? 'bg-muted/70 text-foreground border-border/70'
                  : 'text-muted-foreground hover:text-foreground',
              )}
              aria-current={filters.type === value ? 'true' : undefined}
            >
              <span className="flex items-center gap-2">
                <Icon className="h-3.5 w-3.5 shrink-0" />
                {label}
              </span>
              <span
                className={cn(
                  'text-xs tabular-nums',
                  filters.type === value
                    ? 'text-foreground'
                    : 'text-muted-foreground',
                )}
              >
                {counts[value]}
              </span>
            </Button>
          ))}
        </nav>
      </div>

      {filters.type === 'mod' && (
        <>
          <Separator />
          <ChecklistFilterSection
            title="Tag"
            icon={Tag}
            values={availableTags}
            counts={modTagCounts}
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

      {filters.type === 'map' && (
        <>
          <Separator />
          <ChecklistFilterSection
            title="Location"
            icon={MapPin}
            values={LOCATION_TAGS}
            counts={mapLocationCounts}
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
            counts={mapSourceQualityCounts}
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
            counts={mapLevelOfDetailCounts}
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
            counts={mapSpecialDemandCounts}
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

interface ChecklistFilterSectionProps {
  title: string;
  values: readonly string[];
  counts: Record<string, number>;
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
  counts,
  selected,
  onChange,
  emptyLabel = SEARCH_FILTER_EMPTY_LABELS.generic,
  formatValue = (value) => value,
}: ChecklistFilterSectionProps) {
  const visibleValues = filterVisibleListingValues(values, counts, selected);

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
      {visibleValues.length === 0 ? (
        <p className="text-xs text-muted-foreground px-1">{emptyLabel}</p>
      ) : (
        <div className="space-y-1">
          {visibleValues.map((value) => (
            <Button
              key={value}
              type="button"
              onClick={() => toggle(value)}
              intent="plain"
              size="xs"
              className={cn(
                'w-full justify-between px-2 font-normal text-sm',
                selected.includes(value)
                  ? 'bg-muted/60 text-foreground border-border/60'
                  : 'text-muted-foreground hover:text-foreground',
              )}
            >
              <span className="flex items-center gap-2">
                <Checkbox
                  checked={selected.includes(value)}
                  aria-hidden="true"
                />
                <span>{formatValue(value)}</span>
              </span>
              <span className="text-xs tabular-nums text-muted-foreground">
                {counts[value] ?? 0}
              </span>
            </Button>
          ))}
        </div>
      )}
      {selected.length > 0 && (
        <div className={FILTER_SECTION_CLEAR_CLASS}>
          <Button
            type="button"
            intent="link"
            size="xs"
            onClick={() => onChange([])}
            className="h-auto min-h-0 px-0 py-0 text-xs font-medium text-muted-foreground hover:text-foreground"
          >
            Clear {title.toLowerCase()}
          </Button>
        </div>
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
        Icon && 'flex items-center gap-1.5',
      )}
    >
      {Icon && <Icon className="h-3.5 w-3.5" />}
      {title}
    </p>
  );
}
