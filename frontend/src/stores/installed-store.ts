import { create } from 'zustand';

import type { AssetType } from '@/lib/asset-types';

import { types } from '../../wailsjs/go/models';
import {
  GetActiveProfile,
  UpdateSubscriptions,
} from '../../wailsjs/go/profiles/UserProfiles';
import {
  GetInstalledMapsResponse,
  GetInstalledModsResponse,
} from '../../wailsjs/go/registry/Registry';
import { useDownloadQueueStore } from './download-queue-store';

export class SubscriptionSyncError extends Error {
  readonly status: string;
  readonly profileErrors: types.UserProfilesError[];

  constructor(
    message: string,
    status: string,
    profileErrors: types.UserProfilesError[],
  ) {
    super(message);
    this.name = 'SubscriptionSyncError';
    this.status = status;
    this.profileErrors = profileErrors;
  }
}

function resolveSubscriptionSyncMessage(
  result: types.UpdateSubscriptionsResult,
  fallback: string,
): string {
  if (result.message?.trim()) {
    return result.message;
  }

  const firstError = result.errors?.[0];
  if (firstError?.message?.trim()) {
    return firstError.message;
  }

  return fallback;
}

interface InstalledState {
  installedMods: types.InstalledModInfo[];
  installedMaps: types.InstalledMapInfo[];
  installing: Set<string>;
  uninstalling: Set<string>;
  loading: boolean;
  error: string | null;
  initialized: boolean;

  initialize: () => Promise<void>;
  installMod: (
    id: string,
    version: string,
  ) => Promise<types.UpdateSubscriptionsResult>;
  installMap: (
    id: string,
    version: string,
  ) => Promise<types.UpdateSubscriptionsResult>;
  uninstallMod: (id: string) => Promise<types.UpdateSubscriptionsResult>;
  uninstallMap: (id: string) => Promise<types.UpdateSubscriptionsResult>;
  uninstallAssets: (
    assets: Array<{ id: string; type: AssetType }>,
  ) => Promise<types.UpdateSubscriptionsResult>;
  cancelPendingInstall: (
    type: AssetType,
    id: string,
  ) => Promise<types.UpdateSubscriptionsResult>;
  acknowledgeCancelledInstall: (id: string) => void;
  isInstalled: (id: string) => boolean;
  getInstalledVersion: (id: string) => string | null;
  isOperating: (id: string) => boolean;
  isInstalling: (id: string) => boolean;
  isUninstalling: (id: string) => boolean;
  updateInstalledLists: () => Promise<void>;
}

export const useInstalledStore = create<InstalledState>((set, get) => {
  const getInstalledLists = async () => {
    const [modsResponse, mapsResponse] = await Promise.all([
      GetInstalledModsResponse(),
      GetInstalledMapsResponse(),
    ]);
    if (modsResponse.status !== 'success') {
      throw new Error(modsResponse.message || 'Failed to load installed mods');
    }
    if (mapsResponse.status !== 'success') {
      throw new Error(mapsResponse.message || 'Failed to load installed maps');
    }

    return {
      installedMods: modsResponse.mods || [],
      installedMaps: mapsResponse.maps || [],
    };
  };

  const setOperationState = (
    key: 'installing' | 'uninstalling',
    id: string,
    active: boolean,
  ) => {
    set((state) => {
      const next = new Set(state[key]);
      if (active) {
        next.add(id);
      } else {
        next.delete(id);
      }

      return { [key]: next } as Pick<InstalledState, typeof key>;
    });
  };

  const setOperationStateForIds = (
    key: 'installing' | 'uninstalling',
    ids: string[],
    active: boolean,
  ) => {
    set((state) => {
      const next = new Set(state[key]);
      for (const id of ids) {
        if (active) {
          next.add(id);
        } else {
          next.delete(id);
        }
      }

      return { [key]: next } as Pick<InstalledState, typeof key>;
    });
  };

  const applySubscriptionMutation = async (
    assets: Record<string, types.SubscriptionUpdateItem>,
    action: 'subscribe' | 'unsubscribe',
  ) => {
    if (Object.keys(assets).length === 0) {
      throw new Error('No assets provided for subscription update');
    }

    const activeProfileResult = await GetActiveProfile();
    if (activeProfileResult.status !== 'success') {
      throw new Error(
        activeProfileResult.message || 'Failed to resolve active profile',
      );
    }
    const request = new types.UpdateSubscriptionsRequest({
      profileId: activeProfileResult.profile.id,
      assets,
      action,
      forceSync: true,
    });
    const result = await UpdateSubscriptions(request);
    if (result.status === 'error') {
      throw new SubscriptionSyncError(
        resolveSubscriptionSyncMessage(result, 'Subscription update failed'),
        result.status,
        result.errors ?? [],
      );
    }
    return result;
  };

  const installAsset = async (
    id: string,
    version: string,
    assetType: AssetType,
  ) => {
    useDownloadQueueStore.getState().enqueue();
    setOperationState('installing', id, true);
    set({ error: null });

    try {
      const response = await applySubscriptionMutation(
        {
          [id]: new types.SubscriptionUpdateItem({
            version,
            type: assetType,
          }),
        },
        'subscribe',
      );
      set({ ...(await getInstalledLists()) });
      return response;
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
      throw err;
    } finally {
      setOperationState('installing', id, false);
      useDownloadQueueStore.getState().complete();
    }
  };

  const uninstallAssets = async (
    assets: Array<{ id: string; type: AssetType }>,
  ) => {
    if (assets.length === 0) {
      throw new Error('No assets provided for uninstall');
    }

    const ids = assets.map((asset) => asset.id);
    const subscriptionAssets = assets.reduce<
      Record<string, types.SubscriptionUpdateItem>
    >((accumulator, asset) => {
      accumulator[asset.id] = new types.SubscriptionUpdateItem({
        version: '',
        type: asset.type,
      });
      return accumulator;
    }, {});

    setOperationStateForIds('uninstalling', ids, true);
    set({ error: null });

    try {
      const response = await applySubscriptionMutation(
        subscriptionAssets,
        'unsubscribe',
      );
      set({ ...(await getInstalledLists()) });
      return response;
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
      throw err;
    } finally {
      setOperationStateForIds('uninstalling', ids, false);
    }
  };

  return {
    installedMods: [],
    installedMaps: [],
    installing: new Set<string>(),
    uninstalling: new Set<string>(),
    loading: false,
    error: null,
    initialized: false,

    initialize: async () => {
      if (get().initialized) return;
      set({ loading: true, error: null });
      try {
        set({
          ...(await getInstalledLists()),
          initialized: true,
          loading: false,
        });
      } catch (err) {
        set({
          error: err instanceof Error ? err.message : String(err),
          loading: false,
        });
      }
    },

    updateInstalledLists: async () => {
      set({ loading: true, error: null });
      try {
        set({ ...(await getInstalledLists()), loading: false });
      } catch (err) {
        set({
          error: err instanceof Error ? err.message : String(err),
          loading: false,
        });
      }
    },

    installMod: (id: string, version: string) =>
      installAsset(id, version, 'mod'),

    installMap: (id: string, version: string) =>
      installAsset(id, version, 'map'),

    uninstallMod: (id: string) => uninstallAssets([{ id, type: 'mod' }]),

    uninstallMap: (id: string) => uninstallAssets([{ id, type: 'map' }]),

    uninstallAssets,

    cancelPendingInstall: async (type: AssetType, id: string) => {
      return uninstallAssets([{ id, type }]);
    },

    acknowledgeCancelledInstall: (id: string) => {
      set((state) => {
        if (!state.installing.has(id)) {
          return state;
        }
        const nextInstalling = new Set(state.installing);
        nextInstalling.delete(id);
        return { installing: nextInstalling };
      });
    },

    isInstalled: (id: string) => {
      const { installedMods, installedMaps } = get();
      return (
        installedMods.some((m) => m.id === id) ||
        installedMaps.some((m) => m.id === id)
      );
    },

    getInstalledVersion: (id: string) => {
      const { installedMods, installedMaps } = get();
      const mod = installedMods.find((m) => m.id === id);
      if (mod) return mod.version;
      const map = installedMaps.find((m) => m.id === id);
      if (map) return map.version;
      return null;
    },

    isOperating: (id: string) => {
      return get().installing.has(id) || get().uninstalling.has(id);
    },

    isInstalling: (id: string) => get().installing.has(id),

    isUninstalling: (id: string) => get().uninstalling.has(id),
  };
});
