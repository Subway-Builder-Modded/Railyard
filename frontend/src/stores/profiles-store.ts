import { create } from 'zustand';

import { types } from '../../wailsjs/go/models';
import {
  CreateProfile,
  DeleteProfile,
  ListProfiles,
  RenameProfile,
  SwapProfile,
} from '../../wailsjs/go/profiles/UserProfiles';

// ProfilesStoreState wraps around the backend profiles API and manages the state of user profiles for the frontend.
interface ProfilesStoreState {
  profiles: types.UserProfile[];
  archiveSizes: Record<string, number>;
  subscriptionSizes: Record<string, number>;
  activeProfileID: string;
  loading: boolean;
  loadProfiles: () => Promise<void>;
  createProfile: (name: string) => Promise<types.UserProfileResult>;
  renameProfile: (
    profileID: string,
    name: string,
  ) => Promise<types.UserProfileResult>;
  deleteProfile: (profileID: string) => Promise<types.UserProfileResult>;
  swapProfile: (
    profileID: string,
    forceSwap: boolean,
  ) => Promise<types.UserProfileResult>;
}

export const useProfilesStore = create<ProfilesStoreState>((set) => ({
  profiles: [],
  archiveSizes: {},
  subscriptionSizes: {},
  activeProfileID: '',
  loading: false,

  loadProfiles: async () => {
    set({ loading: true });
    try {
      const result = await ListProfiles();
      if (result.status !== 'success') {
        throw new Error(result.message || 'Failed to load profiles');
      }
      set({
        profiles: result.profiles ?? [],
        archiveSizes: result.archiveSizes ?? {},
        subscriptionSizes: result.subscriptionSizes ?? {},
        activeProfileID: result.activeProfileId ?? '',
      });
    } finally {
      set({ loading: false });
    }
  },

  createProfile: async (name) => {
    return CreateProfile(new types.CreateProfileRequest({ name }));
  },

  renameProfile: async (profileID, name) => {
    return RenameProfile(profileID, name);
  },

  deleteProfile: async (profileID) => {
    return DeleteProfile(profileID);
  },

  swapProfile: async (profileID, forceSwap) => {
    return SwapProfile(
      new types.SwapProfileRequest({
        profileId: profileID,
        forceSwap,
      }),
    );
  },
}));
