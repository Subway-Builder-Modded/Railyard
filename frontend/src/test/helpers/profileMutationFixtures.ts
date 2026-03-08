import { types } from "../../../wailsjs/go/models";

export function activeProfileFixture(profileId: string = "__default__", existingSubscriptions: types.Subscriptions = { maps: {}, mods: {} }): types.UserProfile {
  return new types.UserProfile({
    id: profileId,
    uuid: "uuid",
    name: "Default",
    uiPreferences: {
      theme: "dark",
      defaultPerPage: 12,
    },
    systemPreferences: {
      refreshRegistryOnStartup: true,
    },
    favorites: {
      authors: [],
      maps: [],
      mods: [],
    },
    subscriptions: existingSubscriptions,
  });
}

export function updateSubscriptionsSuccess(message: string = "ok"): types.UpdateSubscriptionsResult {
  return new types.UpdateSubscriptionsResult({
    status: "success",
    message,
    profile: activeProfileFixture(),
    persisted: true,
    operations: [],
  });
}

export function updateSubscriptionsError(message: string): types.UpdateSubscriptionsResult {
  return new types.UpdateSubscriptionsResult({
    status: "error",
    message,
    profile: activeProfileFixture(),
    persisted: false,
    operations: [],
  });
}
