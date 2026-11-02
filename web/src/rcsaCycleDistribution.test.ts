import { expect, it } from "vitest";
import type { RCSACycleSummary, RCSAStatus } from "./rcsaTypes";
import { rcsaCycleStageDistribution } from "./rcsaCycleDistribution";

function cycle(status: RCSAStatus): RCSACycleSummary {
  return { cycle: { status } } as RCSACycleSummary;
}

it("shows every cycle stage without calling challenge a completed assessment", () => {
  const segments = rcsaCycleStageDistribution([
    cycle("DRAFT"), cycle("ASSESSMENT_OPEN"),
    cycle("AWAITING_CHALLENGE"), cycle("AWAITING_CHALLENGE"),
    cycle("COMPLETED"), cycle("CANCELLED"),
  ]);
  expect(segments.map((segment) => [segment.id, segment.count])).toEqual([
    ["DRAFT", 1], ["ASSESSMENT_OPEN", 1], ["AWAITING_CHALLENGE", 2],
    ["COMPLETED", 1], ["CANCELLED", 1],
  ]);
  expect(segments[2]?.label).toMatch(/challenge/);
});
