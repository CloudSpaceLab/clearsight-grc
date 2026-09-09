import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { FormTemplateProposal } from "../../formsTypes";
import { acceptFormProposal, loadFormProposal } from "../../formsApi";
import { ApiError } from "../../http";
import { FormProposalReview } from "./FormProposalReview";

vi.mock("../../formsApi", () => ({
  acceptFormProposal: vi.fn(),
  loadFormProposal: vi.fn(),
  rejectFormProposal: vi.fn(),
}));

const proposal: FormTemplateProposal = {
  id: "proposal-1",
  source_kind: "DOCUMENT",
  source_document_id: "document-1",
  source_document_version: 4,
  source_sha256: "a".repeat(64),
  status: "REVIEW_REQUIRED",
  proposed_contract: {
    scoring_mode: "NONE",
    presentation: { default_mode: "AUTOMATIC", allow_mode_switch: true },
    sections: [{ id: "identity", title: "Vendor identity" }],
    fields: [
      { id: "registered_name", section_id: "identity", label: "Registered name", type: "short_text", required: false },
      { id: "certificate", section_id: "identity", label: "Certificate of operation", type: "file", required: false },
    ],
  },
  field_changes: [
    { id: "change-name", kind: "ADD_FIELD", field: { id: "registered_name", section_id: "identity", label: "Registered name", type: "short_text", required: false }, anchor: { page: 2, paragraph: "p-4" }, confidence: .96, unresolved: ["REQUIREDNESS_UNKNOWN"] },
    { id: "change-certificate", kind: "ADD_FIELD", field: { id: "certificate", section_id: "identity", label: "Certificate of operation", type: "file", required: false }, anchor: { page: 3, paragraph: "p-8" }, confidence: .72, unresolved: ["REQUIREDNESS_UNKNOWN"] },
  ],
  unresolved_items: [{ code: "REQUIREDNESS_UNKNOWN", message: "The source does not establish whether this field is mandatory; an author must decide.", field_change_id: "change-name", anchor: { page: 2, paragraph: "p-4" } }],
  provenance: { proposal_version: "FORM_TEMPLATE_PROPOSAL_V1", source_document_id: "document-1", source_sha256: "a".repeat(64), source_version: 4, extraction_status: "EXTRACTED" },
  created_by: "author-1",
  created_at: "2026-08-29T08:00:00Z",
  updated_at: "2026-08-29T08:00:01Z",
  version: 2,
};

beforeEach(() => {
  vi.mocked(acceptFormProposal).mockReset();
  vi.mocked(loadFormProposal).mockReset();
  vi.mocked(loadFormProposal).mockResolvedValue(proposal);
  vi.mocked(acceptFormProposal).mockResolvedValue({ ...proposal, status: "ACCEPTED", result_template_id: "template-1", result_template_version: 1, accepted_change_ids: ["change-name", "change-certificate"], version: 3 });
});

describe("FormProposalReview", () => {
  it("matches each spreadsheet excerpt to its own source row", () => {
    const rowProposal = { ...proposal, field_changes: proposal.field_changes.map((change, index) => ({ ...change, anchor: { sheet: "Sheet 1", row_start: index + 2, row_end: index + 2 } })) };
    const { container } = render(<FormProposalReview proposal={rowProposal} sourceElements={[
      { ref: "row-2", kind: "TABLE", text: "First row source", anchor: { sheet: "Sheet 1", row_start: 2, row_end: 2 } },
      { ref: "row-3", kind: "TABLE", text: "Second row source", anchor: { sheet: "Sheet 1", row_start: 3, row_end: 3 } },
    ]} onProposalChange={() => undefined}/>);
    expect([...container.querySelectorAll("blockquote")].map((quote) => quote.textContent)).toEqual(["First row source", "Second row source"]);
  });

  it("reviews an unsectioned server proposal with null optional lists and creates its selected draft", async () => {
    const wireProposal = JSON.parse(JSON.stringify({
      ...proposal,
      proposed_contract: { ...proposal.proposed_contract, sections: null, fields: proposal.proposed_contract.fields.map((field) => ({ ...field, section_id: undefined })) },
      unresolved_items: null,
    })) as FormTemplateProposal;
    render(<FormProposalReview proposal={wireProposal} onProposalChange={() => undefined}/>);
    expect(screen.getByRole("textbox", { name: "Registered name" })).toBeTruthy();
    fireEvent.click(screen.getByRole("checkbox", { name: "Include Certificate of operation" }));
    fireEvent.click(screen.getByRole("button", { name: "Create draft from selected fields" }));
    await waitFor(() => expect(acceptFormProposal).toHaveBeenCalledWith("proposal-1", 2, ["change-name"]));
  });

  it.each(["GENERATING", "FAILED"] as const)("renders %s before proposal lists are populated", (status) => {
    const wireProposal = JSON.parse(JSON.stringify({ ...proposal, status, field_changes: null, unresolved_items: null, proposed_contract: { ...proposal.proposed_contract, fields: null, sections: null } })) as FormTemplateProposal;
    render(<FormProposalReview proposal={wireProposal} onProposalChange={() => undefined}/>);
    expect(screen.getByText(status === "GENERATING" ? "Preparing proposed fields from the imported document…" : "Field proposal needs attention")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Create draft from selected fields" })).toBeNull();
  });

  it("shows source anchors, confidence, unresolved decisions, and the real capture preview", () => {
    render(<FormProposalReview proposal={proposal} sourceTitle="vendor-questionnaire.docx" sourceElements={[{ ref: "p-4", kind: "FORM_CONTROL", text: "Registered name", anchor: { page: 2, paragraph: "p-4" } }]} onProposalChange={() => undefined}/>);
    expect(screen.getByRole("heading", { name: "Review proposed form fields" })).toBeTruthy();
    expect(screen.getByText("Page 2 · paragraph p-4")).toBeTruthy();
    expect(screen.getByText("96% confidence")).toBeTruthy();
    expect(screen.getByText(/source does not establish whether this field is mandatory/i)).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Response preview" })).toBeTruthy();
    expect(screen.getByRole("textbox", { name: "Registered name" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Preview Classic" })).toBeNull();
    expect(screen.getByRole("button", { name: "Show all questions" })).toBeTruthy();
    expect(screen.getByText(/Scoring weights were not inferred/i)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Review response" }));
    expect(screen.getByRole("heading", { name: "Response review preview" })).toBeTruthy();
    expect(screen.getByText(/No response will be submitted/i)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Return to questions" }));
    expect(screen.getByRole("textbox", { name: "Registered name" })).toBeTruthy();
  });

  it("creates a draft from only the selected changes using the proposal version", async () => {
    const changed = vi.fn();
    render(<FormProposalReview proposal={proposal} onProposalChange={changed}/>);
    fireEvent.click(screen.getByRole("checkbox", { name: "Include Certificate of operation" }));
    fireEvent.click(screen.getByRole("button", { name: "Create draft from selected fields" }));
    await waitFor(() => expect(acceptFormProposal).toHaveBeenCalledWith("proposal-1", 2, ["change-name"]));
    expect(changed).toHaveBeenCalledWith(expect.objectContaining({ status: "ACCEPTED", result_template_id: "template-1" }));
  });

  it("reloads the exact proposal after a version conflict and explains recovery", async () => {
    vi.mocked(acceptFormProposal).mockRejectedValueOnce(new ApiError(409, "proposal changed", "form_proposal_conflict"));
    const latest = { ...proposal, version: 3, field_changes: proposal.field_changes.slice(0, 1) };
    vi.mocked(loadFormProposal).mockResolvedValueOnce(latest);
    render(<FormProposalReview proposal={proposal} onProposalChange={() => undefined}/>);
    fireEvent.click(screen.getByRole("button", { name: "Create draft from selected fields" }));
    expect((await screen.findByRole("alert")).textContent).toMatch(/changed while you were reviewing/i);
    await waitFor(() => expect(loadFormProposal).toHaveBeenCalledWith("proposal-1"));
    await waitFor(() => expect(screen.getByRole("alert").textContent).toContain("Latest version loaded. Review the selected fields before creating a draft."));
  });

  it.each([
    ["form_proposal_source_changed", "The source document changed. Reload the import and create a new proposal."],
    ["form_proposal_state_conflict", "This proposal cannot create a draft in its current state. Reload the import."],
  ])("preserves the recovery for %s without claiming a version changed", async (code, message) => {
    vi.mocked(acceptFormProposal).mockRejectedValueOnce(new ApiError(409, message, code));
    render(<FormProposalReview proposal={proposal} onProposalChange={() => undefined}/>);
    fireEvent.click(screen.getByRole("button", { name: "Create draft from selected fields" }));
    expect((await screen.findByRole("alert")).textContent).toBe(message);
    expect(loadFormProposal).not.toHaveBeenCalled();
    expect((screen.getByRole("checkbox", { name: "Include Registered name" }) as HTMLInputElement).checked).toBe(true);
  });

  it("does not claim a conflicting proposal was reloaded while its refresh is pending or failed", async () => {
    vi.mocked(acceptFormProposal).mockRejectedValueOnce(new ApiError(409, "proposal changed", "form_proposal_conflict"));
    let failRefresh!: (error: Error) => void;
    vi.mocked(loadFormProposal).mockReturnValueOnce(new Promise((_, reject) => { failRefresh = reject; }));
    const changed = vi.fn();
    render(<FormProposalReview proposal={proposal} onProposalChange={changed}/>);
    fireEvent.click(screen.getByRole("button", { name: "Create draft from selected fields" }));
    await waitFor(() => expect(loadFormProposal).toHaveBeenCalledWith("proposal-1"));
    expect(screen.getByRole("alert").textContent).toContain("Reload it before creating a draft.");
    expect(screen.getByRole("alert").textContent).not.toMatch(/latest version.*loaded/i);
    failRefresh(new Error("Failed to fetch"));
    await waitFor(() => expect(screen.getByRole("button", { name: "Create draft from selected fields" }).hasAttribute("disabled")).toBe(false));
    expect(changed).not.toHaveBeenCalled();
    expect(screen.getByRole("alert").textContent).toContain("Reload it before creating a draft.");
  });
});
