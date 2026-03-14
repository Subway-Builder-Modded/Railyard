import { create } from 'zustand';
import { types } from '../../wailsjs/go/models';
import {
  GetConfig,
  SaveConfig,
  ClearConfig,
  OpenMetroMakerDataFolderDialog,
  OpenExecutableDialog,
  UpdateCheckForUpdatesOnLaunch,
  UpdateGithubToken,
  ClearGithubToken,
  CompleteSetup,
} from '../../wailsjs/go/config/Config';

interface ConfigState {
  config: types.AppConfig | null;
  validation: types.ConfigPathValidation | null;
  hasGithubToken: boolean;
  loading: boolean;
  error: string | null;
  initialized: boolean;

  isConfigured: () => boolean;
  initialize: () => Promise<void>;
  openDataFolderDialog: (allowAutoDetect: boolean) => Promise<types.SetConfigPathResult>;
  openExecutableDialog: (allowAutoDetect: boolean) => Promise<types.SetConfigPathResult>;
  saveConfig: () => Promise<void>;
  clearConfig: () => Promise<void>;
  updateGithubToken: (token: string) => Promise<types.ResolveConfigResult>;
  clearGithubToken: () => Promise<types.ResolveConfigResult>;
  updateCheckForUpdatesOnLaunch: (checkForUpdates: boolean) => Promise<types.ResolveConfigResult>;
  completeSetup: () => Promise<void>;
}

export const useConfigStore = create<ConfigState>((set, get) => ({
  config: null,
  validation: null,
  hasGithubToken: false,
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
        hasGithubToken: result.hasGithubToken,
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
        hasGithubToken: result.resolveConfigResult.hasGithubToken,
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
        hasGithubToken: result.resolveConfigResult.hasGithubToken,
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
      set({
        config: result.config,
        validation: result.validation,
        hasGithubToken: result.hasGithubToken,
        loading: false,
      });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err), loading: false });
    }
  },

  clearConfig: async () => {
    set({ loading: true, error: null });
    try {
      await ClearConfig();
      const result = await GetConfig();
      set({
        config: result.config,
        validation: result.validation,
        hasGithubToken: result.hasGithubToken,
        loading: false,
      });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err), loading: false });
    }
  },

  updateGithubToken: async (token: string) => {
    set({ error: null });
    try {
      const result = await UpdateGithubToken(token.trim());
      set({
        config: result.config,
        validation: result.validation,
        hasGithubToken: result.hasGithubToken,
      });
      return result;
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
      throw err;
    }
  },

  clearGithubToken: async () => {
    set({ error: null });
    try {
      const result = await ClearGithubToken();
      set({
        config: result.config,
        validation: result.validation,
        hasGithubToken: result.hasGithubToken,
      });
      return result;
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
      throw err;
    }
  },

  updateCheckForUpdatesOnLaunch: async (checkForUpdates: boolean) => {
    set({ error: null });
    try {
      const result = await UpdateCheckForUpdatesOnLaunch(checkForUpdates);
      set({
        config: result.config,
        validation: result.validation,
        hasGithubToken: result.hasGithubToken,
      });
      return result;
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
      throw err;
    }
  },

  completeSetup: async () => {
    set({ loading: true, error: null });
    try {
      const result = await CompleteSetup();
      set({
        config: result.config,
        validation: result.validation,
        hasGithubToken: result.hasGithubToken,
        loading: false,
      });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err), loading: false });
    }
  },
}));
