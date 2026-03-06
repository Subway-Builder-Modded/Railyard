import { create } from 'zustand';
import { types } from '../../wailsjs/go/models';
import { GetInstalledMods, GetInstalledMaps } from '../../wailsjs/go/registry/Registry';
import { InstallMod, InstallMap, UninstallMod, UninstallMap } from '../../wailsjs/go/downloader/Downloader';

interface InstalledState {
  installedMods: types.InstalledModInfo[];
  installedMaps: types.InstalledMapInfo[];
  installing: Set<string>;
  uninstalling: Set<string>;
  loading: boolean;
  error: string | null;
  initialized: boolean;

  initialize: () => Promise<void>;
  installMod: (id: string, version: string) => Promise<types.GenericResponse>;
  installMap: (id: string, version: string) => Promise<types.MapExtractResponse>;
  uninstallMod: (id: string) => Promise<types.GenericResponse>;
  uninstallMap: (id: string) => Promise<types.GenericResponse>;
  isInstalled: (id: string) => boolean;
  getInstalledVersion: (id: string) => string | null;
  isOperating: (id: string) => boolean;
}

export const useInstalledStore = create<InstalledState>((set, get) => ({
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

  installMod: async (id: string, version: string) => {
    set({ installing: new Set([...get().installing, id]), error: null });
    try {
      const response = await InstallMod(id, version);
      if (response.status !== "success") {
        throw new Error(response.message || "Install failed");
      }
      const [mods, maps] = await Promise.all([GetInstalledMods(), GetInstalledMaps()]);
      const next = new Set(get().installing);
      next.delete(id);
      set({ installing: next, installedMods: mods || [], installedMaps: maps || [] });
      return response;
    } catch (err) {
      const next = new Set(get().installing);
      next.delete(id);
      set({ installing: next, error: err instanceof Error ? err.message : String(err) });
      throw err;
    }
  },

  installMap: async (id: string, version: string) => {
    set({ installing: new Set([...get().installing, id]), error: null });
    try {
      const response = await InstallMap(id, version);
      if (response.status !== "success") {
        throw new Error(response.message || "Install failed");
      }
      const [mods, maps] = await Promise.all([GetInstalledMods(), GetInstalledMaps()]);
      const next = new Set(get().installing);
      next.delete(id);
      set({ installing: next, installedMods: mods || [], installedMaps: maps || [] });
      return response;
    } catch (err) {
      const next = new Set(get().installing);
      next.delete(id);
      set({ installing: next, error: err instanceof Error ? err.message : String(err) });
      throw err;
    }
  },

  uninstallMod: async (id: string) => {
    set({ uninstalling: new Set([...get().uninstalling, id]), error: null });
    try {
      const response = await UninstallMod(id);
      if (response.status !== "success") {
        throw new Error(response.message || "Uninstall failed");
      }
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
      const response = await UninstallMap(id);
      if (response.status !== "success") {
        throw new Error(response.message || "Uninstall failed");
      }
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
}));
