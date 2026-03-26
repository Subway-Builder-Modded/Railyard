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
import { getLocalAccentClasses } from '@/lib/local-accent';
import { isProfileSwapUnavailable } from '@/lib/profile-swap';
import { cn } from '@/lib/utils';
import { useGameStore } from '@/stores/game-store';
import { useProfileStore } from '@/stores/profile-store';

import { types } from '../../wailsjs/go/models';
import {
  CreateProfile,
  DeleteProfile,
  ListProfiles,
  RenameProfile,
  SwapProfile,
} from '../../wailsjs/go/profiles/UserProfiles';

const MAX_PROFILES = 5;
const UPDATE_ACCENT = getLocalAccentClasses('update');
const FILES_ACCENT = getLocalAccentClasses('files');

function profileCounts(profile: types.UserProfile) {
  return {
    maps: Object.keys(profile.subscriptions?.maps ?? {}).length,
    mods: Object.keys(profile.subscriptions?.mods ?? {}).length,
  };
}

export function ProfilesPage() {
  const gameRunning = useGameStore((s) => s.running);
  const refreshActiveProfile = useProfileStore((s) => s.refreshActiveProfile);
  const [profiles, setProfiles] = useState<types.UserProfile[]>([]);
  const [archiveSizes, setArchiveSizes] = useState<Record<string, number>>({});
  const [activeProfileID, setActiveProfileID] = useState('');
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [createName, setCreateName] = useState('');
  const [createLoading, setCreateLoading] = useState(false);
  const [renameTarget, setRenameTarget] = useState<types.UserProfile | null>(
    null,
  );
  const [renameName, setRenameName] = useState('');
  const [renameLoading, setRenameLoading] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<types.UserProfile | null>(
    null,
  );
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [swapTarget, setSwapTarget] = useState<types.UserProfile | null>(null);
  const [swapLoading, setSwapLoading] = useState(false);
  const [swapArchiveWarningOpen, setSwapArchiveWarningOpen] = useState(false);
  const [expandedProfileID, setExpandedProfileID] = useState<string | null>(
    null,
  );

  const canCreate = profiles.length < MAX_PROFILES;

  const loadProfiles = useCallback(async () => {
    setLoading(true);
    try {
      const result = await ListProfiles();
      if (result.status !== 'success') {
        throw new Error(result.message || 'Failed to load profiles');
      }
      setProfiles(result.profiles ?? []);
      setArchiveSizes(result.archiveSizes ?? {});
      setActiveProfileID(result.activeProfileId ?? '');
      setExpandedProfileID((current) => {
        if (!current) return current;
        return (result.profiles ?? []).some((profile) => profile.id === current)
          ? current
          : null;
      });
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : 'Failed to load profiles',
      );
      setProfiles([]);
      setArchiveSizes({});
      setActiveProfileID('');
      setExpandedProfileID(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadProfiles();
  }, [loadProfiles]);

  const sortedProfiles = useMemo(() => profiles, [profiles]);

  const handleCreate = useCallback(async () => {
    const name = createName.trim();
    if (!name) return;
    if (!canCreate) {
      toast.warning(`Maximum of ${MAX_PROFILES} profiles reached`);
      return;
    }
    setCreateLoading(true);
    try {
      const result = await CreateProfile(
        new types.CreateProfileRequest({ name }),
      );
      if (result.status !== 'success') {
        throw new Error(result.message || 'Failed to create profile');
      }
      setCreateOpen(false);
      setCreateName('');
      toast.success(`Created profile "${result.profile?.name ?? name}"`);
      await loadProfiles();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to create profile');
    } finally {
      setCreateLoading(false);
    }
  }, [canCreate, createName, loadProfiles]);

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return;
    setDeleteLoading(true);
    try {
      const result = await DeleteProfile(deleteTarget.id);
      if (result.status !== 'success') {
        throw new Error(result.message || 'Failed to delete profile');
      }
      toast.success(`Deleted profile "${deleteTarget.name}"`);
      setDeleteTarget(null);
      await loadProfiles();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to delete profile');
    } finally {
      setDeleteLoading(false);
    }
  }, [deleteTarget, loadProfiles]);

  const handleRename = useCallback(async () => {
    if (!renameTarget) return;
    const name = renameName.trim();
    if (!name) return;

    setRenameLoading(true);
    try {
      const result = await RenameProfile(renameTarget.id, name);
      if (result.status !== 'success') {
        throw new Error(result.message || 'Failed to rename profile');
      }
      toast.success(`Renamed profile to "${result.profile?.name ?? name}"`);
      setRenameTarget(null);
      setRenameName('');
      await loadProfiles();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to rename profile');
    } finally {
      setRenameLoading(false);
    }
  }, [renameName, renameTarget, loadProfiles]);

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
      if (!swapTarget) return;
      if (gameRunning) {
        toast.warning('Cannot switch profiles while the game is running.');
        return;
      }
      setSwapLoading(true);
      try {
        const result = await SwapProfile(
          new types.SwapProfileRequest({
            profileId: swapTarget.id,
            forceSwap,
          }),
        );

        if (result.status === 'success') {
          toast.success(`Switched to "${swapTarget.name}"`);
          setSwapTarget(null);
          setSwapArchiveWarningOpen(false);
          await refreshActiveProfile();
          await loadProfiles();
          return;
        }

        if (result.status === 'warn') {
          if (!forceSwap && hasArchiveConflictError(result)) {
            setSwapArchiveWarningOpen(true);
            return;
          }
          toast.warning(result.message || 'Profile switched with warnings');
          setSwapTarget(null);
          setSwapArchiveWarningOpen(false);
          await refreshActiveProfile();
          await loadProfiles();
          return;
        }

        if (result.profile?.id === swapTarget.id) {
          // Backend may have switched active profile before a restore/sync error.
          // Refresh UI state so swap controls reflect the new active profile.
          setSwapTarget(null);
          setSwapArchiveWarningOpen(false);
          await refreshActiveProfile();
          await loadProfiles();
        }

        throw new Error(result.message || 'Failed to switch profile');
      } catch (err) {
        toast.error(
          err instanceof Error ? err.message : 'Failed to switch profile',
        );
      } finally {
        setSwapLoading(false);
      }
    },
    [
      swapTarget,
      loadProfiles,
      hasArchiveConflictError,
      gameRunning,
      refreshActiveProfile,
    ],
  );

  return (
    <div className="space-y-6">
      <PageHeading
        icon={CircleUser}
        title="Profiles"
        description="Manage user profiles and subscriptions."
      />

      <div className="mx-auto max-w-4xl space-y-3">
        {loading ? (
          <div className="flex items-center justify-center rounded-xl border border-border bg-card p-8 text-muted-foreground">
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            Loading profiles...
          </div>
        ) : null}

        {!loading &&
          sortedProfiles.map((profile) => {
            const isActive = profile.id === activeProfileID;
            const isExpanded = expandedProfileID === profile.id;
            const counts = profileCounts(profile);
            const archiveSizeBytes = archiveSizes[profile.id];
            const archiveSizeDisplay =
              archiveSizeBytes === undefined
                ? 'Unknown'
                : `${(archiveSizeBytes / (1024 * 1024)).toFixed(2)} MB`;
            const swapUnavailable = isProfileSwapUnavailable({
              gameRunning,
              targetIsActive: isActive,
              swapLoading,
            });

            return (
              <div
                key={profile.id}
                className={cn(
                  'overflow-hidden rounded-xl border bg-card transition-colors',
                  isActive
                    ? 'border-[color-mix(in_srgb,var(--profiles-primary)_45%,transparent)]'
                    : 'border-border',
                )}
              >
                <button
                  type="button"
                  className="w-full px-4 py-4 text-left hover:bg-accent/30"
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

                    <div className="flex w-24 shrink-0 items-center justify-end gap-1">
                      {!isActive ? (
                        <Button
                          type="button"
                          size="icon"
                          variant="ghost"
                          className={cn('shrink-0', UPDATE_ACCENT.iconButton)}
                          disabled={swapUnavailable}
                          onClick={(event) => {
                            event.stopPropagation();
                            setSwapTarget(profile);
                            setSwapArchiveWarningOpen(false);
                          }}
                          aria-label={`Switch to profile ${profile.name}`}
                        >
                          <ArrowLeftRight className="h-4 w-4" />
                        </Button>
                      ) : null}
                      <Button
                        type="button"
                        size="icon"
                        variant="ghost"
                        className="text-[var(--folder-primary)] hover:bg-[color-mix(in_srgb,var(--folder-primary)_15%,transparent)] hover:text-[var(--folder-primary)]"
                        onClick={(event) => {
                          event.stopPropagation();
                          setRenameTarget(profile);
                          setRenameName(profile.name);
                        }}
                        aria-label={`Rename profile ${profile.name}`}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      {profile.id !== '__default__' ? (
                        <Button
                          type="button"
                          size="icon"
                          variant="ghost"
                          className="text-[var(--uninstall-primary)] hover:bg-[color-mix(in_srgb,var(--uninstall-primary)_15%,transparent)] hover:text-[var(--uninstall-primary)]"
                          onClick={(event) => {
                            event.stopPropagation();
                            setDeleteTarget(profile);
                          }}
                          aria-label={`Delete profile ${profile.name}`}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      ) : null}
                    </div>

                    <div className="col-start-2 col-end-3 mt-1 grid grid-cols-1 gap-2 text-sm sm:grid-cols-3">
                      <div className="rounded-md border border-border/70 bg-muted/20 px-2 py-1">
                        <p className="text-[11px] uppercase tracking-wide text-muted-foreground">
                          Maps
                        </p>
                        <p className="font-semibold">{counts.maps}</p>
                      </div>
                      <div className="rounded-md border border-border/70 bg-muted/20 px-2 py-1">
                        <p className="text-[11px] uppercase tracking-wide text-muted-foreground">
                          Mods
                        </p>
                        <p className="font-semibold">{counts.mods}</p>
                      </div>
                      <div className="rounded-md border border-border/70 bg-muted/20 px-2 py-1">
                        <p className="text-[11px] uppercase tracking-wide text-muted-foreground">
                          Archive Size
                        </p>
                        <p className="font-semibold">{archiveSizeDisplay}</p>
                      </div>
                    </div>
                  </div>
                </button>

                {isExpanded ? (
                  <div className="grid gap-3 border-t border-border/70 bg-muted/10 px-4 py-3 md:grid-cols-3">
                    <section className="rounded-lg border border-border/70 bg-card/50 p-3">
                      <h4 className="text-xs font-semibold uppercase tracking-wide text-[var(--profiles-primary)]">
                        Subscriptions
                      </h4>
                      <p className="mt-2 text-sm text-muted-foreground">
                        Detailed subscription breakdown will be added here.
                      </p>
                    </section>
                    <section className="rounded-lg border border-border/70 bg-card/50 p-3">
                      <h4 className="text-xs font-semibold uppercase tracking-wide text-[var(--profiles-primary)]">
                        UI Preferences
                      </h4>
                      <p className="mt-2 text-sm text-muted-foreground">
                        Theme, paging, and view preferences will be shown here.
                      </p>
                    </section>
                    <section className="rounded-lg border border-border/70 bg-card/50 p-3">
                      <h4 className="text-xs font-semibold uppercase tracking-wide text-[var(--profiles-primary)]">
                        System Preferences
                      </h4>
                      <p className="mt-2 text-sm text-muted-foreground">
                        Startup and runtime preferences will be shown here.
                      </p>
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
            onClick={() => setCreateOpen(true)}
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

      <AppDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        icon={CircleUserRound}
        title="Create Profile"
        description="Enter a profile name."
        tone="profiles"
        confirm={{
          label: 'Create',
          onConfirm: () => void handleCreate(),
          loading: createLoading,
        }}
      >
        <div className="space-y-2">
          <label htmlFor="profile-name" className="text-sm font-medium">
            Profile Name
          </label>
          <Input
            id="profile-name"
            value={createName}
            onChange={(event) => setCreateName(event.target.value)}
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

      <AppDialog
        open={swapTarget !== null && !swapArchiveWarningOpen}
        onOpenChange={(open) => {
          if (!open) {
            setSwapTarget(null);
            setSwapArchiveWarningOpen(false);
          }
        }}
        icon={ArrowLeftRight}
        title="Switch Profile"
        description={
          swapTarget
            ? `Switch active profile to "${swapTarget.name}"?`
            : 'Switch active profile?'
        }
        tone="update"
        confirm={{
          label: 'Switch',
          onConfirm: () => void handleSwap(false),
          loading: swapLoading,
        }}
      />

      <AppDialog
        open={swapTarget !== null && swapArchiveWarningOpen}
        onOpenChange={(open) => {
          if (!open) {
            setSwapTarget(null);
            setSwapArchiveWarningOpen(false);
          }
        }}
        icon={AlertTriangle}
        title="Confirm Profile Switch"
        description={
          swapTarget
            ? `Profile "${swapTarget.name}" has a missing or stale archive and may require additional downloads during sync. Continue?`
            : 'Selected profile has a missing or stale archive. Continue?'
        }
        tone="files"
        confirm={{
          label: 'Continue',
          onConfirm: () => void handleSwap(true),
          loading: swapLoading,
        }}
      >
        <div
          className={`rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground ${FILES_ACCENT.dialogPanel}`}
        >
          <p className="font-medium text-foreground">
            Target Profile: {swapTarget?.name ?? ''}
          </p>
          <p className="mt-1 font-mono">UUID: {swapTarget?.uuid ?? ''}</p>
        </div>
      </AppDialog>

      <AppDialog
        open={renameTarget !== null}
        onOpenChange={(open) => {
          if (!open) {
            setRenameTarget(null);
            setRenameName('');
          }
        }}
        icon={Pencil}
        title="Rename Profile"
        description={
          renameTarget
            ? `Enter a new name for "${renameTarget.name}".`
            : 'Enter a new profile name.'
        }
        tone="files"
        confirm={{
          label: 'Rename',
          onConfirm: () => void handleRename(),
          loading: renameLoading,
        }}
      >
        <div className="space-y-2">
          <label htmlFor="rename-profile-name" className="text-sm font-medium">
            Profile Name
          </label>
          <Input
            id="rename-profile-name"
            value={renameName}
            onChange={(event) => setRenameName(event.target.value)}
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

      <AppDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        icon={Trash2}
        title="Delete Profile"
        description={
          deleteTarget
            ? `Delete profile "${deleteTarget.name}"? This cannot be undone.`
            : 'Delete this profile?'
        }
        tone="uninstall"
        confirm={{
          label: 'Delete',
          onConfirm: () => void handleDelete(),
          loading: deleteLoading,
        }}
      />
    </div>
  );
}
