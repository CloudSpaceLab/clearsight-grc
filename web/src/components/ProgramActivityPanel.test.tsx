import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ProgramAggregate } from "../types";
import type { ProgramReviewDigest } from "../programReviewApi";
import { ProgramActivityPanel } from "./ProgramActivityPanel";

const aggregate = {
  program: { id: "program-1", tenant_id: "tenant-1", name: "Security", code: "SEC",
    version: 4, created_at: "2026-09-01T08:00:00Z", updated_at: "2026-10-17T10:00:00Z" },
  current_state: { id: "state-1", program_version: 4, projection_version: 8,
    generated_at: "2026-10-17T11:00:00Z", dimensions: {}, reasons: [], open_matter_count: 1 },
} as unknown as ProgramAggregate;

const digest: ProgramReviewDigest = {
  program_id: "program-1", state: "CHANGED", review_required: true,
  current_program_version: 4, current_projection_version: 8, current_overall: "EVIDENCE_INSUFFICIENT",
  checkpoint: { id: "checkpoint-1", tenant_id: "tenant-1", program_id: "program-1",
    principal_id: "reviewer-1", program_version: 3, projection_version: 7,
    accepted_at: "2026-10-15T09:00:00Z" },
  open_matter_count: 1,
  changes: [{ kind: "REQUIREMENT", summary: "Access review requirement updated", object_type: "REQUIREMENT", object_id: "requirement-1" }],
  changes_total: 4, changes_omitted: 3, history_truncated: true,
  current_exceptions: [], current_exceptions_total: 1,
  new_exceptions: [], new_exceptions_total: 2,
  resolved_exceptions: [], resolved_exceptions_total: 1,
};

describe("approved Program Activity view", () => {
  it("uses scoped review-digest changes and reports omitted history without manufacturing events", () => {
    const onReload = vi.fn();
    render(<ProgramActivityPanel aggregate={aggregate} digest={digest} onReload={onReload}>
      <button type="button">Load historical snapshot</button>
    </ProgramActivityPanel>);
    expect(screen.getByRole("heading", { name: "Program changes and review history" })).toBeTruthy();
    expect(screen.getByRole("region", { name: "Changes since last review" })).toBeTruthy();
    expect(screen.getByText("4 changes since the last review")).toBeTruthy();
    expect(screen.getByText("Access review requirement updated")).toBeTruthy();
    expect(screen.getByText("Review required")).toBeTruthy();
    expect(screen.getByText(/Only part of the change history was returned/)).toBeTruthy();
    expect(screen.getByText("2 new exceptions recorded.")).toBeTruthy();
    expect(screen.getByText("1 exception resolved.")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Load historical snapshot" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Reload review status" })).toBeNull();
    expect(onReload).not.toHaveBeenCalled();
    expect(screen.queryByText("reviewer-1")).toBeNull();
  });

  it("does not assert no changes from stale Program/projection data or another Program", () => {
    const onReload = vi.fn();
    const { rerender } = render(<ProgramActivityPanel aggregate={aggregate}
      digest={{ ...digest, state: "CURRENT", program_id: "program-2" }} onReload={onReload}/>);
    expect(screen.getByText("Review changes unavailable")).toBeTruthy();
    expect(screen.queryByText("No recorded changes since the last review")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Reload review status" }));
    expect(onReload).toHaveBeenCalledTimes(1);
    rerender(<ProgramActivityPanel aggregate={aggregate} digest={{
      ...digest, state: "CURRENT", current_projection_version: 7,
    }} onReload={onReload}/>);
    expect(screen.getByText("Review changes unavailable")).toBeTruthy();
    expect(screen.queryByText("No recorded changes since the last review")).toBeNull();
  });

  it("rejects absent, foreign-tenant and contradictory accepted checkpoints", () => {
    const reviewed = { ...digest, state: "CURRENT", review_required: false,
      changes: [], changes_total: 0, changes_omitted: 0, history_truncated: false };
    const view = render(<ProgramActivityPanel aggregate={aggregate}
      digest={{ ...reviewed, checkpoint: undefined }} onReload={vi.fn()}/>);
    expect(screen.getByText("Review changes unavailable")).toBeTruthy();
    expect(screen.queryByText("No recorded changes since the last review")).toBeNull();

    view.rerender(<ProgramActivityPanel aggregate={aggregate}
      digest={{ ...reviewed, checkpoint: { ...digest.checkpoint!, tenant_id: "another-tenant" } }}
      onReload={vi.fn()}/>);
    expect(screen.getByText("Review changes unavailable")).toBeTruthy();

    view.rerender(<ProgramActivityPanel aggregate={aggregate}
      digest={{ ...reviewed, review_required: true }} onReload={vi.fn()}/>);
    expect(screen.getByText("Review changes unavailable")).toBeTruthy();

    view.rerender(<ProgramActivityPanel aggregate={aggregate}
      digest={{ ...reviewed, state: "NO_BASELINE" }} onReload={vi.fn()}/>);
    expect(screen.getByText("Review changes unavailable")).toBeTruthy();
    expect(screen.queryByText("No accepted review yet")).toBeNull();
  });

  it("rejects a digest if the Program projection is not current", () => {
    const stale = { ...aggregate, current_state: {
      ...aggregate.current_state!, program_version: 3,
    }} as ProgramAggregate;
    render(<ProgramActivityPanel aggregate={stale} digest={digest} onReload={vi.fn()}/>);
    expect(screen.getByText("Review changes unavailable")).toBeTruthy();
    expect(screen.queryByText("4 changes since the last review")).toBeNull();
  });

  it("distinguishes first review, accepted current review and unavailable read", () => {
    const view = render(<ProgramActivityPanel aggregate={aggregate} digest={null} onReload={vi.fn()}/>);
    expect(screen.getByText("Review changes unavailable")).toBeTruthy();
    view.rerender(<ProgramActivityPanel aggregate={aggregate} digest={{
      ...digest, state: "NO_BASELINE", checkpoint: undefined, changes: [], changes_total: 0,
      changes_omitted: 0, history_truncated: false,
    }} onReload={vi.fn()}/>);
    expect(screen.getByText("No accepted review yet")).toBeTruthy();
    expect(screen.queryByText("No recorded changes since the last review")).toBeNull();
    view.rerender(<ProgramActivityPanel aggregate={aggregate} digest={{
      ...digest, state: "CURRENT", review_required: false,
      changes: [], changes_total: 0, changes_omitted: 0, history_truncated: false,
    }} onReload={vi.fn()}/>);
    expect(screen.getByText("No recorded changes since the last review")).toBeTruthy();
    expect(screen.queryByText("4 changes since the last review")).toBeNull();
  });
});
