import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadCaptureDraft, saveCaptureDraft, submitInternalCaptureRequest } from "../captureApi";
import { ApiError } from "../http";
import type { CaptureRequest } from "../types";
import { CapturePanel } from "./CapturePanel";

vi.mock("../captureApi", async () => ({
  ...await vi.importActual<typeof import("../captureApi")>("../captureApi"),
  loadCaptureDraft: vi.fn(),
  saveCaptureDraft: vi.fn(),
  submitInternalCaptureRequest: vi.fn(),
}));

const request: CaptureRequest = {
  id: "request-1",
  title: "Confirm the current control owner",
  purpose: "Confirm who owns this process now.",
  why_you: "You own the affected process.",
  status: "READY",
  sensitivity: "INTERNAL",
  estimated_minutes: 2,
  deadline: "2027-08-09T12:00:00Z",
  known_facts: { process: "Treasury operations" },
  fields: [{ id: "owner", label: "Current owner", type: "text", required: true }],
  version: 3,
};

const multiFieldRequest: CaptureRequest = {
  ...request,
  id: "request-2",
  title: "Confirm annual-return evidence ownership",
  fields: [
    { id: "owner", label: "Processor register owner", type: "text", required: true },
    { id: "review_date", label: "DPCO review date", type: "date", required: true },
  ],
};

const wizardRequest: CaptureRequest = {
  ...request,
  id: "request-wizard",
  presentation: { default_mode: "WIZARD", allow_mode_switch: false },
  sections: [{ id: "company", title: "Company" }, { id: "evidence", title: "Evidence" }],
  fields: [
    { id: "owner", section_id: "company", label: "Current owner", type: "short_text", required: true },
    { id: "note", section_id: "evidence", label: "Evidence note", type: "long_text", required: false },
  ],
};

describe("CapturePanel", () => {
  beforeEach(() => {
    vi.mocked(loadCaptureDraft).mockReset();
    vi.mocked(saveCaptureDraft).mockReset();
  });

  it("uses a short input and reviews the exact response before submitting", async () => {
    vi.mocked(submitInternalCaptureRequest).mockResolvedValue({ request_id: request.id, status: "SUBMITTED", submitted_at: "2026-08-06T19:30:00Z" });
    render(<CapturePanel request={request}/>);

    const owner = screen.getByRole("textbox", { name: /Current owner/ }) as HTMLInputElement;
    expect(owner.tagName).toBe("INPUT");
    fireEvent.change(owner, { target: { value: "Treasury Technology" } });
    fireEvent.click(screen.getByRole("button", { name: "Review response" }));

    expect(screen.getByRole("heading", { name: "Check your response" })).toBeTruthy();
    expect(screen.getByText("Treasury Technology")).toBeTruthy();
    expect(submitInternalCaptureRequest).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "Submit response" }));
    await waitFor(() => expect(submitInternalCaptureRequest).toHaveBeenCalledWith(request.id, 3, { owner: { text: "Treasury Technology" } }));
    expect(await screen.findByRole("heading", { name: "Response submitted" })).toBeTruthy();
    expect(screen.getByText(/recorded for evidence review/i)).toBeTruthy();
  });

  it("uses the native date control and preserves multiple answers", () => {
    render(<CapturePanel request={multiFieldRequest}/>);

    fireEvent.change(screen.getByRole("textbox", { name: /Processor register owner/ }), { target: { value: "Privacy Operations" } });
    const date = screen.getByLabelText(/DPCO review date/) as HTMLInputElement;
    expect(date.type).toBe("date");
    fireEvent.change(date, { target: { value: "2027-03-01" } });

    const review = screen.getByRole("button", { name: "Review response" }) as HTMLButtonElement;
    expect(review.disabled).toBe(false);
    fireEvent.click(review);

    expect(screen.getByText("Privacy Operations")).toBeTruthy();
    expect(screen.getByText(/Mar 1, 2027|1 Mar 2027/)).toBeTruthy();
  });

  it("uses large tap choices for short option lists", () => {
    render(<CapturePanel request={{ ...request, fields: [{ id: "present", label: "Is the ATM present?", type: "single_select", required: true, options: ["Yes", "No"] }] }}/>);
    const yes = screen.getByRole("radio", { name: "Yes" });
    const no = screen.getByRole("radio", { name: "No" });
    expect(yes).toBeTruthy();
    expect(no).toBeTruthy();
    fireEvent.click(yes);
    expect((screen.getByRole("button", { name: "Review response" }) as HTMLButtonElement).disabled).toBe(false);
  });

  it("uploads a photo and reviews it without exposing the artifact id", async () => {
    const upload = vi.fn().mockResolvedValue({ id: "artifact-secret-id", request_id: request.id, file_name: "atm.jpg", media_type: "image/jpeg", size_bytes: 1200, sha256: "hash", status: "STORED_UNSCANNED" });
    render(<CapturePanel request={{ ...request, fields: [{ id: "photo", label: "Site photo", type: "photo", required: true, accepted_formats: ["image/jpeg"] }] }} onUploadArtifact={upload}/>);
    const input = screen.getByLabelText(/Site photo/) as HTMLInputElement;
    const file = new File(["photo"], "atm.jpg", { type: "image/jpeg" });
    fireEvent.change(input, { target: { files: [file] } });
	await waitFor(() => expect(upload).toHaveBeenCalledWith(request.id, file, "photo"));
    expect(screen.getByText(/atm\.jpg/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Review response" }));
    expect(screen.getByText(/Photo attached · atm.jpg/)).toBeTruthy();
    expect(screen.queryByText("artifact-secret-id")).toBeNull();
  });

	it("uploads, reviews and removes multiple files without losing successful uploads", async () => {
	  const upload = vi.fn()
		.mockResolvedValueOnce({ id: "artifact-policy", request_id: request.id, file_name: "policy.pdf", media_type: "application/pdf", size_bytes: 1200, sha256: "hash-1", status: "STORED_UNSCANNED" })
		.mockResolvedValueOnce({ id: "artifact-register", request_id: request.id, file_name: "register.xlsx", media_type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", size_bytes: 2200, sha256: "hash-2", status: "STORED_UNSCANNED" });
	  const submit = vi.fn().mockResolvedValue({ submitted_at: "2026-08-26T12:00:00Z" });
	  render(<CapturePanel request={{ ...request, fields: [{ id: "documents", label: "Due diligence documents", type: "file", required: true, accepted_formats: ["application/pdf", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"], constraints: { min_files: 1, max_files: 3, max_file_bytes: 5000, max_total_file_bytes: 10000 } }] }} onUploadArtifact={upload} onSubmit={submit}/>);

	  const input = screen.getByLabelText(/Due diligence documents/) as HTMLInputElement;
	  expect(input.multiple).toBe(true);
	  const policy = new File(["policy"], "policy.pdf", { type: "application/pdf" });
	  const register = new File(["register"], "register.xlsx", { type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" });
	  fireEvent.change(input, { target: { files: [policy, register] } });
	  await waitFor(() => expect(upload).toHaveBeenCalledTimes(2));
	  expect(upload).toHaveBeenNthCalledWith(1, request.id, policy, "documents");
	  expect(upload).toHaveBeenNthCalledWith(2, request.id, register, "documents");
	  expect(screen.getByText("policy.pdf")).toBeTruthy();
	  expect(screen.getByText("register.xlsx")).toBeTruthy();

	  fireEvent.click(screen.getAllByRole("button", { name: "Remove" })[0]!);
	  expect(screen.queryByText("policy.pdf")).toBeNull();
	  fireEvent.click(screen.getByRole("button", { name: "Review response" }));
	  expect(screen.getByText(/1 file attached · register.xlsx/)).toBeTruthy();
	  fireEvent.click(screen.getByRole("button", { name: "Submit response" }));
	  await waitFor(() => expect(submit).toHaveBeenCalledWith(expect.objectContaining({ id: request.id }), { documents: { artifact_ids: ["artifact-register"] } }));
	});

  it("preserves the previous valid attachment when a replacement upload fails", async () => {
    const upload = vi.fn()
      .mockResolvedValueOnce({ id: "artifact-original", request_id: request.id, file_name: "original.jpg", media_type: "image/jpeg", size_bytes: 1200, sha256: "hash-1", status: "STORED_UNSCANNED" })
      .mockRejectedValueOnce(new ApiError(503, "Upload failed", "unavailable"));
    render(<CapturePanel request={{ ...request, fields: [{ id: "photo", label: "Site photo", type: "photo", required: true, accepted_formats: ["image/jpeg"] }] }} onUploadArtifact={upload}/>);

    const input = screen.getByLabelText(/Site photo/) as HTMLInputElement;
    const original = new File(["original"], "original.jpg", { type: "image/jpeg" });
    fireEvent.change(input, { target: { files: [original] } });
    await waitFor(() => expect(screen.getByRole("button", { name: "Replace photo" })).toBeTruthy());

    const replacement = new File(["replacement"], "replacement.jpg", { type: "image/jpeg" });
    fireEvent.change(input, { target: { files: [replacement] } });
	expect((await screen.findByRole("alert")).textContent).toMatch(/previous file remains selected/i);
    expect(screen.getByText(/original\.jpg/)).toBeTruthy();

    const review = screen.getByRole("button", { name: "Review response" }) as HTMLButtonElement;
    expect(review.disabled).toBe(false);
    fireEvent.click(review);
    expect(screen.getByText(/Photo attached · original.jpg/)).toBeTruthy();
  });

  it("ignores an upload completion after the active request changes", async () => {
    let resolveUpload!: (value: { id: string; request_id: string; file_name: string; media_type: string; size_bytes: number; sha256: string; status: "STORED_UNSCANNED" }) => void;
    const upload = vi.fn().mockImplementation(() => new Promise((resolve) => { resolveUpload = resolve; }));
    const photoRequest: CaptureRequest = { ...request, fields: [{ id: "photo", label: "Site photo", type: "photo", required: true, accepted_formats: ["image/jpeg"] }] };
    const { rerender } = render(<CapturePanel request={photoRequest} onUploadArtifact={upload}/>);

    fireEvent.change(screen.getByLabelText(/Site photo/), { target: { files: [new File(["old"], "old.jpg", { type: "image/jpeg" })] } });
    await waitFor(() => expect(upload).toHaveBeenCalledTimes(1));
    rerender(<CapturePanel request={{ ...request, id: "request-new", version: 1 }} onUploadArtifact={upload}/>);
    resolveUpload({ id: "artifact-old", request_id: request.id, file_name: "old.jpg", media_type: "image/jpeg", size_bytes: 100, sha256: "hash-old", status: "STORED_UNSCANNED" });

    await waitFor(() => expect(screen.queryByText(/old\.jpg/)).toBeNull());
    expect((screen.getByRole("textbox", { name: /Current owner/ }) as HTMLInputElement).value).toBe("");
    expect(screen.getByRole("button", { name: "Review response" })).toBeTruthy();
  });

  it("ignores a submission completion after the active request changes", async () => {
    let resolveSubmit!: (value: { submitted_at: string }) => void;
    const submit = vi.fn().mockImplementation(() => new Promise((resolve) => { resolveSubmit = resolve; }));
    const { rerender } = render(<CapturePanel request={request} onSubmit={submit}/>);

    fireEvent.change(screen.getByRole("textbox", { name: /Current owner/ }), { target: { value: "Treasury Technology" } });
    fireEvent.click(screen.getByRole("button", { name: "Review response" }));
    fireEvent.click(screen.getByRole("button", { name: "Submit response" }));
    await waitFor(() => expect(submit).toHaveBeenCalledTimes(1));

    rerender(<CapturePanel request={{ ...request, id: "request-new", version: 1 }} onSubmit={submit}/>);
    resolveSubmit({ submitted_at: "2026-08-07T21:30:00Z" });

    await waitFor(() => expect(screen.queryByRole("heading", { name: "Response submitted" })).toBeNull());
    expect((screen.getByRole("textbox", { name: /Current owner/ }) as HTMLInputElement).value).toBe("");
  });

  it("records an external submission as a response without claiming verification", async () => {
    const submit = vi.fn().mockResolvedValue({ submitted_at: "2026-08-07T21:30:00Z" });
    render(<CapturePanel request={request} external onSubmit={submit}/>);

    fireEvent.change(screen.getByRole("textbox", { name: /Current owner/ }), { target: { value: "Treasury Technology" } });
    fireEvent.click(screen.getByRole("button", { name: "Review response" }));
    fireEvent.click(screen.getByRole("button", { name: "Submit response" }));

    expect(await screen.findByRole("heading", { name: "Submitted" })).toBeTruthy();
    expect(screen.getByText("Your response was recorded.")).toBeTruthy();
    expect(screen.queryByText("Your verification was recorded.")).toBeNull();
  });

  it("normalizes server-valid field types and accepted media formats in the browser", () => {
    render(<CapturePanel request={{ ...request, fields: [{ id: "photo", label: "Site photo", type: " PHOTO ", required: true, accepted_formats: [" IMAGE/JPEG ; charset=binary "] }] }}/>);
    const input = screen.getByLabelText(/Site photo/) as HTMLInputElement;
    expect(input.accept).toBe("image/jpeg");
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("shows external respondents the deadline, data boundary and recovery path before the form", () => {
    render(<CapturePanel request={request} external/>);

    expect(screen.getByText(/Due 9 Aug 2027/)).toBeTruthy();
    expect(screen.getByText("Your answers and files are shared with the organization that sent this request.")).toBeTruthy();
    expect(screen.getByText("For changes to the request or your access, contact the person who sent this link.")).toBeTruthy();
  });

  it("submits typed vendor-document metadata with the uploaded artifact", async () => {
    const submit = vi.fn().mockResolvedValue({ submitted_at: "2026-08-07T21:30:00Z" });
    const upload = vi.fn().mockResolvedValue({ id: "artifact-certificate", request_id: request.id, file_name: "iso.pdf", media_type: "application/pdf", size_bytes: 2400, sha256: "hash", status: "STORED_UNSCANNED" });
    render(<CapturePanel request={{ ...request, fields: [{ id: "certificate", label: "ISO certificate", type: "vendor_document", required: true, accepted_formats: ["application/pdf"] }] }} onSubmit={submit} onUploadArtifact={upload}/>);

    fireEvent.change(screen.getByLabelText("Document type"), { target: { value: "ISO 27001 certificate" } });
    fireEvent.change(screen.getByLabelText("Document reference"), { target: { value: "CERT-2026-81" } });
    const file = new File(["certificate"], "iso.pdf", { type: "application/pdf" });
    fireEvent.change(screen.getByLabelText(/ISO certificate file/), { target: { files: [file] } });
	await waitFor(() => expect(upload).toHaveBeenCalledWith(request.id, file, "certificate"));

    fireEvent.click(screen.getByRole("button", { name: "Review response" }));
    expect(screen.getByText(/ISO 27001 certificate · CERT-2026-81/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Submit response" }));
    await waitFor(() => expect(submit).toHaveBeenCalledWith(expect.objectContaining({ id: request.id }), {
      certificate: { document: { artifact_id: "artifact-certificate", document_type: "ISO 27001 certificate", reference: "CERT-2026-81" } },
    }));
  });

  it("omits an answer when a controlling response hides its field", async () => {
    const submit = vi.fn().mockResolvedValue({ submitted_at: "2026-08-07T21:30:00Z" });
    render(<CapturePanel request={{
      ...request,
      fields: [
        { id: "handles_data", label: "Handles customer data", type: "yes_no", required: true },
        { id: "location", label: "Processing location", type: "short_text", required: true, condition: { field_id: "handles_data", operator: "EQUALS", values: ["Yes"] } },
      ],
    }} onSubmit={submit}/>);

    fireEvent.click(screen.getByRole("radio", { name: "Yes" }));
    fireEvent.change(screen.getByRole("textbox", { name: /Processing location/ }), { target: { value: "Lagos" } });
    fireEvent.click(screen.getByRole("radio", { name: "No" }));
    expect(screen.queryByRole("textbox", { name: /Processing location/ })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Review response" }));
    fireEvent.click(screen.getByRole("button", { name: "Submit response" }));
    await waitFor(() => expect(submit).toHaveBeenCalledWith(expect.objectContaining({ id: request.id }), { handles_data: { text: "No" } }));
  });

  it("labels each collapsed optional external note with its request field", () => {
    render(<CapturePanel request={{ ...request, fields: [
      { id: "visit_note", label: "Anything the reviewer should know?", type: "long_text", required: false },
      { id: "safety_note", label: "Any safety concern?", type: "long_text", required: false },
    ] }} external/>);
    expect(screen.getByText("Anything the reviewer should know?", { selector: "summary" })).toBeTruthy();
    expect(screen.getByText("Any safety concern?", { selector: "summary" })).toBeTruthy();
  });

  it("clears stale answers when the same request advances to a new version", () => {
    const { rerender } = render(<CapturePanel request={request}/>);
    fireEvent.change(screen.getByRole("textbox", { name: /Current owner/ }), { target: { value: "Old owner" } });
    rerender(<CapturePanel request={{ ...request, version: 4 }}/>);
    expect((screen.getByRole("textbox", { name: /Current owner/ }) as HTMLInputElement).value).toBe("");
  });

  it("keeps terminal requests read-only", () => {
    render(<CapturePanel request={{ ...request, status: "EXPIRED" }}/>);
    expect(screen.getByRole("heading", { name: "This request has expired" })).toBeTruthy();
    expect(screen.queryByRole("textbox")).toBeNull();
    expect(screen.queryByRole("button", { name: "Review response" })).toBeNull();
  });

  it("distinguishes loading and forbidden states without exposing request fields", () => {
    const { rerender } = render(<CapturePanel request={null} state="loading"/>);
    expect(screen.getByRole("heading", { name: "Loading request" })).toBeTruthy();
    rerender(<CapturePanel request={null} state="forbidden"/>);
    expect(screen.getByRole("heading", { name: "You cannot open this request" })).toBeTruthy();
    expect(screen.queryByText("Treasury operations")).toBeNull();
  });

  it("surfaces an optimistic conflict and keeps the response available for reload", async () => {
    const reload = vi.fn();
    vi.mocked(submitInternalCaptureRequest).mockRejectedValue(new ApiError(409, "The request changed", "version_conflict"));
    render(<CapturePanel request={request} onReload={reload}/>);
    fireEvent.change(screen.getByRole("textbox", { name: /Current owner/ }), { target: { value: "Treasury Technology" } });
    fireEvent.click(screen.getByRole("button", { name: "Review response" }));
    fireEvent.click(screen.getByRole("button", { name: "Submit response" }));
    expect((await screen.findByRole("alert")).textContent).toMatch(/changed while you were working/i);
    fireEvent.click(screen.getByRole("button", { name: "Reload request" }));
    expect(reload).toHaveBeenCalledTimes(1);
  });

  it("fails closed for a genuinely unknown field contract", () => {
    render(<CapturePanel request={{ ...request, fields: [{ id: "unknown", label: "Unrecognized field", type: "biometric_scan", required: true }] }}/>);
    expect(screen.getByRole("alert").textContent).toMatch(/cannot be collected here/i);
    expect((screen.getByRole("button", { name: "Review response" }) as HTMLButtonElement).disabled).toBe(true);
  });

  it("resumes and autosaves only an external bearer-session draft", async () => {
    vi.mocked(loadCaptureDraft).mockResolvedValue({ answers: { owner: { text: "Saved owner" } }, presentation_mode: "CLASSIC", version: 4 });
    vi.mocked(saveCaptureDraft).mockResolvedValue({ answers: { owner: { text: "Current owner" } }, presentation_mode: "CLASSIC", version: 5, updated_at: "2026-08-26T12:00:00Z" });
    render(<CapturePanel request={request} external sessionToken="session-secret"/>);

    const owner = await screen.findByDisplayValue("Saved owner");
    fireEvent.change(owner, { target: { value: "Current owner" } });

    await waitFor(() => expect(saveCaptureDraft).toHaveBeenCalledWith("session-secret", {
      answers: { owner: { text: "Current owner" } }, presentation_mode: "CLASSIC", expected_version: 4,
    }), { timeout: 1500 });
    expect(await screen.findByText("Saved")).toBeTruthy();
  });

	it("restores draft attachment identifiers before appending another file", async () => {
	  vi.mocked(loadCaptureDraft).mockResolvedValue({ answers: { documents: { artifact_ids: ["artifact-existing"] } }, presentation_mode: "CLASSIC", version: 2 });
	  vi.mocked(saveCaptureDraft).mockResolvedValue({ answers: {}, presentation_mode: "CLASSIC", version: 3, updated_at: "2026-08-26T12:00:00Z" });
	  const upload = vi.fn().mockResolvedValue({ id: "artifact-new", request_id: request.id, file_name: "current.pdf", media_type: "application/pdf", size_bytes: 900, sha256: "hash", status: "STORED_UNSCANNED" });
	  const submit = vi.fn().mockResolvedValue({ submitted_at: "2026-08-26T12:00:00Z" });
	  render(<CapturePanel request={{ ...request, fields: [{ id: "documents", label: "Due diligence documents", type: "file", required: true, accepted_formats: ["application/pdf"], constraints: { max_files: 3 } }] }} external sessionToken="session-secret" onUploadArtifact={upload} onSubmit={submit}/>);

	  expect(await screen.findByText("Previously uploaded file")).toBeTruthy();
	  const file = new File(["current"], "current.pdf", { type: "application/pdf" });
	  fireEvent.change(screen.getByLabelText(/Due diligence documents/), { target: { files: [file] } });
	  await waitFor(() => expect(screen.getByText("current.pdf")).toBeTruthy());
	  fireEvent.click(screen.getByRole("button", { name: "Review response" }));
	  fireEvent.click(screen.getByRole("button", { name: "Submit response" }));
	  await waitFor(() => expect(submit).toHaveBeenCalledWith(expect.objectContaining({ id: request.id }), { documents: { artifact_ids: ["artifact-existing", "artifact-new"] } }));
	});

  it("keeps local entries when autosave fails and offers one retry", async () => {
    vi.mocked(loadCaptureDraft).mockResolvedValue({ answers: {}, presentation_mode: "AUTOMATIC", version: 0 });
    vi.mocked(saveCaptureDraft).mockRejectedValue(new ApiError(503, "Draft unavailable", "draft_unavailable"));
    render(<CapturePanel request={request} external sessionToken="session-secret"/>);

    const owner = await screen.findByRole("textbox", { name: /Current owner/ });
    fireEvent.change(owner, { target: { value: "Treasury Technology" } });

    expect(await screen.findByText("Could not save", {}, { timeout: 1500 })).toBeTruthy();
    expect(screen.getByText(/entries remain on this screen/i)).toBeTruthy();
    expect((owner as HTMLInputElement).value).toBe("Treasury Technology");
    expect(screen.getByRole("button", { name: "Try again" })).toBeTruthy();
  });

  it("refreshes only the draft version before retrying a conflict", async () => {
    vi.mocked(loadCaptureDraft)
      .mockResolvedValueOnce({ answers: {}, presentation_mode: "AUTOMATIC", version: 2 })
      .mockResolvedValueOnce({ answers: { owner: { text: "Other tab value" } }, presentation_mode: "AUTOMATIC", version: 7 });
    vi.mocked(saveCaptureDraft)
      .mockRejectedValueOnce(new ApiError(409, "Draft changed", "draft_conflict"))
      .mockResolvedValueOnce({ answers: { owner: { text: "Treasury Technology" } }, presentation_mode: "AUTOMATIC", version: 8 });
    render(<CapturePanel request={request} external sessionToken="session-secret"/>);

    const owner = await screen.findByRole("textbox", { name: /Current owner/ }) as HTMLInputElement;
    fireEvent.change(owner, { target: { value: "Treasury Technology" } });
    expect(await screen.findByText("Could not save", {}, { timeout: 1500 })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    await waitFor(() => expect(saveCaptureDraft).toHaveBeenLastCalledWith("session-secret", {
      answers: { owner: { text: "Treasury Technology" } }, presentation_mode: "AUTOMATIC", expected_version: 7,
    }));
    expect(owner.value).toBe("Treasury Technology");
    expect(await screen.findByText("Saved")).toBeTruthy();
  });

  it("reports ended draft access without clearing the current response", async () => {
    vi.mocked(loadCaptureDraft).mockRejectedValue(new ApiError(401, "Session unavailable", "session_unavailable"));
    render(<CapturePanel request={request} external sessionToken="ended-session"/>);

    expect(await screen.findByText("Access ended")).toBeTruthy();
    const owner = screen.getByRole("textbox", { name: /Current owner/ }) as HTMLInputElement;
    fireEvent.change(owner, { target: { value: "Treasury Technology" } });
    expect(owner.value).toBe("Treasury Technology");
    expect(saveCaptureDraft).not.toHaveBeenCalled();
  });

  it("does not leave a Wizard section until its delayed draft save completes", async () => {
    let finishSave!: (value: { answers: { owner: { text: string } }; presentation_mode: "WIZARD"; version: number; updated_at: string }) => void;
    vi.mocked(loadCaptureDraft).mockResolvedValue({ answers: {}, presentation_mode: "WIZARD", version: 0 });
    vi.mocked(saveCaptureDraft).mockImplementation(() => new Promise((resolve) => { finishSave = resolve; }));
    render(<CapturePanel request={wizardRequest} external sessionToken="session-secret"/>);

    const owner = await screen.findByRole("textbox", { name: /Current owner/ }) as HTMLInputElement;
    await waitFor(() => expect(loadCaptureDraft).toHaveBeenCalledTimes(1));
    fireEvent.change(owner, { target: { value: "Treasury Technology" } });
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));

    await waitFor(() => expect(saveCaptureDraft).toHaveBeenCalledTimes(1));
    expect(screen.getByRole("heading", { name: "Company" })).toBeTruthy();
    expect(owner.value).toBe("Treasury Technology");

    finishSave({ answers: { owner: { text: "Treasury Technology" } }, presentation_mode: "WIZARD", version: 1, updated_at: "2026-08-26T12:00:00Z" });
    expect(await screen.findByRole("heading", { name: "Evidence" })).toBeTruthy();
  });

  it("saves newer Wizard entries after an earlier background save finishes", async () => {
    let finishFirst!: (value: { answers: { owner: { text: string } }; presentation_mode: "WIZARD"; version: number }) => void;
    let finishSecond!: (value: { answers: { owner: { text: string } }; presentation_mode: "WIZARD"; version: number }) => void;
    vi.mocked(loadCaptureDraft).mockResolvedValue({ answers: {}, presentation_mode: "WIZARD", version: 0 });
    vi.mocked(saveCaptureDraft)
      .mockImplementationOnce(() => new Promise((resolve) => { finishFirst = resolve; }))
      .mockImplementationOnce(() => new Promise((resolve) => { finishSecond = resolve; }));
    render(<CapturePanel request={wizardRequest} external sessionToken="session-secret"/>);

    const owner = await screen.findByRole("textbox", { name: /Current owner/ }) as HTMLInputElement;
    fireEvent.change(owner, { target: { value: "First owner" } });
    await waitFor(() => expect(saveCaptureDraft).toHaveBeenCalledTimes(1), { timeout: 1500 });

    fireEvent.change(owner, { target: { value: "Current owner" } });
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    finishFirst({ answers: { owner: { text: "First owner" } }, presentation_mode: "WIZARD", version: 1 });

    await waitFor(() => expect(saveCaptureDraft).toHaveBeenCalledTimes(2));
    expect(screen.getByRole("heading", { name: "Company" })).toBeTruthy();
    expect(saveCaptureDraft).toHaveBeenLastCalledWith("session-secret", {
      answers: { owner: { text: "Current owner" } }, presentation_mode: "WIZARD", expected_version: 1,
    });

    finishSecond({ answers: { owner: { text: "Current owner" } }, presentation_mode: "WIZARD", version: 2 });
    expect(await screen.findByRole("heading", { name: "Evidence" })).toBeTruthy();
  });

  it("keeps Wizard entries on the current section when saving is unavailable", async () => {
    vi.mocked(loadCaptureDraft).mockResolvedValue({ answers: {}, presentation_mode: "WIZARD", version: 0 });
    vi.mocked(saveCaptureDraft).mockRejectedValue(new ApiError(503, "Draft unavailable", "draft_unavailable"));
    render(<CapturePanel request={wizardRequest} external sessionToken="session-secret"/>);

    const owner = await screen.findByRole("textbox", { name: /Current owner/ }) as HTMLInputElement;
    fireEvent.change(owner, { target: { value: "Treasury Technology" } });
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));

    expect(await screen.findByText("Could not save")).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Company" })).toBeTruthy();
    expect(owner.value).toBe("Treasury Technology");
    expect(screen.getByRole("button", { name: "Try again" })).toBeTruthy();
  });

  it("refreshes the draft version and preserves Wizard entries after a navigation conflict", async () => {
    vi.mocked(loadCaptureDraft)
      .mockResolvedValueOnce({ answers: {}, presentation_mode: "WIZARD", version: 2 })
      .mockResolvedValueOnce({ answers: { owner: { text: "Other tab value" } }, presentation_mode: "WIZARD", version: 7 });
    vi.mocked(saveCaptureDraft)
      .mockRejectedValueOnce(new ApiError(409, "Draft changed", "draft_conflict"))
      .mockResolvedValueOnce({ answers: { owner: { text: "Treasury Technology" } }, presentation_mode: "WIZARD", version: 8 });
    render(<CapturePanel request={wizardRequest} external sessionToken="session-secret"/>);

    const owner = await screen.findByRole("textbox", { name: /Current owner/ }) as HTMLInputElement;
    fireEvent.change(owner, { target: { value: "Treasury Technology" } });
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByText("Could not save")).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Company" })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(await screen.findByText("Saved")).toBeTruthy();
    expect(owner.value).toBe("Treasury Technology");
    expect(saveCaptureDraft).toHaveBeenLastCalledWith("session-secret", {
      answers: { owner: { text: "Treasury Technology" } }, presentation_mode: "WIZARD", expected_version: 7,
    });

    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByRole("heading", { name: "Evidence" })).toBeTruthy();
  });

  it("does not use response draft APIs for an internal request", async () => {
    render(<CapturePanel request={request}/>);
    fireEvent.change(screen.getByRole("textbox", { name: /Current owner/ }), { target: { value: "Treasury Technology" } });
    await new Promise((resolve) => setTimeout(resolve, 600));
    expect(loadCaptureDraft).not.toHaveBeenCalled();
    expect(saveCaptureDraft).not.toHaveBeenCalled();
    expect(screen.queryByText("Saving")).toBeNull();
  });
});
