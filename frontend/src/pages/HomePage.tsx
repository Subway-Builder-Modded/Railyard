import { useMemo } from "react";
import { Link } from "wouter";
import { useRegistryStore } from "@/stores/registry-store";
import { useInstalledStore } from "@/stores/installed-store";
import { ItemCard } from "@/components/shared/ItemCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { CardSkeletonGrid } from "@/components/shared/CardSkeletonGrid";
import { ErrorBanner } from "@/components/shared/ErrorBanner";
import { Button } from "@/components/ui/button";
import { Compass, ArrowRight, Library, Search } from "lucide-react";
import type { AssetType } from "@/lib/asset-types";

export function HomePage() {
  const { mods, maps, loading, error } = useRegistryStore();
  const { installedMods, installedMaps } = useInstalledStore();

  const installedIds = useMemo(() => {
    const ids = new Set<string>();
    for (const m of installedMods) ids.add(m.id);
    for (const m of installedMaps) ids.add(m.id);
    return ids;
  }, [installedMods, installedMaps]);

  const installedCount = installedMods.length + installedMaps.length;

  const discoverItems = useMemo(() => {
    const items: Array<{ type: AssetType; item: typeof mods[number] | typeof maps[number] }> = [];
    // Interleave mods and maps for variety, excluding already-installed items
    const maxLen = Math.max(mods.length, maps.length);
    for (let i = 0; i < maxLen && items.length < 8; i++) {
      if (i < mods.length && items.length < 8 && !installedIds.has(mods[i].id)) items.push({ type: "mod", item: mods[i] });
      if (i < maps.length && items.length < 8 && !installedIds.has(maps[i].id)) items.push({ type: "map", item: maps[i] });
    }
    return items;
  }, [mods, maps, installedIds]);

  return (
    <div className="space-y-10">
      <section>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold tracking-tight">Jump Back In</h2>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Link href="/library">
            <div className="group relative bg-card border border-border rounded-lg p-6 cursor-pointer transition-all duration-150 hover:border-foreground/20 hover:shadow-sm flex items-center gap-4">
              <div className="flex items-center justify-center h-12 w-12 rounded-lg bg-primary/10 text-primary shrink-0">
                <Library className="h-6 w-6" />
              </div>
              <div className="min-w-0 flex-1">
                <h3 className="font-semibold text-sm text-foreground">
                  My Library
                </h3>
                <p className="text-xs text-muted-foreground mt-0.5">
                  {installedCount > 0
                    ? `${installedCount} item${installedCount !== 1 ? "s" : ""} installed`
                    : "No content installed yet"}
                </p>
              </div>
              <ArrowRight className="h-4 w-4 text-muted-foreground group-hover:text-foreground transition-colors shrink-0" />
            </div>
          </Link>

          <Link href="/search">
            <div className="group relative bg-card border border-border rounded-lg p-6 cursor-pointer transition-all duration-150 hover:border-foreground/20 hover:shadow-sm flex items-center gap-4">
              <div className="flex items-center justify-center h-12 w-12 rounded-lg bg-primary/10 text-primary shrink-0">
                <Search className="h-6 w-6" />
              </div>
              <div className="min-w-0 flex-1">
                <h3 className="font-semibold text-sm text-foreground">
                  Browse
                </h3>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Discover and install maps and mods
                </p>
              </div>
              <ArrowRight className="h-4 w-4 text-muted-foreground group-hover:text-foreground transition-colors shrink-0" />
            </div>
          </Link>
        </div>
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
