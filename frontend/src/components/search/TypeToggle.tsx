import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Package, MapPin } from "lucide-react";
import type { AssetType } from "@/lib/asset-types";

interface TypeToggleProps {
  value: AssetType;
  onChange: (value: AssetType) => void;
}

export function TypeToggle({ value, onChange }: TypeToggleProps) {
  return (
    <ToggleGroup
      type="single"
      value={value}
      onValueChange={(v) => {
        if (v === "mod" || v === "map") onChange(v);
      }}
    >
      <ToggleGroupItem value="map" aria-label="Maps">
        <MapPin className="h-4 w-4 mr-1.5" />
        Maps
      </ToggleGroupItem>
      <ToggleGroupItem value="mod" aria-label="Mods">
        <Package className="h-4 w-4 mr-1.5" />
        Mods
      </ToggleGroupItem>
    </ToggleGroup>
  );
}
