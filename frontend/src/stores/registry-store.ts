import { create } from 'zustand';
import { types } from '../../wailsjs/go/models';
import { GetMods, GetMaps, Refresh } from '../../wailsjs/go/registry/Registry';

interface RegistryState {
  mods: types.ModManifest[];
  maps: types.MapManifest[];
  loading: boolean;
  refreshing: boolean;
  error: string | null;
  initialized: boolean;
  initialize: () => Promise<void>;
  refresh: () => Promise<void>;
}

export const useRegistryStore = create<RegistryState>((set, get) => ({
  mods: [],
  maps: [],
  loading: false,
  refreshing: false,
  error: null,
  initialized: false,

  initialize: async () => {
    if (get().initialized) return;
    set({ loading: true, error: null });
    try {
      const [mods, maps] = await Promise.all([GetMods(), GetMaps()]);
      set({ mods: mods || [], maps: maps || [], initialized: true, loading: false });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err), loading: false });
    }
  },

  refresh: async () => {
    set({ refreshing: true, error: null });
    try {
      await Refresh();
      const [mods, maps] = await Promise.all([GetMods(), GetMaps()]);
      set({ mods: mods || [], maps: maps || [], refreshing: false });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err), refreshing: false });
    }
  },
}));
