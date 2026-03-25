import { create } from 'zustand';

import type { AssetType } from '@/lib/asset-types';
import {
  normalizeSearchViewMode,
  type SearchViewMode,
} from '@/lib/search-view-mode';

import { types } from '../../wailsjs/go/models';
import {
  GetActiveProfile,
  ResetUserProfiles,
  UpdateSubscriptions,
  UpdateSystemPreferences,
  UpdateUIPreferences,
} from '../../wailsjs/go/profiles/UserProfiles';

interface UIPreferencesPayload {
  theme: string;
  defaultPerPage: number;
  searchViewMode: SearchViewMode;
}

interface UpdateCommandLineArgsPayload {
  extraMemorySize?: number;
  useDevTools?: boolean;
}

const DEFAULT_UI_PREFERENCES: UIPreferencesPayload = {
  theme: 'dark',
  defaultPerPage: 12,
  searchViewMode: 'full',
};

function resolveUIPreferences(
  profile: types.UserProfile | null,
): UIPreferencesPayload {
  const uiPreferences = profile?.uiPreferences as
    | (types.UIPreferences & { searchViewMode?: unknown })
    | undefined;

  return {
    theme: uiPreferences?.theme ?? DEFAULT_UI_PREFERENCES.theme,
    defaultPerPage:
      uiPreferences?.defaultPerPage ?? DEFAULT_UI_PREFERENCES.defaultPerPage,
    searchViewMode: normalizeSearchViewMode(
      uiPreferences?.searchViewMode,
      DEFAULT_UI_PREFERENCES.searchViewMode,
    ),
  };
}

function resolveSystemPreferences(
  profile: types.UserProfile | null,
): types.SystemPreferences {
  return {
    refreshRegistryOnStartup:
      profile?.systemPreferences?.refreshRegistryOnStartup ?? false,
    extraMemorySize: profile?.systemPreferences?.extraMemorySize ?? 0,
    useDevTools: profile?.systemPreferences?.useDevTools ?? false,
  };
}

interface ProfileState {
  profile: types.UserProfile | null;
  loading: boolean;
  error: string | null;
  initialized: boolean;

  initialize: () => Promise<void>;
  isSubscribed: (type: AssetType, id: string) => boolean;
  theme: () => string;
  defaultPerPage: () => number;
  searchViewMode: () => SearchViewMode;
  updateUIPreferences: (
    updates: Partial<UIPreferencesPayload>,
  ) => Promise<void>;
  updateSubscription: (
    type: AssetType,
    id: string,
    action: 'subscribe' | 'unsubscribe',
    version: string,
  ) => Promise<void>;
  resetProfile: () => Promise<void>;
  updateCommandLineArgs: (
    preferences: Partial<UpdateCommandLineArgsPayload>,
  ) => Promise<void>;
}

export const useProfileStore = create<ProfileState>((set, get) => ({
  profile: null,
  loading: false,
  error: null,
  initialized: false,

  initialize: async () => {
    if (get().initialized) return;
    set({ loading: true, error: null });
    try {
      const result = await GetActiveProfile();
      if (result.status !== 'success') {
        throw new Error(result.message || 'Failed to load active profile');
      }
      set({ profile: result.profile, initialized: true, loading: false });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : String(err),
        initialized: true,
        loading: false,
      });
    }
  },

  isSubscribed: (type: AssetType, id: string) => {
    const profile = get().profile;
    if (!profile?.subscriptions) return false;
    const subs =
      type === 'mod' ? profile.subscriptions.mods : profile.subscriptions.maps;
    return subs ? id in subs : false;
  },

  theme: () => resolveUIPreferences(get().profile).theme,
  defaultPerPage: () => resolveUIPreferences(get().profile).defaultPerPage,
  searchViewMode: () => resolveUIPreferences(get().profile).searchViewMode,

  updateUIPreferences: async (updates) => {
    const nextPreferences: UIPreferencesPayload = {
      ...resolveUIPreferences(get().profile),
      ...updates,
    };

    const result = await UpdateUIPreferences(
      nextPreferences as unknown as types.UIPreferences,
    );
    if (result.status !== 'success') {
      throw new Error(result.message || 'Failed to update UI preferences');
    }
    set({ profile: result.profile });
  },

  updateSubscription: async (type, id, action, version) => {
    // Always resolve a fresh profile to avoid stale IDs from cached state
    const activeResult = await GetActiveProfile();
    if (activeResult.status !== 'success') {
      throw new Error(
        activeResult.message || 'Failed to resolve active profile',
      );
    }
    const freshProfile = activeResult.profile;

    const request = new types.UpdateSubscriptionsRequest({
      profileId: freshProfile.id,
      assets: { [id]: new types.SubscriptionUpdateItem({ version, type }) },
      action,
      applyMode: 'persist_and_sync',
    });

    const result = await UpdateSubscriptions(request);
    if (result.status === 'error') throw new Error(result.message);
    set({ profile: result.profile });
  },

  resetProfile: async () => {
    set({ loading: true, error: null });
    try {
      const resetResult = await ResetUserProfiles();
      set({ profile: resetResult.profile, loading: false });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : String(err),
        loading: false,
      });
    }
  },

  updateCommandLineArgs: async (preferences) => {
    set({ loading: true, error: null });
    try {
      const payload = {
        ...resolveSystemPreferences(get().profile),
        ...preferences,
      };
      const result = await UpdateSystemPreferences(payload);
      if (result.status === 'error') {
        throw new Error(
          result.message || 'Failed to update system preferences',
        );
      }
      set({ profile: result.profile, loading: false });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : String(err),
        loading: false,
      });
    }
  },
}));
