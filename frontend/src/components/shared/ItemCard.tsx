import { CheckCircle, Download, MapPin, Package, Users } from 'lucide-react';
import { useLayoutEffect, useMemo, useRef, useState } from 'react';
import { Link } from 'wouter';

import { Badge } from '@/components/ui/badge';
import { type AssetType, assetTypeToListingPath } from '@/lib/asset-types';
import { getCountryFlagIcon } from '@/lib/flags';
import { formatSourceQuality } from '@/lib/map-filter-values';
import { MAX_CARD_BADGES } from '@/lib/search';
import type { SearchViewMode } from '@/lib/search-view-mode';
import { cn } from '@/lib/utils';

import type { types } from '../../../wailsjs/go/models';
import { GalleryImage } from './GalleryImage';

interface ItemCardProps {
  type: AssetType;
  item: types.ModManifest | types.MapManifest;
  installedVersion?: string;
  totalDownloads?: number;
  viewMode?: SearchViewMode;
}

interface ItemCardPresentation {
  isMap: boolean;
  badges: string[];
  mapCityCode: string;
  mapCountry: string;
  mapPopulation?: number;
  CountryFlag: ReturnType<typeof getCountryFlagIcon> | null;
  showDownloads: boolean;
}

function isMapManifest(
  item: types.ModManifest | types.MapManifest,
): item is types.MapManifest {
  return 'city_code' in item;
}

function buildItemCardPresentation(
  item: types.ModManifest | types.MapManifest,
  totalDownloads?: number,
): ItemCardPresentation {
  const isMap = isMapManifest(item);
  const mapBadges = isMap
    ? [
        item.location,
        formatSourceQuality(item.source_quality),
        item.level_of_detail,
        ...(item.special_demand ?? []),
      ].filter((value): value is string => Boolean(value))
    : [];

  const mapCityCode = isMap ? item.city_code!.trim() : '';
  const mapCountry = isMap ? item.country!.trim().toUpperCase() : '';

  return {
    isMap,
    badges: isMap ? mapBadges : (item.tags ?? []),
    mapCityCode,
    mapCountry,
    mapPopulation: isMap ? item.population : undefined,
    CountryFlag: isMap ? getCountryFlagIcon(mapCountry) : null,
    showDownloads: typeof totalDownloads === 'number',
  };
}

function MapLocationMeta({
  cityCode,
  country,
  CountryFlag,
}: {
  cityCode: string;
  country: string;
  CountryFlag: ReturnType<typeof getCountryFlagIcon> | null;
}) {
  if (!cityCode && !country) return null;

  return (
    <div className="shrink-0 text-right">
      {cityCode && (
        <span className="block text-xs font-mono font-bold text-foreground leading-none">
          {cityCode}
        </span>
      )}
      {country && (
        <span className="inline-flex items-center justify-end gap-1 text-xs text-muted-foreground">
          {CountryFlag && <CountryFlag className="h-3 w-4 rounded-[1px]" />}
          <span>{country.toUpperCase()}</span>
        </span>
      )}
    </div>
  );
}

function ItemStats({
  isMap,
  population,
  showDownloads,
  totalDownloads,
  className,
}: {
  isMap: boolean;
  population?: number;
  showDownloads: boolean;
  totalDownloads?: number;
  className?: string;
}) {
  if (!(isMap && (population ?? 0) > 0) && !showDownloads) return null;

  return (
    <div
      className={cn(
        'flex flex-col gap-1 text-xs text-muted-foreground shrink-0',
        className,
      )}
    >
      {isMap && (population ?? 0) > 0 && (
        <StatMetric icon={Users} value={population!} />
      )}
      {showDownloads && <StatMetric icon={Download} value={totalDownloads!} />}
    </div>
  );
}

function StatMetric({
  icon: Icon,
  value,
  className,
}: {
  icon: typeof Users | typeof Download;
  value: number;
  className?: string;
}) {
  return (
    <div
      className={cn(
        'flex items-center gap-1 text-xs text-muted-foreground',
        className,
      )}
    >
      <Icon className="h-3 w-3" aria-hidden="true" />
      <span>{value.toLocaleString()}</span>
    </div>
  );
}

function ItemBadges({
  badges,
  align = 'right',
  compact = false,
  wrap = false,
}: {
  badges: string[];
  align?: 'left' | 'right';
  compact?: boolean;
  wrap?: boolean;
}) {
  if (badges.length === 0) return null;

  const justifyClass = align === 'left' ? 'justify-start' : 'justify-end';
  const badgeClassName = compact
    ? 'text-[11px] px-1.5 py-0 h-5'
    : 'text-xs px-1.5 py-0';

  const badgesLimited = useMemo(
    () => badges.slice(0, MAX_CARD_BADGES),
    [badges],
  );
  const baseOverflowCount = Math.max(0, badges.length - badgesLimited.length);

  if (wrap) {
    return (
      <div className={cn('flex flex-wrap gap-1', justifyClass)}>
        {badgesLimited.map((badge) => (
          <Badge key={badge} variant="secondary" className={badgeClassName}>
            {badge}
          </Badge>
        ))}
        {baseOverflowCount > 0 && (
          <Badge variant="outline" className={badgeClassName}>
            +{baseOverflowCount}
          </Badge>
        )}
      </div>
    );
  }

  const containerRef = useRef<HTMLDivElement | null>(null);
  const measureRef = useRef<HTMLDivElement | null>(null);
  const [visibleCount, setVisibleCount] = useState(() => badgesLimited.length);

  useLayoutEffect(() => {
    const container = containerRef.current;
    const measure = measureRef.current;
    if (!container || !measure) return;

    const update = () => {
      const availableWidth = container.clientWidth;
      const gap = Number.parseFloat(getComputedStyle(container).columnGap) || 0;
      const badgeEls = Array.from(
        measure.querySelectorAll<HTMLElement>('[data-measure="badge"]'),
      );
      const badgeWidths = badgeEls.map(
        (el) => el.getBoundingClientRect().width,
      );

      const overflowEl = measure.querySelector<HTMLElement>(
        '[data-measure="overflow"]',
      );

      const measureOverflowWidth = (count: number) => {
        if (!overflowEl) return 0;
        overflowEl.textContent = `+${count}`;
        return overflowEl.getBoundingClientRect().width;
      };

      const totalFor = (count: number) => {
        const used = badgeWidths.slice(0, count).reduce((sum, w) => sum + w, 0);
        return used + Math.max(0, count - 1) * gap;
      };

      let nextVisible = badgeWidths.length;
      while (nextVisible > 0 && totalFor(nextVisible) > availableWidth) {
        nextVisible -= 1;
      }

      const totalBadges = badgeWidths.length;
      const totalOverflowCount =
        baseOverflowCount + (totalBadges - nextVisible);
      if (totalOverflowCount > 0) {
        const overflowWidth = measureOverflowWidth(totalOverflowCount);

        while (nextVisible >= 0) {
          const widthWithoutOverflow = totalFor(nextVisible);
          const hasSomeBadges = nextVisible > 0;
          const widthWithOverflow =
            widthWithoutOverflow + (hasSomeBadges ? gap : 0) + overflowWidth;

          if (widthWithOverflow <= availableWidth) break;
          nextVisible -= 1;
        }
      }

      setVisibleCount(Math.max(0, Math.min(totalBadges, nextVisible)));
    };

    update();

    const ro = new ResizeObserver(() => update());
    ro.observe(container);
    return () => ro.disconnect();
  }, [badgesLimited, baseOverflowCount]);

  const finalOverflowCount =
    baseOverflowCount + (badgesLimited.length - visibleCount);

  return (
    <>
      <div
        ref={containerRef}
        className={cn('flex flex-nowrap gap-1 overflow-hidden', justifyClass)}
      >
        {badgesLimited.slice(0, visibleCount).map((badge) => (
          <Badge key={badge} variant="secondary" className={badgeClassName}>
            {badge}
          </Badge>
        ))}
        {finalOverflowCount > 0 && (
          <Badge variant="outline" className={badgeClassName}>
            +{finalOverflowCount}
          </Badge>
        )}
      </div>

      <div
        ref={measureRef}
        aria-hidden="true"
        className="pointer-events-none absolute -left-[99999px] -top-[99999px] flex gap-1 opacity-0"
      >
        {badgesLimited.map((badge) => (
          <Badge
            key={badge}
            data-measure="badge"
            variant="secondary"
            className={badgeClassName}
          >
            {badge}
          </Badge>
        ))}
        <Badge
          data-measure="overflow"
          variant="outline"
          className={badgeClassName}
        >
          +{badges.length}
        </Badge>
      </div>
    </>
  );
}

export function ItemCard({
  type,
  item,
  installedVersion,
  totalDownloads,
  viewMode = 'full',
}: ItemCardProps) {
  const presentation = buildItemCardPresentation(item, totalDownloads);

  if (viewMode === 'list') {
    return (
      <Link
        href={`/project/${assetTypeToListingPath(type)}/${item.id}`}
        className="block w-full"
      >
        <article
          className={cn(
            'group relative bg-card border border-border rounded-lg overflow-hidden cursor-pointer transition-all duration-150 hover:border-foreground/20 hover:shadow-sm',
            installedVersion && 'ring-1 ring-primary/40',
          )}
        >
          <div className="flex flex-col sm:flex-row">
            <div className="relative h-44 sm:h-36 sm:w-48 md:w-52 overflow-hidden bg-muted shrink-0">
              {installedVersion && (
                <div className="absolute top-2 right-2 z-10">
                  <Badge variant="success" className="gap-1 text-xs shadow-sm">
                    <CheckCircle className="h-2.5 w-2.5" />
                    {installedVersion}
                  </Badge>
                </div>
              )}
              <div className="absolute top-2 left-2 z-10">
                <span className="inline-flex items-center gap-1 bg-background/80 backdrop-blur-sm border border-border/50 text-foreground text-xs font-medium px-2 py-0.5 rounded-full">
                  {presentation.isMap ? (
                    <MapPin className="h-2.5 w-2.5" />
                  ) : (
                    <Package className="h-2.5 w-2.5" />
                  )}
                  {presentation.isMap ? 'Map' : 'Mod'}
                </span>
              </div>
              <GalleryImage
                type={type}
                id={item.id}
                imagePath={item.gallery?.[0]}
                className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.02]"
              />
            </div>

            <div className="flex flex-col flex-1 p-3 gap-2 min-w-0">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <h3 className="font-semibold text-sm leading-snug text-foreground truncate">
                    {item.name}
                  </h3>
                  <p className="text-xs text-muted-foreground mt-0.5 truncate">
                    by {item.author}
                  </p>
                </div>
                {presentation.isMap && (
                  <MapLocationMeta
                    cityCode={presentation.mapCityCode}
                    country={presentation.mapCountry}
                    CountryFlag={presentation.CountryFlag}
                  />
                )}
              </div>

              <p className="text-xs text-muted-foreground leading-relaxed line-clamp-1">
                {item.description}
              </p>

              <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between mt-auto">
                <ItemStats
                  isMap={presentation.isMap}
                  population={presentation.mapPopulation}
                  showDownloads={presentation.showDownloads}
                  totalDownloads={totalDownloads}
                />
                <ItemBadges
                  badges={presentation.badges}
                  align="left"
                  wrap={false}
                />
              </div>
            </div>
          </div>
        </article>
      </Link>
    );
  }

  if (viewMode === 'compact') {
    const hasMapPopulation =
      presentation.isMap && (presentation.mapPopulation ?? 0) > 0;
    const hasDownloads = presentation.showDownloads;

    return (
      <Link
        href={`/project/${assetTypeToListingPath(type)}/${item.id}`}
        className="block w-full"
      >
        <article
          className={cn(
            'group relative bg-card border border-border rounded-lg overflow-hidden cursor-pointer transition-all duration-150 hover:border-foreground/20 hover:shadow-sm h-full flex flex-col',
            installedVersion && 'ring-1 ring-primary/40',
          )}
        >
          <div className="relative aspect-[16/10] overflow-hidden bg-muted shrink-0">
            {installedVersion && (
              <div className="absolute top-2 right-2 z-10">
                <Badge
                  variant="success"
                  className="gap-1 text-[11px] h-5 px-1.5 shadow-sm"
                >
                  <CheckCircle className="h-2.5 w-2.5" />
                  {installedVersion}
                </Badge>
              </div>
            )}
            <div className="absolute top-2 left-2 z-10">
              <span className="inline-flex items-center gap-1 bg-background/80 backdrop-blur-sm border border-border/50 text-foreground text-xs font-medium px-2 py-0.5 rounded-full">
                {presentation.isMap ? (
                  <MapPin className="h-2.5 w-2.5" />
                ) : (
                  <Package className="h-2.5 w-2.5" />
                )}
                {presentation.isMap ? 'Map' : 'Mod'}
              </span>
            </div>
            <GalleryImage
              type={type}
              id={item.id}
              imagePath={item.gallery?.[0]}
              className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.02]"
            />
          </div>

          <div className="flex flex-col flex-1 p-3 gap-2.5">
            <div className="flex items-start justify-between gap-2">
              <div className="min-w-0 flex-1">
                <h3 className="font-semibold text-sm leading-snug text-foreground truncate">
                  {item.name}
                </h3>
                <p className="text-[11px] text-muted-foreground mt-0.5 truncate">
                  by {item.author}
                </p>
              </div>
              {presentation.isMap && (
                <MapLocationMeta
                  cityCode={presentation.mapCityCode}
                  country={presentation.mapCountry}
                  CountryFlag={presentation.CountryFlag}
                />
              )}
            </div>

            <p className="text-[11px] text-muted-foreground leading-relaxed line-clamp-2 flex-1">
              {item.description}
            </p>

            {(hasDownloads || hasMapPopulation) && (
              <div className="flex items-end justify-between gap-2 mt-auto min-h-4">
                <div className="min-w-0">
                  {hasDownloads && (
                    <StatMetric icon={Download} value={totalDownloads ?? 0} />
                  )}
                </div>
                <div className="min-w-0 text-right">
                  {hasMapPopulation && (
                    <StatMetric
                      icon={Users}
                      value={presentation.mapPopulation ?? 0}
                      className="justify-end"
                    />
                  )}
                </div>
              </div>
            )}
          </div>
        </article>
      </Link>
    );
  }

  return (
    <Link
      href={`/project/${assetTypeToListingPath(type)}/${item.id}`}
      className="block w-full"
    >
      <article
        className={cn(
          'group relative bg-card border border-border rounded-lg overflow-hidden cursor-pointer transition-all duration-150 hover:border-foreground/20 hover:shadow-sm h-full flex flex-col',
          installedVersion && 'ring-1 ring-primary/40',
        )}
      >
        {/* Thumbnail */}
        <div className="relative aspect-video overflow-hidden bg-muted shrink-0">
          {installedVersion && (
            <div className="absolute top-2 right-2 z-10">
              <Badge variant="success" className="gap-1 text-xs shadow-sm">
                <CheckCircle className="h-2.5 w-2.5" />
                {installedVersion}
              </Badge>
            </div>
          )}
          {/* Type pill */}
          <div className="absolute top-2 left-2 z-10">
            <span className="inline-flex items-center gap-1 bg-background/80 backdrop-blur-sm border border-border/50 text-foreground text-xs font-medium px-2 py-0.5 rounded-full">
              {presentation.isMap ? (
                <MapPin className="h-2.5 w-2.5" />
              ) : (
                <Package className="h-2.5 w-2.5" />
              )}
              {presentation.isMap ? 'Map' : 'Mod'}
            </span>
          </div>
          <GalleryImage
            type={type}
            id={item.id}
            imagePath={item.gallery?.[0]}
            className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.02]"
          />
        </div>

        {/* Card body */}
        <div className="flex flex-col flex-1 p-4 gap-3">
          {/* Title row */}
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0 flex-1">
              <h3 className="font-semibold text-sm leading-snug text-foreground truncate">
                {item.name}
              </h3>
              <p className="text-xs text-muted-foreground mt-0.5 truncate">
                by {item.author}
              </p>
            </div>
            {presentation.isMap && (
              <MapLocationMeta
                cityCode={presentation.mapCityCode}
                country={presentation.mapCountry}
                CountryFlag={presentation.CountryFlag}
              />
            )}
          </div>

          {/* Description */}
          <p className="text-xs text-muted-foreground leading-relaxed line-clamp-2 flex-1">
            {item.description}
          </p>

          {/* Footer: population + tags */}
          <div className="flex items-end justify-between gap-2 mt-auto">
            <ItemStats
              isMap={presentation.isMap}
              population={presentation.mapPopulation}
              showDownloads={presentation.showDownloads}
              totalDownloads={totalDownloads}
            />
            <ItemBadges badges={presentation.badges} />
          </div>
        </div>
      </article>
    </Link>
  );
}
