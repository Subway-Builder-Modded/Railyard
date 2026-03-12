import { useState } from "react";
import { Link } from "wouter";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Checkbox } from "@/components/ui/checkbox";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { ChevronUp, Download, FolderOpen, Trash2 } from "lucide-react";
import { UninstallDialog } from "@/components/dialogs/UninstallDialog";
import { useLibraryStore } from "@/stores/library-store";
import { useConfigStore } from "@/stores/config-store";
import { type InstalledTaggedItem } from "@/hooks/use-filtered-installed-items";
import { cn } from "@/lib/utils";
import { types } from "../../../wailsjs/go/models";
import { MAX_CARD_BADGES } from "@/lib/search";
import { formatSourceQuality } from "@/lib/map-filter-values";
import { getCountryFlagIcon } from "@/lib/flags";
import { type LibrarySortOption } from "@/stores/library-store";
import { toast } from "sonner";
import type { AssetType } from "@/lib/asset-types";
import { assetTypeToListingPath } from "@/lib/asset-types";

interface LibraryTableProps {
  items: InstalledTaggedItem[];
  updatesAvailable: Map<string, types.VersionInfo>;
  sort: LibrarySortOption;
  activeType: AssetType;
  onToggleNameSort: () => void;
  onToggleCountrySort: () => void;
}

function composeItemKey(item: InstalledTaggedItem): string {
  return `${item.type}-${item.item.id}`;
}

export function LibraryTable({
  items,
  updatesAvailable,
  sort,
  activeType,
  onToggleNameSort,
  onToggleCountrySort,
}: LibraryTableProps) {
  const { selectedIds, toggleSelected, selectAll, clearSelection } =
    useLibraryStore();
  const showCountryColumn = activeType === "map";
  const isNameDesc = sort === "name-desc";
  const isNameSort = sort === "name-asc" || sort === "name-desc";
  const isCountryDesc = sort === "country-desc";
  const isCountrySort = sort === "country-asc" || sort === "country-desc";

  const allKeys = items.map(composeItemKey);
  const allSelected =
    items.length > 0 && allKeys.every((k) => selectedIds.has(k));
  const someSelected =
    !allSelected && allKeys.some((k) => selectedIds.has(k));

  const handleSelectAll = () => {
    if (allSelected) {
      clearSelection();
    } else {
      selectAll(allKeys);
    }
  };

  return (
    <div className="rounded-md border border-border">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead className="w-10">
              <Checkbox
                checked={allSelected ? true : someSelected ? "indeterminate" : false}
                onCheckedChange={handleSelectAll}
                aria-label="Select all"
              />
            </TableHead>
            <TableHead>
              <button
                className="flex items-center gap-1 text-foreground font-medium"
                type="button"
                onClick={onToggleNameSort}
              >
                Name
                <ChevronUp
                  className={cn(
                    "h-3.5 w-3.5 text-muted-foreground transition-transform duration-200",
                    !isNameSort && "opacity-30",
                    isNameDesc && "rotate-180",
                  )}
                />
              </button>
            </TableHead>
            {showCountryColumn && (
              <TableHead className="w-32 text-center">
                <button
                  className="mx-auto flex items-center gap-1 text-foreground font-medium"
                  type="button"
                  onClick={onToggleCountrySort}
                >
                  Country
                  <ChevronUp
                    className={cn(
                      "h-3.5 w-3.5 text-muted-foreground transition-transform duration-200",
                      !isCountrySort && "opacity-30",
                      isCountryDesc && "rotate-180",
                    )}
                  />
                </button>
              </TableHead>
            )}
            <TableHead className="w-28 text-center">Version</TableHead>
            <TableHead className="w-24"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((entry) => {
            const key = composeItemKey(entry);
            const isSelected = selectedIds.has(key);
            const hasUpdate = updatesAvailable.has(entry.item.id);

            return (
              <LibraryTableRow
                key={key}
                entry={entry}
                isSelected={isSelected}
                hasUpdate={hasUpdate}
                showCountryColumn={showCountryColumn}
                onToggleSelect={() => toggleSelected(key)}
              />
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}

interface LibraryTableRowProps {
  entry: InstalledTaggedItem;
  isSelected: boolean;
  hasUpdate: boolean;
  showCountryColumn: boolean;
  onToggleSelect: () => void;
}

function LibraryTableRow({
  entry,
  isSelected,
  hasUpdate,
  showCountryColumn,
  onToggleSelect,
}: LibraryTableRowProps) {
  const [uninstallOpen, setUninstallOpen] = useState(false);
  const metroMakerDataPath = useConfigStore((s) => s.config?.metroMakerDataPath);

  const isMap = entry.type === "map";
  const map = isMap ? (entry.item as types.MapManifest) : null;
  const mapBadges = map
    ? [
        map.location,
        formatSourceQuality(map.source_quality),
        map.level_of_detail,
        ...(map.special_demand ?? []),
      ].filter((value): value is string => Boolean(value))
    : [];
  const badges = isMap ? mapBadges : (entry.item.tags ?? []);
  const mapCountry = map?.country?.trim().toUpperCase() ?? "";
  const CountryFlag = isMap ? getCountryFlagIcon(mapCountry) : null;

  const resolveInstallFolderPath = (): string | null => {
    if (!metroMakerDataPath) return null;

    if (entry.type === "mod") {
      return `${metroMakerDataPath}\\mods\\${entry.item.id}`;
    }

    const cityCode = (map?.city_code ?? "").trim();
    if (!cityCode) {
      return `${metroMakerDataPath}\\cities\\data`;
    }

    return `${metroMakerDataPath}\\cities\\data\\${cityCode}`;
  };

  const handleOpenInstallFolder = () => {
    const folderPath = resolveInstallFolderPath();
    if (!folderPath) return;

    void (async () => {
      try {
        const go = (window as { go?: { main?: { App?: { OpenInFileExplorer?: (path: string) => Promise<{ status?: string; message?: string }> } } } }).go?.main?.App;
        if (!go?.OpenInFileExplorer) {
          toast.error("File explorer integration not available");
          return;
        }
        const result = await go.OpenInFileExplorer(folderPath);
        if (result?.status === "error") {
          toast.error(result?.message || "Failed to open install folder");
        }
      } catch (err) {
        toast.error(err instanceof Error ? err.message : String(err));
      }
    })();
  };


  return (
    <>
      <TableRow
        data-state={isSelected ? "selected" : undefined}
        className={cn("group transition-colors", isSelected && "bg-muted/50")}
      >
        {/* Checkbox */}
        <TableCell>
          <Checkbox
            checked={isSelected}
            onCheckedChange={onToggleSelect}
            aria-label={`Select ${entry.item.name}`}
          />
        </TableCell>

        {/* Name + author + badges */}
        <TableCell>
          <div className="min-w-0">
            <div className="flex items-center justify-between gap-4">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2 flex-wrap">
                  {entry.inRegistry ? (
                    <Link
                      href={`/project/${assetTypeToListingPath(entry.type)}/${entry.item.id}`}
                      className="font-medium text-sm text-foreground hover:underline truncate"
                    >
                      {entry.item.name}
                    </Link>
                  ) : (
                    <span className="font-medium text-sm text-foreground truncate">
                      {entry.item.name}
                    </span>
                  )}
                  {hasUpdate && (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Badge
                          variant="default"
                          className="gap-1 text-xs px-1.5 py-0 shrink-0 cursor-default"
                        >
                          <Download className="h-2.5 w-2.5" />
                        </Badge>
                      </TooltipTrigger>
                      <TooltipContent>Update available</TooltipContent>
                    </Tooltip>
                  )}
                </div>
                <p className="text-xs text-muted-foreground truncate">
                  by {entry.item.author}
                </p>
              </div>

              {badges.length > 0 && (
                <div
                  className={cn(
                    "shrink-0 flex items-center gap-1 self-center justify-start text-left",
                    isMap && "ml-1",
                  )}
                >
                  {badges.slice(0, MAX_CARD_BADGES).map((badge) => (
                    <Badge
                      key={badge}
                      variant="secondary"
                      className="text-xs px-1.5 py-0"
                    >
                      {badge}
                    </Badge>
                  ))}
                  {badges.length > MAX_CARD_BADGES && (
                    <Badge variant="outline" className="text-xs px-1.5 py-0">
                      +{badges.length - MAX_CARD_BADGES}
                    </Badge>
                  )}
                </div>
              )}
            </div>
          </div>
        </TableCell>

        {/* Country (maps only) */}
        {showCountryColumn && (
          <TableCell className="align-middle text-center">
            {isMap && mapCountry ? (
              <div className="mx-auto flex items-center justify-center gap-1.5 whitespace-nowrap">
                {CountryFlag && <CountryFlag className="h-3.5 w-5 rounded-[1px] shrink-0" />}
                <span className="font-mono text-sm font-bold text-foreground">{mapCountry}</span>
              </div>
            ) : (
              <span className="block h-5" aria-hidden="true" />
            )}
          </TableCell>
        )}

        {/* Version */}
        <TableCell className="align-middle text-center">
          <p className="text-sm font-mono tabular-nums text-foreground text-center whitespace-nowrap">
            {entry.installedVersion}
          </p>
        </TableCell>

        {/* Delete button */}
        <TableCell>
          <div className="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-yellow-500 hover:text-yellow-400 hover:bg-yellow-500/10"
              onClick={handleOpenInstallFolder}
              aria-label="Open install folder"
              disabled={!metroMakerDataPath}
            >
              <FolderOpen className="h-4 w-4" />
            </Button>

            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10"
              onClick={() => setUninstallOpen(true)}
              aria-label="Delete"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        </TableCell>
      </TableRow>

      {/* Uninstall dialog */}
      {uninstallOpen && (
        <UninstallDialog
          open={uninstallOpen}
          onOpenChange={setUninstallOpen}
          type={entry.type}
          id={entry.item.id}
          name={entry.item.name}
        />
      )}
    </>
  );
}
