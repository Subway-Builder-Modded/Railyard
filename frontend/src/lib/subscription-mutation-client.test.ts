import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useGameStore } from '@/stores/game-store';

const {
  mockUpdateSubscriptions,
  mockUpdateSubscriptionsToLatest,
  mockImportAsset,
  mockResolveActiveProfileID,
  mockToLatestUpdateRequestTargets,
} = vi.hoisted(() => ({
  mockUpdateSubscriptions: vi.fn(),
  mockUpdateSubscriptionsToLatest: vi.fn(),
  mockImportAsset: vi.fn(),
  mockResolveActiveProfileID: vi.fn(),
  mockToLatestUpdateRequestTargets: vi.fn(),
}));

vi.mock('../../wailsjs/go/profiles/UserProfiles', () => ({
  UpdateSubscriptions: mockUpdateSubscriptions,
  UpdateSubscriptionsToLatest: mockUpdateSubscriptionsToLatest,
  ImportAsset: mockImportAsset,
}));

vi.mock('@/lib/subscription-updates', () => ({
  resolveActiveProfileID: mockResolveActiveProfileID,
  toLatestUpdateRequestTargets: mockToLatestUpdateRequestTargets,
}));

import {
  applyLatestSubscriptionUpdatesForActiveProfile,
  importAssetForActiveProfile,
  isSubscriptionMutationLockedError,
  mutateSubscriptionsForActiveProfile,
  SubscriptionMutationLockedError,
} from '@/lib/subscription-mutation-client';

import { types } from '../../wailsjs/go/models';

describe('subscription-mutation-client', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useGameStore.setState({ running: false });
    mockResolveActiveProfileID.mockResolvedValue('profile-a');
    mockToLatestUpdateRequestTargets.mockReturnValue([
      { assetId: 'map-1', type: 'map' },
    ]);
  });

  it('blocks mutation calls while game is running', async () => {
    useGameStore.setState({ running: true });

    await expect(
      mutateSubscriptionsForActiveProfile({
        assets: {
          'map-1': new types.SubscriptionUpdateItem({
            type: 'map',
            version: 'v1.0.0',
          }),
        },
        action: 'subscribe',
      }),
    ).rejects.toBeInstanceOf(SubscriptionMutationLockedError);

    expect(mockUpdateSubscriptions).not.toHaveBeenCalled();
    expect(mockResolveActiveProfileID).not.toHaveBeenCalled();
  });

  it('delegates unlocked mutation calls to UpdateSubscriptions', async () => {
    mockUpdateSubscriptions.mockResolvedValue({
      status: 'success',
      message: 'ok',
    });

    await mutateSubscriptionsForActiveProfile({
      assets: {
        'map-1': new types.SubscriptionUpdateItem({
          type: 'map',
          version: 'v1.0.0',
        }),
      },
      action: 'subscribe',
      replaceOnConflict: true,
    });

    expect(mockResolveActiveProfileID).toHaveBeenCalledTimes(1);
    expect(mockUpdateSubscriptions).toHaveBeenCalledTimes(1);
    const request = mockUpdateSubscriptions.mock.calls[0][0];
    expect(request.profileId).toBe('profile-a');
    expect(request.action).toBe('subscribe');
    expect(request.applyMode).toBe('persist_and_sync');
    expect(request.replaceOnConflict).toBe(true);
  });

  it('delegates unlocked latest apply and import calls', async () => {
    mockUpdateSubscriptionsToLatest.mockResolvedValue({
      status: 'success',
      message: 'ok',
    });
    mockImportAsset.mockResolvedValue({
      status: 'success',
      message: 'ok',
    });

    await applyLatestSubscriptionUpdatesForActiveProfile({
      targets: [{ id: 'map-1', type: 'map' }],
    });
    await importAssetForActiveProfile({
      assetType: 'map',
      zipPath: '/tmp/map.zip',
      replaceOnConflict: false,
    });

    expect(mockUpdateSubscriptionsToLatest).toHaveBeenCalledTimes(1);
    const latestRequest = mockUpdateSubscriptionsToLatest.mock.calls[0][0];
    expect(latestRequest.profileId).toBe('profile-a');
    expect(latestRequest.apply).toBe(true);
    expect(latestRequest.targets).toEqual([{ assetId: 'map-1', type: 'map' }]);

    expect(mockImportAsset).toHaveBeenCalledTimes(1);
    const importRequest = mockImportAsset.mock.calls[0][0];
    expect(importRequest.profileId).toBe('profile-a');
    expect(importRequest.assetType).toBe('map');
    expect(importRequest.zipPath).toBe('/tmp/map.zip');
  });

  it('detects typed lock errors by class and code shape', () => {
    expect(
      isSubscriptionMutationLockedError(new SubscriptionMutationLockedError()),
    ).toBe(true);
    expect(
      isSubscriptionMutationLockedError({
        code: 'subscription_mutation_locked',
        message: 'Cannot modify subscriptions while the game is running.',
      }),
    ).toBe(true);
    expect(
      isSubscriptionMutationLockedError({
        code: 'different_code',
        message: 'nope',
      }),
    ).toBe(false);
  });
});
