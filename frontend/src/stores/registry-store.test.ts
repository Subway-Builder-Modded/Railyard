import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useRegistryStore } from './registry-store';

const {
  mockGetModsResponse,
  mockGetMapsResponse,
  mockGetIntegrityReportResponse,
  mockRefreshResponse,
  mockGetDownloadCountsByAssetType,
} = vi.hoisted(() => ({
  mockGetModsResponse: vi.fn(),
  mockGetMapsResponse: vi.fn(),
  mockGetIntegrityReportResponse: vi.fn(),
  mockRefreshResponse: vi.fn(),
  mockGetDownloadCountsByAssetType: vi.fn(),
}));

vi.mock('../../wailsjs/go/registry/Registry', () => ({
  GetModsResponse: mockGetModsResponse,
  GetMapsResponse: mockGetMapsResponse,
  GetIntegrityReportResponse: mockGetIntegrityReportResponse,
  RefreshResponse: mockRefreshResponse,
  GetDownloadCountsByAssetType: mockGetDownloadCountsByAssetType,
}));

describe('useRegistryStore download totals', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    useRegistryStore.setState({
      mods: [],
      maps: [],
      mapIntegrity: null,
      modIntegrity: null,
      modDownloadTotals: {},
      mapDownloadTotals: {},
      downloadTotalsLoaded: false,
      loading: false,
      refreshing: false,
      error: null,
      initialized: false,
    });
  });

  it('loads and caches cumulative totals by asset type', async () => {
    mockGetDownloadCountsByAssetType
      .mockResolvedValueOnce({
        status: 'success',
        message: 'ok',
        assetType: 'mod',
        counts: {
          mod_a: { '1.0.0': 2, '1.1.0': 3 },
          mod_b: { '2.0.0': 7 },
        },
      })
      .mockResolvedValueOnce({
        status: 'success',
        message: 'ok',
        assetType: 'map',
        counts: {
          map_a: { '1.0.0': 11 },
        },
      });

    await useRegistryStore.getState().ensureDownloadTotals();

    const state = useRegistryStore.getState();
    expect(mockGetDownloadCountsByAssetType).toHaveBeenCalledTimes(2);
    expect(mockGetDownloadCountsByAssetType).toHaveBeenNthCalledWith(1, 'mod');
    expect(mockGetDownloadCountsByAssetType).toHaveBeenNthCalledWith(2, 'map');
    expect(state.modDownloadTotals).toEqual({ mod_a: 5, mod_b: 7 });
    expect(state.mapDownloadTotals).toEqual({ map_a: 11 });
    expect(state.downloadTotalsLoaded).toBe(true);
  });

  it('keeps zero/default totals on non-success responses', async () => {
    mockGetDownloadCountsByAssetType
      .mockResolvedValueOnce({
        status: 'error',
        message: 'failed',
        assetType: 'mod',
        counts: {},
      })
      .mockResolvedValueOnce({
        status: 'warn',
        message: 'partial',
        assetType: 'map',
        counts: {},
      });

    await useRegistryStore.getState().ensureDownloadTotals();

    const state = useRegistryStore.getState();
    expect(state.modDownloadTotals).toEqual({});
    expect(state.mapDownloadTotals).toEqual({});
    expect(state.downloadTotalsLoaded).toBe(true);
  });

  it('deduplicates concurrent totals loads with an in-flight promise', async () => {
    mockGetDownloadCountsByAssetType
      .mockResolvedValueOnce({
        status: 'success',
        message: 'ok',
        assetType: 'mod',
        counts: { mod_a: { '1.0.0': 1 } },
      })
      .mockResolvedValueOnce({
        status: 'success',
        message: 'ok',
        assetType: 'map',
        counts: { map_a: { '1.0.0': 2 } },
      });

    await Promise.all([
      useRegistryStore.getState().ensureDownloadTotals(),
      useRegistryStore.getState().ensureDownloadTotals(),
      useRegistryStore.getState().ensureDownloadTotals(),
    ]);

    expect(mockGetDownloadCountsByAssetType).toHaveBeenCalledTimes(2);
    expect(useRegistryStore.getState().downloadTotalsLoaded).toBe(true);
  });

  it('skips re-fetching when totals are already loaded', async () => {
    useRegistryStore.setState({ downloadTotalsLoaded: true });
    await useRegistryStore.getState().ensureDownloadTotals();
    expect(mockGetDownloadCountsByAssetType).not.toHaveBeenCalled();
  });

  it('recomputes totals during refresh', async () => {
    mockRefreshResponse.mockResolvedValue({ status: 'success', message: 'ok' });
    mockGetModsResponse.mockResolvedValue({
      status: 'success',
      message: 'ok',
      mods: [],
    });
    mockGetMapsResponse.mockResolvedValue({
      status: 'success',
      message: 'ok',
      maps: [],
    });
    mockGetIntegrityReportResponse
      .mockResolvedValueOnce({
        status: 'success',
        message: 'ok',
        report: { listings: {} },
      })
      .mockResolvedValueOnce({
        status: 'success',
        message: 'ok',
        report: { listings: {} },
      });
    mockGetDownloadCountsByAssetType
      .mockResolvedValueOnce({
        status: 'success',
        message: 'ok',
        assetType: 'mod',
        counts: { mod_c: { '1.0.0': 9 } },
      })
      .mockResolvedValueOnce({
        status: 'success',
        message: 'ok',
        assetType: 'map',
        counts: { map_c: { '1.0.0': 4, '1.1.0': 6 } },
      });

    await useRegistryStore.getState().refresh();

    const state = useRegistryStore.getState();
    expect(mockRefreshResponse).toHaveBeenCalledTimes(1);
    expect(mockGetModsResponse).toHaveBeenCalledTimes(1);
    expect(mockGetMapsResponse).toHaveBeenCalledTimes(1);
    expect(mockGetIntegrityReportResponse).toHaveBeenCalledTimes(2);
    expect(mockGetIntegrityReportResponse).toHaveBeenNthCalledWith(1, 'map');
    expect(mockGetIntegrityReportResponse).toHaveBeenNthCalledWith(2, 'mod');
    expect(state.modDownloadTotals).toEqual({ mod_c: 9 });
    expect(state.mapDownloadTotals).toEqual({ map_c: 10 });
    expect(state.downloadTotalsLoaded).toBe(true);
    expect(state.refreshing).toBe(false);
  });
});
