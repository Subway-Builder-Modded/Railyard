import { beforeEach, describe, expect, it, vi } from "vitest";
import { useRegistryStore } from "./registry-store";

const {
  mockGetMods,
  mockGetMaps,
  mockRefresh,
  mockGetDownloadCountsByAssetType,
} = vi.hoisted(() => ({
  mockGetMods: vi.fn(),
  mockGetMaps: vi.fn(),
  mockRefresh: vi.fn(),
  mockGetDownloadCountsByAssetType: vi.fn(),
}));

vi.mock("../../wailsjs/go/registry/Registry", () => ({
  GetMods: mockGetMods,
  GetMaps: mockGetMaps,
  Refresh: mockRefresh,
  GetDownloadCountsByAssetType: mockGetDownloadCountsByAssetType,
}));

describe("useRegistryStore download totals", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, "warn").mockImplementation(() => {});
    useRegistryStore.setState({
      mods: [],
      maps: [],
      modDownloadTotals: {},
      mapDownloadTotals: {},
      downloadTotalsLoaded: false,
      loading: false,
      refreshing: false,
      error: null,
      initialized: false,
    });
  });

  it("loads and caches cumulative totals by asset type", async () => {
    mockGetDownloadCountsByAssetType
      .mockResolvedValueOnce({
        status: "success",
        message: "ok",
        assetType: "mod",
        counts: {
          mod_a: { "1.0.0": 2, "1.1.0": 3 },
          mod_b: { "2.0.0": 7 },
        },
      })
      .mockResolvedValueOnce({
        status: "success",
        message: "ok",
        assetType: "map",
        counts: {
          map_a: { "1.0.0": 11 },
        },
      });

    await useRegistryStore.getState().ensureDownloadTotals();

    const state = useRegistryStore.getState();
    expect(mockGetDownloadCountsByAssetType).toHaveBeenCalledTimes(2);
    expect(mockGetDownloadCountsByAssetType).toHaveBeenNthCalledWith(1, "mod");
    expect(mockGetDownloadCountsByAssetType).toHaveBeenNthCalledWith(2, "map");
    expect(state.modDownloadTotals).toEqual({ mod_a: 5, mod_b: 7 });
    expect(state.mapDownloadTotals).toEqual({ map_a: 11 });
    expect(state.downloadTotalsLoaded).toBe(true);
  });

  it("keeps zero/default totals on non-success responses", async () => {
    mockGetDownloadCountsByAssetType
      .mockResolvedValueOnce({
        status: "error",
        message: "failed",
        assetType: "mod",
        counts: {},
      })
      .mockResolvedValueOnce({
        status: "warn",
        message: "partial",
        assetType: "map",
        counts: {},
      });

    await useRegistryStore.getState().ensureDownloadTotals();

    const state = useRegistryStore.getState();
    expect(state.modDownloadTotals).toEqual({});
    expect(state.mapDownloadTotals).toEqual({});
    expect(state.downloadTotalsLoaded).toBe(true);
  });

  it("deduplicates concurrent totals loads with an in-flight promise", async () => {
    mockGetDownloadCountsByAssetType
      .mockResolvedValueOnce({
        status: "success",
        message: "ok",
        assetType: "mod",
        counts: { mod_a: { "1.0.0": 1 } },
      })
      .mockResolvedValueOnce({
        status: "success",
        message: "ok",
        assetType: "map",
        counts: { map_a: { "1.0.0": 2 } },
      });

    await Promise.all([
      useRegistryStore.getState().ensureDownloadTotals(),
      useRegistryStore.getState().ensureDownloadTotals(),
      useRegistryStore.getState().ensureDownloadTotals(),
    ]);

    expect(mockGetDownloadCountsByAssetType).toHaveBeenCalledTimes(2);
    expect(useRegistryStore.getState().downloadTotalsLoaded).toBe(true);
  });

  it("recomputes totals during refresh", async () => {
    mockRefresh.mockResolvedValue(undefined);
    mockGetMods.mockResolvedValue([]);
    mockGetMaps.mockResolvedValue([]);
    mockGetDownloadCountsByAssetType
      .mockResolvedValueOnce({
        status: "success",
        message: "ok",
        assetType: "mod",
        counts: { mod_c: { "1.0.0": 9 } },
      })
      .mockResolvedValueOnce({
        status: "success",
        message: "ok",
        assetType: "map",
        counts: { map_c: { "1.0.0": 4, "1.1.0": 6 } },
      });

    await useRegistryStore.getState().refresh();

    const state = useRegistryStore.getState();
    expect(mockRefresh).toHaveBeenCalledTimes(1);
    expect(mockGetMods).toHaveBeenCalledTimes(1);
    expect(mockGetMaps).toHaveBeenCalledTimes(1);
    expect(state.modDownloadTotals).toEqual({ mod_c: 9 });
    expect(state.mapDownloadTotals).toEqual({ map_c: 10 });
    expect(state.downloadTotalsLoaded).toBe(true);
    expect(state.refreshing).toBe(false);
  });
});
