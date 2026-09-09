import type { FormTemplateProposal, FindingAssessment } from "./formsTypes";
import type { DocumentExtractedElement } from "./documentTypes";

const assessments: FindingAssessment[] = [
  { id: "assessment-a", label: "Sample Harbor Services · Payments · 6 February 2025", sheet: "Findings", row_start: 2, row_end: 2, finding_count: 1 },
  { id: "assessment-b", label: "Sample Northstar Hosting · Hosting · 10 March 2025", sheet: "Findings", row_start: 3, row_end: 3, finding_count: 1 },
];
export const findingSourceElements: DocumentExtractedElement[] = assessments.map((item, index) => ({ kind: "TABLE", text: `${item.label}\nFinding: ${index ? "Recovery test is overdue" : "Access review is overdue"}\nRecommendation: Supply the current review report.\nResponsibility: ${index ? "Service owner" : "Security owner"}\nHistorical deadline: 30 April 2025`, anchor: { sheet: item.sheet, row_start: item.row_start, row_end: item.row_end } }));

export function findingFollowUpFixture(assessmentID = ""): FormTemplateProposal {
  const assessment = assessments.find((item) => item.id === assessmentID);
  const fieldDefinitions = assessment
    ? [{ label: "Response to finding", type: "long_text" as const, required: true }, { label: "Action or explanation", type: "long_text" as const, required: true }, { label: "Remediation owner", type: "short_text" as const, required: true }, { label: "Proposed completion date", type: "date" as const, required: false }, { label: "Supporting evidence", type: "file" as const, required: false }]
    : assessments.map((item) => ({ label: `Update on finding: ${item.label}`, type: "long_text" as const, required: false }));
  const fields = fieldDefinitions.map((field, index) => ({ ...field, id: `field-${assessmentID || "general"}-${index}`, section_id: "findings", description: "Historical source record. Confirm the current position before requesting a response.", ...(field.type === "file" ? { accepted_formats: ["pdf", "docx", "xlsx"] } : {}) }));
  return {
    id: `sample-${assessmentID || "general"}`, source_kind: "DOCUMENT", source_document_id: "sample-register", source_document_version: 1, source_sha256: "a".repeat(64), finding_assessment_id: assessmentID || undefined,
    status: "REVIEW_REQUIRED", proposed_contract: { scoring_mode: "NONE", presentation: { default_mode: "CLASSIC", allow_mode_switch: true }, sections: [{ id: "findings", title: "Historical findings", help: assessment ? findingSourceElements[assessments.indexOf(assessment)]?.text : undefined }], fields },
    field_changes: fields.map((field, index) => ({ id: `change-${field.id}`, kind: "ADD_FIELD", field, anchor: findingSourceElements[assessment ? assessments.indexOf(assessment) : index]!.anchor, confidence: 0.8 })),
    unresolved_items: [{ code: "HISTORICAL_FINDING_REVIEW", message: "Confirm the current finding status and recipient before approving this draft." }],
    provenance: { proposal_version: assessment ? "FINDING_FOLLOW_UP_V2" : "FORM_TEMPLATE_PROPOSAL_V2", source_document_id: "sample-register", source_sha256: "a".repeat(64), source_version: 1, extraction_status: "EXTRACTED", finding_assessments: assessment ? [assessment] : assessments },
    created_by: "sample-author", created_at: "2026-09-09T10:00:00Z", updated_at: "2026-09-09T10:00:01Z", version: 2,
  };
}
