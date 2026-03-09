import { useMemo } from "react";
import { useRegistryStore } from "@/stores/registry-store";
import { useFilteredItems } from "@/hooks/use-filtered-items";
import { SearchBar } from "@/components/search/SearchBar";
import { SidebarFilters } from "@/components/search/SidebarFilters";
import { ItemCard } from "@/components/shared/ItemCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { CardSkeletonGrid } from "@/components/shared/CardSkeletonGrid";
import { ErrorBanner } from "@/components/shared/ErrorBanner";
import { Pagination } from "@/components/shared/Pagination";
import { SortSelect } from "@/components/search/SortSelect";
import { SearchX } from "lucide-react";

export function SearchPage() {
  const { mods, maps, loading, error } = useRegistryStore();

  const allTags = useMemo(() => {
    const modTags = mods.flatMap((m) => m.tags ?? []);
    const mapTags = maps.flatMap((m) => m.tags ?? []);
    return [...new Set([...modTags, ...mapTags])].sort();
  }, [mods, maps]);

  const {
    items,
    page,
    totalPages,
    totalResults,
    query,
    type,
    selectedTags,
    sort,
    perPage,
    setQuery,
    setType,
    setSelectedTags,
    setSort,
    setPage,
    setPerPage,
  } = useFilteredItems({ mods, maps });

  if (error) return <ErrorBanner message={error} />;

  const modCount = mods.length;
  const mapCount = maps.length;

  return (
    <div className="space-y-5">
      {/* Page header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground text-balance">Browse</h1>
        <p className="text-sm text-muted-foreground mt-0.5">
          Discover and install maps and mods for Subway Builder.
        </p>
      </div>

      {/* Search bar — full width at top */}
      <SearchBar query={query} onQueryChange={setQuery} />

      {/* Two-column layout: sidebar + results */}
      <div className="flex gap-6 items-start">
        {/* Sidebar */}
        <aside className="w-52 shrink-0">
          <SidebarFilters
            type={type}
            onTypeChange={setType}
            availableTags={allTags}
            selectedTags={selectedTags}
            onTagsChange={setSelectedTags}
            modCount={modCount}
            mapCount={mapCount}
          />
        </aside>

        {/* Main results area */}
        <div className="flex-1 min-w-0 space-y-4">
          {/* Results toolbar */}
          <div className="flex items-center justify-between gap-3">
            <p className="text-sm text-muted-foreground">
              {loading ? (
                <span className="inline-block h-4 w-24 bg-muted rounded animate-pulse" />
              ) : (
                <>
                  <span className="font-medium text-foreground">{totalResults}</span>{" "}
                  result{totalResults !== 1 ? "s" : ""}
                  {query && (
                    <span className="ml-1">
                      for <span className="italic">"{query}"</span>
                    </span>
                  )}
                </>
              )}
            </p>
            <SortSelect value={sort} onChange={setSort} />
          </div>

          {/* Cards / empty / loading */}
          {loading ? (
            <CardSkeletonGrid count={perPage} />
          ) : items.length === 0 ? (
            <EmptyState
              icon={SearchX}
              title="No results found"
              description={
                query
                  ? `No items match "${query}"`
                  : "No items match the current filters"
              }
            />
          ) : (
            <>
              <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                {items.map(({ type: itemType, item }) => (
                  <ItemCard
                    key={`${itemType}-${item.id}`}
                    type={itemType}
                    item={item}
                  />
                ))}
              </div>
              <Pagination
                page={page}
                totalPages={totalPages}
                totalResults={totalResults}
                perPage={perPage}
                onPageChange={setPage}
                onPerPageChange={setPerPage}
              />
            </>
          )}
        </div>
      </div>
    </div>
  );
}
