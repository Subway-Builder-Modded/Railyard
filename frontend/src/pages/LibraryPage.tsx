import { useMemo, useState, useEffect, useCallback } from "react";
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
  ChevronDown,
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

  // Merge installed items with their registry manifests
  const installedItems = useMemo<InstalledTaggedItem[]>(() => {
    const items: InstalledTaggedItem[] = [];
    for (const installed of installedMods) {
      const manifest = mods.find((m) => m.id === installed.id);
      if (manifest) {
        items.push({
          type: "mods",
          item: manifest,
          installedVersion: installed.version,
        });
      }
    }
    for (const installed of installedMaps) {
      const manifest = maps.find((m) => m.id === installed.id);
      if (manifest) {
        items.push({
          type: "maps",
          item: manifest,
          installedVersion: installed.version,
        });
      }
    }
    return items;
  }, [mods, maps, installedMods, installedMaps]);

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

  // Track available updates
  const [updatesAvailable, setUpdatesAvailable] = useState<
    Map<string, types.VersionInfo>
  >(new Map());
  const [checkingUpdates, setCheckingUpdates] = useState(false);
  const [updatingAll, setUpdatingAll] = useState(false);

  // Check for updates
  const checkForUpdates = useCallback(async () => {
    if (installedItems.length === 0) return;
    setCheckingUpdates(true);
    const updates = new Map<string, types.VersionInfo>();

    try {
      for (const entry of installedItems) {
        try {
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
        } catch {
          // Skip items that fail to fetch versions
        }
      }
    } finally {
      setUpdatesAvailable(updates);
      setCheckingUpdates(false);
    }
  }, [installedItems]);

  // Check for updates on mount
  useEffect(() => {
    checkForUpdates();
  }, []);  // eslint-disable-line react-hooks/exhaustive-deps

  const updatesCount = updatesAvailable.size;

  // Handle "Update All"
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

  const modCount = installedItems.filter((i) => i.type === "mods").length;
  const mapCount = installedItems.filter((i) => i.type === "maps").length;

  // "Updates available" filter
  const [showingUpdatesOnly, setShowingUpdatesOnly] = useState(false);

  // When showing updates only, further filter the paginated items
  const displayedItems = showingUpdatesOnly
    ? paginatedItems.filter((item) => updatesAvailable.has(item.item.id))
    : paginatedItems;

  const displayedTotalResults = showingUpdatesOnly
    ? allFilteredItems.filter((item) => updatesAvailable.has(item.item.id)).length
    : totalResults;

  return (
    <div className="space-y-5">
      {/* Page header */}
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

      {/* Search bar + Install Content button */}
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
            <ChevronDown className="h-3.5 w-3.5 ml-0.5 opacity-60" />
          </Button>
        </Link>
      </div>

      {/* Empty state */}
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
          {/* Tabs & actions row */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span>
                <span className="font-medium text-foreground">
                  {displayedTotalResults}
                </span>{" "}
                project{displayedTotalResults !== 1 ? "s" : ""}
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

          {/* Two-column layout: sidebar + table */}
          <div className="flex gap-6 items-start">
            {/* Sidebar */}
            <aside className="w-52 shrink-0">
              <LibrarySidebar
                filters={filters}
                onFiltersChange={setFilters}
                modCount={modCount}
                mapCount={mapCount}
                totalCount={installedItems.length}
                updatesCount={updatesCount}
                onShowUpdatesOnly={() => setShowingUpdatesOnly((prev) => !prev)}
                showingUpdatesOnly={showingUpdatesOnly}
              />
            </aside>

            {/* Main content area */}
            <div className="flex-1 min-w-0 space-y-4">
              {/* Table */}
              {displayedItems.length === 0 ? (
                <EmptyState
                  icon={Inbox}
                  title="No results found"
                  description={
                    showingUpdatesOnly
                      ? "No updates available for the current filter"
                      : filters.query
                        ? `No installed items match "${filters.query}"`
                        : "No items match the current filters"
                  }
                />
              ) : (
                <>
                  <LibraryTable
                    items={displayedItems}
                    updatesAvailable={updatesAvailable}
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

              {/* Action bar (fixed at bottom when items selected) */}
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
