import { CheckCircle, CircleFadingArrowUp, OctagonX, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';

import { AppDialog } from '@/components/dialogs/AppDialog';
import { Button } from '@/components/ui/button';
import { type InstalledTaggedItem } from '@/hooks/use-filtered-installed-items';
import { getLocalAccentClasses } from '@/lib/local-accent';
import {
  type AssetTarget,
  composeAssetKey,
  type PendingUpdatesByKey,
  type PendingUpdateTarget,
  toPendingUpdateTargets,
} from '@/lib/subscription-updates';
import { useInstalledStore } from '@/stores/installed-store';
import { useLibraryStore } from '@/stores/library-store';

interface LibraryActionBarProps {
  allItems: InstalledTaggedItem[];
  pendingUpdatesByKey: PendingUpdatesByKey;
  onRefreshPendingUpdates: () => Promise<void>;
}

const ENTRIES_PREVIEW_LIMIT = 10;
const UPDATE_ACCENT_BUTTON_CLASS = getLocalAccentClasses('update').solidButton;

export function LibraryActionBar({
  allItems,
  pendingUpdatesByKey,
  onRefreshPendingUpdates,
}: LibraryActionBarProps) {
  const { selectedIds, removeSelected } = useLibraryStore();
  const { uninstallAssets, updateAssetsToLatest } = useInstalledStore();
  const [uninstallTargets, setUninstallTargets] = useState<
    AssetTarget[] | null
  >(null);
  const [uninstallLoading, setUninstallLoading] = useState(false);
  const [updateTargets, setUpdateTargets] = useState<
    PendingUpdateTarget[] | null
  >(null);
  const [updateLoading, setUpdateLoading] = useState(false);

  if (selectedIds.size === 0) return null;

  const selectedTargets: AssetTarget[] = allItems
    .filter((item) => selectedIds.has(composeAssetKey(item.type, item.item.id)))
    .map((item) => ({
      type: item.type,
      id: item.item.id,
      name: item.item.name,
    }));

  const selectedUpdateTargets = toPendingUpdateTargets(
    selectedTargets,
    pendingUpdatesByKey,
  );

  const handleRemove = () => {
    setUninstallTargets(selectedTargets);
  };

  const handleUpdate = () => {
    setUpdateTargets(selectedUpdateTargets);
  };

  const handleConfirmUninstall = async () => {
    if (!uninstallTargets || uninstallTargets.length === 0) return;
    setUninstallLoading(true);
    try {
      await uninstallAssets(
        uninstallTargets.map((t) => ({ id: t.id, type: t.type })),
      );
      const count = uninstallTargets.length;
      toast.success(
        count === 1
          ? `${uninstallTargets[0].name} has been uninstalled.`
          : `${count} assets have been uninstalled.`,
      );
      const removedKeys = uninstallTargets.map((t) =>
        composeAssetKey(t.type, t.id),
      );
      removeSelected(removedKeys);
      void onRefreshPendingUpdates();
      setUninstallTargets(null);
    } catch {
      toast.error('Failed to uninstall selected assets.');
    } finally {
      setUninstallLoading(false);
    }
  };

  const handleConfirmUpdate = async () => {
    if (!updateTargets || updateTargets.length === 0) return;
    setUpdateLoading(true);
    try {
      await updateAssetsToLatest(
        updateTargets.map((t) => ({ id: t.id, type: t.type })),
      );
      const count = updateTargets.length;
      toast.success(
        count === 1
          ? `${updateTargets[0].name} has been updated.`
          : `${count} assets have been updated.`,
      );
      void onRefreshPendingUpdates();
      setUpdateTargets(null);
    } catch {
      toast.error(
        updateTargets.length === 1
          ? `Failed to update ${updateTargets[0].name}.`
          : 'Failed to update one or more selected assets.',
      );
    } finally {
      setUpdateLoading(false);
    }
  };

  const uninstallCount = uninstallTargets?.length ?? 0;
  const updateCount = updateTargets?.length ?? 0;

  const sortedUpdateTargets = updateTargets
    ? [...updateTargets].sort((a, b) => a.name.localeCompare(b.name))
    : [];
  const updatePreviewEntries = sortedUpdateTargets.slice(0, ENTRIES_PREVIEW_LIMIT);
  const updateRemainingCount = Math.max(
    0,
    sortedUpdateTargets.length - updatePreviewEntries.length,
  );

  return (
    <>
      <div className="flex items-center gap-2 px-4 py-2 bg-muted/50 border border-border rounded-lg animate-in slide-in-from-bottom-2 duration-200">
        <div className="flex items-center gap-1.5 mr-2">
          <CheckCircle className="h-4 w-4 text-primary" />
          <span className="text-sm font-medium text-foreground">
            {selectedIds.size} selected
          </span>
        </div>

        {selectedUpdateTargets.length > 0 && (
          <Button
            size="sm"
            onClick={handleUpdate}
            className={`gap-1.5 ${UPDATE_ACCENT_BUTTON_CLASS}`}
          >
            <CircleFadingArrowUp className="h-3.5 w-3.5" />
            Update Selected
          </Button>
        )}

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

      {uninstallTargets && uninstallTargets.length > 0 && (
        <AppDialog
          open={uninstallTargets.length > 0}
          onOpenChange={(open) => {
            if (!open) setUninstallTargets(null);
          }}
          title={`Uninstall`}
          description={
            uninstallCount === 1
              ? `This will remove all installed files for the selected ${uninstallTargets[0].type === 'mod' ? 'mod' : 'map'}. You can reinstall it later from the Browse page.`
              : `This will remove all installed files for the selected ${uninstallTargets[0].type === 'mod' ? 'mods' : 'maps'}. You can reinstall them later from the Browse page.`
          }
          icon={OctagonX}
          tone="uninstall"
          confirm={{
            label: 'Uninstall',
            onConfirm: handleConfirmUninstall,
            loading: uninstallLoading,
          }}
        />
      )}

      {updateTargets && updateTargets.length > 0 && (
        <AppDialog
          open={updateTargets.length > 0}
          onOpenChange={(open) => {
            if (!open) setUpdateTargets(null);
          }}
          title={'Update'}
          description={
            updateCount === 1 && updateTargets[0]
              ? `This will update the selected ${updateTargets[0].type === 'mod' ? 'mod' : 'map'} to its latest available version.`
              : `This will update the selected ${updateTargets[0].type === 'mod' ? 'mods' : 'maps'} to their latest available versions.`
          }
          icon={CircleFadingArrowUp}
          tone="update"
          confirm={{
            label: 'Update',
            onConfirm: handleConfirmUpdate,
            loading: updateLoading,
          }}
        >
          {updatePreviewEntries.length > 0 && (
            <div className="max-h-48 overflow-y-auto rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
              <ul className="space-y-1">
                {updatePreviewEntries.map((t) => (
                  <li key={`${t.type}-${t.id}`} className="flex gap-2">
                    <span className="min-w-0 flex-1 truncate">{t.name}</span>
                    <span className="font-mono tabular-nums text-foreground">
                      {t.currentVersion} &rarr; {t.latestVersion}
                    </span>
                  </li>
                ))}
                {updateRemainingCount > 0 && (
                  <li className="pt-1 text-right font-medium text-muted-foreground">
                    +{updateRemainingCount} more
                  </li>
                )}
              </ul>
            </div>
          )}
        </AppDialog>
      )}
    </>
  );
}
