import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { FormPoliciesView } from "./FormPoliciesView";

const api = vi.hoisted(() => ({
  listFormResponsePolicies: vi.fn(), createFormResponsePolicy: vi.fn(), simulateFormResponsePolicy: vi.fn(),
  submitFormResponsePolicy: vi.fn(), approveFormResponsePolicy: vi.fn(), activateFormResponsePolicy: vi.fn(),
  suspendFormResponsePolicy: vi.fn(), rollbackFormResponsePolicy: vi.fn(),
}));
const monitoring = vi.hoisted(() => ({ loadFormTemplates: vi.fn() }));
vi.mock("../../formPoliciesApi", () => api);
vi.mock("../../monitoringApi", () => monitoring);

const policy = {
  id: "policy-1", code: "POOR-VENDOR-CERT", name: "Review poor vendor certification scores", purpose: "Create a review issue when certification evidence scores below the approved level.",
  action_class: "FORM_RESPONSE_CREATE_MATTER", automation_policy_id: "automation-1", automation_policy_version: 2,
  eligibility: { form_template_id: "form-1", form_template_version: 4, subject_types: ["VENDOR"], current_only: true, minimum_coverage: 0.8, bands: ["HIGH", "CRITICAL"] },
  action: { type: "VENDOR_DEFICIENCY", priority: 2, title_template: "Review vendor certification score", summary_template: "A completed vendor certification response requires review.", requested_handling: "Review the response and confirm remediation." },
  blast_radius: { per_run: 10, per_day: 50 }, outcome_contract: { expected_outcome: "The score is no longer adverse or an accepted treatment is recorded.", check_after_minutes: 1440, failure_response: "ESCALATE" },
  rollout: "SHADOW", status: "DRAFT", maker_id: "maker-1", checksum: "sum", version: 1, record_version: 3, created_at: "2026-09-01T09:00:00Z", updated_at: "2026-09-01T09:00:00Z",
};

const scoredForm = {
  id: "form-1", tenant_id: "tenant-1", legal_entity_id: "entity-1", code: "VENDOR-CERTIFICATION", name: "Vendor certification", purpose: "Collect certification evidence.",
  status: "ACTIVE", is_current: true, version: 4, scoring_mode: "COMPLIANCE", score_profile: { version: "1", mode: "COMPLIANCE", direction: "LOW_IS_POOR", contributions: [], bands: [] }, fields: [],
  created_at: "2026-09-01T09:00:00Z", updated_at: "2026-09-01T09:00:00Z",
};

beforeEach(() => {
  for (const value of Object.values(api)) value.mockReset();
  monitoring.loadFormTemplates.mockReset();
  api.listFormResponsePolicies.mockResolvedValue([policy]);
  monitoring.loadFormTemplates.mockResolvedValue([scoredForm]);
  api.simulateFormResponsePolicy.mockResolvedValue({ id: "simulation-1", policy_id: "policy-1", policy_version: 1, population_count: 42, eligible_count: 5, would_create_count: 3, would_reuse_count: 1, blast_suppressed_count: 1, restricted_excluded_count: 2, observed_at: "2026-09-01T10:00:00Z", expires_at: "2026-09-01T11:00:00Z" });
});

describe("FormPoliciesView", () => {
  it("loads stored policies and scored form choices independently", async () => {
    render(<FormPoliciesView/>);
    expect((await screen.findAllByText("Review poor vendor certification scores")).length).toBeGreaterThan(0);
    expect(screen.getAllByText("Draft").length).toBeGreaterThan(0);
    expect(screen.getAllByRole("button", { name: "Simulate impact" })).toHaveLength(1);
    await waitFor(() => expect(monitoring.loadFormTemplates).toHaveBeenCalledTimes(1));

    fireEvent.click(screen.getByRole("button", { name: "Simulate impact" }));
    await waitFor(() => expect(api.simulateFormResponsePolicy).toHaveBeenCalledWith("policy-1", 3));
    expect(await screen.findByText("3 new issues")).toBeTruthy();
    expect(screen.getByText("2 restricted responses excluded")).toBeTruthy();
  });

  it("keeps existing policies available when approved form choices cannot load", async () => {
    monitoring.loadFormTemplates.mockRejectedValue(new Error("Approved scoring forms cannot be checked right now."));
    render(<FormPoliciesView/>);
    expect((await screen.findAllByText("Review poor vendor certification scores")).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole("button", { name: "Create policy" }));
    expect(await screen.findByText("Approved scoring forms cannot be checked right now.")).toBeTruthy();
    expect(screen.getAllByText("Review poor vendor certification scores").length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole("button", { name: "Retry form choices" }));
    expect(monitoring.loadFormTemplates).toHaveBeenCalledTimes(2);
  });

  it("does not offer unscored active forms for policy creation", async () => {
    monitoring.loadFormTemplates.mockResolvedValue([{ ...scoredForm, id: "form-none", scoring_mode: "NONE", score_profile: undefined }]);
    render(<FormPoliciesView/>);
    await screen.findAllByText("Review poor vendor certification scores");
    fireEvent.click(screen.getByRole("button", { name: "Create policy" }));
    expect(await screen.findByText(/No active approved scoring forms are available/)).toBeTruthy();
    expect((screen.getByRole("button", { name: "Create policy draft" }) as HTMLButtonElement).disabled).toBe(true);
  });

  it("offers a real recovery action when the stored population cannot be loaded", async () => {
    api.listFormResponsePolicies.mockRejectedValue(new Error("Response policies cannot be checked right now."));
    render(<FormPoliciesView/>);
    expect(await screen.findByRole("alert")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Retry loading policies" }));
    expect(api.listFormResponsePolicies).toHaveBeenCalledTimes(2);
  });
});
