import {
  AlertTriangle,
  ChevronDown,
  FolderOpen,
  Gamepad2,
  Github,
  RefreshCw,
  Settings,
  Shield,
} from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';

import { PageHeading } from '@/components/shared/PageHeading';
import { ThemePicker, type ThemeValue } from '@/components/shared/ThemePicker';
import { Badge } from '@/components/ui/badge';
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
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  isSearchViewMode,
  normalizeSearchViewMode,
} from '@/lib/search-view-mode';
import { useConfigStore } from '@/stores/config-store';
import { useProfileStore } from '@/stores/profile-store';

import {
  GetPlatform,
  GetTotalMemory,
  InstallLinuxSandbox,
  ManuallyCheckForUpdates,
  SandboxIsInstalled,
} from '../../wailsjs/go/main/App';

const PAGE_SIZE_OPTIONS = [12, 24, 48] as const;

const VALID_THEMES = new Set<ThemeValue>(['dark', 'light', 'system']);

const THEME_LABELS: Record<ThemeValue, string> = {
  dark: 'Dark',
  light: 'Light',
  system: 'System',
};

function normalizeThemeValue(theme: unknown): ThemeValue {
  if (theme === 'system' || theme === 'light' || theme === 'dark') {
    return theme;
  }

  if (typeof theme === 'string') {
    const lowered = theme.toLowerCase();
    if (lowered.startsWith('light')) return 'light';
    if (lowered.startsWith('dark')) return 'dark';
  }

  return 'dark';
}

export function SettingsPage() {
  const {
    config,
    validation,
    hasGithubToken,
    githubTokenValid,
    openDataFolderDialog,
    openExecutableDialog,
    saveConfig,
    clearConfig,
    updateGithubToken,
    clearGithubToken,
    updateCheckForUpdatesOnLaunch,
  } = useConfigStore();
  const profile = useProfileStore((s) => s.profile);
  const resetProfile = useProfileStore((s) => s.resetProfile);
  const updateUIPreferences = useProfileStore((s) => s.updateUIPreferences);
  const updateCommandLineArgs = useProfileStore((s) => s.updateCommandLineArgs);
  const [showThemePreviews, setShowThemePreviews] = useState(false);
  const [extraMemoryDraft, setExtraMemoryDraft] = useState('');
  const [MAX_MEMORY_MB, setMaxMemoryMB] = useState<number | null>(null);

  const MIN_MEMORY_MB = 4096;

  useEffect(() => {
    GetTotalMemory().then((totalMemoryMB) => {
      setMaxMemoryMB(Math.floor(totalMemoryMB * (5 / 8)));
    });
  }, []);

  useEffect(() => {
    if (profile?.systemPreferences?.extraMemorySize !== -1) {
      setExtraMemoryDraft(
        String(profile?.systemPreferences?.extraMemorySize ?? ''),
      );
    } else {
      setExtraMemoryDraft('');
    }
  }, [profile?.systemPreferences?.extraMemorySize]);

  const handleCheckToken = async () => {
    let req = await fetch('https://api.github.com/rate_limit', {
      headers: {
        Authorization: `token ${githubTokenDraft.trim()}`,
      },
    });
    if (req.status === 200) {
      toast.success('GitHub token is valid!');
    } else {
      toast.error('GitHub token is invalid. Please check and try again.');
    }
  };

  const [platform, setPlatform] = useState<string>('unknown');
  useMemo(() => {
    GetPlatform().then((response) => {
      if (response.status === 'success') {
        setPlatform(response.platform || 'unknown');
      }
    });
  }, []);

  const [sandboxInstalled, setSandboxInstalled] = useState(false);
  useMemo(() => {
    if (platform !== 'linux') return;
    SandboxIsInstalled().then((response) => {
      if (response.status === 'success') {
        setSandboxInstalled(response.installed);
      }
    });
  }, [platform]);

  const handleInstallSandbox = async () => {
    try {
      const response = await InstallLinuxSandbox();
      if (response.status === 'error') {
        throw new Error(response.message || 'Failed to install Linux sandbox');
      }
      setSandboxInstalled(true);
      toast.success('Linux sandbox installed successfully.');
    } catch (e) {
      toast.error(
        'Failed to install Linux sandbox. Check the logs for details.',
      );
    }
  };

  const [confirmAction, setConfirmAction] = useState<
    'config' | 'profile' | null
  >(null);
  const [githubTokenDialogOpen, setGithubTokenDialogOpen] = useState(false);
  const [githubTokenDraft, setGithubTokenDraft] = useState('');

  const handleUpdatesCheck = async () => {
    try {
      const response = await ManuallyCheckForUpdates();
      if (response.status === 'error') {
        throw new Error(response.message || 'Failed to check for updates');
      }
      toast.success('No updates found, or installation was cancelled.');
    } catch {
      toast.error('Failed to check for updates.');
    }
  };

  const handleChangeUpdatesOnLaunch = async () => {
    try {
      const newValue = !config?.checkForUpdatesOnLaunch;
      await updateCheckForUpdatesOnLaunch(newValue);
      toast.success(
        `Check for updates on launch ${newValue ? 'enabled' : 'disabled'}.`,
      );
    } catch {
      toast.error('Failed to update check for updates on launch setting.');
    }
  };

  const handleSaveExtraMemory = async () => {
    if (!profile) return;
    const parsed = Number.parseInt(extraMemoryDraft, 10);

    if (
      !Number.isFinite(parsed) ||
      parsed < MIN_MEMORY_MB ||
      parsed > MAX_MEMORY_MB!
    ) {
      toast.error(
        'Extra memory size must be between 4096 MB and no more than about 60% of your system memory (' +
          MAX_MEMORY_MB! +
          ' MB).',
      );
      return;
    }

    try {
      await updateCommandLineArgs({ extraMemorySize: parsed });
      toast.success('Extra memory size updated.');
    } catch {
      toast.error('Failed to update extra memory size.');
    }
  };

  const handleClearExtraMemory = async () => {
    if (!profile) return;
    try {
      setExtraMemoryDraft('');
      await updateCommandLineArgs({ extraMemorySize: -1 });
      toast.success('Extra memory size cleared.');
    } catch {
      toast.error('Failed to clear extra memory size.');
    }
  };

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

  const handleThemeChange = async (theme: ThemeValue) => {
    if (!profile || !VALID_THEMES.has(theme)) return;

    try {
      await updateUIPreferences({ theme });
      toast.success('Theme updated.');
    } catch {
      toast.error('Failed to update theme.');
    }
  };

  const handleDefaultPerPageChange = async (value: string) => {
    if (!profile) return;
    const parsed = Number.parseInt(value, 10);

    if (
      !PAGE_SIZE_OPTIONS.includes(parsed as (typeof PAGE_SIZE_OPTIONS)[number])
    )
      return;

    try {
      await updateUIPreferences({ defaultPerPage: parsed });
      toast.success('Default cards per page updated.');
    } catch {
      toast.error('Failed to update default cards per page.');
    }
  };

  const handleDefaultSearchViewModeChange = async (value: string) => {
    if (!profile) {
      console.warn(
        '[settings] Cannot update default browse view mode: profile is not loaded.',
      );
      return;
    }

    if (!isSearchViewMode(value)) {
      console.warn(
        `[settings] Ignoring invalid browse view mode value: ${String(value)}`,
      );
      return;
    }

    try {
      await updateUIPreferences({ searchViewMode: value });
      toast.success('Default browse view mode updated.');
    } catch (error) {
      console.warn(
        '[settings] Failed to persist default browse view mode preference.',
        error,
      );
      toast.error('Failed to update default browse view mode.');
    }
  };

  const handleChangeDataFolder = async () => {
    try {
      const result = await openDataFolderDialog(false);
      if (result.source === 'cancelled') return;
      await saveConfig();
      toast.success('Data folder path updated.');
    } catch {
      toast.error('Failed to update data folder path.');
    }
  };

  const handleChangeExecutable = async () => {
    try {
      const result = await openExecutableDialog(false);
      if (result.source === 'cancelled') return;
      await saveConfig();
      toast.success('Executable path updated.');
    } catch {
      toast.error('Failed to update executable path.');
    }
  };

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

  const handleSaveGithubToken = async () => {
    try {
      await updateGithubToken(githubTokenDraft);
      await saveConfig();
      setGithubTokenDraft('');
      setGithubTokenDialogOpen(false);
      toast.success('GitHub token updated.');
    } catch {
      toast.error('Failed to update GitHub token.');
    }
  };

  const handleClearGithubToken = async () => {
    try {
      await clearGithubToken();
      await saveConfig();
      setGithubTokenDraft('');
      setGithubTokenDialogOpen(false);
      toast.success('GitHub token cleared.');
    } catch {
      toast.error('Failed to clear GitHub token.');
    }
  };

  return (
    <div className="space-y-6">
      <PageHeading
        icon={Settings}
        title="Settings"
        description="Configure Railyard and customize your experience."
      />

      <Card>
        <CardHeader>
          <CardTitle>Global Settings</CardTitle>
          <CardDescription>
            Configure system-wide behavior and defaults.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3 min-w-0">
              <FolderOpen className="h-5 w-5 shrink-0 text-muted-foreground" />
              <div className="min-w-0">
                <p className="text-sm font-medium">Data Folder</p>
                <p className="text-xs text-muted-foreground font-mono truncate">
                  {config?.metroMakerDataPath || 'Not set'}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Badge
                variant={
                  validation?.metroMakerDataPathValid
                    ? 'success'
                    : config?.metroMakerDataPath
                      ? 'destructive'
                      : 'outline'
                }
              >
                {validation?.metroMakerDataPathValid
                  ? 'Valid'
                  : config?.metroMakerDataPath
                    ? 'Invalid'
                    : 'Not Set'}
              </Badge>
              <Button
                variant="outline"
                size="sm"
                onClick={handleChangeDataFolder}
              >
                Change
              </Button>
            </div>
          </div>

          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3 min-w-0">
              <Gamepad2 className="h-5 w-5 shrink-0 text-muted-foreground" />
              <div className="min-w-0">
                <p className="text-sm font-medium">Game Executable</p>
                <p className="text-xs text-muted-foreground font-mono truncate">
                  {config?.executablePath || 'Not set'}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Badge
                variant={
                  validation?.executablePathValid
                    ? 'success'
                    : config?.executablePath
                      ? 'destructive'
                      : 'outline'
                }
              >
                {validation?.executablePathValid
                  ? 'Valid'
                  : config?.executablePath
                    ? 'Invalid'
                    : 'Not Set'}
              </Badge>
              <Button
                variant="outline"
                size="sm"
                onClick={handleChangeExecutable}
              >
                Change
              </Button>
            </div>
          </div>

          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3 min-w-0">
              <Github className="h-5 w-5 shrink-0 text-muted-foreground" />
              <div className="min-w-0">
                <p className="text-sm font-medium">GitHub Token (Optional)</p>
                <p className="text-xs text-muted-foreground font-mono truncate">
                  {hasGithubToken ? '********' : 'Not set'}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Badge variant={hasGithubToken ? 'default' : 'outline'}>
                {hasGithubToken
                  ? githubTokenValid
                    ? 'Set, Valid'
                    : 'Set, Invalid'
                  : 'Unset'}
              </Badge>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setGithubTokenDialogOpen(true)}
              >
                Change
              </Button>
            </div>
          </div>

          {platform == 'linux' && (
            <div className="flex items-center justify-between gap-4">
              <div className="flex items-center gap-3 min-w-0">
                <Shield className="h-5 w-5 shrink-0 text-muted-foreground" />
                <div className="min-w-0">
                  <p className="text-sm font-medium">
                    Linux Sandbox (Optional)
                  </p>
                  <p className="text-xs text-muted-foreground">
                    Install the sandbox to potentially improve compatibility and
                    security on Linux.
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <Badge variant={sandboxInstalled ? 'default' : 'outline'}>
                  {sandboxInstalled ? 'Installed' : 'Not Installed'}
                </Badge>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleInstallSandbox}
                  disabled={sandboxInstalled}
                >
                  Install
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>UI Preferences</CardTitle>
          <CardDescription>
            Display preferences from your profile.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium">Theme</label>
              <Button
                variant="outline"
                type="button"
                onClick={() => setShowThemePreviews((current) => !current)}
                className="w-35 justify-between font-normal"
                aria-expanded={showThemePreviews}
              >
                {
                  THEME_LABELS[
                    normalizeThemeValue(profile?.uiPreferences?.theme)
                  ]
                }
                <ChevronDown
                  className={`size-4 shrink-0 text-muted-foreground opacity-50 transition-transform ${showThemePreviews ? 'rotate-180' : ''}`}
                />
              </Button>
            </div>

            {showThemePreviews && (
              <ThemePicker
                value={normalizeThemeValue(profile?.uiPreferences?.theme)}
                onChange={handleThemeChange}
              />
            )}
          </div>

          <div className="flex items-center justify-between">
            <label className="text-sm font-medium">Default Per Page</label>
            <Select
              value={String(profile?.uiPreferences?.defaultPerPage ?? 12)}
              onValueChange={handleDefaultPerPageChange}
            >
              <SelectTrigger className="w-35">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="12">12</SelectItem>
                <SelectItem value="24">24</SelectItem>
                <SelectItem value="48">48</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="flex items-center justify-between">
            <label className="text-sm font-medium">Default Browse View</label>
            <Select
              value={normalizeSearchViewMode(
                (
                  profile?.uiPreferences as
                    | { searchViewMode?: unknown }
                    | undefined
                )?.searchViewMode,
              )}
              onValueChange={handleDefaultSearchViewModeChange}
            >
              <SelectTrigger className="w-35">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="full">Full</SelectItem>
                <SelectItem value="compact">Compact</SelectItem>
                <SelectItem value="list">List</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>System Preferences</CardTitle>
          <CardDescription>
            System behavior preferences and update settings.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <label className="text-sm font-medium">
              Check For Updates On Launch
            </label>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={handleChangeUpdatesOnLaunch}
              >
                {config?.checkForUpdatesOnLaunch ? 'Disable' : 'Enable'}
              </Button>
              <Button variant="outline" size="sm" onClick={handleUpdatesCheck}>
                <RefreshCw />
                Check For Updates
              </Button>
            </div>
          </div>

          <div className="flex items-center justify-between">
            <label className="text-sm font-medium">
              Extra Memory for Game (MB)
            </label>
            <div className="flex items-center gap-2">
              <Input
                type="number"
                min={MIN_MEMORY_MB}
                max={MAX_MEMORY_MB ?? undefined}
                placeholder={
                  MAX_MEMORY_MB !== null ? MAX_MEMORY_MB!.toString() : '8192'
                }
                value={extraMemoryDraft}
                onChange={(event) => setExtraMemoryDraft(event.target.value)}
                className="w-[8lvh]"
              />
              <Button
                variant="outline"
                size="sm"
                onClick={handleClearExtraMemory}
              >
                Clear
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={handleSaveExtraMemory}
              >
                Save
              </Button>
            </div>
          </div>

          <div className="flex items-center justify-between">
            <label className="text-sm font-medium">Use Developer Tools</label>
            <Button variant="outline" size="sm" onClick={handleToggleDevTools}>
              {profile?.systemPreferences.useDevTools ? 'Disable' : 'Enable'}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Dialog
        open={githubTokenDialogOpen}
        onOpenChange={(open) => {
          setGithubTokenDialogOpen(open);
          if (!open) {
            setGithubTokenDraft('');
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit GitHub Token</DialogTitle>
            <DialogDescription>
              Provide a GitHub token to avoid unauthorized Github API rate
              limits.
            </DialogDescription>
          </DialogHeader>
          <Input
            type="password"
            placeholder={hasGithubToken ? '********' : 'github_pat_...'}
            value={githubTokenDraft}
            onChange={(event) => setGithubTokenDraft(event.target.value)}
            className="font-mono"
          />
          <DialogFooter className="gap-2">
            <Button
              variant="outline"
              onClick={handleClearGithubToken}
              disabled={!hasGithubToken}
            >
              Clear
            </Button>
            <Button variant="outline" onClick={handleCheckToken}>
              Check
            </Button>
            <Button
              onClick={handleSaveGithubToken}
              disabled={githubTokenDraft.trim() === ''}
            >
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Card className="ring-destructive/35 bg-[color-mix(in_oklab,var(--color-destructive)_6%,transparent)]">
        <CardHeader>
          <CardTitle className="text-destructive">Danger Zone</CardTitle>
          <CardDescription>
            Irreversible actions that reset your data.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Reset Configuration</p>
              <p className="text-xs text-muted-foreground">
                Clear all saved paths and settings.
              </p>
            </div>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setConfirmAction('config')}
            >
              Reset Config
            </Button>
          </div>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Reset Profile</p>
              <p className="text-xs text-muted-foreground">
                Clear your profile and preferences.
              </p>
            </div>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setConfirmAction('profile')}
            >
              Reset Profile
            </Button>
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
    </div>
  );
}
