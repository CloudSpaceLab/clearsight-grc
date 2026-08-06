import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import type { RuntimeContext } from "./api";
import { loadContext, loadReadiness, loadToday } from "./api";

vi.mock("./components/RoleAwareOnboarding", () => ({ RoleAwareOnboarding: () => null }));
vi.mock("./api", () => ({
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
    tenant: { id: "bank-demo", name: "Demo Bank" },
    legal_entity: { id: "bank-ng", name: "Demo Bank Nigeria" },
    actor: { id: "role-cro", name: "Chief Risk Officer", role_codes: ["CRO", "EXECUTIVE"] },
    mode: "memory",
    demo_mode: demoMode,
    capabilities: { document_import: true, reference_journeys: demoMode },
  };
}

beforeEach(() => {
  window.history.replaceState(null, "", "#today");
  vi.mocked(loadToday).mockResolvedValue([]);
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

  it("exposes the stakeholder reference experience when demo mode is on", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(true));
    render(<App />);

    expect((await screen.findAllByRole("button", { name: /Explore/ })).length).toBeGreaterThan(0);
    expect(screen.getAllByRole("button", { name: /Imports/ }).length).toBeGreaterThan(0);
    expect(document.documentElement.dataset.clearsightDemo).toBe("on");
  });

  it("opens the exact Program encoded by a Today intervention", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    vi.mocked(loadToday).mockResolvedValue([{ id: "today-program", type: "PROGRAM", title: "Review privacy programme", why_now: "Evidence changed.", scope: "Privacy", state: "Evidence incomplete", evidence: "One gap", owner: "DPO", due_at: "2026-08-09T12:00:00Z", primary_action: "Review reasons", action_target_type: "PROGRAM", action_target_id: "program-123" }]);
    render(<App />);

    fireEvent.click(await screen.findByRole("button", { name: "Review and act" }));
    await screen.findByRole("heading", { name: "Programs" });
    expect(window.location.hash).toBe("#programs/program-123");
  });
});
