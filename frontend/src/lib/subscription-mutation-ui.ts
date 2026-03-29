import type { ReactNode } from 'react';
import { toast } from 'sonner';

import {
  isSubscriptionMutationLockedError,
  SUBSCRIPTION_MUTATION_LOCK_MESSAGE,
} from '@/lib/subscription-mutation-client';
import { useGameStore } from '@/stores/game-store';

type LockAwareDialogConfirm = {
  disabled?: boolean;
  disabledReason?: ReactNode;
};

export function useSubscriptionMutationLockState() {
  const locked = useGameStore((s) => s.running);
  return {
    locked,
    reason: locked ? SUBSCRIPTION_MUTATION_LOCK_MESSAGE : undefined,
  };
}

export function withLockAwareConfirm<T>(
  confirm: T & LockAwareDialogConfirm,
  locked: boolean,
  reason = SUBSCRIPTION_MUTATION_LOCK_MESSAGE,
): T & LockAwareDialogConfirm {
  return {
    ...confirm,
    disabled: confirm.disabled || locked,
    disabledReason:
      confirm.disabledReason ?? (locked ? reason : confirm.disabledReason),
  };
}

export function handleSubscriptionMutationError(
  error: unknown,
  fallback: string | ((error: unknown) => void),
): boolean {
  if (isSubscriptionMutationLockedError(error)) {
    toast.warning(SUBSCRIPTION_MUTATION_LOCK_MESSAGE);
    return true;
  }

  if (typeof fallback === 'function') {
    fallback(error);
  } else {
    toast.error(fallback);
  }

  return false;
}
