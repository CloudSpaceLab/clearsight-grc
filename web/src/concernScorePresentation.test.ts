import { describe, expect, it } from "vitest";
import { concernScoreText } from "./concernScorePresentation";

describe("concern score presentation", () => {
  it.each([
    [0, "0 / 100 concern points"],
    [-0, "0 / 100 concern points"],
    [100, "100 / 100 concern points"],
    [62, "62 / 100 concern points"],
    [74.99, "74.99 / 100 concern points"],
    [24.99999999999, "24.99999999999 / 100 concern points"],
  ])("preserves %s without implying probability", (score, expected) => {
    expect(concernScoreText(score)).toBe(expected);
    expect(concernScoreText(score)).not.toContain("%");
  });

  it.each([undefined, null])("keeps missing values separate from zero", (score) => {
    expect(concernScoreText(score)).toBe("Not assessed");
    expect(concernScoreText(score, "compact")).toBe("Not assessed");
  });

  it.each([NaN, Infinity, -Infinity, -1, 100.01])("does not present invalid score %s", (score) => {
    expect(concernScoreText(score)).toBe("Score unavailable");
    expect(concernScoreText(score, "compact")).toBe("Score unavailable");
  });

  it("supports a compact value under an explicit Concern score heading", () => {
    expect(concernScoreText(62, "compact")).toBe("62 / 100");
    expect(concernScoreText(74.99, "compact")).toBe("74.99 / 100");
  });
});
