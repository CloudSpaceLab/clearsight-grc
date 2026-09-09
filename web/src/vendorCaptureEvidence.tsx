import { useState } from "react";
import type { CaptureAnswers, CapturePresentationMode, CaptureRequest } from "./types";
import { CapturePanel } from "./components/CapturePanel";
import { CaptureWorkspaceRecoveryProvider } from "./components/capture/CaptureWorkspaceRecoveryContext";
import { CaptureReview } from "./components/capture/CaptureReview";
import { Notice } from "./components/ui";

export type VendorCaptureEvidenceState = "received" | "missing" | "conditional" | "recovered" | "review" | "all-held";

export function vendorCaptureEvidenceRequest(state: VendorCaptureEvidenceState, mode: CapturePresentationMode = "CLASSIC"): CaptureRequest {
  return {
    id: `sample-vendor-capture-${state}`, title: "Sample · Payment service evidence", purpose: "Confirm the service contact and provide the requested service evidence.",
    why_you: "You maintain your company's service records.", status: "IN_PROGRESS", sensitivity: "CONFIDENTIAL", estimated_minutes: 2,
    deadline: "2099-09-30T16:00:00Z", known_facts: { vendor: "Sample · Delta Payment Services", service: "Payment processing" }, version: 4,
    presentation: { default_mode: mode, allow_mode_switch: true }, sections: [{ id: "service", title: "Service evidence" }],
    fields: [
      ...(state === "all-held" ? [] : [{ id: "contact", section_id: "service", label: "Service contact", type: "short_text", required: true }]),
      ...(state === "conditional" ? [{ id: "regulated", section_id: "service", label: "Does this service require an operating certificate?", type: "yes_no", required: true }] : []),
      { id: "certificate", section_id: "service", label: "Operating certificate", type: "vendor_document", required: true, accepted_formats: ["application/pdf"], constraints: { min_files: 1 }, collection_received: state !== "missing", condition: state === "conditional" ? { field_id: "regulated", operator: "EQUALS", values: ["Yes"] } : undefined },
      ...(state === "all-held" ? [] : [{ id: "attest", section_id: "service", label: "Response confirmation", type: "attestation", required: true, attestation: "I confirm that I am authorized to provide this service response." }]),
    ],
  };
}

export function VendorCaptureEvidencePage({ state = "received", mode = "CLASSIC" }: { state?: VendorCaptureEvidenceState; mode?: CapturePresentationMode }) {
  const [reviewing, setReviewing] = useState(state === "review");
  const [submitted, setSubmitted] = useState(false);
  const request = vendorCaptureEvidenceRequest(state, mode);
  const initialAnswers: CaptureAnswers = state === "recovered"
    ? { contact: { text: "Ada Okoro, service operations" }, certificate: { document: { artifact_id: "", document_type: "Operating certificate" } } }
    : state === "review" ? { contact: { text: "Ada Okoro, service operations" }, attest: { text: "true" } } : {};
  return <main className="external-capture-shell">
    <header className="external-capture-brand"><div className="brand-mark" aria-label="ClearSight">C</div><div><strong>ClearSight</strong><span>Evidence response</span></div></header>
    <Notice tone="info">Sample data · Vendor evidence request, 8 September 2026. Responses and files in this example are not sent to a bank.</Notice>
    <section className="external-capture-work">
      <div className="external-session-hint">Sample invited respondent · Delta Payment Services</div>
      {submitted ? <Notice tone="info">Sample response submitted. Awaiting review.</Notice> : reviewing ? <CaptureReview request={request} fields={request.fields} answers={initialAnswers} attachments={{}} external submitting={false} error={null} errorKind={null} onEdit={() => setReviewing(false)} onSubmit={() => setSubmitted(true)}/>
        : <CaptureWorkspaceRecoveryProvider value={{ initialPage: 0, filesToReselect: state === "recovered" ? ["certificate"] : [], onPageChange: () => undefined }}>
          <CapturePanel request={request} external workspacePersistence={{ key: request.id, initialAnswers, initialPresentationMode: mode, saveState: "saved_server", onChange: () => undefined, onFlush: async () => true, onRetry: () => undefined }} onSubmit={async () => ({ submitted_at: "2026-09-08T12:00:00Z" })} onUploadArtifact={async () => { throw new Error("This sample request does not save uploaded files."); }}/>
        </CaptureWorkspaceRecoveryProvider>}
    </section>
  </main>;
}
