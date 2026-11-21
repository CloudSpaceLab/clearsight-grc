import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MetricMemberDrill } from "./MetricMemberDrill";

describe("MetricMemberDrill", () => {
  it("distinguishes recorded indicator severity using text and status color", () => {
    const page = { source_id: "snapshot", metric_id: "indicator_breaches", definition_revision: "v1", count: 2,
      items: ["CRITICAL", "HIGH"].map((state) => ({ member_id: state, target_type: "RISK" as const, target_title: `${state} indicator`, state, accessible: true, target_id: state })) };
    render(<MetricMemberDrill label="Indicator breaches" state="live" page={page} hasPrevious={false}
      onPrevious={() => {}} onNext={() => {}} onRetry={() => {}} onOpenRisk={() => {}}/>);
    expect(screen.getByText("Critical").className).toContain("cs-tone--error");
    expect(screen.getByText("High").className).toContain("cs-tone--warning");
  });

  it("labels an appetite breach without inventing a risk severity", () => {
    const page = { source_id: "snapshot", metric_id: "risks_outside_appetite", definition_revision: "v1", count: 1,
      items: [{ member_id: "m1", target_type: "RISK" as const, target_title: "Service disruption", state: "BREACHED", accessible: true, target_id: "risk-1" }] };
    render(<MetricMemberDrill label="Outside appetite" state="live" page={page} hasPrevious={false}
      onPrevious={() => {}} onNext={() => {}} onRetry={() => {}} onOpenRisk={() => {}}/>);
    expect(screen.getByText("Outside appetite").className).toContain("cs-tone--error");
    expect(screen.queryByText("Critical")).toBeNull();
  });
  it("keeps inaccessible historical members in the exact count but disables record navigation", () => {
    const onOpenMatter = vi.fn();
    const onOpenProgram = vi.fn();
    render(<MetricMemberDrill
      label="Critical and high"
      state="live"
      page={{
        source_id: "snapshot-1",
        metric_id: "critical_high_open",
        definition_revision: "home-oversight-v3",
        count: 2,
        items: [
          {
            member_id: "member-redacted",
            target_type: "MATTER",
            target_title: "Record access changed",
            state: "ACCESS_CHANGED",
            accessible: false,
          },
          {
            member_id: "member-visible",
            target_type: "MATTER",
            target_id: "matter-visible",
            target_title: "Visible control gap",
            state: "ASSESSMENT",
            accessible: true,
          },
        ],
      }}
      hasPrevious={false}
      onPrevious={() => {}}
      onNext={() => {}}
      onRetry={() => {}}
      onOpenMatter={onOpenMatter}
      onOpenProgram={onOpenProgram}
    />);

    expect(screen.getByText("2 records in this metric snapshot.")).toBeTruthy();
    expect(screen.getByText("Some records are included in this snapshot but are no longer accessible.")).toBeTruthy();
    expect(screen.getByText("Access changed")).toBeTruthy();

    const redactedAction = screen.getByRole("button", { name: /Open record for Record access changed/i }) as HTMLButtonElement;
    expect(redactedAction.disabled).toBe(true);
    fireEvent.click(redactedAction);
    expect(onOpenMatter).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: /Open record for Visible control gap/i }));
    expect(onOpenMatter).toHaveBeenCalledWith("matter-visible");
  });
});
