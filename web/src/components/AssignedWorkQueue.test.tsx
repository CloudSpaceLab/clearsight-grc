import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { AttentionItem } from "../types";
import { AssignedWorkQueue } from "./AssignedWorkQueue";

const request: AttentionItem = {
  id: "request-1", type: "EVIDENCE_REQUEST", title: "Review branch documents",
  why_now: "New documents received", scope: "Branch A", state: "Awaiting review",
  evidence: "Submitted response", owner: "Reviewer", due_at: "2030-04-01T12:00:00Z",
  primary_action: "Review documents", action_target_type: "EVIDENCE_REQUEST", action_target_id: "evidence-1",
  intervention_class: "REVIEW",
};
const approval: AttentionItem = {
  ...request, id: "approval-2", type: "MATTER", title: "Authorize issue remediation",
  why_now: "Decision pending", scope: "Operations", state: "Pending approval",
  evidence: "Issue revision 3", owner: "Authorizer",
  primary_action: "Inspect approval basis", action_target_type: "MATTER", action_target_id: "matter-2",
  intervention_class: "AUTHORIZATION",
  authority: { responsibility: "AUTHORIZER", decision_type: "matter.resolve", materiality: 3 },
};

describe("AssignedWorkQueue approved Work presentation", () => {
  it("selects an exact actor-visible record without executing a decision", () => {
    const onOpenItem = vi.fn();
    const onInspectAuthority = vi.fn();
    render(<AssignedWorkQueue presentation="workspace" state="live" items={[request, approval]} onOpenItem={onOpenItem} onInspectAuthority={onInspectAuthority}/>);
    expect(screen.getByRole("heading", { name: "Assigned to me" })).toBeTruthy();
    expect(screen.getByText(/2 items loaded/)).toBeTruthy();
    const table = screen.getByRole("table", { name: "Assigned work" });
    const selected = screen.getByRole("complementary", { name: "Selected work" });
    expect(within(selected).getByRole("heading", { name: request.title })).toBeTruthy();
    expect(within(selected).queryByRole("button", { name: "Check authority" })).toBeNull();
    fireEvent.focus(within(table).getByRole("row", { name: approval.title }));
    expect(within(selected).getByRole("heading", { name: approval.title })).toBeTruthy();
    expect(within(selected).getByText("Issue revision 3")).toBeTruthy();
    expect(onOpenItem).not.toHaveBeenCalled();
    expect(onInspectAuthority).not.toHaveBeenCalled();
    fireEvent.click(within(table).getByRole("button", { name: `Inspect for ${approval.title}` }));
    const inspection = screen.getByRole("dialog", { name: `Selected work: ${approval.title}` });
    expect(within(inspection).getByText("Issue revision 3")).toBeTruthy();
    fireEvent.click(within(inspection).getByRole("button", { name: "Check authority" }));
    expect(onInspectAuthority).toHaveBeenCalledWith(approval);
    fireEvent.click(within(inspection).getByRole("button", { name: "Open issue" }));
    expect(onOpenItem).toHaveBeenCalledOnce();
    expect(onOpenItem).toHaveBeenCalledWith(approval);
  });

  it("does not offer a material action without an exact linked record", () => {
    render(<AssignedWorkQueue presentation="workspace" state="live"
      items={[{ ...request, action_target_id: undefined, authority: approval.authority }]}
      onOpenItem={vi.fn()} onInspectAuthority={vi.fn()}/>);
    const detail = screen.getByRole("complementary", { name: "Selected work" });
    expect(within(detail).getByText("No linked record is available.")).toBeTruthy();
    expect(within(detail).queryByRole("button", { name: "Open request" })).toBeNull();
    expect(within(detail).queryByRole("button", { name: "Check authority" })).toBeNull();
  });

  it("drops an inspected record when a new scoped feed replaces it", () => {
    const open = vi.fn();
    const { rerender } = render(<AssignedWorkQueue presentation="workspace" state="live" items={[request]} onOpenItem={open}/>);
    fireEvent.click(within(screen.getByRole("table", { name: "Assigned work" })).getByRole("button", { name: `Inspect for ${request.title}` }));
    expect(screen.getByRole("dialog", { name: `Selected work: ${request.title}` })).toBeTruthy();
    rerender(<AssignedWorkQueue presentation="workspace" state="live"
      items={[{ ...request, title: "Other scope request", action_target_id: "another-id" }]} onOpenItem={open}/>);
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(within(screen.getByRole("complementary", { name: "Selected work" })).getByRole("heading", { name: "Other scope request" })).toBeTruthy();
    expect(open).not.toHaveBeenCalled();
  });

  it("filters the loaded actor queue without inventing organization-wide metrics", () => {
    const now = Date.now();
    const overdue = { ...request, id: "late", title: "Late request", due_at: new Date(now - 3600000).toISOString() };
    const soon = { ...request, id: "soon", title: "Due tomorrow", due_at: new Date(now + 86400000).toISOString() };
    const exception = { ...request, id: "gap", title: "Evidence gap", intervention_class: "EVIDENCE_EXCEPTION" as const, due_at: "" };
    render(<AssignedWorkQueue presentation="workspace" state="live" items={[overdue, soon, approval, exception]} onOpenItem={vi.fn()}/>);
    const measures = screen.getByLabelText("Work measures");
    expect(within(measures).getByRole("button", { name: /Overdue: 1/ })).toBeTruthy();
    expect(within(measures).getByRole("button", { name: /Due next 7 days: 1/ })).toBeTruthy();
    expect(within(measures).getByRole("button", { name: /Your approvals: 1/ })).toBeTruthy();
    expect(within(measures).getByRole("button", { name: /Evidence needed: 1/ })).toBeTruthy();
    fireEvent.click(within(measures).getByRole("button", { name: /Overdue: 1/ }));
    expect(within(screen.getByRole("table", { name: "Assigned work" })).getAllByRole("row")).toHaveLength(2);
    expect(within(screen.getByRole("table", { name: "Assigned work" })).getByRole("row", { name: "Late request" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
    fireEvent.click(screen.getByRole("tab", { name: "Approvals" }));
    expect(within(screen.getByRole("table", { name: "Assigned work" })).getByRole("row", { name: approval.title })).toBeTruthy();
    expect(within(screen.getByRole("table", { name: "Assigned work" })).getAllByRole("row")).toHaveLength(2);
  });


  it("searches only the bounded actor-visible Work feed and clears combined filters", () => {
    render(<AssignedWorkQueue presentation="workspace" state="live" items={[request, approval]} onOpenItem={vi.fn()}/>);
    const search = screen.getByRole("searchbox", { name: "Search loaded work" });
    fireEvent.change(search, { target: { value: "Authorize issue" } });
    const table = screen.getByRole("table", { name: "Assigned work" });
    expect(within(table).getByRole("row", { name: approval.title })).toBeTruthy();
    expect(within(table).queryByRole("row", { name: request.title })).toBeNull();
    expect(screen.getByText(/1 of 2 loaded items/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
    expect((search as HTMLInputElement).value).toBe("");
    expect(within(table).getByRole("row", { name: request.title })).toBeTruthy();
  });

  it("uses the server snapshot time for deadline classification when provided", () => {
    const snapshot = "2026-10-19T12:00:00Z";
    const nearDeadline = { ...request, due_at: "2026-10-20T12:00:00Z" };
    render(<AssignedWorkQueue presentation="workspace" state="live" asOf={snapshot} items={[nearDeadline]} onOpenItem={vi.fn()}/>);
    const measures = screen.getByLabelText("Work measures");
    expect(within(measures).getByRole("button", { name: /Due next 7 days: 1/ })).toBeTruthy();
    expect(within(measures).getByRole("button", { name: /Overdue: 0/ })).toBeTruthy();
  });

  it("qualifies loaded counts when the authorized backend reports more assignments", () => {
    render(<AssignedWorkQueue presentation="workspace" state="live" items={[request]}
      hasMore onOpenItem={vi.fn()}/>);
    expect(screen.getByText("More assignments available. The queue shows loaded work.")).toBeTruthy();
    expect(screen.getByText("1 item loaded")).toBeTruthy();
  });

  it("preserves the shared brief presentation and fail-closed unavailable state", () => {
    const { rerender } = render(<AssignedWorkQueue state="live" items={[request]} onOpenItem={vi.fn()}/>);
    expect(screen.getByRole("article")).toBeTruthy();
    expect(screen.queryByRole("table")).toBeNull();
    rerender(<AssignedWorkQueue presentation="workspace" state="unavailable" items={[request]} onOpenItem={vi.fn()} onRetry={vi.fn()}/>);
    expect(screen.queryByRole("table")).toBeNull();
    expect(screen.queryByRole("complementary", { name: "Selected work" })).toBeNull();
    expect(screen.getByRole("button", { name: "Try again" })).toBeTruthy();
  });
});
