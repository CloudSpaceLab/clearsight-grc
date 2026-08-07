import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { submitCaptureRequest } from "../api";
import { ApiError } from "../http";
import type { CaptureRequest } from "../types";
import { CapturePanel } from "./CapturePanel";

vi.mock("../api", () => ({ submitCaptureRequest: vi.fn() }));

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

describe("CapturePanel", () => {
  it("uses a short input and reviews the exact response before submitting", async () => {
    vi.mocked(submitCaptureRequest).mockResolvedValue({ request_id: request.id, status: "SUBMITTED", submitted_at: "2026-08-06T19:30:00Z" });
    render(<CapturePanel request={request}/>);

    const owner = screen.getByRole("textbox", { name: /Current owner/ }) as HTMLInputElement;
    expect(owner.tagName).toBe("INPUT");
    fireEvent.change(owner, { target: { value: "Treasury Technology" } });
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));

    expect(screen.getByRole("heading", { name: "Check your response" })).toBeTruthy();
    expect(screen.getByText("Treasury Technology")).toBeTruthy();
    expect(submitCaptureRequest).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "Submit response" }));
    await waitFor(() => expect(submitCaptureRequest).toHaveBeenCalledWith(request.id, 3, { owner: "Treasury Technology" }));
    expect(await screen.findByRole("heading", { name: "Response submitted" })).toBeTruthy();
    expect(screen.getByText(/evidence quality is reviewed separately/i)).toBeTruthy();
  });

  it("uses the native date control and preserves multiple answers", () => {
    render(<CapturePanel request={multiFieldRequest}/>);

    fireEvent.change(screen.getByRole("textbox", { name: /Processor register owner/ }), { target: { value: "Privacy Operations" } });
    const date = screen.getByLabelText(/DPCO review date/) as HTMLInputElement;
    expect(date.type).toBe("date");
    fireEvent.change(date, { target: { value: "2027-03-01" } });

    const review = screen.getByRole("button", { name: "Review and submit" }) as HTMLButtonElement;
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
    expect((screen.getByRole("button", { name: "Review and submit" }) as HTMLButtonElement).disabled).toBe(false);
  });

  it("uploads a photo and reviews it without exposing the artifact id", async () => {
    const upload = vi.fn().mockResolvedValue({ id: "artifact-secret-id", request_id: request.id, file_name: "atm.jpg", media_type: "image/jpeg", size_bytes: 1200, sha256: "hash", status: "STORED_UNSCANNED" });
    render(<CapturePanel request={{ ...request, fields: [{ id: "photo", label: "Site photo", type: "photo", required: true, accepted_formats: ["image/jpeg"] }] }} onUploadArtifact={upload}/>);
    const input = screen.getByLabelText(/Site photo/) as HTMLInputElement;
    const file = new File(["photo"], "atm.jpg", { type: "image/jpeg" });
    fireEvent.change(input, { target: { files: [file] } });
    await waitFor(() => expect(upload).toHaveBeenCalledWith(request.id, file));
    expect(screen.getByText(/atm\.jpg/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
    expect(screen.getByText(/Photo attached · atm.jpg/)).toBeTruthy();
    expect(screen.queryByText("artifact-secret-id")).toBeNull();
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
    expect((await screen.findByRole("alert")).textContent).toMatch(/previous attachment is still selected/i);
    expect(screen.getByText(/original\.jpg/)).toBeTruthy();

    const review = screen.getByRole("button", { name: "Review and submit" }) as HTMLButtonElement;
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
    expect((screen.getByRole("button", { name: "Review and submit" }) as HTMLButtonElement).disabled).toBe(true);
  });

  it("ignores a submission completion after the active request changes", async () => {
    let resolveSubmit!: (value: { submitted_at: string }) => void;
    const submit = vi.fn().mockImplementation(() => new Promise((resolve) => { resolveSubmit = resolve; }));
    const { rerender } = render(<CapturePanel request={request} onSubmit={submit}/>);

    fireEvent.change(screen.getByRole("textbox", { name: /Current owner/ }), { target: { value: "Treasury Technology" } });
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
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
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
    fireEvent.click(screen.getByRole("button", { name: "Submit verification" }));

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
    expect(screen.queryByRole("button", { name: "Review and submit" })).toBeNull();
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
    vi.mocked(submitCaptureRequest).mockRejectedValue(new ApiError(409, "The request changed", "version_conflict"));
    render(<CapturePanel request={request} onReload={reload}/>);
    fireEvent.change(screen.getByRole("textbox", { name: /Current owner/ }), { target: { value: "Treasury Technology" } });
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
    fireEvent.click(screen.getByRole("button", { name: "Submit response" }));
    expect((await screen.findByRole("alert")).textContent).toMatch(/changed while you were working/i);
    fireEvent.click(screen.getByRole("button", { name: "Reload request" }));
    expect(reload).toHaveBeenCalledTimes(1);
  });

  it("fails closed for a genuinely unknown field contract", () => {
    render(<CapturePanel request={{ ...request, fields: [{ id: "unknown", label: "Unrecognized field", type: "biometric_scan", required: true }] }}/>);
    expect(screen.getByRole("alert").textContent).toMatch(/cannot safely collect/i);
    expect((screen.getByRole("button", { name: "Review and submit" }) as HTMLButtonElement).disabled).toBe(true);
  });
});
