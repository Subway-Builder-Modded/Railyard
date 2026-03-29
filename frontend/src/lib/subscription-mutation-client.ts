import type { AssetType } from '@/lib/asset-types';
import {
  isSubscriptionMutationLocked,
  isSubscriptionMutationLockedError as isSubscriptionMutationLockedErrorLike,
  SUBSCRIPTION_MUTATION_LOCK_ERROR_CODE,
  SUBSCRIPTION_MUTATION_LOCK_MESSAGE,
  type SubscriptionMutationLockedErrorLike,
} from '@/lib/subscription-mutation-lock';
import {
  resolveActiveProfileID,
  toLatestUpdateRequestTargets,
} from '@/lib/subscription-updates';
import { useGameStore } from '@/stores/game-store';

import { types } from '../../wailsjs/go/models';
import {
  ImportAsset,
  UpdateSubscriptions,
  UpdateSubscriptionsToLatest,
} from '../../wailsjs/go/profiles/UserProfiles';

/**
 * This module provides a thin wrapper over the Wails-generated client functions for subscription mutations, adding error handling for mutation locks and ensuring that the active profile ID is resolved before making requests. 
 * 
 * This wrapper exists to centralize gating of subscription mutations based on the current game state. Direct calls to UserProfiles wails functions will be considered invalid as part of the ESLint rules.
 */

export class SubscriptionMutationLockedError extends Error {
  readonly code = SUBSCRIPTION_MUTATION_LOCK_ERROR_CODE;

  constructor(message = SUBSCRIPTION_MUTATION_LOCK_MESSAGE) {
    super(message);
    this.name = 'SubscriptionMutationLockedError';
  }
}

export {
  SUBSCRIPTION_MUTATION_LOCK_ERROR_CODE,
  SUBSCRIPTION_MUTATION_LOCK_MESSAGE,
};

// isSubscriptionMutationLockedError is a helper type guard that checks if a given error is a SubscriptionMutationLockedError, either by instance or by shape.
export function isSubscriptionMutationLockedError(
  error: unknown,
): error is SubscriptionMutationLockedErrorLike {
  if (error instanceof SubscriptionMutationLockedError) {
    return true;
  }
  return isSubscriptionMutationLockedErrorLike(error);
}

// ensureSubscriptionMutationUnlocked checks if subscription mutations are currently locked based on the game state, and throws a SubscriptionMutationLockedError if the game is currently running. 
// This is used to prevent destructive subscription mutations that may adversely affect game state
function ensureSubscriptionMutationUnlocked() {
  if (isSubscriptionMutationLocked(useGameStore.getState().running)) {
    throw new SubscriptionMutationLockedError();
  }
}

// mutateSubscriptionsForActiveProfile is a wrapper around the UpdateSubscriptions Wails function
export async function mutateSubscriptionsForActiveProfile(args: {
  assets: Record<string, types.SubscriptionUpdateItem>;
  action: 'subscribe' | 'unsubscribe';
  replaceOnConflict?: boolean;
}): Promise<types.UpdateSubscriptionsResult> {
  ensureSubscriptionMutationUnlocked();

  const profileId = await resolveActiveProfileID();
  return UpdateSubscriptions(
    new types.UpdateSubscriptionsRequest({
      profileId,
      assets: args.assets,
      action: args.action,
      applyMode: 'persist_and_sync',
      replaceOnConflict: args.replaceOnConflict ?? false,
    }),
  );
}

// applyLatestSubscriptionUpdatesForActiveProfile is a wrapper around the UpdateSubscriptionsToLatest Wails function
export async function applyLatestSubscriptionUpdatesForActiveProfile(args: {
  targets?: Pick<{ id: string; type: AssetType }, 'id' | 'type'>[];
}): Promise<types.UpdateSubscriptionsResult> {
  ensureSubscriptionMutationUnlocked();

  const profileId = await resolveActiveProfileID();
  return UpdateSubscriptionsToLatest(
    new types.UpdateSubscriptionsToLatestRequest({
      profileId,
      apply: true,
      targets: args.targets ? toLatestUpdateRequestTargets(args.targets) : [],
    }),
  );
}

// importAssetForActiveProfile is a wrapper around the ImportAsset Wails function
export async function importAssetForActiveProfile(args: {
  assetType: AssetType;
  zipPath: string;
  replaceOnConflict?: boolean;
}): Promise<types.UpdateSubscriptionsResult> {
  ensureSubscriptionMutationUnlocked();

  const profileId = await resolveActiveProfileID();
  return ImportAsset(
    new types.ImportAssetRequest({
      profileId,
      assetType: args.assetType,
      zipPath: args.zipPath,
      replaceOnConflict: args.replaceOnConflict ?? false,
    }),
  );
}
