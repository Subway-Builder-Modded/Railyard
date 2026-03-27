export interface SwapAvailabilityInput {
  gameRunning: boolean;
  targetIsActive: boolean;
  swapLoading: boolean;
}

export function isProfileSwapUnavailable({
  gameRunning,
  targetIsActive,
  swapLoading,
}: SwapAvailabilityInput): boolean {
  return gameRunning || targetIsActive || swapLoading;
}
