import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadFormTemplatePage, loadFormTemplateRevision } from "../formsApi";
import { ApiError } from "../http";
import { applyMatterFormRemediation, createMatterFormRemediation, loadMatterFormRemediations, sendMatterFormRemediation } from "../matterFormRemediationApi";
import type { MatterFormRemediationState } from "../matterFormRemediationApi";
import type { MatterOperation } from "../matterOperationsApi";
import type { MatterAggregate } from "../types";
import { MatterFormRemediationPanel } from "./MatterFormRemediationPanel";

vi.mock("../formsApi", () => ({ loadFormTemplatePage: vi.fn(), loadFormTemplateRevision: vi.fn() }));
vi.mock("../matterFormRemediationApi", () => ({
  applyMatterFormRemediation: vi.fn(),
  createMatterFormRemediation: vi.fn(),
  loadMatterFormRemediations: vi.fn(),
  sendMatterFormRemediation: vi.fn(),
}));

const aggregate: MatterAggregate = {
  type_label: "Control gap", status_label: "Initial review", next_action: "Confirm scope and owner",
  matter: {
    id: "matter-1", tenant_id: "bank-1", legal_entity_id: "entity-1", reference: "GAP-1", type: "CONTROL_GAP", status: "TRIAGE", priority: 4,
    title: "Restore an unavailable source", summary: "Collect the missing evidence through an approved form.", scope: { access: "INTERNAL" }, known_facts: {},
    missing_facts: ["Source owner", "Recovery evidence"], contradictions: [], owner_principal_id: "owner-1",
    created_at: "2026-09-01T09:00:00Z", updated_at: "2026-09-02T09:00:00Z", version: 7,
  },
  links: [{ id: "link-1", program_id: "program-1", relationship: "AFFECTS" }], decisions: [], actions: [],
  verification_contracts: [{ id: "contract-1", expected_outcome: "The source is restored and current.", observation_period_minutes: 60, failure_response: "Keep the issue open.", status: "ACTIVE" }],
  verification_results: [], response_packages: [], closure: { ready: false, reasons: ["Evidence is still missing."] },
};

const ownerOperation: MatterOperation = {
  command: "matter.context.change", label: "Update facts and missing information", responsibility: "ACCOUNTABLE_OWNER", can_act: true,
  assigned_to: { id: "owner-1", display_name: "Program Owner", kind: "PERSON", role: "PROGRAM_OWNER" }, reason: "You own this issue.",
};
const reviewerOperation: MatterOperation = {
  command: "matter.outcome.record", label: "Record outcome", responsibility: "REVIEWER", can_act: true,
  assigned_to: { id: "reviewer-1", display_name: "Independent Reviewer", kind: "PERSON", role: "REVIEWER" }, reason: "You review this outcome.",
};

const binding = {
  id: "binding-1", legal_entity_id: "entity-1", program_id: "program-1", matter_id: "matter-1", matter_version_at_binding: 7,
  form_template_id: "form-1", form_template_version: 2,
  mappings: [
    { field_id: "source_owner", missing_item: "Source owner", fact_key: "source_owner" },
    { field_id: "recovery_evidence", missing_item: "Recovery evidence", fact_key: "recovery_evidence" },
  ],
  verification_contract_id: "contract-1", subject_type: "MATTER" as const, subject_id: "matter-1", purpose: "Supply mapped issue evidence.",
  audience_class: "EXTERNAL" as const, responder_class: "ISSUE_EVIDENCE_CONTACT", status: "ACTIVE" as const,
  effective_from: "2026-09-02T09:00:00Z", created_at: "2026-09-02T09:00:00Z", version: 1,
};

async function choose(label: string, option: string) {
  fireEvent.click(screen.getByRole("button", { name: new RegExp(label, "i") }));
  fireEvent.click(await screen.findByRole("option", { name: option }));
  await waitFor(() => expect(screen.queryByRole("listbox")).toBeNull());
}

describe("MatterFormRemediationPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(loadMatterFormRemediations).mockResolvedValue([]);
    vi.mocked(loadFormTemplatePage).mockResolvedValue({
      items: [{ template: { id: "form-1", name: "Annual return remediation" }, active_version: 2, active_status: "ACTIVE" }],
    } as never);
    vi.mocked(loadFormTemplateRevision).mockResolvedValue({
      id: "form-1", tenant_id: "bank-1", legal_entity_id: "entity-1", code: "MATTER-REMEDIATION", name: "Annual return remediation",
      purpose: "Collect exact remediation evidence.", status: "ACTIVE", is_current: true, version: 2, sensitivity: "INTERNAL", scoring_mode: "NONE",
      presentation: { default_mode: "AUTOMATIC", allow_mode_switch: false }, sections: [{ id: "evidence", title: "Evidence" }],
      fields: [
        { id: "source_owner", section_id: "evidence", label: "Source owner", type: "short_text", required: true },
        { id: "recovery_evidence", section_id: "evidence", label: "Recovery evidence", type: "long_text", required: true },
      ],
      created_at: "2026-09-01T09:00:00Z", updated_at: "2026-09-02T09:00:00Z",
    } as never);
  });

  it("creates one immutable mapping and sends the approved form through the normal delivery path", async () => {
    vi.mocked(createMatterFormRemediation).mockResolvedValue(binding);
    vi.mocked(sendMatterFormRemediation).mockResolvedValue({ binding, next_action: "Open response" });

    render(<MatterFormRemediationPanel aggregate={aggregate} operations={[ownerOperation]} onUpdated={vi.fn()} onMappingsChange={vi.fn()}/>);
    expect(screen.getByText("Send an approved form for the outstanding items. Review the response before updating this issue.")).toBeTruthy();
    expect(await screen.findByText("No approved form has been sent for the outstanding items in this issue.", { exact: false })).toBeTruthy();
    fireEvent.click(await screen.findByRole("button", { name: "Send linked form" }));
    expect(await screen.findByRole("heading", { name: "Send an approved form" })).toBeTruthy();
    expect(screen.getByText("Choose which form answer supplies each outstanding item. These choices are fixed after the request is sent.")).toBeTruthy();

    await choose("Approved form revision", "Annual return remediation · v2");
    await choose("Response field for Source owner", "Source owner");
    await choose("Response field for Recovery evidence", "Recovery evidence");
    await choose("Outcome check", "The source is restored and current.");
    fireEvent.change(await screen.findByLabelText(/^Recipient email/), { target: { value: "evidence.contact@example.com" } });
    fireEvent.change(await screen.findByLabelText(/^Due date/), { target: { value: "2026-09-12" } });
    fireEvent.click(screen.getAllByRole("button", { name: "Send linked form" }).at(-1)!);

    await waitFor(() => expect(createMatterFormRemediation).toHaveBeenCalledWith("matter-1", {
      legalEntityID: "entity-1", expectedMatterVersion: 7, programID: "program-1", formTemplateID: "form-1", formTemplateVersion: 2,
      mappings: [
        { field_id: "source_owner", missing_item: "Source owner", fact_key: "source_owner" },
        { field_id: "recovery_evidence", missing_item: "Recovery evidence", fact_key: "recovery_evidence" },
      ],
      verificationContractID: "contract-1",
    }));
    expect(sendMatterFormRemediation).toHaveBeenCalledWith("matter-1", "binding-1", expect.objectContaining({
      bindingVersion: 1, email: "evidence.contact@example.com", deadline: "2026-09-12T23:59:59.000Z",
    }));
  });

  it("applies only the exact current response as an independent review step", async () => {
    const state: MatterFormRemediationState = {
      binding,
      request: { id: "request-1", title: "Restore source evidence", status: "COMPLETED", deadline: "2026-09-12T23:59:59Z" },
      response: { id: "response-revision-1", revision: 3, current: true, state: "FINAL", completed_at: "2026-09-03T10:00:00Z" },
      next_action: "Review evidence",
    };
    vi.mocked(loadMatterFormRemediations).mockResolvedValue([state]);
    vi.mocked(applyMatterFormRemediation).mockResolvedValue({
      matter: { ...aggregate, matter: { ...aggregate.matter, missing_facts: [], version: 8 } },
      application: { id: "application-1", response_revision_id: "response-revision-1", matter_version: 8, applied_at: "2026-09-03T10:05:00Z" },
    });
    const updated = vi.fn();

    render(<MatterFormRemediationPanel aggregate={aggregate} operations={[reviewerOperation]} onUpdated={updated} onMappingsChange={vi.fn()}/>);
    expect(await screen.findByText("Explain why this response answers the outstanding items.")).toBeTruthy();
    fireEvent.change(await screen.findByLabelText("Review basis"), { target: { value: "This final response supplies both mapped items." } });
    fireEvent.click(screen.getByRole("button", { name: "Apply response" }));

    await waitFor(() => expect(applyMatterFormRemediation).toHaveBeenCalledWith("matter-1", "binding-1", {
      bindingVersion: 1, expectedMatterVersion: 7, responseRevisionID: "response-revision-1",
      rationale: "This final response supplies both mapped items.",
    }));
    expect(updated).toHaveBeenCalledWith(expect.objectContaining({ matter: expect.objectContaining({ version: 8, missing_facts: [] }) }));
  });

  it("keeps a poor response open and offers correction instead of applying it", async () => {
    vi.mocked(loadMatterFormRemediations).mockResolvedValue([{
      binding,
      request: { id: "request-1", title: "Restore source evidence", status: "COMPLETED", deadline: "2026-09-12T23:59:59Z" },
      response: { id: "response-revision-1", revision: 3, current: true, state: "FINAL", completed_at: "2026-09-03T10:00:00Z" },
      next_action: "Request correction",
    }]);
    const openRequest = vi.fn();

    render(<MatterFormRemediationPanel aggregate={aggregate} operations={[reviewerOperation]} onUpdated={vi.fn()} onOpenRequest={openRequest} onMappingsChange={vi.fn()}/>);
    expect(await screen.findByText("The response score is below the approved threshold. Review it or request a correction; this issue remains open.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Review response" }));
    expect(openRequest).toHaveBeenCalledWith("request-1");
    expect(screen.queryByRole("button", { name: "Apply response" })).toBeNull();
    expect(applyMatterFormRemediation).not.toHaveBeenCalled();
  });

  it("states the remaining outcome work after information is added", async () => {
    vi.mocked(loadMatterFormRemediations).mockResolvedValue([{
      binding,
      request: { id: "request-1", title: "Restore source evidence", status: "COMPLETED", deadline: "2026-09-12T23:59:59Z" },
      response: { id: "response-revision-1", revision: 3, current: true, state: "FINAL", completed_at: "2026-09-03T10:00:00Z" },
      application: { id: "application-1", response_revision_id: "response-revision-1", matter_version: 8, applied_at: "2026-09-03T10:05:00Z" },
      next_action: "Check outcome",
    }]);

    render(<MatterFormRemediationPanel aggregate={aggregate} operations={[reviewerOperation]} onUpdated={vi.fn()} onMappingsChange={vi.fn()}/>);

    expect(await screen.findByText("Information added · confirm the result before closing")).toBeTruthy();
  });

  it("keeps the named linked-request population visible while loading", () => {
    vi.mocked(loadMatterFormRemediations).mockReturnValue(new Promise(() => undefined));

    render(<MatterFormRemediationPanel aggregate={aggregate} operations={[ownerOperation]} onUpdated={vi.fn()} onMappingsChange={vi.fn()}/>);

    expect(screen.getByText("Checking linked requests and responses…")).toBeTruthy();
  });

  it("names a linked-request load failure and its affected action", async () => {
    vi.mocked(loadMatterFormRemediations).mockRejectedValue(new Error("Request service unavailable"));

    render(<MatterFormRemediationPanel aggregate={aggregate} operations={[ownerOperation]} onUpdated={vi.fn()} onMappingsChange={vi.fn()}/>);

    expect((await screen.findByRole("alert")).textContent).toContain("Linked form action could not be completed.");
    expect(screen.getByRole("alert").textContent).toContain("Request service unavailable");
  });

  it("opens a sent linked request without asking for another evidence record", async () => {
    vi.mocked(loadMatterFormRemediations).mockResolvedValue([{
      binding,
      request: { id: "request-1", title: "Restore source evidence", status: "READY", deadline: "2026-09-12T23:59:59Z" },
      next_action: "Open response",
    }]);
    const openRequest = vi.fn();

    render(<MatterFormRemediationPanel aggregate={aggregate} operations={[ownerOperation]} onUpdated={vi.fn()} onOpenRequest={openRequest} onMappingsChange={vi.fn()}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Open response" }));

    expect(openRequest).toHaveBeenCalledWith("request-1");
  });

  it("preserves the review basis when the issue changed", async () => {
    vi.mocked(loadMatterFormRemediations).mockResolvedValue([{
      binding,
      request: { id: "request-1", title: "Restore source evidence", status: "COMPLETED", deadline: "2026-09-12T23:59:59Z" },
      response: { id: "response-revision-1", revision: 3, current: true, state: "FINAL", completed_at: "2026-09-03T10:00:00Z" },
      next_action: "Review evidence",
    }]);
    vi.mocked(applyMatterFormRemediation).mockRejectedValue(new ApiError(409, "changed", "version_conflict"));

    render(<MatterFormRemediationPanel aggregate={aggregate} operations={[reviewerOperation]} onUpdated={vi.fn()} onMappingsChange={vi.fn()}/>);
    const basis = await screen.findByLabelText("Review basis");
    fireEvent.change(basis, { target: { value: "The response answers both outstanding items with current evidence." } });
    fireEvent.click(screen.getByRole("button", { name: "Apply response" }));

    expect((await screen.findByRole("alert")).textContent).toContain("Reload the issue before applying the response.");
    expect((basis as HTMLTextAreaElement).value).toContain("answers both outstanding items");
  });
});
