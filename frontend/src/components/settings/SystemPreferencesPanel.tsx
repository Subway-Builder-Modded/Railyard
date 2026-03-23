import { Shield, Terminal } from 'lucide-react';
import { useMemo, useState } from 'react';
import { toast } from 'sonner';

import { SettingRow } from '@/components/settings/SettingRow';
import { SettingToggleButton } from '@/components/settings/SettingToggleButton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { useProfileStore } from '@/stores/profile-store';

import {
  GetPlatform,
  InstallLinuxSandbox,
  SandboxIsInstalled,
} from '../../../wailsjs/go/main/App';

export function SystemPreferencesPanel() {
  const profile = useProfileStore((s) => s.profile);
  const updateCommandLineArgs = useProfileStore((s) => s.updateCommandLineArgs);

  const [platform, setPlatform] = useState<string>('unknown');
  const [sandboxInstalled, setSandboxInstalled] = useState(false);

  useMemo(() => {
    GetPlatform().then((response) => {
      if (response.status === 'success') setPlatform(response.platform || 'unknown');
    });
  }, []);

  useMemo(() => {
    if (platform !== 'linux') return;
    SandboxIsInstalled().then((response) => {
      if (response.status === 'success') setSandboxInstalled(response.installed);
    });
  }, [platform]);

  const handleToggleDevTools = async () => {
    if (!profile) return;
    const newValue = !profile.systemPreferences?.useDevTools;
    try {
      await updateCommandLineArgs({ useDevTools: newValue });
      toast.success(`Developer tools ${newValue ? 'enabled' : 'disabled'}.`);
    } catch {
      toast.error('Failed to update developer tools setting.');
    }
  };

  const handleInstallSandbox = async () => {
    try {
      const response = await InstallLinuxSandbox();
      if (response.status === 'error') throw new Error(response.message || 'Failed to install Linux sandbox');
      setSandboxInstalled(true);
      toast.success('Linux sandbox installed successfully.');
    } catch {
      toast.error('Failed to install Linux sandbox. Check the logs for details.');
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>System Preferences</CardTitle>
        <CardDescription>Performance, update behavior, and developer options.</CardDescription>
      </CardHeader>
      <CardContent className="px-6 py-0">
        <div className="divide-y divide-border">
          {platform === 'linux' && (
            <SettingRow
              icon={<Shield className="h-4 w-4" />}
              iconClassName="bg-[color-mix(in_oklab,var(--install-primary)_12%,transparent)] text-[var(--install-primary)]"
              label="Linux Sandbox"
              badge={
                <Badge size="sm" variant={sandboxInstalled ? 'success' : 'outline'}>
                  {sandboxInstalled ? 'Installed' : 'Not Installed'}
                </Badge>
              }
              description="Improves mod compatibility and security on Linux"
              action={
                <Button
                  variant="default"
                  size="sm"
                  onClick={handleInstallSandbox}
                  disabled={sandboxInstalled}
                >
                  Install
                </Button>
              }
            />
          )}
          {/* Fix your game devs :/
          <SettingRow
            icon={<Database className="h-4 w-4" />}
            label="Extra Memory for Game"
            description={
              MAX_MEMORY_MB !== null
                ? `Allowed range: ${MIN_MEMORY_MB.toLocaleString()} – ${MAX_MEMORY_MB.toLocaleString()} MB`
                : 'Additional memory allocation passed to the game process'
            }
            action={
              <div className="flex items-center gap-1.5">
                <Input
                  type="number"
                  min={MIN_MEMORY_MB}
                  max={MAX_MEMORY_MB ?? undefined}
                  placeholder={MAX_MEMORY_MB !== null ? MAX_MEMORY_MB.toString() : '8192'}
                  value={extraMemoryDraft}
                  onChange={(event) => setExtraMemoryDraft(event.target.value)}
                  className="h-8 w-24 text-sm"
                />
                <Button
                  variant="outline"
                  size="sm"
                  className={UNINSTALL_ACCENT.solidButton}
                  onClick={handleClearExtraMemory}
                >
                  Clear
                </Button>
                <Button variant="outline" size="sm" onClick={handleSaveExtraMemory}>
                  Save
                </Button>
              </div>
            }
          />
          */}
          <SettingRow
            icon={<Terminal className="h-4 w-4" />}
            label="Developer Tools"
            description="Enable Chromium DevTools for inspecting the app interface"
            action={
              <SettingToggleButton
                enabled={!!profile?.systemPreferences?.useDevTools}
                onToggle={handleToggleDevTools}
                enabledLabel="Enabled"
                disabledLabel="Disabled"
              />
            }
          />
        </div>
      </CardContent>
    </Card>
  );
}
