import { describe, expect, it } from 'vitest';

import {
  composeAssetKey,
  getPendingSubscriptionUpdate,
  indexPendingSubscriptionUpdates,
  toLatestUpdateRequestTargets,
  toPendingUpdateTargets,
} from '@/lib/subscription-updates';

import { types } from '../../wailsjs/go/models';

describe('subscription-updates helpers', () => {
  it('indexes pending updates by asset key', () => {
    const pending = indexPendingSubscriptionUpdates([
      {
        assetId: 'map-a',
        type: 'map',
        currentVersion: '1.0.0',
        latestVersion: '1.2.0',
      } as types.PendingSubscriptionUpdate,
      {
        assetId: 'mod-a',
        type: 'mod',
        currentVersion: '2.0.0',
        latestVersion: '2.1.0',
      } as types.PendingSubscriptionUpdate,
    ]);

    expect(Object.keys(pending)).toEqual(['map-map-a', 'mod-mod-a']);
    expect(pending['map-map-a']?.latestVersion).toBe('1.2.0');
    expect(pending['mod-mod-a']?.latestVersion).toBe('2.1.0');
  });

  it('returns empty object for undefined or empty input', () => {
    expect(indexPendingSubscriptionUpdates(undefined)).toEqual({});
    expect(indexPendingSubscriptionUpdates([])).toEqual({});
  });

  it('derives only updateable targets from selection', () => {
    const pending = indexPendingSubscriptionUpdates([
      {
        assetId: 'map-a',
        type: 'map',
        currentVersion: '1.0.0',
        latestVersion: '1.2.0',
      } as types.PendingSubscriptionUpdate,
    ]);

    const targets = toPendingUpdateTargets(
      [
        { id: 'map-a', type: 'map', name: 'Map A' },
        { id: 'mod-a', type: 'mod', name: 'Mod A' },
      ],
      pending,
    );

    expect(targets).toEqual([
      {
        id: 'map-a',
        type: 'map',
        name: 'Map A',
        currentVersion: '1.0.0',
        latestVersion: '1.2.0',
      },
    ]);
  });

  it('builds scoped latest-update request targets', () => {
    const requestTargets = toLatestUpdateRequestTargets([
      { id: 'map-a', type: 'map' },
      { id: 'mod-b', type: 'mod' },
    ]);

    expect(requestTargets).toEqual([
      new types.SubscriptionUpdateTarget({ assetId: 'map-a', type: 'map' }),
      new types.SubscriptionUpdateTarget({ assetId: 'mod-b', type: 'mod' }),
    ]);
  });

  it('composes stable asset keys', () => {
    expect(composeAssetKey('map', 'map-a')).toBe('map-map-a');
    expect(composeAssetKey('mod', 'mod-x')).toBe('mod-mod-x');
  });

  it('looks up a pending update by type and id', () => {
    const pending = indexPendingSubscriptionUpdates([
      {
        assetId: 'map-a',
        type: 'map',
        currentVersion: '1.0.0',
        latestVersion: '1.2.0',
      } as types.PendingSubscriptionUpdate,
    ]);

    const found = getPendingSubscriptionUpdate(pending, 'map', 'map-a');
    const wrongType = getPendingSubscriptionUpdate(pending, 'mod', 'map-a');
    const wrongId = getPendingSubscriptionUpdate(pending, 'map', 'map-b');

    expect(found?.latestVersion).toBe('1.2.0');
    expect(wrongType).toBeUndefined();
    expect(wrongId).toBeUndefined();
  });
});
