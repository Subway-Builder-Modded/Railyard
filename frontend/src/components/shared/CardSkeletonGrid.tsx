import { Skeleton } from '@/components/ui/skeleton';

import { ResponsiveCardGrid } from './ResponsiveCardGrid';

interface CardSkeletonGridProps {
  count?: number;
  preset?: 'default' | 'compact';
}

export function CardSkeletonGrid({
  count = 6,
  preset = 'default',
}: CardSkeletonGridProps) {
  return (
    <ResponsiveCardGrid preset={preset}>
      {Array.from({ length: count }).map((_, i) => (
        <div
          key={i}
          className="bg-card border border-border rounded-lg overflow-hidden flex flex-col"
        >
          {/* Thumbnail skeleton */}
          <Skeleton className="aspect-video w-full rounded-none" />

          {/* Body skeleton */}
          <div className="p-4 flex flex-col gap-3">
            {/* Title row */}
            <div className="flex items-start justify-between gap-2">
              <div className="flex-1 space-y-1.5">
                <Skeleton className="h-4 w-3/4" />
                <Skeleton className="h-3 w-1/3" />
              </div>
            </div>

            {/* Description lines */}
            <div className="space-y-1.5">
              <Skeleton className="h-3 w-full" />
              <Skeleton className="h-3 w-4/5" />
            </div>

            {/* Footer: tags */}
            <div className="flex items-center justify-end gap-1 mt-1">
              <Skeleton className="h-4 w-12 rounded-full" />
              <Skeleton className="h-4 w-16 rounded-full" />
              <Skeleton className="h-4 w-10 rounded-full" />
            </div>
          </div>
        </div>
      ))}
    </ResponsiveCardGrid>
  );
}
