import {
  AlertTriangle,
  ArrowDownToLine,
  Calendar,
  CheckCircle,
  Download,
  FileText,
  Loader2,
  Tag,
} from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';
import { Link } from 'wouter';

import { AssetActionDialog } from '@/components/dialogs/AssetActionDialog';
import { InstallErrorDialog } from '@/components/dialogs/InstallErrorDialog';
import { PrereleaseConfirmDialog } from '@/components/dialogs/PrereleaseConfirmDialog';
import { SubscriptionSyncErrorDialog } from '@/components/dialogs/SubscriptionSyncErrorDialog';
import { EmptyState } from '@/components/shared/EmptyState';
import { ErrorBanner } from '@/components/shared/ErrorBanner';
import { SortableHeaderCell } from '@/components/shared/SortableHeaderCell';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import semver from 'semver';

import type { AssetType } from '@/lib/asset-types';
import { assetTypeToListingPath } from '@/lib/asset-types';
import { getLocalAccentClasses } from '@/lib/local-accent';
import { isCompatible } from '@/lib/semver';
import {
  hasCancellationSyncErrors,
  hasOnlySilentSyncWarnings,
  isCancellationSyncError,
  toSubscriptionSyncErrorState,
} from '@/lib/subscription-sync-error';
import { cn } from '@/lib/utils';
import { useDownloadQueueStore } from '@/stores/download-queue-store';
import {
  AssetConflictError,
  useInstalledStore,
} from '@/stores/installed-store';

import type { types } from '../../../wailsjs/go/models';

type VersionSortField = 'version' | 'date' | 'downloads';
interface VersionSortState {
  field: VersionSortField;
  direction: 'asc' | 'desc';
}

const VERSION_TEXT_FIELDS = new Set<string>();

const DEFAULT_SORT: VersionSortState = { field: 'date', direction: 'desc' };

function sortVersions(
  versions: types.VersionInfo[],
  sort: VersionSortState,
): types.VersionInfo[] {
  return [...versions].sort((a, b) => {
    let cmp = 0;
    if (sort.field === 'date') {
      cmp = new Date(a.date).getTime() - new Date(b.date).getTime();
    } else if (sort.field === 'downloads') {
      cmp = a.downloads - b.downloads;
    } else {
      const av = semver.coerce(a.version);
      const bv = semver.coerce(b.version);
      cmp =
        av && bv
          ? semver.compare(av, bv)
          : a.version.localeCompare(b.version, undefined, { numeric: true });
    }
    return sort.direction === 'asc' ? cmp : -cmp;
  });
}

function formatDate(dateStr: string) {
  try {
    return new Date(dateStr).toLocaleDateString(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  } catch {
    return dateStr;
  }
}

interface ProjectVersionsProps {
  type: AssetType;
  itemId: string;
  itemName: string;
  versions: types.VersionInfo[];
  loading: boolean;
  error: string | null;
  gameVersion: string;
}

const INSTALL_ACCENT = getLocalAccentClasses('install');
const FILES_ACCENT = getLocalAccentClasses('files');

export function ProjectVersions({
  type,
  itemId,
  itemName,
  versions,
  loading,
  error,
  gameVersion,
}: ProjectVersionsProps) {
  const [sort, setSort] = useState<VersionSortState>(DEFAULT_SORT);
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

  const handleSort = (field: VersionSortField) => {
    setSort((prev) =>
      prev.field === field
        ? { field, direction: prev.direction === 'asc' ? 'desc' : 'asc' }
        : { field, direction: 'desc' },
    );
  };

  if (loading) {
    return (
      <div className="overflow-hidden rounded-xl border border-border bg-card">
        <div className="border-b border-border bg-muted/20 px-4 py-2">
          <Skeleton className="h-4 w-48" />
        </div>
        <div className="divide-y divide-border/50">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="flex items-center gap-3 px-4 py-3">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-4 w-20 ml-auto" />
              <Skeleton className="h-8 w-8 rounded-lg" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return <ErrorBanner message={error} />;
  }

  if (versions.length === 0) {
    return <EmptyState icon={FileText} title="No versions available" />;
  }

  const hasAnyGameVersion = versions.some((v) => v.game_version);
  const sorted = sortVersions(versions, sort);
  const typeListingPath = assetTypeToListingPath(type);

  return (
    <>
      <div className="overflow-hidden rounded-xl border border-border bg-card">
        <div className="flex items-center gap-4 border-b border-border bg-muted/20 px-4 py-2">
          <div className="flex-1 min-w-0">
            <SortableHeaderCell
              label="Version"
              field="version"
              icon={Tag}
              sort={sort}
              textFields={VERSION_TEXT_FIELDS}
              onSort={handleSort}
            />
          </div>
          <div className="w-[7rem] shrink-0 hidden sm:block">
            <SortableHeaderCell
              label="Date"
              field="date"
              icon={Calendar}
              sort={sort}
              textFields={VERSION_TEXT_FIELDS}
              onSort={handleSort}
            />
          </div>
          <div className="w-[6.5rem] shrink-0 hidden lg:block">
            <SortableHeaderCell
              label="Downloads"
              field="downloads"
              icon={ArrowDownToLine}
              sort={sort}
              textFields={VERSION_TEXT_FIELDS}
              onSort={handleSort}
            />
          </div>
          <div className="hidden lg:block w-px self-stretch bg-border/50 mx-2" aria-hidden />
          <div className="w-[7rem] shrink-0 flex items-center justify-center" aria-hidden />
        </div>

        <div className="divide-y divide-border/50">
          {sorted.map((v) => {
            const isThisInstalled = installedVersion === v.version;
            const installing = isInstalling(itemId);
            const installingVersion = getInstallingVersion(itemId);
            const uninstalling = isUninstalling(itemId);
            const compat = isCompatible(gameVersion, v.game_version);
            const incompatible = compat === false;

            return (
              <div
                key={v.version}
                className={cn(
                  'flex items-center gap-4 px-4 py-3 transition-colors hover:bg-muted/30',
                  incompatible && 'opacity-50',
                )}
              >
                <div className="flex-1 min-w-0">
                  <Link
                    href={`/project/${typeListingPath}/${itemId}/changelog/${encodeURIComponent(v.version)}`}
                    className="group inline-flex flex-col"
                  >
                    <span className="flex items-center gap-2">
                      <span className="text-sm font-semibold text-foreground group-hover:underline">
                        {v.version}
                      </span>
                      {v.prerelease && (
                        <Badge size="sm" className="border-amber-500/40 bg-amber-500/15 text-amber-600 dark:border-amber-400/40 dark:bg-amber-400/15 dark:text-amber-400">
                          Beta
                        </Badge>
                      )}
                    </span>
                    {v.name && v.name !== v.version && (
                      <span className="mt-0.5 text-xs text-muted-foreground truncate max-w-[20rem]">
                        {v.name}
                      </span>
                    )}
                  </Link>
                  {hasAnyGameVersion && v.game_version && (
                    <span className="mt-0.5 block text-xs text-muted-foreground">
                      {v.game_version}
                    </span>
                  )}
                </div>

                <div className="w-[7rem] shrink-0 hidden sm:block">
                  <span className="text-sm text-muted-foreground">
                    {formatDate(v.date)}
                  </span>
                </div>

                <div className="w-[6.5rem] shrink-0 hidden lg:flex items-center gap-1.5 text-sm text-muted-foreground">
                  <ArrowDownToLine className="h-3.5 w-3.5" />
                  {v.downloads.toLocaleString()}
                </div>

                <div className="hidden lg:block w-px self-stretch bg-border/50 mx-2" />
                <div className="w-[7rem] shrink-0 flex items-center justify-center">
                  {uninstalling ? (
                    <Button variant="outline" size="icon-xs" disabled>
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    </Button>
                  ) : installing ? (
                    installingVersion === v.version ? (
                      <Button variant="outline" size="icon-xs" disabled>
                        <Loader2 className="h-3.5 w-3.5 animate-spin" />
                      </Button>
                    ) : null
                  ) : isThisInstalled ? (
                    <Badge variant="success" size="sm" className="gap-1">
                      <CheckCircle className="h-2.5 w-2.5" />
                      Installed
                    </Badge>
                  ) : incompatible ? (
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span>
                            <Button
                              variant="outline"
                              size="icon-xs"
                              disabled
                              className={INSTALL_ACCENT.outlineButton}
                            >
                              <Download className="h-3.5 w-3.5" />
                            </Button>
                          </span>
                        </TooltipTrigger>
                        <TooltipContent>
                          Not compatible with your game version (you have{' '}
                          {gameVersion}, need {v.game_version})
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  ) : (
                    <Button
                      variant="outline"
                      size="icon-xs"
                      className={INSTALL_ACCENT.outlineButton}
                      onClick={() => handleInstall(v.version, v.prerelease)}
                    >
                      <Download className="h-3.5 w-3.5" />
                    </Button>
                  )}
                </div>
              </div>
            );
          })}
        </div>
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
            if (!open) setConflictState(null);
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
    </>
  );
}
