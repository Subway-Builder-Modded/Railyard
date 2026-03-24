import { Loader2, type LucideIcon } from 'lucide-react';
import { type ReactNode } from 'react';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  getLocalAccentClasses,
  getToneVarsClass,
  type LocalAccentTone,
} from '@/lib/local-accent';

interface AppDialogConfirm {
  label: ReactNode;
  cancelLabel?: string;
  onConfirm: () => void;
  loading?: boolean;
}

interface AppDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  icon: LucideIcon;
  description: ReactNode;
  tone: LocalAccentTone;
  confirm?: AppDialogConfirm;
  children?: ReactNode;
}

export function AppDialog({
  open,
  onOpenChange,
  title,
  icon: Icon,
  description,
  tone,
  confirm,
  children,
}: AppDialogProps) {
  const accent = getLocalAccentClasses(tone);
  const toneVarsClass = getToneVarsClass(tone);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Icon
              className={`h-5 w-5 ${toneVarsClass} text-[var(--local-tone-primary)]`}
            />
            {title}
          </DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        {children}

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={confirm?.loading}
            className={accent.dialogCancel}
          >
            {confirm ? (confirm.cancelLabel ?? 'Cancel') : 'Close'}
          </Button>
          {confirm && (
            <Button
              onClick={confirm.onConfirm}
              disabled={confirm.loading}
              className={`gap-1.5 ${accent.solidButton}`}
            >
              {confirm.loading && (
                <Loader2 className="h-4 w-4 animate-spin" />
              )}
              {confirm.label}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
