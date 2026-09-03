import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { StrictMode } from "react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import type { RuntimeContext } from "./api";
import { loadCaptureRequest, loadContext, loadEvidenceRequest, loadEvidenceRequests, loadReadiness, loadToday } from "./api";
import type { AttentionItem, EvidenceRequest } from "./types";
import { declareWrongCaptureRecipient, reassignCaptureRecipient } from "./captureApi";
import { ApiError } from "./http";

const { listEvidenceRecipientCandidates } = vi.hoisted(() => ({ listEvidenceRecipientCandidates: vi.fn() }));

vi.mock("./components/RoleAwareOnboarding", async () => {
  const React = await import("react");
  return { RoleAwareOnboarding: ({ onStep, surface }: { onStep: (step: { intent?: string; view?: string }) => void | Promise<void>; surface: string }) => {
    const [busy, setBusy] = React.useState(false);
    const [error, setError] = React.useState("");
    async function openGuideAction() {
      setBusy(true); setError("");
      try { await onStep({ intent: "open-vendor-due-diligence", view: "vendors" }); }
      catch { setError("This guide step could not be opened. Try again."); }
      finally { setBusy(false); }
    }
    async function openNextVendorTask() {
      setBusy(true); setError("");
      try { await onStep({ intent: "open-vendor-next-action", view: "vendors" }); }
      catch { setError("This guide step could not be opened. Try again."); }
      finally { setBusy(false); }
    }
    return <aside aria-label={`${surface === "VENDORS" ? "Vendor" : "Today"} guide`}><output data-testid="onboarding-surface">{surface}</output><button type="button" disabled={busy} onClick={() => void openGuideAction()}>Review due diligence</button><button type="button" disabled={busy} onClick={() => void openNextVendorTask()}>Open next vendor task</button>{error && <p role="alert">{error}</p>}</aside>;
  }};
});
vi.mock("./components/VendorsWorkspace", () => ({
  VendorsWorkspace: ({ onOpenRequest, guideIntent, targetID }: { onOpenRequest?: (requestID: string) => void; guideIntent?: { type: string }; targetID?: string }) => <><output data-testid="vendor-guide-intent">{guideIntent?.type}</output><output data-testid="vendor-target">{targetID}</output><button type="button" onClick={() => onOpenRequest?.("request-vendor-1")}>Review vendor request</button></>,
}));
vi.mock("./captureApi", () => ({
  declareWrongCaptureRecipient: vi.fn(),
  reassignCaptureRecipient: vi.fn(),
  uploadInternalCaptureArtifact: vi.fn(),
}));
vi.mock("./api", () => ({
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
  submitCaptureRequest: vi.fn(),
}));
vi.mock("./evidenceRequestAdminApi", async (importOriginal) => ({
  ...await importOriginal<typeof import("./evidenceRequestAdminApi")>(),
  listEvidenceRecipientCandidates,
}));

type RuntimeWithCapabilities = RuntimeContext & {
  demo_mode: boolean;
  capabilities: { document_import: boolean; reference_journeys: boolean; oversight_read?: boolean };
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
});

describe("runtime navigation", () => {
  it("does not replace an unavailable actor queue with static sample work in demo presentation", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(true));
    vi.mocked(loadToday).mockRejectedValue(new Error("Today projection unavailable"));

    render(<App presentation="demo"/>);

    expect(await screen.findByRole("heading", { name: "Today is unavailable" })).toBeTruthy();
    expect(screen.queryByText("Review proposed digital-channel requirements")).toBeNull();
  });

  it("keeps import capability in administration and removes reference tooling when demo mode is off", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    render(<App />);

    expect((await screen.findAllByRole("button", { name: "Forms" })).length).toBeGreaterThan(0);
    await waitFor(() => expect(document.documentElement.dataset.clearsightDemo).toBe("off"));
    expect(screen.getByLabelText("Administration")).toBeTruthy();
    expect(screen.queryByRole("button", { name: /Imports/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /Explore/ })).toBeNull();
    expect(screen.queryByText("Demo environment")).toBeNull();
  });

  it("shows organization oversight only when the verified runtime grants oversight read", async () => {
    vi.mocked(loadContext).mockResolvedValue({ ...runtime(false), capabilities: { ...runtime(false).capabilities, oversight_read: true } });
    render(<App />);

    expect((await screen.findAllByRole("button", { name: "Oversight" })).length).toBeGreaterThan(0);
  });

  it("does not turn platform administration into risk oversight access", async () => {
    vi.mocked(loadContext).mockResolvedValue({
      ...runtime(false),
      actor: { id: "system-admin", name: "System Administrator", role_codes: ["SYSTEM_ADMIN"] },
      capabilities: { ...runtime(false).capabilities, oversight_read: false },
    });
    render(<App />);

    await screen.findByText("Nothing needs your action right now");
    expect(screen.queryByRole("button", { name: "Oversight" })).toBeNull();
  });

  it("provides Vendors as a first-class navigation destination", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    render(<App />);

    const vendorButtons = await screen.findAllByRole("button", { name: "Vendors" });
    expect(vendorButtons.length).toBeGreaterThan(0);
    const vendorButton = vendorButtons[0];
    if (!vendorButton) throw new Error("Vendors navigation is missing");
    fireEvent.click(vendorButton);
    expect(vendorButton.getAttribute("aria-current")).toBe("page");
    expect(window.location.hash).toBe("#vendors");
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

    const vendorButton = (await screen.findAllByRole("button", { name: "Vendors" }))[0];
    if (!vendorButton) throw new Error("Vendors navigation is missing");
    fireEvent.click(vendorButton);

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
    expect(window.location.hash).toBe("#vendors/relationship-b");
  });

  it("cancels a slow vendor guide action when navigation leaves Vendors", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    render(<App />);

    fireEvent.click(await screen.findByRole("button", { name: "Review due diligence" }));
    await screen.findByTestId("vendor-guide-intent");
    const todayButton = (await screen.findAllByRole("button", { name: "Today" }))[0];
    if (!todayButton) throw new Error("Today navigation is missing");
    fireEvent.click(todayButton);

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

    const vendorButton = (await screen.findAllByRole("button", { name: "Vendors" }))[0];
    if (!vendorButton) throw new Error("Vendors navigation is missing");
    fireEvent.click(vendorButton);
    fireEvent.click(await screen.findByRole("button", { name: "Review vendor request" }));

    expect(window.location.hash).toBe("#work/evidence/request-vendor-1");
    expect(await screen.findByText("Vendor due diligence request")).toBeTruthy();
    expect(loadEvidenceRequest).toHaveBeenCalledWith("request-vendor-1");
  });

  it("keeps stakeholder reference journeys secondary when explicit demo presentation is on", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(true));
    render(<App presentation="demo"/>);

    await screen.findByText("Stakeholder demo");
    expect(screen.getByText("Demo environment")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Reference journeys" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: /Imports/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /Explore/ })).toBeNull();
    await waitFor(() => expect(document.documentElement.dataset.clearsightDemo).toBe("on"));
  });

  it("uses live API data without demo-only presentation when requested", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(true));
    render(<App presentation="live-preview" />);

    await screen.findByText("Non-production data");
    await waitFor(() => expect(document.documentElement.dataset.clearsightDemo).toBe("off"));
    expect(screen.queryByText("Stakeholder demo")).toBeNull();
    expect(screen.getByText("Demo environment")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Reference journeys" })).toBeNull();
    expect(screen.queryByRole("button", { name: /Explore/ })).toBeNull();
  });

  it("opens the exact Program encoded by a Today intervention", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    vi.mocked(loadToday).mockResolvedValue({ items: [{ id: "today-program", type: "PROGRAM", title: "Review privacy programme", why_now: "Evidence changed.", scope: "Privacy", state: "Evidence incomplete", evidence: "One gap", owner: "DPO", due_at: "2026-08-09T12:00:00Z", primary_action: "Review reasons", action_target_type: "PROGRAM", action_target_id: "program-123" }], generated_at: "2026-08-07T15:00:00Z" });
    render(<App />);

    fireEvent.click(await screen.findByRole("button", { name: "Open program" }));
    await screen.findByRole("heading", { name: "Programs" });
    expect(window.location.hash).toBe("#programs/program-123/overview");
  });

  it("does not expose the Today response shortcut to a non-recipient in demo mode", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(true));
    const denied = evidenceRequest({ recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "another-person", state: "ASSIGNED" } });
    const exact = deferred<EvidenceRequest>();
    vi.mocked(loadToday).mockResolvedValue({ items: [evidenceAttention(denied.id)], generated_at: "2026-08-07T15:00:00Z" });
    vi.mocked(loadEvidenceRequest).mockReturnValue(exact.promise);
    render(<App presentation="demo"/>);

    await screen.findByText("Stakeholder demo");
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
    fireEvent.click(screen.getAllByRole("button", { name: /Work/ })[0]!);
    fireEvent.click(await screen.findByRole("button", { name: "Evidence review" }));
    fireEvent.change(await screen.findByLabelText("Why should it be reassigned?"), { target: { value: "The account owner must respond." } });
    fireEvent.click(screen.getByRole("button", { name: "Return to requester" }));
    await waitFor(() => expect(declareWrongCaptureRecipient).toHaveBeenCalled());
    fireEvent.click(screen.getAllByRole("button", { name: /Today/ })[0]!);

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
    fireEvent.click(screen.getAllByRole("button", { name: /Today/ })[0]!);

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
    fireEvent.click(screen.getAllByRole("button", { name: /Today/ })[0]!);

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
    fireEvent.click(screen.getAllByRole("button", { name: /Today/ })[0]!);
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
    fireEvent.click(screen.getAllByRole("button", { name: /Today/ })[0]!);

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
    fireEvent.click(screen.getAllByRole("button", { name: /Today/ })[0]!);

    expect(screen.queryAllByRole("button", { name: "Respond to evidence request" })).toHaveLength(0);
  });
});
