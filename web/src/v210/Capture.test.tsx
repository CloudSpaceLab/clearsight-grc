import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { Capture } from "./Capture";
import { StrictMode } from "react";
import { ApiError } from "../http";

const api = vi.hoisted(() => ({ startFormAccess: vi.fn(), redeemFormAccess: vi.fn(), sendFormAccessOTP: vi.fn(), verifyFormAccessOTP: vi.fn(), loadFormResponseWorkspace: vi.fn(), saveFormResponseWorkspace: vi.fn(), submitFormResponseWorkspace: vi.fn(), uploadCaptureSessionArtifact: vi.fn() }));
vi.mock("../captureApi", () => api);

const workspace = { workspace: { id: "w", distribution_id: "d", status: "OPEN", version: 3 }, answers: {}, field_sequences: {}, presentation_mode: "AUTOMATIC" };
const payload = { session: { id: "s", distribution_id: "d", request_id: "r", audience_hint: "a***@example.com", assurance: "LINK_POSSESSION" }, request: { id: "r", status: "READY", title: "Access review", purpose: "Review service access", fields: [{ id: "q", type: "yes_no", label: "Is access reviewed?", required: true }] }, workspace };
beforeEach(() => { vi.clearAllMocks(); sessionStorage.clear(); api.startFormAccess.mockResolvedValue({ policy: "DIRECT_MAGIC_LINK" }); api.redeemFormAccess.mockResolvedValue({ session_id: "s", session_token: "session-secret", request_id: "r", distribution_id: "d", assurance: "LINK_POSSESSION" }); api.loadFormResponseWorkspace.mockResolvedValue(payload); });

it("opens a direct invitation without persisting its credentials", async () => {
  render(<StrictMode><Capture invitationToken="invitation-secret"/></StrictMode>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  expect(await screen.findByRole("heading", { name: "Access review" })).toBeTruthy();
  expect(api.loadFormResponseWorkspace).toHaveBeenCalledWith("session-secret");
  expect(sessionStorage.length).toBe(0);
  expect(document.body.textContent).not.toContain("invitation-secret");
  expect(document.body.textContent).not.toContain("session-secret");
  expect(api.startFormAccess).toHaveBeenCalledTimes(1);
  expect(api.redeemFormAccess).toHaveBeenCalledTimes(1);
});

it("verifies a code for a masked recipient", async () => {
  api.startFormAccess.mockResolvedValue({ policy: "SHARED_LINK_EMAIL_OTP", recipients: [{ selector_id: "recipient", hint: "a***@example.com" }] });
  api.sendFormAccessOTP.mockResolvedValue({ challenge_id: "challenge", hint: "a***@example.com" });
  api.verifyFormAccessOTP.mockResolvedValue({ session_id: "s", session_token: "session-secret", request_id: "r", distribution_id: "d", assurance: "EMAIL_VERIFIED" });
  api.loadFormResponseWorkspace.mockResolvedValue({ ...payload, session: { ...payload.session, assurance: "EMAIL_VERIFIED" } });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.click(await screen.findByLabelText("a***@example.com"));
  fireEvent.click(screen.getByRole("button", { name: "Send code" }));
  fireEvent.change(await screen.findByLabelText("Verification code"), { target: { value: "123456" } });
  fireEvent.click(screen.getByRole("button", { name: "Verify code" }));
  expect(await screen.findByRole("heading", { name: "Access review" })).toBeTruthy();
  expect(api.verifyFormAccessOTP).toHaveBeenCalledWith("invitation-secret", "challenge", "123456");
});

it("uses only a selected masked recipient for shared-link verification", async () => {
  api.startFormAccess.mockResolvedValue({ policy: "SHARED_LINK_EMAIL_OTP", recipients: [{ selector_id: "first", hint: "a***@example.com" }, { selector_id: "second", hint: "b***@example.com" }] });
  api.sendFormAccessOTP.mockResolvedValue({ challenge_id: "challenge", hint: "b***@example.com" });
  api.verifyFormAccessOTP.mockResolvedValue({ session_id: "s", session_token: "session-secret", request_id: "r", distribution_id: "d", assurance: "EMAIL_VERIFIED" });
  api.loadFormResponseWorkspace.mockResolvedValue({ ...payload, session: { ...payload.session, assurance: "EMAIL_VERIFIED" } });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  expect(await screen.findByLabelText("a***@example.com")).toBeTruthy();
  expect(screen.queryByRole("textbox", { name: /email|phone/i })).toBeNull();
  fireEvent.click(screen.getByLabelText("b***@example.com"));
  fireEvent.click(screen.getByRole("button", { name: "Send code" }));
  expect((await screen.findByLabelText("Verification code")).getAttribute("autocomplete")).toBe("one-time-code");
  expect(api.sendFormAccessOTP).toHaveBeenCalledWith("invitation-secret", "second");
  fireEvent.change(screen.getByLabelText("Verification code"), { target: { value: "123456" } });
  fireEvent.click(screen.getByRole("button", { name: "Verify code" }));
  expect(await screen.findByRole("heading", { name: "Access review" })).toBeTruthy();
  expect(sessionStorage.length).toBe(0);
});

it("does not expose access errors when an invitation cannot be started", async () => {
  api.startFormAccess.mockRejectedValueOnce(new Error("internal provider diagnostic"));
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  expect(await screen.findByRole("heading", { name: "Request unavailable" })).toBeTruthy();
  expect(screen.getByText("Contact the requester for a new invitation.")).toBeTruthy();
  expect(screen.queryByText(/internal provider diagnostic/)).toBeNull();
  expect(api.redeemFormAccess).not.toHaveBeenCalled();
});

it("saves and submits an email-verified response with its current workspace version", async () => {
  api.startFormAccess.mockResolvedValue({ policy: "DIRECT_LINK_EMAIL_OTP", recipients: [{ selector_id: "recipient", hint: "a***@example.com" }] });
  api.sendFormAccessOTP.mockResolvedValue({ challenge_id: "challenge", hint: "a***@example.com" });
  api.verifyFormAccessOTP.mockResolvedValue({ session_id: "s", session_token: "session-secret", request_id: "r", distribution_id: "d", assurance: "EMAIL_VERIFIED" });
  api.loadFormResponseWorkspace.mockResolvedValue({ ...payload, session: { ...payload.session, assurance: "EMAIL_VERIFIED" } });
  api.saveFormResponseWorkspace.mockResolvedValue({ ...workspace, workspace: { ...workspace.workspace, version: 4 }, answers: { q: { text: "Yes" } }, field_sequences: { q: 1 } });
  api.submitFormResponseWorkspace.mockResolvedValue({ workspace: { ...workspace.workspace, version: 5 }, revision: { ID: "revision", SubmissionID: "submission", Current: true }, submission: { submission_id: "submission", request_id: "r", status: "SUBMITTED" } });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.change(await screen.findByLabelText("Verification code"), { target: { value: "123456" } });
  fireEvent.click(screen.getByRole("button", { name: "Verify code" }));
  fireEvent.change(await screen.findByLabelText("Is access reviewed?"), { target: { value: "Yes" } });
  fireEvent.click(screen.getByRole("button", { name: "Save response" }));
  await screen.findByText("Response saved.");
  fireEvent.click(screen.getByRole("button", { name: "Submit response" }));
  expect(await screen.findByRole("heading", { name: "Response submitted" })).toBeTruthy();
  expect(api.saveFormResponseWorkspace).toHaveBeenCalledWith("session-secret", expect.objectContaining({ expected_version: 3, edits: [{ field_id: "q", value: { text: "Yes" }, base_sequence: 0 }] }));
  expect(api.submitFormResponseWorkspace).toHaveBeenCalledWith("session-secret", { expected_version: 4, attestation_field_ids: [] });
  expect(sessionStorage.length).toBe(0);
});

it("does not open a workspace whose assurance differs from the verified invitation", async () => {
  api.startFormAccess.mockResolvedValue({ policy: "DIRECT_LINK_EMAIL_OTP", recipients: [{ selector_id: "recipient", hint: "a***@example.com" }] });
  api.sendFormAccessOTP.mockResolvedValue({ challenge_id: "challenge", hint: "a***@example.com" });
  api.verifyFormAccessOTP.mockResolvedValue({ session_id: "s", session_token: "session-secret", request_id: "r", distribution_id: "d", assurance: "EMAIL_VERIFIED" });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.change(await screen.findByLabelText("Verification code"), { target: { value: "123456" } });
  fireEvent.click(screen.getByRole("button", { name: "Verify code" }));
  expect(await screen.findByRole("heading", { name: "Request unavailable" })).toBeTruthy();
  expect(screen.queryByText("Is access reviewed?")).toBeNull();
});

it("offers another code without exposing diagnostics after a generic verification failure", async () => {
  api.startFormAccess.mockResolvedValue({ policy: "DIRECT_LINK_EMAIL_OTP", recipients: [{ selector_id: "recipient", hint: "a***@example.com" }] });
  api.sendFormAccessOTP.mockResolvedValue({ challenge_id: "challenge", hint: "a***@example.com" });
  api.verifyFormAccessOTP.mockRejectedValueOnce(new ApiError(401, "internal provider diagnostic", "form_access_failed"));
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.change(await screen.findByLabelText("Verification code"), { target: { value: "123456" } });
  fireEvent.click(screen.getByRole("button", { name: "Verify code" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Code could not be verified.");
  expect(screen.queryByText(/internal provider diagnostic/)).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Request another code" }));
  fireEvent.click(screen.getByRole("button", { name: "Send code" }));
  expect(await screen.findByLabelText("Verification code")).toBeTruthy();
  expect(api.sendFormAccessOTP).toHaveBeenCalledTimes(2);
});

it("sends a direct invitation code to its sole selected recipient", async () => {
  api.startFormAccess.mockResolvedValue({ policy: "DIRECT_LINK_EMAIL_OTP", recipients: [{ selector_id: "recipient", hint: "a***@example.com" }] });
  api.sendFormAccessOTP.mockResolvedValue({ challenge_id: "challenge", hint: "a***@example.com" });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  expect(await screen.findByLabelText("Verification code")).toBeTruthy();
  expect(api.sendFormAccessOTP).toHaveBeenCalledWith("invitation-secret", "recipient");
  expect(screen.queryByRole("radio")).toBeNull();
});

it("allows a direct recipient to retry sending a code after a delivery failure", async () => {
  api.startFormAccess.mockResolvedValue({ policy: "DIRECT_LINK_EMAIL_OTP", recipients: [{ selector_id: "recipient", hint: "a***@example.com" }] });
  api.sendFormAccessOTP.mockRejectedValueOnce(new Error("unavailable")).mockResolvedValueOnce({ challenge_id: "challenge", hint: "a***@example.com" });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Code could not be sent");
  fireEvent.click(screen.getByRole("button", { name: "Send code" }));
  expect(await screen.findByLabelText("Verification code")).toBeTruthy();
  expect(api.sendFormAccessOTP).toHaveBeenCalledTimes(2);
});

it("does not send a direct code when the recipient selection is ambiguous", async () => {
  api.startFormAccess.mockResolvedValue({ policy: "DIRECT_LINK_EMAIL_OTP", recipients: [{ selector_id: "first", hint: "a***@example.com" }, { selector_id: "second", hint: "b***@example.com" }] });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  expect(await screen.findByRole("heading", { name: "Request unavailable" })).toBeTruthy();
  expect(api.sendFormAccessOTP).not.toHaveBeenCalled();
});

it("rejects a workspace from a different distribution", async () => {
  api.loadFormResponseWorkspace.mockResolvedValue({ ...payload, workspace: { ...workspace, workspace: { ...workspace.workspace, distribution_id: "other" } } });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  expect(await screen.findByRole("heading", { name: "Request unavailable" })).toBeTruthy();
  expect(screen.queryByText("Is access reviewed?")).toBeNull();
});

it("rejects a response workspace for a different invitation session", async () => {
  api.loadFormResponseWorkspace.mockResolvedValue({ ...payload, session: { ...payload.session, id: "other" } });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  expect(await screen.findByRole("heading", { name: "Request unavailable" })).toBeTruthy();
  expect(screen.queryByText("Is access reviewed?")).toBeNull();
});

it("keeps the local response when conflict recovery returns another request", async () => {
  api.saveFormResponseWorkspace.mockRejectedValueOnce(new ApiError(409, "Conflict", "workspace_conflict"));
  api.loadFormResponseWorkspace.mockResolvedValueOnce(payload).mockResolvedValueOnce({ ...payload, request: { ...payload.request, id: "other" } });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.change(await screen.findByLabelText("Is access reviewed?"), { target: { value: "Yes" } });
  fireEvent.click(screen.getByRole("button", { name: "Save response" }));
  fireEvent.click(await screen.findByRole("button", { name: "Compare answers" }));
  expect(await screen.findByText("Latest answers unavailable. Try again.")).toBeTruthy();
  expect((screen.getByLabelText("Is access reviewed?") as HTMLSelectElement).value).toBe("Yes");
  expect(screen.queryByRole("button", { name: "Apply choices" })).toBeNull();
});

it("retains entered answers and blocks retries after a save conflict", async () => {
  api.saveFormResponseWorkspace.mockRejectedValue(new Error("conflict"));
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.change(await screen.findByLabelText("Is access reviewed?"), { target: { value: "Yes" } });
  fireEvent.click(screen.getByRole("button", { name: "Save response" }));
  expect(await screen.findByRole("alert")).toBeTruthy();
  expect((screen.getByLabelText("Is access reviewed?") as HTMLSelectElement).value).toBe("Yes");
  expect(screen.queryByRole("button", { name: "Submit response" })).toBeNull();
  expect(api.saveFormResponseWorkspace).toHaveBeenCalledWith("session-secret", { expected_version: 3, presentation_mode: "AUTOMATIC", edits: [{ field_id: "q", value: { text: "Yes" }, base_sequence: 0 }] });
});

it("warns before leaving with unsaved answers", async () => {
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.change(await screen.findByLabelText("Is access reviewed?"), { target: { value: "Yes" } });
  const event = new Event("beforeunload", { cancelable: true });
  window.dispatchEvent(event);
  expect(event.defaultPrevented).toBe(true);
});

it("submits only after the current response is saved and confirms a receipt", async () => {
  api.saveFormResponseWorkspace.mockResolvedValue({ ...workspace, workspace: { ...workspace.workspace, version: 4 }, answers: { q: { text: "Yes" } }, field_sequences: { q: 1 } });
  api.submitFormResponseWorkspace.mockResolvedValue({ workspace: { ...workspace.workspace, version: 5 }, revision: { ID: "revision", SubmissionID: "submission", Current: true }, submission: { submission_id: "submission", request_id: "r", status: "SUBMITTED", submitted_at: "2026-11-20T00:00:00Z" } });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.change(await screen.findByLabelText("Is access reviewed?"), { target: { value: "Yes" } });
  fireEvent.click(screen.getByRole("button", { name: "Save response" }));
  await screen.findByText("Response saved.");
  fireEvent.click(screen.getByRole("button", { name: "Submit response" }));
  expect(await screen.findByRole("heading", { name: "Response submitted" })).toBeTruthy();
  await waitFor(() => expect(api.submitFormResponseWorkspace).toHaveBeenCalledWith("session-secret", { expected_version: 4, attestation_field_ids: [] }));
});

it("blocks submission for unsupported questions", async () => {
  api.loadFormResponseWorkspace.mockResolvedValue({ ...payload, request: { ...payload.request, fields: [{ id: "f", type: "vendor_document", label: "Audit report", required: true }] } });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  expect(await screen.findByText("Audit report is unavailable. Contact the requester." )).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Submit response" })).toBeNull();
});

it("uploads a required document and saves its exact artifact identifier", async () => {
  api.loadFormResponseWorkspace.mockResolvedValue({ ...payload, request: { ...payload.request, fields: [{ id: "f", type: "file", label: "Audit report", required: true }] } });
  api.uploadCaptureSessionArtifact.mockResolvedValue({ id: "artifact-1", request_id: "r", file_name: "report.pdf", status: "PENDING_SCAN" });
  api.saveFormResponseWorkspace.mockResolvedValue({ ...workspace, workspace: { ...workspace.workspace, version: 4 }, answers: { f: { artifact_ids: ["artifact-1"] } }, field_sequences: { f: 1 } });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.change(await screen.findByLabelText("Audit report"), { target: { files: [new File(["file"], "report.pdf", { type: "application/pdf" })] } });
  await screen.findByText("1 document attached");
  fireEvent.click(screen.getByRole("button", { name: "Save response" }));
  await waitFor(() => expect(api.saveFormResponseWorkspace).toHaveBeenCalledWith("session-secret", { expected_version: 3, presentation_mode: "AUTOMATIC", edits: [{ field_id: "f", value: { artifact_ids: ["artifact-1"] }, base_sequence: 0 }] }));
});

it("does not offer a second submission when the receipt cannot be confirmed", async () => {
  api.loadFormResponseWorkspace.mockResolvedValue({ ...payload, workspace: { ...workspace, answers: { q: { text: "Yes" } } } });
  api.submitFormResponseWorkspace.mockResolvedValue({ workspace: { ...workspace.workspace, version: 4 }, revision: { ID: "revision", SubmissionID: "different", Current: true }, submission: { submission_id: "submission", request_id: "r" } });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.click(await screen.findByRole("button", { name: "Submit response" }));
  expect(await screen.findByRole("heading", { name: "Submission status unknown" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Submit response" })).toBeNull();
});

it("supports ordinary capture questions with explicit capture intent", async () => {
  api.loadFormResponseWorkspace.mockResolvedValue({ ...payload, request: { ...payload.request, fields: [{ id: "q", type: "yes_no", label: "Is access reviewed?", required: true, collection_intent: "CAPTURE" }] } });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  expect(await screen.findByLabelText("Is access reviewed?")).toBeTruthy();
});

it("lets the respondent compare and keep a local answer after a save conflict", async () => {
  api.saveFormResponseWorkspace.mockRejectedValueOnce(new ApiError(409, "Changed"));
  api.loadFormResponseWorkspace.mockResolvedValueOnce(payload).mockResolvedValueOnce({ ...payload, workspace: { ...workspace, workspace: { ...workspace.workspace, version: 4 }, answers: { q: { text: "No" } }, field_sequences: { q: 2 } } });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.change(await screen.findByLabelText("Is access reviewed?"), { target: { value: "Yes" } });
  fireEvent.click(screen.getByRole("button", { name: "Save response" }));
  expect(await screen.findByText("The response changed. Compare answers before saving.")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Compare answers" }));
  fireEvent.click(await screen.findByLabelText("Keep your answer: Yes"));
  fireEvent.click(screen.getByRole("button", { name: "Apply choices" }));
  expect((screen.getByLabelText("Is access reviewed?") as HTMLSelectElement).value).toBe("Yes");
  api.saveFormResponseWorkspace.mockResolvedValueOnce({ ...workspace, workspace: { ...workspace.workspace, version: 5 }, answers: { q: { text: "Yes" } }, field_sequences: { q: 3 } });
  fireEvent.click(screen.getByRole("button", { name: "Save response" }));
  await waitFor(() => expect(api.saveFormResponseWorkspace).toHaveBeenLastCalledWith("session-secret", expect.objectContaining({ expected_version: 4, edits: [{ field_id: "q", value: { text: "Yes" }, base_sequence: 2 }] })));
});

it("compares the saved answer with a newer answer after a submission conflict", async () => {
  api.loadFormResponseWorkspace.mockResolvedValueOnce({ ...payload, workspace: { ...workspace, answers: { q: { text: "Yes" } }, field_sequences: { q: 1 } } }).mockResolvedValueOnce({ ...payload, workspace: { ...workspace, workspace: { ...workspace.workspace, version: 4 }, answers: { q: { text: "No" } }, field_sequences: { q: 2 } } });
  api.submitFormResponseWorkspace.mockRejectedValueOnce(new ApiError(409, "Conflict", "workspace_conflict"));
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.click(await screen.findByRole("button", { name: "Submit response" }));
  fireEvent.click(await screen.findByRole("button", { name: "Compare answers" }));
  expect(await screen.findByLabelText("Keep your answer: Yes")).toBeTruthy();
  expect(screen.getByLabelText("Use latest answer: No")).toBeTruthy();
  fireEvent.click(screen.getByLabelText("Keep your answer: Yes"));
  fireEvent.click(screen.getByRole("button", { name: "Apply choices" }));
  expect((screen.getByLabelText("Is access reviewed?") as HTMLSelectElement).value).toBe("Yes");
  expect(screen.getByRole("button", { name: "Save response" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Submit response" })).toBeNull();
});

it("can use the latest answer after a submission conflict without resaving the old one", async () => {
  api.loadFormResponseWorkspace.mockResolvedValueOnce({ ...payload, workspace: { ...workspace, answers: { q: { text: "Yes" } }, field_sequences: { q: 1 } } }).mockResolvedValueOnce({ ...payload, workspace: { ...workspace, workspace: { ...workspace.workspace, version: 4 }, answers: { q: { text: "No" } }, field_sequences: { q: 2 } } });
  api.submitFormResponseWorkspace.mockRejectedValueOnce(new ApiError(409, "Conflict", "workspace_conflict"));
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.click(await screen.findByRole("button", { name: "Submit response" }));
  fireEvent.click(await screen.findByRole("button", { name: "Compare answers" }));
  fireEvent.click(await screen.findByLabelText("Use latest answer: No"));
  fireEvent.click(screen.getByRole("button", { name: "Apply choices" }));
  expect((screen.getByLabelText("Is access reviewed?") as HTMLSelectElement).value).toBe("No");
  expect(screen.queryByRole("button", { name: "Save response" })).toBeNull();
  expect(screen.getByRole("button", { name: "Submit response" })).toBeTruthy();
});

it("does not offer to choose between changed documents after a submission conflict", async () => {
  const request = { ...payload.request, fields: [{ id: "q", type: "file", label: "Audit report", required: true }] };
  api.loadFormResponseWorkspace.mockResolvedValueOnce({ ...payload, request, workspace: { ...workspace, answers: { q: { artifact_ids: ["old"] } }, field_sequences: { q: 1 } } }).mockResolvedValueOnce({ ...payload, request, workspace: { ...workspace, workspace: { ...workspace.workspace, version: 4 }, answers: { q: { artifact_ids: ["new"] } }, field_sequences: { q: 2 } } });
  api.submitFormResponseWorkspace.mockRejectedValueOnce(new ApiError(409, "Conflict", "workspace_conflict"));
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.click(await screen.findByRole("button", { name: "Submit response" }));
  fireEvent.click(await screen.findByRole("button", { name: "Compare answers" }));
  expect(await screen.findByText("Document answers changed. Contact the requester before continuing.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Apply choices" })).toBeNull();
});

it("distinguishes a definite submission conflict from an uncertain submission", async () => {
  api.loadFormResponseWorkspace.mockResolvedValue({ ...payload, workspace: { ...workspace, answers: { q: { text: "Yes" } } } });
  api.submitFormResponseWorkspace.mockRejectedValueOnce(new ApiError(409, "Changed"));
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.click(await screen.findByRole("button", { name: "Submit response" }));
  expect(await screen.findByText("The response changed. Compare answers before submitting.")).toBeTruthy();
  expect(screen.queryByRole("heading", { name: "Submission status unknown" })).toBeNull();
});

it("preserves unrelated edits when the latest answer is selected for a conflict", async () => {
  const withTwo = { ...payload, request: { ...payload.request, fields: [...payload.request.fields, { id: "detail", type: "short_text", label: "Reason", required: false }] } };
  api.saveFormResponseWorkspace.mockRejectedValueOnce(new ApiError(409, "Changed"));
  api.loadFormResponseWorkspace.mockResolvedValueOnce(withTwo).mockResolvedValueOnce({ ...withTwo, workspace: { ...workspace, workspace: { ...workspace.workspace, version: 4 }, answers: { q: { text: "No" } }, field_sequences: { q: 2 } } });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.change(await screen.findByLabelText("Is access reviewed?"), { target: { value: "Yes" } });
  fireEvent.change(screen.getByLabelText("Reason"), { target: { value: "Quarterly review" } });
  fireEvent.click(screen.getByRole("button", { name: "Save response" }));
  fireEvent.click(await screen.findByRole("button", { name: "Compare answers" }));
  fireEvent.click(await screen.findByLabelText("Use latest answer: No"));
  fireEvent.click(screen.getByRole("button", { name: "Apply choices" }));
  expect((screen.getByLabelText("Is access reviewed?") as HTMLSelectElement).value).toBe("No");
  expect((screen.getByLabelText("Reason") as HTMLInputElement).value).toBe("Quarterly review");
  api.saveFormResponseWorkspace.mockResolvedValueOnce({ ...workspace, workspace: { ...workspace.workspace, version: 5 }, answers: { q: { text: "No" }, detail: { text: "Quarterly review" } }, field_sequences: { q: 2, detail: 1 } });
  fireEvent.click(screen.getByRole("button", { name: "Save response" }));
  await waitFor(() => expect(api.saveFormResponseWorkspace).toHaveBeenLastCalledWith("session-secret", expect.objectContaining({ expected_version: 4, edits: [{ field_id: "detail", value: { text: "Quarterly review" }, base_sequence: 0 }] })));
});

it("preserves answers after a definite save validation error", async () => {
  api.saveFormResponseWorkspace.mockRejectedValueOnce(new ApiError(422, "Invalid"));
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.change(await screen.findByLabelText("Is access reviewed?"), { target: { value: "Yes" } });
  fireEvent.click(screen.getByRole("button", { name: "Save response" }));
  expect(await screen.findByText("Response not saved. Check your answers and try again.")).toBeTruthy();
  expect((screen.getByLabelText("Is access reviewed?") as HTMLSelectElement).value).toBe("Yes");
  expect(screen.getByRole("button", { name: "Save response" })).toBeTruthy();
});

it("keeps the form open after a definite submission validation error", async () => {
  api.loadFormResponseWorkspace.mockResolvedValue({ ...payload, workspace: { ...workspace, answers: { q: { text: "Yes" } } } });
  api.submitFormResponseWorkspace.mockRejectedValueOnce(new ApiError(400, "Invalid"));
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.click(await screen.findByRole("button", { name: "Submit response" }));
  expect(await screen.findByText("Response not submitted. Check your answers and try again.")).toBeTruthy();
  expect(screen.getByRole("button", { name: "Submit response" })).toBeTruthy();
});

it("does not offer ambiguous choices for conflicting documents", async () => {
  const docRequest = { ...payload, request: { ...payload.request, fields: [{ id: "f", type: "file", label: "Audit report", required: true }] } };
  api.loadFormResponseWorkspace.mockResolvedValueOnce(docRequest).mockResolvedValueOnce({ ...docRequest, workspace: { ...workspace, workspace: { ...workspace.workspace, version: 4 }, answers: { f: { artifact_ids: ["server-file"] } }, field_sequences: { f: 1 } } });
  api.uploadCaptureSessionArtifact.mockResolvedValue({ id: "local-file", request_id: "r", status: "STORED_UNSCANNED", file_name: "local.pdf" });
  api.saveFormResponseWorkspace.mockRejectedValueOnce(new ApiError(409, "Changed"));
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.change(await screen.findByLabelText("Audit report"), { target: { files: [new File(["file"], "local.pdf", { type: "application/pdf" })] } });
  await screen.findByText("1 document attached");
  fireEvent.click(screen.getByRole("button", { name: "Save response" }));
  fireEvent.click(await screen.findByRole("button", { name: "Compare answers" }));
  expect(await screen.findByText("Document answers changed. Contact the requester before continuing.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Apply choices" })).toBeNull();
});

it("shows an unscanned document status and rejects a quarantined upload", async () => {
  api.loadFormResponseWorkspace.mockResolvedValue({ ...payload, request: { ...payload.request, fields: [{ id: "f", type: "file", label: "Audit report", required: true }] } });
  api.uploadCaptureSessionArtifact.mockResolvedValueOnce({ id: "bad", request_id: "r", status: "QUARANTINED", file_name: "bad.pdf" }).mockResolvedValueOnce({ id: "new", request_id: "r", status: "STORED_UNSCANNED", file_name: "new.pdf" });
  render(<Capture invitationToken="invitation-secret"/>);
  fireEvent.click(screen.getByRole("button", { name: "Open request" }));
  fireEvent.change(await screen.findByLabelText("Audit report"), { target: { files: [new File(["file"], "bad.pdf")] } });
  expect(await screen.findByText("Document unavailable. Select another file.")).toBeTruthy();
  expect(screen.queryByText("1 document attached")).toBeNull();
  fireEvent.change(screen.getByLabelText("Audit report"), { target: { files: [new File(["file"], "new.pdf")] } });
  expect(await screen.findByText("new.pdf · Not scanned at upload")).toBeTruthy();
});
