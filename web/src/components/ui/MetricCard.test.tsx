import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MetricCard } from "./index";

describe("MetricCard", () => {
  it("keeps value, quality and drill action in one accessible control", () => {
    const open = vi.fn();
    render(<MetricCard
      label="Critical and high"
      value={7}
      detail="Open priority 4–5 issues"
      meta="20 checked · 2 unknown"
      tone="error"
      quality="partial"
      actionLabel="Review interventions"
      onPress={open}
    />);

    const card = screen.getByRole("button", { name: /Critical and high: 7/ });
    expect(card.getAttribute("data-quality")).toBe("partial");
    expect(card.className).toContain("cs-tone--unknown");
    expect(screen.getByText("Coverage incomplete")).toBeTruthy();
    fireEvent.click(card);
    expect(open).toHaveBeenCalledTimes(1);
  });

  it("does not render incomplete zero as a success tone", () => {
    render(<MetricCard label="Overdue" value={0} tone="success" quality="unknown" detail="Open issues past due"/>);

    const card = screen.getByRole("article", { name: /Overdue: 0/ });
    expect(card.className).toContain("cs-tone--unknown");
    expect(card.className).not.toContain("cs-tone--success");
    expect(screen.getByText("Coverage unknown")).toBeTruthy();
  });

  it("preserves selected state for a drillable metric", () => {
    render(<MetricCard label="Routing gaps" value={3} isSelected onPress={() => undefined} actionLabel="Show routing gaps"/>);

    expect(screen.getByRole("button", { name: /Routing gaps: 3/ }).getAttribute("aria-pressed")).toBe("true");
  });


  it("communicates movement in text rather than relying on tone", () => {
    render(<MetricCard
      label="Overdue"
      value={4}
      delta="3 lower · Improved"
      tone="success"
      quality="current"
    />);

    const card = screen.getByRole("article", { name: /Overdue: 4.*3 lower · Improved.*Current/ });
    expect(card).toBeTruthy();
    expect(screen.getByText("3 lower · Improved")).toBeTruthy();
  });
});
