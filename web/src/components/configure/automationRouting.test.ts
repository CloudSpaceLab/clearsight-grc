import { expect, it } from "vitest";
import { automationHash, automationRoute } from "./automationRouting";

it("round-trips policy, filters and source tabs through the existing Automation route", () => {
  const route = { tab: "activity" as const, policy: "policy/a+b", q: "Evidence & renewal", status: "DRAFT", action: "REVERSIBLE_WRITE" };
  expect(automationRoute(automationHash(route))).toEqual(route);
  expect(automationHash({ tab: "automations", q: "" })).toBe("#configure/automation");
});

it("bounds URL values and ignores unknown tabs without discarding valid filters", () => {
  expect(automationRoute(`#configure/automation?tab=missing&policy=${"x".repeat(257)}&q=renewal`)).toEqual({ tab: "automations", q: "renewal", policy: undefined, action: undefined, status: undefined });
});
