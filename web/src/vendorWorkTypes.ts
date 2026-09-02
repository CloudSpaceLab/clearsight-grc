import type { CapturePresentation, CapturePresentationMode } from "./types";
import type { VendorLinkTargetType } from "./vendorLinkTypes";
import type { VendorAssessmentDocument, VendorAssessmentReviewAnswer } from "./vendorAssessmentTypes";

export type VendorWorkState = "PREPARING" | "AWAITING_VENDOR" | "RESPONSE_RECEIVED" | "UNDER_REVIEW" | "CHANGES_REQUESTED" | "ACCEPTED" | "CANCELLED";
export type VendorWorkDeliveryState = "NOT_SENT" | "LINK_CREATED_EMAIL_NOT_SENT" | "DELIVERED" | "RETRY_REQUIRED";
export type VendorWorkRequestKind = "GENERAL" | "CERTIFICATION_REFRESH";

export type VendorWorkRequest = {
  id: string;
  tenant_id: string;
  legal_entity_id: string;
  relationship_id: string;
  relationship_link_id: string;
  target_type: VendorLinkTargetType;
  target_id: string;
  request_kind: VendorWorkRequestKind;
  purpose: string;
  instructions: string;
  owner_principal_id: string;
  reviewer_principal_id?: string;
  form_template_id: string;
  form_template_version: number;
  presentation: CapturePresentationMode;
  current_request_id?: string;
  current_invitation_id?: string;
  current_capture_sequence: number;
  submission_id?: string;
  state: VendorWorkState;
  delivery_state: VendorWorkDeliveryState;
  recovery?: string;
  review_rationale?: string;
  cancellation_reason?: string;
  due_at: string;
  version: number;
  created_at: string;
  updated_at: string;
  response_received_at?: string;
  review_started_at?: string;
  accepted_at?: string;
  cancelled_at?: string;
};

export type VendorWorkPage = { items: VendorWorkRequest[]; next_cursor?: string };
export type VendorWorkQuery = { relationship_id?: string; target_type?: VendorLinkTargetType; target_id?: string; cursor?: string; limit?: number };

export type PrepareVendorWorkInput = {
  relationship_link_id: string;
  request_kind: VendorWorkRequestKind;
  purpose: string;
  instructions: string;
  form_template_id: string;
  form_template_version: number;
  presentation: CapturePresentationMode;
  vendor_audience: string;
  due_at: string;
};

export type SendVendorWorkInput = { expected_version: number; vendor_audience: string; invitation_ttl_minutes: number };
export type VendorWorkSendOutcome = {
  work: VendorWorkRequest;
  invitation?: { invitation_id: string; expires_at: string };
  delivery?: { status: string };
  state: VendorWorkDeliveryState;
  recovery?: string;
  capture_url?: string;
};
export type RequestVendorWorkChangesInput = SendVendorWorkInput & { message: string; field_ids: string[]; due_at: string };

export type VendorWorkResponseView = {
  work: VendorWorkRequest;
  request: {
    request_id: string;
    status: string;
    deadline: string;
    form_template_id: string;
    form_template_version: number;
    presentation: CapturePresentation;
  };
  response: { submission_id: string; request_id: string; submitted_at: string };
  answers: VendorAssessmentReviewAnswer[];
  documents: VendorAssessmentDocument[];
};
