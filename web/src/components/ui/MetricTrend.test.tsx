import { render } from "@testing-library/react";
import { expect, it } from "vitest";
import { MetricTrend } from "./MetricTrend";

it("preserves missing-day gaps instead of drawing through them", () => {
  const { container } = render(<MetricTrend
    ariaLabel="Risk movement"
    points={[
      { id: "1", at: "2026-10-01T12:00:00Z", label: "Oct 1", value: 4 },
      { id: "2", at: "2026-10-02T12:00:00Z", label: "Oct 2", value: 5 },
      { id: "3", at: "2026-10-05T12:00:00Z", label: "Oct 5", value: 3 },
      { id: "4", at: "2026-10-06T12:00:00Z", label: "Oct 6", value: 2 },
    ]}
  />);

  expect(container.querySelectorAll(".cs-metric-trend__line")).toHaveLength(2);
  expect(container.querySelectorAll(".cs-metric-trend__point")).toHaveLength(4);
  expect(container.querySelector("table")?.textContent).toContain("Oct 1");
  expect(container.querySelector("table")?.textContent).toContain("2");
});
