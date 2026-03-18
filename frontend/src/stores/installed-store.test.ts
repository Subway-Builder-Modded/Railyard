import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { AssetType } from '@/lib/asset-types';
import {
  activeProfileResultSuccess,
  updateSubscriptionsError,
  updateSubscriptionsSuccess,
  updateSubscriptionsWarn,
} from '@/test/helpers/profileMutationFixtures';

import { useInstalledStore } from './installed-store';

const {
  mockGetInstalledModsResponse,
  mockGetInstalledMapsResponse,
  mockGetActiveProfile,
  mockUpdateSubscriptions,
  mockInstallMapFiles,
  mockInstallModFiles,
  mockUninstallMapFiles,
  mockUninstallModFiles,
} = vi.hoisted(() => ({
  mockGetInstalledModsResponse: vi.fn(),
  mockGetInstalledMapsResponse: vi.fn(),
  mockGetActiveProfile: vi.fn(),
  mockUpdateSubscriptions: vi.fn(),
  mockInstallMapFiles: vi.fn(),
  mockInstallModFiles: vi.fn(),
  mockUninstallMapFiles: vi.fn(),
  mockUninstallModFiles: vi.fn(),
}));

vi.mock('../../wailsjs/go/registry/Registry', () => ({
  GetInstalledModsResponse: mockGetInstalledModsResponse,
  GetInstalledMapsResponse: mockGetInstalledMapsResponse,
}));

vi.mock('../../wailsjs/go/profiles/UserProfiles', () => ({
  GetActiveProfile: mockGetActiveProfile,
  UpdateSubscriptions: mockUpdateSubscriptions,
}));

vi.mock('../../wailsjs/go/downloader/Downloader', () => ({
  InstallMap: mockInstallMapFiles,
  InstallMod: mockInstallModFiles,
  UninstallMap: mockUninstallMapFiles,
  UninstallMod: mockUninstallModFiles,
}));

type ProfilesRequest = {
  profileId: string;
  action: 'subscribe' | 'unsubscribe';
  assetId: string;
  assetType: AssetType;
  version: string;
};

function validateProfilesRequest(expected: ProfilesRequest) {
  expect(mockUpdateSubscriptions).toHaveBeenCalledTimes(1);
  const request = mockUpdateSubscriptions.mock.calls[0][0];
  expect(request.profileId).toBe(expected.profileId);
  expect(request.action).toBe(expected.action);
  expect(request.forceSync).toBe(true);
  expect(request.assets[expected.assetId].type).toBe(expected.assetType);
  expect(request.assets[expected.assetId].version).toBe(expected.version);
}

function validateInstallationRefreshes(expectedCalls: number) {
  expect(mockGetInstalledModsResponse).toHaveBeenCalledTimes(expectedCalls);
  expect(mockGetInstalledMapsResponse).toHaveBeenCalledTimes(expectedCalls);
}

function validateFinalState(
  lane: 'installing' | 'uninstalling',
  assetId: string,
  error: string | null,
) {
  const state = useInstalledStore.getState();
  expect(state[lane].has(assetId)).toBe(false);
  if (error === null) {
    expect(state.error).toBeNull();
  } else {
    expect(state.error).toContain(error);
  }
}

describe('useInstalledStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useInstalledStore.setState({
      installedMods: [],
      installedMaps: [],
      installing: new Set<string>(),
      uninstalling: new Set<string>(),
      loading: false,
      error: null,
      initialized: false,
    });
    mockInstallMapFiles.mockResolvedValue({ status: 'success', message: '' });
    mockInstallModFiles.mockResolvedValue({ status: 'success', message: '' });
    mockUninstallMapFiles.mockResolvedValue({ status: 'success', message: '' });
    mockUninstallModFiles.mockResolvedValue({ status: 'success', message: '' });
  });

  it('installMap correctly updates subscriptions and refreshes installed lists', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockUpdateSubscriptions.mockResolvedValue(
      updateSubscriptionsSuccess('subscriptions updated'),
    );
    mockGetInstalledModsResponse.mockResolvedValue({
      status: 'success',
      message: 'ok',
      mods: [{ id: 'mod-1', version: '1.0.0' }],
    });
    mockGetInstalledMapsResponse.mockResolvedValue({
      status: 'success',
      message: 'ok',
      maps: [{ id: 'map-1', version: '2.0.0', config: { code: 'AAA' } }],
    });

    await useInstalledStore.getState().installMap('map-1', '2.0.0');

    validateProfilesRequest({
      profileId: 'profile-a',
      action: 'subscribe',
      assetId: 'map-1',
      assetType: 'map',
      version: '2.0.0',
    });
    validateInstallationRefreshes(1);
    validateFinalState('installing', 'map-1', null);
  });

  it('uninstallMap correctly updates subscriptions and refreshes installed lists on success', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockUpdateSubscriptions.mockResolvedValue(
      updateSubscriptionsSuccess('subscriptions updated'),
    );
    mockGetInstalledModsResponse.mockResolvedValue({
      status: 'success',
      message: 'ok',
      mods: [{ id: 'mod-1', version: '1.0.0' }],
    });
    mockGetInstalledMapsResponse.mockResolvedValue({
      status: 'success',
      message: 'ok',
      maps: [],
    });

    await useInstalledStore.getState().uninstallMap('map-7');

    validateProfilesRequest({
      profileId: 'profile-a',
      action: 'unsubscribe',
      assetId: 'map-7',
      assetType: 'map',
      version: '',
    });
    validateInstallationRefreshes(1);
    validateFinalState('uninstalling', 'map-7', null);
  });

  it('installMod errors when profile mutation fails', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockUpdateSubscriptions.mockResolvedValue(
      updateSubscriptionsError('Install failed'),
    );

    await expect(
      useInstalledStore.getState().installMod('mod-2', '1.2.3'),
    ).rejects.toThrow('Install failed');

    validateProfilesRequest({
      profileId: 'profile-a',
      action: 'subscribe',
      assetId: 'mod-2',
      assetType: 'mod',
      version: '1.2.3',
    });
    validateInstallationRefreshes(0);
    validateFinalState('installing', 'mod-2', 'Install failed');
  });

  it('installMap resolves when profile mutation returns warn', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockUpdateSubscriptions.mockResolvedValue(
      updateSubscriptionsWarn('sync completed with warnings'),
    );
    mockGetInstalledModsResponse.mockResolvedValue({
      status: 'success',
      message: 'ok',
      mods: [{ id: 'mod-1', version: '1.0.0' }],
    });
    mockGetInstalledMapsResponse.mockResolvedValue({
      status: 'success',
      message: 'ok',
      maps: [{ id: 'map-1', version: '2.0.0', config: { code: 'AAA' } }],
    });

    const result = await useInstalledStore
      .getState()
      .installMap('map-1', '2.0.0');

    validateProfilesRequest({
      profileId: 'profile-a',
      action: 'subscribe',
      assetId: 'map-1',
      assetType: 'map',
      version: '2.0.0',
    });
    validateInstallationRefreshes(1);
    validateFinalState('installing', 'map-1', null);
    expect(result.status).toBe('warn');
    expect(result.message).toContain('sync completed with warnings');
  });

  it('uninstallMod errors when profile mutation fails', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockUpdateSubscriptions.mockResolvedValue(
      updateSubscriptionsError('Uninstall failed'),
    );

    await expect(
      useInstalledStore.getState().uninstallMod('mod-9'),
    ).rejects.toThrow('Uninstall failed');

    validateProfilesRequest({
      profileId: 'profile-a',
      action: 'unsubscribe',
      assetId: 'mod-9',
      assetType: 'mod',
      version: '',
    });
    validateInstallationRefreshes(0);
    validateFinalState('uninstalling', 'mod-9', 'Uninstall failed');
  });

  it('cancelPendingInstall routes through unsubscribe and tolerates warn', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockUpdateSubscriptions.mockResolvedValue(
      updateSubscriptionsWarn('not installed; nothing to do'),
    );
    mockGetInstalledModsResponse.mockResolvedValue({
      status: 'success',
      message: 'ok',
      mods: [],
    });
    mockGetInstalledMapsResponse.mockResolvedValue({
      status: 'success',
      message: 'ok',
      maps: [],
    });

    const result = await useInstalledStore
      .getState()
      .cancelPendingInstall('map', 'map-42');

    validateProfilesRequest({
      profileId: 'profile-a',
      action: 'unsubscribe',
      assetId: 'map-42',
      assetType: 'map',
      version: '',
    });
    validateInstallationRefreshes(1);
    validateFinalState('uninstalling', 'map-42', null);
    expect(result.status).toBe('warn');
  });

  it('acknowledgeCancelledInstall removes item from installing lane idempotently', () => {
    useInstalledStore.setState((state) => ({
      ...state,
      installing: new Set(['map-1', 'map-2']),
    }));

    useInstalledStore.getState().acknowledgeCancelledInstall('map-1');
    expect(useInstalledStore.getState().installing.has('map-1')).toBe(false);
    expect(useInstalledStore.getState().installing.has('map-2')).toBe(true);

    useInstalledStore.getState().acknowledgeCancelledInstall('missing-map');
    expect(useInstalledStore.getState().installing.has('map-2')).toBe(true);
  });
});
