import { useEffect, useRef, useState } from 'react';

import { ErrorBanner } from '@/components/shared/ErrorBanner';
import { ItemCard } from '@/components/shared/ItemCard';
import { Skeleton } from '@/components/ui/skeleton';
import type { AssetType } from '@/lib/asset-types';
import type { TaggedItem } from '@/lib/tagged-items';

const DISCOVER_CARD_MIN_WIDTH = 220;
const DISCOVER_CARD_GAP = 12;
const DISCOVER_MIN_LAYOUT_WIDTH =
  DISCOVER_CARD_MIN_WIDTH * 2 + DISCOVER_CARD_GAP;

interface DiscoverSectionGridProps {
  items: TaggedItem[];
  getInstalledVersion: (id: string) => string | null;
  getTotalDownloads: (type: AssetType, id: string) => number;
  loading: boolean;
  error?: string | null;
  emptyMessage: string;
}

export function DiscoverSectionGrid({
  items,
  getInstalledVersion,
  getTotalDownloads,
  loading,
  error,
  emptyMessage,
}: DiscoverSectionGridProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [visibleCount, setVisibleCount] = useState(1);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const updateVisibleCount = () => {
      const width = container.clientWidth;
      if (width <= 0) return;

      const effectiveWidth = Math.max(width, DISCOVER_MIN_LAYOUT_WIDTH);
      const nextCount = Math.max(
        1,
        Math.floor(
          (effectiveWidth + DISCOVER_CARD_GAP) /
            (DISCOVER_CARD_MIN_WIDTH + DISCOVER_CARD_GAP),
        ),
      );
      setVisibleCount(nextCount);
    };

    updateVisibleCount();

    let observer: ResizeObserver | null = null;
    if (typeof ResizeObserver !== 'undefined') {
      observer = new ResizeObserver(updateVisibleCount);
      observer.observe(container);
    } else {
      window.addEventListener('resize', updateVisibleCount);
    }

    return () => {
      observer?.disconnect();
      window.removeEventListener('resize', updateVisibleCount);
    };
  }, []);

  const displayedItems = items.slice(0, visibleCount);
  const gridColumns = `repeat(${Math.max(1, loading ? visibleCount : displayedItems.length)}, minmax(0, 1fr))`;

  if (error) return <ErrorBanner message={error} />;

  if (loading) {
    return (
      <div
        ref={containerRef}
        className="grid gap-3"
        style={{ gridTemplateColumns: gridColumns }}
      >
        {Array.from({ length: visibleCount }).map((_, i) => (
          <div
            key={i}
            className="min-w-0 overflow-hidden rounded-xl border border-border bg-card"
          >
            <Skeleton className="aspect-[16/10] w-full rounded-none" />
            <div className="space-y-2 p-3">
              <Skeleton className="h-3.5 w-3/4" />
              <Skeleton className="h-3 w-1/2" />
              <Skeleton className="h-3 w-full" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div ref={containerRef}>
        <p className="py-4 text-sm text-muted-foreground">{emptyMessage}</p>
      </div>
    );
  }

  return (
    <div
      ref={containerRef}
      className="grid gap-3"
      style={{ gridTemplateColumns: gridColumns }}
    >
      {displayedItems.map(({ type, item }) => (
        <div key={`${type}-${item.id}`} className="min-w-0">
          <ItemCard
            type={type}
            item={item}
            viewMode="compact"
            installedVersion={getInstalledVersion(item.id) ?? undefined}
            totalDownloads={getTotalDownloads(type, item.id)}
          />
        </div>
      ))}
    </div>
  );
}
