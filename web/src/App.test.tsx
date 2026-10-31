import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { StrictMode, type ComponentProps } from "react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import type { RuntimeContext } from "./api";
import { loadCaptureRequest, loadContext, loadEvidenceRequest, loadEvidenceRequests, loadReadiness, loadToday, switchLegalEntity } from "./api";
import type { AttentionItem, EvidenceRequest } from "./types";
import { declareWrongCaptureRecipient, reassignCaptureRecipient } from "./captureApi";
import { ApiError } from "./http";
import { loadFormTemplatePage } from "./formsApi";
import { loadNotifications, type InAppNotification, type NotificationPage } from "./notificationApi";
import { loadGroupOversight, type GroupOversightSnapshot } from "./groupOversightApi";
import type { ReportingPeriodQuery } from "./reportingPeriod";

const { listEvidenceRecipientCandidates, subscribeInvalidations } = vi.hoisted(() => ({ listEvidenceRecipientCandidates: vi.fn(), subscribeInvalidations: vi.fn() }));

vi.mock("./formsApi", () => ({
  loadFormTemplateItem: vi.fn().mockRejectedValue(new Error("No exact form selected")),
  loadFormTemplatePage: vi.fn().mockResolvedValue({ items: [] }),
  loadFormTemplateRevision: vi.fn(),
  loadReusableFormTemplateRefs: vi.fn().mockResolvedValue([]),
  loadStarterTemplates: vi.fn().mockResolvedValue([]),
  loadSavedFormViews: vi.fn().mockResolvedValue([]),
}));
vi.mock("./submittedDocumentApi", () => ({ loadDocuments: vi.fn().mockResolvedValue({ items: [] }) }));
vi.mock("./notificationApi", () => ({
  loadNotifications: vi.fn().mockResolvedValue({ items: [], unread_count: 0, as_of: "2026-10-02T09:00:00Z" }),
  markNotificationRead: vi.fn(),
}));
vi.mock("./groupOversightApi", () => ({
  loadGroupOversight: vi.fn(),
}));
vi.mock("./invalidationApi", () => ({ subscribeInvalidations }));

vi.mock("./components/RoleAwareOnboarding", async () => {
  const React = await import("react");
  return { RoleAwareOnboarding: ({ onStep, surface }: { onStep: (step: { intent?: string; view?: string }) => void | Promise<void>; surface: string }) => {
    const [busy, setBusy] = React.useState(false);
    const [error, setError] = React.useState("");
    async function openStep(step: { intent?: string; view?: string }) {
      setBusy(true); setError("");
      try { await onStep(step); }
      catch (error) {
        setError(error instanceof Error && error.message === "guide_step_unavailable"
          ? "No eligible item is available for this step."
          : "This guide step could not be opened. Try again.");
      }
      finally { setBusy(false); }
    }
    return <aside aria-label={`${surface === "VENDORS" ? "Vendor" : "Oversight"} guide`}>
      <output data-testid="onboarding-surface">{surface}</output>
      <button type="button" disabled={busy} onClick={() => void openStep({ intent: "open-vendor-due-diligence", view: "vendors" })}>Review due diligence</button>
      <button type="button" disabled={busy} onClick={() => void openStep({ intent: "open-vendor-next-action", view: "vendors" })}>Open next vendor task</button>
      {surface === "TODAY" && <>
        <button type="button" disabled={busy} onClick={() => void openStep({ intent: "open-first-attention", view: "today" })}>Review first assigned item</button>
        <button type="button" disabled={busy} onClick={() => void openStep({ intent: "open-routing", view: "today" })}>Check approval route</button>
        <button type="button" disabled={busy} onClick={() => void openStep({ intent: "open-capture", view: "today" })}>Open assigned response</button>
        <button type="button" disabled={busy} onClick={() => void openStep({ view: "configure" })}>Open guided configuration</button>
      </>}
      {error && <p role="alert">{error}</p>}
    </aside>;
  }};
});
vi.mock("./components/VendorsWorkspace", () => ({
  VendorsWorkspace: ({ onOpenRequest, guideIntent, targetID }: { onOpenRequest?: (requestID: string) => void; guideIntent?: { type: string }; targetID?: string }) => <><output data-testid="vendor-guide-intent">{guideIntent?.type}</output><output data-testid="vendor-target">{targetID}</output><button type="button" onClick={() => onOpenRequest?.("request-vendor-1")}>Review vendor request</button></>,
}));
vi.mock("./components/risks/RisksWorkspace", () => ({
  RisksWorkspace: ({ targetID, organizationScopeID, embedded }: ComponentProps<typeof import("./components/risks/RisksWorkspace")["RisksWorkspace"]>) => <section>{embedded ? <h2>Risks</h2> : <h1>Risks</h1>}<output data-testid="risk-target">{targetID}</output><output data-testid="risk-scope">{organizationScopeID}</output></section>,
}));
vi.mock("./components/rcsa/RCSAWorkspace", () => ({
  RCSAWorkspace: ({ embedded, targetID }: ComponentProps<typeof import("./components/rcsa/RCSAWorkspace")["RCSAWorkspace"]>) => <section>{embedded ? <h2>Review cycles</h2> : <h1>RCSA cycles</h1>}<output data-testid="rcsa-target">{targetID}</output></section>,
}));
vi.mock("./components/oversight/GroupOversightWorkspace", () => ({
  GroupOversightWorkspace: ({ onOpenLegalEntity }: { onOpenLegalEntity: (legalEntityID: string) => void }) => <section><h1>Group posture</h1><button type="button" onClick={() => onOpenLegalEntity("entity-ng-uuid")}>Open current OpCo</button></section>,
}));
vi.mock("./components/losses/LossesWorkspace", () => ({
  LossesWorkspace: ({ targetID, organizationScopeID, organizationScopeName, embedded }: ComponentProps<typeof import("./components/losses/LossesWorkspace")["LossesWorkspace"]>) => <section>{embedded ? <h2>Losses</h2> : <h1>Losses</h1>}<output data-testid="loss-target">{targetID}</output><output data-testid="loss-scope">{organizationScopeID}:{organizationScopeName}</output></section>,
}));
vi.mock("./components/reports/ReportsWorkspace", () => ({
  ReportsWorkspace: ({ legalEntityID, organizationScopeID, onOpenInsights }: {
    legalEntityID?: string; organizationScopeID?: string; onOpenInsights?: (period?: ReportingPeriodQuery) => void;
  }) => <section><h1>Reports &amp; insights</h1><output data-testid="report-scope">{legalEntityID}:{organizationScopeID}</output>
    <button onClick={() => onOpenInsights?.({ start_date: "2026-09-08", end_date: "2026-10-07" })}>Open Insights workspace</button>
  </section>,
}));
vi.mock("./components/notifications/NotificationsWorkspace", async () => {
  const React = await import("react");
  return { NotificationsWorkspace: ({ inbox, tab, onTab, legalEntityID, legalEntityName, actorName, canReadActivity, onOpenWork }: ComponentProps<typeof import("./components/notifications/NotificationsWorkspace")["NotificationsWorkspace"]>) => {
    const [draft, setDraft] = React.useState("");
    return <section aria-label="Notifications workspace">
      <h1>Notifications</h1>
      <output data-testid="notification-tab">{tab}</output>
      <output data-testid="notification-entity">{legalEntityID}:{legalEntityName}</output>
      <output data-testid="notification-actor">{actorName}</output>
      <output data-testid="notification-activity-permission">{String(canReadActivity)}</output>
      <output data-testid="notification-inbox-state">{inbox.state}</output>
      <output data-testid="notification-items">{inbox.items.map((item) => item.title).join(" | ")}</output>
      <label>Notification local draft<input value={draft} onChange={(event) => setDraft(event.target.value)}/></label>
      <button type="button" onClick={() => onTab("inbox")}>Show inbox</button>
      <button type="button" onClick={() => onTab("activity")}>Show activity</button>
      <button type="button" onClick={() => onTab("preferences")}>Show preferences</button>
      <button type="button" onClick={inbox.refresh}>Refresh shared inbox</button>
      <button type="button" onClick={onOpenWork}>Return to Work</button>
    </section>;
  } };
});
vi.mock("./components/insights/InsightsWorkspace", () => ({
  InsightsWorkspace: ({ organizationScopeID, riskLossPeriod, scopeAuthorized, onOpenReports, embedded }: ComponentProps<typeof import("./components/insights/InsightsWorkspace")["InsightsWorkspace"]>) => <section>{embedded ? <h2>Key indicators</h2> : <h1>Insights</h1>}<output data-testid="insights-scope">{organizationScopeID}</output>
    <output data-testid="insights-period">{riskLossPeriod?.start_date}:{riskLossPeriod?.end_date}</output>
    <output data-testid="insights-scope-authorized">{String(scopeAuthorized)}</output>
    {!embedded && <button onClick={onOpenReports}>View reports</button>}
  </section>,
}));
vi.mock("./captureApi", () => ({
  declareWrongCaptureRecipient: vi.fn(),
  reassignCaptureRecipient: vi.fn(),
  uploadInternalCaptureArtifact: vi.fn(),
}));
vi.mock("./api", () => ({
  loadActorWorkDueCounts: vi.fn().mockRejectedValue(new Error("Source counts unavailable in shell test")),
  loadAutomationPolicies: vi.fn().mockResolvedValue([]),
  loadCaptureRequest: vi.fn(),
  loadContext: vi.fn(),
  loadEvidenceRequest: vi.fn(),
  loadEvidenceRequests: vi.fn().mockResolvedValue([]),
  loadEvidenceSources: vi.fn().mockResolvedValue([]),
  loadIntegrity: vi.fn().mockResolvedValue([]),
  loadMatter: vi.fn(),
  loadMatterSummaries: vi.fn().mockResolvedValue({ items: [], generated_at: "2026-08-06T15:00:00Z" }),
  loadPolicies: vi.fn().mockResolvedValue([]),
  loadProgram: vi.fn(),
  loadProgramSummaries: vi.fn().mockResolvedValue({ items: [], generated_at: "2026-08-06T15:00:00Z" }),
  loadProjectionHealth: vi.fn().mockResolvedValue([]),
  loadReadiness: vi.fn(),
  loadToday: vi.fn(),
  loadWorkflowTasks: vi.fn().mockResolvedValue([]),
  reconcileProgramState: vi.fn(),
  resolveAuthority: vi.fn(),
  searchOrganizationScopes: vi.fn().mockResolvedValue({ items: [], has_more: false }),
  submitCaptureRequest: vi.fn(),
  switchLegalEntity: vi.fn(),
}));
vi.mock("./evidenceRequestAdminApi", async (importOriginal) => ({
  ...await importOriginal<typeof import("./evidenceRequestAdminApi")>(),
  listEvidenceRecipientCandidates,
}));

type RuntimeWithCapabilities = RuntimeContext & {
  demo_mode: boolean;
  capabilities: { document_import: boolean; reference_journeys: boolean; oversight_read?: boolean; scope_switch?: boolean; identity_read?: boolean; platform_operations_read?: boolean };
  actor: RuntimeContext["actor"] & { role_codes: string[] };
};

function runtime(demoMode: boolean): RuntimeWithCapabilities {
  return {
    tenant: { id: "bank-demo", name: "Clear Bank" },
    legal_entity: { id: "bank-ng", name: "Clear Bank Nigeria" },
    actor: { id: "role-cro", name: "Chief Risk Officer", role_codes: ["CRO", "EXECUTIVE"] },
    mode: "memory",
    demo_mode: demoMode,
    capabilities: { document_import: true, reference_journeys: demoMode },
  };
}

function primaryLink(label: string) {
  return within(screen.getByRole("complementary", { name: "Primary navigation" }))
    .getByRole("link", { name: label });
}

function switchableRuntime(): RuntimeWithCapabilities {
  return {
    ...runtime(false),
    scope_hierarchy: {
      state: "COMPLETE",
      root: { id: "tenant-uuid", code: "bank-demo", name: "Clear Bank", kind: "ORGANIZATION" },
      current: { id: "entity-ng-uuid", code: "bank-ng", name: "Clear Bank Nigeria", kind: "LEGAL_ENTITY", parent_id: "tenant-uuid", jurisdiction: "NG", current: true },
      legal_entities: [
        { id: "entity-ng-uuid", code: "bank-ng", name: "Clear Bank Nigeria", kind: "LEGAL_ENTITY", parent_id: "tenant-uuid", jurisdiction: "NG", current: true },
        { id: "entity-gh-uuid", code: "bank-gh", name: "Clear Bank Ghana", kind: "LEGAL_ENTITY", parent_id: "tenant-uuid", jurisdiction: "GH" },
      ],
    },
    capabilities: { document_import: true, reference_journeys: false, scope_switch: true },
  };
}

function groupSnapshot(): GroupOversightSnapshot {
  return {
    revision_id: "group-run-1",
    generated_at: "2026-10-03T20:00:00Z",
    projection_version: "group-oversight-v1",
    freshness: "CURRENT",
    coverage: { authorized_children: 2, included_children: 2, missing_children: 0, stale_children: 0, complete: true },
    record_coverage: { population: 10, excluded: 0, unknown: 0 },
    counts: { critical_high: 5, overdue: 2, due_soon: 1, routing_failures: 1, unassigned: 0, outcome_failures: 3 },
    children: [
      {
        legal_entity_id: "entity-ng-uuid", legal_entity_code: "bank-ng", legal_entity_name: "Clear Bank Nigeria",
        jurisdiction: "NG", state: "AVAILABLE", child_snapshot_id: "snapshot-ng", child_generated_at: "2026-10-03T20:00:00Z",
        child_projection_version: "oversight-v5", coverage: { population: 4, excluded: 0, unknown: 0 },
        counts: { critical_high: 2, overdue: 1, due_soon: 1, routing_failures: 0, unassigned: 0, outcome_failures: 1 },
      },
      {
        legal_entity_id: "entity-gh-uuid", legal_entity_code: "bank-gh", legal_entity_name: "Clear Bank Ghana",
        jurisdiction: "GH", state: "AVAILABLE", child_snapshot_id: "snapshot-gh", child_generated_at: "2026-10-03T20:00:00Z",
        child_projection_version: "oversight-v5", coverage: { population: 6, excluded: 0, unknown: 0 },
        counts: { critical_high: 3, overdue: 1, due_soon: 0, routing_failures: 1, unassigned: 0, outcome_failures: 2 },
      },
    ],
  };
}

function evidenceRequest(overrides: Partial<EvidenceRequest> = {}): EvidenceRequest {
  return {
    id: "request-assigned",
    tenant_id: "bank-demo",
    subject_type: "PROGRAM",
    subject_id: "program-1",
    title: "Confirm assigned evidence",
    purpose: "Confirm the completed evidence review.",
    why_you: "The assigned person owns this response.",
    sensitivity: "INTERNAL",
    audience_type: "INTERNAL",
    recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "role-cro", state: "ASSIGNED" },
    created_by: "requester-1",
    estimated_minutes: 3,
    deadline: "2099-08-30T12:00:00Z",
    known_facts: {},
    fields: [],
    status: "READY",
    version: 1,
    created_at: "2026-08-25T12:00:00Z",
    updated_at: "2026-08-25T12:00:00Z",
    ...overrides,
  };
}

function evidenceAttention(requestID: string) {
  return {
    id: "today-evidence",
    type: "EVIDENCE_REQUEST",
    title: "Confirm assigned evidence",
    why_now: "A response is due.",
    scope: "Program evidence",
    state: "Response required",
    evidence: "Known facts included",
    owner: "Assigned respondent",
    due_at: "2099-08-30T12:00:00Z",
    primary_action: "Respond to the request",
    action_target_type: "EVIDENCE_REQUEST" as const,
    action_target_id: requestID,
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((next, fail) => { resolve = next; reject = fail; });
  return { promise, resolve, reject };
}

function secondScope() {
  return {
    ...runtime(false),
    tenant: { id: "bank-second", name: "Second Bank" },
    legal_entity: { id: "bank-second-ng", name: "Second Bank Nigeria" },
  };
}

function notification(id: string, title: string): InAppNotification {
  return {
    id, title, kind: "ASSIGNMENT", summary: "Review the assigned issue.",
    subject_type: "MATTER", subject_id: "matter-1", action_path: "#work/matters/matter-1",
    occurred_at: "2026-10-21T09:00:00Z",
  };
}

beforeAll(() => {
  Object.defineProperty(Element.prototype, "scrollIntoView", { configurable: true, value: vi.fn() });
});

beforeEach(() => {
  window.history.replaceState(null, "", "#today");
  vi.mocked(loadToday).mockResolvedValue({ items: [], generated_at: "2026-08-07T15:00:00Z" });
  vi.mocked(loadEvidenceRequests).mockResolvedValue([]);
  vi.mocked(loadEvidenceRequest).mockRejectedValue(new Error("No evidence request selected"));
  vi.mocked(loadCaptureRequest).mockRejectedValue(new Error("Demo fallback must not be used"));
  vi.mocked(declareWrongCaptureRecipient).mockRejectedValue(new Error("Recipient lifecycle command not configured"));
  vi.mocked(reassignCaptureRecipient).mockRejectedValue(new Error("Recipient lifecycle command not configured"));
  listEvidenceRecipientCandidates.mockRejectedValue(new Error("Recipient candidates not configured"));
  vi.mocked(loadReadiness).mockRejectedValue(new Error("No readiness baseline"));
  vi.mocked(switchLegalEntity).mockReset();
  vi.mocked(switchLegalEntity).mockResolvedValue();
  vi.mocked(loadGroupOversight).mockReset();
  vi.mocked(loadGroupOversight).mockRejectedValue(new Error("Group posture unavailable"));
  vi.mocked(loadNotifications).mockReset();
  vi.mocked(loadNotifications).mockResolvedValue({ items: [], unread_count: 0, as_of: "2026-10-02T09:00:00Z" });
  subscribeInvalidations.mockReset();
  subscribeInvalidations.mockReturnValue(() => {});
});

describe("realtime invalidation integration", () => {
  it("reloads Today after an actor invalidation without a page reload", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    vi.mocked(loadToday)
      .mockResolvedValueOnce({ items: [], generated_at: "2026-08-07T15:00:00Z" })
      .mockResolvedValueOnce({ items: [evidenceAttention("request-live")], generated_at: "2026-08-07T15:01:00Z" });

    render(<App/>);
    await waitFor(() => expect(subscribeInvalidations).toHaveBeenCalledTimes(1));
    const handler = subscribeInvalidations.mock.calls[0]?.[0] as ((event: { revision: string }) => void) | undefined;
    expect(handler).toBeTruthy();

    act(() => handler?.({ revision: "rev-2" }));

    await waitFor(() => expect(loadToday).toHaveBeenCalledTimes(2));
    expect(await screen.findByText("Confirm assigned evidence")).toBeTruthy();
  });

  it("does not open an EventSource in demo mode", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(true));
    render(<App/>);
    await waitFor(() => expect(loadContext).toHaveBeenCalled());
    expect(subscribeInvalidations).not.toHaveBeenCalled();
  });
});

describe("notification shell integration", () => {
  it("reloads notification delivery metadata when verified scope changes", async () => {
    vi.mocked(loadContext).mockResolvedValueOnce(runtime(false)).mockResolvedValueOnce(secondScope());
    vi.mocked(loadNotifications)
      .mockResolvedValueOnce({ items: [], unread_count: 2, as_of: "2026-10-02T09:00:00Z" })
      .mockResolvedValueOnce({ items: [], unread_count: 0, as_of: "2026-10-02T09:01:00Z" });

    const view = render(<App presentation="demo"/>);
    expect(await screen.findByRole("button", { name: "Notifications, 2 unread" })).toBeTruthy();

    view.rerender(<App presentation="live-preview"/>);
    await screen.findAllByText("Second Bank");
    await waitFor(() => expect(loadNotifications).toHaveBeenCalledTimes(2));
    expect(screen.getByRole("button", { name: "Notifications" })).toBeTruthy();
  });

  it("opens the full Notifications route from the bell and shares one inbox owner across local tabs", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    vi.mocked(loadNotifications)
      .mockResolvedValueOnce({ items: [notification("notice-1", "Assigned issue review")], unread_count: 2, as_of: "2026-10-21T09:00:00Z" })
      .mockResolvedValueOnce({ items: [notification("notice-2", "New review assignment")], unread_count: 1, as_of: "2026-10-21T09:01:00Z" });
    render(<App/>);

    fireEvent.click(await screen.findByRole("button", { name: "Notifications, 2 unread" }));
    await screen.findByRole("heading", { name: "Notifications" });
    expect(window.location.hash).toBe("#notifications");
    expect(primaryLink("Work & Approvals").getAttribute("aria-current")).toBe("page");
    expect(primaryLink("Home").getAttribute("aria-current")).toBeNull();
    expect(screen.queryByRole("dialog", { name: /notifications/i })).toBeNull();
    expect(screen.getByTestId("notification-items").textContent).toBe("Assigned issue review");
    expect(loadNotifications).toHaveBeenCalledTimes(1);

    fireEvent.change(screen.getByLabelText("Notification local draft"), { target: { value: "retained local selection" } });
    for (const tab of ["preferences", "activity", "inbox"] as const) {
      fireEvent.click(screen.getByRole("button", { name: `Show ${tab}` }));
      expect(window.location.hash).toBe(tab === "inbox" ? "#notifications" : `#notifications/${tab}`);
      expect(screen.getByTestId("notification-tab").textContent).toBe(tab);
      expect((screen.getByLabelText("Notification local draft") as HTMLInputElement).value).toBe("retained local selection");
      expect(loadNotifications).toHaveBeenCalledTimes(1);
    }

    fireEvent.click(screen.getByRole("button", { name: "Refresh shared inbox" }));
    await screen.findByRole("button", { name: "Notifications, 1 unread" });
    expect(screen.getByTestId("notification-items").textContent).toBe("New review assignment");
    expect(loadNotifications).toHaveBeenCalledTimes(2);
    expect(loadNotifications).toHaveBeenLastCalledWith(expect.objectContaining({ limit: 25, unreadOnly: false }));

    fireEvent.click(screen.getByRole("button", { name: "Return to Work" }));
    expect(window.location.hash).toBe("#work");
    expect(screen.queryByRole("region", { name: "Notifications workspace" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Notifications, 1 unread" }));
    await screen.findByRole("heading", { name: "Notifications" });
    expect(screen.getByTestId("notification-items").textContent).toBe("New review assignment");
    expect(loadNotifications).toHaveBeenCalledTimes(2);
  });

  it.each(["preferences", "activity"] as const)("restores the %s local tab from its URL and follows browser navigation", async (tab) => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    window.history.replaceState(null, "", `#notifications/${tab}`);
    render(<App/>);

    await screen.findByRole("heading", { name: "Notifications" });
    expect(screen.getByTestId("notification-tab").textContent).toBe(tab);
    expect(primaryLink("Work & Approvals").getAttribute("aria-current")).toBe("page");
    await waitFor(() => expect(loadNotifications).toHaveBeenCalledTimes(1));

    act(() => {
      window.history.replaceState(null, "", "#notifications");
      window.dispatchEvent(new PopStateEvent("popstate"));
    });
    expect(screen.getByTestId("notification-tab").textContent).toBe("inbox");
    expect(loadNotifications).toHaveBeenCalledTimes(1);
  });

  it("waits for verified context and does not infer Activity permission from an administrator role", async () => {
    const context = deferred<RuntimeWithCapabilities>();
    vi.mocked(loadContext).mockReturnValue(context.promise);
    window.history.replaceState(null, "", "#notifications/activity");
    render(<App/>);

    expect(screen.getByText("Loading your notification access…")).toBeTruthy();
    expect(screen.queryByRole("region", { name: "Notifications workspace" })).toBeNull();
    expect(loadNotifications).not.toHaveBeenCalled();
    const verified = runtime(false);
    verified.actor.role_codes = ["ADMIN"];
    await act(async () => { context.resolve(verified); });

    await screen.findByRole("heading", { name: "Notifications" });
    expect(screen.getByTestId("notification-activity-permission").textContent).toBe("false");
    expect(screen.getByTestId("notification-entity").textContent).toBe("bank-ng:Clear Bank Nigeria");
    expect(screen.getByTestId("notification-actor").textContent).toBe("Chief Risk Officer");
    await waitFor(() => expect(loadNotifications).toHaveBeenCalledTimes(1));
  });

  it("passes granted Activity access and immediately removes it when the same verified actor loses the capability", async () => {
    const granted = runtime(false);
    granted.capabilities.platform_operations_read = true;
    const revoked = runtime(false);
    revoked.capabilities.platform_operations_read = false;
    vi.mocked(loadContext).mockResolvedValueOnce(granted).mockResolvedValueOnce(revoked);
    window.history.replaceState(null, "", "#notifications/activity");
    const view = render(<App presentation="demo"/>);

    await screen.findByRole("heading", { name: "Notifications" });
    expect(screen.getByTestId("notification-activity-permission").textContent).toBe("true");
    await waitFor(() => expect(loadNotifications).toHaveBeenCalledTimes(1));
    fireEvent.change(screen.getByLabelText("Notification local draft"), { target: { value: "same actor draft" } });
    view.rerender(<App presentation="live-preview"/>);

    await waitFor(() => expect(screen.getByTestId("notification-activity-permission").textContent).toBe("false"));
    expect(window.location.hash).toBe("#notifications/activity");
    expect((screen.getByLabelText("Notification local draft") as HTMLInputElement).value).toBe("same actor draft");
    expect(loadNotifications).toHaveBeenCalledTimes(1);
  });

  it("does not restore a former scope or Activity permission when an older context request finishes last", async () => {
    const oldContext = deferred<RuntimeWithCapabilities>();
    const older = runtime(false);
    older.capabilities.platform_operations_read = true;
    const current = { ...runtime(false), ...secondScope(), actor: { id: "current-reviewer", name: "Current Reviewer", role_codes: ["REVIEWER"] } };
    current.capabilities.platform_operations_read = false;
    vi.mocked(loadContext).mockReturnValueOnce(oldContext.promise).mockResolvedValueOnce(current);
    vi.mocked(loadNotifications).mockResolvedValue({ items: [notification("current", "Current account update")], unread_count: 1, as_of: "2026-10-21T09:02:00Z" });
    window.history.replaceState(null, "", "#notifications/activity");
    const view = render(<App presentation="demo"/>);
    expect(loadNotifications).not.toHaveBeenCalled();
    view.rerender(<App presentation="live-preview"/>);

    await screen.findByRole("button", { name: "Notifications, 1 unread" });
    expect(screen.getByTestId("notification-entity").textContent).toBe("bank-second-ng:Second Bank Nigeria");
    expect(screen.getByTestId("notification-activity-permission").textContent).toBe("false");
    await act(async () => { oldContext.resolve(older); });

    expect(screen.getByTestId("notification-entity").textContent).toBe("bank-second-ng:Second Bank Nigeria");
    expect(screen.getByTestId("notification-actor").textContent).toBe("Current Reviewer");
    expect(screen.getByTestId("notification-activity-permission").textContent).toBe("false");
    expect(loadNotifications).toHaveBeenCalledTimes(1);
  });

  it.each(["tenant", "legal entity", "actor"] as const)("replaces the shared inbox and local workspace state when the verified %s changes", async (dimension) => {
    const initial = runtime(false);
    initial.capabilities.platform_operations_read = true;
    const next = runtime(false);
    next.capabilities.platform_operations_read = false;
    if (dimension === "tenant") next.tenant = { id: "bank-second", name: "Second Bank" };
    if (dimension === "legal entity") next.legal_entity = { id: "bank-gh", name: "Clear Bank Ghana" };
    if (dimension === "actor") next.actor = { id: "role-reviewer", name: "Operations Reviewer", role_codes: ["REVIEWER"] };
    const oldInbox = deferred<NotificationPage>();
    const currentInbox = deferred<NotificationPage>();
    vi.mocked(loadContext).mockResolvedValueOnce(initial).mockResolvedValueOnce(next);
    vi.mocked(loadNotifications).mockReturnValueOnce(oldInbox.promise).mockReturnValueOnce(currentInbox.promise);
    window.history.replaceState(null, "", "#notifications/preferences");
    const view = render(<App presentation="demo"/>);

    await screen.findByRole("heading", { name: "Notifications" });
    await waitFor(() => expect(loadNotifications).toHaveBeenCalledTimes(1));
    const oldSignal = vi.mocked(loadNotifications).mock.calls[0]?.[0]?.signal;
    fireEvent.change(screen.getByLabelText("Notification local draft"), { target: { value: "former scope state" } });
    view.rerender(<App presentation="live-preview"/>);
    await waitFor(() => expect(loadNotifications).toHaveBeenCalledTimes(2));

    expect(oldSignal?.aborted).toBe(true);
    expect((screen.getByLabelText("Notification local draft") as HTMLInputElement).value).toBe("");
    expect(screen.getByTestId("notification-activity-permission").textContent).toBe("false");
    expect(screen.getByTestId("notification-entity").textContent).toBe(`${next.legal_entity.id}:${next.legal_entity.name}`);
    expect(screen.getByTestId("notification-actor").textContent).toBe(next.actor.name);
    expect(screen.getByTestId("notification-inbox-state").textContent).toBe("loading");
    expect(window.location.hash).toBe("#notifications/preferences");

    await act(async () => { currentInbox.resolve({ items: [notification("current", "Current account update")], unread_count: 1, as_of: "2026-10-21T09:02:00Z" }); });
    await screen.findByRole("button", { name: "Notifications, 1 unread" });
    await act(async () => { oldInbox.resolve({ items: [notification("old", "Former account update")], unread_count: 9, as_of: "2026-10-21T09:00:00Z" }); });

    expect(screen.getByTestId("notification-items").textContent).toBe("Current account update");
    expect(screen.queryByText("Former account update")).toBeNull();
    expect(screen.queryByRole("button", { name: "Notifications, 9 unread" })).toBeNull();
    expect(loadNotifications).toHaveBeenCalledTimes(2);
  });
});

describe("legal entity scope selector", () => {
  it("opens Enterprise Setup Organization from the scope selector's Organization & access action", async () => {
    const scoped = switchableRuntime();
    scoped.capabilities.identity_read = true;
    scoped.capabilities.config_read = true;
    vi.mocked(loadContext).mockResolvedValue(scoped);
    render(<App/>);

    fireEvent.click(await screen.findByRole("button", { name: /Organization scope/ }));
    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });
    fireEvent.click(within(dialog).getByRole("button", { name: "Organization & access" }));
    await waitFor(() => expect(window.location.hash).toBe("#configure/access/positions"));
    expect(await screen.findByRole("heading", { name: "Enterprise setup" })).toBeTruthy();
  });

  it("keeps the context bar quiet when only one scope is available", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    render(<App/>);

    await screen.findByText("Clear Bank Nigeria");
    expect(screen.queryByRole("button", { name: /Organization scope/ })).toBeNull();
  });

  it("fails closed when the advertised hierarchy is incomplete", async () => {
    const incomplete = switchableRuntime();
    incomplete.scope_hierarchy = { ...incomplete.scope_hierarchy!, state: "TRUNCATED" };
    vi.mocked(loadContext).mockResolvedValue(incomplete);
    render(<App/>);

    await screen.findByText("Clear Bank Nigeria");
    expect(screen.queryByRole("button", { name: /Organization scope/ })).toBeNull();
  });

  it("places the authorized scope selector beside passive organization identity, not shell actions", async () => {
    vi.mocked(loadContext).mockResolvedValue(switchableRuntime());
    render(<App/>);

    const context = await screen.findByLabelText("Active workspace context");
    const scope = context.querySelector(".context-scope");
    const identity = scope?.querySelector(".context-identity");
    const actions = context.querySelector(".context-role");
    if (!(scope instanceof HTMLElement) || !(identity instanceof HTMLElement) || !(actions instanceof HTMLElement)) {
      throw new Error("Shell identity or action context is missing");
    }
    expect(within(identity).getByText("Clear Bank")).toBeTruthy();
    expect(within(identity).getByText("Clear Bank Nigeria")).toBeTruthy();
    const trigger = within(scope).getByRole("button", { name: /Organization scope/ });
    expect(trigger.textContent).toContain("Clear Bank Nigeria");
    expect(within(actions).queryByRole("button", { name: /Organization scope/ })).toBeNull();
    expect(within(context).queryByText("Non-production data")).toBeNull();
  });

  it("opens Group Home from the organization root without changing the legal-entity session", async () => {
    const scoped = switchableRuntime();
    scoped.capabilities.oversight_read = true;
    vi.mocked(loadContext).mockResolvedValue(scoped);
    vi.mocked(loadGroupOversight).mockResolvedValue(groupSnapshot());
    vi.mocked(switchLegalEntity).mockClear();
    window.history.replaceState(null, "", "#oversight");
    render(<App/>);

    fireEvent.click(await screen.findByRole("button", { name: /Organization scope/ }));
    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });
    fireEvent.click(await within(dialog).findByRole("button", { name: "Group, Clear Bank" }));

    expect(await screen.findByRole("heading", { name: "Group posture" })).toBeTruthy();
    expect(window.location.hash).toBe("#oversight?scope=group");
    expect(screen.getByRole("button", { name: /Organization scope/ }).textContent).toContain("Clear Bank · Group");
    expect(switchLegalEntity).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "Open current OpCo" }));
    await waitFor(() => expect(window.location.hash).toBe("#oversight"));
    expect(switchLegalEntity).not.toHaveBeenCalled();
  });

  it("fails closed when a direct Group route is not authorized", async () => {
    vi.mocked(loadContext).mockResolvedValue(switchableRuntime());
    vi.mocked(loadGroupOversight).mockRejectedValue(new Error("forbidden"));
    window.history.replaceState(null, "", "#oversight?scope=group&metric=overdue");
    render(<App/>);

    await waitFor(() => expect(window.location.hash).toBe("#oversight?metric=overdue"));
    expect(screen.queryByRole("heading", { name: "Group posture" })).toBeNull();
  });

  it("shows server-authorized organization scopes even when there is only one legal entity", async () => {
    const scoped = switchableRuntime();
    const entity = scoped.scope_hierarchy!.legal_entities[0]!;
    scoped.scope_hierarchy = {
      ...scoped.scope_hierarchy!,
      legal_entities: [entity],
      organization_scopes: [
        { id: "scope-bank", code: "BANK", name: "BANK", kind: "ORGANIZATION_UNIT", parent_id: entity.id, department_path: ["BANK"] },
        { id: "scope-risk", code: "RISK", name: "RISK", kind: "DEPARTMENT", parent_id: "scope-bank", department_path: ["BANK", "RISK"], filterable: true },
        { id: "scope-operations", code: "OPERATIONS", name: "OPERATIONS", kind: "DEPARTMENT", parent_id: "scope-bank", department_path: ["BANK", "OPERATIONS"] },
        { id: "scope-payments", code: "PAYMENTS", name: "PAYMENTS", kind: "DEPARTMENT", parent_id: "scope-operations", department_path: ["BANK", "OPERATIONS", "PAYMENTS"] },
      ],
    };
    scoped.capabilities.scope_switch = false;
    vi.mocked(loadContext).mockResolvedValue(scoped);
    render(<App/>);

    const trigger = await screen.findByRole("button", { name: /Organization scope/ });
    fireEvent.click(trigger);
    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });

    expect(within(dialog).getByText("BANK")).toBeTruthy();
    expect(within(dialog).getByText("OPERATIONS")).toBeTruthy();
    expect(within(dialog).getByText("PAYMENTS")).toBeTruthy();
    expect(within(dialog).getByRole("button", { name: /RISK/ })).toBeTruthy();
  });

  it("selects a filterable organization scope from the persistent Scope control", async () => {
    const scoped = switchableRuntime();
    scoped.scope_hierarchy!.organization_scopes = [
      { id: "scope-bank", code: "BANK", name: "BANK", kind: "ORGANIZATION_UNIT", parent_id: scoped.scope_hierarchy!.current.id, department_path: ["BANK"] },
      { id: "scope-risk", code: "RISK", name: "Risk", kind: "DEPARTMENT", parent_id: "scope-bank", department_path: ["BANK", "RISK"], filterable: true },
    ];
    scoped.capabilities.oversight_read = true;
    vi.mocked(loadContext).mockResolvedValue(scoped);
    render(<App/>);

    fireEvent.click(await screen.findByRole("button", { name: /Organization scope/ }));
    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });
    fireEvent.click(within(dialog).getByRole("button", { name: /Risk/ }));

    await waitFor(() => expect(screen.getByRole("button", { name: /Organization scope/ }).textContent).toContain("Risk"));
  });

  it("shows only server-authorized legal entities and retains the current selection", async () => {
    vi.mocked(loadContext).mockResolvedValue(switchableRuntime());
    render(<App/>);

    const trigger = await screen.findByRole("button", { name: /Organization scope/ });
    expect(trigger.textContent).toContain("Clear Bank Nigeria");
    fireEvent.click(trigger);
    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });
    expect(within(dialog).getByText("Clear Bank")).toBeTruthy();
    expect(within(dialog).getByRole("button", { name: /Clear Bank Nigeria/ }).getAttribute("aria-pressed")).toBe("true");
    expect(within(dialog).getByRole("button", { name: /Clear Bank Ghana/ })).toBeTruthy();
    expect(within(dialog).queryByRole("button", { name: /^Clear Bank$/ })).toBeNull();
  });

  it("keeps the current scope and shows recovery copy when session switching fails", async () => {
    vi.mocked(loadContext).mockResolvedValue(switchableRuntime());
    vi.mocked(switchLegalEntity).mockRejectedValueOnce(new Error("unavailable"));
    render(<App/>);

    fireEvent.click(await screen.findByRole("button", { name: /Organization scope/ }));
    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });
    fireEvent.click(within(dialog).getByRole("button", { name: /Clear Bank Ghana/ }));

    await waitFor(() => expect(switchLegalEntity).toHaveBeenCalledWith("entity-gh-uuid"));
    expect((await screen.findByRole("alert")).textContent).toContain("Legal entity could not be changed. Try again.");
    expect(screen.getByRole("button", { name: /Organization scope/ }).textContent).toContain("Clear Bank Nigeria");
  });
});

describe("scope and direct navigation integration", () => {
  it("keeps verified legal-entity switching available while navigating Risk and Loss destinations", async () => {
    vi.mocked(loadContext).mockResolvedValue(switchableRuntime());
    window.history.replaceState(null, "", "#programs");
    render(<App/>);

    const scopeTrigger = await screen.findByRole("button", { name: /Organization scope/ });
    expect(scopeTrigger.textContent).toContain("Clear Bank Nigeria");

    fireEvent.click(primaryLink("Risk & Losses"));

    expect(await screen.findByRole("heading", { name: "Risks" })).toBeTruthy();
    expect(window.location.hash).toBe("#risks");
    expect(screen.getByRole("button", { name: /Organization scope/ }).textContent).toContain("Clear Bank Nigeria");

    fireEvent.click(primaryLink("Loss events"));
    expect(await screen.findByRole("heading", { name: "Losses" })).toBeTruthy();
    expect(window.location.hash).toBe("#losses");
    expect(screen.getByRole("button", { name: /Organization scope/ }).textContent).toContain("Clear Bank Nigeria");

    fireEvent.click(screen.getByRole("button", { name: /Organization scope/ }));
    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });
    expect(within(dialog).getByRole("button", { name: /Clear Bank Ghana/ })).toBeTruthy();
  });
});

describe("organization scope navigation persistence", () => {
  it("keeps an authoritative area across Programs, Risks and Losses and clears it on unsupported destinations", async () => {
    const scoped = switchableRuntime();
    scoped.scope_hierarchy!.organization_scopes = [
      { id: "scope-bank", code: "BANK", name: "BANK", kind: "ORGANIZATION_UNIT", parent_id: scoped.scope_hierarchy!.current.id, department_path: ["BANK"] },
      { id: "scope-risk", code: "RISK", name: "Risk", kind: "DEPARTMENT", parent_id: "scope-bank", department_path: ["BANK", "RISK"], filterable: true },
    ];
    vi.mocked(loadContext).mockResolvedValue(scoped);
    window.history.replaceState(null, "", "#programs");
    render(<App/>);

    fireEvent.click(await screen.findByRole("button", { name: /Organization scope/ }));
    let dialog = await screen.findByRole("dialog", { name: "Change organization scope" });
    fireEvent.click(within(dialog).getByRole("button", { name: /Risk/ }));
    expect(window.location.hash).toBe("#programs");
    await waitFor(() => expect(screen.getByRole("button", { name: /Organization scope/ }).textContent).toContain("Risk"));

    fireEvent.click(primaryLink("Risk & Losses"));
    expect(await screen.findByRole("heading", { name: "Risks" })).toBeTruthy();
    await waitFor(() => expect(screen.getByRole("button", { name: /Organization scope/ }).textContent).toContain("Risk"));
    expect(screen.getByTestId("risk-scope").textContent).toBe("scope-risk");

    fireEvent.click(primaryLink("Loss events"));
    expect(await screen.findByRole("heading", { name: "Losses" })).toBeTruthy();
    await waitFor(() => expect(screen.getByRole("button", { name: /Organization scope/ }).textContent).toContain("Risk"));
    expect(screen.getByTestId("loss-scope").textContent).toBe("scope-risk:BANK / RISK");

    fireEvent.click(primaryLink("Vendors"));
    await waitFor(() => expect(screen.getByRole("button", { name: /Organization scope/ }).textContent).toContain("Clear Bank Nigeria"));
    expect(screen.getByRole("button", { name: /Organization scope/ }).textContent).not.toContain("Risk");
  });
});

describe("Risk and loss local navigation", () => {
  it("keeps one local menu while navigating all four real collection routes", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    window.history.replaceState(null, "", "#risks");
    render(<App/>);
    await screen.findByRole("heading", { name: "Risk & Losses", level: 1 });
    const menu = screen.getByRole("tablist", { name: "Risk and loss sections" });
    expect(within(menu).getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["Risks", "RCSA", "Key indicators", "Loss events"]);

    for (const [label, hash, heading] of [
      ["RCSA", "#rcsa", "Review cycles"],
      ["Key indicators", "#insights", "Key indicators"],
      ["Loss events", "#losses", "Losses"],
      ["Risks", "#risks", "Risks"],
    ]) {
      fireEvent.click(within(menu).getByRole("tab", { name: label! }));
      expect(await screen.findByRole("heading", { name: heading!, level: 2 })).toBeTruthy();
      expect(window.location.hash).toBe(hash);
      expect(screen.getByRole("tablist", { name: "Risk and loss sections" })).toBe(menu);
      expect(within(menu).getByRole("tab", { name: label!, selected: true })).toBeTruthy();
      expect(primaryLink("Risk & Losses").getAttribute("aria-current")).toBe("page");
      expect(screen.getAllByRole("heading", { level: 1 })).toHaveLength(1);
      expect(screen.getAllByRole("button", { name: "View reports" })).toHaveLength(1);
    }
  });

  it.each([
    ["risks", "risk-target", "Risks"],
    ["losses", "loss-target", "Loss events"],
  ])("restores the %s collection menu after history removes the record target", async (route, targetTestID, tabLabel) => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    window.history.replaceState(null, "", `#${route}/record-1`);
    render(<App/>);
    expect(await screen.findByTestId(targetTestID)).toHaveProperty("textContent", "record-1");
    expect(screen.queryByRole("tablist", { name: "Risk and loss sections" })).toBeNull();

    act(() => {
      window.history.replaceState(null, "", `#${route}`);
      window.dispatchEvent(new PopStateEvent("popstate"));
    });
    expect(await screen.findByRole("heading", { name: "Risk & Losses", level: 1 })).toBeTruthy();
    expect(screen.getByRole("tab", { name: tabLabel, selected: true })).toBeTruthy();
    expect(screen.getByTestId(targetTestID).textContent).toBe("");
  });
});

describe("Reports organization scope", () => {
  const areaID = "8f100000-0000-4000-8000-000000000001";
  beforeEach(() => {
    const scoped = switchableRuntime();
    scoped.scope_hierarchy!.organization_scopes = [
      { id: areaID, code: "RISK", name: "Risk", kind: "DEPARTMENT", parent_id: scoped.scope_hierarchy!.current.id, department_path: ["BANK", "RISK"], filterable: true },
    ];
    vi.mocked(loadContext).mockResolvedValue(scoped);
  });

  it("preserves the verified organization area when opening Reports from Programs", async () => {
    window.history.replaceState(null, "", "#programs");
    render(<App/>);
    fireEvent.click(await screen.findByRole("button", { name: /Organization scope/ }));
    fireEvent.click(within(await screen.findByRole("dialog", { name: "Change organization scope" })).getByRole("button", { name: /Risk/ }));
    fireEvent.click(primaryLink("Reports"));

    expect(await screen.findByRole("heading", { name: "Reports & insights" })).toBeTruthy();
    expect(window.location.hash).toBe("#reports");
    expect(screen.getByTestId("report-scope").textContent).toBe(`bank-ng:${areaID}`);
    expect(screen.getByRole("button", { name: /Organization scope/ }).textContent).toContain("Risk");
  });

  it("keeps Reports open when changing its area and retains that selection through Key indicators", async () => {
    window.history.replaceState(null, "", "#reports");
    render(<App/>);
    await screen.findByRole("heading", { name: "Reports & insights" });
    fireEvent.click(await screen.findByRole("button", { name: /Organization scope/ }));
    fireEvent.click(within(await screen.findByRole("dialog", { name: "Change organization scope" })).getByRole("button", { name: /Risk/ }));

    expect(window.location.hash).toBe("#reports");
    expect(screen.getByTestId("report-scope").textContent).toBe(`bank-ng:${areaID}`);
    fireEvent.click(primaryLink("Risk & Losses"));
    expect(await screen.findByRole("heading", { name: "Risks" })).toBeTruthy();
    fireEvent.click(primaryLink("Key indicators"));
    expect(await screen.findByTestId("insights-scope")).toHaveProperty("textContent", areaID);
    expect(window.location.hash).toBe("#insights");
    expect(primaryLink("Risk & Losses").getAttribute("aria-current")).toBe("page");
    fireEvent.click(screen.getByRole("button", { name: "View reports" }));
    expect(await screen.findByTestId("report-scope")).toHaveProperty("textContent", `bank-ng:${areaID}`);
    expect(window.location.hash).toBe("#reports");

    fireEvent.click(screen.getByRole("button", { name: /Organization scope/ }));
    fireEvent.click(within(await screen.findByRole("dialog", { name: "Change organization scope" })).getByRole("button", { name: /Clear Bank Nigeria/ }));
    expect(screen.getByTestId("report-scope").textContent).toBe("bank-ng:");
    expect(window.location.hash).toBe("#reports");
    expect(switchLegalEntity).not.toHaveBeenCalled();
  });

  it("preserves the verified report period and scope in the Insights route", async () => {
    window.history.replaceState(null, "", "#reports");
    render(<App/>);
    await screen.findByRole("heading", { name: "Reports & insights" });
    fireEvent.click(await screen.findByRole("button", { name: /Organization scope/ }));
    fireEvent.click(within(await screen.findByRole("dialog", { name: "Change organization scope" })).getByRole("button", { name: /Risk/ }));
    fireEvent.click(screen.getByRole("button", { name: "Open Insights workspace" }));

    expect(await screen.findByTestId("insights-period")).toHaveProperty("textContent", "2026-09-08:2026-10-07");
    expect(screen.getByTestId("insights-scope").textContent).toBe(areaID);
    expect(screen.getByTestId("insights-scope-authorized").textContent).toBe("true");
    expect(window.location.hash).toBe(`#insights?view=risk-loss&start_date=2026-09-08&end_date=2026-10-07&legal_entity_id=bank-ng&organization_scope_id=${areaID}`);
    expect(primaryLink("Reports").getAttribute("aria-current")).toBe("page");
    expect(primaryLink("Risk & Losses").getAttribute("aria-current")).toBeNull();
    expect(screen.queryByRole("tablist", { name: "Risk and loss sections" })).toBeNull();

    act(() => window.dispatchEvent(new HashChangeEvent("hashchange")));
    expect(screen.getByTestId("insights-period").textContent).toBe("2026-09-08:2026-10-07");
    expect(screen.getByTestId("insights-scope-authorized").textContent).toBe("true");
    expect(primaryLink("Reports").getAttribute("aria-current")).toBe("page");
    fireEvent.click(screen.getByRole("button", { name: "View reports" }));
    expect(await screen.findByTestId("report-scope")).toHaveProperty("textContent", `bank-ng:${areaID}`);
  });
});

describe("runtime navigation", () => {
  it("opens Templates from the direct Forms link after a section-only change", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    window.history.replaceState(null, "", "#forms");
    const app = render(<App/>);
    await screen.findByRole("tab", { name: "Templates", selected: true }, { timeout: 5000 });

    for (let visit = 0; visit < 2; visit++) {
      fireEvent.click(screen.getByRole("tab", { name: "Documents" }));
      expect(screen.getByRole("tabpanel", { name: "Documents" })).toBeTruthy();
      expect(window.location.hash).toBe("#forms?section=documents");

      fireEvent.click(primaryLink("Programs"));
      await screen.findByRole("heading", { name: "Programs" });
      expect(window.location.hash).toBe("#programs");

      fireEvent.click(primaryLink("Forms"));
      expect(window.location.hash).toBe("#forms");
      expect(await screen.findByRole("tab", { name: "Templates", selected: true })).toBeTruthy();
      expect(screen.getByRole("tabpanel", { name: "Templates" })).toBeTruthy();
    }

    app.unmount();
    render(<App/>);
    expect(await screen.findByRole("tab", { name: "Templates", selected: true })).toBeTruthy();
  });

  it("restores Documents when history remounts Forms after leaving through app navigation", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    window.history.replaceState(null, "", "#forms");
    render(<App/>);
    await screen.findByRole("tab", { name: "Templates", selected: true }, { timeout: 5000 });
    fireEvent.click(screen.getByRole("tab", { name: "Documents" }));
    fireEvent.click(primaryLink("Home"));
    expect(screen.queryByRole("tab", { name: "Documents" })).toBeNull();

    window.history.replaceState(null, "", "#forms?section=documents");
    fireEvent(window, new Event("popstate"));
    fireEvent(window, new Event("hashchange"));
    expect(await screen.findByRole("tab", { name: "Documents", selected: true })).toBeTruthy();
    expect(screen.getByRole("tabpanel", { name: "Documents" })).toBeTruthy();
    expect(primaryLink("Forms").getAttribute("aria-current")).toBe("page");
  });

  it("preserves template filters when app target navigation opens and closes form detail", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    vi.mocked(loadFormTemplatePage).mockResolvedValueOnce({ items: [{ template: {
      id: "template-a", tenant_id: "bank-demo", legal_entity_id: "bank-ng", code: "VENDOR", name: "Vendor due diligence",
      purpose: "Collect current vendor evidence.", status: "DRAFT", is_current: false, version: 1,
      created_at: "2026-08-27T09:00:00Z", updated_at: "2026-08-27T10:00:00Z", sensitivity: "INTERNAL",
      scoring_mode: "NONE", presentation: { default_mode: "AUTOMATIC", allow_mode_switch: true },
      sections: [{ id: "general", title: "General" }],
      fields: [{ id: "question_1", section_id: "general", label: "Registered name", type: "short_text", required: true }],
    } }] });
    window.history.replaceState(null, "", "#forms?search=vendor&limit=50");
    render(<App/>);
    fireEvent.click(await screen.findByRole("button", { name: "Details for Vendor due diligence" }, { timeout: 5000 }));
    expect(await screen.findByRole("dialog", { name: "Selected form template" })).toBeTruthy();
    expect(window.location.hash).toBe("#forms/template-a?search=vendor&limit=50");
    fireEvent.click(screen.getByRole("button", { name: "Close form detail" }));
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Selected form template" })).toBeNull());
    expect(window.location.hash).toBe("#forms?search=vendor&limit=50");
    expect((screen.getByRole("searchbox") as HTMLInputElement).value).toBe("vendor");
    expect(screen.getByRole("tab", { name: "Templates", selected: true })).toBeTruthy();
  });

  it("does not replace an unavailable actor queue with static sample work in demo presentation", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(true));
    vi.mocked(loadToday).mockRejectedValue(new Error("Today projection unavailable"));

    render(<App presentation="demo"/>);

    expect(await screen.findByRole("heading", { name: "Assigned work is unavailable" })).toBeTruthy();
    expect(screen.queryByText("Review proposed digital-channel requirements")).toBeNull();
  });

  it("keeps import capability in administration and removes reference tooling when demo mode is off", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    render(<App />);

    expect(primaryLink("Forms")).toBeTruthy();
    await waitFor(() => expect(document.documentElement.dataset.clearsightDemo).toBe("off"));
    expect(await screen.findByLabelText("Administration")).toBeTruthy();
    expect(screen.queryByRole("button", { name: /Imports/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /Explore/ })).toBeNull();
    expect(screen.queryByText("Demo environment")).toBeNull();
  });

  it("opens Work on the assigned queue without loading the evidence workspace", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention("request-assigned")], generated_at: "2026-08-07T15:00:00Z" });
    vi.mocked(loadEvidenceRequests).mockClear();
    render(<App />);

    fireEvent.click(primaryLink("Work & Approvals"));

    expect(window.location.hash).toBe("#work");
    expect(await screen.findByRole("tab", { name: "My work", selected: true })).toBeTruthy();
    expect(within(screen.getByRole("table", { name: "Assigned work" })).getByText("Confirm assigned evidence")).toBeTruthy();
    expect(within(screen.getByRole("complementary", { name: "Selected work" })).getByRole("heading", { name: "Confirm assigned evidence" })).toBeTruthy();
    expect(loadEvidenceRequests).not.toHaveBeenCalled();
  });

  it("shows organization oversight only when the verified runtime grants oversight read", async () => {
    vi.mocked(loadContext).mockResolvedValue({ ...runtime(false), capabilities: { ...runtime(false).capabilities, oversight_read: true } });
    render(<App />);

    expect(await screen.findByRole("tab", { name: "Oversight" })).toBeTruthy();
    expect(primaryLink("Home")).toBeTruthy();
  });

  it("keeps legacy Today work under the explicit My work Home intent", async () => {
    vi.mocked(loadContext).mockResolvedValue({ ...runtime(false), capabilities: { ...runtime(false).capabilities, oversight_read: true } });
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention("request-assigned")], generated_at: "2026-08-07T15:00:00Z" });

    render(<App />);

    await waitFor(() => expect(window.location.hash).toBe("#oversight"));
    expect(screen.queryByText("Confirm assigned evidence")).toBeNull();
    fireEvent.click(await screen.findByRole("tab", { name: "My work" }));
    expect(window.location.hash).toBe("#oversight?tab=my-work");
    expect(await screen.findByText("Confirm assigned evidence")).toBeTruthy();
    const primaryNavigation = screen.getByRole("complementary", { name: "Primary navigation" });
    expect(within(primaryNavigation).queryByRole("link", { name: "Today" })).toBeNull();
    expect(primaryLink("Home").getAttribute("aria-current")).toBe("page");
  });

  it("keeps the Home destination scoped to assigned work without organization oversight access", async () => {
    vi.mocked(loadContext).mockResolvedValue({
      ...runtime(false),
      actor: { id: "system-admin", name: "System Administrator", role_codes: ["SYSTEM_ADMIN"] },
      capabilities: { ...runtime(false).capabilities, oversight_read: false },
    });
    render(<App />);

    await screen.findByText("Nothing needs your action right now");
    expect(screen.queryByRole("tab", { name: "Oversight" })).toBeNull();
    expect(primaryLink("Home")).toBeTruthy();
  });

  it("exposes the source menu as direct destinations without duplicate Portfolio navigation", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    window.history.replaceState(null, "", "#forms");
    render(<App />);

    const primary = screen.getByRole("complementary", { name: "Primary navigation" });
    expect(primaryLink("Forms").getAttribute("aria-current")).toBe("page");
    await within(primary).findByRole("link", { name: "Settings" });
    expect(within(primary).getAllByRole("link").map((link) => link.getAttribute("href"))).toEqual([
      "#oversight", "#programs/library", "#programs", "#forms", "#vendors", "#risks", "#ropa", "#work", "#reports", "#configure",
    ]);
    expect(within(primary).queryByRole("link", { name: "Portfolio" })).toBeNull();
    expect(screen.queryByRole("navigation", { name: "Portfolio lenses" })).toBeNull();
    expect(screen.queryByRole("navigation", { name: "Mobile navigation" })).toBeNull();
  });

  it("keeps Reports directly available in primary navigation", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    render(<App />);

    const primaryNavigation = screen.getByRole("complementary", { name: "Primary navigation" });
    expect(await within(primaryNavigation).findByRole("link", { name: "Reports" })).toBeTruthy();
  });

  it("navigates between Frameworks and Programs even though they share one route family", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    window.history.replaceState(null, "", "#oversight");
    render(<App />);

    fireEvent.click(primaryLink("Frameworks"));
    expect(window.location.hash).toBe("#programs/library");
    expect(await screen.findByRole("heading", { name: "Framework Library" })).toBeTruthy();
    expect(primaryLink("Frameworks").getAttribute("aria-current")).toBe("page");
    expect(primaryLink("Programs").getAttribute("aria-current")).toBeNull();

    fireEvent.click(primaryLink("Programs"));

    expect(window.location.hash).toBe("#programs");
    expect(await screen.findByRole("heading", { name: "Programs" })).toBeTruthy();
    expect(primaryLink("Programs").getAttribute("aria-current")).toBe("page");
    expect(primaryLink("Frameworks").getAttribute("aria-current")).toBeNull();

    fireEvent.click(primaryLink("Frameworks"));
    expect(window.location.hash).toBe("#programs/library");
    expect(await screen.findByRole("heading", { name: "Framework Library" })).toBeTruthy();
  });

  it("opens Vendors directly while keeping its route stable", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    render(<App />);

    fireEvent.click(primaryLink("Vendors"));

    expect(window.location.hash).toBe("#vendors");
    expect(primaryLink("Vendors").getAttribute("aria-current")).toBe("page");
  });

  it("restores exact risk and loss targets and menu selection on browser history events", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    window.history.replaceState(null, "", "#risks/risk%2F1");
    render(<App/>);

    expect(await screen.findByTestId("risk-target")).toHaveProperty("textContent", "risk/1");
    expect(primaryLink("Risk & Losses").getAttribute("aria-current")).toBe("page");
    expect(primaryLink("Risks").getAttribute("aria-current")).toBe("location");

    act(() => {
      window.history.replaceState(null, "", "#losses/loss%2F1");
      window.dispatchEvent(new PopStateEvent("popstate"));
    });
    expect(await screen.findByTestId("loss-target")).toHaveProperty("textContent", "loss/1");
    expect(primaryLink("Loss events").getAttribute("aria-current")).toBe("location");
    expect(primaryLink("Risks").getAttribute("aria-current")).toBeNull();

    act(() => {
      window.history.replaceState(null, "", "#risks/risk%2F1");
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });
    expect(await screen.findByTestId("risk-target")).toHaveProperty("textContent", "risk/1");
    expect(primaryLink("Risks").getAttribute("aria-current")).toBe("location");
    expect(primaryLink("Loss events").getAttribute("aria-current")).toBeNull();
  });

  it("shows Settings only after runtime verification and opens the existing configuration route", async () => {
    const context = deferred<RuntimeContext>();
    vi.mocked(loadContext).mockReturnValueOnce(context.promise);
    render(<App/>);
    const primary = screen.getByRole("complementary", { name: "Primary navigation" });
    expect(within(primary).queryByRole("link", { name: "Settings" })).toBeNull();

    await act(async () => { context.resolve(runtime(false)); });
    fireEvent.click(await within(primary).findByRole("link", { name: "Settings" }));

    expect(window.location.hash).toBe("#configure");
    // The real lazy workspace can need more than one second for its first module transform.
    expect(await screen.findByRole("heading", { name: "Configuration" }, { timeout: 5000 })).toBeTruthy();
    expect(primaryLink("Settings").getAttribute("aria-current")).toBe("page");
  });

  it("keeps Settings hidden and rejects its direct route when configuration is denied", async () => {
    const context = runtime(false);
    vi.mocked(loadContext).mockResolvedValue({ ...context, capabilities: { ...context.capabilities, config_read: false } });
    window.history.replaceState(null, "", "#configure");
    render(<App/>);

    await waitFor(() => expect(window.location.hash).toBe("#oversight"));
    const primary = screen.getByRole("complementary", { name: "Primary navigation" });
    expect(within(primary).queryByRole("link", { name: "Settings" })).toBeNull();
    expect(screen.queryByLabelText("Administration")).toBeNull();
    expect(screen.queryByRole("heading", { name: "Configuration" })).toBeNull();
    await waitFor(() => expect(primaryLink("Home").getAttribute("aria-current")).toBe("page"));
  });

  it("does not advance an empty attention or routing guide destination", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    render(<App />);

    fireEvent.click(await screen.findByRole("button", { name: "Review first assigned item" }));
    expect((await screen.findByRole("alert")).textContent).toContain("No eligible item is available");
    expect(window.location.hash).toBe("#oversight");

    fireEvent.click(screen.getByRole("button", { name: "Check approval route" }));
    expect((await screen.findByRole("alert")).textContent).toContain("No eligible item is available");
    expect(window.location.hash).toBe("#oversight");
  });

  it("does not open a guide response without a revalidated actor-eligible request", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    render(<App />);

    fireEvent.click(await screen.findByRole("button", { name: "Open assigned response" }));
    expect((await screen.findByRole("alert")).textContent).toContain("No eligible item is available");
    expect(screen.queryByRole("heading", { name: "Confirm assigned evidence" })).toBeNull();
    expect(window.location.hash).toBe("#oversight");
  });

  it("keeps a guide response step open when eligibility is revoked on revalidation", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    const assigned = evidenceRequest();
    const returned = evidenceRequest({
      recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "role-cro", state: "REASSIGNMENT_REQUIRED", revision: 2 },
      version: 2,
    });
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention(assigned.id)], generated_at: "2026-10-16T09:00:00Z" });
    vi.mocked(loadEvidenceRequest).mockResolvedValueOnce(assigned).mockResolvedValueOnce(returned);
    render(<App />);

    await screen.findAllByRole("button", { name: "Respond to evidence request" });
    fireEvent.click(screen.getByRole("button", { name: "Open assigned response" }));

    await waitFor(() => expect(loadEvidenceRequest).toHaveBeenCalledWith(assigned.id, "capture_revalidation"));
    expect(await screen.findByText("This guide step could not be opened. Try again.")).toBeTruthy();
    expect(await screen.findByRole("heading", { name: "You cannot open this request" })).toBeTruthy();
  });

  it("does not navigate a first-run guide into a disabled Configure workspace", async () => {
    const context = runtime(false);
    vi.mocked(loadContext).mockResolvedValue({ ...context, capabilities: { ...context.capabilities, config_read: false } });
    render(<App />);

    fireEvent.click(await screen.findByRole("button", { name: "Open guided configuration" }));
    expect((await screen.findByRole("alert")).textContent).toContain("No eligible item is available");
    expect(window.location.hash).toBe("#oversight");
  });

  it("opens the first eligible attention target through the existing route", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    vi.mocked(loadToday).mockResolvedValue({
      generated_at: "2026-10-16T10:00:00Z",
      items: [
        { id: "incomplete", type: "MATTER", title: "Unrouted item", why_now: "Unassigned", scope: "Risk", state: "OPEN", evidence: "", owner: "",
          due_at: "2099-01-01T00:00:00Z", primary_action: "Review item" },
        { id: "program", type: "PROGRAM", title: "Review Program", why_now: "Approval due", scope: "Bank", state: "OPEN", evidence: "", owner: "CRO",
          due_at: "2099-01-01T00:00:00Z", primary_action: "Review Program", action_target_type: "PROGRAM", action_target_id: "program-valid" },
      ],
    });
    render(<App />);

    await screen.findAllByText("Review Program");
    fireEvent.click(screen.getByRole("button", { name: "Review first assigned item" }));
    await screen.findByRole("region", { name: "Program record" });
    expect(screen.queryByRole("heading", { name: "Programs" })).toBeNull();
    expect(window.location.hash).toBe("#programs/program-valid/overview");
  });

  it("passes the due-diligence guide intent to the Vendors workspace", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    render(<App />);

    fireEvent.click(await screen.findByRole("button", { name: "Review due diligence" }));
    expect((await screen.findByTestId("vendor-guide-intent")).textContent).toBe("open-vendor-due-diligence");
    expect(window.location.hash).toBe("#vendors");
  });

  it("passes the next-action guide intent to the Vendors workspace", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    render(<App />);

    fireEvent.click(await screen.findByRole("button", { name: "Open next vendor task" }));
    expect((await screen.findByTestId("vendor-guide-intent")).textContent).toBe("open-vendor-next-action");
    expect(window.location.hash).toBe("#vendors");
  });

  it("keeps the Vendors workspace usable while its guide is open", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    render(<App />);

    fireEvent.click(primaryLink("Vendors"));

    expect((await screen.findByTestId("onboarding-surface")).textContent).toBe("VENDORS");
    expect(screen.getByRole("complementary", { name: "Vendor guide" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Review vendor request" })).toBeTruthy();
  });

  it("preserves the selected vendor when a guide starts from a vendor deep link", async () => {
    window.history.replaceState(null, "", "#vendors/relationship-b");
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    render(<App />);

    fireEvent.click(await screen.findByRole("button", { name: "Review due diligence" }));
    expect((await screen.findByTestId("vendor-target")).textContent).toBe("relationship-b");
    expect(window.location.hash).toBe("#vendors/register/relationship-b");
  });

  it("cancels a slow vendor guide action when navigation leaves Vendors", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    render(<App />);

    fireEvent.click(await screen.findByRole("button", { name: "Review due diligence" }));
    await screen.findByTestId("vendor-guide-intent");
    fireEvent.click(primaryLink("Home"));

    await waitFor(() => expect((screen.getByRole("button", { name: "Review due diligence" }) as HTMLButtonElement).disabled).toBe(false));
    expect(screen.getByRole("alert").textContent).toContain("This guide step could not be opened. Try again.");
  });

  it("opens the exact evidence request selected from the vendor relationship", async () => {
    const request: EvidenceRequest = {
      id: "request-vendor-1", tenant_id: "bank-demo", subject_type: "VENDOR_RELATIONSHIP", subject_id: "relationship-1",
      title: "Vendor due diligence request", purpose: "Collect the current vendor response.", why_you: "Relationship owner review",
      sensitivity: "CONFIDENTIAL", audience_type: "VENDOR", estimated_minutes: 12, deadline: "2026-09-20T17:00:00Z",
      known_facts: { vendor: "Acme Processing Limited" }, fields: [], status: "READY", version: 1,
      created_at: "2026-08-26T12:00:00Z", updated_at: "2026-08-26T12:00:00Z",
    };
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    vi.mocked(loadEvidenceRequest).mockResolvedValue(request);
    render(<App />);

    fireEvent.click(primaryLink("Vendors"));
    fireEvent.click(await screen.findByRole("button", { name: "Review vendor request" }));

    expect(window.location.hash).toBe("#work/evidence/request-vendor-1");
    expect(await screen.findByText("Vendor due diligence request")).toBeTruthy();
    expect(loadEvidenceRequest).toHaveBeenCalledWith("request-vendor-1");
  });

  it("keeps stakeholder reference journeys secondary when explicit demo presentation is on", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(true));
    render(<App presentation="demo"/>);

    await screen.findByText("Demo environment");
    expect(screen.queryByText("Stakeholder demo")).toBeNull();
    expect(screen.getByRole("button", { name: "Reference journeys" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: /Imports/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /Explore/ })).toBeNull();
    await waitFor(() => expect(document.documentElement.dataset.clearsightDemo).toBe("on"));
  });

  it("uses live API data without demo-only presentation when requested", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(true));
    render(<App presentation="live-preview" />);

    await screen.findByText("Demo environment");
    await waitFor(() => expect(document.documentElement.dataset.clearsightDemo).toBe("off"));
    expect(screen.queryByText("Stakeholder demo")).toBeNull();
    expect(screen.queryByText("Non-production data")).toBeNull();
    expect(screen.queryByRole("button", { name: "Reference journeys" })).toBeNull();
    expect(screen.queryByRole("button", { name: /Explore/ })).toBeNull();
  });

  it("opens the exact Program encoded by a Today intervention", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    vi.mocked(loadToday).mockResolvedValue({ items: [{ id: "today-program", type: "PROGRAM", title: "Review privacy programme", why_now: "Evidence changed.", scope: "Privacy", state: "Evidence incomplete", evidence: "One gap", owner: "DPO", due_at: "2026-08-09T12:00:00Z", primary_action: "Review reasons", action_target_type: "PROGRAM", action_target_id: "program-123" }], generated_at: "2026-08-07T15:00:00Z" });
    render(<App />);

    fireEvent.click(await screen.findByRole("button", { name: "Open program" }));
    await screen.findByRole("region", { name: "Program record" });
    expect(screen.queryByRole("heading", { name: "Programs" })).toBeNull();
    expect(window.location.hash).toBe("#programs/program-123/overview");
  });

  it("does not expose the Today response shortcut to a non-recipient in demo mode", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(true));
    const denied = evidenceRequest({ recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "another-person", state: "ASSIGNED" } });
    const exact = deferred<EvidenceRequest>();
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention(denied.id)], generated_at: "2026-08-07T15:00:00Z" });
    vi.mocked(loadEvidenceRequest).mockReturnValue(exact.promise);
    render(<App presentation="demo"/>);

    await screen.findByText("Demo environment");
    expect(screen.queryByText("Stakeholder demo")).toBeNull();
    await waitFor(() => expect(loadEvidenceRequest).toHaveBeenCalledWith(denied.id, "eligibility_preload"));
    await act(async () => { exact.resolve(denied); });
    expect(screen.queryAllByRole("button", { name: "Respond to evidence request" })).toHaveLength(0);
  });

  it("opens only an eligible request assigned to the verified actor from Today", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    const assigned = evidenceRequest();
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention(assigned.id)], generated_at: "2026-08-07T15:00:00Z" });
    vi.mocked(loadEvidenceRequest).mockResolvedValue(assigned);
    render(<App />);

    const shortcuts = await screen.findAllByRole("button", { name: "Respond to evidence request" });
    fireEvent.click(shortcuts[0]!);

    expect(await screen.findByRole("heading", { name: "Confirm assigned evidence" })).toBeTruthy();
    expect(loadCaptureRequest).not.toHaveBeenCalled();
  });

  it("opens an authorized multi-section assessment as a full-page workspace", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    const assigned = evidenceRequest({
      sections: [
        { id: "identity", title: "Identity" },
        { id: "controls", title: "Controls" },
        { id: "evidence", title: "Evidence" },
      ],
      fields: [
        { id: "name", section_id: "identity", label: "Registered name", type: "short_text", required: true },
        { id: "control", section_id: "controls", label: "Control description", type: "long_text", required: false },
        { id: "proof", section_id: "evidence", label: "Supporting document", type: "file", required: true },
      ],
      presentation: { default_mode: "WIZARD", allow_mode_switch: false },
    });
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention(assigned.id)], generated_at: "2026-08-07T15:00:00Z" });
    vi.mocked(loadEvidenceRequest).mockResolvedValue(assigned);
    render(<App />);

    fireEvent.click((await screen.findAllByRole("button", { name: "Respond to evidence request" }))[0]!);
    const workspace = await screen.findByRole("region", { name: "Internal assessment workspace" });
    expect(within(workspace).getByRole("heading", { name: assigned.title, level: 1 })).toBeTruthy();
    const facts = within(workspace).getByRole("group", { name: "Assessment details" });
    expect(within(facts).getByText("Ready for response")).toBeTruthy();
    expect(within(facts).getByText("Due 30 Aug 2099")).toBeTruthy();
    expect(within(facts).getByText("About 3 min")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Save draft" })).toBeNull();
    expect(screen.queryByRole("dialog", { name: "Evidence request" })).toBeNull();
    const completion = within(workspace).getByRole("region", { name: "Assessment completion" });
    expect(within(completion).getByText("0 of 3 answered")).toBeTruthy();
    expect(within(completion).getByText("0%")).toBeTruthy();
    const views = within(workspace).getByRole("navigation", { name: "Assessment views" });
    expect(within(views).getByText("Questions").getAttribute("aria-current")).toBe("page");
    expect(within(views).getByRole("button", { name: "Excel import" }).hasAttribute("disabled")).toBe(true);
    expect(Boolean(views.compareDocumentPosition(completion) & Node.DOCUMENT_POSITION_FOLLOWING)).toBe(true);
    fireEvent.click(within(views).getByRole("button", { name: "Review" }));
    expect(within(workspace).getByText("Check the highlighted answers")).toBeTruthy();
    const context = within(workspace).getByText("Request context").closest("details");
    expect(context).toBeTruthy();
    expect(Boolean(completion.compareDocumentPosition(context!) & Node.DOCUMENT_POSITION_FOLLOWING)).toBe(true);
    expect(within(workspace).getAllByRole("progressbar")).toHaveLength(1);
    const sections = within(workspace).getByRole("navigation", { name: "Assessment sections" });
    expect(within(sections).getByRole("button", { name: "Identity, 0 of 1 answered" })).toBeTruthy();
    expect(within(sections).getByRole("button", { name: "Controls, 0 of 1 answered" })).toBeTruthy();
    expect(within(sections).getByRole("button", { name: "Evidence, 0 of 1 answered" })).toBeTruthy();
    fireEvent.change(within(workspace).getByRole("textbox", { name: /Registered name/ }), { target: { value: "Example company" } });
    expect(within(sections).getByRole("button", { name: "Identity, 1 of 1 answered" })).toBeTruthy();
    expect(within(sections).getByRole("button", { name: "Controls, 0 of 1 answered" })).toBeTruthy();
    expect(within(workspace).getByRole("complementary", { name: "Response readiness and evidence" })).toBeTruthy();
    expect(within(workspace).getByRole("button", { name: "Continue" })).toBeTruthy();
    expect(loadEvidenceRequest).toHaveBeenCalledWith(assigned.id, "capture_revalidation");
    fireEvent.click(within(workspace).getByRole("button", { name: "Back to work" }));
    expect(screen.queryByRole("region", { name: "Internal assessment workspace" })).toBeNull();
  });

  it("keeps short automatic and terminal multi-section responses out of the wide workspace", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    const assigned = evidenceRequest({
      sections: [{ id: "first", title: "First" }, { id: "second", title: "Second" }],
      fields: [
        { id: "field-1", section_id: "first", label: "First answer", type: "short_text", required: false },
        { id: "field-2", section_id: "second", label: "Second answer", type: "short_text", required: false },
      ],
      presentation: { default_mode: "AUTOMATIC", allow_mode_switch: false },
    });
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention(assigned.id)], generated_at: "2026-08-07T15:00:00Z" });
    vi.mocked(loadEvidenceRequest).mockResolvedValue(assigned);
    const view = render(<App />);

    fireEvent.click((await screen.findAllByRole("button", { name: "Respond to evidence request" }))[0]!);
    let dialog = await screen.findByRole("dialog", { name: "Evidence request" });
    await within(dialog).findByRole("navigation", { name: "Request sections" });
    expect(dialog.closest(".cs-sheet")?.classList.contains("cs-sheet--default")).toBe(true);

    view.unmount();
    vi.mocked(loadEvidenceRequest).mockResolvedValue({
      ...assigned,
      presentation: { default_mode: "WIZARD", allow_mode_switch: false },
      status: "READY",
      deadline: "2000-01-01T00:00:00Z",
    });
    render(<App />);
    fireEvent.click((await screen.findAllByRole("button", { name: "Respond to evidence request" }))[0]!);
    dialog = await screen.findByRole("dialog", { name: "Evidence request" });
    await within(dialog).findByText("This request has expired");
    expect(dialog.closest(".cs-sheet")?.classList.contains("cs-sheet--default")).toBe(true);
  });

  it("keeps a classic multi-section response in the focused default sheet", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    const assigned = evidenceRequest({
      sections: [{ id: "first", title: "First" }, { id: "second", title: "Second" }],
      fields: [
        { id: "first-name", section_id: "first", label: "First response", type: "short_text", required: false },
        { id: "second-name", section_id: "second", label: "Second response", type: "short_text", required: false },
      ],
      presentation: { default_mode: "CLASSIC", allow_mode_switch: false },
    });
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention(assigned.id)], generated_at: "2026-08-07T15:00:00Z" });
    vi.mocked(loadEvidenceRequest).mockResolvedValue(assigned);
    render(<App />);

    fireEvent.click((await screen.findAllByRole("button", { name: "Respond to evidence request" }))[0]!);
    const dialog = await screen.findByRole("dialog", { name: "Evidence request" });
    await within(dialog).findByRole("navigation", { name: "Request sections" });
    expect(dialog.closest(".cs-sheet")?.classList.contains("cs-sheet--default")).toBe(true);
    expect(dialog.closest(".cs-sheet")?.classList.contains("assessment-capture-sheet")).toBe(false);
    expect(within(dialog).getByRole("navigation", { name: "Request sections" })).toBeTruthy();
  });

  it("revalidates the current request before opening capture", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    const assigned = evidenceRequest();
    const returned = evidenceRequest({
      recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "role-cro", state: "REASSIGNMENT_REQUIRED", revision: 2 },
      version: 2,
    });
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention(assigned.id)], generated_at: "2026-08-07T15:00:00Z" });
    vi.mocked(loadEvidenceRequest).mockResolvedValueOnce(assigned).mockResolvedValueOnce(returned);
    render(<App />);

    const shortcuts = await screen.findAllByRole("button", { name: "Respond to evidence request" });
    fireEvent.click(shortcuts[0]!);

    expect(await screen.findByRole("heading", { name: "You cannot open this request" })).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "Confirm assigned evidence" })).toBeNull();
  });

  it("shows a terminal request discovered while revalidating an authorized Today shortcut", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    const assigned = evidenceRequest();
    const expired = evidenceRequest({ status: "EXPIRED", version: 2 });
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention(assigned.id)], generated_at: "2026-08-07T15:00:00Z" });
    vi.mocked(loadEvidenceRequest).mockResolvedValueOnce(assigned).mockResolvedValueOnce(expired);
    render(<App />);

    const shortcuts = await screen.findAllByRole("button", { name: "Respond to evidence request" });
    fireEvent.click(shortcuts[0]!);

    expect(await screen.findByRole("heading", { name: "This request has expired" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Review and submit" })).toBeNull();
  });

  it("keeps the static not-found shortcut through StrictMode preloads and revalidates on open", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(true));
    const assigned = evidenceRequest();
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention(assigned.id)], generated_at: "2026-08-07T15:00:00Z" });
    vi.mocked(loadEvidenceRequest).mockImplementation((_id, intent) => intent === "eligibility_preload"
      ? Promise.resolve(assigned)
      : Promise.reject(new ApiError(404, "The request is no longer available.", "request_not_found")));
    const view = render(<StrictMode><App key="initial" /></StrictMode>);

    await screen.findAllByRole("button", { name: "Respond to evidence request" });
    view.rerender(<StrictMode><App key="strict-remount" /></StrictMode>);
    const shortcuts = await screen.findAllByRole("button", { name: "Respond to evidence request" });
    await waitFor(() => expect(vi.mocked(loadEvidenceRequest).mock.calls.filter((call) => call[1] === "eligibility_preload").length).toBeGreaterThanOrEqual(2));
    expect(vi.mocked(loadEvidenceRequest).mock.calls.every((call) => call[1] === "eligibility_preload")).toBe(true);
    fireEvent.click(shortcuts[0]!);

    await waitFor(() => expect(loadEvidenceRequest).toHaveBeenCalledWith(assigned.id, "capture_revalidation"));
    expect(await screen.findByRole("heading", { name: "This request is no longer available" })).toBeTruthy();
  });

  it("keeps the static terminal shortcut through StrictMode preloads and shows expiry on open", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(true));
    const assigned = evidenceRequest();
    const expired = evidenceRequest({ status: "EXPIRED", version: 2 });
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention(assigned.id)], generated_at: "2026-08-07T15:00:00Z" });
    vi.mocked(loadEvidenceRequest).mockImplementation((_id, intent) => Promise.resolve(intent === "eligibility_preload" ? assigned : expired));
    const view = render(<StrictMode><App key="initial" /></StrictMode>);

    await screen.findAllByRole("button", { name: "Respond to evidence request" });
    view.rerender(<StrictMode><App key="strict-remount" /></StrictMode>);
    const shortcuts = await screen.findAllByRole("button", { name: "Respond to evidence request" });
    await waitFor(() => expect(vi.mocked(loadEvidenceRequest).mock.calls.filter((call) => call[1] === "eligibility_preload").length).toBeGreaterThanOrEqual(2));
    expect(vi.mocked(loadEvidenceRequest).mock.calls.every((call) => call[1] === "eligibility_preload")).toBe(true);
    fireEvent.click(shortcuts[0]!);

    await waitFor(() => expect(loadEvidenceRequest).toHaveBeenCalledWith(assigned.id, "capture_revalidation"));
    expect(await screen.findByRole("heading", { name: "This request has expired" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Review and submit" })).toBeNull();
  });

  it("removes Today capture entry after the assigned actor returns the request", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    const assigned = evidenceRequest();
    const returned = evidenceRequest({
      recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "role-cro", state: "REASSIGNMENT_REQUIRED", revision: 2 },
      version: 2,
    });
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention(assigned.id)], generated_at: "2026-08-07T15:00:00Z" });
    vi.mocked(loadEvidenceRequest).mockResolvedValue(assigned);
    vi.mocked(loadEvidenceRequests).mockResolvedValue([assigned]);
    vi.mocked(declareWrongCaptureRecipient).mockResolvedValue(returned);
    render(<App />);

    await screen.findAllByRole("button", { name: "Respond to evidence request" });
    fireEvent.click(primaryLink("Work & Approvals"));
    fireEvent.click(await screen.findByRole("button", { name: "Related work" }));
    fireEvent.click(await screen.findByRole("button", { name: "Evidence requests" }));
    fireEvent.change(await screen.findByLabelText("Why should it be reassigned?"), { target: { value: "The account owner must respond." } });
    fireEvent.click(screen.getByRole("button", { name: "Return to requester" }));
    await waitFor(() => expect(declareWrongCaptureRecipient).toHaveBeenCalled());
    fireEvent.click(primaryLink("Home"));

    await waitFor(() => expect(screen.queryAllByRole("button", { name: "Respond to evidence request" })).toHaveLength(0));
  });

  it("keeps a newer workspace version when an older Today preload finishes last", async () => {
    window.history.replaceState(null, "", "#work/evidence/request-assigned");
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention("request-assigned")], generated_at: "2026-08-07T15:00:00Z" });
    const list = deferred<EvidenceRequest[]>();
    const exact = deferred<EvidenceRequest>();
    const older = evidenceRequest({ version: 1 });
    const newer = evidenceRequest({ version: 2, recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "another-person", state: "ASSIGNED" } });
    vi.mocked(loadEvidenceRequests).mockReturnValue(list.promise);
    vi.mocked(loadEvidenceRequest).mockReturnValue(exact.promise);
    render(<App />);

    await waitFor(() => {
      expect(loadEvidenceRequests).toHaveBeenCalled();
      expect(loadEvidenceRequest).toHaveBeenCalledWith("request-assigned", "eligibility_preload");
    });
    await act(async () => { list.resolve([newer]); });
    await act(async () => { exact.resolve(older); });

    await screen.findByText("Confirm assigned evidence");
    expect(screen.queryByRole("button", { name: "Open request" })).toBeNull();
  });

  it("keeps a newer Today version when an older workspace list finishes last", async () => {
    window.history.replaceState(null, "", "#work/evidence/request-assigned");
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention("request-assigned")], generated_at: "2026-08-07T15:00:00Z" });
    const list = deferred<EvidenceRequest[]>();
    const exact = deferred<EvidenceRequest>();
    const older = evidenceRequest({ version: 1 });
    const newer = evidenceRequest({ version: 2, recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "another-person", state: "ASSIGNED" } });
    vi.mocked(loadEvidenceRequests).mockReturnValue(list.promise);
    vi.mocked(loadEvidenceRequest).mockReturnValue(exact.promise);
    render(<App />);

    await waitFor(() => {
      expect(loadEvidenceRequests).toHaveBeenCalled();
      expect(loadEvidenceRequest).toHaveBeenCalledWith("request-assigned", "eligibility_preload");
    });
    await act(async () => { exact.resolve(newer); });
    await act(async () => { list.resolve([older]); });

    await screen.findByText("Confirm assigned evidence");
    expect(screen.queryByRole("button", { name: "Open request" })).toBeNull();
  });

  it("keeps an exact Today entity when the workspace list fails after the exact preload", async () => {
    window.history.replaceState(null, "", "#work/evidence");
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention("request-assigned")], generated_at: "2026-08-07T15:00:00Z" });
    const list = deferred<EvidenceRequest[]>();
    const exact = deferred<EvidenceRequest>();
    vi.mocked(loadEvidenceRequests).mockReturnValue(list.promise);
    vi.mocked(loadEvidenceRequest).mockReturnValue(exact.promise);
    render(<App />);

    await waitFor(() => {
      expect(loadEvidenceRequests).toHaveBeenCalled();
      expect(loadEvidenceRequest).toHaveBeenCalledWith("request-assigned", "eligibility_preload");
    });
    await act(async () => { exact.resolve(evidenceRequest()); });
    await act(async () => { list.reject(new Error("Workspace list unavailable")); });
    fireEvent.click(primaryLink("Home"));

    expect(await screen.findAllByRole("button", { name: "Respond to evidence request" })).toHaveLength(2);
  });

  it("keeps an exact Today entity when the workspace list fails before the exact preload", async () => {
    window.history.replaceState(null, "", "#work/evidence");
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention("request-assigned")], generated_at: "2026-08-07T15:00:00Z" });
    const list = deferred<EvidenceRequest[]>();
    const exact = deferred<EvidenceRequest>();
    vi.mocked(loadEvidenceRequests).mockReturnValue(list.promise);
    vi.mocked(loadEvidenceRequest).mockReturnValue(exact.promise);
    render(<App />);

    await waitFor(() => {
      expect(loadEvidenceRequests).toHaveBeenCalled();
      expect(loadEvidenceRequest).toHaveBeenCalledWith("request-assigned", "eligibility_preload");
    });
    await act(async () => { list.reject(new Error("Workspace list unavailable")); });
    await act(async () => { exact.resolve(evidenceRequest()); });
    fireEvent.click(primaryLink("Home"));

    expect(await screen.findAllByRole("button", { name: "Respond to evidence request" })).toHaveLength(2);
  });

  it("does not render an exact Today entity omitted from a successful workspace list", async () => {
    window.history.replaceState(null, "", "#work/evidence");
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention("request-assigned")], generated_at: "2026-08-07T15:00:00Z" });
    const list = deferred<EvidenceRequest[]>();
    const exact = deferred<EvidenceRequest>();
    vi.mocked(loadEvidenceRequests).mockReturnValue(list.promise);
    vi.mocked(loadEvidenceRequest).mockReturnValue(exact.promise);
    render(<App />);

    await waitFor(() => {
      expect(loadEvidenceRequests).toHaveBeenCalled();
      expect(loadEvidenceRequest).toHaveBeenCalledWith("request-assigned", "eligibility_preload");
    });
    await act(async () => { exact.resolve(evidenceRequest()); });
    await act(async () => { list.resolve([]); });

    expect(await screen.findByRole("heading", { name: "No evidence requests in this scope" })).toBeTruthy();
    expect(screen.queryByText("Confirm assigned evidence")).toBeNull();
    fireEvent.click(primaryLink("Home"));
    expect(await screen.findAllByRole("button", { name: "Respond to evidence request" })).toHaveLength(2);
  });

  it("ignores a targeted request that completes after verified scope changes", async () => {
    window.history.replaceState(null, "", "#work/evidence/stale-target");
    vi.mocked(loadContext).mockResolvedValueOnce(runtime(false)).mockResolvedValueOnce(secondScope());
    vi.mocked(loadToday).mockResolvedValue({ items: [], generated_at: "2026-08-07T15:00:00Z" });
    vi.mocked(loadEvidenceRequests).mockResolvedValue([]);
    const targeted = deferred<EvidenceRequest>();
    const currentTarget = deferred<EvidenceRequest>();
    vi.mocked(loadEvidenceRequest).mockReturnValueOnce(targeted.promise).mockReturnValueOnce(currentTarget.promise);
    const view = render(<App presentation="demo"/>);

    await waitFor(() => expect(loadEvidenceRequest).toHaveBeenCalledWith("stale-target"));
    view.rerender(<App presentation="live-preview"/>);
    await screen.findAllByText("Second Bank");
    await waitFor(() => expect(loadEvidenceRequest).toHaveBeenCalledTimes(2));
    await act(async () => { targeted.resolve(evidenceRequest({ id: "stale-target", title: "Stale first-bank request" })); });

    expect(screen.queryByText("Stale first-bank request")).toBeNull();
    await act(async () => { currentTarget.resolve(evidenceRequest({ id: "stale-target", tenant_id: "bank-second", title: "Current second-bank request" })); });
    expect(await screen.findByText("Current second-bank request")).toBeTruthy();
  });

  it("ignores declare-wrong completion from the prior verified scope", async () => {
    window.history.replaceState(null, "", "#work/evidence");
    const assigned = evidenceRequest();
    const currentSecond = evidenceRequest({ tenant_id: "bank-second", version: 2 });
    const staleReturned = evidenceRequest({
      version: 3,
      recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "role-cro", state: "REASSIGNMENT_REQUIRED", revision: 3 },
    });
    vi.mocked(loadContext).mockResolvedValueOnce(runtime(false)).mockResolvedValueOnce(secondScope());
    vi.mocked(loadToday)
      .mockResolvedValueOnce({ items: [], generated_at: "2026-08-07T15:00:00Z" })
      .mockResolvedValueOnce({ items: [evidenceAttention(currentSecond.id)], generated_at: "2026-08-07T15:01:00Z" });
    vi.mocked(loadEvidenceRequests).mockResolvedValueOnce([assigned]).mockResolvedValueOnce([currentSecond]);
    vi.mocked(loadEvidenceRequest).mockResolvedValue(currentSecond);
    const command = deferred<EvidenceRequest>();
    vi.mocked(declareWrongCaptureRecipient).mockReturnValue(command.promise);
    const view = render(<App presentation="demo"/>);

    fireEvent.change(await screen.findByLabelText("Why should it be reassigned?"), { target: { value: "The account owner must respond." } });
    fireEvent.click(screen.getByRole("button", { name: "Return to requester" }));
    await waitFor(() => expect(declareWrongCaptureRecipient).toHaveBeenCalled());
    view.rerender(<App presentation="live-preview"/>);
    await screen.findAllByText("Second Bank");
    await waitFor(() => expect(loadEvidenceRequest).toHaveBeenCalledWith(currentSecond.id, "eligibility_preload"));
    await act(async () => { command.resolve(staleReturned); });
    fireEvent.click(primaryLink("Home"));

    expect(await screen.findAllByRole("button", { name: "Respond to evidence request" })).toHaveLength(2);
  });

  it("ignores reassignment completion from the prior verified scope", async () => {
    window.history.replaceState(null, "", "#work/evidence");
    const requesterView = evidenceRequest({
      created_by: "role-cro",
      recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "another-person", state: "ASSIGNED" },
    });
    const currentSecond = evidenceRequest({
      tenant_id: "bank-second",
      version: 2,
      created_by: "requester-2",
      recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "another-person", state: "ASSIGNED" },
    });
    const staleReassigned = evidenceRequest({
      version: 3,
      created_by: "role-cro",
      recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "role-cro", state: "ASSIGNED", revision: 3 },
    });
    vi.mocked(loadContext).mockResolvedValueOnce(runtime(false)).mockResolvedValueOnce(secondScope());
    vi.mocked(loadToday)
      .mockResolvedValueOnce({ items: [], generated_at: "2026-08-07T15:00:00Z" })
      .mockResolvedValueOnce({ items: [evidenceAttention(currentSecond.id)], generated_at: "2026-08-07T15:01:00Z" });
    vi.mocked(loadEvidenceRequests).mockResolvedValueOnce([requesterView]).mockResolvedValueOnce([currentSecond]);
    vi.mocked(loadEvidenceRequest).mockResolvedValue(currentSecond);
    const command = deferred<EvidenceRequest>();
    vi.mocked(reassignCaptureRecipient).mockReturnValue(command.promise);
    listEvidenceRecipientCandidates.mockResolvedValue({ items: [{ principal_id: "role-cro", display_name: "Chief Risk Officer" }], has_more: false });
    const view = render(<App presentation="demo"/>);

    fireEvent.click(await screen.findByText("View details"));
    await screen.findByRole("option", { name: "Chief Risk Officer" });
    fireEvent.change(screen.getByLabelText("New assigned person"), { target: { value: "role-cro" } });
    fireEvent.change(screen.getByLabelText("Reason for change"), { target: { value: "The current owner must respond." } });
    fireEvent.click(screen.getByRole("button", { name: "Save recipient" }));
    await waitFor(() => expect(reassignCaptureRecipient).toHaveBeenCalled());
    view.rerender(<App presentation="live-preview"/>);
    await screen.findAllByText("Second Bank");
    await waitFor(() => expect(loadEvidenceRequest).toHaveBeenCalledWith(currentSecond.id, "eligibility_preload"));
    await act(async () => { command.resolve(staleReassigned); });
    fireEvent.click(primaryLink("Home"));

    expect(screen.queryAllByRole("button", { name: "Respond to evidence request" })).toHaveLength(0);
  });
});
