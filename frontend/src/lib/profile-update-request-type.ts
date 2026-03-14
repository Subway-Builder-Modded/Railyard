export const UPDATE_SUBSCRIPTIONS_REQUEST_TYPES = [
  "update_subscriptions",
  "latest_check",
  "latest_apply",
] as const;

export type UpdateSubscriptionsRequestType = (typeof UPDATE_SUBSCRIPTIONS_REQUEST_TYPES)[number];
