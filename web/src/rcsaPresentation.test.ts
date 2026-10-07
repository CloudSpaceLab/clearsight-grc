import { expect, it } from "vitest";
import { formatRCSADate, formatRCSAPeriod, rcsaPhasePath } from "./rcsaPresentation";

it("formats RCSA periods deterministically in UTC", () => {
  expect(formatRCSADate("2026-09-30T23:59:59Z")).toBe("30 Sep 2026");
  expect(formatRCSAPeriod("2026-07-01T00:00:00Z", "2026-09-30T23:59:59Z"))
    .toBe("1 Jul 2026 – 30 Sep 2026");
});

it("treats risk acceptance and remediation verification as alternative governed outcomes", () => {
  const accepted = rcsaPhasePath("RISK_ACCEPTANCE");
  expect(accepted.find((step) => step.id === "RISK_ACCEPTANCE")?.state).toBe("current");
  expect(accepted.find((step) => step.id === "REMEDIATION_VERIFICATION")?.state).toBe("not_required");

  const remediation = rcsaPhasePath("REMEDIATION_VERIFICATION");
  expect(remediation.find((step) => step.id === "RISK_ACCEPTANCE")?.state).toBe("not_required");
  expect(remediation.find((step) => step.id === "REMEDIATION_VERIFICATION")?.state).toBe("current");
});

it("keeps collection and independent challenge sequential before the outcome branch", () => {
  expect(rcsaPhasePath("INDEPENDENT_CHALLENGE").map((step) => [step.id, step.state])).toEqual([
    ["COLLECTION", "complete"],
    ["INDEPENDENT_CHALLENGE", "current"],
    ["RISK_ACCEPTANCE", "pending"],
    ["REMEDIATION_VERIFICATION", "pending"],
  ]);
});


it("does not call a completed challenge outcome pending when the branch is unclassified", () => {
  const complete = rcsaPhasePath("COMPLETE");
  expect(complete.find((step) => step.id === "RISK_ACCEPTANCE")?.state).toBe("unknown");
  expect(complete.find((step) => step.id === "REMEDIATION_VERIFICATION")?.state).toBe("unknown");
});
