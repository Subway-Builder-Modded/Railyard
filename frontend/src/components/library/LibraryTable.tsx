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
import { GalleryImage } from "@/components/shared/GalleryImage";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Download,
  ToggleRight,
  MoreVertical,
} from "lucide-react";
import { useLibraryStore } from "@/stores/library-store";
import { type InstalledTaggedItem } from "@/hooks/use-filtered-installed-items";
import { cn } from "@/lib/utils";
import { types } from "../../../wailsjs/go/models";

interface LibraryTableProps {
  items: InstalledTaggedItem[];
  updatesAvailable: Map<string, types.VersionInfo>;
}

function composeItemKey(item: InstalledTaggedItem): string {
  return `${item.type}-${item.item.id}`;
}

export function LibraryTable({ items, updatesAvailable }: LibraryTableProps) {
  const { selectedIds, toggleSelected, selectAll, clearSelection } =
    useLibraryStore();

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
            <TableHead className="w-12"></TableHead>
            <TableHead>
              <button
                className="flex items-center gap-1 text-foreground font-medium"
                type="button"
              >
                Name
                <span className="text-muted-foreground text-xs">▲</span>
              </button>
            </TableHead>
            <TableHead>Version</TableHead>
            <TableHead className="w-20 text-center">
              <ToggleRight className="h-4 w-4 mx-auto text-muted-foreground" />
            </TableHead>
            <TableHead className="w-12"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((entry) => {
            const key = composeItemKey(entry);
            const isSelected = selectedIds.has(key);
            const hasUpdate = updatesAvailable.has(entry.item.id);
            const updateVersion = updatesAvailable.get(entry.item.id);

            return (
              <LibraryTableRow
                key={key}
                entry={entry}
                itemKey={key}
                isSelected={isSelected}
                hasUpdate={hasUpdate}
                updateVersion={updateVersion}
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
  itemKey: string;
  isSelected: boolean;
  hasUpdate: boolean;
  updateVersion?: types.VersionInfo;
  onToggleSelect: () => void;
}

function LibraryTableRow({
  entry,
  isSelected,
  hasUpdate,
  onToggleSelect,
}: LibraryTableRowProps) {
  return (
    <TableRow
      data-state={isSelected ? "selected" : undefined}
      className={cn(
        "group transition-colors",
        isSelected && "bg-muted/50",
      )}
    >
      {/* Checkbox */}
      <TableCell>
        <Checkbox
          checked={isSelected}
          onCheckedChange={onToggleSelect}
          aria-label={`Select ${entry.item.name}`}
        />
      </TableCell>

      {/* Icon / thumbnail */}
      <TableCell className="p-0">
        <div className="w-10 h-10 rounded-md overflow-hidden bg-muted flex items-center justify-center">
          <GalleryImage
            type={entry.type}
            id={entry.item.id}
            imagePath={entry.item.gallery?.[0]}
            className="h-10 w-10 object-cover"
          />
        </div>
      </TableCell>

      {/* Name + author */}
      <TableCell>
        <div className="flex items-center gap-3 min-w-0">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <Link
                href={`/project/${entry.type}/${entry.item.id}`}
                className="font-medium text-sm text-foreground hover:underline truncate"
              >
                {entry.item.name}
              </Link>
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
        </div>
      </TableCell>

      {/* Version */}
      <TableCell>
        <div className="space-y-0.5">
          <p className="text-sm font-mono text-foreground">
            {entry.installedVersion}
          </p>
          <p className="text-xs text-muted-foreground truncate max-w-[200px]">
            {buildFileName(entry)}
          </p>
        </div>
      </TableCell>

      {/* Toggle (enabled/disabled) */}
      <TableCell className="text-center">
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              className="inline-flex items-center justify-center"
              aria-label="Enabled"
            >
              <ToggleRight className="h-5 w-5 text-primary" />
            </button>
          </TooltipTrigger>
          <TooltipContent>Enabled</TooltipContent>
        </Tooltip>
      </TableCell>

      {/* More menu */}
      <TableCell>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 text-muted-foreground hover:text-foreground opacity-0 group-hover:opacity-100 transition-opacity"
          aria-label="More actions"
        >
          <MoreVertical className="h-4 w-4" />
        </Button>
      </TableCell>
    </TableRow>
  );
}

function buildFileName(entry: InstalledTaggedItem): string {
  const id = entry.item.id;
  const version = entry.installedVersion;
  const suffix = entry.type === "maps" ? "" : ".jar";
  return `${id}-${version}${suffix}`;
}
