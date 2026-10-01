import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MetricCard } from "./MetricCard";

describe("MetricCard", () => {
  it("keeps value, status and coverage text together", () => {
    render(<MetricCard
      label="Overdue"
      value={4}
      status="Past due"
      tone="warning"
      detail="Open issues past their due date"
      meta="42 in scope · 1 excluded · 2 unknown"
    />);

    expect(screen.getByText("Overdue")).toBeTruthy();
    expect(screen.getByText("4")).toBeTruthy();
    expect(screen.getByText("Past due")).toBeTruthy();
    expect(screen.getByText("42 in scope · 1 excluded · 2 unknown")).toBeTruthy();
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("exposes an interactive metric as one pressed-state control", () => {
    const onPress = vi.fn();
    render(<MetricCard
      label="Critical and high"
      value={7}
      status="Needs attention"
      tone="error"
      detail="Open priority 4–5 issues"
      actionLabel="Show matching interventions"
      isSelected
      ariaControls="attention"
      onPress={onPress}
    />);

    const control = screen.getByRole("button", { name: /Critical and high.*7.*Needs attention/i });
    expect(control.getAttribute("aria-pressed")).toBe("true");
    expect(control.getAttribute("aria-controls")).toBe("attention");
    fireEvent.click(control);
    expect(onPress).toHaveBeenCalledTimes(1);
  });
});
