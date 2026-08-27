import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import axe from "axe-core";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { loadContext, loadProgramSummaries } from "../api";
import { ApiError } from "../http";
import { createFormTemplate, loadFormTemplates, transitionFormTemplate } from "../monitoringApi";
import type { FormTemplate } from "../monitoringTypes";
import { vendorDueDiligenceStarterForm } from "../vendorDueDiligenceForm";
import { VendorFormReadiness } from "./VendorFormReadiness";

vi.mock("../api", () => ({ loadContext: vi.fn(), loadProgramSummaries: vi.fn() }));
vi.mock("../monitoringApi", () => ({ createFormTemplate: vi.fn(), loadFormTemplates: vi.fn(), transitionFormTemplate: vi.fn() }));

const program = {
  id: "program-1", tenant_id: "bank", legal_entity_id: "entity", code: "TPRM", name: "Third-party risk management", type: "OPERATIONAL", status: "ACTIVE", owning_function: "Risk", scope: {}, effective_from: "2026-01-01T00:00:00Z", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-08-01T00:00:00Z", version: 3,
};
const summary = { program, state_label: "Attention needed", overall_state: "EVIDENCE_INSUFFICIENT" as const, reasons: [], reasons_total: 0, reasons_omitted: 0, open_matter_count: 0, requirement_count: 2, safeguard_count: 2, evidence_check_count: 1, program_version: 3, assessed_program_version: 3, projection_version: 3, projection_stale: false };

function form(status: FormTemplate["status"], actor = "maker-1", version = 1): FormTemplate {
  return { id: "form-1", tenant_id: "bank", legal_entity_id: "entity", program_id: program.id, ...vendorDueDiligenceStarterForm, status, is_current: status === "ACTIVE", version, created_by: "maker-1", submitted_by: status === "PENDING_APPROVAL" || status === "ACTIVE" ? actor : undefined, created_at: "2026-08-27T10:00:00Z", updated_at: "2026-08-27T10:00:00Z" };
}

function setup(actorID = "maker-1") {
  vi.mocked(loadContext).mockResolvedValue({ tenant: { id: "bank", name: "Bank" }, legal_entity: { id: "entity", name: "Bank Nigeria" }, actor: { id: actorID, name: actorID }, mode: "live" });
  vi.mocked(loadProgramSummaries).mockResolvedValue({ items: [summary], generated_at: "2026-08-27T10:00:00Z" });
}

describe("VendorFormReadiness", () => {
  beforeEach(() => { vi.resetAllMocks(); });

  it("creates the canonical draft and sends it for independent approval", async () => {
    setup();
    vi.mocked(loadFormTemplates).mockResolvedValue([]);
    vi.mocked(createFormTemplate).mockResolvedValue(form("DRAFT"));
    vi.mocked(transitionFormTemplate).mockResolvedValue(form("PENDING_APPROVAL", "maker-1", 2));
    render(<div className="app-shell"><main>Workspace</main><VendorFormReadiness onClose={vi.fn()} onReady={vi.fn()}/></div>);

    fireEvent.change(await screen.findByLabelText("Program"), { target: { value: "program-1" } });
    fireEvent.click(await screen.findByRole("button", { name: "Create form and send for approval" }));

    await waitFor(() => expect(createFormTemplate).toHaveBeenCalledWith("program-1", vendorDueDiligenceStarterForm));
    expect(transitionFormTemplate).toHaveBeenCalledWith("program-1", "form-1", 1, "PENDING_APPROVAL");
    expect(await screen.findByText("Waiting for an independent reviewer")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Activate due-diligence form" })).toBeNull();
    const results = await axe.run(document.body, { rules: { "color-contrast": { enabled: false } } });
    expect(results.violations.map((violation) => violation.id)).toEqual([]);
  });

  it("allows a different signed-in reviewer to activate the pending form", async () => {
    setup("reviewer-1");
    const pending = form("PENDING_APPROVAL", "maker-1", 2);
    const active = { ...pending, status: "ACTIVE" as const, is_current: true, approved_by: "reviewer-1", version: 3 };
    vi.mocked(loadFormTemplates).mockResolvedValue([pending]);
    vi.mocked(transitionFormTemplate).mockResolvedValue(active);
    const onReady = vi.fn();
    render(<div className="app-shell"><main>Workspace</main><VendorFormReadiness onClose={vi.fn()} onReady={onReady}/></div>);

    fireEvent.change(await screen.findByLabelText("Program"), { target: { value: "program-1" } });
    fireEvent.click(await screen.findByRole("button", { name: "Activate due-diligence form" }));

    await waitFor(() => expect(transitionFormTemplate).toHaveBeenCalledWith("program-1", "form-1", 2, "ACTIVE"));
    expect(onReady).toHaveBeenCalledWith(active);
  });

  it("preserves the setup state and explains authority or version failures", async () => {
    setup("reviewer-1");
    const pending = form("PENDING_APPROVAL", "maker-1", 2);
    vi.mocked(loadFormTemplates).mockResolvedValue([pending]);
    vi.mocked(transitionFormTemplate).mockRejectedValueOnce(new ApiError(403, "forbidden")).mockRejectedValueOnce(new ApiError(409, "conflict"));
    render(<div className="app-shell"><main>Workspace</main><VendorFormReadiness onClose={vi.fn()} onReady={vi.fn()}/></div>);

    fireEvent.change(await screen.findByLabelText("Program"), { target: { value: "program-1" } });
    fireEvent.click(await screen.findByRole("button", { name: "Activate due-diligence form" }));
    expect(await screen.findByText(/current role cannot approve/i)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Activate due-diligence form" }));
    expect(await screen.findByText(/form changed/i)).toBeTruthy();
    expect(screen.getByRole("button", { name: "Reload form status" })).toBeTruthy();
  });
});
