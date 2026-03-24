import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { AssetType } from '@/lib/asset-types';
import {
  activeProfileResultSuccess,
  updateSubscriptionsError,
  updateSubscriptionsSuccess,
  updateSubscriptionsWarn,
  updateSubscriptionsWithConflicts,
} from '@/test/helpers/profileMutationFixtures';

import { useDownloadQueueStore } from './download-queue-store';
import {
  AssetConflictError,
  InvalidMapCodeError,
  SubscriptionSyncError,
  useInstalledStore,
} from './installed-store';

const {
  mockGetInstalledModsResponse,
  mockGetInstalledMapsResponse,
  mockGetActiveProfile,
  mockUpdateSubscriptions,
  mockImportAsset,
  mockUpdateSubscriptionsToLatest,
} = vi.hoisted(() => ({
  mockGetInstalledModsResponse: vi.fn(),
  mockGetInstalledMapsResponse: vi.fn(),
  mockGetActiveProfile: vi.fn(),
  mockUpdateSubscriptions: vi.fn(),
  mockImportAsset: vi.fn(),
  mockUpdateSubscriptionsToLatest: vi.fn(),
}));

vi.mock('../../wailsjs/go/registry/Registry', () => ({
  GetInstalledModsResponse: mockGetInstalledModsResponse,
  GetInstalledMapsResponse: mockGetInstalledMapsResponse,
}));

vi.mock('../../wailsjs/go/profiles/UserProfiles', () => ({
  GetActiveProfile: mockGetActiveProfile,
  UpdateSubscriptions: mockUpdateSubscriptions,
  ImportAsset: mockImportAsset,
  UpdateSubscriptionsToLatest: mockUpdateSubscriptionsToLatest,
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

const installedListsSuccess = () => {
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
};

describe('useInstalledStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useInstalledStore.setState({
      installedMods: [],
      installedMaps: [],
      installing: new Set<string>(),
      installingVersionById: {},
      uninstalling: new Set<string>(),
      loading: false,
      error: null,
      initialized: false,
    });
    useDownloadQueueStore.setState({ total: 0, completed: 0 });
    mockUpdateSubscriptionsToLatest.mockResolvedValue(
      updateSubscriptionsSuccess('latest apply ok'),
    );
  });

  it('installMap correctly updates subscriptions and refreshes installed lists', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockUpdateSubscriptions.mockResolvedValue(
      updateSubscriptionsSuccess('subscriptions updated'),
    );
    installedListsSuccess();

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

  it('installMod throws SubscriptionSyncError when profile mutation fails', async () => {
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

  it('installMod throws SubscriptionSyncError instance on error', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockUpdateSubscriptions.mockResolvedValue(
      updateSubscriptionsError('Sync error'),
    );

    await expect(
      useInstalledStore.getState().installMod('mod-3', '1.0.0'),
    ).rejects.toBeInstanceOf(SubscriptionSyncError);
  });

  it('installMap resolves when profile mutation returns warn', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockUpdateSubscriptions.mockResolvedValue(
      updateSubscriptionsWarn('sync completed with warnings'),
    );
    installedListsSuccess();

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

  it('installMap throws AssetConflictError when warn has conflicts', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockUpdateSubscriptions.mockResolvedValue(
      updateSubscriptionsWithConflicts('Map code conflict', [
        { existingMapId: 'map-other', incomingMapId: 'map-1', code: 'ABC' },
      ]),
    );

    await expect(
      useInstalledStore.getState().installMap('map-1', '2.0.0'),
    ).rejects.toBeInstanceOf(AssetConflictError);

    validateFinalState('installing', 'map-1', 'Map code conflict');
  });

  it('uninstallMod throws SubscriptionSyncError when profile mutation fails', async () => {
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
      installingVersionById: { 'map-1': '1.0.0', 'map-2': '2.0.0' },
    }));

    useInstalledStore.getState().acknowledgeCancelledInstall('map-1');
    expect(useInstalledStore.getState().installing.has('map-1')).toBe(false);
    expect(
      useInstalledStore.getState().getInstallingVersion('map-1'),
    ).toBeNull();
    expect(useInstalledStore.getState().installing.has('map-2')).toBe(true);
    expect(useInstalledStore.getState().getInstallingVersion('map-2')).toBe(
      '2.0.0',
    );

    useInstalledStore.getState().acknowledgeCancelledInstall('missing-map');
    expect(useInstalledStore.getState().installing.has('map-2')).toBe(true);
  });

  it('uninstallAssets removes multiple assets in one request', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockUpdateSubscriptions.mockResolvedValue(
      updateSubscriptionsSuccess('batch uninstall ok'),
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

    const result = await useInstalledStore.getState().uninstallAssets([
      { id: 'mod-1', type: 'mod' },
      { id: 'map-1', type: 'map' },
    ]);

    expect(mockUpdateSubscriptions).toHaveBeenCalledTimes(1);
    const request = mockUpdateSubscriptions.mock.calls[0][0];
    expect(request.action).toBe('unsubscribe');
    expect(request.assets['mod-1'].type).toBe('mod');
    expect(request.assets['map-1'].type).toBe('map');
    validateInstallationRefreshes(1);
    expect(result.status).toBe('success');
    expect(useInstalledStore.getState().uninstalling.has('mod-1')).toBe(false);
    expect(useInstalledStore.getState().uninstalling.has('map-1')).toBe(false);
  });

  it('updateAssetsToLatest invokes latest_apply with scoped targets and refreshes lists', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    installedListsSuccess();

    const result = await useInstalledStore.getState().updateAssetsToLatest([
      { id: 'map-1', type: 'map' },
      { id: 'mod-1', type: 'mod' },
    ]);

    expect(mockUpdateSubscriptionsToLatest).toHaveBeenCalledTimes(1);
    const request = mockUpdateSubscriptionsToLatest.mock.calls[0][0];
    expect(request.profileId).toBe('profile-a');
    expect(request.apply).toBe(true);
    expect(request.targets).toEqual([
      { assetId: 'map-1', type: 'map' },
      { assetId: 'mod-1', type: 'mod' },
    ]);
    validateInstallationRefreshes(1);
    expect(useInstalledStore.getState().installing.has('map-1')).toBe(false);
    expect(useInstalledStore.getState().installing.has('mod-1')).toBe(false);
    expect(result.status).toBe('success');
  });

  it('updateAssetsToLatest surfaces latest_apply errors', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockUpdateSubscriptionsToLatest.mockResolvedValue(
      updateSubscriptionsError('Failed to apply latest updates'),
    );

    await expect(
      useInstalledStore
        .getState()
        .updateAssetsToLatest([{ id: 'map-1', type: 'map' }]),
    ).rejects.toThrow('Failed to apply latest updates');
    expect(useInstalledStore.getState().installing.has('map-1')).toBe(false);
  });

  it('importMapFromZip succeeds and refreshes installed lists', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockImportAsset.mockResolvedValue({
      status: 'success',
      message: 'imported ok',
      conflicts: [],
      errors: [],
    });
    installedListsSuccess();

    const result = await useInstalledStore
      .getState()
      .importMapFromZip('/path/to/map.zip');

    expect(mockImportAsset).toHaveBeenCalledTimes(1);
    const request = mockImportAsset.mock.calls[0][0];
    expect(request.profileId).toBe('profile-a');
    expect(request.assetType).toBe('map');
    expect(request.zipPath).toBe('/path/to/map.zip');
    expect(request.replaceOnConflict).toBe(false);
    validateInstallationRefreshes(1);
    expect(result.status).toBe('success');
    expect(useInstalledStore.getState().error).toBeNull();
  });

  it('importMapFromZip throws AssetConflictError on warn with conflicts', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockImportAsset.mockResolvedValue({
      status: 'warn',
      message: 'conflict detected',
      conflicts: [
        { existingMapId: 'map-other', incomingMapId: 'map-1', code: 'ABC' },
      ],
      errors: [],
    });

    await expect(
      useInstalledStore.getState().importMapFromZip('/path/to/map.zip'),
    ).rejects.toBeInstanceOf(AssetConflictError);
  });

  it('importMapFromZip throws InvalidMapCodeError on invalid map code error', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockImportAsset.mockResolvedValue({
      status: 'error',
      message: 'Invalid map code',
      conflicts: [],
      errors: [
        {
          downloaderErrorType: 'install_invalid_map_code',
          message: 'bad code',
        },
      ],
    });

    await expect(
      useInstalledStore.getState().importMapFromZip('/path/to/map.zip'),
    ).rejects.toBeInstanceOf(InvalidMapCodeError);
  });

  it('importMapFromZip throws SubscriptionSyncError on generic error', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockImportAsset.mockResolvedValue({
      status: 'error',
      message: 'Import failed',
      conflicts: [],
      errors: [],
    });

    await expect(
      useInstalledStore.getState().importMapFromZip('/path/to/map.zip'),
    ).rejects.toBeInstanceOf(SubscriptionSyncError);
  });

  it('importMapFromZip passes replaceOnConflict flag to the request', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );
    mockImportAsset.mockResolvedValue({
      status: 'success',
      message: 'ok',
      conflicts: [],
      errors: [],
    });
    installedListsSuccess();

    await useInstalledStore
      .getState()
      .importMapFromZip('/path/to/map.zip', true);

    const request = mockImportAsset.mock.calls[0][0];
    expect(request.replaceOnConflict).toBe(true);
  });

  it('tracks installingVersion for the duration of an install', async () => {
    mockGetActiveProfile.mockResolvedValue(
      activeProfileResultSuccess('profile-a'),
    );

    let versionDuringInstall: string | null = null;

    mockUpdateSubscriptions.mockImplementation(async () => {
      versionDuringInstall = useInstalledStore
        .getState()
        .getInstallingVersion('mod-5');
      return updateSubscriptionsSuccess('ok');
    });
    mockGetInstalledModsResponse.mockResolvedValue({
      status: 'success',
      message: 'ok',
      mods: [{ id: 'mod-5', version: '3.0.0' }],
    });
    mockGetInstalledMapsResponse.mockResolvedValue({
      status: 'success',
      message: 'ok',
      maps: [],
    });

    await useInstalledStore.getState().installMod('mod-5', '3.0.0');

    expect(versionDuringInstall).toBe('3.0.0');
    expect(
      useInstalledStore.getState().getInstallingVersion('mod-5'),
    ).toBeNull();
  });
});
