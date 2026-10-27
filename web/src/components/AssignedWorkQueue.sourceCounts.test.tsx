import { render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadActorWorkDueCounts, type ActorWorkDueCounts } from "../api";
import type { AttentionItem } from "../types";
import { AssignedWorkQueue } from "./AssignedWorkQueue";

vi.mock("../api", () => ({ loadActorWorkDueCounts: vi.fn(), loadActorWorkPage: vi.fn() }));

const scope = { tenantID: "bank", legalEntityID: "entity", principalID: "actor" };
const assignmentCounts: ActorWorkDueCounts = {
  total: 110, overdue: 43, due_soon: 25, undated: 9,
  as_of: "2026-10-27T09:00:00Z",
  coverage: "WORKFLOW_ASSIGNMENTS_ONLY",
  count_basis: "SOURCE_SNAPSHOT",
  scope: { tenant_id: scope.tenantID, legal_entity_id: scope.legalEntityID, principal_id: scope.principalID },
};
const localItem: AttentionItem = {
  id: "local-evidence", type: "EVIDENCE_REQUEST", title: "Review received evidence",
  why_now: "Submission received", scope: "Ops", state: "Ready",
  evidence: "Submitted", owner: "Reviewer", due_at: "", primary_action: "Inspect",
  intervention_class: "EVIDENCE_EXCEPTION",
};

beforeEach(() => vi.resetAllMocks());

describe("source-qualified Work deadline measures", () => {
  it("uses complete workflow deadline counts without inventing approval/evidence totals", async () => {
    vi.mocked(loadActorWorkDueCounts).mockResolvedValue(assignmentCounts);
    render(<AssignedWorkQueue presentation="workspace" state="live" items={[localItem]}
      sourceSearchScopeKey="bank:entity:actor:0" sourceCountScope={scope} onOpenItem={vi.fn()}/>);
    const metrics = screen.getByLabelText("Loaded work measures");
    expect(await within(metrics).findByRole("button", { name: /Overdue: 43.*Workflow assignments/ })).toBeTruthy();
    expect(within(metrics).getByRole("button", { name: /Due next 7 days: 25.*Workflow assignments/ })).toBeTruthy();
    expect(within(metrics).getByRole("button", { name: /Your approvals: 0.*Loaded work/ })).toBeTruthy();
    expect(within(metrics).getByRole("button", { name: /Evidence needed: 1.*Loaded work/ })).toBeTruthy();
    expect(screen.getByText("Deadline totals: workflow assignments only. Approvals and evidence: loaded work.")).toBeTruthy();
  });

  it("fails closed when a count response belongs to another tenant", async () => {
    vi.mocked(loadActorWorkDueCounts).mockResolvedValue({
      ...assignmentCounts, scope: { ...assignmentCounts.scope, tenant_id: "other" },
    });
    render(<AssignedWorkQueue presentation="workspace" state="live" items={[localItem]}
      sourceSearchScopeKey="bank:entity:actor:0" sourceCountScope={scope} onOpenItem={vi.fn()}/>);
    expect(await screen.findByText("Assignment totals unavailable. Showing loaded work measures.")).toBeTruthy();
    const metrics = screen.getByLabelText("Loaded work measures");
    expect(within(metrics).getByRole("button", { name: /Overdue: 0.*Loaded work/ })).toBeTruthy();
    expect(within(metrics).queryByRole("button", { name: /Overdue: 43/ })).toBeNull();
  });

  it("drops stale source counts immediately when the actor scope changes", async () => {
    vi.mocked(loadActorWorkDueCounts).mockResolvedValueOnce(assignmentCounts)
      .mockImplementationOnce(() => new Promise<ActorWorkDueCounts>(() => {}));
    const { rerender } = render(<AssignedWorkQueue presentation="workspace" state="live" items={[localItem]}
      sourceSearchScopeKey="bank:entity:actor:0" sourceCountScope={scope} onOpenItem={vi.fn()}/>);
    await waitFor(() => expect(screen.getByRole("button", { name: /Overdue: 43/ })).toBeTruthy());
    rerender(<AssignedWorkQueue presentation="workspace" state="live" items={[localItem]}
      sourceSearchScopeKey="bank:entity:another:0"
      sourceCountScope={{ ...scope, principalID: "another" }} onOpenItem={vi.fn()}/>);
    expect(screen.getByRole("button", { name: /Overdue: 0.*Loaded work/ })).toBeTruthy();
    expect(screen.queryByRole("button", { name: /Overdue: 43/ })).toBeNull();
  });

  it("does not request global counts when the selected organization area is unsupported", () => {
    render(<AssignedWorkQueue presentation="workspace" state="live" items={[localItem]}
      onOpenItem={vi.fn()}/>);
    expect(loadActorWorkDueCounts).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: /Overdue: 0.*Loaded work/ })).toBeTruthy();
  });
});
