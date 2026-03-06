import { Link } from "wouter";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { GalleryImage } from "./GalleryImage";
import { cn } from "@/lib/utils";
import { Users, CheckCircle } from "lucide-react";
import { types } from "../../../wailsjs/go/models";

interface ItemCardProps {
  type: "mods" | "maps";
  item: types.ModManifest | types.MapManifest;
  installedVersion?: string;
}

function isMapManifest(item: types.ModManifest | types.MapManifest): item is types.MapManifest {
  return 'city_code' in item;
}

export function ItemCard({ type, item, installedVersion }: ItemCardProps) {
  return (
    <Link href={`/project/${type}/${item.id}`}>
      <Card className={cn("overflow-hidden cursor-pointer transition-colors hover:bg-accent/50 h-full flex flex-col", installedVersion && "border-l-2 border-l-primary")}>
        <div className="relative aspect-video overflow-hidden">
          {installedVersion && (
            <Badge className="absolute top-2 right-2 z-10 gap-1">
              <CheckCircle className="h-3 w-3" />
              {installedVersion}
            </Badge>
          )}
          <GalleryImage
            type={type}
            id={item.id}
            imagePath={item.gallery?.[0]}
            className="h-full"
          />
        </div>
        <CardHeader className="pb-2">
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0">
              <CardTitle className="text-base truncate">{item.name}</CardTitle>
              <CardDescription className="text-sm">by {item.author}</CardDescription>
            </div>
            {isMapManifest(item) && item.city_code && (
              <div className="flex-shrink-0 text-right">
                <span className="text-xs font-mono font-bold text-foreground">{item.city_code}</span>
                <span className="text-xs text-muted-foreground ml-1">{item.country}</span>
              </div>
            )}
          </div>
        </CardHeader>
        <CardContent className="flex-1 flex flex-col justify-between gap-3">
          <div>
            <p className="text-sm text-muted-foreground line-clamp-2">{item.description}</p>
            {isMapManifest(item) && item.population > 0 && (
              <div className="flex items-center gap-1 mt-1.5 text-xs text-muted-foreground">
                <Users className="h-3 w-3" />
                <span>Pop. {item.population.toLocaleString()}</span>
              </div>
            )}
          </div>
          {item.tags && item.tags.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {item.tags.slice(0, 4).map((tag) => (
                <Badge key={tag} variant="secondary" className="text-xs">
                  {tag}
                </Badge>
              ))}
              {item.tags.length > 4 && (
                <Badge variant="outline" className="text-xs">+{item.tags.length - 4}</Badge>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </Link>
  );
}
