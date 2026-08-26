import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import axe from "axe-core";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../http";
import { loadFormTemplates } from "../monitoringApi";
import type { FormTemplate } from "../monitoringTypes";
import { completeVendorAssessment, createVendorAssessmentDeficiency, loadCurrentVendorAssessment, loadVendorAssessment, reissueVendorAssessmentRequest, requestVendorAssessmentClarification, retryVendorAssessmentSetup, reviewVendorAssessmentDocument, sendVendorAssessmentRequest, startVendorAssessment, startVendorAssessmentReview, vendorAssessmentDocumentURL } from "../vendorAssessmentApi";
import type { VendorAssessment, VendorAssessmentReviewView } from "../vendorAssessmentTypes";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import { createVendorRelationship, loadVendorIdentity, loadVendorRelationship, loadVendorRelationships, removeApprovedVendorLogo, updateVendorIdentity, updateVendorRelationship, uploadApprovedVendorLogo } from "../vendorApi";
import { VendorsWorkspace } from "./VendorsWorkspace";

vi.mock("../vendorApi", async (importOriginal) => ({
  ...await importOriginal<typeof import("../vendorApi")>(),
  createVendorRelationship: vi.fn(),
  loadVendorRelationship: vi.fn(),
  loadVendorRelationships: vi.fn(),
  loadVendorIdentity: vi.fn(),
  removeApprovedVendorLogo: vi.fn(),
  updateVendorIdentity: vi.fn(),
  updateVendorRelationship: vi.fn(),
  uploadApprovedVendorLogo: vi.fn(),
}));

vi.mock("../monitoringApi", () => ({ loadFormTemplates: vi.fn() }));
vi.mock("../vendorAssessmentApi", () => ({
  completeVendorAssessment: vi.fn(),
  createVendorAssessmentDeficiency: vi.fn(),
  loadCurrentVendorAssessment: vi.fn(),
  loadVendorAssessment: vi.fn(),
  reissueVendorAssessmentRequest: vi.fn(),
  requestVendorAssessmentClarification: vi.fn(),
  retryVendorAssessmentSetup: vi.fn(),
  reviewVendorAssessmentDocument: vi.fn(),
  sendVendorAssessmentRequest: vi.fn(),
  startVendorAssessment: vi.fn(),
  startVendorAssessmentReview: vi.fn(),
  vendorAssessmentDocumentURL: vi.fn(),
}));
let showVendorWorkAction = false;
const vendorWorkAction = vi.fn();
vi.mock("./VendorWorkPanel", () => ({ VendorWorkPanel: ({ relationshipID }: { relationshipID: string }) => <section className="vendor-work-panel" tabIndex={-1} data-testid={`vendor-work-relationship-${relationshipID}`}>Vendor requests{showVendorWorkAction && <button type="button" className="primary-button" onClick={vendorWorkAction}>Request vendor work</button>}</section> }));

const record: VendorRelationshipAggregate = {
  vendor: { id: "vendor-1", tenant_id: "bank", legal_name: "Acme Processing Limited", trading_name: "Acme", registration_ref: "RC-10001", jurisdiction: "Nigeria", source_id: "procurement", external_ref: "vendor-10001", status: "ACTIVE", created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 1 },
  relationship: { id: "relationship-1", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "vendor-1", service_name: "Card transaction processing", business_owner_principal_id: "owner-1", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "PROPOSED", created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 1 },
};

const activeVendorForm: FormTemplate = {
  id: "form-1", tenant_id: "bank", code: "VENDOR-DUE-DILIGENCE", name: "Vendor security and privacy review",
  purpose: "Collect the information required for the vendor review.", presentation: { default_mode: "WIZARD", allow_mode_switch: true }, sections: [], fields: [],
  status: "ACTIVE", is_current: true, version: 3, created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z",
};

function assessment(status: VendorAssessment["status"]): VendorAssessment {
  return {
    id: "assessment-1", tenant_id: "bank", legal_entity_id: "entity", relationship_id: "relationship-1",
    review_kind: "ONBOARDING", source_trigger: "INITIAL", stable_episode_key: "episode-1", status, form_template_id: "form-1", form_template_version: 3,
    review_due_at: "2099-09-30T23:59:59Z", started_by_principal_id: "owner-1", started_at: "2026-08-26T10:00:00Z",
    review_matter_id: status === "SETUP_PENDING" ? undefined : "matter-1", current_request_id: status === "COLLECTING" ? "request-1" : undefined,
    version: 3, created_at: "2026-08-26T10:00:00Z", updated_at: "2026-08-28T12:00:00Z",
  };
}

function review(status: VendorAssessment["status"]): VendorAssessmentReviewView {
  return {
    assessment: {
      ...assessment(status),
      submission_id: "submission-1",
      submitted_at: "2026-08-28T11:00:00Z",
      review_started_at: status === "SUBMITTED" ? undefined : "2026-08-28T12:00:00Z",
      completed_at: status === "COMPLETED" ? "2026-08-29T12:00:00Z" : undefined,
      conclusion: status === "COMPLETED" ? "SATISFACTORY_WITH_CONDITIONS" : undefined,
      conclusion_rationale: status === "COMPLETED" ? "Proceed after the recorded access-control action is complete." : undefined,
      conclusion_uncertainty: status === "COMPLETED" ? "The next resilience exercise remains due." : undefined,
    },
    requests: [{ request_id: "request-1", purpose: "INITIAL", sequence: 1, origin_sequence: 1, status: "SUBMITTED", deadline: "2026-08-27T23:59:59Z", form_template_id: "form-1", form_template_version: 3 }],
    response: { submission_id: "submission-1", request_id: "request-1", submitted_at: "2026-08-28T11:00:00Z", answer_count: 2, artifact_count: 1 },
    answers: [
      { field_id: "security-testing", label: "Independent security testing", type: "YES_NO", required: true, visibility: "VISIBLE", value: { text: "Yes" }, provenance: { origin: "SOURCE_PREFILLED", binding_id: "binding-1", binding_version: 4, source_receipt: { source_id: "procurement", observed_at: "2026-08-27T09:30:00Z" } } },
      { field_id: "hidden-follow-up", label: "Follow-up detail", type: "LONG_TEXT", required: true, visibility: "CONDITIONALLY_OMITTED" },
    ],
    coverage: { visible_fields: 1, answered_fields: 1, required_fields: 1, answered_required: 1, ratio: 1 },
    documents: [{ field_id: "security-report", artifact_id: "artifact-1", file_name: "independent-security-test.pdf", media_type: "application/pdf", size_bytes: 64000, artifact_status: "AVAILABLE", evidence_class: "VENDOR_SUPPLIED", document_type: "SECURITY_TEST", expires_on: "2027-08-01" }],
    provisional_score: { score: 86, coverage: 1, rule_results: [] },
    matters: [{ matter_id: "matter-1", type: "VENDOR_DEFICIENCY", status: "OPEN", title: "Access-control evidence gap" }],
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => { resolve = resolvePromise; reject = rejectPromise; });
  return { promise, resolve, reject };
}

beforeEach(() => {
  vi.clearAllMocks();
  showVendorWorkAction = false;
  vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [record] });
  vi.mocked(loadVendorRelationship).mockResolvedValue(record);
  vi.mocked(loadVendorIdentity).mockResolvedValue({ vendor: record.vendor, brand: { state: "UNAVAILABLE", version: 0, event_version: 0 } });
  vi.mocked(loadFormTemplates).mockResolvedValue([activeVendorForm]);
  vi.mocked(loadCurrentVendorAssessment).mockRejectedValue(new ApiError(404, "Not found"));
});

describe("VendorsWorkspace", () => {
  it("opens due diligence for the first loaded vendor when guided", async () => {
    const completed = vi.fn();
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" guideIntent={{ id: 1, type: "open-vendor-due-diligence" }} onGuideIntentCompleted={completed}/>);

    const heading = await screen.findByRole("heading", { name: "Due diligence" });
    expect(document.activeElement).toBe(heading);
    expect(completed).toHaveBeenCalledOnce();
    expect(screen.getAllByText("Card transaction processing").length).toBeGreaterThan(0);
  });

  it("opens vendor requests for the first loaded vendor when guided", async () => {
    const completed = vi.fn();
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" guideIntent={{ id: 1, type: "open-vendor-work" }} onGuideIntentCompleted={completed}/>);

    const panel = await screen.findByTestId("vendor-work-relationship-relationship-1");
    expect(document.activeElement).toBe(panel);
    expect(completed).toHaveBeenCalledOnce();
  });

  it("opens the Add vendor form when guided and the register is empty", async () => {
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [] });
    const completed = vi.fn();
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" guideIntent={{ id: 1, type: "open-vendor-due-diligence" }} onGuideIntentCompleted={completed}/>);

    const legalName = await screen.findByLabelText("Legal name");
    await waitFor(() => expect(document.activeElement).toBe(legalName));
    expect(completed).toHaveBeenCalledOnce();
  });

  it("opens the Add vendor legal-name field for the next vendor task when the register is empty", async () => {
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [] });
    const completed = vi.fn();
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" guideIntent={{ id: 1, type: "open-vendor-next-action" }} onGuideIntentCompleted={completed}/>);

    const legalName = await screen.findByLabelText("Legal name");
    await waitFor(() => expect(document.activeElement).toBe(legalName));
    await waitFor(() => expect(completed).toHaveBeenCalledWith(1));
  });

  it("focuses but does not execute the first due-diligence action for the next vendor task", async () => {
    const completed = vi.fn();
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" guideIntent={{ id: 1, type: "open-vendor-next-action" }} onGuideIntentCompleted={completed}/>);

    const action = await screen.findByRole("button", { name: "Start due diligence" });
    await waitFor(() => expect(document.activeElement).toBe(action));
    expect(startVendorAssessment).not.toHaveBeenCalled();
    expect(completed).toHaveBeenCalledWith(1);
  });

  it("preserves the currently selected vendor when opening its next task", async () => {
    const second = { ...record, vendor: { ...record.vendor, id: "vendor-2", legal_name: "Beacon Hosting Limited" }, relationship: { ...record.relationship, id: "relationship-2", vendor_id: "vendor-2", service_name: "Cloud hosting" } };
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [record, second] });
    const completed = vi.fn();
    const view = render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Beacon Hosting Limited, Cloud hosting" }));
    view.rerender(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" guideIntent={{ id: 1, type: "open-vendor-next-action" }} onGuideIntentCompleted={completed}/>);

    await waitFor(() => expect(screen.getByRole("button", { name: "Beacon Hosting Limited, Cloud hosting" }).getAttribute("aria-current")).toBe("true"));
    await waitFor(() => expect(completed).toHaveBeenCalledWith(1));
  });

  it("moves a completed due-diligence review to the first vendor-work action without executing it", async () => {
    showVendorWorkAction = true;
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("COMPLETED") });
    vi.mocked(loadVendorAssessment).mockResolvedValue(review("COMPLETED"));
    const completed = vi.fn();
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" guideIntent={{ id: 1, type: "open-vendor-next-action" }} onGuideIntentCompleted={completed}/>);

    const action = await screen.findByRole("button", { name: "Request vendor work" });
    await waitFor(() => expect(document.activeElement).toBe(action));
    expect(vendorWorkAction).not.toHaveBeenCalled();
    expect(completed).toHaveBeenCalledWith(1);
  });

  it("fails truthfully instead of offering Add vendor when the populated register has no enabled task", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("COMPLETED") });
    vi.mocked(loadVendorAssessment).mockResolvedValue(review("COMPLETED"));
    const completed = vi.fn();
    const failed = vi.fn();
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" guideIntent={{ id: 1, type: "open-vendor-next-action" }} onGuideIntentCompleted={completed} onGuideIntentFailed={failed}/>);

    const addVendor = screen.getByRole("button", { name: "Add vendor" });
    await waitFor(() => expect(failed).toHaveBeenCalledWith(1));
    expect(document.activeElement).not.toBe(addVendor);
    expect(completed).not.toHaveBeenCalled();
  });

  it("walks only the loaded register and selects the next relationship with an enabled task", async () => {
    const second = { ...record, vendor: { ...record.vendor, id: "vendor-2", legal_name: "Beacon Hosting Limited" }, relationship: { ...record.relationship, id: "relationship-2", vendor_id: "vendor-2", service_name: "Cloud hosting" } };
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [record, second] });
    vi.mocked(loadCurrentVendorAssessment).mockImplementation(async (relationshipID) => {
      if (relationshipID === "relationship-1") return { assessment: assessment("COMPLETED") };
      throw new ApiError(404, "Not found");
    });
    vi.mocked(loadVendorAssessment).mockResolvedValue(review("COMPLETED"));
    const completed = vi.fn();
    const failed = vi.fn();
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" guideIntent={{ id: 1, type: "open-vendor-next-action" }} onGuideIntentCompleted={completed} onGuideIntentFailed={failed}/>);

    const action = await screen.findByRole("button", { name: "Start due diligence" });
    await waitFor(() => expect(document.activeElement).toBe(action));
    expect(screen.getByRole("button", { name: "Beacon Hosting Limited, Cloud hosting" }).getAttribute("aria-current")).toBe("true");
    expect(loadCurrentVendorAssessment).toHaveBeenCalledWith("relationship-1");
    expect(loadCurrentVendorAssessment).toHaveBeenCalledWith("relationship-2");
    expect(loadVendorRelationships).toHaveBeenCalledOnce();
    expect(completed).toHaveBeenCalledWith(1);
    expect(failed).not.toHaveBeenCalled();
  });

  it("uses immediate scrolling when reduced motion is requested", async () => {
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue({ matches: true, addEventListener: vi.fn(), removeEventListener: vi.fn() }));
    const scroll = vi.spyOn(HTMLElement.prototype, "scrollIntoView").mockImplementation(() => undefined);
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" guideIntent={{ id: 1, type: "open-vendor-next-action" }} onGuideIntentCompleted={vi.fn()}/>);

    await waitFor(() => expect(document.activeElement).toBe(screen.getByRole("button", { name: "Start due diligence" })));
    expect(scroll).toHaveBeenCalledWith({ behavior: "auto", block: "center" });
    scroll.mockRestore();
    vi.unstubAllGlobals();
  });

  it("keeps the newer register result when an older load fails", async () => {
    let rejectOlder!: (reason?: unknown) => void;
    vi.mocked(loadVendorRelationships)
      .mockImplementationOnce(() => new Promise((_, reject) => { rejectOlder = reject; }))
      .mockResolvedValueOnce({ items: [record] });
    const view = render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria"/>);
    view.rerender(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    expect(await screen.findByRole("button", { name: /Acme Processing Limited/ })).toBeTruthy();
    rejectOlder(new Error("older request failed"));
    await waitFor(() => expect(screen.getByRole("button", { name: /Acme Processing Limited/ })).toBeTruthy());
  });

  it("restores the register without completing a failed guide intent", async () => {
    const failed = vi.fn();
    const completed = vi.fn();
    const view = render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria"/>);
    await screen.findByRole("button", { name: /Acme Processing Limited/ });
    vi.mocked(loadVendorRelationships).mockReset().mockRejectedValueOnce(new Error("unavailable")).mockResolvedValueOnce({ items: [record] });
    view.rerender(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" guideIntent={{ id: 1, type: "open-vendor-work" }} onGuideIntentFailed={failed} onGuideIntentCompleted={completed}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Try again" }));
    expect(failed).toHaveBeenCalledOnce();
    expect(await screen.findByRole("button", { name: /Acme Processing Limited/ })).toBeTruthy();
    expect(completed).not.toHaveBeenCalled();
  });

  it("acknowledges a due-diligence guide at the available error workspace", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockRejectedValue(new ApiError(503, "Unavailable"));
    const completed = vi.fn();
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" guideIntent={{ id: 1, type: "open-vendor-due-diligence" }} onGuideIntentCompleted={completed}/>);

    const errorWorkspace = await screen.findByText("Due diligence is unavailable");
    await waitFor(() => expect(document.activeElement).toBe(errorWorkspace.closest(".vdd-workspace")));
    expect(completed).toHaveBeenCalledOnce();
  });

  it("waits for a slow guided load before acknowledging the action", async () => {
    const completed = vi.fn();
    const view = render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria"/>);
    await screen.findByRole("button", { name: /Acme Processing Limited/ });
    let resolveGuidedLoad!: (value: { items: VendorRelationshipAggregate[] }) => void;
    vi.mocked(loadVendorRelationships).mockReset().mockImplementationOnce(() => new Promise((resolve) => { resolveGuidedLoad = resolve; }));
    view.rerender(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" guideIntent={{ id: 1, type: "open-vendor-due-diligence" }} onGuideIntentCompleted={completed}/>);

    await waitFor(() => expect(loadVendorRelationships).toHaveBeenCalledOnce());
    expect(completed).not.toHaveBeenCalled();
    resolveGuidedLoad({ items: [record] });
    expect(await screen.findByRole("heading", { name: "Due diligence" })).toBeTruthy();
    expect(completed).toHaveBeenCalledOnce();
  });

  it("shows the scoped vendor register and record details", async () => {
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria"/>);
    expect(await screen.findByRole("heading", { name: "Vendors" })).toBeTruthy();
    fireEvent.click(await screen.findByRole("button", { name: /Acme Processing Limited/ }));
    expect(screen.getByText("Card transaction processing")).toBeTruthy();
    expect(screen.getByText("owner-1")).toBeTruthy();
    expect(screen.getByText("Version 1")).toBeTruthy();
    expect(await screen.findByRole("heading", { name: "Due diligence" })).toBeTruthy();
    expect(screen.getByTestId("vendor-work-relationship-relationship-1")).toBeTruthy();
  });

  it("names each relationship with its vendor and service", async () => {
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria"/>);

    expect(await screen.findByRole("button", { name: "Acme Processing Limited, Card transaction processing" })).toBeTruthy();
  });

  it("uses bounded server search and loads the next relationship page", async () => {
    const second = { ...record, vendor: { ...record.vendor, id: "vendor-2", legal_name: "Beacon Hosting Limited" }, relationship: { ...record.relationship, id: "relationship-2", vendor_id: "vendor-2", service_name: "Cloud hosting" } };
    vi.mocked(loadVendorRelationships)
      .mockResolvedValueOnce({ items: [record], next_cursor: "cursor-1" })
      .mockResolvedValueOnce({ items: [second] })
      .mockResolvedValueOnce({ items: [second] });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria"/>);

    expect(await screen.findByText("More relationships are available.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Load more vendors" }));
    await waitFor(() => expect(loadVendorRelationships).toHaveBeenCalledWith({ cursor: "cursor-1", limit: 50 }));
    expect(await screen.findByRole("button", { name: "Beacon Hosting Limited, Cloud hosting" })).toBeTruthy();

    fireEvent.change(screen.getByLabelText("Search vendors and services"), { target: { value: "Beacon RC-20002" } });
    fireEvent.click(screen.getByRole("button", { name: "Search vendors" }));
    await waitFor(() => expect(loadVendorRelationships).toHaveBeenCalledWith({ search: "Beacon RC-20002", limit: 50 }));
    expect(screen.getByText("Showing 1 matching relationship")).toBeTruthy();
  });

  it("creates another service relationship using an explicitly selected existing vendor", async () => {
    vi.mocked(loadVendorRelationships)
      .mockResolvedValueOnce({ items: [record] })
      .mockResolvedValueOnce({ items: [record] });
    vi.mocked(createVendorRelationship).mockResolvedValue({ ...record, relationship: { ...record.relationship, id: "relationship-2", service_name: "Settlement support" } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria"/>);

    fireEvent.click(await screen.findByRole("button", { name: "Add vendor" }));
    fireEvent.change(screen.getByLabelText("Legal name"), { target: { value: "Acme Processing" } });
    fireEvent.click(screen.getByRole("button", { name: "Find existing vendor" }));
    expect(await screen.findByText("Possible vendor matches")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Use Acme Processing Limited for a new service relationship" }));
    fireEvent.change(screen.getByLabelText("Service supplied"), { target: { value: "Settlement support" } });
    fireEvent.click(screen.getByRole("button", { name: "Add vendor relationship" }));

    await waitFor(() => expect(createVendorRelationship).toHaveBeenCalledWith(expect.objectContaining({
      existing_relationship_id: "relationship-1", legal_name: "Acme Processing Limited", service_name: "Settlement support",
    })));
    expect(await screen.findByText("Vendor relationship added.")).toBeTruthy();
  });

  it("treats a scoped missing assessment as not started and selects only the current active vendor form", async () => {
    vi.mocked(loadFormTemplates).mockResolvedValue([
      { ...activeVendorForm, id: "draft-form", status: "DRAFT", version: 4 },
      { ...activeVendorForm, id: "other-form", code: "ACCESS-REVIEW", name: "Access review" },
      activeVendorForm,
    ]);
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    expect(await screen.findByText("No due diligence review has been started for this vendor relationship.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Start due diligence" }));
    expect(screen.getByText("Vendor security and privacy review")).toBeTruthy();
    expect(screen.queryByText("Access review")).toBeNull();
    expect(loadCurrentVendorAssessment).toHaveBeenCalledWith("relationship-1");
    expect(screen.getAllByRole("button").filter((button) => button.classList.contains("primary-button") && !(button as HTMLButtonElement).disabled)).toHaveLength(1);
  });

  it("shows an assessment load failure instead of presenting a new assessment", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockRejectedValue(new ApiError(503, "Unavailable"));
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    expect((await screen.findByRole("alert")).textContent).toContain("Due diligence is unavailable");
    expect(screen.queryByRole("button", { name: "Start due diligence" })).toBeNull();
  });

  it("reloads approved forms without leaving the selected vendor", async () => {
    vi.mocked(loadFormTemplates)
      .mockRejectedValueOnce(new ApiError(503, "Unavailable"))
      .mockResolvedValueOnce([activeVendorForm]);
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    fireEvent.click(await screen.findByRole("button", { name: "Reload forms" }));
    expect(await screen.findByRole("button", { name: "Start due diligence" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Acme Processing Limited" })).toBeTruthy();
    expect(loadFormTemplates).toHaveBeenCalledTimes(2);
  });

  it("starts due diligence with the selected relationship and exact current form", async () => {
    vi.mocked(startVendorAssessment).mockResolvedValue(assessment("SETUP_PENDING"));
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    await screen.findByText("No due diligence review has been started for this vendor relationship.");
    fireEvent.click(screen.getByRole("button", { name: "Start due diligence" }));
    fireEvent.change(screen.getByLabelText("Review due date"), { target: { value: "2099-09-30" } });
    fireEvent.click(screen.getByRole("button", { name: "Start due diligence" }));

    await waitFor(() => expect(startVendorAssessment).toHaveBeenCalledWith("relationship-1", {
      relationship_version: 1, form_template_id: "form-1", form_template_version: 3, review_due_at: "2099-09-30T23:59:59.000Z",
    }));
    expect(await screen.findByText("Setup in progress")).toBeTruthy();
  });

  it("starts a scheduled reassessment for an eligible existing relationship", async () => {
    const active = { ...record, relationship: { ...record.relationship, status: "ACTIVE" as const } };
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [active] });
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: review("COMPLETED").assessment });
    vi.mocked(loadVendorAssessment).mockResolvedValue(review("COMPLETED"));
    vi.mocked(startVendorAssessment).mockResolvedValue({ ...assessment("SETUP_PENDING"), review_kind: "PERIODIC", source_trigger: "annual-review-2099" });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    expect(await screen.findByText(/Independent security testing/)).toBeTruthy();
    fireEvent.click(await screen.findByRole("button", { name: "Start reassessment" }));
    fireEvent.change(screen.getByLabelText("Review type"), { target: { value: "PERIODIC" } });
    fireEvent.change(screen.getByLabelText("Review reference"), { target: { value: "annual-review-2099" } });
    fireEvent.change(screen.getByLabelText("Review due date"), { target: { value: "2099-09-30" } });
    fireEvent.click(screen.getByRole("button", { name: "Start reassessment" }));

    await waitFor(() => expect(startVendorAssessment).toHaveBeenCalledWith("relationship-1", {
      relationship_version: 1,
      review_kind: "PERIODIC",
      source_trigger: "annual-review-2099",
      form_template_id: "form-1",
      form_template_version: 3,
      review_due_at: "2099-09-30T23:59:59.000Z",
    }));
    expect(screen.queryByText("Proceed after the recorded access-control action is complete.")).toBeNull();
    expect(screen.queryByText(/Independent security testing/)).toBeNull();
  });

  it("restarts cancelled onboarding with the cancelled assessment as the stable source", async () => {
    const cancelled = { ...assessment("CANCELLED"), cancellation_reason: "The collection scope changed." };
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: cancelled });
    vi.mocked(startVendorAssessment).mockResolvedValue({ ...assessment("SETUP_PENDING"), id: "assessment-2", source_trigger: "RESTART:assessment-1" });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    fireEvent.click(await screen.findByRole("button", { name: "Restart due diligence" }));
    fireEvent.change(screen.getByLabelText("Review due date"), { target: { value: "2099-09-30" } });
    fireEvent.click(screen.getByRole("button", { name: "Restart due diligence" }));

    await waitFor(() => expect(startVendorAssessment).toHaveBeenCalledWith("relationship-1", {
      relationship_version: 1,
      review_kind: "ONBOARDING",
      restart_assessment_id: "assessment-1",
      form_template_id: "form-1",
      form_template_version: 3,
      review_due_at: "2099-09-30T23:59:59.000Z",
    }));
  });

  it("refreshes setup status and translates a terminal failure code into a safe recovery", async () => {
    vi.mocked(loadCurrentVendorAssessment)
      .mockResolvedValueOnce({ assessment: assessment("SETUP_PENDING"), setup: { assessment_id: "assessment-1", state: "LEASED" } })
      .mockResolvedValueOnce({ assessment: assessment("SETUP_PENDING"), setup: { assessment_id: "assessment-1", state: "FAILED", failure_code: "MATTER_CREATE_FAILED" } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    fireEvent.click(await screen.findByRole("button", { name: "View setup status" }));
    expect(await screen.findByText("The review work item could not be created. Retry assessment setup; no duplicate review will be created.")).toBeTruthy();
    expect(loadCurrentVendorAssessment).toHaveBeenCalledTimes(2);
  });

  it("retries failed setup with the current version and shows the queued setup state", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("SETUP_PENDING"), setup: { assessment_id: "assessment-1", state: "FAILED", attempts: 3, failure_code: "MATTER_CREATE_FAILED" } });
    vi.mocked(retryVendorAssessmentSetup).mockResolvedValue({
      assessment: { ...assessment("SETUP_PENDING"), version: 4 },
      setup: { assessment_id: "assessment-1", state: "READY", attempts: 0, next_attempt_at: "2026-08-26T10:10:00Z" },
    });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    fireEvent.click(await screen.findByRole("button", { name: "Retry due diligence setup" }));

    await waitFor(() => expect(retryVendorAssessmentSetup).toHaveBeenCalledWith("assessment-1", { expected_version: 3 }));
    expect(await screen.findByText("Assessment setup queued. Review setup will continue in the background.")).toBeTruthy();
    expect(screen.getByText("Setup in progress")).toBeTruthy();
    expect(screen.getByRole("button", { name: "View setup status" })).toBeTruthy();
    expect(screen.getAllByRole("button").filter((button) => button.classList.contains("primary-button") && !(button as HTMLButtonElement).disabled)).toHaveLength(1);
  });

  it("opens the exact current evidence request when collection is in progress", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("COLLECTING") });
    const onOpenRequest = vi.fn();
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1" onOpenRequest={onOpenRequest}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Review request status" }));
    expect(onOpenRequest).toHaveBeenCalledWith("request-1");
  });

  it("sends the vendor request and preserves a truthful partial-delivery recovery", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("READY_TO_SEND") });
    vi.mocked(sendVendorAssessmentRequest).mockResolvedValue({
      assessment: { ...assessment("COLLECTING"), current_request_id: "request-1" },
      request: { id: "request-1", status: "READY", deadline: "2099-09-20T23:59:59Z" },
      state: "LINK_CREATED_EMAIL_NOT_SENT", recovery: "Copy the secure link or retry delivery.", capture_url: "https://capture.example.test/?capture_invite=secret",
    });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1" onOpenRequest={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Send due diligence request" }));
    fireEvent.change(screen.getByLabelText("Vendor contact email"), { target: { value: "security@vendor.example" } });
    fireEvent.change(screen.getByLabelText("Response due date"), { target: { value: "2099-09-20" } });
    fireEvent.click(screen.getByRole("button", { name: "Send due diligence request" }));

    expect((await screen.findByRole("alert")).textContent).toContain("Email delivery did not complete");
    expect(sendVendorAssessmentRequest).toHaveBeenCalledWith("assessment-1", expect.objectContaining({ audience: "security@vendor.example", expected_version: 3 }));
    expect(screen.queryByText("security@vendor.example")).toBeNull();
  });

  it("reissues the current request without replacing the request-status action", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("COLLECTING") });
    vi.mocked(reissueVendorAssessmentRequest).mockResolvedValue({
      assessment: { ...assessment("COLLECTING"), version: 4 }, request: { id: "request-1", status: "READY" }, state: "DELIVERED",
    });
    const onOpenRequest = vi.fn();
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1" onOpenRequest={onOpenRequest}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Send replacement link" }));
    fireEvent.change(screen.getByLabelText("Vendor contact email"), { target: { value: "security@vendor.example" } });
    fireEvent.click(screen.getByRole("button", { name: "Send replacement link" }));

    await waitFor(() => expect(reissueVendorAssessmentRequest).toHaveBeenCalledWith("assessment-1", {
      expected_version: 3, audience: "security@vendor.example", invitation_ttl_minutes: 1440,
    }));
    expect(await screen.findByText("Replacement link sent. Previous access to this request has ended.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Review request status" }));
    expect(onOpenRequest).toHaveBeenCalledWith("request-1");
  });

  it("loads the submitted response and starts review with the current assessment version", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("SUBMITTED") });
    vi.mocked(loadVendorAssessment).mockResolvedValue(review("SUBMITTED"));
    vi.mocked(startVendorAssessmentReview).mockResolvedValue({ ...assessment("UNDER_REVIEW"), version: 4 });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    expect(await screen.findByText(/Independent security testing/)).toBeTruthy();
    expect(screen.getByText("1 of 1 required answers received")).toBeTruthy();
    expect(screen.getByText("Provisional score: 86 of 100 · Form version 3")).toBeTruthy();
    expect(screen.getByText("Prefilled from procurement · Observed 27 Aug 2026")).toBeTruthy();
    expect(screen.getByText("Scan complete · Vendor supplied evidence")).toBeTruthy();
    fireEvent.click(await screen.findByRole("button", { name: "Review vendor response" }));

    await waitFor(() => expect(startVendorAssessmentReview).toHaveBeenCalledWith("assessment-1", { expected_version: 3 }));
    expect(await screen.findByRole("button", { name: "Record assessment conclusion" })).toBeTruthy();
  });

  it("records the reviewer conclusion without changing the vendor relationship", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("UNDER_REVIEW") });
    vi.mocked(loadVendorAssessment).mockResolvedValue(review("UNDER_REVIEW"));
    vi.mocked(completeVendorAssessment).mockResolvedValue(review("COMPLETED").assessment);
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    fireEvent.click(await screen.findByRole("button", { name: "Record assessment conclusion" }));
    fireEvent.change(screen.getByLabelText("Conclusion"), { target: { value: "SATISFACTORY_WITH_CONDITIONS" } });
    fireEvent.change(screen.getByLabelText("Assessment basis"), { target: { value: "Proceed after the recorded access-control action is complete." } });
    fireEvent.click(screen.getByRole("button", { name: "Record assessment conclusion" }));

    await waitFor(() => expect(completeVendorAssessment).toHaveBeenCalledWith("assessment-1", expect.objectContaining({
      expected_version: 3,
      conclusion: "SATISFACTORY_WITH_CONDITIONS",
      rationale: "Proceed after the recorded access-control action is complete.",
    })));
    expect(await screen.findByText("Assessment conclusion recorded. The vendor relationship status has not changed.")).toBeTruthy();
  });

  it("sends clarification through the secure outcome contract and clears the raw audience", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("UNDER_REVIEW") });
    vi.mocked(loadVendorAssessment).mockResolvedValue(review("UNDER_REVIEW"));
    vi.mocked(requestVendorAssessmentClarification).mockResolvedValue({
      assessment: { ...assessment("COLLECTING"), version: 4, current_request_id: "request-2" },
      state: "DELIVERED",
    });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1" onOpenRequest={vi.fn()}/>);

    fireEvent.click(await screen.findByRole("button", { name: "Request clarification" }));
    fireEvent.click(screen.getByLabelText("Independent security testing"));
    fireEvent.change(screen.getByLabelText("What the vendor must provide"), { target: { value: "Provide the current independent test report." } });
    fireEvent.change(screen.getByLabelText("Vendor contact email"), { target: { value: "security@vendor.example" } });
    fireEvent.change(screen.getByLabelText("Response due date"), { target: { value: "2099-09-12" } });
    fireEvent.click(screen.getByRole("button", { name: "Send clarification request" }));

    await waitFor(() => expect(requestVendorAssessmentClarification).toHaveBeenCalledWith("assessment-1", expect.objectContaining({
      expected_version: 3, request_fields: ["security-testing"], audience: "security@vendor.example", invitation_ttl_minutes: 1440,
    })));
    expect(await screen.findByText("Clarification sent. The assessment will return to review when the vendor responds.")).toBeTruthy();
    expect(screen.queryByText("security@vendor.example")).toBeNull();
  });

  it("reloads canonical findings after the deficiency command instead of adding one locally", async () => {
    const initial = review("UNDER_REVIEW");
    const refreshed = { ...initial, assessment: { ...initial.assessment, version: 4 }, matters: [...initial.matters, { matter_id: "matter-2", type: "VENDOR_DEFICIENCY", status: "OPEN", title: "Current security test required" }] };
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("UNDER_REVIEW") });
    vi.mocked(loadVendorAssessment).mockResolvedValueOnce(initial).mockResolvedValueOnce(refreshed);
    vi.mocked(createVendorAssessmentDeficiency).mockResolvedValue({
      assessment: refreshed.assessment,
      matter: { matter: { id: "matter-2", reference: "MAT-002", title: "Current security test required", status: "OPEN" } },
    });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    fireEvent.click(await screen.findByRole("button", { name: "Record finding" }));
    fireEvent.change(screen.getByLabelText("Finding reference"), { target: { value: "security-test-report" } });
    fireEvent.change(screen.getByLabelText("Finding title"), { target: { value: "Current security test required" } });
    fireEvent.change(screen.getByLabelText("Finding details"), { target: { value: "The submitted report is no longer current for this review." } });
    fireEvent.change(screen.getByLabelText("Action due date"), { target: { value: "2099-09-20" } });
    fireEvent.click(screen.getByRole("button", { name: "Record finding" }));

    await waitFor(() => expect(createVendorAssessmentDeficiency).toHaveBeenCalledWith("assessment-1", expect.objectContaining({ expected_version: 3, trigger_key: "security-test-report" })));
    expect(await screen.findByText("Current security test required")).toBeTruthy();
    expect(loadVendorAssessment).toHaveBeenCalledTimes(2);
  });

  it("renders the refreshed review returned by a document decision", async () => {
    const initial = review("UNDER_REVIEW");
    const refreshed = { ...initial, assessment: { ...initial.assessment, version: 4 }, documents: initial.documents.map((document) => ({ ...document, status: "VALIDATED", evidence_class: "BANK_VALIDATED" })) };
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("UNDER_REVIEW") });
    vi.mocked(loadVendorAssessment).mockResolvedValue(initial);
    vi.mocked(reviewVendorAssessmentDocument).mockResolvedValue(refreshed);
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    fireEvent.click(await screen.findByRole("button", { name: "Validate document" }));
    fireEvent.change(screen.getByLabelText("Evidence class"), { target: { value: "BANK_VALIDATED" } });
    fireEvent.click(screen.getByRole("button", { name: "Record validation" }));

    await waitFor(() => expect(reviewVendorAssessmentDocument).toHaveBeenCalledWith("assessment-1", "artifact-1", {
      expected_version: 3, decision: "VALIDATE", document_type: "SECURITY_TEST", evidence_class: "BANK_VALIDATED", valid_until: "2027-08-01",
    }));
    expect(await screen.findByText("Validated · Bank validated evidence")).toBeTruthy();
  });

  it("opens an authorized assessment document in a separate browser context", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: assessment("UNDER_REVIEW") });
    vi.mocked(loadVendorAssessment).mockResolvedValue(review("UNDER_REVIEW"));
    vi.mocked(vendorAssessmentDocumentURL).mockReturnValue("/api/v1/vendor-assessments/assessment-1/requests/request-1/documents/artifact-1/open");
    const openDocument = vi.spyOn(window, "open").mockImplementation(() => null);
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    fireEvent.click(await screen.findByRole("button", { name: "Open document" }));

    expect(vendorAssessmentDocumentURL).toHaveBeenCalledWith("assessment-1", "request-1", "artifact-1");
    expect(openDocument).toHaveBeenCalledWith("/api/v1/vendor-assessments/assessment-1/requests/request-1/documents/artifact-1/open", "_blank", "noopener,noreferrer");
    openDocument.mockRestore();
  });

  it("shows a completed assessment as a read-only review record", async () => {
    vi.mocked(loadCurrentVendorAssessment).mockResolvedValue({ assessment: review("COMPLETED").assessment });
    vi.mocked(loadVendorAssessment).mockResolvedValue(review("COMPLETED"));
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    expect(await screen.findByText("Proceed after the recorded access-control action is complete.")).toBeTruthy();
    expect(screen.getByText("The next resilience exercise remains due.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "View completed assessment" })).toBeNull();
  });

  it("states the exact empty population and next action", async () => {
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [] });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria"/>);
    expect(await screen.findByText("No vendor relationships found for Clear Bank Nigeria.")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Add vendor" })).toBeTruthy();
  });

  it("creates a vendor relationship without browser-supplied identity", async () => {
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [] });
    vi.mocked(createVendorRelationship).mockResolvedValue(record);
    const onTarget = vi.fn();
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onTarget={onTarget}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Add vendor" }));
    fireEvent.change(screen.getByLabelText("Legal name"), { target: { value: "Acme Processing Limited" } });
    fireEvent.change(screen.getByLabelText("Service supplied"), { target: { value: "Card transaction processing" } });
    fireEvent.change(screen.getByLabelText("Criticality"), { target: { value: "IMPORTANT" } });
    fireEvent.change(screen.getByLabelText("Privacy role"), { target: { value: "PROCESSOR" } });
    fireEvent.click(screen.getByRole("button", { name: "Add vendor relationship" }));
    await waitFor(() => expect(createVendorRelationship).toHaveBeenCalled());
    const call = vi.mocked(createVendorRelationship).mock.calls[0];
    if (!call) throw new Error("createVendorRelationship was not called");
    expect(call[0]).toEqual(expect.not.objectContaining({ tenant_id: expect.anything(), legal_entity_id: expect.anything(), actor_id: expect.anything() }));
    expect(onTarget).toHaveBeenCalledWith("relationship-1");
    expect(await screen.findByText("Vendor relationship added.")).toBeTruthy();
  });

  it("preserves entered values when a concurrent update wins", async () => {
    vi.mocked(updateVendorRelationship).mockRejectedValue(new ApiError(409, "This vendor relationship changed."));
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor relationship" }));
    expect(screen.queryByLabelText("Legal name")).toBeNull();
    expect(screen.getByText("These details are shared across the bank and cannot be changed from this service relationship.")).toBeTruthy();
    const service = screen.getByLabelText("Service supplied") as HTMLInputElement;
    fireEvent.change(service, { target: { value: "Card processing and settlement" } });
    fireEvent.click(screen.getByRole("button", { name: "Save vendor relationship" }));
    expect(await screen.findByText("This record changed. Your entries are still here; reload the record before saving again.")).toBeTruthy();
    expect(service.value).toBe("Card processing and settlement");
  });

  it("keeps a relationship draft open while register navigation is unavailable", async () => {
    const second = { ...record, vendor: { ...record.vendor, id: "vendor-2", legal_name: "Beacon Hosting Limited" }, relationship: { ...record.relationship, id: "relationship-2", vendor_id: "vendor-2", service_name: "Cloud hosting" } };
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [record, second] });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor relationship" }));
    const service = screen.getByLabelText("Service supplied") as HTMLInputElement;
    fireEvent.change(service, { target: { value: "Card processing draft" } });
    expect(screen.queryByRole("button", { name: "Add vendor" })).toBeNull();
    expect((screen.getByRole("searchbox", { name: "Search vendors and services" }) as HTMLInputElement).disabled).toBe(true);
    const otherVendor = screen.getByRole("button", { name: "Beacon Hosting Limited, Cloud hosting" }) as HTMLButtonElement;
    expect(otherVendor.disabled).toBe(true);
    fireEvent.click(otherVendor);
    expect(service.value).toBe("Card processing draft");
  });

  it("updates only relationship-scoped fields", async () => {
    vi.mocked(updateVendorRelationship).mockResolvedValue({ ...record, relationship: { ...record.relationship, service_name: "Card settlement", version: 2 } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor relationship" }));
    fireEvent.change(screen.getByLabelText("Service supplied"), { target: { value: "Card settlement" } });
    fireEvent.click(screen.getByRole("button", { name: "Save vendor relationship" }));
    await waitFor(() => expect(updateVendorRelationship).toHaveBeenCalled());
    const call = vi.mocked(updateVendorRelationship).mock.calls[0];
    if (!call) throw new Error("updateVendorRelationship was not called");
    expect(call[1]).toEqual({
      expected_version: 1, service_name: "Card settlement", criticality: "IMPORTANT", privacy_role: "PROCESSOR",
      effective_from: undefined, renewal_at: undefined,
    });
    expect(call[1]).toEqual(expect.not.objectContaining({ legal_name: expect.anything(), registration_ref: expect.anything() }));
  });

  it("renders a discovered same-origin icon in the register and selected vendor heading", async () => {
    const branded = { ...record, brand: { state: "WEBSITE_ICON", source: "VENDOR_WEBSITE", asset_token: "brand-4", version: 4, event_version: 4 } as const };
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [branded] });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    const icons = await screen.findAllByRole("img", { name: "Acme Processing Limited icon" });
    expect(icons).toHaveLength(1);
    expect(icons[0]?.getAttribute("src")).toBe("/api/v1/vendor-identities/vendor-1/brand?version=brand-4");
    expect(document.querySelector('img[src^="http"]')).toBeNull();
    expect(screen.getByText("Website icon available")).toBeTruthy();
  });

  it.each([
    ["PENDING", "Website icon pending"],
    ["UNAVAILABLE", "Vendor icon unavailable"],
    ["APPROVED_LOGO", "Approved logo"],
    ["WEBSITE_ICON", "Website icon available"],
  ] as const)("shows the %s brand state without changing due-diligence status", async (state, label) => {
    const branded = { ...record, brand: { state, version: 2, event_version: 2 } };
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [branded] });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    expect(await screen.findByText(label)).toBeTruthy();
    expect(await screen.findByRole("button", { name: "Start due diligence" })).toBeTruthy();
  });

  it("updates shared vendor details in every loaded relationship without changing relationship versions", async () => {
    const second = { ...record, relationship: { ...record.relationship, id: "relationship-2", service_name: "Settlement reporting", version: 7 } };
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [record, second] });
    vi.mocked(loadVendorIdentity).mockResolvedValue({ vendor: record.vendor, brand: { state: "UNAVAILABLE", version: 0, event_version: 0 } });
    vi.mocked(updateVendorIdentity).mockResolvedValue({ vendor: { ...record.vendor, legal_name: "Acme Payments Limited", website_domain: "acme.example", version: 2 }, brand: { state: "PENDING", version: 1, event_version: 1 } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);

    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    const legalName = await screen.findByLabelText("Legal name");
    fireEvent.change(legalName, { target: { value: "Acme Payments Limited" } });
    fireEvent.change(screen.getByLabelText("Website domain"), { target: { value: "acme.example" } });
    fireEvent.click(screen.getByRole("button", { name: "Save vendor details" }));

    await waitFor(() => expect(updateVendorIdentity).toHaveBeenCalledWith("vendor-1", expect.objectContaining({ expected_version: 1, legal_name: "Acme Payments Limited", website_domain: "acme.example" })));
    expect(await screen.findByText("Vendor details updated.")).toBeTruthy();
    expect(screen.getAllByText("Acme Payments Limited").length).toBeGreaterThanOrEqual(2);
    fireEvent.click(screen.getByRole("button", { name: "Acme Payments Limited, Settlement reporting" }));
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor relationship" }));
    fireEvent.click(screen.getByRole("button", { name: "Save vendor relationship" }));
    await waitFor(() => expect(updateVendorRelationship).toHaveBeenCalledWith("relationship-2", expect.objectContaining({ expected_version: 7 })));
  });

  it("rejects a URL, path, credentials, port and IP before updating a vendor identity", async () => {
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    const website = await screen.findByLabelText("Website domain");
    for (const value of ["https://acme.example", "acme.example/icon", "user@acme.example", "acme.example:443", "127.0.0.1"]) {
      fireEvent.change(website, { target: { value } });
      fireEvent.click(screen.getByRole("button", { name: "Save vendor details" }));
      expect(await screen.findByText(/website hostname only/i)).toBeTruthy();
    }
    expect(updateVendorIdentity).not.toHaveBeenCalled();
  });

  it("moves keyboard focus into the vendor identity form and passes axe", async () => {
    const { container } = render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    const legalName = await screen.findByLabelText("Legal name");
    await waitFor(() => expect(document.activeElement).toBe(legalName));
    const results = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
    expect(results.violations.map((violation) => violation.id)).toEqual([]);
    expect(screen.getByRole("button", { name: "Cancel" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Save vendor details" })).toBeTruthy();
  });

  it("keeps the vendor draft open and removes Add vendor while editing shared details", async () => {
    const second = { ...record, vendor: { ...record.vendor, id: "vendor-2", legal_name: "Beacon Hosting Limited" }, relationship: { ...record.relationship, id: "relationship-2", vendor_id: "vendor-2", service_name: "Cloud hosting" } };
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [record, second] });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    const legalName = await screen.findByLabelText("Legal name") as HTMLInputElement;
    fireEvent.change(legalName, { target: { value: "Acme Payments draft" } });
    expect(screen.queryByRole("button", { name: "Add vendor" })).toBeNull();
    expect(screen.getByRole("button", { name: "Return to relationship" })).toBeTruthy();
    const searchInput = screen.getByRole("searchbox", { name: "Search vendors and services" }) as HTMLInputElement;
    const searchButton = screen.getByRole("button", { name: "Search vendors" }) as HTMLButtonElement;
    const otherVendor = screen.getByRole("button", { name: "Beacon Hosting Limited, Cloud hosting" }) as HTMLButtonElement;
    expect(searchInput.disabled).toBe(true);
    expect(searchButton.disabled).toBe(true);
    expect(otherVendor.disabled).toBe(true);
    fireEvent.submit(searchButton.closest("form")!);
    fireEvent.click(otherVendor);
    expect(loadVendorRelationships).toHaveBeenCalledOnce();
    expect(legalName.value).toBe("Acme Payments draft");
  });

  it("locks brand mutations while an identity save is pending", async () => {
    const pending = deferred<Awaited<ReturnType<typeof updateVendorIdentity>>>();
    vi.mocked(updateVendorIdentity).mockReturnValue(pending.promise);
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    const file = new File([new Uint8Array(30)], "approved.png", { type: "image/png" });
    fireEvent.change(await screen.findByLabelText("Approved logo file"), { target: { files: [file] } });
    fireEvent.click(screen.getByRole("button", { name: "Save vendor details" }));
    await waitFor(() => expect(updateVendorIdentity).toHaveBeenCalledOnce());
    expect(screen.getByRole("button", { name: "Use approved logo" }).hasAttribute("disabled")).toBe(true);
    expect((screen.getByLabelText("Approved logo file") as HTMLInputElement).disabled).toBe(true);
    fireEvent.click(screen.getByRole("button", { name: "Use approved logo" }));
    expect(uploadApprovedVendorLogo).not.toHaveBeenCalled();
    pending.resolve({ vendor: { ...record.vendor, version: 2 }, brand: { state: "UNAVAILABLE", version: 0, event_version: 0 } });
    await waitFor(() => expect(screen.queryByText("Saving…")).toBeNull());
  });

  it("locks identity mutations while a brand save is pending", async () => {
    const pending = deferred<Awaited<ReturnType<typeof uploadApprovedVendorLogo>>>();
    vi.mocked(uploadApprovedVendorLogo).mockReturnValue(pending.promise);
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    const legalName = await screen.findByLabelText("Legal name") as HTMLInputElement;
    const file = new File([new Uint8Array(30)], "approved.png", { type: "image/png" });
    fireEvent.change(screen.getByLabelText("Approved logo file"), { target: { files: [file] } });
    fireEvent.click(screen.getByRole("button", { name: "Use approved logo" }));
    await waitFor(() => expect(uploadApprovedVendorLogo).toHaveBeenCalledOnce());
    expect(legalName.disabled).toBe(true);
    expect(screen.getByRole("button", { name: "Save vendor details" }).hasAttribute("disabled")).toBe(true);
    fireEvent.submit(legalName.closest("form")!);
    expect(updateVendorIdentity).not.toHaveBeenCalled();
    pending.resolve({ vendor: record.vendor, brand: { state: "APPROVED_LOGO", source: "APPROVED_UPLOAD", asset_token: "approved-1", version: 1, event_version: 1 } });
    expect(await screen.findByText("Approved logo saved.")).toBeTruthy();
  });

  it("keeps the current brand dimension when an identity response contains stale brand data", async () => {
    const currentBrand = { state: "APPROVED_LOGO", source: "APPROVED_UPLOAD", asset_token: "approved-2", version: 2, event_version: 2 } as const;
    vi.mocked(loadVendorIdentity).mockResolvedValue({ vendor: record.vendor, brand: currentBrand });
    vi.mocked(updateVendorIdentity).mockResolvedValue({ vendor: { ...record.vendor, version: 2 }, brand: { state: "PENDING", version: 1, event_version: 1 } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    fireEvent.click(await screen.findByRole("button", { name: "Save vendor details" }));
    expect(await screen.findByText("Vendor details updated.")).toBeTruthy();
    expect(screen.getByText("Approved logo")).toBeTruthy();
  });

  it("keeps the current identity dimension when a brand response contains stale vendor data", async () => {
    vi.mocked(loadVendorIdentity).mockResolvedValue({ vendor: { ...record.vendor, legal_name: "Acme Payments Limited", version: 2 }, brand: { state: "UNAVAILABLE", version: 0, event_version: 0 } });
    vi.mocked(uploadApprovedVendorLogo).mockResolvedValue({ vendor: record.vendor, brand: { state: "APPROVED_LOGO", source: "APPROVED_UPLOAD", asset_token: "approved-1", version: 1, event_version: 1 } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    const file = new File([new Uint8Array(30)], "approved.png", { type: "image/png" });
    fireEvent.change(await screen.findByLabelText("Approved logo file"), { target: { files: [file] } });
    fireEvent.click(screen.getByRole("button", { name: "Use approved logo" }));
    expect(await screen.findByText("Approved logo saved.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Return to relationship" }));
    expect(await screen.findByRole("heading", { name: "Acme Payments Limited" })).toBeTruthy();
    expect(screen.getByText("Vendor version 2")).toBeTruthy();
  });

  it("accepts a newer brand dimension returned with an identity update", async () => {
    vi.mocked(loadVendorIdentity).mockResolvedValue({ vendor: record.vendor, brand: { state: "PENDING", version: 1, event_version: 1 } });
    vi.mocked(updateVendorIdentity).mockResolvedValue({ vendor: { ...record.vendor, version: 2 }, brand: { state: "APPROVED_LOGO", source: "APPROVED_UPLOAD", asset_token: "approved-2", version: 2, event_version: 2 } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    fireEvent.click(await screen.findByRole("button", { name: "Save vendor details" }));
    expect(await screen.findByText("Vendor details updated.")).toBeTruthy();
    expect(screen.getByText("Approved logo")).toBeTruthy();
  });

  it("accepts a newer identity dimension returned with a brand update", async () => {
    vi.mocked(loadVendorIdentity).mockResolvedValue({ vendor: record.vendor, brand: { state: "UNAVAILABLE", version: 0, event_version: 0 } });
    vi.mocked(uploadApprovedVendorLogo).mockResolvedValue({ vendor: { ...record.vendor, legal_name: "Acme Payments Limited", version: 2 }, brand: { state: "APPROVED_LOGO", source: "APPROVED_UPLOAD", asset_token: "approved-1", version: 1, event_version: 1 } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    const file = new File([new Uint8Array(30)], "approved.png", { type: "image/png" });
    fireEvent.change(await screen.findByLabelText("Approved logo file"), { target: { files: [file] } });
    fireEvent.click(screen.getByRole("button", { name: "Use approved logo" }));
    expect(await screen.findByText("Approved logo saved.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Return to relationship" }));
    expect(await screen.findByRole("heading", { name: "Acme Payments Limited" })).toBeTruthy();
    expect(screen.getByText("Vendor version 2")).toBeTruthy();
  });

  it("reloads an identity conflict without replacing the user's entries", async () => {
    vi.mocked(loadVendorIdentity)
      .mockResolvedValueOnce({ vendor: record.vendor, brand: { state: "UNAVAILABLE", version: 0, event_version: 0 } })
      .mockResolvedValueOnce({ vendor: { ...record.vendor, legal_name: "Acme Processing PLC", version: 2 }, brand: { state: "PENDING", version: 1, event_version: 1 } });
    vi.mocked(updateVendorIdentity)
      .mockRejectedValueOnce(new ApiError(409, "changed"))
      .mockResolvedValueOnce({ vendor: { ...record.vendor, legal_name: "Acme Payments draft", version: 3 }, brand: { state: "PENDING", version: 1, event_version: 1 } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    const legalName = await screen.findByLabelText("Legal name") as HTMLInputElement;
    fireEvent.change(legalName, { target: { value: "Acme Payments draft" } });
    fireEvent.click(screen.getByRole("button", { name: "Save vendor details" }));
    fireEvent.click(await screen.findByRole("button", { name: "Reload current vendor" }));
    await waitFor(() => expect(loadVendorIdentity).toHaveBeenCalledTimes(2));
    expect(legalName.value).toBe("Acme Payments draft");
    fireEvent.click(screen.getByRole("button", { name: "Save vendor details" }));
    await waitFor(() => expect(updateVendorIdentity).toHaveBeenLastCalledWith("vendor-1", expect.objectContaining({ expected_version: 2, legal_name: "Acme Payments draft" })));
  });

  it("reloads a logo conflict while preserving the staged file and rotating the rejected request key", async () => {
    vi.mocked(loadVendorIdentity)
      .mockResolvedValueOnce({ vendor: record.vendor, brand: { state: "UNAVAILABLE", version: 0, event_version: 0 } })
      .mockResolvedValueOnce({ vendor: record.vendor, brand: { state: "PENDING", version: 2, event_version: 2 } });
    vi.mocked(uploadApprovedVendorLogo)
      .mockRejectedValueOnce(new ApiError(409, "changed"))
      .mockResolvedValueOnce({ vendor: record.vendor, brand: { state: "APPROVED_LOGO", source: "APPROVED_UPLOAD", asset_token: "approved-3", version: 3, event_version: 3 } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    const file = new File([new Uint8Array(30)], "approved.png", { type: "image/png" });
    fireEvent.change(await screen.findByLabelText("Approved logo file"), { target: { files: [file] } });
    fireEvent.click(screen.getByRole("button", { name: "Use approved logo" }));
    fireEvent.click(await screen.findByRole("button", { name: "Reload current vendor" }));
    await screen.findByText("Current vendor details reloaded. Your entries and selected file are unchanged.");
    expect(screen.getByText("approved.png · 30 B")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Use approved logo" }));
    await waitFor(() => expect(uploadApprovedVendorLogo).toHaveBeenCalledTimes(2));
    expect(vi.mocked(uploadApprovedVendorLogo).mock.calls[1]?.[2]).toBe(2);
    expect(vi.mocked(uploadApprovedVendorLogo).mock.calls[1]?.[3]).not.toBe(vi.mocked(uploadApprovedVendorLogo).mock.calls[0]?.[3]);
  });

  it("offers an exact reload when a committed logo response is degraded", async () => {
    vi.mocked(uploadApprovedVendorLogo).mockResolvedValue({ status: "COMMITTED", aggregate_type: "VENDOR_BRAND", aggregate_id: "vendor-1", version: 1, response_degraded: true });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    const file = new File([new Uint8Array(30)], "approved.png", { type: "image/png" });
    fireEvent.change(await screen.findByLabelText("Approved logo file"), { target: { files: [file] } });
    fireEvent.click(screen.getByRole("button", { name: "Use approved logo" }));
    expect(await screen.findByText("The approved logo was saved, but the updated vendor could not be loaded. Reload the current vendor to confirm the saved icon.")).toBeTruthy();
    expect(screen.getByText("approved.png · 30 B")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Reload current vendor" })).toBeTruthy();
  });

  it("stages and replaces an approved logo before explicit upload", async () => {
    vi.mocked(uploadApprovedVendorLogo).mockResolvedValue({ vendor: record.vendor, brand: { state: "APPROVED_LOGO", source: "APPROVED_UPLOAD", asset_token: "approved-2", version: 2, event_version: 2 } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    expect(await screen.findByRole("button", { name: "Choose logo" })).toBeTruthy();
    const input = await screen.findByLabelText("Approved logo file");
    const first = new File([new Uint8Array(30)], "first.png", { type: "image/png" });
    const replacement = new File([new Uint8Array(40)], "approved.webp", { type: "image/webp" });
    fireEvent.change(input, { target: { files: [first] } });
    expect(screen.getByText("first.png · 30 B")).toBeTruthy();
    expect(screen.getByText("Selected file is ready to save.")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Replace logo" })).toBeTruthy();
    expect(uploadApprovedVendorLogo).not.toHaveBeenCalled();
    fireEvent.change(input, { target: { files: [replacement] } });
    expect(screen.getByText("approved.webp · 40 B")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Use approved logo" }));
    await waitFor(() => expect(uploadApprovedVendorLogo).toHaveBeenCalledWith("vendor-1", replacement, 0, expect.stringMatching(/^vendor-brand-/)));
    expect(await screen.findByText("Approved logo saved.")).toBeTruthy();
  });

  it("rejects SVG and files over 512 KiB before upload", async () => {
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    const input = await screen.findByLabelText("Approved logo file");
    fireEvent.change(input, { target: { files: [new File(["<svg/>"] , "logo.svg", { type: "image/svg+xml" })] } });
    expect(await screen.findByText("Select a PNG, JPEG, WebP or ICO file.")).toBeTruthy();
    fireEvent.change(input, { target: { files: [new File([new Uint8Array(524289)], "large.png", { type: "image/png" })] } });
    expect(await screen.findByText("Select a logo no larger than 512 KiB.")).toBeTruthy();
    expect(uploadApprovedVendorLogo).not.toHaveBeenCalled();
  });

  it.each([
    [409, "The vendor logo changed. Reload the current vendor, then try the selected file again."],
    [403, "Your current role cannot change the approved vendor logo. The selected file is still here."],
    [503, "The approved logo could not be saved. The selected file is still here; try again."],
  ])("preserves the selected logo after a %s upload failure", async (status, message) => {
    vi.mocked(uploadApprovedVendorLogo).mockRejectedValue(new ApiError(status, "failed"));
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    const input = await screen.findByLabelText("Approved logo file");
    const file = new File([new Uint8Array(30)], "approved.png", { type: "image/png" });
    fireEvent.change(input, { target: { files: [file] } });
    fireEvent.click(screen.getByRole("button", { name: "Use approved logo" }));
    expect(await screen.findByText(message)).toBeTruthy();
    expect(screen.getByText("approved.png · 30 B")).toBeTruthy();
  });

  it("reuses the upload request key and brand version after an ambiguous failure", async () => {
    vi.mocked(uploadApprovedVendorLogo)
      .mockRejectedValueOnce(new ApiError(503, "response unavailable"))
      .mockResolvedValueOnce({ vendor: record.vendor, brand: { state: "APPROVED_LOGO", source: "APPROVED_UPLOAD", asset_token: "approved-2", version: 1, event_version: 1 } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    const file = new File([new Uint8Array(30)], "approved.png", { type: "image/png" });
    fireEvent.change(await screen.findByLabelText("Approved logo file"), { target: { files: [file] } });
    fireEvent.click(screen.getByRole("button", { name: "Use approved logo" }));
    expect(await screen.findByText(/selected file is still here/i)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Use approved logo" }));
    await waitFor(() => expect(uploadApprovedVendorLogo).toHaveBeenCalledTimes(2));
    expect(vi.mocked(uploadApprovedVendorLogo).mock.calls[0]?.slice(2)).toEqual(vi.mocked(uploadApprovedVendorLogo).mock.calls[1]?.slice(2));
  });

  it("removes an approved logo and restores the website-icon state", async () => {
    const approved = { ...record, brand: { state: "APPROVED_LOGO", source: "APPROVED_UPLOAD", asset_token: "approved-2", version: 2, event_version: 2 } as const };
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [approved] });
    vi.mocked(loadVendorIdentity).mockResolvedValue({ vendor: approved.vendor, brand: approved.brand });
    vi.mocked(removeApprovedVendorLogo).mockResolvedValue({ vendor: approved.vendor, brand: { state: "WEBSITE_ICON", source: "VENDOR_WEBSITE", asset_token: "website-1", version: 3, event_version: 3 } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    expect(await screen.findByText(/restores the website icon/i)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Remove approved logo" }));
    await waitFor(() => expect(removeApprovedVendorLogo).toHaveBeenCalledWith("vendor-1", 2, expect.stringMatching(/^vendor-brand-/)));
    expect(await screen.findByText("Website icon restored.")).toBeTruthy();
  });

  it("reuses the removal request key and brand version after an ambiguous failure", async () => {
    const approved = { ...record, brand: { state: "APPROVED_LOGO", source: "APPROVED_UPLOAD", asset_token: "approved-2", version: 2, event_version: 2 } as const };
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [approved] });
    vi.mocked(loadVendorIdentity).mockResolvedValue({ vendor: approved.vendor, brand: approved.brand });
    vi.mocked(removeApprovedVendorLogo)
      .mockRejectedValueOnce(new ApiError(503, "response unavailable"))
      .mockResolvedValueOnce({ vendor: approved.vendor, brand: { state: "WEBSITE_ICON", source: "VENDOR_WEBSITE", asset_token: "website-1", version: 3, event_version: 3 } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    const action = await screen.findByRole("button", { name: "Remove approved logo" });
    fireEvent.click(action);
    expect(await screen.findByText("The approved logo could not be removed. Try again.")).toBeTruthy();
    fireEvent.click(action);
    await waitFor(() => expect(removeApprovedVendorLogo).toHaveBeenCalledTimes(2));
    expect(vi.mocked(removeApprovedVendorLogo).mock.calls[0]?.slice(1)).toEqual(vi.mocked(removeApprovedVendorLogo).mock.calls[1]?.slice(1));
  });

  it("reloads a removal conflict and retries with the current brand version", async () => {
    const approved = { ...record, brand: { state: "APPROVED_LOGO", source: "APPROVED_UPLOAD", asset_token: "approved-2", version: 2, event_version: 2 } as const };
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [approved] });
    vi.mocked(loadVendorIdentity)
      .mockResolvedValueOnce({ vendor: approved.vendor, brand: approved.brand })
      .mockResolvedValueOnce({ vendor: approved.vendor, brand: { ...approved.brand, version: 3, event_version: 3 } });
    vi.mocked(removeApprovedVendorLogo)
      .mockRejectedValueOnce(new ApiError(409, "changed"))
      .mockResolvedValueOnce({ vendor: approved.vendor, brand: { state: "UNAVAILABLE", version: 4, event_version: 4 } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor details" }));
    fireEvent.click(await screen.findByRole("button", { name: "Remove approved logo" }));
    fireEvent.click(await screen.findByRole("button", { name: "Reload current vendor" }));
    await screen.findByText("Current vendor details reloaded. Your entries are unchanged.");
    fireEvent.click(screen.getByRole("button", { name: "Remove approved logo" }));
    await waitFor(() => expect(removeApprovedVendorLogo).toHaveBeenCalledTimes(2));
    expect(vi.mocked(removeApprovedVendorLogo).mock.calls[1]?.[1]).toBe(3);
    expect(vi.mocked(removeApprovedVendorLogo).mock.calls[1]?.[2]).not.toBe(vi.mocked(removeApprovedVendorLogo).mock.calls[0]?.[2]);
    expect(await screen.findByText("Approved logo removed. The vendor monogram is shown until a website icon is available.")).toBeTruthy();
  });
});
