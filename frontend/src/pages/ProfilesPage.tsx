import {
  AlertTriangle,
  ArrowLeftRight,
  CircleUser,
  CircleUserRound,
  Loader2,
  Pencil,
  Plus,
  Trash2,
} from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';

import { AppDialog } from '@/components/dialogs/AppDialog';
import { PageHeading } from '@/components/shared/PageHeading';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useProfilesState } from '@/hooks/use-profile-dialogs';
import { getLocalAccentClasses } from '@/lib/local-accent';
import { isProfileSwapUnavailable } from '@/lib/profile-swap';
import { cn } from '@/lib/utils';
import { useGameStore } from '@/stores/game-store';
import { useInstalledStore } from '@/stores/installed-store';
import { useProfileStore } from '@/stores/profile-store';
import { useProfilesStore } from '@/stores/profiles-store';

import type { types } from '../../wailsjs/go/models';

const MAX_PROFILES = 5;
const DEFAULT_PROFILE_ID = '__default__';
const UPDATE_ACCENT = getLocalAccentClasses('update');
const FILES_ACCENT = getLocalAccentClasses('files');
const UNINSTALL_ACCENT = getLocalAccentClasses('uninstall');
const UI_THEME_DISPLAY_NAMES: Record<string, string> = {
  dark: 'Dark',
  light: 'Light',
};
const UI_SEARCH_VIEW_MODE_DISPLAY_NAMES: Record<string, string> = {
  card: 'Card',
  list: 'List',
};

function profileCounts(profile: types.UserProfile) {
  return {
    maps:
      Object.keys(profile.subscriptions?.maps ?? {}).length +
      Object.keys(profile.subscriptions?.localMaps ?? {}).length,
    mods: Object.keys(profile.subscriptions?.mods ?? {}).length,
  };
}

function sortProfilesForDisplay(profiles: types.UserProfile[]) {
  return [...profiles].sort((a, b) => {
    if (a.id === DEFAULT_PROFILE_ID) return -1;
    if (b.id === DEFAULT_PROFILE_ID) return 1;
    return a.name.localeCompare(b.name, undefined, { sensitivity: 'base' });
  });
}

function formatStorageSize(bytes: number | undefined): string {
  const safeBytes =
    typeof bytes === 'number' && Number.isFinite(bytes) && bytes > 0 ? bytes : 0;

  if (safeBytes < 1024) return `${safeBytes} B`;
  if (safeBytes < 1024 ** 2) return `${(safeBytes / 1024).toFixed(1)} KB`;
  if (safeBytes < 1024 ** 3) return `${(safeBytes / 1024 ** 2).toFixed(1)} MB`;
  return `${(safeBytes / 1024 ** 3).toFixed(1)} GB`;
}

function formatPreference(value: boolean | undefined): string {
  if (value === undefined) return 'Default';
  return value ? 'Enabled' : 'Disabled';
}

function humanizeKebabCase(value: string): string {
  return value
    .split('-')
    .map((part) =>
      part.length > 0 ? `${part[0].toUpperCase()}${part.slice(1)}` : part,
    )
    .join(' ');
}

function resolveDisplayName(
  value: string | undefined,
  displayNames: Record<string, string>,
  fallback: string,
): string {
  const normalized = value?.trim().toLowerCase();
  if (!normalized) return fallback;
  return displayNames[normalized] ?? humanizeKebabCase(normalized);
}

function ProfileStat({
  label,
  value,
}: {
  label: string;
  value: string | number;
}) {
  return (
    <div className="rounded-md border border-border/70 bg-muted/20 px-2 py-1">
      <p className="text-[11px] uppercase tracking-wide text-muted-foreground">
        {label}
      </p>
      <p className="font-semibold">{value}</p>
    </div>
  );
}

export function ProfilesPage() {
  const gameStore = useGameStore();
  const installedStore = useInstalledStore();
  const profileStore = useProfileStore();
  const profilesStore = useProfilesStore();
  const { dialogs, create, rename, remove, swap } = useProfilesState();
  const [expandedProfileID, setExpandedProfileID] = useState<string | null>(
    null,
  );

  const canCreate = profilesStore.profiles.length < MAX_PROFILES;

  useEffect(() => {
    profilesStore.loadProfiles().catch((err) => {
      toast.error(
        err instanceof Error ? err.message : 'Failed to load profiles',
      );
    });
  }, [profilesStore.loadProfiles]);

  useEffect(() => {
    setExpandedProfileID((current) => {
      if (!current) return current;
      return profilesStore.profiles.some((profile) => profile.id === current)
        ? current
        : null;
    });
  }, [profilesStore.profiles]);

  const sortedProfiles = useMemo(
    () => sortProfilesForDisplay(profilesStore.profiles),
    [profilesStore.profiles],
  );

  const refreshProfileContext = useCallback(async () => {
    await profileStore.refreshActiveProfile();
    await installedStore.updateInstalledLists();
    await profilesStore.loadProfiles();
  }, [
    installedStore.updateInstalledLists,
    profileStore.refreshActiveProfile,
    profilesStore,
  ]);

  const handleCreate = useCallback(async () => {
    const name = dialogs.create.name.trim();
    if (!name) return;
    if (!canCreate) {
      toast.warning(`Maximum of ${MAX_PROFILES} profiles reached`);
      return;
    }
    create.setLoading(true);
    try {
      const result = await profilesStore.createProfile(name);
      if (result.status !== 'success') {
        throw new Error(result.message || 'Failed to create profile');
      }
      create.reset();
      toast.success(`Created profile "${result.profile?.name ?? name}"`);
      await profilesStore.loadProfiles();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : 'Failed to create profile',
      );
    } finally {
      create.setLoading(false);
    }
  }, [canCreate, create, dialogs.create.name, profilesStore]);

  const handleDelete = useCallback(async () => {
    if (!dialogs.remove.target) return;
    remove.setLoading(true);
    try {
      const result = await profilesStore.deleteProfile(
        dialogs.remove.target.id,
      );
      if (result.status !== 'success') {
        throw new Error(result.message || 'Failed to delete profile');
      }
      toast.success(`Deleted profile "${dialogs.remove.target.name}"`);
      remove.close();
      await profilesStore.loadProfiles();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : 'Failed to delete profile',
      );
    } finally {
      remove.setLoading(false);
    }
  }, [dialogs.remove.target, profilesStore, remove]);

  const handleRename = useCallback(async () => {
    if (!dialogs.rename.target) return;
    const name = dialogs.rename.name.trim();
    if (!name) return;

    rename.setLoading(true);
    try {
      const result = await profilesStore.renameProfile(
        dialogs.rename.target.id,
        name,
      );
      if (result.status !== 'success') {
        throw new Error(result.message || 'Failed to rename profile');
      }
      toast.success(`Renamed profile to "${result.profile?.name ?? name}"`);
      rename.close();
      await profilesStore.loadProfiles();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : 'Failed to rename profile',
      );
    } finally {
      rename.setLoading(false);
    }
  }, [dialogs.rename.name, dialogs.rename.target, rename, profilesStore]);

  const hasArchiveConflictError = useCallback(
    (result: types.UserProfileResult) =>
      (result.errors ?? []).some(
        (item) =>
          item.errorType === 'archive_missing' ||
          item.errorType === 'archive_stale',
      ),
    [],
  );

  const handleSwap = useCallback(
    async (forceSwap: boolean) => {
      if (!dialogs.swap.target) return;
      if (gameStore.running) {
        toast.warning('Cannot switch profiles while the game is running.');
        return;
      }
      swap.setLoading(true);
      try {
        const result = await profilesStore.swapProfile(
          dialogs.swap.target.id,
          forceSwap,
        );

        if (result.status === 'success') {
          toast.success(`Switched to "${dialogs.swap.target.name}"`);
          swap.close();
          await refreshProfileContext();
          return;
        }

        if (result.status === 'warn') {
          if (!forceSwap && hasArchiveConflictError(result)) {
            swap.setArchiveWarningOpen(true);
            return;
          }
          toast.warning(result.message || 'Profile switched with warnings');
          swap.close();
          await refreshProfileContext();
          return;
        }

        if (result.profile?.id === dialogs.swap.target.id) {
          // Backend may have switched active profile before a restore/sync error.
          // Refresh UI state so swap controls reflect the new active profile.
          swap.close();
          await refreshProfileContext();
        }

        throw new Error(result.message || 'Failed to switch profile');
      } catch (err) {
        toast.error(
          err instanceof Error ? err.message : 'Failed to switch profile',
        );
      } finally {
        swap.setLoading(false);
      }
    },
    [
      dialogs.swap.target,
      hasArchiveConflictError,
      gameStore.running,
      refreshProfileContext,
      swap,
    ],
  );

  // Dialog for creating a new profile. Currently limited to just entering a name for the new profile, but this could be extended in the future to include preset options.
  const createDialogProps = {
    open: dialogs.create.open,
    onOpenChange: create.setOpen,
    icon: CircleUserRound,
    title: 'Create Profile',
    description: 'Enter a profile name.',
    tone: 'profiles' as const,
    confirm: {
      label: 'Create',
      onConfirm: () => void handleCreate(),
      loading: dialogs.create.loading,
    },
  };

  // Initial dialog for switching from the active profile to a different profile
  const switchDialogProps = {
    open: dialogs.swap.target !== null && !dialogs.swap.archiveWarningOpen,
    onOpenChange: (open: boolean) => {
      if (!open) {
        swap.close();
      }
    },
    icon: ArrowLeftRight,
    title: 'Switch Profile',
    description: dialogs.swap.target
      ? `Switch active profile to "${dialogs.swap.target.name}"?`
      : 'Switch active profile?',
    tone: 'update' as const,
    confirm: {
      label: 'Switch',
      onConfirm: () => void handleSwap(false),
      loading: dialogs.swap.loading,
    },
  };

  // Dialog for swap confirmation. When archives are missing or stale, the backend forces a confirmation step before switching profiles since a restore and/or sync may be required which could result in a long tail of download operations.
  const switchWarningDialogProps = {
    open: dialogs.swap.target !== null && dialogs.swap.archiveWarningOpen,
    onOpenChange: (open: boolean) => {
      if (!open) {
        swap.close();
      }
    },
    icon: AlertTriangle,
    title: 'Confirm Profile Switch',
    description: dialogs.swap.target
      ? `Profile "${dialogs.swap.target.name}" has a missing or stale archive and may require additional downloads during sync. Continue?`
      : 'Selected profile has a missing or stale archive. Continue?',
    tone: 'files' as const,
    confirm: {
      label: 'Continue',
      onConfirm: () => void handleSwap(true),
      loading: dialogs.swap.loading,
    },
  };

  // Dialog for renaming an existing profile. Other profile mutations are done indirectly either from the settings menu or from the library/listing pages.
  const renameDialogProps = {
    open: dialogs.rename.target !== null,
    onOpenChange: (open: boolean) => {
      if (!open) {
        rename.close();
      }
    },
    icon: Pencil,
    title: 'Rename Profile',
    description: dialogs.rename.target
      ? `Enter a new name for "${dialogs.rename.target.name}".`
      : 'Enter a new profile name.',
    tone: 'files' as const,
    confirm: {
      label: 'Rename',
      onConfirm: () => void handleRename(),
      loading: dialogs.rename.loading,
    },
  };

  // Dialog for confirming profile deletion, which requires explicit confirmation since it's a destructive action that cannot be undone.
  const deleteDialogProps = {
    open: dialogs.remove.target !== null,
    onOpenChange: (open: boolean) => {
      if (!open) {
        remove.close();
      }
    },
    icon: Trash2,
    title: 'Delete Profile',
    description: dialogs.remove.target
      ? `Delete profile "${dialogs.remove.target.name}"? This cannot be undone.`
      : 'Delete this profile?',
    tone: 'uninstall' as const,
    confirm: {
      label: 'Delete',
      onConfirm: () => void handleDelete(),
      loading: dialogs.remove.loading,
    },
  };

  return (
    <div className="space-y-6">
      <PageHeading
        icon={CircleUser}
        title="Profiles"
        description="Manage user profiles."
      />

      <div className="mx-auto max-w-4xl space-y-3">
        {profilesStore.loading ? (
          <div className="flex items-center justify-center rounded-xl border border-border bg-card p-8 text-muted-foreground">
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            Loading profiles...
          </div>
        ) : null}

        {!profilesStore.loading &&
          sortedProfiles.map((profile) => {
            const isActive = profile.id === profilesStore.activeProfileID;
            const isExpanded = expandedProfileID === profile.id;
            const counts = profileCounts(profile);
            const archiveSizeBytes = profilesStore.archiveSizes[profile.id];
            const subscriptionSizeBytes =
              profilesStore.subscriptionSizes[profile.id];
            const hasSubscriptions = counts.maps + counts.mods > 0;
            const showSubscriptionSize =
              isActive && subscriptionSizeBytes !== undefined;
            const showArchiveSize =
              !showSubscriptionSize &&
              hasSubscriptions &&
              archiveSizeBytes !== undefined;
            const sizeLabel = showSubscriptionSize
              ? 'Subscription Size'
              : 'Archive Size';
            const sizeValue = showSubscriptionSize
              ? formatStorageSize(subscriptionSizeBytes)
              : showArchiveSize
                ? formatStorageSize(archiveSizeBytes)
                : formatStorageSize(0);
            const swapUnavailable = isProfileSwapUnavailable({
              gameRunning: gameStore.running,
              targetIsActive: isActive,
              swapLoading: dialogs.swap.loading,
            });
            const mapSubscriptions = Object.keys(
              profile.subscriptions?.maps ?? {},
            ).length;
            const localMapSubscriptions = Object.keys(
              profile.subscriptions?.localMaps ?? {},
            ).length;
            const modSubscriptions = Object.keys(
              profile.subscriptions?.mods ?? {},
            ).length;
            const uiTheme = resolveDisplayName(
              profile.uiPreferences?.theme,
              UI_THEME_DISPLAY_NAMES,
              'System Default',
            );
            const uiSearchViewMode = resolveDisplayName(
              profile.uiPreferences?.searchViewMode,
              UI_SEARCH_VIEW_MODE_DISPLAY_NAMES,
              'Default',
            );
            const uiDefaultPerPage =
              profile.uiPreferences?.defaultPerPage !== undefined
                ? profile.uiPreferences.defaultPerPage
                : 'Default';

            return (
              <div
                key={profile.id}
                className={cn(
                  'overflow-hidden rounded-xl border bg-card transition-colors hover:bg-[color-mix(in_srgb,var(--profiles-primary)_12%,transparent)]',
                  isActive
                    ? 'border-[color-mix(in_srgb,var(--profiles-primary)_45%,transparent)]'
                    : 'border-border',
                )}
              >
                <button
                  type="button"
                  className="w-full px-4 py-4 text-left"
                  onClick={() =>
                    setExpandedProfileID((current) =>
                      current === profile.id ? null : profile.id,
                    )
                  }
                >
                  <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-start gap-4">
                    <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-xl border border-border bg-muted/30 text-[var(--profiles-primary)]">
                      <CircleUser className="h-8 w-8" />
                    </div>

                    <div className="min-w-0">
                      <div className="flex items-center gap-2">
                        <h3 className="truncate text-lg font-semibold">
                          {profile.name}
                        </h3>
                        {isActive ? (
                          <span className="rounded-full border border-[color-mix(in_srgb,var(--profiles-primary)_40%,transparent)] bg-[color-mix(in_srgb,var(--profiles-primary)_14%,transparent)] px-2 py-0.5 text-xs font-semibold text-[var(--profiles-primary)]">
                            Active
                          </span>
                        ) : null}
                      </div>
                      <p className="mt-0.5 font-mono text-xs text-muted-foreground">
                        {profile.uuid}
                      </p>
                    </div>

                    <div className="grid w-[7.75rem] shrink-0 grid-cols-3 justify-items-center gap-0.5">
                      {!isActive ? (
                        <Button
                          type="button"
                          size="icon"
                          variant="ghost"
                          className={cn(
                            'col-start-1 shrink-0',
                            UPDATE_ACCENT.iconButton,
                          )}
                          disabled={swapUnavailable}
                          onClick={(event) => {
                            event.stopPropagation();
                            swap.open(profile);
                          }}
                          aria-label={`Switch to profile ${profile.name}`}
                        >
                          <ArrowLeftRight className="h-4 w-4" />
                        </Button>
                      ) : (
                        <span className="size-10" aria-hidden="true" />
                      )}
                      <Button
                        type="button"
                        size="icon"
                        variant="ghost"
                        className={cn(
                          'col-start-2 shrink-0',
                          FILES_ACCENT.iconButton,
                        )}
                        onClick={(event) => {
                          event.stopPropagation();
                          rename.open(profile);
                        }}
                        aria-label={`Rename profile ${profile.name}`}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      {!isActive && profile.id !== DEFAULT_PROFILE_ID ? (
                        <Button
                          type="button"
                          size="icon"
                          variant="ghost"
                          className={cn(
                            'col-start-3 shrink-0',
                            UNINSTALL_ACCENT.iconButton,
                          )}
                          onClick={(event) => {
                            event.stopPropagation();
                            remove.open(profile);
                          }}
                          aria-label={`Delete profile ${profile.name}`}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      ) : (
                        <span className="size-10" aria-hidden="true" />
                      )}
                    </div>

                    <div className="col-start-2 col-end-3 mt-1 grid grid-cols-1 gap-2 text-sm sm:grid-cols-3">
                      <ProfileStat label="Maps" value={counts.maps} />
                      <ProfileStat label="Mods" value={counts.mods} />
                      <ProfileStat label={sizeLabel} value={sizeValue} />
                    </div>
                  </div>
                </button>

                {isExpanded ? (
                  <div className="grid gap-3 border-t border-border/70 bg-muted/10 px-4 py-3 md:grid-cols-3">
                    <section className="rounded-lg border border-border/70 bg-transparent p-3">
                      <h4 className="text-xs font-semibold uppercase tracking-wide text-[var(--profiles-primary)]">
                        Subscriptions
                      </h4>
                      <div className="mt-2 space-y-1.5 text-sm">
                        <div className="flex items-center justify-between">
                          <span className="text-muted-foreground">Maps</span>
                          <span className="font-semibold">
                            {mapSubscriptions}
                          </span>
                        </div>
                        <div className="flex items-center justify-between">
                          <span className="text-muted-foreground">
                            Local Maps
                          </span>
                          <span className="font-semibold">
                            {localMapSubscriptions}
                          </span>
                        </div>
                        <div className="flex items-center justify-between">
                          <span className="text-muted-foreground">Mods</span>
                          <span className="font-semibold">
                            {modSubscriptions}
                          </span>
                        </div>
                      </div>
                    </section>
                    <section className="rounded-lg border border-border/70 bg-transparent p-3">
                      <h4 className="text-xs font-semibold uppercase tracking-wide text-[var(--profiles-primary)]">
                        UI Preferences
                      </h4>
                      <div className="mt-2 space-y-1.5 text-sm">
                        <div className="flex items-center justify-between gap-3">
                          <span className="text-muted-foreground">Theme</span>
                          <span className="truncate font-semibold">
                            {uiTheme}
                          </span>
                        </div>
                        <div className="flex items-center justify-between gap-3">
                          <span className="text-muted-foreground">Per Page</span>
                          <span className="font-semibold">
                            {uiDefaultPerPage}
                          </span>
                        </div>
                        <div className="flex items-center justify-between gap-3">
                          <span className="text-muted-foreground">
                            Search View
                          </span>
                          <span className="truncate font-semibold">
                            {uiSearchViewMode}
                          </span>
                        </div>
                      </div>
                    </section>
                    <section className="rounded-lg border border-border/70 bg-transparent p-3">
                      <h4 className="text-xs font-semibold uppercase tracking-wide text-[var(--profiles-primary)]">
                        System Preferences
                      </h4>
                      <div className="mt-2 space-y-1.5 text-sm">
                        <div className="flex items-center justify-between gap-3">
                          <span className="text-muted-foreground">
                            Registry Refresh
                          </span>
                          <span className="font-semibold">
                            {formatPreference(
                              profile.systemPreferences
                                ?.refreshRegistryOnStartup,
                            )}
                          </span>
                        </div>
                        <div className="flex items-center justify-between gap-3">
                          <span className="text-muted-foreground">
                            Dev Tools
                          </span>
                          <span className="font-semibold">
                            {profile.systemPreferences?.useDevTools
                              ? 'Enabled'
                              : 'Disabled'}
                          </span>
                        </div>
                      </div>
                    </section>
                  </div>
                ) : null}
              </div>
            );
          })}

        {canCreate ? (
          <button
            type="button"
            className="flex w-full items-center justify-center gap-2 rounded-xl border border-dashed border-[color-mix(in_srgb,var(--profiles-primary)_45%,transparent)] bg-[color-mix(in_srgb,var(--profiles-primary)_10%,transparent)] px-4 py-6 text-lg font-semibold text-[var(--profiles-primary)] transition-colors hover:bg-[color-mix(in_srgb,var(--profiles-primary)_16%,transparent)]"
            onClick={create.open}
          >
            <Plus className="h-6 w-6" />
            Create Profile
          </button>
        ) : (
          <div className="rounded-xl border border-border bg-card px-4 py-3 text-sm text-muted-foreground">
            Maximum of {MAX_PROFILES} profiles reached.
          </div>
        )}
      </div>

      <AppDialog {...createDialogProps}>
        <div className="space-y-2">
          <label htmlFor="profile-name" className="text-sm font-medium">
            Profile Name
          </label>
          <Input
            id="profile-name"
            value={dialogs.create.name}
            onChange={(event) => create.setName(event.target.value)}
            placeholder="New profile"
            autoFocus
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault();
                void handleCreate();
              }
            }}
          />
        </div>
      </AppDialog>

      <AppDialog {...switchDialogProps} />

      <AppDialog {...switchWarningDialogProps}>
        <div
          className={`rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground ${FILES_ACCENT.dialogPanel}`}
        >
          <p className="font-medium text-foreground">
            Target Profile: {dialogs.swap.target?.name ?? ''}
          </p>
          <p className="mt-1 font-mono">
            UUID: {dialogs.swap.target?.uuid ?? ''}
          </p>
        </div>
      </AppDialog>

      <AppDialog {...renameDialogProps}>
        <div className="space-y-2">
          <label htmlFor="rename-profile-name" className="text-sm font-medium">
            Profile Name
          </label>
          <Input
            id="rename-profile-name"
            value={dialogs.rename.name}
            onChange={(event) => rename.setName(event.target.value)}
            placeholder="Profile name"
            autoFocus
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault();
                void handleRename();
              }
            }}
          />
        </div>
      </AppDialog>

      <AppDialog {...deleteDialogProps} />
    </div>
  );
}
