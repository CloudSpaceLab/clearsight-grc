import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import type { RuntimeContext } from "./api";
import { loadContext, loadOnboardingGuide, loadOnboardingState, loadReadiness, loadToday } from "./api";

vi.mock("./api", () => ({
  loadCaptureRequest: vi.fn(),
  loadContext: vi.fn(),
  loadEvidenceRequests: vi.fn(),
  loadEvidenceSources: vi.fn(),
  loadIntegrity: vi.fn(),
  loadOnboardingGuide: vi.fn(),
  loadOnboardingState: vi.fn(),
  loadPolicies: vi.fn(),
  loadProjectionHealth: vi.fn(),
  loadReadiness: vi.fn(),
  loadToday: vi.fn(),
  loadWorkflowTasks: vi.fn(),
  reconcileProgramState: vi.fn(),
  resolveAuthority: vi.fn(),
  saveOnboardingState: vi.fn(),
}));

type RuntimeWithCapabilities = RuntimeContext & {
  demo_mode: boolean;
  capabilities: { document_import: boolean; reference_journeys: boolean };
};

function runtime(demoMode: boolean): RuntimeWithCapabilities {
  return {
    tenant: { id: "bank-demo", name: "Demo Bank" },
    legal_entity: { id: "bank-ng", name: "Demo Bank Nigeria" },
    actor: { id: "role-cro", name: "Chief Risk Officer" },
    mode: "memory",
    demo_mode: demoMode,
    capabilities: { document_import: true, reference_journeys: demoMode },
  };
}

beforeEach(() => {
  vi.mocked(loadToday).mockResolvedValue([]);
  vi.mocked(loadReadiness).mockRejectedValue(new Error("No readiness baseline"));
  vi.mocked(loadOnboardingGuide).mockRejectedValue(new Error("No guide"));
  vi.mocked(loadOnboardingState).mockRejectedValue(new Error("No guide state"));
});

describe("runtime demo mode", () => {
  it("keeps real imports available and removes reference navigation when demo mode is off", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(false));
    render(<App />);

    expect(await screen.findByRole("button", { name: /Imports/ })).toBeTruthy();
    await waitFor(() => expect(document.documentElement.dataset.clearsightDemo).toBe("off"));
    expect(screen.queryByRole("button", { name: /Explore/ })).toBeNull();
  });

  it("exposes the stakeholder reference experience when demo mode is on", async () => {
    vi.mocked(loadContext).mockResolvedValue(runtime(true));
    render(<App />);

    expect(await screen.findByRole("button", { name: /Explore/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /Imports/ })).toBeTruthy();
    expect(document.documentElement.dataset.clearsightDemo).toBe("on");
  });
});
