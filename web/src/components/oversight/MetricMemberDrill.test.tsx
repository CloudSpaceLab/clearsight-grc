import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MetricMemberDrill } from "./MetricMemberDrill";

describe("MetricMemberDrill", () => {
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

    expect(screen.getByText("2 exact records in this metric snapshot.")).toBeTruthy();
    expect(screen.getByText("Some records are still part of this historical count but their current access has changed.")).toBeTruthy();
    expect(screen.getByText("Access changed")).toBeTruthy();

    const redactedAction = screen.getByRole("button", { name: /Open record for Record access changed/i }) as HTMLButtonElement;
    expect(redactedAction.disabled).toBe(true);
    fireEvent.click(redactedAction);
    expect(onOpenMatter).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: /Open record for Visible control gap/i }));
    expect(onOpenMatter).toHaveBeenCalledWith("matter-visible");
  });
});
