import { useEffect, useMemo } from "react";
import { Link } from "wouter";
import { useRegistryStore } from "@/stores/registry-store";
import { useInstalledStore } from "@/stores/installed-store";
import {
  useFilteredInstalledItems,
} from "@/hooks/use-filtered-installed-items";
import { SearchBar } from "@/components/search/SearchBar";
import { LibrarySidebar } from "@/components/library/LibrarySidebar";
import { LibraryTable } from "@/components/library/LibraryTable";
import { LibraryActionBar } from "@/components/library/LibraryActionBar";
import { EmptyState } from "@/components/shared/EmptyState";
import { Pagination } from "@/components/shared/Pagination";
import { ErrorBanner } from "@/components/shared/ErrorBanner";
import { Button } from "@/components/ui/button";
import {
  Inbox,
  Plus,
} from "lucide-react";

export function LibraryPage() {
  const {
    mods,
    maps,
    modDownloadTotals,
    mapDownloadTotals,
    ensureDownloadTotals,
  } = useRegistryStore();
  const { installedMods, installedMaps } = useInstalledStore();

  useEffect(() => {
    ensureDownloadTotals();
  }, [ensureDownloadTotals]);

  const modManifestById = useMemo(
    () => new Map(mods.map((manifest) => [manifest.id, manifest])),
    [mods],
  );
  const mapManifestById = useMemo(
    () => new Map(maps.map((manifest) => [manifest.id, manifest])),
    [maps],
  );

  const missingInstalledItems = useMemo(() => {
    const missingMods = installedMods
      .filter((installed) => !modManifestById.has(installed.id))
      .map((installed) => `mod:${installed.id}`);
    const missingMaps = installedMaps
      .filter((installed) => !mapManifestById.has(installed.id))
      .map((installed) => `map:${installed.id}`);

    return [...missingMods, ...missingMaps];
  }, [installedMaps, installedMods, mapManifestById, modManifestById]);

  const installedItems = useMemo(() => {
    const modItems = installedMods.flatMap((installed) => {
      const manifest = modManifestById.get(installed.id);
      return manifest
        ? [{ type: "mod" as const, item: manifest, installedVersion: installed.version }]
        : [];
    });
    const mapItems = installedMaps.flatMap((installed) => {
      const manifest = mapManifestById.get(installed.id);
      return manifest
        ? [{ type: "map" as const, item: manifest, installedVersion: installed.version }]
        : [];
    });

    return [...modItems, ...mapItems];
  }, [installedMods, installedMaps, modManifestById, mapManifestById]);

  const {
    items: paginatedItems,
    allFilteredItems,
    page,
    totalPages,
    totalResults,
    filters,
    setFilters,
    setPage,
  } = useFilteredInstalledItems({
    items: installedItems,
    modDownloadTotals,
    mapDownloadTotals,
  });

  const modCount = installedItems.filter((i) => i.type === "mod").length;
  const mapCount = installedItems.filter((i) => i.type === "map").length;

  const availableTags = useMemo(() => {
    const tags = new Set(mods.flatMap((item) => item.tags ?? []));
    return Array.from(tags).sort();
  }, [mods]);

  const availableSpecialDemand = useMemo(() => {
    const tags = new Set(maps.flatMap((item) => item.special_demand ?? []));
    return Array.from(tags).sort();
  }, [maps]);

  const totalProjects = installedItems.length;

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Library
          </h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            Manage your installed maps and mods.
          </p>
        </div>
      </div>

      {missingInstalledItems.length > 0 && (
        <ErrorBanner
          message={
            missingInstalledItems.length === 1
              ? `Installed content is missing from the registry: ${missingInstalledItems[0]}`
              : `${missingInstalledItems.length} installed items are missing from the registry.`
          }
        />
      )}

      <div className="flex items-center gap-3">
        <div className="flex-1">
          <SearchBar
            query={filters.query}
            onQueryChange={(value) =>
              setFilters((prev) => ({ ...prev, query: value }))
            }
          />
        </div>
        <Link href="/search">
          <Button className="gap-1.5 shrink-0">
            <Plus className="h-4 w-4" />
            Install Content
          </Button>
        </Link>
      </div>

      {installedItems.length === 0 ? (
        <EmptyState
          icon={Inbox}
          title="No content installed"
          description="Your library is empty. Browse the registry to discover and install community content."
        >
          <Link href="/search">
            <Button className="gap-1.5">
              <Plus className="h-4 w-4" />
              Install Content
            </Button>
          </Link>
        </EmptyState>
      ) : (
        <>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span>
                <span className="font-medium text-foreground">
                  {totalProjects}
                </span>{" "}
                Assets
              </span>
            </div>
          </div>

          <div className="flex gap-6 items-start">
            <aside className="w-52 shrink-0">
              <LibrarySidebar
                filters={filters}
                onFiltersChange={setFilters}
                modCount={modCount}
                mapCount={mapCount}
                availableTags={availableTags}
                availableSpecialDemand={availableSpecialDemand}
              />
            </aside>

            <div className="flex-1 min-w-0 space-y-4">
              {paginatedItems.length === 0 ? (
                <EmptyState
                  icon={Inbox}
                  title={filters.type === "map" ? "No maps found" : "No mods found"}
                  description={
                    filters.query
                      ? `No installed ${filters.type} match "${filters.query}"`
                      : `No installed ${filters.type} match the current filters`
                  }
                  className="py-16"
                />
              ) : (
                <>
                  <LibraryTable
                    items={paginatedItems}
                    activeType={filters.type}
                    sort={filters.sort}
                    onSortChange={(value) =>
                      setFilters((prev) => ({
                        ...prev,
                        sort: value,
                      }))
                    }
                  />
                  <Pagination
                    page={page}
                    totalPages={totalPages}
                    totalResults={totalResults}
                    perPage={filters.perPage}
                    onPageChange={setPage}
                    onPerPageChange={(value) =>
                      setFilters((prev) => ({
                        ...prev,
                        perPage: value,
                      }))
                    }
                  />
                </>
              )}

              <div className="sticky bottom-4">
                <LibraryActionBar allItems={allFilteredItems} />
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
