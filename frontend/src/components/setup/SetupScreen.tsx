import {
  Check,
  CheckCircle,
  ChevronRight,
  Eye,
  EyeOff,
  FolderSearch,
  Loader2,
  RefreshCw,
  X,
  XCircle,
} from 'lucide-react';
import { type ChangeEvent, useState } from 'react';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import { useConfigStore } from '@/stores/config-store';

import { SETUP_STEPS, type SetupStepState } from './setup-steps';

interface PathDisplayProps {
  path: string;
  valid: boolean;
}

function PathDisplay({ path, valid }: PathDisplayProps) {
  return (
    <div
      className={cn(
        'flex items-center gap-3 rounded-lg border p-3 transition-colors duration-200',
        valid
          ? 'border-chart-2/30 bg-chart-2/5'
          : 'border-destructive/30 bg-destructive/5',
      )}
    >
      {valid ? (
        <CheckCircle className="size-4 shrink-0 text-chart-2" />
      ) : (
        <XCircle className="size-4 shrink-0 text-destructive" />
      )}
      <code className="flex-1 truncate text-xs font-mono text-muted-foreground">
        {path}
      </code>
    </div>
  );
}

interface PasswordInputProps {
  value: string;
  onChange: (e: ChangeEvent<HTMLInputElement>) => void;
  placeholder?: string;
  invalid?: boolean;
  className?: string;
}

function PasswordInput({
  value,
  onChange,
  placeholder,
  invalid,
  className,
}: PasswordInputProps) {
  const [show, setShow] = useState(false);

  return (
    <div className={cn('relative', className)}>
      <Input
        type={show ? 'text' : 'password'}
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        aria-invalid={invalid || undefined}
        className="font-mono pr-9 [&::-ms-reveal]:hidden focus-visible:ring-0 focus-visible:border-border"
      />
      {value.length > 0 && (
        <button
          type="button"
          tabIndex={-1}
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => setShow((s) => !s)}
          className="absolute right-0 top-0 flex h-9 w-9 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
        >
          {show ? (
            <EyeOff className="size-3.5" />
          ) : (
            <Eye className="size-3.5" />
          )}
        </button>
      )}
    </div>
  );
}

interface StepIndicatorProps {
  activeStep: number;
}

function StepIndicator({ activeStep }: StepIndicatorProps) {
  return (
    <div className="flex items-start justify-center">
      {SETUP_STEPS.map((s, i) => {
        const isDone = i < activeStep;
        const isActive = i === activeStep;

        return (
          <div key={s.title} className="flex items-start">
            <div className="flex flex-col items-center gap-1.5">
              <div
                className={cn(
                  'flex h-7 w-7 items-center justify-center rounded-full text-xs font-semibold transition-all duration-300',
                  isDone && 'bg-chart-2/15 text-chart-2',
                  isActive && 'bg-primary text-primary-foreground',
                  !isDone && !isActive && 'bg-muted text-muted-foreground',
                )}
              >
                {isDone ? (
                  <Check className="size-3.5" strokeWidth={2.5} />
                ) : (
                  i + 1
                )}
              </div>
              <span
                className={cn(
                  'whitespace-nowrap text-[11px] transition-colors duration-300',
                  isActive
                    ? 'font-medium text-foreground'
                    : 'text-muted-foreground',
                )}
              >
                {s.label}
              </span>
            </div>
            {i < SETUP_STEPS.length - 1 && (
              <div
                className={cn(
                  'mx-1 h-px w-8 self-start transition-colors duration-300',
                  isDone ? 'bg-chart-2/40' : 'bg-border',
                )}
                style={{ marginTop: '0.875rem' }}
              />
            )}
          </div>
        );
      })}
    </div>
  );
}

export function SetupScreen() {
  const [step, setStep] = useState(0);
  const [saving, setSaving] = useState(false);
  const [checkForUpdates, setCheckForUpdates] = useState<boolean | null>(null);
  const [githubToken, setGithubToken] = useState('');
  const [tokenState, setTokenState] = useState<'idle' | 'valid' | 'invalid'>(
    'idle',
  );

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
    const req = await fetch('https://api.github.com/rate_limit', {
      headers: { Authorization: `token ${githubToken.trim()}` },
    });
    if (req.status === 200) {
      setTokenState('valid');
      toast.success('GitHub token is valid!');
    } else {
      setTokenState('invalid');
      toast.error('GitHub token is invalid. Please check and try again.');
    }
  };

  const handleTokenChange = (e: ChangeEvent<HTMLInputElement>) => {
    setGithubToken(e.target.value);
    if (tokenState !== 'idle') setTokenState('idle');
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
      const trimmedToken = githubToken.trim();
      if (trimmedToken !== '') {
        await updateGithubToken(trimmedToken);
      }
      await completeSetup();
    } finally {
      setSaving(false);
    }
  };

  const stepState: SetupStepState = { validation, tokenState, checkForUpdates };
  const canProceed = SETUP_STEPS[step]!.canProceed(stepState);

  const stepData = SETUP_STEPS[step]!;
  const StepIcon = stepData.icon;

  return (
    <div className="fixed inset-0 z-50 flex flex-col items-center justify-center bg-background text-foreground">
      <div className="flex w-full max-w-md flex-col items-center gap-8 px-6">
        <div className="relative flex flex-col items-center gap-3 text-center">
          <div className="pointer-events-none absolute -top-10 left-1/2 h-36 w-full -translate-x-1/2 rounded-full blur-3xl bg-gradient-to-r from-transparent via-primary/35 to-transparent dark:via-primary/22" />
          <div className="relative z-10 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary/10 ring-1 ring-primary/20">
            <StepIcon className="size-7 text-primary" aria-hidden="true" />
          </div>
          <h1 className="relative z-10 text-3xl font-black tracking-tight">
            {stepData.title}
          </h1>
          <p className="relative z-10 max-w-xs text-sm text-muted-foreground">
            {stepData.description}
          </p>
        </div>

        <StepIndicator activeStep={step} />

        <Card className="w-full">
          <div className="flex flex-col gap-5 p-6">
            <div className="space-y-3">
              {step === 0 && (
                <>
                  <div className="flex gap-2">
                    <Button
                      className="flex-1"
                      onClick={() => handleDataFolder(true)}
                    >
                      <FolderSearch />
                      Auto-detect
                    </Button>
                    <Button
                      variant="outline"
                      className="flex-1 border-primary text-primary hover:bg-primary/10"
                      onClick={() => handleDataFolder(false)}
                    >
                      Browse...
                    </Button>
                  </div>
                  {config?.metroMakerDataPath && (
                    <PathDisplay
                      path={config.metroMakerDataPath}
                      valid={validation?.metroMakerDataPathValid ?? false}
                    />
                  )}
                </>
              )}

              {step === 1 && (
                <>
                  <div className="flex gap-2">
                    <Button
                      className="flex-1"
                      onClick={() => handleExecutable(true)}
                    >
                      <FolderSearch />
                      Auto-detect
                    </Button>
                    <Button
                      variant="outline"
                      className="flex-1 border-primary text-primary hover:bg-primary/10"
                      onClick={() => handleExecutable(false)}
                    >
                      Browse...
                    </Button>
                  </div>
                  {config?.executablePath && (
                    <PathDisplay
                      path={config.executablePath}
                      valid={validation?.executablePathValid ?? false}
                    />
                  )}
                </>
              )}

              {step === 2 && (
                <div className="space-y-3">
                  <div className="flex items-center gap-2">
                    <PasswordInput
                      className="flex-1"
                      value={githubToken}
                      onChange={handleTokenChange}
                      placeholder="github_pat_..."
                      invalid={tokenState === 'invalid'}
                    />
                    <Button
                      variant="outline"
                      size="sm"
                      className="shrink-0 focus-visible:ring-0 focus-visible:outline-none"
                      onClick={handleCheckToken}
                      disabled={githubToken.trim() === ''}
                    >
                      Check
                    </Button>
                  </div>
                  {tokenState !== 'idle' && (
                    <div
                      className={cn(
                        'flex items-center gap-2 rounded-lg border px-3 py-2.5',
                        tokenState === 'valid'
                          ? 'border-chart-2/30 bg-chart-2/5 text-chart-2'
                          : 'border-destructive/30 bg-destructive/5 text-destructive',
                      )}
                    >
                      {tokenState === 'valid' ? (
                        <CheckCircle className="size-3.5 shrink-0" />
                      ) : (
                        <XCircle className="size-3.5 shrink-0" />
                      )}
                      <p className="text-xs">
                        {tokenState === 'valid'
                          ? 'Token is valid.'
                          : 'Invalid token. Clear the field to skip, or enter a valid token.'}
                      </p>
                    </div>
                  )}
                </div>
              )}

              {step === 3 && (
                <div className="grid grid-cols-2 gap-3">
                  {(
                    [
                      {
                        value: true,
                        label: 'Yes',
                        sub: 'Keep me updated',
                        icon: RefreshCw,
                      },
                      {
                        value: false,
                        label: 'No',
                        sub: "I'll check manually",
                        icon: X,
                      },
                    ] as const
                  ).map(({ value, label, sub, icon: Icon }) => (
                    <button
                      key={String(value)}
                      type="button"
                      onClick={() => setCheckForUpdates(value)}
                      className={cn(
                        'flex flex-col items-center gap-2.5 rounded-xl border p-5 text-center transition-all duration-150 ease-out',
                        value === true
                          ? checkForUpdates === true
                            ? 'border-primary bg-primary/10 text-primary'
                            : 'border-border text-foreground hover:border-primary/40 hover:bg-primary/5'
                          : checkForUpdates === false
                            ? 'border-destructive bg-destructive/10 text-destructive'
                            : 'border-border text-foreground hover:border-destructive/40 hover:bg-destructive/5',
                      )}
                    >
                      <Icon className="size-5" />
                      <div>
                        <p className="text-sm font-semibold">{label}</p>
                        <p
                          className={cn(
                            'text-xs',
                            value === true && checkForUpdates === true
                              ? 'text-primary/70'
                              : value === false && checkForUpdates === false
                                ? 'text-destructive/70'
                                : 'text-muted-foreground',
                          )}
                        >
                          {sub}
                        </p>
                      </div>
                    </button>
                  ))}
                </div>
              )}
            </div>

            <div className="flex items-center justify-between">
              {step > 0 ? (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setStep((s) => s - 1)}
                  className="border-primary text-primary hover:bg-primary/10"
                >
                  Back
                </Button>
              ) : (
                <div />
              )}
              {step < SETUP_STEPS.length - 1 ? (
                <Button
                  onClick={() => setStep((s) => s + 1)}
                  disabled={!canProceed}
                >
                  {step === 2 && githubToken.trim() === '' ? 'Skip' : 'Next'}
                  <ChevronRight className="ml-1 size-4" />
                </Button>
              ) : (
                <Button onClick={handleFinish} disabled={!canProceed || saving}>
                  {saving && <Loader2 className="mr-2 size-4 animate-spin" />}
                  Finish Setup
                </Button>
              )}
            </div>
          </div>
        </Card>
      </div>
    </div>
  );
}
