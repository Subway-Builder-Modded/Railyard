import { useMemo } from "react";
import { Link } from "wouter";
import { useRegistryStore } from "@/stores/registry-store";
import { useInstalledStore } from "@/stores/installed-store";
import { ItemCard } from "@/components/shared/ItemCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { CardSkeletonGrid } from "@/components/shared/CardSkeletonGrid";
import { ErrorBanner } from "@/components/shared/ErrorBanner";
import { Button } from "@/components/ui/button";
import { Download, Compass, ArrowRight } from "lucide-react";

export function HomePage() {
  const { mods, maps, loading, error } = useRegistryStore();
  const { installedMods, installedMaps } = useInstalledStore();

  const installedItems = useMemo(() => {
    const items: Array<{ type: "mods" | "maps"; item: typeof mods[number] | typeof maps[number]; installedVersion: string }> = [];
    for (const installed of installedMods) {
      const manifest = mods.find((m) => m.id === installed.id);
      if (manifest) items.push({ type: "mods", item: manifest, installedVersion: installed.version });
    }
    for (const installed of installedMaps) {
      const manifest = maps.find((m) => m.id === installed.id);
      if (manifest) items.push({ type: "maps", item: manifest, installedVersion: installed.version });
    }
    return items;
  }, [mods, maps, installedMods, installedMaps]);

  const installedIds = useMemo(() => {
    const ids = new Set<string>();
    for (const m of installedMods) ids.add(m.id);
    for (const m of installedMaps) ids.add(m.id);
    return ids;
  }, [installedMods, installedMaps]);

  const discoverItems = useMemo(() => {
    const items: Array<{ type: "mods" | "maps"; item: typeof mods[number] | typeof maps[number] }> = [];
    // Interleave mods and maps for variety, excluding already-installed items
    const maxLen = Math.max(mods.length, maps.length);
    for (let i = 0; i < maxLen && items.length < 8; i++) {
      if (i < mods.length && items.length < 8 && !installedIds.has(mods[i].id)) items.push({ type: "mods", item: mods[i] });
      if (i < maps.length && items.length < 8 && !installedIds.has(maps[i].id)) items.push({ type: "maps", item: maps[i] });
    }
    return items;
  }, [mods, maps, installedIds]);

  return (
    <div className="space-y-10">
      {/* Installed Section */}
      <section>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold tracking-tight">Installed</h2>
        </div>
        {installedItems.length === 0 ? (
          <EmptyState
            icon={Download}
            title="No mods or maps installed yet"
            description="Browse the registry to discover and install community content."
          >
            <Link href="/search">
              <Button>
                Browse Registry
                <ArrowRight className="h-4 w-4 ml-1.5" />
              </Button>
            </Link>
          </EmptyState>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {installedItems.map(({ type, item, installedVersion }) => (
              <ItemCard key={`${type}-${item.id}`} type={type} item={item} installedVersion={installedVersion} />
            ))}
          </div>
        )}
      </section>

      {/* Discover Section */}
      <section>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold tracking-tight">Discover</h2>
          <Link href="/search">
            <Button variant="ghost" size="sm">
              View all
              <ArrowRight className="h-4 w-4 ml-1" />
            </Button>
          </Link>
        </div>

        {error && <ErrorBanner message={error} />}

        {loading ? (
          <CardSkeletonGrid count={6} />
        ) : discoverItems.length === 0 ? (
          <EmptyState
            icon={Compass}
            title="Registry is empty"
            description="No mods or maps are available yet. Try refreshing."
          />
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {discoverItems.map(({ type, item }) => (
              <ItemCard key={`${type}-${item.id}`} type={type} item={item} />
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
