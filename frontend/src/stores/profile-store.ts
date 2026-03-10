import { create } from 'zustand';
import { types } from '../../wailsjs/go/models';
import { GetActiveProfile, UpdateSubscriptions, ResetUserProfiles, UpdateUIPreferences } from '../../wailsjs/go/profiles/UserProfiles';

interface ProfileState {
  profile: types.UserProfile | null;
  loading: boolean;
  error: string | null;
  initialized: boolean;

  initialize: () => Promise<void>;
  isSubscribed: (type: "mods" | "maps", id: string) => boolean;
  theme: () => string;
  defaultPerPage: () => number;
  updateUIPreferences: (theme: string, defaultPerPage: number) => Promise<void>;
  updateSubscription: (
    type: "mods" | "maps",
    id: string,
    action: "subscribe" | "unsubscribe",
    version: string,
  ) => Promise<void>;
  resetProfile: () => Promise<void>;
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
      set({ profile: result.profile, initialized: true, loading: false });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err), loading: false });
    }
  },

  isSubscribed: (type: "mods" | "maps", id: string) => {
    const profile = get().profile;
    if (!profile?.subscriptions) return false;
    const subs = type === "mods" ? profile.subscriptions.mods : profile.subscriptions.maps;
    return subs ? id in subs : false;
  },

  theme: () => get().profile?.uiPreferences?.theme ?? "dark",
  defaultPerPage: () => get().profile?.uiPreferences?.defaultPerPage ?? 12,

  updateUIPreferences: async (theme, defaultPerPage) => {
    const result = await UpdateUIPreferences(new types.UIPreferences({ theme, defaultPerPage }));
    if (result.status !== "success") {
      throw new Error(result.message || "Failed to update UI preferences");
    }
    set({ profile: result.profile });
  },

  updateSubscription: async (type, id, action, version) => {
    const profile = get().profile;
    if (!profile) return;

    const assetType = type === "mods" ? "mod" : "map";
    const request = new types.UpdateSubscriptionsRequest({
      profileId: profile.id,
      assets: { [id]: new types.SubscriptionUpdateItem({ version, type: assetType }) },
      action,
      forceSync: true,
    });

    const result = await UpdateSubscriptions(request);
    if (result.status === "error") throw new Error(result.message);
    set({ profile: result.profile });
  },

  resetProfile: async () => {
    set({ loading: true, error: null });
    try {
      const resetResult = await ResetUserProfiles();
      set({ profile: resetResult.profile, loading: false });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err), loading: false });
    }
  },
}));
