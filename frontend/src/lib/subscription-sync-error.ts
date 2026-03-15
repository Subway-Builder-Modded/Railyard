import { SubscriptionSyncError } from '@/stores/installed-store';

import type { types } from '../../wailsjs/go/models';

export interface SubscriptionSyncErrorState {
  version: string;
  message: string;
  errors: types.UserProfilesError[];
}

const CANCELLATION_DOWNLOADER_ERROR_TYPES = new Set<string>([
  'uninstall_not_installed',
]);
const SILENT_WARNING_ERROR_TYPES = new Set<string>(['sync_superseded']);

export function toSubscriptionSyncErrorState(
  err: unknown,
  version: string,
): SubscriptionSyncErrorState | null {
  if (!(err instanceof SubscriptionSyncError)) {
    return null;
  }

  return {
    version,
    message: err.message,
    errors: err.profileErrors,
  };
}

export function hasCancellationSyncErrors(
  errors: types.UserProfilesError[] | undefined | null,
): boolean {
  return (errors ?? []).some((profileError) =>
    CANCELLATION_DOWNLOADER_ERROR_TYPES.has(
      profileError.downloaderErrorType ?? '',
    ),
  );
}

export function hasOnlySilentSyncWarnings(
  errors: types.UserProfilesError[] | undefined | null,
): boolean {
  const values = errors ?? [];
  if (values.length === 0) {
    return false;
  }
  return values.every((profileError) =>
    SILENT_WARNING_ERROR_TYPES.has(profileError.errorType),
  );
}

export function isCancellationSyncError(
  err: SubscriptionSyncErrorState | null | undefined,
): boolean {
  if (!err) {
    return false;
  }
  return hasCancellationSyncErrors(err.errors);
}
