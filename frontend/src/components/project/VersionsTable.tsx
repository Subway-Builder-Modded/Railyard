import {
  AlertTriangle,
  ArrowDownToLine,
  CheckCircle,
  Download,
  FileText,
  Loader2,
} from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';

import { AssetActionDialog } from '@/components/dialogs/AssetActionDialog';
import { InstallErrorDialog } from '@/components/dialogs/InstallErrorDialog';
import { PrereleaseConfirmDialog } from '@/components/dialogs/PrereleaseConfirmDialog';
import { SubscriptionSyncErrorDialog } from '@/components/dialogs/SubscriptionSyncErrorDialog';
import { EmptyState } from '@/components/shared/EmptyState';
import { ErrorBanner } from '@/components/shared/ErrorBanner';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import type { AssetType } from '@/lib/asset-types';
import { getLocalAccentClasses } from '@/lib/local-accent';
import { isCompatible } from '@/lib/semver';
import {
  hasCancellationSyncErrors,
  hasOnlySilentSyncWarnings,
  isCancellationSyncError,
  toSubscriptionSyncErrorState,
} from '@/lib/subscription-sync-error';
import { useDownloadQueueStore } from '@/stores/download-queue-store';
import {
  AssetConflictError,
  useInstalledStore,
} from '@/stores/installed-store';

import type { types } from '../../../wailsjs/go/models';

interface VersionsTableProps {
  type: AssetType;
  itemId: string;
  itemName: string;
  update: types.UpdateConfig;
  versions: types.VersionInfo[];
  loading: boolean;
  error: string | null;
  gameVersion: string;
}

const INSTALL_ACCENT = getLocalAccentClasses('install');
const FILES_ACCENT = getLocalAccentClasses('files');

export function VersionsTable({
  type,
  itemId,
  itemName,
  versions,
  loading,
  error,
  gameVersion,
}: VersionsTableProps) {
  const {
    getInstalledVersion,
    installMod,
    installMap,
    isInstalling,
    getInstallingVersion,
    isUninstalling,
  } = useInstalledStore();
  const cancellationToastId = `cancel-install-${type}-${itemId}`;
  const installedVersion = getInstalledVersion(itemId);
  const [installError, setInstallError] = useState<{
    version: string;
    message: string;
  } | null>(null);
  const [prereleasePrompt, setPrereleasePrompt] = useState<{
    version: string;
  } | null>(null);
  const [subscriptionSyncError, setSubscriptionSyncError] = useState<{
    version: string;
    message: string;
    errors: types.UserProfilesError[];
  } | null>(null);
  const [conflictState, setConflictState] = useState<{
    version: string;
    conflict: types.MapCodeConflict;
  } | null>(null);

  const doInstall = async (version: string, replaceOnConflict = false) => {
    try {
      let result: types.UpdateSubscriptionsResult;
      if (type === 'mod') {
        result = await installMod(itemId, version);
      } else {
        result = await installMap(itemId, version, replaceOnConflict);
      }
      if (result.status === 'warn') {
        if (hasCancellationSyncErrors(result.errors)) {
          toast.success(`Cancelled pending install for ${itemName}.`, {
            id: cancellationToastId,
          });
        } else if (!hasOnlySilentSyncWarnings(result.errors)) {
          toast.warning(
            result.message ||
              `Install for ${itemName} completed with warnings.`,
          );
        } else {
          // Suppress expected stale-sync warnings from burst queue updates.
        }
        return;
      }
      const { completed, total } = useDownloadQueueStore.getState();
      const queueText = total > 1 ? ` (${completed}/${total} Downloaded)` : '';
      toast.success(`Installed ${version} successfully.${queueText}`);
    } catch (err) {
      if (err instanceof AssetConflictError && err.conflicts.length > 0) {
        setConflictState({ version, conflict: err.conflicts[0] });
        return;
      }

      const syncError = toSubscriptionSyncErrorState(err, version);
      if (syncError) {
        if (
          useInstalledStore.getState().isUninstalling(itemId) ||
          isCancellationSyncError(syncError)
        ) {
          toast.success(`Cancelled pending install for ${itemName}.`, {
            id: cancellationToastId,
          });
          return;
        }
        setSubscriptionSyncError(syncError);
      } else {
        setInstallError({
          version,
          message: err instanceof Error ? err.message : String(err),
        });
      }
    }
  };

  const handleInstall = (version: string, prerelease: boolean) => {
    if (prerelease) {
      setPrereleasePrompt({ version });
    } else {
      doInstall(version);
    }
  };

  if (loading) {
    return (
      <div className="space-y-3">
        <h2 className="text-xl font-semibold">Versions</h2>
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-3">
        <h2 className="text-xl font-semibold">Versions</h2>
        <ErrorBanner message={error} />
      </div>
    );
  }

  if (versions.length === 0) {
    return (
      <div className="space-y-3">
        <h2 className="text-xl font-semibold">Versions</h2>
        <EmptyState icon={FileText} title="No versions available" />
      </div>
    );
  }

  const formatDate = (dateStr: string) => {
    try {
      return new Date(dateStr).toLocaleDateString(undefined, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      });
    } catch {
      return dateStr;
    }
  };

  const hasAnyGameVersion = versions.some((v) => v.game_version);

  return (
    <div className="space-y-3">
      <h2 className="text-xl font-semibold">Versions</h2>
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Version</TableHead>
              <TableHead>Date</TableHead>
              {hasAnyGameVersion && <TableHead>Game Version</TableHead>}
              <TableHead>Changelog</TableHead>
              <TableHead>Downloads</TableHead>
              <TableHead className="w-25"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {versions.map((v) => {
              const isThisInstalled = installedVersion === v.version;
              const installing = isInstalling(itemId);
              const installingVersion = getInstallingVersion(itemId);
              const uninstalling = isUninstalling(itemId);
              const compat = isCompatible(gameVersion, v.game_version);
              const incompatible = compat === false;

              return (
                <TableRow
                  key={v.version}
                  className={incompatible ? 'opacity-50' : ''}
                >
                  <TableCell className="font-mono font-medium">
                    <span className="flex items-center gap-2">
                      {v.version}
                      {v.prerelease && (
                        <Badge variant="outline" size="sm">
                          Beta
                        </Badge>
                      )}
                    </span>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {formatDate(v.date)}
                  </TableCell>
                  {hasAnyGameVersion && (
                    <TableCell className="text-muted-foreground font-mono text-xs">
                      {v.game_version || '\u2014'}
                    </TableCell>
                  )}
                  <TableCell className="text-sm text-muted-foreground max-w-xs truncate">
                    {v.changelog}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    <div className="flex items-center gap-1.5">
                      <ArrowDownToLine className="h-3 w-3" />
                      {v.downloads.toLocaleString()}
                    </div>
                  </TableCell>
                  <TableCell>
                    {uninstalling ? (
                      <Button variant="outline" size="sm" disabled>
                        <Loader2 className="h-4 w-4 animate-spin" />
                      </Button>
                    ) : installing ? (
                      installingVersion === v.version ? (
                        <Button variant="outline" size="sm" disabled>
                          <Loader2 className="h-4 w-4 animate-spin" />
                        </Button>
                      ) : (
                        <span className="inline-flex h-8 w-8" />
                      )
                    ) : isThisInstalled ? (
                      <Badge variant="success" className="gap-1">
                        <CheckCircle className="h-3 w-3" />
                        Installed
                      </Badge>
                    ) : incompatible ? (
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span>
                              <Button
                                variant="outline"
                                size="sm"
                                disabled
                                className={INSTALL_ACCENT.outlineButton}
                              >
                                <Download className="h-4 w-4" />
                              </Button>
                            </span>
                          </TooltipTrigger>
                          <TooltipContent>
                            Not compatible with your installed game version (you
                            have {gameVersion}, need {v.game_version})
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    ) : (
                      <Button
                        variant="outline"
                        size="sm"
                        className={INSTALL_ACCENT.outlineButton}
                        onClick={() => handleInstall(v.version, v.prerelease)}
                      >
                        <Download className="h-4 w-4" />
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>

      {prereleasePrompt && (
        <PrereleaseConfirmDialog
          open={!!prereleasePrompt}
          onOpenChange={(open) => {
            if (!open) setPrereleasePrompt(null);
          }}
          itemName={itemName}
          version={prereleasePrompt.version}
          onConfirm={() => doInstall(prereleasePrompt.version)}
        />
      )}

      {installError && (
        <InstallErrorDialog
          open={!!installError}
          onOpenChange={(open) => {
            if (!open) setInstallError(null);
          }}
          itemName={itemName}
          version={installError.version}
          error={installError.message}
        />
      )}

      {subscriptionSyncError && (
        <SubscriptionSyncErrorDialog
          open={!!subscriptionSyncError}
          onOpenChange={(open) => {
            if (!open) setSubscriptionSyncError(null);
          }}
          itemName={itemName}
          version={subscriptionSyncError.version}
          message={subscriptionSyncError.message}
          errors={subscriptionSyncError.errors}
        />
      )}

      {conflictState && (
        <AssetActionDialog
          open={!!conflictState}
          onOpenChange={(open) => {
            if (!open) {
              setConflictState(null);
            }
          }}
          loading={false}
          icon={AlertTriangle}
          iconClassName="h-5 w-5 text-[var(--files-primary)]"
          title={`Replace conflicting map for ${itemName}?`}
          description={`Installing ${itemName} ${conflictState.version} conflicts with an existing map. Replace the existing map to continue.`}
          conflict={conflictState.conflict}
          confirmLabel="Replace"
          confirmClassName={FILES_ACCENT.solidButton}
          tone="files"
          onConfirm={() => {
            const version = conflictState.version;
            setConflictState(null);
            void doInstall(version, true);
          }}
        />
      )}
    </div>
  );
}
