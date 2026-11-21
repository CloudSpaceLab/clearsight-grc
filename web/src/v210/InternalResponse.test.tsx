import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { InternalResponse } from "./InternalResponse";

const client = vi.hoisted(() => ({ loadInternalWorkspace: vi.fn(), saveInternalWorkspace: vi.fn(), submitInternalWorkspace: vi.fn(), uploadInternalDocument: vi.fn() }));
vi.mock("./internalResponseApi", () => client);
const payload = { request: { id: "req-1", title: "Access review", status: "READY", fields: [{ id: "q1", label: "Access reviewed?", type: "yes_no", required: true }] }, workspace: { workspace: { id: "workspace-1", distribution_id: "distribution-1", status: "OPEN", version: 1 }, answers: {}, field_sequences: {}, presentation_mode: "AUTOMATIC" } };
beforeEach(() => { vi.clearAllMocks(); client.loadInternalWorkspace.mockResolvedValue(payload); });

it("saves and submits using the exact authenticated request and version", async () => {
  const saved = { ...payload.workspace, workspace: { ...payload.workspace.workspace, version: 2 }, answers: { q1: { text: "Yes" } }, field_sequences: { q1: 2 } };
  client.saveInternalWorkspace.mockResolvedValue(saved);
  client.submitInternalWorkspace.mockResolvedValue({ workspace: { ...saved.workspace, version: 3 }, revision: { id: "revision-1", submission_id: "submission-1", current: true }, submission: { request_id: "req-1", submission_id: "submission-1" } });
  render(<InternalResponse requestId="req-1"/>);
  fireEvent.change(await screen.findByLabelText("Access reviewed?"), { target: { value: "Yes" } });
  fireEvent.click(screen.getByRole("button", { name: "Save response" }));
  await screen.findByText("Response saved.");
  expect(client.saveInternalWorkspace).toHaveBeenCalledWith("req-1", expect.objectContaining({ expected_version: 1, edits: [{ field_id: "q1", base_sequence: 0, value: { text: "Yes" } }] }));
  fireEvent.click(screen.getByRole("button", { name: "Submit response" }));
  expect(await screen.findByRole("heading", { name: "Response submitted" })).toBeTruthy();
  expect(client.submitInternalWorkspace).toHaveBeenCalledWith("req-1", { expected_version: 2, attestation_field_ids: [] });
});

it("does not display a response returned for a different request", async () => {
  client.loadInternalWorkspace.mockResolvedValue({ ...payload, request: { ...payload.request, id: "other" } });
  render(<InternalResponse requestId="req-1"/>);
  expect(await screen.findByRole("heading", { name: "Form unavailable" })).toBeTruthy();
  expect(screen.queryByText("Access reviewed?")).toBeNull();
});

it("retains local answers on a conflict until the user compares them", async () => {
  const { ApiError } = await import("../http");
  client.saveInternalWorkspace.mockRejectedValue(new ApiError(409, "Changed"));
  render(<InternalResponse requestId="req-1"/>);
  fireEvent.change(await screen.findByLabelText("Access reviewed?"), { target: { value: "Yes" } });
  fireEvent.click(screen.getByRole("button", { name: "Save response" }));
  expect(await screen.findByText("The response changed. Compare answers before saving.")).toBeTruthy();
  expect((screen.getByLabelText("Access reviewed?") as HTMLSelectElement).value).toBe("Yes");
  expect(screen.queryByRole("button", { name: "Submit response" })).toBeNull();
});

it("uploads and saves a required document for the exact request", async () => {
  client.loadInternalWorkspace.mockResolvedValue({ ...payload, request: { ...payload.request, fields: [{ id: "file", label: "Audit report", type: "file", required: true }] } });
  client.uploadInternalDocument.mockResolvedValue({ id: "artifact-1", request_id: "req-1", file_name: "audit.pdf", status: "AVAILABLE" });
  render(<InternalResponse requestId="req-1"/>);
  const file = new File(["document"], "audit.pdf", { type: "application/pdf" });
  fireEvent.change(await screen.findByLabelText("Audit report"), { target: { files: [file] } });
  expect(await screen.findByText("audit.pdf · Scanned at upload")).toBeTruthy();
  expect(client.uploadInternalDocument).toHaveBeenCalledWith("req-1", file, "file");
  fireEvent.click(screen.getByRole("button", { name: "Save response" }));
  expect(client.saveInternalWorkspace).toHaveBeenCalledWith("req-1", expect.objectContaining({ edits: [{ field_id: "file", base_sequence: 0, value: { artifact_ids: ["artifact-1"] } }] }));
});

it("does not attach an upload returned for another request", async () => {
  client.loadInternalWorkspace.mockResolvedValue({ ...payload, request: { ...payload.request, fields: [{ id: "file", label: "Audit report", type: "file", required: true }] } });
  client.uploadInternalDocument.mockResolvedValue({ id: "artifact-other", request_id: "other", status: "AVAILABLE" });
  render(<InternalResponse requestId="req-1"/>);
  fireEvent.change(await screen.findByLabelText("Audit report"), { target: { files: [new File(["document"], "audit.pdf")] } });
  expect(await screen.findByText("Document upload could not be confirmed. Contact the requester.")).toBeTruthy();
  expect((screen.getByLabelText("Audit report") as HTMLInputElement).disabled).toBe(true);
  expect(screen.queryByRole("button", { name: "Save response" })).toBeNull();
});

it("withholds another submit after an unknown outcome", async () => {
  client.loadInternalWorkspace.mockResolvedValue({ ...payload, workspace: { ...payload.workspace, answers: { q1: { text: "Yes" } } } });
  client.submitInternalWorkspace.mockRejectedValue(new Error("connection lost"));
  render(<InternalResponse requestId="req-1"/>);
  fireEvent.click(await screen.findByRole("button", { name: "Submit response" }));
  expect(await screen.findByRole("heading", { name: "Submission status unknown" })).toBeTruthy();
  await waitFor(() => expect(screen.queryByRole("button", { name: "Submit response" })).toBeNull());
});
