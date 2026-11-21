import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { VendorCollection } from "./VendorCollection";

const collection = vi.hoisted(() => ({ loadVendorCollection: vi.fn(), reconcileVendorEvidence: vi.fn() }));
const documents = vi.hoisted(() => ({ loadDocuments: vi.fn() }));
vi.mock("../vendorCollectionApi", () => collection);
vi.mock("../submittedDocumentApi", () => documents);
const data = { assessment_id: "assessment", assessment_version: 4, request_id: "request", request_version: 2, can_reconcile: true, fields: [{ field_id: "field", label: "Audit report", type: "vendor_document", required: true, collection_state: "MISSING", bank_review_state: "PENDING" }] };
const file = { id: "occurrence", relationship_id: "vendor", submission_channel: "MAGIC_LINK", submission_id: "submission", field_id: "source-field", artifact_id: "artifact", artifact_status: "AVAILABLE", file_name: "Audit report.pdf", current: true, submitted_at: "2026-11-20" };
beforeEach(() => { vi.clearAllMocks(); collection.loadVendorCollection.mockResolvedValue(data); documents.loadDocuments.mockResolvedValue({ items: [file] }); });

it("links a scoped document to the exact request and keeps review separate", async () => {
  collection.reconcileVendorEvidence.mockResolvedValue({ collection: { ...data, assessment_version: 5, request_version: 3, fields: [{ ...data.fields[0], collection_state: "REUSED" }] }, receipt: { id: "receipt", source: file } });
  const reload = vi.fn(); render(<VendorCollection assessmentId="assessment" relationshipId="vendor" onChanged={reload} onUncertain={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Select document for Audit report" }));
  expect(await screen.findByText("Audit report.pdf")).toBeTruthy();
  fireEvent.click(screen.getByLabelText(/Audit report.pdf/));
  fireEvent.change(screen.getByLabelText("Reason"), { target: { value: "The report covers the same service." } });
  fireEvent.click(screen.getByRole("button", { name: "Link document" }));
  expect(await screen.findByText("Document linked. Review is still required.")).toBeTruthy();
  expect(collection.reconcileVendorEvidence).toHaveBeenCalledWith("assessment", "field", { expected_version: 4, request_id: "request", expected_request_version: 2, source_submission_id: "submission", source_field_id: "source-field", source_artifact_id: "artifact", rationale: "The report covers the same service." });
  expect(documents.loadDocuments).toHaveBeenCalledWith(expect.objectContaining({ relationship_id: "vendor", current_only: true }), expect.any(AbortSignal));
  expect(reload).toHaveBeenCalled();
});

it("does not offer linking when the server does not authorize it", async () => {
  collection.loadVendorCollection.mockResolvedValue({ ...data, can_reconcile: false });
  render(<VendorCollection assessmentId="assessment" relationshipId="vendor" onChanged={() => {}} onUncertain={() => {}}/>);
  expect(await screen.findByText("Audit report")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Select document for Audit report" })).toBeNull();
});

it("does not retry an unconfirmed link", async () => {
  collection.reconcileVendorEvidence.mockRejectedValue(new Error("lost connection"));
  render(<VendorCollection assessmentId="assessment" relationshipId="vendor" onChanged={() => {}} onUncertain={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Select document for Audit report" }));
  fireEvent.click(await screen.findByLabelText(/Audit report.pdf/));
  fireEvent.change(screen.getByLabelText("Reason"), { target: { value: "The report covers this service." } });
  fireEvent.click(screen.getByRole("button", { name: "Link document" }));
  expect(await screen.findByText("Document link status unknown. Reload the assessment before trying again.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Link document" })).toBeNull();
  await waitFor(() => expect(collection.reconcileVendorEvidence).toHaveBeenCalledTimes(1));
});

it("keeps the selected document and reason fixed while the link is being recorded", async () => {
  let finish!: (value: unknown) => void;
  collection.reconcileVendorEvidence.mockReturnValue(new Promise((resolve) => { finish = resolve; }));
  render(<VendorCollection assessmentId="assessment" relationshipId="vendor" onChanged={() => {}} onUncertain={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Select document for Audit report" }));
  const choice = await screen.findByLabelText(/Audit report.pdf/) as HTMLInputElement;
  fireEvent.click(choice);
  const reason = screen.getByLabelText("Reason") as HTMLTextAreaElement;
  fireEvent.change(reason, { target: { value: "The report covers this service." } });
  fireEvent.click(screen.getByRole("button", { name: "Link document" }));
  expect(choice.disabled).toBe(true);
  expect(reason.disabled).toBe(true);
  finish({ collection: { ...data, assessment_version: 5, request_version: 3 }, receipt: { id: "receipt", source: file } });
  expect(await screen.findByText("Document linked. Review is still required.")).toBeTruthy();
});

it("loads each vendor document page only once when the button is activated twice", async () => {
  let finish!: (value: unknown) => void;
  documents.loadDocuments.mockResolvedValueOnce({ items: [file], next_cursor: "page-2" }).mockReturnValueOnce(new Promise((resolve) => { finish = resolve; }));
  render(<VendorCollection assessmentId="assessment" relationshipId="vendor" onChanged={() => {}} onUncertain={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Select document for Audit report" }));
  const more = await screen.findByRole("button", { name: "Load more documents" });
  act(() => { fireEvent.click(more); fireEvent.click(more); });
  expect(documents.loadDocuments).toHaveBeenCalledTimes(2);
  finish({ items: [{ ...file, id: "second", file_name: "Second.pdf" }] });
  expect(await screen.findByText("Second.pdf")).toBeTruthy();
});

it("discards a late document page after choosing the document field again", async () => {
  let finishOld!: (value: unknown) => void;
  let finishNew!: (value: unknown) => void;
  documents.loadDocuments.mockResolvedValueOnce({ items: [file], next_cursor: "old-page" })
    .mockReturnValueOnce(new Promise((resolve) => { finishOld = resolve; }))
    .mockResolvedValueOnce({ items: [file], next_cursor: "new-page" })
    .mockReturnValueOnce(new Promise((resolve) => { finishNew = resolve; }));
  render(<VendorCollection assessmentId="assessment" relationshipId="vendor" onChanged={() => {}} onUncertain={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Select document for Audit report" }));
  fireEvent.click(await screen.findByRole("button", { name: "Load more documents" }));
  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  fireEvent.click(screen.getByRole("button", { name: "Select document for Audit report" }));
  fireEvent.click(await screen.findByRole("button", { name: "Load more documents" }));
  await act(async () => finishOld({ items: [{ ...file, id: "old", file_name: "Old.pdf" }] }));
  expect(screen.queryByText("Old.pdf")).toBeNull();
  expect((screen.getByRole("button", { name: "Loading…" }) as HTMLButtonElement).disabled).toBe(true);
  await act(async () => finishNew({ items: [{ ...file, id: "new", file_name: "New.pdf" }] }));
  expect(await screen.findByText("New.pdf")).toBeTruthy();
});

it("does not confirm a linked document when the receipt names another response revision", async () => {
  const selectedFile = { ...file, response_revision_id: "source-revision-2" };
  documents.loadDocuments.mockResolvedValue({ items: [selectedFile] });
  collection.reconcileVendorEvidence.mockResolvedValue({ collection: { ...data, assessment_version: 5, request_version: 3 }, receipt: { id: "receipt", source: { ...selectedFile, response_revision_id: "source-revision-1" } } });
  const changed = vi.fn(); const uncertain = vi.fn();
  render(<VendorCollection assessmentId="assessment" relationshipId="vendor" onChanged={changed} onUncertain={uncertain}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Select document for Audit report" }));
  fireEvent.click(await screen.findByLabelText(/Audit report.pdf/));
  fireEvent.change(screen.getByLabelText("Reason"), { target: { value: "The report covers this service." } });
  fireEvent.click(screen.getByRole("button", { name: "Link document" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Document link status unknown");
  expect(uncertain).toHaveBeenCalledOnce();
  expect(changed).not.toHaveBeenCalled();
});

it("confirms a link only when the selected response revision is returned", async () => {
  const selectedFile = { ...file, response_revision_id: "source-revision-2" };
  documents.loadDocuments.mockResolvedValue({ items: [selectedFile] });
  collection.reconcileVendorEvidence.mockResolvedValue({ collection: { ...data, assessment_version: 5, request_version: 3 }, receipt: { id: "receipt", source: selectedFile } });
  const changed = vi.fn();
  render(<VendorCollection assessmentId="assessment" relationshipId="vendor" onChanged={changed} onUncertain={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Select document for Audit report" }));
  fireEvent.click(await screen.findByLabelText(/Audit report.pdf/));
  fireEvent.change(screen.getByLabelText("Reason"), { target: { value: "The report covers this service." } });
  fireEvent.click(screen.getByRole("button", { name: "Link document" }));
  expect(await screen.findByText("Document linked. Review is still required.")).toBeTruthy();
  expect(collection.reconcileVendorEvidence).toHaveBeenCalledWith("assessment", "field", expect.objectContaining({ source_response_revision_id: "source-revision-2" }));
  expect(changed).toHaveBeenCalledOnce();
});
