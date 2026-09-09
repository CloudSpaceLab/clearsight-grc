import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { CaptureAnswers, CaptureField, CaptureRequest } from "../types";
import { CapturePanel } from "./CapturePanel";
import { CaptureWorkspaceRecoveryProvider } from "./capture/CaptureWorkspaceRecoveryContext";
import { captureContract, keepVisibleAnswers, validateCaptureFields, visibleCaptureFields } from "./capture/contract";

const receivedDocument: CaptureField & { collection_received: boolean } = {
  id: "certificate", label: "Operating certificate", type: "vendor_document", required: true,
  constraints: { min_files: 1 }, collection_received: true,
};
const request: CaptureRequest = {
  id: "request-reuse", title: "Vendor evidence request", purpose: "Provide current service evidence.",
  why_you: "You maintain your company's service records.", status: "IN_PROGRESS", sensitivity: "CONFIDENTIAL",
  estimated_minutes: 2, deadline: "2099-09-30T00:00:00Z", known_facts: {}, version: 3,
  fields: [receivedDocument, { id: "contact", label: "Service contact", type: "short_text", required: true }],
};

describe("already received vendor documents in respondent capture", () => {
  it("shows receipt without offering an empty submission when every required document is held", () => {
    const onSubmit = vi.fn();
    render(<CapturePanel request={{ ...request, fields: [receivedDocument] }} external onSubmit={onSubmit}/>);
    expect(screen.getByRole("heading", { name: "Received" })).toBeTruthy();
    expect(screen.getByText("All required documents received. No response needed.")).toBeTruthy();
    expect(screen.getByText("Operating certificate")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Review and submit" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Submit evidence" })).toBeNull();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("retains actual optional answers when every required document is held", () => {
    render(<CapturePanel request={{ ...request, fields: [receivedDocument, { id: "note", label: "Service changes", type: "short_text", required: false }] }} external workspacePersistence={{ key: "optional-answer", initialAnswers: { note: { text: "Contact changed" } }, initialPresentationMode: "CLASSIC", saveState: "saved_server", onChange: vi.fn(), onFlush: vi.fn().mockResolvedValue(true), onRetry: vi.fn() }}/>);
    expect(screen.queryByRole("heading", { name: "Received" })).toBeNull();
    expect((screen.getByRole("textbox", { name: "Service changes" }) as HTMLInputElement).value).toBe("Contact changed");
    expect(screen.getByRole("button", { name: "Review and submit" })).toBeTruthy();
  });

  it("does not claim all documents received while another required item's condition is unresolved", () => {
    render(<CapturePanel request={{ ...request, fields: [receivedDocument, { id: "regulated", label: "Regulated service", type: "yes_no", required: false }, { id: "license", label: "Service licence", type: "vendor_document", required: true, condition: { field_id: "regulated", operator: "EQUALS", values: ["Yes"] } }] }} external/>);
    expect(screen.queryByRole("heading", { name: "Received" })).toBeNull();
    expect(screen.getByRole("radio", { name: "Yes" })).toBeTruthy();
  });

  it("requires only outstanding answers without inventing a document answer", () => {
    const answers: CaptureAnswers = {};
    expect(validateCaptureFields(request.fields, answers)).toEqual([{ fieldID: "contact", message: "Service contact is required" }]);
    expect(answers).toEqual({});
    expect(keepVisibleAnswers(captureContract(request), { contact: { text: "Ada Okoro" } })).toEqual({ contact: { text: "Ada Okoro" } });
  });

  it("does not treat receipt flags on other field types as submitted answers", () => {
    for (const type of ["file", "photo", "signature", "short_text", "attestation"]) {
      expect(validateCaptureFields([{ ...receivedDocument, type }], {})).toEqual([{ fieldID: "certificate", message: "Operating certificate is required" }]);
    }
  });

  it("still requires an upload when the server no longer reports the document received", () => {
    expect(validateCaptureFields([{ ...receivedDocument, collection_received: false }], {})).toEqual([{ fieldID: "certificate", message: "Operating certificate is required" }]);
  });

  it.each(["CLASSIC", "WIZARD"] as const)("shows receipt separately from new responses and submits only answers in %s", async (mode) => {
    const onSubmit = vi.fn().mockResolvedValue({ submitted_at: "2026-09-08T12:00:00Z" });
    const onUploadArtifact = vi.fn();
    render(<CapturePanel request={{ ...request, presentation: { default_mode: mode, allow_mode_switch: false } }} external onSubmit={onSubmit} onUploadArtifact={onUploadArtifact}/>);

    expect(within(screen.getByRole("group", { name: "Operating certificate" })).getByText("Document received")).toBeTruthy();
    expect(screen.getByText("No upload needed.")).toBeTruthy();
    expect(document.querySelector('input[type="file"]')).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
    expect(screen.getByRole("alert").textContent).toContain("Service contact is required");
    expect(screen.getByRole("alert").textContent).not.toContain("Operating certificate is required");
    fireEvent.change(screen.getByRole("textbox", { name: /Service contact/ }), { target: { value: "Ada Okoro" } });
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
    const held = screen.getByRole("region", { name: "Received documents" });
    expect(within(held).getByText("Operating certificate")).toBeTruthy();
    expect(within(held).queryByText("Not provided")).toBeNull();
    expect(screen.queryByRole("region", { name: "New files and documents" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Submit evidence" }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ id: request.id }), { contact: { text: "Ada Okoro" } }));
    expect(onUploadArtifact).not.toHaveBeenCalled();
  });

  it("does not request file reselection or manufacture artifact bytes during draft recovery", async () => {
    const onChange = vi.fn();
    const onFlush = vi.fn().mockResolvedValue(true);
    const onSubmit = vi.fn().mockResolvedValue({ submitted_at: "2026-09-08T12:00:00Z" });
    render(<CaptureWorkspaceRecoveryProvider value={{ initialPage: 0, filesToReselect: ["certificate"], onPageChange: vi.fn() }}>
      <CapturePanel request={request} external onSubmit={onSubmit} workspacePersistence={{ key: "recovered-request", initialAnswers: { contact: { text: "Ada Okoro" } }, initialPresentationMode: "CLASSIC", saveState: "saved_server", onChange, onFlush, onRetry: vi.fn() }}/>
    </CaptureWorkspaceRecoveryProvider>);
    expect(screen.queryByText("Reselect file to upload")).toBeNull();
    fireEvent.change(screen.getByRole("textbox", { name: /Service contact/ }), { target: { value: "Ada Okoro, service team" } });
    await waitFor(() => expect(onChange).toHaveBeenCalledWith({ contact: { text: "Ada Okoro, service team" } }, "CLASSIC"));
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
    fireEvent.click(screen.getByRole("button", { name: "Submit evidence" }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledWith(request, { contact: { text: "Ada Okoro, service team" } }));
    for (const [answers] of onChange.mock.calls) expect(answers).not.toHaveProperty("certificate");
  });

  it("keeps conditional visibility separate from receipt and leaves unanswered controllers required", () => {
    const conditionalRequest: CaptureRequest = { ...request, fields: [
      { id: "regulated", label: "Regulated service", type: "yes_no", required: true },
      { ...receivedDocument, condition: { field_id: "regulated", operator: "EQUALS", values: ["Yes"] } },
    ] };
    const contract = captureContract(conditionalRequest);
    expect(validateCaptureFields(visibleCaptureFields(contract, {}), {})).toEqual([{ fieldID: "regulated", message: "Regulated service is required" }]);
    expect(visibleCaptureFields(contract, { regulated: { text: "No" } }).map((field) => field.id)).toEqual(["regulated"]);
    expect(validateCaptureFields(visibleCaptureFields(contract, { regulated: { text: "Yes" } }), { regulated: { text: "Yes" } })).toEqual([]);
  });

  it("lets the respondent clear an unfinished document draft after receipt without requiring another upload", async () => {
    const onSubmit = vi.fn().mockResolvedValue({ submitted_at: "2026-09-08T12:00:00Z" });
    const draft: CaptureAnswers = { contact: { text: "Ada Okoro" }, certificate: { document: { artifact_id: "", document_type: "Operating certificate" } } };
    render(<CapturePanel request={request} external onSubmit={onSubmit} workspacePersistence={{ key: "partial-document", initialAnswers: draft, initialPresentationMode: "CLASSIC", saveState: "saved_server", onChange: vi.fn(), onFlush: vi.fn().mockResolvedValue(true), onRetry: vi.fn() }}/>);
    expect(screen.getByText("Document received")).toBeTruthy();
    expect((screen.getByRole("textbox", { name: "Document type" }) as HTMLInputElement).value).toBe("Operating certificate");
    expect(screen.getByText("Document draft retained. Complete it to send another document, or remove it to use the received document.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Remove document draft" }));
    expect(document.querySelector('input[type="file"]')).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
    fireEvent.click(screen.getByRole("button", { name: "Submit evidence" }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledWith(request, { contact: { text: "Ada Okoro" } }));
  });

  it("preserves an existing respondent document answer and identifies it during review", () => {
    const draft: CaptureAnswers = { contact: { text: "Ada Okoro" }, certificate: { document: { artifact_id: "respondent-artifact", document_type: "Renewed operating certificate", reference: "2026-001" } } };
    render(<CapturePanel request={request} external workspacePersistence={{ key: "complete-document", initialAnswers: draft, initialPresentationMode: "CLASSIC", saveState: "saved_server", onChange: vi.fn(), onFlush: vi.fn().mockResolvedValue(true), onRetry: vi.fn() }}/>);
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
    expect(within(screen.getByRole("region", { name: "New files and documents" })).getByText("Renewed operating certificate · 2026-001")).toBeTruthy();
    expect(keepVisibleAnswers(captureContract(request), draft)).toEqual(draft);
  });
});
