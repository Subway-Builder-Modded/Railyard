import { useState } from "react";
import { Button } from "@/components/ui/button";
import { useLibraryStore } from "@/stores/library-store";
import { useInstalledStore } from "@/stores/installed-store";
import { UninstallDialog } from "@/components/dialogs/UninstallDialog";
import {
  Download,
  Share2,
  Ban,
  Trash2,
  CheckCircle,
} from "lucide-react";
import { type InstalledTaggedItem } from "@/hooks/use-filtered-installed-items";
import { types } from "../../../wailsjs/go/models";
import { toast } from "sonner";
import type { AssetType } from "@/lib/asset-types";

interface LibraryActionBarProps {
  allItems: InstalledTaggedItem[];
  updatesAvailable: Map<string, types.VersionInfo>;
}

export function LibraryActionBar({
  allItems,
  updatesAvailable,
}: LibraryActionBarProps) {
  const { selectedIds, clearSelection } = useLibraryStore();
  const { installMod, installMap } = useInstalledStore();
  const [uninstallTarget, setUninstallTarget] = useState<{
    type: AssetType;
    id: string;
    name: string;
  } | null>(null);
  const [updating, setUpdating] = useState(false);

  if (selectedIds.size === 0) return null;

  // Resolve selected items
  const selectedItems = allItems.filter((item) =>
    selectedIds.has(`${item.type}-${item.item.id}`),
  );

  // Check if any selected item has an update
  const selectedWithUpdates = selectedItems.filter((item) =>
    updatesAvailable.has(item.item.id),
  );
  const hasUpdates = selectedWithUpdates.length > 0;

  const handleUpdate = async () => {
    if (selectedWithUpdates.length === 0) return;
    setUpdating(true);
    try {
      for (const item of selectedWithUpdates) {
        const updateInfo = updatesAvailable.get(item.item.id);
        if (!updateInfo) continue;
        if (item.type === "mod") {
          await installMod(item.item.id, updateInfo.version);
        } else {
          await installMap(item.item.id, updateInfo.version);
        }
      }
      toast.success(
        `Updated ${selectedWithUpdates.length} item${selectedWithUpdates.length !== 1 ? "s" : ""} successfully.`,
      );
      clearSelection();
    } catch (err) {
      toast.error(
        `Failed to update: ${err instanceof Error ? err.message : String(err)}`,
      );
    } finally {
      setUpdating(false);
    }
  };

  const handleRemove = () => {
    if (selectedItems.length === 1) {
      const item = selectedItems[0];
      setUninstallTarget({
        type: item.type,
        id: item.item.id,
        name: item.item.name,
      });
    } else {
      // For multiple items, confirm via toast for now
      // A bulk uninstall dialog could be added later
      setUninstallTarget({
        type: selectedItems[0].type,
        id: selectedItems[0].item.id,
        name: `${selectedItems.length} items`,
      });
    }
  };

  return (
    <>
      <div className="flex items-center gap-2 px-4 py-2 bg-muted/50 border border-border rounded-lg animate-in slide-in-from-bottom-2 duration-200">
        <div className="flex items-center gap-1.5 mr-2">
          <CheckCircle className="h-4 w-4 text-primary" />
          <span className="text-sm font-medium text-foreground">
            {selectedIds.size} selected
          </span>
        </div>

        {hasUpdates && (
          <Button
            size="sm"
            onClick={handleUpdate}
            disabled={updating}
            className="gap-1.5"
          >
            <Download className="h-3.5 w-3.5" />
            Update
          </Button>
        )}

        <Button variant="outline" size="sm" className="gap-1.5" disabled>
          <Share2 className="h-3.5 w-3.5" />
          Share
        </Button>

        <Button variant="outline" size="sm" className="gap-1.5" disabled>
          <Ban className="h-3.5 w-3.5" />
          Disable
        </Button>

        <Button
          variant="destructive"
          size="sm"
          onClick={handleRemove}
          className="gap-1.5"
        >
          <Trash2 className="h-3.5 w-3.5" />
          Remove
        </Button>
      </div>

      {uninstallTarget && (
        <UninstallDialog
          open={!!uninstallTarget}
          onOpenChange={(open) => {
            if (!open) {
              setUninstallTarget(null);
              clearSelection();
            }
          }}
          type={uninstallTarget.type}
          id={uninstallTarget.id}
          name={uninstallTarget.name}
        />
      )}
    </>
  );
}
