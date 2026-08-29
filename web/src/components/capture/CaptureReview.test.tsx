import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { CaptureField, CaptureRequest } from "../../types";
import { CaptureReview } from "./CaptureReview";

const baseline = { target_key: "VENDOR.IDENTITY.REGISTERED_ADDRESS", subject_type: "VENDOR_RELATIONSHIP", subject_id: "relationship-1", record_id: "vendor-1", record_version: 4, display_value: "12 Marina Road", source_label: "Validated vendor record", observed_or_confirmed_at: "2026-08-01T10:00:00Z" };
const fields: CaptureField[] = [
  { id: "address", label: "Registered address", type: "long_text", required: true, collection_intent: "CONFIRM_OR_CORRECT", record_baseline: baseline },
  { id: "name", label: "Registered name", type: "short_text", required: true, collection_intent: "CONFIRM_OR_CORRECT", record_baseline: { ...baseline, target_key: "VENDOR.IDENTITY.LEGAL_NAME", display_value: "Acme Limited" } },
  { id: "certificate", label: "Operating certificate", type: "vendor_document", required: true, collection_intent: "REPLACE_HELD_DOCUMENT", record_baseline: { ...baseline, target_key: "VENDOR.DOCUMENT.OPERATING_CERTIFICATE", display_value: "Certificate 2025" } },
  { id: "policy", label: "Security policy", type: "file", required: false },
];
const request: CaptureRequest = { id: "request-1", title: "Vendor record refresh", purpose: "Confirm current records.", why_you: "You maintain these records.", status: "IN_PROGRESS", sensitivity: "CONFIDENTIAL", estimated_minutes: 10, deadline: "2099-09-30T00:00:00Z", known_facts: {}, fields, version: 2 };

describe("CaptureReview", () => {
  it("groups confirmations, proposed updates and document changes before submission", () => {
    render(<CaptureReview request={request} fields={fields} answers={{ address: { text: "12 Marina Road" }, name: { text: "Acme Holdings Limited" }, certificate: { document: { artifact_id: "artifact-1", document_type: "OPERATING_CERTIFICATE" } }, policy: { artifact_ids: ["artifact-2"] } }} attachments={{}} submitting={false} error={null} errorKind={null} onEdit={vi.fn()} onSubmit={vi.fn()}/>);

    expect(within(screen.getByRole("region", { name: "Confirmed held information" })).getByText("Registered address")).toBeTruthy();
    expect(within(screen.getByRole("region", { name: "Proposed updates" })).getByText("Registered name")).toBeTruthy();
    expect(within(screen.getByRole("region", { name: "Replacement documents" })).getByText("Operating certificate")).toBeTruthy();
    expect(within(screen.getByRole("region", { name: "New files and documents" })).getByText("Security policy")).toBeTruthy();
  });
});
