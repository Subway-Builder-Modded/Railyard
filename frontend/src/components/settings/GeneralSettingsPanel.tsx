import { FolderOpen, FolderSearch, Gamepad2, KeyRound, RefreshCw } from 'lucide-react';
import { useState } from 'react';
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { getLocalAccentClasses } from '@/lib/local-accent';
import { useConfigStore } from '@/stores/config-store';

import { ManuallyCheckForUpdates, OpenInFileExplorer } from '../../../wailsjs/go/main/App';

const FILES_ACCENT = getLocalAccentClasses('files');
const UPDATE_ACCENT = getLocalAccentClasses('update');

export function GeneralSettingsPanel() {
  const {
    config,
    validation,
    hasGithubToken,
    githubTokenValid,
    openDataFolderDialog,
    openExecutableDialog,
    saveConfig,
    clearConfig: _clearConfig,
    updateGithubToken,
    clearGithubToken,
    updateCheckForUpdatesOnLaunch,
  } = useConfigStore();

  const [githubTokenDialogOpen, setGithubTokenDialogOpen] = useState(false);
  const [githubTokenDraft, setGithubTokenDraft] = useState('');

  const handleRevealPath = async (path: string | undefined) => {
    if (!path) return;
    try {
      const result = await OpenInFileExplorer(path);
      if (result?.status === 'error') toast.error(result.message || 'Failed to open location.');
    } catch {
      toast.error('Failed to open location.');
    }
  };

  const handleUpdatesCheck = async () => {
    try {
      const response = await ManuallyCheckForUpdates();
      if (response.status === 'error') throw new Error(response.message || 'Failed to check for updates');
      toast.success('No updates found, or installation was cancelled.');
    } catch {
      toast.error('Failed to check for updates.');
    }
  };

  const handleChangeUpdatesOnLaunch = async () => {
    try {
      const newValue = !config?.checkForUpdatesOnLaunch;
      await updateCheckForUpdatesOnLaunch(newValue);
      toast.success(`Check for updates on launch ${newValue ? 'enabled' : 'disabled'}.`);
    } catch {
      toast.error('Failed to update check for updates on launch setting.');
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

  const handleCheckToken = async () => {
    const req = await fetch('https://api.github.com/rate_limit', {
      headers: { Authorization: `token ${githubTokenDraft.trim()}` },
    });
    if (req.status === 200) {
      toast.success('GitHub token is valid!');
    } else {
      toast.error('GitHub token is invalid. Please check and try again.');
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
    <>
      <Card>
        <CardHeader>
          <CardTitle>General Settings</CardTitle>
          <CardDescription>Configure game paths, tokens, and system integrations.</CardDescription>
        </CardHeader>
        <CardContent className="px-6 py-0">
          <div className="divide-y divide-border">
            <SettingRow
              icon={<RefreshCw className="h-4 w-4" />}
              iconClassName="bg-[color-mix(in_oklab,var(--update-primary)_12%,transparent)] text-[var(--update-primary)]"
              label="Check for Updates"
              description="Automatically look for Railyard updates when the app starts"
              action={
                <>
                  <Button
                    variant="outline"
                    size="sm"
                    className={UPDATE_ACCENT.solidButton}
                    onClick={handleUpdatesCheck}
                  >
                    <RefreshCw className="size-3.5" />
                    Check Now
                  </Button>
                  <SettingToggleButton
                    accent="update"
                    enabled={!!config?.checkForUpdatesOnLaunch}
                    onToggle={handleChangeUpdatesOnLaunch}
                    enabledLabel="Enabled"
                    disabledLabel="Disabled"
                  />
                </>
              }
            />

            <SettingRow
              icon={<FolderOpen className="h-4 w-4" />}
              iconClassName="bg-[color-mix(in_oklab,var(--files-primary)_12%,transparent)] text-[var(--files-primary)]"
              label="Data Folder"
              badge={
                <Badge
                  size="sm"
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
              }
              description={
                <span className="block max-w-xs truncate font-mono">
                  {config?.metroMakerDataPath || 'Not configured'}
                </span>
              }
              action={
                <>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={!config?.metroMakerDataPath}
                    className={FILES_ACCENT.solidButton}
                    onClick={() => handleRevealPath(config?.metroMakerDataPath)}
                  >
                    <FolderSearch className="size-3.5" />
                    Reveal
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    className={FILES_ACCENT.outlineButton}
                    onClick={handleChangeDataFolder}
                  >
                    Change
                  </Button>
                </>
              }
            />

            <SettingRow
              icon={<Gamepad2 className="h-4 w-4" />}
              iconClassName="bg-[color-mix(in_oklab,var(--files-primary)_12%,transparent)] text-[var(--files-primary)]"
              label="Game Executable"
              badge={
                <Badge
                  size="sm"
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
              }
              description={
                <span className="block max-w-xs truncate font-mono">
                  {config?.executablePath || 'Not configured'}
                </span>
              }
              action={
                <>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={!config?.executablePath}
                    className={FILES_ACCENT.solidButton}
                    onClick={() => handleRevealPath(config?.executablePath)}
                  >
                    <FolderSearch className="size-3.5" />
                    Reveal
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    className={FILES_ACCENT.outlineButton}
                    onClick={handleChangeExecutable}
                  >
                    Change
                  </Button>
                </>
              }
            />

            <SettingRow
              icon={<KeyRound className="h-4 w-4" />}
              label="Token"
              badge={
                <Badge
                  size="sm"
                  variant={
                    hasGithubToken
                      ? githubTokenValid
                        ? 'success'
                        : 'destructive'
                      : 'outline'
                  }
                >
                  {hasGithubToken
                    ? githubTokenValid
                      ? 'Valid'
                      : 'Invalid'
                    : 'Not Set'}
                </Badge>
              }
              description={
                hasGithubToken
                  ? '••••••••••••••••'
                  : 'Allows for an increased API rate limit (Optional)'
              }
              action={
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setGithubTokenDialogOpen(true)}
                >
                  {hasGithubToken ? 'Update' : 'Add'}
                </Button>
              }
            />
          </div>
        </CardContent>
      </Card>

      <Dialog
        open={githubTokenDialogOpen}
        onOpenChange={(open) => {
          setGithubTokenDialogOpen(open);
          if (!open) setGithubTokenDraft('');
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit GitHub Token</DialogTitle>
            <DialogDescription>
              Provide a GitHub token to avoid unauthorized GitHub API rate limits.
            </DialogDescription>
          </DialogHeader>
          <Input
            type="password"
            placeholder={hasGithubToken ? '••••••••••••••••' : 'github_pat_...'}
            value={githubTokenDraft}
            onChange={(event) => setGithubTokenDraft(event.target.value)}
            className="font-mono"
          />
          <DialogFooter className="gap-2">
            <Button variant="outline" onClick={handleClearGithubToken} disabled={!hasGithubToken}>
              Clear
            </Button>
            <Button variant="outline" onClick={handleCheckToken}>
              Check
            </Button>
            <Button onClick={handleSaveGithubToken} disabled={githubTokenDraft.trim() === ''}>
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
