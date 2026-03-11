export const SEARCH_BAR_PLACEHOLDER = "Search by name, author, description...";

export const SEARCH_FILTER_EMPTY_LABELS = {
  generic: "No options available",
  tags: "No tags available",
  specialDemand: "No special demand tags available",
} as const;

export const MAX_CARD_BADGES = 3;

export const FUSE_SEARCH_OPTIONS = {
  keys: ["searchText"],
  threshold: 0.35,
  ignoreLocation: true,
  ignoreFieldNorm: true,
  minMatchCharLength: 1,
  shouldSort: false,
  ignoreDiacritics: true,
};
