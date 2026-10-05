import { expect, it } from "vitest";
import { formatIndicatorPeriod } from "./indicatorPresentation";

it("distinguishes point-in-time observations from explicit reporting periods", () => {
  expect(formatIndicatorPeriod()).toBe("Point-in-time observation");

  const period = formatIndicatorPeriod("2026-07-01T00:00:00Z", "2026-07-31T23:59:59Z");
  expect(period).not.toBe("Point-in-time observation");
  expect(period).toContain("2026");
});
