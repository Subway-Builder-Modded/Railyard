import { create } from "zustand";
import { types } from "../../wailsjs/go/models";
import {
  GetDownloadCountsByAssetType,
  GetMaps,
  GetMods,
  Refresh,
} from "../../wailsjs/go/registry/Registry";
import { ASSET_TYPES, type AssetType } from "@/lib/asset-types";
import { toCumulativeDownloadTotals } from "@/lib/download-totals";

interface RegistryState {
  mods: types.ModManifest[];
  maps: types.MapManifest[];
  modDownloadTotals: Record<string, number>;
  mapDownloadTotals: Record<string, number>;
  downloadTotalsLoaded: boolean;
  loading: boolean;
  refreshing: boolean;
  error: string | null;
  initialized: boolean;
  ensureDownloadTotals: () => Promise<void>;
  initialize: () => Promise<void>;
  refresh: () => Promise<void>;
}

let downloadTotalsRequest: Promise<void> | null = null;

function emptyRecordByAssetType<T>(factory: () => T): Record<AssetType, T> {
  return Object.fromEntries(
    ASSET_TYPES.map((assetType) => [assetType, factory()]),
  ) as Record<AssetType, T>;
}

export const useRegistryStore = create<RegistryState>((set, get) => ({
  mods: [],
  maps: [],
  modDownloadTotals: {},
  mapDownloadTotals: {},
  downloadTotalsLoaded: false,
  loading: false,
  refreshing: false,
  error: null,
  initialized: false,

  ensureDownloadTotals: async () => {
    if (get().downloadTotalsLoaded) return;
    if (downloadTotalsRequest) {
      await downloadTotalsRequest;
      return;
    }

    downloadTotalsRequest = (async () => {
      try {
        const results = await Promise.all(
          ASSET_TYPES.map((assetType) => GetDownloadCountsByAssetType(assetType)),
        );

        const totalsByAsset = emptyRecordByAssetType<Record<string, number>>(
          () => ({}),
        );

        results.forEach((result, index) => {
          const assetType = ASSET_TYPES[index];
          if (result.status === "success") {
            totalsByAsset[assetType] = toCumulativeDownloadTotals(result.counts);
            return;
          }
          console.warn(
            `[downloads:${assetType}] Failed to load download counts: ${result.message}`,
          );
        });

        set({
          modDownloadTotals: totalsByAsset.mod,
          mapDownloadTotals: totalsByAsset.map,
          downloadTotalsLoaded: true,
        });
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        console.warn(`[downloads] Failed to load download counts: ${message}`);
        set({
          modDownloadTotals: {},
          mapDownloadTotals: {},
          downloadTotalsLoaded: true,
        });
      } finally {
        downloadTotalsRequest = null;
      }
    })();

    await downloadTotalsRequest;
  },

  initialize: async () => {
    if (get().initialized) return;
    set({ loading: true, error: null });
    try {
      const [mods, maps] = await Promise.all([GetMods(), GetMaps()]);
      set({
        mods: mods || [],
        maps: maps || [],
        initialized: true,
        loading: false,
      });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err), loading: false });
    }
  },

  refresh: async () => {
    set({ refreshing: true, error: null });
    try {
      await Refresh();
      const [mods, maps] = await Promise.all([GetMods(), GetMaps()]);
      set({
        mods: mods || [],
        maps: maps || [],
        modDownloadTotals: {},
        mapDownloadTotals: {},
        downloadTotalsLoaded: false,
      });
      await get().ensureDownloadTotals();
      set({ refreshing: false });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err), refreshing: false });
    }
  },
}));
