import { beforeEach, describe, expect, it, vi } from 'vitest';

import { types } from '../../wailsjs/go/models';
import { useProfilesStore } from './profiles-store';

const {
  mockListProfiles,
  mockCreateProfile,
  mockRenameProfile,
  mockDeleteProfile,
  mockSwapProfile,
} = vi.hoisted(() => ({
  mockListProfiles: vi.fn(),
  mockCreateProfile: vi.fn(),
  mockRenameProfile: vi.fn(),
  mockDeleteProfile: vi.fn(),
  mockSwapProfile: vi.fn(),
}));

vi.mock('../../wailsjs/go/profiles/UserProfiles', () => ({
  ListProfiles: mockListProfiles,
  CreateProfile: mockCreateProfile,
  RenameProfile: mockRenameProfile,
  DeleteProfile: mockDeleteProfile,
  SwapProfile: mockSwapProfile,
}));

describe('useProfilesStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useProfilesStore.setState({
      profiles: [],
      archiveSizes: {},
      subscriptionSizes: {},
      activeProfileID: '',
      loading: false,
    });
  });

  it('loadProfiles populates state on success', async () => {
    mockListProfiles.mockResolvedValue({
      status: 'success',
      message: 'ok',
      activeProfileId: 'profile_1',
      profiles: [{ id: 'profile_1', name: 'Default' }],
      archiveSizes: { profile_1: 100 },
      subscriptionSizes: { profile_1: 200 },
    });

    await useProfilesStore.getState().loadProfiles();

    const state = useProfilesStore.getState();
    expect(mockListProfiles).toHaveBeenCalledTimes(1);
    expect(state.loading).toBe(false);
    expect(state.activeProfileID).toBe('profile_1');
    expect(state.profiles).toEqual([{ id: 'profile_1', name: 'Default' }]);
    expect(state.archiveSizes).toEqual({ profile_1: 100 });
    expect(state.subscriptionSizes).toEqual({ profile_1: 200 });
  });

  it('loadProfiles throws on non-success and resets loading', async () => {
    mockListProfiles.mockResolvedValue({
      status: 'error',
      message: 'Failed to load profiles',
      profiles: [],
    });

    await expect(useProfilesStore.getState().loadProfiles()).rejects.toThrow(
      'Failed to load profiles',
    );
    expect(useProfilesStore.getState().loading).toBe(false);
  });

  it('createProfile sends CreateProfileRequest', async () => {
    mockCreateProfile.mockResolvedValue({
      status: 'success',
      message: 'created',
      profile: { id: 'profile_2', name: 'Test' },
    });

    await useProfilesStore.getState().createProfile('Test');

    expect(mockCreateProfile).toHaveBeenCalledTimes(1);
    const request = mockCreateProfile.mock.calls[0][0];
    expect(request).toBeInstanceOf(types.CreateProfileRequest);
    expect(request.name).toBe('Test');
  });

  it('renameProfile forwards profile id and name', async () => {
    mockRenameProfile.mockResolvedValue({
      status: 'success',
      message: 'renamed',
      profile: { id: 'profile_2', name: 'Renamed' },
    });

    await useProfilesStore.getState().renameProfile('profile_2', 'Renamed');

    expect(mockRenameProfile).toHaveBeenCalledTimes(1);
    expect(mockRenameProfile).toHaveBeenCalledWith('profile_2', 'Renamed');
  });

  it('deleteProfile forwards profile id', async () => {
    mockDeleteProfile.mockResolvedValue({
      status: 'success',
      message: 'deleted',
    });

    await useProfilesStore.getState().deleteProfile('profile_2');

    expect(mockDeleteProfile).toHaveBeenCalledTimes(1);
    expect(mockDeleteProfile).toHaveBeenCalledWith('profile_2');
  });

  it('swapProfile sends SwapProfileRequest', async () => {
    mockSwapProfile.mockResolvedValue({
      status: 'success',
      message: 'swapped',
      profile: { id: 'profile_2', name: 'Target' },
    });

    await useProfilesStore.getState().swapProfile('profile_2', true);

    expect(mockSwapProfile).toHaveBeenCalledTimes(1);
    const request = mockSwapProfile.mock.calls[0][0];
    expect(request).toBeInstanceOf(types.SwapProfileRequest);
    expect(request.profileId).toBe('profile_2');
    expect(request.forceSwap).toBe(true);
  });
});
