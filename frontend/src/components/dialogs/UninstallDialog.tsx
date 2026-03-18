import { AlertTriangle, Loader2 } from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import type { AssetType } from '@/lib/asset-types';
import { useInstalledStore } from '@/stores/installed-store';

interface UninstallTarget {
  type: AssetType;
  id: string;
  name: string;
}

interface UninstallDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUninstallSuccess?: (targets: UninstallTarget[]) => void;
  type?: AssetType;
  id?: string;
  name?: string;
  targets?: UninstallTarget[];
}

export function UninstallDialog({
  open,
  onOpenChange,
  onUninstallSuccess,
  type,
  id,
  name,
  targets,
}: UninstallDialogProps) {
  const { uninstallAssets } = useInstalledStore();
  const [loading, setLoading] = useState(false);

  const uninstallTargets: UninstallTarget[] =
    targets ?? (type && id && name ? [{ type, id, name }] : []);
  const itemCount = uninstallTargets.length;
  const firstTarget = uninstallTargets[0];
  const titleName =
    itemCount === 1 ? (firstTarget?.name ?? 'item') : `${itemCount} items`;
  const singleType = itemCount === 1 ? firstTarget?.type : null;

  const handleUninstall = async () => {
    if (itemCount === 0) return;

    setLoading(true);
    try {
      await uninstallAssets(
        uninstallTargets.map((target) => ({
          id: target.id,
          type: target.type,
        })),
      );

      toast.success(
        itemCount === 1
          ? `${titleName} has been uninstalled.`
          : `${itemCount} assets have been uninstalled.`,
      );
      onUninstallSuccess?.(uninstallTargets);
      onOpenChange(false);
    } catch {
      toast.error(
        itemCount === 1
          ? `Failed to uninstall ${titleName}.`
          : `Failed to uninstall one or more selected assets.`,
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-destructive" />
            Uninstall {titleName}?
          </DialogTitle>
          <DialogDescription>
            {itemCount === 1
              ? `This will remove all installed files for this ${singleType === 'mod' ? 'mod' : 'map'}. You can reinstall it later from the registry.`
              : 'This will remove all installed files for the selected assets. You can reinstall them later from the registry.'}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={loading}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleUninstall}
            disabled={loading}
          >
            {loading && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
            Uninstall
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
