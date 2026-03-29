import { beforeEach, describe, expect, it, vi } from 'vitest';

const { mockToastError, mockToastWarning } = vi.hoisted(() => ({
  mockToastWarning: vi.fn(),
  mockToastError: vi.fn(),
}));

vi.mock('sonner', () => ({
  toast: {
    warning: mockToastWarning,
    error: mockToastError,
  },
}));

import {
  SUBSCRIPTION_MUTATION_LOCK_MESSAGE,
  SubscriptionMutationLockedError,
} from '@/lib/subscription-mutation-client';
import {
  handleSubscriptionMutationError,
  withLockAwareConfirm,
} from '@/lib/subscription-mutation-ui';

describe('subscription-mutation-ui', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('applies disabled state and reason to dialog confirms when locked', () => {
    const confirm = withLockAwareConfirm(
      {
        label: 'Confirm',
        onConfirm: vi.fn(),
      },
      true,
    );

    expect(confirm.disabled).toBe(true);
    expect(confirm.disabledReason).toBe(SUBSCRIPTION_MUTATION_LOCK_MESSAGE);
  });

  it('keeps explicit disabled reason when already provided', () => {
    const confirm = withLockAwareConfirm(
      {
        label: 'Confirm',
        onConfirm: vi.fn(),
        disabledReason: 'Already disabled',
      },
      true,
    );

    expect(confirm.disabled).toBe(true);
    expect(confirm.disabledReason).toBe('Already disabled');
  });

  it('routes lock errors to warning toast', () => {
    const handled = handleSubscriptionMutationError(
      new SubscriptionMutationLockedError(),
      'fallback error',
    );

    expect(handled).toBe(true);
    expect(mockToastWarning).toHaveBeenCalledTimes(1);
    expect(mockToastError).not.toHaveBeenCalled();
  });

  it('routes non-lock errors to fallback toast', () => {
    const handled = handleSubscriptionMutationError(
      new Error('boom'),
      'fallback error',
    );

    expect(handled).toBe(false);
    expect(mockToastWarning).not.toHaveBeenCalled();
    expect(mockToastError).toHaveBeenCalledWith('fallback error');
  });
});
