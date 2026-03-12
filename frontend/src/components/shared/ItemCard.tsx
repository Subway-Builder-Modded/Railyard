import { Link } from "wouter";
import { Badge } from "@/components/ui/badge";
import { GalleryImage } from "./GalleryImage";
import { cn } from "@/lib/utils";
import { Users, CheckCircle, Package, MapPin, Download } from "lucide-react";
import { formatSourceQuality } from "@/lib/map-filter-values";
import { MAX_CARD_BADGES } from "@/lib/search";
import { getCountryFlagIcon } from "@/lib/flags";
import { assetTypeToListingPath, type AssetType } from "@/lib/asset-types";
import { types } from "../../../wailsjs/go/models";

interface ItemCardProps {
  type: AssetType;
  item: types.ModManifest | types.MapManifest;
  installedVersion?: string;
  totalDownloads?: number;
}

function isMapManifest(
  item: types.ModManifest | types.MapManifest,
): item is types.MapManifest {
  return "city_code" in item;
}

export function ItemCard({
  type,
  item,
  installedVersion,
  totalDownloads,
}: ItemCardProps) {
  const isMap = isMapManifest(item);
  const mapBadges = isMap
    ? [
        item.location,
        formatSourceQuality(item.source_quality),
        item.level_of_detail,
        ...(item.special_demand ?? []),
      ].filter((value): value is string => Boolean(value))
    : [];
  const badges = isMap ? mapBadges : (item.tags ?? []);

  const mapCityCode = isMap ? item.city_code!.trim() : "";
  const mapCountry = isMap ? item.country!.trim().toUpperCase() : "";
  const CountryFlag = isMap ? getCountryFlagIcon(mapCountry) : null;
  const showDownloads = typeof totalDownloads === "number";

  return (
    <Link href={`/project/${assetTypeToListingPath(type)}/${item.id}`}>
      <article
        className={cn(
          "group relative bg-card border border-border rounded-lg overflow-hidden cursor-pointer transition-all duration-150 hover:border-foreground/20 hover:shadow-sm h-full flex flex-col",
          installedVersion && "ring-1 ring-primary/40",
        )}
      >
        <div className="relative aspect-video overflow-hidden bg-muted shrink-0">
          {installedVersion && (
            <div className="absolute top-2 right-2 z-10">
              <Badge className="gap-1 text-xs shadow-sm">
                <CheckCircle className="h-2.5 w-2.5" />
                {installedVersion}
              </Badge>
            </div>
          )}
          <div className="absolute top-2 left-2 z-10">
            <span className="inline-flex items-center gap-1 bg-background/80 backdrop-blur-sm border border-border/50 text-foreground text-xs font-medium px-2 py-0.5 rounded-full">
              {isMap ? <MapPin className="h-2.5 w-2.5" /> : <Package className="h-2.5 w-2.5" />}
              {isMap ? "Map" : "Mod"}
            </span>
          </div>
          <GalleryImage
            type={type}
            id={item.id}
            imagePath={item.gallery?.[0]}
            className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.02]"
          />
        </div>

        <div className="flex flex-col flex-1 p-4 gap-3">
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0 flex-1">
              <h3 className="font-semibold text-sm leading-snug text-foreground truncate">
                {item.name}
              </h3>
              <p className="text-xs text-muted-foreground mt-0.5 truncate">
                by {item.author}
              </p>
            </div>
            {isMap && (
              <div className="shrink-0 text-right">
                {mapCityCode && (
                  <span className="block text-xs font-mono font-bold text-foreground leading-none">
                    {mapCityCode}
                  </span>
                )}
                {mapCountry && (
                  <span className="inline-flex items-center justify-end gap-1 text-xs text-muted-foreground">
                    {CountryFlag && <CountryFlag className="h-3 w-4 rounded-[1px]" />}
                    <span>{mapCountry.toUpperCase()}</span>
                  </span>
                )}
              </div>
            )}
          </div>

          <p className="text-xs text-muted-foreground leading-relaxed line-clamp-2 flex-1">
            {item.description}
          </p>

          <div className="flex items-end justify-between gap-2 mt-auto">
            {(isMap && item.population > 0) || showDownloads ? (
              <div className="flex flex-col gap-1 text-xs text-muted-foreground shrink-0">
                {isMap && item.population > 0 && (
                  <div className="flex items-center gap-1">
                    <Users className="h-3 w-3" aria-hidden="true" />
                    <span>{item.population.toLocaleString()}</span>
                  </div>
                )}
                {showDownloads && (
                  <div className="flex items-center gap-1">
                    <Download className="h-3 w-3" aria-hidden="true" />
                    <span>{totalDownloads.toLocaleString()}</span>
                  </div>
                )}
              </div>
            ) : (
              <span />
            )}

            {badges.length > 0 && (
              <div className="flex flex-wrap justify-end gap-1">
                {badges.slice(0, MAX_CARD_BADGES).map((badge) => (
                  <Badge
                    key={badge}
                    variant="secondary"
                    className="text-xs px-1.5 py-0"
                  >
                    {badge}
                  </Badge>
                ))}
                {badges.length > MAX_CARD_BADGES && (
                  <Badge variant="outline" className="text-xs px-1.5 py-0">
                    +{badges.length - MAX_CARD_BADGES}
                  </Badge>
                )}
              </div>
            )}
          </div>
        </div>
      </article>
    </Link>
  );
}
