import { describe, expect, it } from 'vitest';

import { isProfileSwapUnavailable } from '@/lib/profile-swap';

describe('isProfileSwapUnavailable', () => {
  it('returns true when game is running', () => {
    expect(
      isProfileSwapUnavailable({
        gameRunning: true,
        targetIsActive: false,
        swapLoading: false,
      }),
    ).toBe(true);
  });

  it('returns true when target is already active', () => {
    expect(
      isProfileSwapUnavailable({
        gameRunning: false,
        targetIsActive: true,
        swapLoading: false,
      }),
    ).toBe(true);
  });

  it('returns true while swap request is loading', () => {
    expect(
      isProfileSwapUnavailable({
        gameRunning: false,
        targetIsActive: false,
        swapLoading: true,
      }),
    ).toBe(true);
  });

  it('returns false when swap can proceed', () => {
    expect(
      isProfileSwapUnavailable({
        gameRunning: false,
        targetIsActive: false,
        swapLoading: false,
      }),
    ).toBe(false);
  });
});
