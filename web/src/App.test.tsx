import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import type { RuntimeContext } from "./api";
import { loadContext, loadEvidenceRequest, loadReadiness, loadToday } from "./api";
import type { EvidenceRequest } from "./types";

vi.mock("./components/RoleAwareOnboarding", () => ({ RoleAwareOnboarding: () => null }));
vi.mock("./components/VendorsWorkspace", () => ({
  VendorsWorkspace: ({ onOpenRequest }: { onOpenRequest?: (requestID: string) => void }) => <button type="button" onClick={() => onOpenRequest?.("request-vendor-1")}>Review vendor request</button>,
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

type RuntimeWithCapabilities = RuntimeContext & {
  demo_mode: boolean;
  capabilities: { document_import: boolean; reference_journeys: boolean };
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

beforeEach(() => {
  window.history.replaceState(null, "", "#today");
  vi.mocked(loadToday).mockResolvedValue({ items: [], generated_at: "2026-08-07T15:00:00Z" });
  vi.mocked(loadReadiness).mockRejectedValue(new Error("No readiness baseline"));
});

describe("runtime navigation", () => {
  it("keeps real imports available and removes reference navigation when demo mode is off", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    render(<App />);

    expect((await screen.findAllByRole("button", { name: /Imports/ })).length).toBeGreaterThan(0);
    await waitFor(() => expect(document.documentElement.dataset.clearsightDemo).toBe("off"));
    expect(screen.queryByRole("button", { name: /Explore/ })).toBeNull();
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

  it("exposes the stakeholder reference experience when demo mode is on", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(true));
    render(<App />);

    expect((await screen.findAllByRole("button", { name: /Explore/ })).length).toBeGreaterThan(0);
    expect(screen.getAllByRole("button", { name: /Imports/ }).length).toBeGreaterThan(0);
    await waitFor(() => expect(document.documentElement.dataset.clearsightDemo).toBe("on"));
  });

  it("uses live API data without demo-only presentation when requested", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(true));
    render(<App presentation="live-preview" />);

    await screen.findByText("Live preview · Non-production");
    await waitFor(() => expect(document.documentElement.dataset.clearsightDemo).toBe("off"));
    expect(screen.queryByText("Stakeholder demo")).toBeNull();
    expect(screen.queryByRole("button", { name: /Explore/ })).toBeNull();
  });

  it("opens the exact Program encoded by a Today intervention", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    vi.mocked(loadToday).mockResolvedValue({ items: [{ id: "today-program", type: "PROGRAM", title: "Review privacy programme", why_now: "Evidence changed.", scope: "Privacy", state: "Evidence incomplete", evidence: "One gap", owner: "DPO", due_at: "2026-08-09T12:00:00Z", primary_action: "Review reasons", action_target_type: "PROGRAM", action_target_id: "program-123" }], generated_at: "2026-08-07T15:00:00Z" });
    render(<App />);

    fireEvent.click(await screen.findByRole("button", { name: "Open program" }));
    await screen.findByRole("heading", { name: "Programs" });
    expect(window.location.hash).toBe("#programs/program-123");
  });
});
