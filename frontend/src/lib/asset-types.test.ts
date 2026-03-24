import { describe, expect, it } from 'vitest';

import {
  ASSET_TYPES,
  assetTypeToListingPath,
  listingPathToAssetType,
} from './asset-types';

describe('ASSET_TYPES', () => {
  it('contains mod and map', () => {
    expect(ASSET_TYPES).toContain('mod');
    expect(ASSET_TYPES).toContain('map');
    expect(ASSET_TYPES).toHaveLength(2);
  });
});

describe('assetTypeToListingPath', () => {
  it('maps mod to mods', () => {
    expect(assetTypeToListingPath('mod')).toBe('mods');
  });

  it('maps map to maps', () => {
    expect(assetTypeToListingPath('map')).toBe('maps');
  });
});

describe('listingPathToAssetType', () => {
  it('maps mods to mod', () => {
    expect(listingPathToAssetType('mods')).toBe('mod');
  });

  it('maps maps to map', () => {
    expect(listingPathToAssetType('maps')).toBe('map');
  });

  it('returns undefined for unknown paths', () => {
    expect(listingPathToAssetType('unknown')).toBeUndefined();
    expect(listingPathToAssetType('')).toBeUndefined();
  });
});
