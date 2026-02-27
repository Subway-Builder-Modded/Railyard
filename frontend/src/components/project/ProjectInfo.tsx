import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { ExternalLink, MapPin, Users, Globe } from "lucide-react";
import { types } from "../../../wailsjs/go/models";

interface ProjectInfoProps {
  type: "mods" | "maps";
  item: types.ModManifest | types.MapManifest;
}

function isMapManifest(
  item: types.ModManifest | types.MapManifest
): item is types.MapManifest {
  return "city_code" in item;
}

export function ProjectInfo({ item }: ProjectInfoProps) {
  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{item.name}</h1>
        <p className="text-muted-foreground mt-1">by {item.author}</p>
      </div>

      {isMapManifest(item) && (
        <div className="flex items-center gap-4 text-sm">
          {item.city_code && (
            <div className="flex items-center gap-1.5">
              <MapPin className="h-4 w-4 text-muted-foreground" />
              <span className="font-mono font-bold">{item.city_code}</span>
              {item.country && (
                <span className="text-muted-foreground">{item.country}</span>
              )}
            </div>
          )}
          {item.population > 0 && (
            <div className="flex items-center gap-1.5">
              <Users className="h-4 w-4 text-muted-foreground" />
              <span>Pop. {item.population.toLocaleString()}</span>
            </div>
          )}
        </div>
      )}

      <Separator />

      <p className="text-sm leading-relaxed">{item.description}</p>

      {item.tags && item.tags.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {item.tags.map((tag) => (
            <Badge key={tag} variant="secondary">
              {tag}
            </Badge>
          ))}
        </div>
      )}

      {item.source && (
        <Button variant="outline" size="sm" asChild>
          <a href={item.source} target="_blank" rel="noopener noreferrer">
            <Globe className="h-4 w-4 mr-1.5" />
            View Source
            <ExternalLink className="h-3 w-3 ml-1.5" />
          </a>
        </Button>
      )}
    </div>
  );
}
