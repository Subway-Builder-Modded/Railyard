import { create } from 'zustand';

import {
  ClearConfig,
  ClearGithubToken,
  CompleteSetup,
  GetConfig,
  IsGithubTokenValid,
  OpenExecutableDialog,
  OpenMetroMakerDataFolderDialog,
  SaveConfig,
  UpdateCheckForUpdatesOnLaunch,
  UpdateGithubToken,
} from '../../wailsjs/go/config/Config';
import { types } from '../../wailsjs/go/models';

interface ConfigState {
  config: types.AppConfig | null;
  validation: types.ConfigPathValidation | null;
  hasGithubToken: boolean;
  loading: boolean;
  error: string | null;
  initialized: boolean;
  githubTokenValid: boolean;

  isConfigured: () => boolean;
  initialize: () => Promise<void>;
  openDataFolderDialog: (
    allowAutoDetect: boolean,
  ) => Promise<types.SetConfigPathResult>;
  openExecutableDialog: (
    allowAutoDetect: boolean,
  ) => Promise<types.SetConfigPathResult>;
  saveConfig: () => Promise<void>;
  clearConfig: () => Promise<void>;
  updateGithubToken: (token: string) => Promise<types.ResolveConfigResponse>;
  clearGithubToken: () => Promise<types.ResolveConfigResponse>;
  updateCheckForUpdatesOnLaunch: (
    checkForUpdates: boolean,
  ) => Promise<types.ResolveConfigResponse>;
  completeSetup: () => Promise<void>;
}

export const useConfigStore = create<ConfigState>((set, get) => ({
  config: null,
  validation: null,
  hasGithubToken: false,
  loading: false,
  error: null,
  initialized: false,
  githubTokenValid: false,

  isConfigured: () => get().validation?.isConfigured ?? false,

  initialize: async () => {
    if (get().initialized) return;
    set({ loading: true, error: null });
    try {
      const result = await GetConfig();
      if (result.status === 'error') {
        throw new Error(result.message || 'Failed to load config');
      }
      const tokenValid = result.hasGithubToken
        ? (await IsGithubTokenValid()).valid
        : false;
      set({
        config: result.config,
        validation: result.validation,
        hasGithubToken: result.hasGithubToken,
        initialized: true,
        loading: false,
        githubTokenValid: tokenValid,
      });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : String(err),
        initialized: true,
        loading: false,
      });
    }
  },

  openDataFolderDialog: async (allowAutoDetect: boolean) => {
    set({ error: null });
    try {
      const response = await OpenMetroMakerDataFolderDialog(
        new types.SetConfigPathOptions({ allowAutoDetect }),
      );
      if (response.status === 'error') {
        throw new Error(
          response.message || 'Failed to open data folder dialog',
        );
      }
      const result = response.result;
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
      const response = await OpenExecutableDialog(
        new types.SetConfigPathOptions({ allowAutoDetect }),
      );
      if (response.status === 'error') {
        throw new Error(response.message || 'Failed to open executable dialog');
      }
      const result = response.result;
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
      if (result.status === 'error') {
        throw new Error(result.message || 'Failed to save config');
      }
      set({
        config: result.config,
        validation: result.validation,
        hasGithubToken: result.hasGithubToken,
        loading: false,
      });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : String(err),
        loading: false,
      });
    }
  },

  clearConfig: async () => {
    set({ loading: true, error: null });
    try {
      const cleared = await ClearConfig();
      if (cleared.status === 'error') {
        throw new Error(cleared.message || 'Failed to clear config');
      }
      const result = await GetConfig();
      if (result.status === 'error') {
        throw new Error(result.message || 'Failed to load config');
      }
      set({
        config: result.config,
        validation: result.validation,
        hasGithubToken: result.hasGithubToken,
        loading: false,
        githubTokenValid: false,
      });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : String(err),
        loading: false,
      });
    }
  },

  updateGithubToken: async (token: string) => {
    set({ error: null });
    try {
      const result = await UpdateGithubToken(token.trim());
      if (result.status === 'error') {
        throw new Error(result.message || 'Failed to update GitHub token');
      }
      const valid = result.hasGithubToken
        ? (await IsGithubTokenValid()).valid
        : false;
      set({
        config: result.config,
        validation: result.validation,
        hasGithubToken: result.hasGithubToken,
        githubTokenValid: valid,
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
      if (result.status === 'error') {
        throw new Error(result.message || 'Failed to clear GitHub token');
      }
      set({
        config: result.config,
        validation: result.validation,
        hasGithubToken: result.hasGithubToken,
        githubTokenValid: false,
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
      if (result.status === 'error') {
        throw new Error(
          result.message || 'Failed to update check for updates setting',
        );
      }
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
      if (result.status === 'error') {
        throw new Error(result.message || 'Failed to complete setup');
      }
      set({
        config: result.config,
        validation: result.validation,
        hasGithubToken: result.hasGithubToken,
        loading: false,
      });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : String(err),
        loading: false,
      });
    }
  },
}));
