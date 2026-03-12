import { beforeEach, describe, expect, it, vi } from "vitest";
import { useInstalledStore } from "./installed-store";
import { activeProfileResultSuccess, updateSubscriptionsError, updateSubscriptionsSuccess, updateSubscriptionsWarn } from "@/test/helpers/profileMutationFixtures";

const {
  mockGetInstalledMods,
  mockGetInstalledMaps,
  mockGetActiveProfile,
  mockUpdateSubscriptions,
  mockInstallMod,
  mockInstallMap,
  mockUninstallMod,
  mockUninstallMap,
} = vi.hoisted(() => ({
  mockGetInstalledMods: vi.fn(),
  mockGetInstalledMaps: vi.fn(),
  mockGetActiveProfile: vi.fn(),
  mockUpdateSubscriptions: vi.fn(),
  mockInstallMod: vi.fn(),
  mockInstallMap: vi.fn(),
  mockUninstallMod: vi.fn(),
  mockUninstallMap: vi.fn(),
}));

vi.mock("../../wailsjs/go/registry/Registry", () => ({
  GetInstalledMods: mockGetInstalledMods,
  GetInstalledMaps: mockGetInstalledMaps,
}));

vi.mock("../../wailsjs/go/profiles/UserProfiles", () => ({
  GetActiveProfile: mockGetActiveProfile,
  UpdateSubscriptions: mockUpdateSubscriptions,
}));

vi.mock("../../wailsjs/go/downloader/Downloader", () => ({
  InstallMod: mockInstallMod,
  InstallMap: mockInstallMap,
  UninstallMod: mockUninstallMod,
  UninstallMap: mockUninstallMap,
}));

type ProfilesRequest = {
  profileId: string;
  action: "subscribe" | "unsubscribe";
  assetId: string;
  assetType: "map" | "mod";
  version: string;
};

function validateProfilesRequest(expected: ProfilesRequest) {
  expect(mockUpdateSubscriptions).toHaveBeenCalledTimes(1);
  const request = mockUpdateSubscriptions.mock.calls[0][0];
  expect(request.profileId).toBe(expected.profileId);
  expect(request.action).toBe(expected.action);
  expect(request.forceSync).toBe(false);
  expect(request.assets[expected.assetId].type).toBe(expected.assetType);
  expect(request.assets[expected.assetId].version).toBe(expected.version);
}

function validateInstallationRefreshes(expectedCalls: number) {
  expect(mockGetInstalledMods).toHaveBeenCalledTimes(expectedCalls);
  expect(mockGetInstalledMaps).toHaveBeenCalledTimes(expectedCalls);
}

function validateFinalState(
  lane: "installing" | "uninstalling",
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

describe("useInstalledStore", () => {
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
  });

  it("installMap correctly updates subscriptions and refreshes installed lists", async () => {
    mockGetActiveProfile.mockResolvedValue(activeProfileResultSuccess("profile-a"));
    mockUpdateSubscriptions.mockResolvedValue(updateSubscriptionsSuccess("subscriptions updated"));
    mockInstallMap.mockResolvedValue({ status: "success" });
    mockGetInstalledMods.mockResolvedValue([{ id: "mod-1", version: "1.0.0" }]);
    mockGetInstalledMaps.mockResolvedValue([{ id: "map-1", version: "2.0.0", config: { code: "AAA" } }]);

    await useInstalledStore.getState().installMap("map-1", "2.0.0");

    validateProfilesRequest({
      profileId: "profile-a",
      action: "subscribe",
      assetId: "map-1",
      assetType: "map",
      version: "2.0.0",
    });
    validateInstallationRefreshes(1);
    validateFinalState("installing", "map-1", null);
  });

  it("uninstallMap correctly updates subscriptions and refreshes installed lists on success", async () => {
    mockUninstallMap.mockResolvedValue({ status: "success" });
    mockGetActiveProfile.mockResolvedValue(activeProfileResultSuccess("profile-a"));
    mockUpdateSubscriptions.mockResolvedValue(updateSubscriptionsSuccess("subscriptions updated"));
    mockGetInstalledMods.mockResolvedValue([{ id: "mod-1", version: "1.0.0" }]);
    mockGetInstalledMaps.mockResolvedValue([]);

    await useInstalledStore.getState().uninstallMap("map-7");

    validateProfilesRequest({
      profileId: "profile-a",
      action: "unsubscribe",
      assetId: "map-7",
      assetType: "map",
      version: "",
    });
    validateInstallationRefreshes(1);
    validateFinalState("uninstalling", "map-7", null);
  });

  it("installMod errors when profile mutation fails", async () => {
    mockGetActiveProfile.mockResolvedValue(activeProfileResultSuccess("profile-a"));
    mockUpdateSubscriptions.mockResolvedValue(updateSubscriptionsError("Install failed"));

    await expect(useInstalledStore.getState().installMod("mod-2", "1.2.3")).rejects.toThrow("Install failed");

    validateProfilesRequest({
      profileId: "profile-a",
      action: "subscribe",
      assetId: "mod-2",
      assetType: "mod",
      version: "1.2.3",
    });
    validateInstallationRefreshes(0);
    validateFinalState("installing", "mod-2", "Install failed");
  });

  it("installMap does not throw when profile mutation returns warn", async () => {
    mockGetActiveProfile.mockResolvedValue(activeProfileResultSuccess("profile-a"));
    mockUpdateSubscriptions.mockResolvedValue(updateSubscriptionsWarn("sync completed with warnings"));
    mockInstallMap.mockResolvedValue({ status: "success" });
    mockGetInstalledMods.mockResolvedValue([{ id: "mod-1", version: "1.0.0" }]);
    mockGetInstalledMaps.mockResolvedValue([{ id: "map-1", version: "2.0.0", config: { code: "AAA" } }]);

    await expect(useInstalledStore.getState().installMap("map-1", "2.0.0")).resolves.toBeDefined();

    validateProfilesRequest({
      profileId: "profile-a",
      action: "subscribe",
      assetId: "map-1",
      assetType: "map",
      version: "2.0.0",
    });
    validateInstallationRefreshes(1);
    validateFinalState("installing", "map-1", null);
  });

  it("uninstallMod errors when profile mutation fails", async () => {
    mockUninstallMod.mockResolvedValue({ status: "success" });
    mockGetActiveProfile.mockResolvedValue(activeProfileResultSuccess("profile-a"));
    mockUpdateSubscriptions.mockResolvedValue(updateSubscriptionsError("Uninstall failed"));

    await expect(useInstalledStore.getState().uninstallMod("mod-9")).rejects.toThrow("Uninstall failed");

    validateProfilesRequest({
      profileId: "profile-a",
      action: "unsubscribe",
      assetId: "mod-9",
      assetType: "mod",
      version: "",
    });
    validateInstallationRefreshes(0);
    validateFinalState("uninstalling", "mod-9", "Uninstall failed");
  });
});
