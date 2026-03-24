import type { UpdateSubscriptionsRequestType } from '@/lib/profile-update-request-type';

import { types } from '../../../wailsjs/go/models';

const UPDATE_SUBSCRIPTIONS: UpdateSubscriptionsRequestType =
  'update_subscriptions';

export function activeProfileFixture(
  profileId: string = '__default__',
  existingSubscriptions: types.Subscriptions = {
    maps: {},
    localMaps: {},
    mods: {},
  },
): types.UserProfile {
  return new types.UserProfile({
    id: profileId,
    uuid: 'uuid',
    name: 'Default',
    uiPreferences: {
      theme: 'dark',
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

export function activeProfileResultSuccess(
  profileId: string = '__default__',
  existingSubscriptions: types.Subscriptions = {
    maps: {},
    localMaps: {},
    mods: {},
  },
): types.UserProfileResult {
  return new types.UserProfileResult({
    status: 'success',
    message: 'active profile resolved',
    profile: activeProfileFixture(profileId, existingSubscriptions),
    errors: [],
  });
}

export function activeProfileResultError(
  message: string,
): types.UserProfileResult {
  return new types.UserProfileResult({
    status: 'error',
    message,
    profile: activeProfileFixture(),
    errors: [],
  });
}

export function updateSubscriptionsSuccess(
  message: string = 'ok',
): types.UpdateSubscriptionsResult {
  return new types.UpdateSubscriptionsResult({
    status: 'success',
    message,
    requestType: UPDATE_SUBSCRIPTIONS,
    hasUpdates: false,
    pendingCount: 0,
    pendingUpdates: [],
    applied: true,
    profile: activeProfileFixture(),
    persisted: true,
    operations: [],
    errors: [],
    conflicts: [],
  });
}

export function updateSubscriptionsError(
  message: string,
): types.UpdateSubscriptionsResult {
  return new types.UpdateSubscriptionsResult({
    status: 'error',
    message,
    requestType: UPDATE_SUBSCRIPTIONS,
    hasUpdates: false,
    pendingCount: 0,
    pendingUpdates: [],
    applied: false,
    profile: activeProfileFixture(),
    persisted: false,
    operations: [],
    errors: [],
    conflicts: [],
  });
}

export function updateSubscriptionsWarn(
  message: string,
): types.UpdateSubscriptionsResult {
  return new types.UpdateSubscriptionsResult({
    status: 'warn',
    message,
    requestType: UPDATE_SUBSCRIPTIONS,
    hasUpdates: false,
    pendingCount: 0,
    pendingUpdates: [],
    applied: true,
    profile: activeProfileFixture(),
    persisted: true,
    operations: [],
    errors: [],
    conflicts: [],
  });
}

export function updateSubscriptionsWithConflicts(
  message: string,
  conflicts: types.MapCodeConflict[],
): types.UpdateSubscriptionsResult {
  return new types.UpdateSubscriptionsResult({
    status: 'warn',
    message,
    requestType: UPDATE_SUBSCRIPTIONS,
    hasUpdates: false,
    pendingCount: 0,
    pendingUpdates: [],
    applied: false,
    profile: activeProfileFixture(),
    persisted: false,
    operations: [],
    errors: [],
    conflicts,
  });
}
