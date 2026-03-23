import {
  ArrowDown,
  ArrowUp,
  Calendar,
  Download,
  Globe,
  Hash,
  Shuffle,
  Type,
  User,
  Users,
} from 'lucide-react';
import { type ComponentType, useEffect } from 'react';

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select';
import type { AssetType } from '@/lib/asset-types';
import {
  DEFAULT_SORT_STATE,
  getSortOptionsForType,
  type SortField,
  type SortState,
} from '@/lib/constants';
import { cn } from '@/lib/utils';

const TEXT_FIELDS = new Set(['name', 'city_code', 'country', 'author']);

const FIELD_ICONS: Record<SortField, ComponentType<{ className?: string }>> = {
  name: Type,
  city_code: Hash,
  country: Globe,
  author: User,
  population: Users,
  downloads: Download,
  last_updated: Calendar,
  random: Shuffle,
};

function directionArrow(field: SortField, direction: 'asc' | 'desc') {
  const invert = TEXT_FIELDS.has(field);
  const showUp = invert ? direction === 'desc' : direction === 'asc';
  return showUp
    ? <ArrowUp className="h-3.5 w-3.5" aria-hidden />
    : <ArrowDown className="h-3.5 w-3.5" aria-hidden />;
}

interface SortSelectProps {
  value: SortState;
  onChange: (value: SortState) => void;
  tab: AssetType;
}

export function SortSelect({ value, onChange, tab }: SortSelectProps) {
  const sortOptions = getSortOptionsForType(tab);

  const fieldOptions = sortOptions.reduce<Array<{ field: SortField; label: string }>>(
    (acc, opt) => {
      if (!acc.some((f) => f.field === opt.sort.field)) {
        acc.push({ field: opt.sort.field, label: opt.label });
      }
      return acc;
    },
    [],
  );

  const currentFieldValid = fieldOptions.some((f) => f.field === value.field);
  const currentField = currentFieldValid
    ? value.field
    : (fieldOptions[0]?.field ?? DEFAULT_SORT_STATE.field);
  const isRandom = currentField === 'random';

  useEffect(() => {
    if (!fieldOptions.some((f) => f.field === value.field)) {
      onChange(DEFAULT_SORT_STATE);
    }
  }, [tab]);

  const handleFieldChange = (field: string) => {
    if (field === 'random') {
      onChange({ field: 'random', direction: 'asc' });
    } else {
      onChange({ field: field as SortField, direction: value.direction });
    }
  };

  const handleDirectionToggle = () => {
    onChange({ field: value.field, direction: value.direction === 'asc' ? 'desc' : 'asc' });
  };

  return (
    <div className="flex items-center overflow-hidden rounded-xl border border-border/70 bg-background/90 shadow-sm backdrop-blur-md">
      {/* Field dropdown */}
      <Select value={currentField} onValueChange={handleFieldChange}>
        <SelectTrigger
          size="sm"
          className={cn(
            'border-0 bg-transparent shadow-none',
            'dark:bg-transparent dark:hover:bg-transparent',
            'h-8 min-w-[8.5rem] gap-2 px-3',
            'text-xs font-semibold text-muted-foreground',
            'hover:bg-accent/45 hover:text-primary dark:hover:bg-accent/45',
            'data-[state=open]:bg-accent/45 data-[state=open]:text-primary',
            '[&_svg]:!text-current',
            !isRandom && 'rounded-none border-r border-border/60',
          )}
        >
          {(() => {
            const Icon = FIELD_ICONS[currentField];
            return (
              <span className="flex min-w-0 items-center gap-2">
                <Icon className="h-3.5 w-3.5 shrink-0 text-current" aria-hidden />
                <span className="min-w-0 truncate">
                  {fieldOptions.find((f) => f.field === currentField)?.label ?? 'Sort'}
                </span>
              </span>
            );
          })()}
        </SelectTrigger>
        <SelectContent
          side="bottom"
          sideOffset={4}
          position="popper"
          align="end"
          avoidCollisions={false}
          className="rounded-xl border border-border/70 bg-background/95 p-1 shadow-lg backdrop-blur-md"
        >
          {fieldOptions.map((opt) => {
            const Icon = FIELD_ICONS[opt.field];
            return (
              <SelectItem
                key={opt.field}
                value={opt.field}
                className={cn(
                  'rounded-lg text-sm',
                  'data-[highlighted]:bg-accent/45 data-[highlighted]:text-primary',
                  'data-[state=checked]:bg-accent/35 data-[state=checked]:text-primary',
                )}
              >
                <span className="flex items-center gap-2">
                  <Icon className="h-4 w-4 shrink-0" aria-hidden />
                  <span>{opt.label}</span>
                </span>
              </SelectItem>
            );
          })}
        </SelectContent>
      </Select>

      {!isRandom && (
        <button
          type="button"
          onClick={handleDirectionToggle}
          aria-label={
            value.direction === 'asc'
              ? 'Sort ascending — click to sort descending'
              : 'Sort descending — click to sort ascending'
          }
          className={cn(
            'flex h-8 w-8 items-center justify-center',
            'bg-transparent text-muted-foreground transition-colors',
            'hover:bg-accent/45 hover:text-primary',
          )}
        >
          {directionArrow(value.field, value.direction)}
        </button>
      )}
    </div>
  );
}
