import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { CaptureField, CaptureRequest } from "../../types";
import { CaptureReview } from "./CaptureReview";

const baseline = { target_key: "VENDOR.IDENTITY.REGISTERED_ADDRESS", subject_type: "VENDOR_RELATIONSHIP", subject_id: "relationship-1", record_id: "vendor-1", record_version: 4, display_value: "12 Marina Road", source_label: "Validated vendor record", observed_or_confirmed_at: "2026-08-01T10:00:00Z" };
const sections = [
  { id: "identity", title: "Identity records" },
  { id: "documents", title: "Documents" },
];
const fields: CaptureField[] = [
  { id: "address", section_id: "identity", label: "Registered address", type: "long_text", required: true, collection_intent: "CONFIRM_OR_CORRECT", record_baseline: baseline },
  { id: "name", section_id: "identity", label: "Registered name", type: "short_text", required: true, collection_intent: "CONFIRM_OR_CORRECT", record_baseline: { ...baseline, target_key: "VENDOR.IDENTITY.LEGAL_NAME", display_value: "Acme Limited" } },
  { id: "certificate", section_id: "documents", label: "Operating certificate", type: "vendor_document", required: true, collection_intent: "REPLACE_HELD_DOCUMENT", collection_received: true, record_baseline: { ...baseline, target_key: "VENDOR.DOCUMENT.OPERATING_CERTIFICATE", display_value: "Certificate 2025" } },
  { id: "policy", section_id: "documents", label: "Security policy", type: "file", required: false },
  { id: "followup_owner", section_id: "identity", label: "Follow-up owner", type: "short_text", required: true },
];
const answers = { address: { text: "12 Marina Road" }, name: { text: "Acme Limited" }, policy: { artifact_ids: ["artifact-2"] } };
const request: CaptureRequest = {
  id: "request-1", title: "Vendor record refresh", purpose: "Confirm current records.", why_you: "You maintain these records.",
  status: "IN_PROGRESS", sensitivity: "CONFIDENTIAL", estimated_minutes: 10, deadline: "2099-09-30T00:00:00Z", known_facts: {},
  sections, fields, version: 2,
  previous_responses: { name: { value: "Acme Limited", previous_request_id: "request-0", previous_submission_id: "submission-0", previous_submitted_at: "2026-06-15T09:00:00Z" } },
};

describe("CaptureReview", () => {
  it("groups the response under the request's form sections with provenance before submission", () => {
    render(<CaptureReview request={request} fields={fields} answers={answers} attachments={{}} submitting={false} error={null} errorKind={null} onEdit={vi.fn()} onSubmit={vi.fn()}/>);

    expect(screen.getByRole("heading", { name: "Check your response" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Identity records", level: 3 })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Documents", level: 3 })).toBeTruthy();
    expect(screen.getByText("Registered address")).toBeTruthy();
    expect(screen.getByText("Registered name")).toBeTruthy();
    expect(screen.getByText("Operating certificate")).toBeTruthy();
    expect(screen.getByText("Security policy")).toBeTruthy();
    expect(screen.getByText("No upload needed.")).toBeTruthy();
    expect(screen.getByText("Previous response · submitted 15 Jun 2026")).toBeTruthy();
  });

  it("flags only missing answers in the needs-attention view and never the received document", () => {
    render(<CaptureReview request={request} fields={fields} answers={answers} attachments={{}} submitting={false} error={null} errorKind={null} onEdit={vi.fn()} onSubmit={vi.fn()}/>);

    expect(screen.getByText("3 of 5 fields answered")).toBeTruthy();
    expect(screen.getByText("1 field needs attention")).toBeTruthy();
    expect(screen.getByText("1 no answer")).toBeTruthy();
    expect(screen.queryByText("1 no evidence")).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Needs attention" }));
    expect(screen.getByText("Follow-up owner")).toBeTruthy();
    expect(screen.getByText("No answer submitted for this field.")).toBeTruthy();
    expect(screen.queryByText("Operating certificate")).toBeNull();
  });
});