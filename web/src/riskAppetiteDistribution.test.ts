import { expect, it } from "vitest";
import type { RiskSummary, RiskAppetitePosition } from "./riskTypes";
import { riskAppetiteDistribution } from "./riskAppetiteDistribution";

function risk(id: string, position: RiskAppetitePosition, options: {
  assessedVersion?: number;
  appetiteID?: string;
  activeAppetiteID?: string;
  assessment?: boolean;
} = {}): RiskSummary {
  return {
    risk: { id, name: id, version: 3, status: "ACTIVE" },
    latest_assessment: options.assessment === false ? undefined : {
      risk_id: id,
      risk_version: options.assessedVersion ?? 3,
      appetite_statement_id: options.appetiteID ?? "appetite-a",
      appetite_position: position,
    },
    active_appetite: { id: options.activeAppetiteID ?? "appetite-a", status: "ACTIVE" },
  } as unknown as RiskSummary;
}

it("isolates stale or absent appetite assessments instead of implying healthy or breached risk", () => {
  const segments = riskAppetiteDistribution([
    risk("high", "BREACHED"),
    risk("near", "APPROACHING"),
    risk("within", "WITHIN"),
    risk("stale", "BREACHED", { assessedVersion: 2 }),
    risk("revised", "WITHIN", { activeAppetiteID: "appetite-b" }),
    risk("missing", "WITHIN", { assessment: false }),
  ]);

  expect(segments.map(({ id, count }) => [id, count])).toEqual([
    ["breached", 1],
    ["approaching", 1],
    ["within", 1],
    ["unknown", 3],
  ]);
});
