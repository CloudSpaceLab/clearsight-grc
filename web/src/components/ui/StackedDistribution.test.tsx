import { render, screen } from "@testing-library/react";
import { expect, it } from "vitest";
import { StackedDistribution } from "./StackedDistribution";

it("announces all posture populations while keeping the visual rail decorative", () => {
  const { container } = render(<StackedDistribution
    ariaLabel="Program posture"
    segments={[
      { id: "attention", label: "Follow-up", tone: "warning", count: 5 },
      { id: "current", label: "Current", tone: "success", count: 3 },
      { id: "missing", label: "Needs assessment", tone: "unknown", count: 0 },
    ]}
  />);

  expect(screen.getByRole("img", { name: "Program posture: 5 follow-up, 3 current, 0 needs assessment" })).toBeTruthy();
  expect(container.querySelectorAll(".cs-stacked-distribution__part")).toHaveLength(2);
  expect(container.querySelectorAll(".cs-stacked-distribution__legend .cs-stacked-distribution__item")).toHaveLength(2);
});
