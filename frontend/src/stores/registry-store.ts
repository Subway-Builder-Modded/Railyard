import { create } from "zustand";
import { types } from "../../wailsjs/go/models";
import {
  GetDownloadCountsByAssetType,
  GetMaps,
  GetMods,
  Refresh,
} from "../../wailsjs/go/registry/Registry";
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

let requests: Promise<void> | null = null;

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

  // Ensure that download totals are loaded on Browse page, but only fetch them when needed
  ensureDownloadTotals: async () => {
    if (get().downloadTotalsLoaded) return;
    if (requests) {
      await requests;
      return;
    }

    requests = (async () => {
      try {
        // Request download counts for each asset type in parallel
        const [modCountsResult, mapCountsResult] = await Promise.all([
          GetDownloadCountsByAssetType("mod"),
          GetDownloadCountsByAssetType("map"),
        ]);

        // TODO: Make this iterate over all asset types in the future
        const modDownloadTotals =
          modCountsResult.status === "success"
            ? toCumulativeDownloadTotals(modCountsResult.counts)
            : {};
        if (modCountsResult.status !== "success") {
          console.warn(
            `[downloads:mod] Failed to load download counts: ${modCountsResult.message}`
          );
        }

        const mapDownloadTotals =
          mapCountsResult.status === "success"
            ? toCumulativeDownloadTotals(mapCountsResult.counts)
            : {};
        if (mapCountsResult.status !== "success") {
          console.warn(
            `[downloads:map] Failed to load download counts: ${mapCountsResult.message}`
          );
        }

        set({
          modDownloadTotals,
          mapDownloadTotals,
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
        requests = null;
      }
    })();

    await requests;
  },

  // On initial load, fetch mod and map manifests
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

  // On refresh, re-fetch mod and map manifests and reset download totals so they will be re-fetched on demand with ensureDownloadTotals (to avoid stale data)
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
