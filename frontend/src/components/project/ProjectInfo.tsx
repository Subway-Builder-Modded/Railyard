import {
  AlertTriangle,
  CheckCircle,
  CircleFadingArrowUp,
  Download,
  ExternalLink,
  Globe,
  Loader2,
  MapPin,
  Trash2,
  Users,
  X,
} from 'lucide-react';
import { useState } from 'react';
import Markdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import { toast } from 'sonner';

import { AssetActionDialog } from '@/components/dialogs/AssetActionDialog';
import { InstallErrorDialog } from '@/components/dialogs/InstallErrorDialog';
import { PrereleaseConfirmDialog } from '@/components/dialogs/PrereleaseConfirmDialog';
import { SubscriptionSyncErrorDialog } from '@/components/dialogs/SubscriptionSyncErrorDialog';
import { UninstallDialog } from '@/components/dialogs/UninstallDialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import type { AssetType } from '@/lib/asset-types';
import { getLocalAccentClasses } from '@/lib/local-accent';
import { formatSourceQuality } from '@/lib/map-filter-values';
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
import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime';

interface ProjectInfoProps {
  type: AssetType;
  item: types.ModManifest | types.MapManifest;
  latestVersion?: types.VersionInfo;
  latestCompatibleVersion?: types.VersionInfo;
  versionsLoading: boolean;
  gameVersion: string;
}

const INSTALL_ACCENT = getLocalAccentClasses('install');
const UPDATE_ACCENT = getLocalAccentClasses('update');
const FILES_ACCENT = getLocalAccentClasses('files');

function isMapManifest(
  item: types.ModManifest | types.MapManifest,
): item is types.MapManifest {
  return 'city_code' in item;
}

export function ProjectInfo({
  type,
  item,
  latestVersion,
  latestCompatibleVersion,
  versionsLoading,
  gameVersion,
}: ProjectInfoProps) {
  const cancellationToastId = `cancel-install-${type}-${item.id}`;
  const [uninstallOpen, setUninstallOpen] = useState(false);
  const [installError, setInstallError] = useState<{
    version: string;
    message: string;
  } | null>(null);
  const [prereleasePrompt, setPrereleasePrompt] = useState(false);
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
    installMod,
    installMap,
    cancelPendingInstall,
    getInstalledVersion,
    isInstalling,
    isUninstalling,
  } = useInstalledStore();

  const installedVersion = getInstalledVersion(item.id);
  const installing = isInstalling(item.id);
  const uninstalling = isUninstalling(item.id);
  const detailBadges = isMapManifest(item)
    ? [
        item.location,
        formatSourceQuality(item.source_quality),
        item.level_of_detail,
        ...(item.special_demand ?? []),
      ].filter((value): value is string => Boolean(value))
    : (item.tags ?? []);
  // Use the latest compatible version for install/update buttons
  const effectiveVersion = latestCompatibleVersion ?? latestVersion;
  const hasUpdate =
    installedVersion &&
    effectiveVersion &&
    installedVersion !== effectiveVersion.version;
  // No compatible version exists at all
  const noCompatibleVersion =
    gameVersion && latestVersion && !latestCompatibleVersion;

  const handleInstall = async (version: string, replaceOnConflict = false) => {
    try {
      let result: types.UpdateSubscriptionsResult;
      if (type === 'mod') {
        result = await installMod(item.id, version);
      } else {
        result = await installMap(item.id, version, replaceOnConflict);
      }
      if (result.status === 'warn') {
        if (hasCancellationSyncErrors(result.errors)) {
          toast.success(`Cancelled pending install for ${item.name}.`, {
            id: cancellationToastId,
          });
        } else if (!hasOnlySilentSyncWarnings(result.errors)) {
          toast.warning(
            result.message ||
              `Install for ${item.name} completed with warnings.`,
          );
        } else {
          // Suppress expected stale-sync warnings from burst queue updates.
        }
        return;
      }
      const { completed, total } = useDownloadQueueStore.getState();
      const queueText = total > 1 ? ` (${completed}/${total} Downloaded)` : '';
      toast.success(
        `${item.name} ${version} installed successfully.${queueText}`,
      );
    } catch (err) {
      if (err instanceof AssetConflictError && err.conflicts.length > 0) {
        setConflictState({ version, conflict: err.conflicts[0] });
        return;
      }

      const syncError = toSubscriptionSyncErrorState(err, version);
      if (syncError) {
        if (
          useInstalledStore.getState().isUninstalling(item.id) ||
          isCancellationSyncError(syncError)
        ) {
          toast.success(`Cancelled pending install for ${item.name}.`, {
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

  const handleCancelInstall = async () => {
    try {
      await cancelPendingInstall(type, item.id);
      toast.success(`Cancelled pending install for ${item.name}.`, {
        id: cancellationToastId,
      });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  };

  const handleInstallClick = (version: string, prerelease?: boolean) => {
    if (prerelease) {
      setPrereleasePrompt(true);
    } else {
      handleInstall(version);
    }
  };

  const renderInstallButton = (v: types.VersionInfo, label: string) => {
    const isUpdate = label.toLowerCase().includes('update');
    if (noCompatibleVersion) {
      return (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <span>
                <Button
                  size="sm"
                  disabled
                  className={INSTALL_ACCENT.solidButton}
                >
                  <Download className="h-4 w-4 mr-1.5" />
                  {label}
                </Button>
              </span>
            </TooltipTrigger>
            <TooltipContent>
              No version compatible with your installed game version (
              {gameVersion})
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      );
    }
    return (
      <>
        <Button
          size="sm"
          className={
            isUpdate ? UPDATE_ACCENT.solidButton : INSTALL_ACCENT.solidButton
          }
          onClick={() => handleInstallClick(v.version, v.prerelease)}
        >
          {isUpdate ? (
            <CircleFadingArrowUp className="h-4 w-4 mr-1.5" />
          ) : (
            <Download className="h-4 w-4 mr-1.5" />
          )}
          {label}
        </Button>
        {isUpdate && (
          <Button
            variant="destructive"
            size="icon"
            className="h-8 w-8"
            onClick={() => setUninstallOpen(true)}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        )}
      </>
    );
  };

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{item.name}</h1>
          <p className="text-muted-foreground mt-1">by {item.author}</p>
        </div>

        <div className="flex items-center gap-2 shrink-0">
          {versionsLoading ? (
            <Button size="sm" disabled>
              <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
              Loading...
            </Button>
          ) : uninstalling ? (
            <Button size="sm" disabled>
              <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
              Canceling...
            </Button>
          ) : installing ? (
            <Button size="sm" variant="outline" onClick={handleCancelInstall}>
              <X className="h-4 w-4 mr-1.5" />
              Cancel Install
            </Button>
          ) : !installedVersion && effectiveVersion ? (
            renderInstallButton(
              effectiveVersion,
              `Install ${effectiveVersion.version}`,
            )
          ) : hasUpdate && effectiveVersion ? (
            renderInstallButton(
              effectiveVersion,
              `Update to ${effectiveVersion.version}`,
            )
          ) : installedVersion ? (
            <>
              <Badge variant="success" className="gap-1">
                <CheckCircle className="h-3 w-3" />
                Installed {installedVersion}
              </Badge>
              <Button
                variant="destructive"
                size="icon"
                className="h-8 w-8"
                onClick={() => setUninstallOpen(true)}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </>
          ) : null}
        </div>
      </div>

      {isMapManifest(item) && (
        <div className="flex items-center gap-4 text-sm">
          {item.city_code && (
            <div className="flex items-center gap-1.5">
              <MapPin className="h-4 w-4 text-muted-foreground" />
              <span className="font-mono font-bold">{item.city_code}</span>
              {item.country && (
                <span className="text-muted-foreground">{item.country}</span>
              )}
            </div>
          )}
          {item.population > 0 && (
            <div className="flex items-center gap-1.5">
              <Users className="h-4 w-4 text-muted-foreground" />
              <span>{item.population.toLocaleString()}</span>
            </div>
          )}
        </div>
      )}

      <Separator />

      <div className="text-sm leading-relaxed prose prose-sm prose-neutral dark:prose-invert max-w-none">
        <Markdown
          rehypePlugins={[rehypeRaw]}
          components={{
            a: ({ href, children, ...props }) => (
              <a
                {...props}
                href={href}
                onClick={(e) => {
                  if (href) {
                    e.preventDefault();
                    BrowserOpenURL(href);
                  }
                }}
              >
                {children}
              </a>
            ),
          }}
        >
          {item.description}
        </Markdown>
      </div>

      {detailBadges.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {detailBadges.map((badge) => (
            <Badge key={badge} variant="secondary">
              {badge}
            </Badge>
          ))}
        </div>
      )}

      {item.source && (
        <Button
          variant="outline"
          size="sm"
          onClick={() => BrowserOpenURL(item.source!)}
        >
          <Globe className="h-4 w-4 mr-1.5" />
          View Source
          <ExternalLink className="h-3 w-3 ml-1.5" />
        </Button>
      )}

      <UninstallDialog
        open={uninstallOpen}
        onOpenChange={setUninstallOpen}
        type={type}
        id={item.id}
        name={item.name}
      />

      {prereleasePrompt && effectiveVersion && (
        <PrereleaseConfirmDialog
          open={prereleasePrompt}
          onOpenChange={(open) => {
            if (!open) setPrereleasePrompt(false);
          }}
          itemName={item.name}
          version={effectiveVersion.version}
          onConfirm={() => handleInstall(effectiveVersion.version)}
        />
      )}

      {installError && (
        <InstallErrorDialog
          open={!!installError}
          onOpenChange={(open) => {
            if (!open) setInstallError(null);
          }}
          itemName={item.name}
          version={installError.version}
          error={installError.message}
        />
      )}

      {subscriptionSyncError && (
        <SubscriptionSyncErrorDialog
          open={!!subscriptionSyncError}
          onOpenChange={(open) => {
            if (!open) {
              setSubscriptionSyncError(null);
            }
          }}
          itemName={item.name}
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
          title={`Replace conflicting map for ${item.name}?`}
          description={`Installing ${item.name} ${conflictState.version} conflicts with an existing map. Replace the existing map to continue.`}
          conflict={conflictState.conflict}
          confirmLabel="Replace"
          confirmClassName={FILES_ACCENT.solidButton}
          tone="files"
          onConfirm={() => {
            const version = conflictState.version;
            setConflictState(null);
            void handleInstall(version, true);
          }}
        />
      )}
    </div>
  );
}
