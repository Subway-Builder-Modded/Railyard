import {
  ArrowDown,
  ArrowDown10,
  ArrowDownAZ,
  ArrowDownZA,
  ArrowUp,
  ArrowUp10,
  CalendarArrowDown,
  CalendarArrowUp,
  Shuffle,
} from 'lucide-react';

import type { SortOption } from '@/lib/constants';

export function SortOptionIcon({ option }: { option: SortOption }) {
  const { field, direction } = option.sort;

  const baseIconClass = 'h-4 w-4 text-current';

  if (field === 'random') return <Shuffle className={baseIconClass} aria-hidden />;

  if (field === 'last_updated') {
    const Icon = direction === 'asc' ? CalendarArrowUp : CalendarArrowDown;
    return <Icon className={baseIconClass} aria-hidden />;
  }

  if (field === 'downloads' || field === 'population') {
    const Icon = direction === 'asc' ? ArrowUp10 : ArrowDown10;
    return <Icon className={baseIconClass} aria-hidden />;
  }

  if (field === 'name' || field === 'city_code' || field === 'country' || field === 'author') {
    const Icon = direction === 'asc' ? ArrowDownAZ : ArrowDownZA;
    return <Icon className={baseIconClass} aria-hidden />;
  }

  return direction === 'asc' ? (
    <ArrowUp className={baseIconClass} aria-hidden />
  ) : (
    <ArrowDown className={baseIconClass} aria-hidden />
  );
}
