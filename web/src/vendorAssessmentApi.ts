import { requestJSON } from "./http";
import type {
  CompleteVendorAssessmentInput,
  CurrentVendorAssessment,
  CreateVendorAssessmentDeficiencyInput,
  ReviewVendorAssessmentDocumentInput,
  ReissueVendorAssessmentRequestInput,
  RetryVendorAssessmentSetupInput,
  SendVendorAssessmentRequestInput,
  StartVendorAssessmentInput,
  StartVendorAssessmentReviewInput,
  VendorAssessment,
  VendorAssessmentClarificationInput,
  VendorAssessmentFinding,
  VendorAssessmentReviewView,
  VendorAssessmentSendOutcome,
  VendorAssessmentSetupRetryOutcome,
} from "./vendorAssessmentTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export async function loadCurrentVendorAssessment(relationshipID: string): Promise<CurrentVendorAssessment> {
  const response = await requestJSON<VendorAssessment | CurrentVendorAssessment>(apiBase, `/api/v1/vendors/${encodeURIComponent(relationshipID)}/assessments/current`);
  if ("assessment" in response) return response;
  return { assessment: response, setup: undefined };
}

export function loadVendorAssessment(assessmentID: string): Promise<VendorAssessmentReviewView> {
  return requestJSON<VendorAssessmentReviewView>(apiBase, `/api/v1/vendor-assessments/${encodeURIComponent(assessmentID)}`);
}

export function startVendorAssessment(relationshipID: string, input: StartVendorAssessmentInput): Promise<VendorAssessment> {
  return requestJSON<VendorAssessment>(apiBase, `/api/v1/vendors/${encodeURIComponent(relationshipID)}/assessments`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function sendVendorAssessmentRequest(assessmentID: string, input: SendVendorAssessmentRequestInput): Promise<VendorAssessmentSendOutcome> {
  return requestJSON<VendorAssessmentSendOutcome>(apiBase, `/api/v1/vendor-assessments/${encodeURIComponent(assessmentID)}/send-request`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function reissueVendorAssessmentRequest(assessmentID: string, input: ReissueVendorAssessmentRequestInput): Promise<VendorAssessmentSendOutcome> {
  return requestJSON<VendorAssessmentSendOutcome>(apiBase, `/api/v1/vendor-assessments/${encodeURIComponent(assessmentID)}/reissue-request`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function retryVendorAssessmentSetup(assessmentID: string, input: RetryVendorAssessmentSetupInput): Promise<VendorAssessmentSetupRetryOutcome> {
  return requestJSON<VendorAssessmentSetupRetryOutcome>(apiBase, `/api/v1/vendor-assessments/${encodeURIComponent(assessmentID)}/setup/retry`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function startVendorAssessmentReview(assessmentID: string, input: StartVendorAssessmentReviewInput): Promise<VendorAssessment> {
  return requestJSON<VendorAssessment>(apiBase, `/api/v1/vendor-assessments/${encodeURIComponent(assessmentID)}/review/start`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function requestVendorAssessmentClarification(assessmentID: string, input: VendorAssessmentClarificationInput): Promise<VendorAssessment> {
  return requestJSON<VendorAssessment>(apiBase, `/api/v1/vendor-assessments/${encodeURIComponent(assessmentID)}/clarifications`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function reviewVendorAssessmentDocument(assessmentID: string, artifactID: string, input: ReviewVendorAssessmentDocumentInput): Promise<VendorAssessmentReviewView> {
  return requestJSON<VendorAssessmentReviewView>(apiBase, `/api/v1/vendor-assessments/${encodeURIComponent(assessmentID)}/documents/${encodeURIComponent(artifactID)}/validate`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function createVendorAssessmentDeficiency(assessmentID: string, input: CreateVendorAssessmentDeficiencyInput): Promise<VendorAssessmentFinding> {
  return requestJSON<VendorAssessmentFinding>(apiBase, `/api/v1/vendor-assessments/${encodeURIComponent(assessmentID)}/deficiencies`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function completeVendorAssessment(assessmentID: string, input: CompleteVendorAssessmentInput): Promise<VendorAssessment> {
  return requestJSON<VendorAssessment>(apiBase, `/api/v1/vendor-assessments/${encodeURIComponent(assessmentID)}/complete`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}
