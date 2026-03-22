import {
  CheckCircle,
  ChevronRight,
  FolderSearch,
  Gamepad2,
  Github,
  Loader2,
  TrainTrack,
  XCircle,
} from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { useConfigStore } from '@/stores/config-store';

export function SetupScreen() {
  const [step, setStep] = useState(1);
  const [saving, setSaving] = useState(false);
  const [checkForUpdates, setCheckForUpdates] = useState<boolean | null>(null);
  const [githubToken, setGithubToken] = useState('');
  const {
    config,
    validation,
    openDataFolderDialog,
    openExecutableDialog,
    updateCheckForUpdatesOnLaunch,
    updateGithubToken,
    completeSetup,
  } = useConfigStore();

  const handleCheckToken = async () => {
    let req = await fetch('https://api.github.com/rate_limit', {
      headers: {
        Authorization: `token ${githubToken.trim()}`,
      },
    });
    if (req.status === 200) {
      toast.success('GitHub token is valid!');
    } else {
      toast.error('GitHub token is invalid. Please check and try again.');
    }
  };

  const handleDataFolder = async (autoDetect: boolean) => {
    try {
      const result = await openDataFolderDialog(autoDetect);
      if (result.source === 'cancelled') return;
    } catch {
      // error is set in the store
    }
  };

  const handleExecutable = async (autoDetect: boolean) => {
    try {
      const result = await openExecutableDialog(autoDetect);
      if (result.source === 'cancelled') return;
    } catch {
      // error is set in the store
    }
  };

  const handleFinish = async () => {
    setSaving(true);
    try {
      if (checkForUpdates !== null) {
        await updateCheckForUpdatesOnLaunch(checkForUpdates);
      }
      const trimmedGithubToken = githubToken.trim();
      if (trimmedGithubToken !== '') {
        await updateGithubToken(trimmedGithubToken);
      }
      await completeSetup();
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-background">
      <Card className="w-full max-w-lg">
        <CardHeader className="text-center">
          {step === 1 ? (
            <>
              <div className="mx-auto mb-2">
                <TrainTrack className="h-10 w-10 text-primary" />
              </div>
              <CardTitle className="text-2xl">Welcome to Railyard</CardTitle>
              <CardDescription>
                Set the path to your MetroMaker data folder.
              </CardDescription>
            </>
          ) : step === 2 ? (
            <>
              <div className="mx-auto mb-2">
                <Gamepad2 className="h-10 w-10 text-primary" />
              </div>
              <CardTitle className="text-2xl">Game Executable</CardTitle>
              <CardDescription>
                Set the path to your game executable.
              </CardDescription>
            </>
          ) : step === 3 ? (
            <>
              <div className="mx-auto mb-2">
                <Github className="h-10 w-10 text-primary" />
              </div>
              <CardTitle className="text-2xl">Optional GitHub Token</CardTitle>
              <CardDescription>
                Add a token for higher GitHub API limits, or skip and continue.
              </CardDescription>
            </>
          ) : (
            <>
              <div className="mx-auto mb-2">
                <CheckCircle className="h-10 w-10 text-green-600" />
              </div>
              <CardTitle className="text-2xl">
                Automatically Check for Updates
              </CardTitle>
              <CardDescription>
                Would you like Railyard to automatically check for updates when
                it launches? You can change this later in settings.
              </CardDescription>
            </>
          )}

          {/* Step indicators */}
          <div className="flex items-center justify-center gap-2 pt-2">
            <span
              className={`h-2 w-6 rounded-full ${
                step === 1 ? 'bg-primary' : 'bg-muted'
              }`}
            />
            <span
              className={`h-2 w-6 rounded-full ${
                step === 2 ? 'bg-primary' : 'bg-muted'
              }`}
            />
            <span
              className={`h-2 w-6 rounded-full ${
                step === 3 ? 'bg-primary' : 'bg-muted'
              }`}
            />
            <span
              className={`h-2 w-6 rounded-full ${
                step === 4 ? 'bg-primary' : 'bg-muted'
              }`}
            />
          </div>
        </CardHeader>

        <CardContent className="space-y-6">
          {step === 1 ? (
            <>
              <div className="flex flex-col gap-3 sm:flex-row">
                <Button
                  className="flex-1"
                  onClick={() => handleDataFolder(true)}
                >
                  <FolderSearch className="mr-2 h-4 w-4" />
                  Auto-detect
                </Button>
                <Button
                  variant="outline"
                  className="flex-1"
                  onClick={() => handleDataFolder(false)}
                >
                  Browse...
                </Button>
              </div>

              {config?.metroMakerDataPath && (
                <div className="space-y-2">
                  <code className="block rounded bg-muted px-3 py-2 text-sm font-mono break-all">
                    {config.metroMakerDataPath}
                  </code>
                  {validation?.metroMakerDataPathValid ? (
                    <Badge variant="success" className="gap-1">
                      <CheckCircle className="h-3 w-3" />
                      Valid
                    </Badge>
                  ) : (
                    <Badge variant="destructive" className="gap-1">
                      <XCircle className="h-3 w-3" />
                      Invalid
                    </Badge>
                  )}
                </div>
              )}

              <div className="flex justify-end">
                <Button
                  onClick={() => setStep(2)}
                  disabled={!validation?.metroMakerDataPathValid}
                >
                  Next
                  <ChevronRight className="ml-2 h-4 w-4" />
                </Button>
              </div>
            </>
          ) : step === 2 ? (
            <>
              <div className="flex flex-col gap-3 sm:flex-row">
                <Button
                  className="flex-1"
                  onClick={() => handleExecutable(true)}
                >
                  <FolderSearch className="mr-2 h-4 w-4" />
                  Auto-detect
                </Button>
                <Button
                  variant="outline"
                  className="flex-1"
                  onClick={() => handleExecutable(false)}
                >
                  Browse...
                </Button>
              </div>

              {config?.executablePath && (
                <div className="space-y-2">
                  <code className="block rounded bg-muted px-3 py-2 text-sm font-mono break-all">
                    {config.executablePath}
                  </code>
                  {validation?.executablePathValid ? (
                    <Badge variant="success" className="gap-1">
                      <CheckCircle className="h-3 w-3" />
                      Valid
                    </Badge>
                  ) : (
                    <Badge variant="destructive" className="gap-1">
                      <XCircle className="h-3 w-3" />
                      Invalid
                    </Badge>
                  )}
                </div>
              )}

              <div className="flex justify-between">
                <Button variant="outline" onClick={() => setStep(1)}>
                  Back
                </Button>
                <Button
                  onClick={() => setStep(3)}
                  disabled={!validation?.executablePathValid}
                >
                  Next
                  <ChevronRight className="ml-2 h-4 w-4" />
                </Button>
              </div>
            </>
          ) : step === 3 ? (
            <>
              <div className="space-y-2">
                <p className="text-sm font-medium">Optional GitHub token</p>
                <Input
                  type="password"
                  value={githubToken}
                  onChange={(event) => setGithubToken(event.target.value)}
                  placeholder="github_pat_..."
                  className="font-mono whitespace-nowrap overflow-x-auto"
                />
                <p className="text-xs text-muted-foreground">
                  Leave blank to skip. You can add or change this later in
                  Settings.
                </p>
              </div>

              <div className="flex justify-between">
                <Button variant="outline" onClick={() => setStep(2)}>
                  Back
                </Button>
                <Button
                  variant="outline"
                  onClick={handleCheckToken}
                  disabled={githubToken.trim() === ''}
                >
                  <CheckCircle className="mr-2 h-4 w-4" />
                  Check Token
                </Button>
                <Button onClick={() => setStep(4)}>
                  Next
                  <ChevronRight className="ml-2 h-4 w-4" />
                </Button>
              </div>
            </>
          ) : (
            <>
              <div className="space-y-3">
                <div className="flex items-center gap-4 w-full justify-center">
                  <Button
                    variant={checkForUpdates === true ? 'default' : 'outline'}
                    onClick={() => setCheckForUpdates(true)}
                  >
                    Yes
                  </Button>
                  <Button
                    variant={checkForUpdates === false ? 'default' : 'outline'}
                    onClick={() => setCheckForUpdates(false)}
                  >
                    No
                  </Button>
                </div>
              </div>

              <div className="flex justify-between">
                <Button variant="outline" onClick={() => setStep(3)}>
                  Back
                </Button>
                <Button
                  onClick={handleFinish}
                  disabled={checkForUpdates === null || saving}
                >
                  {saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  Finish Setup
                </Button>
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
