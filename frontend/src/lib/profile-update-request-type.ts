export const PROFILE_UPDATE_REQUEST_TYPES = [
  "update_subscriptions",
  "latest_check",
  "latest_apply",
] as const;

export type ProfileUpdateRequestType = (typeof PROFILE_UPDATE_REQUEST_TYPES)[number];
