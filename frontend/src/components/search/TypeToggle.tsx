import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Package, MapPin } from "lucide-react";

interface TypeToggleProps {
  value: "mods" | "maps";
  onChange: (value: "mods" | "maps") => void;
}

export function TypeToggle({ value, onChange }: TypeToggleProps) {
  return (
    <ToggleGroup
      type="single"
      value={value}
      onValueChange={(v) => {
        if (v) onChange(v as "mods" | "maps");
      }}
    >
      <ToggleGroupItem value="maps" aria-label="Maps">
        <MapPin className="h-4 w-4 mr-1.5" />
        Maps
      </ToggleGroupItem>
      <ToggleGroupItem value="mods" aria-label="Mods">
        <Package className="h-4 w-4 mr-1.5" />
        Mods
      </ToggleGroupItem>
    </ToggleGroup>
  );
}
