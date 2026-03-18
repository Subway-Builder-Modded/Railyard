import { CheckCircle, Trash2 } from 'lucide-react';
import { useState } from 'react';

import { UninstallDialog } from '@/components/dialogs/UninstallDialog';
import { Button } from '@/components/ui/button';
import { type InstalledTaggedItem } from '@/hooks/use-filtered-installed-items';
import type { AssetType } from '@/lib/asset-types';
import { useLibraryStore } from '@/stores/library-store';

interface UninstallTarget {
  type: AssetType;
  id: string;
  name: string;
}

interface LibraryActionBarProps {
  allItems: InstalledTaggedItem[];
}

export function LibraryActionBar({ allItems }: LibraryActionBarProps) {
  const { selectedIds, removeSelected } = useLibraryStore();
  const [uninstallTargets, setUninstallTargets] = useState<
    UninstallTarget[] | null
  >(null);

  if (selectedIds.size === 0) return null;

  const selectedItems = allItems.filter((item) =>
    selectedIds.has(`${item.type}-${item.item.id}`),
  );

  const handleRemove = () => {
    setUninstallTargets(
      selectedItems.map((item) => ({
        type: item.type,
        id: item.item.id,
        name: item.item.name,
      })),
    );
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
        <UninstallDialog
          open={uninstallTargets.length > 0}
          onOpenChange={(open) => {
            if (!open) {
              setUninstallTargets(null);
            }
          }}
          onUninstallSuccess={(targets) => {
            const removedKeys = targets.map(
              (target) => `${target.type}-${target.id}`,
            );
            removeSelected(removedKeys);
          }}
          targets={uninstallTargets}
        />
      )}
    </>
  );
}
