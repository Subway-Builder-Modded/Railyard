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

export function isSubscriptionMutationLockedError(
  error: unknown,
): error is SubscriptionMutationLockedErrorLike {
  if (error instanceof SubscriptionMutationLockedError) {
    return true;
  }
  return isSubscriptionMutationLockedErrorLike(error);
}

function ensureSubscriptionMutationUnlocked() {
  if (isSubscriptionMutationLocked(useGameStore.getState().running)) {
    throw new SubscriptionMutationLockedError();
  }
}

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
