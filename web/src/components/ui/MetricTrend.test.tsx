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

it("centers flat and single-point series inside the plot", () => {
  const { container, rerender } = render(<MetricTrend
    ariaLabel="Flat risk movement"
    points={[
      { id: "1", at: "2026-10-01T12:00:00Z", label: "Oct 1", value: 4 },
      { id: "2", at: "2026-10-02T12:00:00Z", label: "Oct 2", value: 4 },
    ]}
  />);
  const flat = [...container.querySelectorAll(".cs-metric-trend__point")];
  expect(flat.map((point) => point.getAttribute("cy"))).toEqual(["90", "90"]);

  rerender(<MetricTrend
    ariaLabel="Single risk movement"
    points={[{ id: "only", at: "2026-10-01T12:00:00Z", label: "Now", value: 4 }]}
  />);
  expect(container.querySelector(".cs-metric-trend__point")?.getAttribute("cx")).toBe("500");
  expect(container.querySelector(".cs-metric-trend__point")?.getAttribute("cy")).toBe("90");
});

it("uses exact display values in the accessible table without changing plot values", () => {
  const { container } = render(<MetricTrend
    ariaLabel="Loss movement"
    points={[
      { id: "1", at: "2026-10-01T12:00:00Z", label: "Oct 1", value: 0.25, displayValue: "₦9,223,372,036,854,775.80" },
      { id: "2", at: "2026-10-02T12:00:00Z", label: "Oct 2", value: 1, displayValue: "₦36,893,488,147,419,103.20" },
    ]}
  />);
  expect(container.querySelector("table")?.textContent).toContain("₦9,223,372,036,854,775.80");
  expect(container.querySelector("table")?.textContent).toContain("₦36,893,488,147,419,103.20");
});
