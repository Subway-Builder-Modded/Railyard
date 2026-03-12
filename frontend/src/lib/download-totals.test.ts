import { describe, expect, it } from "vitest";
import { sumVersionDownloads, toCumulativeDownloadTotals } from "./download-totals";

describe("download totals helpers", () => {
  it("sums all version counts for an asset", () => {
    expect(sumVersionDownloads({ "1.0.0": 12, "1.1.0": 8 })).toBe(20);
  });

  it("returns zero when counts are missing", () => {
    expect(sumVersionDownloads(undefined)).toBe(0);
  });

  it("builds cumulative totals per asset", () => {
    const totals = toCumulativeDownloadTotals({
      map_a: { "1.0.0": 2, "1.1.0": 3 },
      map_b: { "2.0.0": 7 },
    });

    expect(totals).toEqual({
      map_a: 5,
      map_b: 7,
    });
  });
});

