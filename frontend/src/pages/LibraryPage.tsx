import { useMemo, useState, useEffect, useCallback, useRef } from "react";
import { Link } from "wouter";
import { useRegistryStore } from "@/stores/registry-store";
import { useInstalledStore } from "@/stores/installed-store";
import {
  useFilteredInstalledItems,
  type InstalledTaggedItem,
} from "@/hooks/use-filtered-installed-items";
import { SearchBar } from "@/components/search/SearchBar";
import { LibrarySidebar } from "@/components/library/LibrarySidebar";
import { LibraryTable } from "@/components/library/LibraryTable";
import { LibraryActionBar } from "@/components/library/LibraryActionBar";
import { EmptyState } from "@/components/shared/EmptyState";
import { Pagination } from "@/components/shared/Pagination";
import { Button } from "@/components/ui/button";
import {
  Inbox,
  Plus,
  RefreshCw,
  Download,
} from "lucide-react";
import { GetVersions } from "../../wailsjs/go/registry/Registry";
import {
  GetActiveProfile,
  UpdateAllSubscriptionsToLatest,
} from "../../wailsjs/go/profiles/UserProfiles";
import { types } from "../../wailsjs/go/models";
import { toast } from "sonner";

export function LibraryPage() {
  const { mods, maps } = useRegistryStore();
  const { installedMods, installedMaps, updateInstalledLists } =
    useInstalledStore();

  const modManifestById = useMemo(
    () => new Map(mods.map((manifest) => [manifest.id, manifest])),
    [mods],
  );
  const mapManifestById = useMemo(
    () => new Map(maps.map((manifest) => [manifest.id, manifest])),
    [maps],
  );

  const installedItems = useMemo<InstalledTaggedItem[]>(() => {
    const items: InstalledTaggedItem[] = [];
    for (const installed of installedMods) {
      const manifest = modManifestById.get(installed.id);
      items.push({
        type: "mod",
        item: manifest ?? new types.ModManifest({
          id: installed.id,
          name: installed.id,
          author: "Unknown author",
          description: "Installed on disk but missing from the registry.",
          tags: [],
          gallery: [],
          source: "local",
          update: { type: "" },
        }),
        installedVersion: installed.version,
        inRegistry: Boolean(manifest),
      });
    }
    for (const installed of installedMaps) {
      const manifest = mapManifestById.get(installed.id);
      items.push({
        type: "map",
        item: manifest ?? new types.MapManifest({
          id: installed.id,
          name: installed.config?.name || installed.id,
          author: installed.config?.creator || "Unknown author",
          city_code: installed.config?.code || "",
          country: installed.config?.country || "",
          location: "",
          population: installed.config?.population || 0,
          description: installed.config?.description || "Installed on disk but missing from the registry.",
          data_source: "",
          source_quality: "",
          level_of_detail: "",
          special_demand: [],
          tags: [],
          gallery: [],
          source: "local",
          update: { type: "" },
        }),
        installedVersion: installed.version,
        inRegistry: Boolean(manifest),
      });
    }
    return items;
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
  } = useFilteredInstalledItems({ items: installedItems });

  const [updatesAvailable, setUpdatesAvailable] = useState<
    Map<string, types.VersionInfo>
  >(new Map());
  const [checkingUpdates, setCheckingUpdates] = useState(false);
  const [updatingAll, setUpdatingAll] = useState(false);

  const checkForUpdates = useCallback(async () => {
    if (installedItems.length === 0) return;
    setCheckingUpdates(true);
    const updates = new Map<string, types.VersionInfo>();

    try {
      for (const entry of installedItems) {
        try {
          if (!entry.inRegistry || !entry.item.update?.type) continue;
          const source =
            entry.item.update.type === "github"
              ? entry.item.update.repo
              : entry.item.update.url;
          if (!source) continue;

          const versions = await GetVersions(
            entry.item.update.type,
            source,
          );
          if (versions && versions.length > 0) {
            const latest = versions[0];
            if (latest.version !== entry.installedVersion) {
              updates.set(entry.item.id, latest);
            }
          }
        } catch {}
      }
    } finally {
      setUpdatesAvailable(updates);
      setCheckingUpdates(false);
    }
  }, [installedItems]);

  const didCheckUpdates = useRef(false);
  useEffect(() => {
    if (!didCheckUpdates.current) {
      didCheckUpdates.current = true;
      void checkForUpdates();
    }
  }, [checkForUpdates]);

  const updatesCount = updatesAvailable.size;

  const handleUpdateAll = async () => {
    setUpdatingAll(true);
    try {
      const activeResult = await GetActiveProfile();
      if (activeResult.status !== "success") {
        throw new Error(
          activeResult.message || "Failed to resolve active profile",
        );
      }
      const result = await UpdateAllSubscriptionsToLatest(
        activeResult.profile.id,
      );
      if (result.status === "error") {
        throw new Error(result.message || "Update all failed");
      }
      await updateInstalledLists();
      setUpdatesAvailable(new Map());
      toast.success("All content updated to latest versions.");
    } catch (err) {
      toast.error(
        `Update all failed: ${err instanceof Error ? err.message : String(err)}`,
      );
    } finally {
      setUpdatingAll(false);
    }
  };

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

  const [showingUpdatesOnly, setShowingUpdatesOnly] = useState(false);

  const displayedItems = showingUpdatesOnly
    ? paginatedItems.filter((item) => updatesAvailable.has(item.item.id))
    : paginatedItems;

  const totalProjects = installedItems.length;

  useEffect(() => {
    if (filters.type === "mod" && (filters.sort === "country-asc" || filters.sort === "country-desc")) {
      setFilters((prev) => ({ ...prev, sort: "name-asc" }));
    }
  }, [filters.type, filters.sort, setFilters]);

  const handleToggleNameSort = () => {
    setFilters((prev) => ({
      ...prev,
      sort: prev.sort === "name-asc" ? "name-desc" : "name-asc",
    }));
  };

  const handleToggleCountrySort = () => {
    setFilters((prev) => ({
      ...prev,
      sort: prev.sort === "country-asc" ? "country-desc" : "country-asc",
    }));
  };

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
                Projects
              </span>
              {updatesCount > 0 && (
                <span className="text-primary">
                  · {updatesCount} update
                  {updatesCount !== 1 ? "s" : ""} available
                </span>
              )}
            </div>

            <div className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={checkForUpdates}
                disabled={checkingUpdates}
                className="gap-1.5"
              >
                <RefreshCw
                  className={`h-3.5 w-3.5 ${checkingUpdates ? "animate-spin" : ""}`}
                />
                Refresh
              </Button>
              {updatesCount > 0 && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleUpdateAll}
                  disabled={updatingAll}
                  className="gap-1.5"
                >
                  <Download className="h-3.5 w-3.5" />
                  Update all
                </Button>
              )}
            </div>
          </div>

          <div className="flex gap-6 items-start">
            <aside className="w-52 shrink-0">
              <LibrarySidebar
                filters={filters}
                onFiltersChange={setFilters}
                modCount={modCount}
                mapCount={mapCount}
                updatesCount={updatesCount}
                onShowUpdatesOnly={() => setShowingUpdatesOnly((prev) => !prev)}
                showingUpdatesOnly={showingUpdatesOnly}
                availableTags={availableTags}
                availableSpecialDemand={availableSpecialDemand}
              />
            </aside>

            <div className="flex-1 min-w-0 space-y-4">
              {displayedItems.length === 0 ? (
                <EmptyState
                  icon={Inbox}
                  title={filters.type === "map" ? "No maps found" : "No mods found"}
                  description={
                    showingUpdatesOnly
                      ? "No updates available for the current filter"
                      : filters.query
                        ? `No installed ${filters.type} match "${filters.query}"`
                        : `No installed ${filters.type} match the current filters`
                  }
                  className="py-16"
                />
              ) : (
                <>
                  <LibraryTable
                    items={displayedItems}
                    updatesAvailable={updatesAvailable}
                    sort={filters.sort}
                    activeType={filters.type}
                    onToggleNameSort={handleToggleNameSort}
                    onToggleCountrySort={handleToggleCountrySort}
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
                <LibraryActionBar
                  allItems={allFilteredItems}
                  updatesAvailable={updatesAvailable}
                />
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
