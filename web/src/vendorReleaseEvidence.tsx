import { useState } from "react";
import { FormProposalReview } from "./components/forms/FormProposalReview";
import { CaptureForm } from "./components/capture/CaptureForm";
import { vendorDueDiligenceStarterForm } from "./vendorDueDiligenceForm";
import type { CaptureAnswers, CaptureFormContract } from "./types";
import type { FormTemplateProposal } from "./formsTypes";
import { Notice } from "./components/ui";
import "./forms-foundation.css";

export function VendorReleaseEvidencePage({ state }: { state: string }) {
  const [answers, setAnswers] = useState<CaptureAnswers>({ assurance_available: { text: state === "assurance-missing" ? "No" : "Yes" } });
  const [proposal, setProposal] = useState<FormTemplateProposal>(() => {
    const fields = Array.from({ length: 47 }, (_, index) => ({ id: `requirement_${index}`, section_id: "requirements", label: `Sample requirement ${index + 1}: Review service data access`, description: "Evidence: Access review record. Applies to services processing personal data. Review after access changes.", type: "long_text" as const, required: false }));
    return { id: "sample-spreadsheet-proposal", source_kind: "DOCUMENT", status: "REVIEW_REQUIRED", proposed_contract: { scoring_mode: "NONE", presentation: { default_mode: "CLASSIC", allow_mode_switch: true }, sections: [{ id: "requirements", title: "Service requirements" }], fields }, field_changes: fields.map((field, index) => ({ id: `change_${index}`, kind: "ADD_FIELD", field, anchor: { sheet: "Checklist", row_start: index + 2, row_end: index + 2 }, confidence: 0.8 })), unresolved_items: fields.map((_, index) => ({ code: "REQUIREMENT_SCOPE_REVIEW", field_change_id: `change_${index}`, message: "Confirm scope, required answers and evidence before requesting this item." })), provenance: { proposal_version: "FORM_TEMPLATE_PROPOSAL_V1", source_document_id: "sample-source", source_sha256: "a".repeat(64), source_version: 1, extraction_status: "EXTRACTED" }, created_by: "sample-author", created_at: "2026-09-09T10:00:00Z", updated_at: "2026-09-09T10:00:00Z", version: 1 };
  });
  const starter = vendorDueDiligenceStarterForm as CaptureFormContract;
  // Match the nullable lists returned for an unsectioned XLSX proposal.
  const reviewProposal: FormTemplateProposal = state === "spreadsheet-unsectioned" ? JSON.parse(JSON.stringify({
    ...proposal,
    proposed_contract: { ...proposal.proposed_contract, sections: null, fields: proposal.proposed_contract.fields.slice(0, 3).map((field) => ({ ...field, section_id: undefined })) },
    field_changes: proposal.field_changes.slice(0, 3),
    unresolved_items: null,
  })) : state === "spreadsheet-source-rows" ? {
    ...proposal,
    proposed_contract: { ...proposal.proposed_contract, fields: proposal.proposed_contract.fields.slice(0, 2) },
    field_changes: proposal.field_changes.slice(0, 2),
    unresolved_items: proposal.unresolved_items.slice(0, 2),
  } : proposal;
  const sourceElements = state === "spreadsheet-source-rows" ? [
    { kind: "TABLE" as const, text: "Sample requirement 1: Confirm access owners. Evidence: signed access register.", anchor: { sheet: "Checklist", row_start: 2, row_end: 2 } },
    { kind: "TABLE" as const, text: "Sample requirement 2: Review recovery testing. Evidence: recovery exercise report.", anchor: { sheet: "Checklist", row_start: 3, row_end: 3 } },
  ] : [];
  return <main style={{ maxWidth: 1400, margin: "0 auto", padding: 24 }}>
    <Notice tone="info">Sample data · Vendor workflow review. This page does not send requests.</Notice>
    {state.startsWith("spreadsheet") ? <FormProposalReview proposal={reviewProposal} sourceTitle="Sample service checklist.xlsx" sourceElements={sourceElements} onProposalChange={setProposal}/> : <CaptureForm contract={{ ...starter, sections: starter.sections.filter((section) => section.id === "controls"), fields: starter.fields.filter((field) => field.section_id === "controls") }} answers={answers} attachments={{}} mode="CLASSIC" external uploadingField={null} onAnswer={(id, value) => setAnswers((current) => ({ ...current, [id]: value }))} onUpload={() => undefined} onRemoveAttachment={() => undefined} onModeChange={() => undefined} onReview={() => undefined}/>}
  </main>;
}
