import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createAIFormProposal, createAIFormRevisionProposal } from "../../formsApi";
import { ApiError } from "../../http";
import { FormAIComposer } from "./FormAIComposer";
import type { DocumentImport } from "../../documentTypes";

vi.mock("../../formsApi", () => ({
  createAIFormProposal: vi.fn(),
  createAIFormRevisionProposal: vi.fn(),
}));

const generated = {
  id: "proposal-ai",
  source_kind: "AI" as const,
  status: "REVIEW_REQUIRED" as const,
  proposed_contract: { scoring_mode: "NONE" as const, presentation: { default_mode: "AUTOMATIC" as const, allow_mode_switch: true }, sections: [{ id: "general", title: "General" }], fields: [{ id: "risk", section_id: "general", label: "Describe the main service risk", type: "long_text" as const, required: false }] },
  field_changes: [{ id: "change-risk", kind: "ADD_FIELD" as const, field: { id: "risk", section_id: "general", label: "Describe the main service risk", type: "long_text" as const, required: false }, anchor: {}, confidence: .8 }],
  unresolved_items: [], provenance: { proposal_version: "FORM_AI_PROPOSAL_V1", source_document_id: "", source_sha256: "", source_version: 0, extraction_status: "NOT_APPLICABLE", ai: { workload_id: "form-authoring", model_alias: "approved", prompt_version: "FORM_AUTHORING_V1", snapshot_sha256: "b".repeat(64), validation_results: ["contract_normalized"] } },
  created_by: "author-1", created_at: "2026-08-29T08:00:00Z", updated_at: "2026-08-29T08:00:01Z", version: 2,
};

beforeEach(() => {
  vi.mocked(createAIFormProposal).mockReset();
  vi.mocked(createAIFormRevisionProposal).mockReset();
  vi.mocked(createAIFormProposal).mockResolvedValue(generated);
  vi.mocked(createAIFormRevisionProposal).mockResolvedValue({ ...generated, base_template_id: "template-1", base_template_version: 7 });
});

describe("FormAIComposer", () => {
  it("requires a concrete authoring objective and creates a reviewable proposal", async () => {
    const proposed = vi.fn();
    render(<FormAIComposer onProposal={proposed}/>);
    fireEvent.click(screen.getByRole("button", { name: "Generate field proposal" }));
    expect((await screen.findByRole("alert")).textContent).toMatch(/describe the form/i);
    fireEvent.change(screen.getByRole("textbox", { name: "What should this form collect or change?" }), { target: { value: "Collect current vendor ownership and operating evidence." } });
    fireEvent.click(screen.getByRole("button", { name: "Generate field proposal" }));
    await waitFor(() => expect(createAIFormProposal).toHaveBeenCalledWith({ objective: "Collect current vendor ownership and operating evidence." }));
    expect(proposed).toHaveBeenCalledWith(generated);
  });

  it("uses the exact base revision when revising a template", async () => {
    render(<FormAIComposer baseTemplate={{ id: "template-1", name: "Vendor review", version: 7 }} onProposal={() => undefined}/>);
    fireEvent.change(screen.getByRole("textbox", { name: "What should this form collect or change?" }), { target: { value: "Add certificate expiry and renewal questions." } });
    fireEvent.click(screen.getByRole("button", { name: "Propose changes to revision 7" }));
    await waitFor(() => expect(createAIFormRevisionProposal).toHaveBeenCalledWith("template-1", 7, { objective: "Add certificate expiry and renewal questions." }));
  });

  it("keeps manual authoring available when governed AI is unavailable", async () => {
    vi.mocked(createAIFormProposal).mockRejectedValueOnce(new ApiError(503, "Governed AI form authoring is not available.", "form_ai_unavailable"));
    render(<FormAIComposer onProposal={() => undefined}/>);
    fireEvent.change(screen.getByRole("textbox", { name: "What should this form collect or change?" }), { target: { value: "Collect vendor identity." } });
    fireEvent.click(screen.getByRole("button", { name: "Generate field proposal" }));
    expect((await screen.findByRole("alert")).textContent).toMatch(/manual form builder remains available/i);
  });

  it("omits an optional source when no exact passages are selected", async () => {
    const source = { id: "document-1", version: 5, elements: [{ ref: "p-1", kind: "PARAGRAPH", text: "Vendor ownership", anchor: { page: 1 } }] } as DocumentImport;
    render(<FormAIComposer sourceDocument={source} onProposal={() => undefined}/>);
    fireEvent.change(screen.getByRole("textbox", { name: "What should this form collect or change?" }), { target: { value: "Collect vendor ownership." } });
    fireEvent.click(screen.getByRole("button", { name: "Generate field proposal" }));
    await waitFor(() => expect(createAIFormProposal).toHaveBeenCalledWith({ objective: "Collect vendor ownership." }));
  });
});
