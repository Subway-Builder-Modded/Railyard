import { useState } from "react";
import { useInstalledStore } from "@/stores/installed-store";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { AlertTriangle, Loader2 } from "lucide-react";
import type { AssetType } from "@/lib/asset-types";

interface UninstallDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  type: AssetType;
  id: string;
  name: string;
}

export function UninstallDialog({ open, onOpenChange, type, id, name }: UninstallDialogProps) {
  const { uninstallMod, uninstallMap } = useInstalledStore();
  const [loading, setLoading] = useState(false);

  const handleUninstall = async () => {
    setLoading(true);
    try {
      if (type === "mod") {
        await uninstallMod(id);
      } else {
        await uninstallMap(id);
      }
      toast.success(`${name} has been uninstalled.`);
      onOpenChange(false);
    } catch {
      toast.error(`Failed to uninstall ${name}.`);
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
            Uninstall {name}?
          </DialogTitle>
          <DialogDescription>
            This will remove all installed files for this {type === "mod" ? "mod" : "map"}. You can reinstall it later from the registry.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={handleUninstall} disabled={loading}>
            {loading && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
            Uninstall
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
