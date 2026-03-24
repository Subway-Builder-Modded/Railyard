import { AlertTriangle } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { getLocalAccentClasses } from '@/lib/local-accent';

interface PrereleaseConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  itemName: string;
  version: string;
  onConfirm: () => void;
}

const ACCENT = getLocalAccentClasses('files');

export function PrereleaseConfirmDialog({
  open,
  onOpenChange,
  itemName,
  version,
  onConfirm,
}: PrereleaseConfirmDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-yellow-500" />
            Install Beta Release?
          </DialogTitle>
          <DialogDescription>
            <span className="font-semibold text-foreground">{itemName}</span>{' '}
            {version} is a pre-release version and may be unstable or contain
            bugs.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            variant="outline"
            className={ACCENT.outlineButton}
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button
            className={ACCENT.solidButton}
            onClick={() => {
              onOpenChange(false);
              onConfirm();
            }}
          >
            Install Anyway
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
