import * as React from 'react';

import { cn } from '@/lib/utils';

type GridPreset = 'default' | 'compact';

const presetToMinWidth: Record<GridPreset, string> = {
  default: '18rem',
  compact: '16rem',
};

const presetToGapClassName: Record<GridPreset, string> = {
  default: 'gap-4',
  compact: 'gap-3',
};

type ResponsiveCardGridProps = Omit<
  React.ComponentProps<'div'>,
  'style' | 'children'
> & {
  preset?: GridPreset;
  minItemWidth?: number | string;
  style?: React.CSSProperties;
  children: React.ReactNode;
};

export function ResponsiveCardGrid({
  preset = 'default',
  minItemWidth,
  className,
  style,
  children,
  ...props
}: ResponsiveCardGridProps) {
  const resolvedMinWidth =
    minItemWidth === undefined
      ? presetToMinWidth[preset]
      : typeof minItemWidth === 'number'
        ? `${minItemWidth}px`
        : minItemWidth;

  return (
    <div
      data-slot="card-grid"
      className={cn(
        'grid items-stretch [grid-template-columns:repeat(auto-fit,minmax(var(--card-grid-min),1fr))]',
        presetToGapClassName[preset],
        className,
      )}
      style={
        {
          ...(style ?? {}),
          ['--card-grid-min' as unknown as string]: resolvedMinWidth,
        } as React.CSSProperties
      }
      {...props}
    >
      {children}
    </div>
  );
}
