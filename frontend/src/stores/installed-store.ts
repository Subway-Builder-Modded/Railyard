import { create } from 'zustand';
import { types } from '../../wailsjs/go/models';
import { GetInstalledMods, GetInstalledMaps } from '../../wailsjs/go/registry/Registry';
import { GetActiveProfile, UpdateSubscriptions } from '../../wailsjs/go/profiles/UserProfiles';
import {
  InstallMap as InstallMapFiles,
  InstallMod as InstallModFiles,
  UninstallMap as UninstallMapFiles,
  UninstallMod as UninstallModFiles,
} from '../../wailsjs/go/downloader/Downloader';
import { useDownloadQueueStore } from './download-queue-store';
import type { AssetType } from "@/lib/asset-types";

interface InstalledState {
  installedMods: types.InstalledModInfo[];
  installedMaps: types.InstalledMapInfo[];
  installing: Set<string>;
  uninstalling: Set<string>;
  loading: boolean;
  error: string | null;
  initialized: boolean;

  initialize: () => Promise<void>;
  installMod: (id: string, version: string) => Promise<types.UpdateSubscriptionsResult>;
  installMap: (id: string, version: string) => Promise<types.UpdateSubscriptionsResult>;
  uninstallMod: (id: string) => Promise<types.UpdateSubscriptionsResult>;
  uninstallMap: (id: string) => Promise<types.UpdateSubscriptionsResult>;
  isInstalled: (id: string) => boolean;
  getInstalledVersion: (id: string) => string | null;
  isOperating: (id: string) => boolean;
  updateInstalledLists: () => Promise<void>;
}

export const useInstalledStore = create<InstalledState>((set, get) => {
  const applySubscriptionMutation = async (
    id: string,
    version: string,
    assetType: AssetType,
    action: "subscribe" | "unsubscribe",
  ) => {
    const activeProfileResult = await GetActiveProfile();
    if (activeProfileResult.status !== "success") {
      throw new Error(activeProfileResult.message || "Failed to resolve active profile");
    }
    const request = new types.UpdateSubscriptionsRequest({
      profileId: activeProfileResult.profile.id,
      assets: {
        [id]: new types.SubscriptionUpdateItem({
          version,
          type: assetType,
        }),
      },
      action,
      forceSync: true,
    });
    const result = await UpdateSubscriptions(request);
    if (result.status === "error") {
      throw new Error(result.message || "Subscription update failed");
    }
    return result;
  };

  return ({
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
      const [mods, maps] = await Promise.all([GetInstalledMods(), GetInstalledMaps()]);
      set({ installedMods: mods || [], installedMaps: maps || [], initialized: true, loading: false });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err), loading: false });
    }
  },

  updateInstalledLists: async () => {
    set({ loading: true, error: null });
    try {
      const [mods, maps] = await Promise.all([GetInstalledMods(), GetInstalledMaps()]);
      set({ installedMods: mods || [], installedMaps: maps || [], loading: false });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err), loading: false });
    }
  },

  installMod: async (id: string, version: string) => {
    useDownloadQueueStore.getState().enqueue();
    set({ installing: new Set([...get().installing, id]), error: null });
    try {
      const response = await applySubscriptionMutation(id, version, "mod", "subscribe");
      const directResponse = await InstallModFiles(id, version);
      if (directResponse.status === "error") {
        try {
          await applySubscriptionMutation(id, "", "mod", "unsubscribe");
        } catch {}
        throw new Error(directResponse.message || "Install failed");
      }
      const [mods, maps] = await Promise.all([GetInstalledMods(), GetInstalledMaps()]);
      const next = new Set(get().installing);
      next.delete(id);
      set({ installing: next, installedMods: mods || [], installedMaps: maps || [] });
      useDownloadQueueStore.getState().complete();
      return response;
    } catch (err) {
      const next = new Set(get().installing);
      next.delete(id);
      set({ installing: next, error: err instanceof Error ? err.message : String(err) });
      useDownloadQueueStore.getState().complete();
      throw err;
    }
  },

  installMap: async (id: string, version: string) => {
    useDownloadQueueStore.getState().enqueue();
    set({ installing: new Set([...get().installing, id]), error: null });
    try {
      const response = await applySubscriptionMutation(id, version, "map", "subscribe");
      const directResponse = await InstallMapFiles(id, version);
      if (directResponse.status === "error") {
        try {
          await applySubscriptionMutation(id, "", "map", "unsubscribe");
        } catch {}
        throw new Error(directResponse.message || "Install failed");
      }
      const [mods, maps] = await Promise.all([GetInstalledMods(), GetInstalledMaps()]);
      const next = new Set(get().installing);
      next.delete(id);
      set({ installing: next, installedMods: mods || [], installedMaps: maps || [] });
      useDownloadQueueStore.getState().complete();
      return response;
    } catch (err) {
      const next = new Set(get().installing);
      next.delete(id);
      set({ installing: next, error: err instanceof Error ? err.message : String(err) });
      useDownloadQueueStore.getState().complete();
      throw err;
    }
  },

  uninstallMod: async (id: string) => {
    set({ uninstalling: new Set([...get().uninstalling, id]), error: null });
    try {
      const directResponse = await UninstallModFiles(id);
      if (directResponse.status === "error") {
        throw new Error(directResponse.message || "Uninstall failed");
      }
      const response = await applySubscriptionMutation(id, "", "mod", "unsubscribe");
      const [mods, maps] = await Promise.all([GetInstalledMods(), GetInstalledMaps()]);
      const next = new Set(get().uninstalling);
      next.delete(id);
      set({ uninstalling: next, installedMods: mods || [], installedMaps: maps || [] });
      return response;
    } catch (err) {
      const next = new Set(get().uninstalling);
      next.delete(id);
      set({ uninstalling: next, error: err instanceof Error ? err.message : String(err) });
      throw err;
    }
  },

  uninstallMap: async (id: string) => {
    set({ uninstalling: new Set([...get().uninstalling, id]), error: null });
    try {
      const directResponse = await UninstallMapFiles(id);
      if (directResponse.status === "error") {
        throw new Error(directResponse.message || "Uninstall failed");
      }
      const response = await applySubscriptionMutation(id, "", "map", "unsubscribe");
      const [mods, maps] = await Promise.all([GetInstalledMods(), GetInstalledMaps()]);
      const next = new Set(get().uninstalling);
      next.delete(id);
      set({ uninstalling: next, installedMods: mods || [], installedMaps: maps || [] });
      return response;
    } catch (err) {
      const next = new Set(get().uninstalling);
      next.delete(id);
      set({ uninstalling: next, error: err instanceof Error ? err.message : String(err) });
      throw err;
    }
  },

  isInstalled: (id: string) => {
    const { installedMods, installedMaps } = get();
    return installedMods.some((m) => m.id === id) || installedMaps.some((m) => m.id === id);
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
  });
});
