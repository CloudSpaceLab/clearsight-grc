import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { VendorReview } from "../v210/VendorReview";

const assessment = vi.hoisted(() => ({ loadCurrentVendorAssessment: vi.fn(), loadVendorAssessment: vi.fn(), startVendorAssessment: vi.fn(), startVendorAssessmentReview: vi.fn(), completeVendorAssessment: vi.fn(), sendVendorAssessmentRequest: vi.fn(), retryVendorAssessmentSetup: vi.fn(), reissueVendorAssessmentRequest: vi.fn(), requestVendorAssessmentClarification: vi.fn() }));
const api = vi.hoisted(() => ({ loadVendorActivation: vi.fn(), activateVendorRelationship: vi.fn() }));
const authority = vi.hoisted(() => ({ checkVendorAuthority: vi.fn() }));
vi.mock("../vendorAssessmentApi", () => assessment);
vi.mock("../vendorApi", () => api);
vi.mock("../formsApi", () => ({ loadFormTemplatePage: vi.fn() }));
vi.mock("../v210/vendorAuthority", () => authority);
vi.mock("../v210/VendorDocuments", () => ({ VendorDocuments: () => null }));
vi.mock("../v210/VendorFindings", () => ({ VendorFindings: () => null }));
vi.mock("../v210/VendorCollection", () => ({ VendorCollection: () => null }));

it("refreshes activation requirements after assessment progress without changing the relationship", async () => {
  const relationship = { id: "rel-1", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "vendor", business_owner_principal_id: "owner", version: 4, status: "PROPOSED", service_name: "Hosting", criticality: "IMPORTANT", privacy_role: "PROCESSOR" };
  const current = { id: "assessment", relationship_id: relationship.id, version: 7, status: "SUBMITTED", review_kind: "ONBOARDING" };
  const progressed = { ...current, version: 8, status: "UNDER_REVIEW" };
  const policy = { id: "policy", policy_number: 3, version: 5, status: "ACTIVE", effective_from: "2026-01-01T00:00:00Z" };
  const pending = { eligible: false, relationship, policy, gates: [{ code: "CURRENT_ASSESSMENT", satisfied: false, explanation: "Assessment review is pending." }] };
  authority.checkVendorAuthority.mockResolvedValue(true);
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: current });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: current, answers: [], documents: [], matters: [], coverage: {} });
  assessment.startVendorAssessmentReview.mockResolvedValue(progressed);
  api.loadVendorActivation.mockResolvedValue(pending);
  render(<VendorReview relationship={{ vendor: { legal_name: "Atlas Ltd" }, relationship } as any} onBack={() => {}}/>);
  expect(await screen.findByText("Assessment review is pending.")).toBeTruthy();
  assessment.loadCurrentVendorAssessment.mockResolvedValue({ assessment: progressed });
  assessment.loadVendorAssessment.mockResolvedValue({ assessment: progressed, answers: [], documents: [], matters: [], coverage: {} });
  api.loadVendorActivation.mockResolvedValue({ ...pending, eligible: true, gates: [{ code: "CURRENT_ASSESSMENT", satisfied: true, explanation: "Assessment review is complete." }] });
  fireEvent.click(await screen.findByRole("button", { name: "Start review" }));
  expect(await screen.findByText("Assessment review is complete.")).toBeTruthy();
  expect(await screen.findByRole("button", { name: "Activate relationship" })).toBeTruthy();
  expect(assessment.startVendorAssessmentReview).toHaveBeenCalledWith("assessment", { expected_version: 7 });
  expect(api.loadVendorActivation).toHaveBeenLastCalledWith("rel-1");
});
