import { expect, it } from "vitest";
import { formatRCSADate, formatRCSAPeriod } from "./rcsaPresentation";

it("formats RCSA periods deterministically in UTC", () => {
  expect(formatRCSADate("2026-09-30T23:59:59Z")).toBe("30 Sep 2026");
  expect(formatRCSAPeriod("2026-07-01T00:00:00Z", "2026-09-30T23:59:59Z"))
    .toBe("1 Jul 2026 – 30 Sep 2026");
});
