import { describe, expect, it } from "vitest";
import {
  getSortOptionsForType,
  sortKeyToState,
  sortStateToOptionKey,
  SortKey as SortKeyHelper,
  SORT_OPTIONS,
  type SortState,
} from "./constants";

describe("sort helpers", () => {
  it("maps sort key to structured state", () => {
    expect(sortKeyToState("downloads:desc")).toEqual({
      field: "downloads",
      direction: "desc",
    });
    expect(sortKeyToState("random:asc")).toEqual({
      field: "random",
      direction: "asc",
    });
  });

  it("maps structured state to sort key", () => {
    const state: SortState = { field: "downloads", direction: "asc" };
    expect(sortStateToOptionKey(state, "mods")).toBe("downloads:asc");
  });

  it("compares sort keys via helper", () => {
    expect(SortKeyHelper.equals("downloads:asc", "downloads:asc")).toBe(true);
    expect(SortKeyHelper.equals("downloads:asc", "downloads:desc")).toBe(false);
  });

  it("hides population options for mods only", () => {
    const modOptions = getSortOptionsForType("mods");
    const mapOptions = getSortOptionsForType("maps");

    expect(modOptions).toHaveLength(7);
    expect(modOptions.map((opt) => opt.value)).not.toContain("population:asc");
    expect(modOptions.map((opt) => opt.value)).not.toContain("population:desc");
    expect(modOptions.map((opt) => opt.value)).toContain("random:asc");
    expect(modOptions.map((opt) => opt.value)).not.toContain("random:desc");
    expect(mapOptions).toHaveLength(9);
    expect(mapOptions).toEqual(SORT_OPTIONS);
  });

  it("falls back to default when sort key is invalid", () => {
    expect(sortKeyToState("nope")).toEqual({ field: "name", direction: "asc" });
  });
});
