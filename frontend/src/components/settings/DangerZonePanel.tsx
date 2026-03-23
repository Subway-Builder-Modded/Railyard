import { AlertTriangle, Settings } from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';

import { SettingRow } from '@/components/settings/SettingRow';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { useConfigStore } from '@/stores/config-store';
import { useProfileStore } from '@/stores/profile-store';

export function DangerZonePanel() {
  const { clearConfig } = useConfigStore();
  const resetProfile = useProfileStore((s) => s.resetProfile);

  const [confirmAction, setConfirmAction] = useState<'config' | 'profile' | null>(null);

  const handleConfirm = async () => {
    try {
      if (confirmAction === 'config') {
        await clearConfig();
        toast.success('Configuration has been reset.');
      } else if (confirmAction === 'profile') {
        await resetProfile();
        toast.success('Profile has been reset.');
      }
    } catch {
      toast.error(`Failed to reset ${confirmAction}.`);
    } finally {
      setConfirmAction(null);
    }
  };

  return (
    <>
      <Card className="ring-destructive/35 bg-[color-mix(in_oklab,var(--color-destructive)_6%,transparent)]">
        <CardHeader>
          <CardTitle className="text-destructive">Danger Zone</CardTitle>
          <CardDescription>These actions are permanent and cannot be undone.</CardDescription>
        </CardHeader>
        <CardContent className="px-6 py-0">
          <div className="divide-y divide-destructive/15">
            <SettingRow
              icon={<Settings className="h-4 w-4" />}
              iconClassName="bg-destructive/10 text-destructive"
              label="Reset Configuration"
              description="Clear all saved paths and settings. You will need to configure them again from scratch."
              action={
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => setConfirmAction('config')}
                >
                  Reset Config
                </Button>
              }
            />
            <SettingRow
              icon={<AlertTriangle className="h-4 w-4" />}
              iconClassName="bg-destructive/10 text-destructive"
              label="Reset Profile"
              description="Clear your profile and all UI preferences, reverting everything to defaults."
              action={
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => setConfirmAction('profile')}
                >
                  Reset Profile
                </Button>
              }
            />
          </div>
        </CardContent>
      </Card>

      <Dialog
        open={confirmAction !== null}
        onOpenChange={(open) => !open && setConfirmAction(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              Are you sure?
            </DialogTitle>
            <DialogDescription>
              {confirmAction === 'config'
                ? 'This will reset all configuration including game paths. You will need to set them up again.'
                : 'This will reset your profile and all preferences to defaults.'}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmAction(null)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleConfirm}>
              Reset
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
