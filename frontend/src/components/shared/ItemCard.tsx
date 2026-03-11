import { Link } from "wouter";
import { Badge } from "@/components/ui/badge";
import { GalleryImage } from "./GalleryImage";
import { cn } from "@/lib/utils";
import { Users, CheckCircle, Package, MapPin } from "lucide-react";
import { types } from "../../../wailsjs/go/models";

interface ItemCardProps {
  type: "mods" | "maps";
  item: types.ModManifest | types.MapManifest;
  installedVersion?: string;
}

function isMapManifest(item: types.ModManifest | types.MapManifest): item is types.MapManifest {
  return "city_code" in item;
}

export function ItemCard({ type, item, installedVersion }: ItemCardProps) {
  const isMap = isMapManifest(item);
  const mapBadges = isMap
    ? [item.location, item.source_quality, item.level_of_detail, ...(item.special_demand ?? [])].filter(
        (value): value is string => Boolean(value)
      )
    : [];
  const badges = isMap ? mapBadges : item.tags ?? [];

  return (
    <Link href={`/project/${type}/${item.id}`}>
      <article
        className={cn(
          "group relative bg-card border border-border rounded-lg overflow-hidden cursor-pointer transition-all duration-150 hover:border-foreground/20 hover:shadow-sm h-full flex flex-col",
          installedVersion && "ring-1 ring-primary/40"
        )}
      >
        {/* Thumbnail */}
        <div className="relative aspect-video overflow-hidden bg-muted shrink-0">
          {installedVersion && (
            <div className="absolute top-2 right-2 z-10">
              <Badge className="gap-1 text-xs shadow-sm">
                <CheckCircle className="h-2.5 w-2.5" />
                {installedVersion}
              </Badge>
            </div>
          )}
          {/* Type pill */}
          <div className="absolute top-2 left-2 z-10">
            <span className="inline-flex items-center gap-1 bg-background/80 backdrop-blur-sm border border-border/50 text-foreground text-xs font-medium px-2 py-0.5 rounded-full">
              {isMap ? (
                <MapPin className="h-2.5 w-2.5" />
              ) : (
                <Package className="h-2.5 w-2.5" />
              )}
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
            {isMap && item.city_code && (
              <div className="shrink-0 text-right">
                <span className="block text-xs font-mono font-bold text-foreground leading-none">
                  {item.city_code}
                </span>
                {item.country && (
                  <span className="text-xs text-muted-foreground">{item.country}</span>
                )}
              </div>
            )}
          </div>

          {/* Description */}
          <p className="text-xs text-muted-foreground leading-relaxed line-clamp-2 flex-1">
            {item.description}
          </p>

          {/* Footer: population + tags */}
          <div className="flex items-end justify-between gap-2 mt-auto">
            {isMap && item.population > 0 ? (
              <div className="flex items-center gap-1 text-xs text-muted-foreground shrink-0">
                <Users className="h-3 w-3" aria-hidden="true" />
                <span>Pop. {item.population.toLocaleString()}</span>
              </div>
            ) : (
              <span />
            )}

            {badges.length > 0 && (
              <div className="flex flex-wrap justify-end gap-1">
                {badges.slice(0, 3).map((badge) => (
                  <Badge key={badge} variant="secondary" className="text-xs px-1.5 py-0">
                    {badge}
                  </Badge>
                ))}
                {badges.length > 3 && (
                  <Badge variant="outline" className="text-xs px-1.5 py-0">
                    +{badges.length - 3}
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
