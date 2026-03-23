import { Trash2 } from 'lucide-react';

import { AssetActionDialog } from '@/components/dialogs/AssetActionDialog';

interface DeleteLogSessionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}

export function DeleteLogSessionDialog({
  open,
  onOpenChange,
  onConfirm,
}: DeleteLogSessionDialogProps) {
  return (
    <AssetActionDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Delete this session?"
      description="This will permanently remove the selected session logs."
      icon={Trash2}
      iconClassName="h-5 w-5 text-[var(--uninstall-primary)]"
      confirmLabel="Delete Session"
      confirmVariant="destructive"
      tone="uninstall"
      loading={false}
      onConfirm={() => {
        onConfirm();
        onOpenChange(false);
      }}
    />
  );
}
