import { MapPin, Package } from 'lucide-react';

import { SidebarFilters, type SidebarFiltersProps } from '@/components/shared/SidebarFilters';
import { SidebarPanel, SIDEBAR_CONTENT_OFFSET } from '@/components/shared/SidebarPanel';
import type { AssetType } from '@/lib/asset-types';
import { cn } from '@/lib/utils';

export { SIDEBAR_CONTENT_OFFSET };

export interface AssetSidebarPanelProps extends SidebarFiltersProps {
  open: boolean;
  onToggle: () => void;
  ariaLabel: string;
}

const TYPE_BUTTONS: Array<{ type: AssetType; icon: typeof MapPin; label: string }> = [
  { type: 'map', icon: MapPin, label: 'Show maps' },
  { type: 'mod', icon: Package, label: 'Show mods' },
];

export function AssetSidebarPanel({ open, onToggle, ariaLabel, ...filterProps }: AssetSidebarPanelProps) {
  const currentType = filterProps.filters.type;

  const collapsedContent = (
    <>
      {TYPE_BUTTONS.map(({ type, icon: Icon, label }) => {
        const isCurrent = currentType === type;
        return (
          <button
            key={type}
            type="button"
            onClick={() => filterProps.onTypeChange(type)}
            aria-label={label}
            aria-current={isCurrent ? 'true' : undefined}
            className={cn(
              'relative flex h-10 w-full items-center justify-center transition-colors',
              'hover:bg-accent/45 hover:text-primary',
              isCurrent ? 'text-primary' : '',
            )}
          >
            <Icon className="h-4 w-4" />
            {isCurrent && (
              <span
                aria-hidden
                className="absolute right-0 top-1 bottom-1 w-[3px] rounded-full bg-primary"
              />
            )}
          </button>
        );
      })}
    </>
  );

  return (
    <SidebarPanel
      open={open}
      onToggle={onToggle}
      ariaLabel={ariaLabel}
      filters={filterProps.filters}
      collapsedContent={collapsedContent}
    >
      <SidebarFilters {...filterProps} />
    </SidebarPanel>
  );
}
