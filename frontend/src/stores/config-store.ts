import { create } from 'zustand';
import { types } from '../../wailsjs/go/models';
import {
  GetConfig,
  SaveConfig,
  ClearConfig,
  OpenMetroMakerDataFolderDialog,
  OpenExecutableDialog,
} from '../../wailsjs/go/config/Config';

interface ConfigState {
  config: types.AppConfig | null;
  validation: types.ConfigPathValidation | null;
  loading: boolean;
  error: string | null;
  initialized: boolean;

  isConfigured: () => boolean;
  initialize: () => Promise<void>;
  openDataFolderDialog: (allowAutoDetect: boolean) => Promise<types.SetConfigPathResult>;
  openExecutableDialog: (allowAutoDetect: boolean) => Promise<types.SetConfigPathResult>;
  saveConfig: () => Promise<void>;
  clearConfig: () => Promise<void>;
}

export const useConfigStore = create<ConfigState>((set, get) => ({
  config: null,
  validation: null,
  loading: false,
  error: null,
  initialized: false,

  isConfigured: () => get().validation?.isConfigured ?? false,

  initialize: async () => {
    if (get().initialized) return;
    set({ loading: true, error: null });
    try {
      const result = await GetConfig();
      set({
        config: result.config,
        validation: result.validation,
        initialized: true,
        loading: false,
      });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err), loading: false });
    }
  },

  openDataFolderDialog: async (allowAutoDetect: boolean) => {
    set({ error: null });
    try {
      const result = await OpenMetroMakerDataFolderDialog(new types.SetConfigPathOptions({ allowAutoDetect }));
      set({
        config: result.resolveConfigResult.config,
        validation: result.resolveConfigResult.validation,
      });
      return result;
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
      throw err;
    }
  },

  openExecutableDialog: async (allowAutoDetect: boolean) => {
    set({ error: null });
    try {
      const result = await OpenExecutableDialog(new types.SetConfigPathOptions({ allowAutoDetect }));
      set({
        config: result.resolveConfigResult.config,
        validation: result.resolveConfigResult.validation,
      });
      return result;
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
      throw err;
    }
  },

  saveConfig: async () => {
    set({ loading: true, error: null });
    try {
      const result = await SaveConfig();
      set({ config: result.config, validation: result.validation, loading: false });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err), loading: false });
    }
  },

  clearConfig: async () => {
    set({ loading: true, error: null });
    try {
      await ClearConfig();
      const result = await GetConfig();
      set({ config: result.config, validation: result.validation, loading: false });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err), loading: false });
    }
  },
}));
