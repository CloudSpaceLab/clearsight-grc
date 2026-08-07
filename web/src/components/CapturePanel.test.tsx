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
    expect(screen.getByText(/Mar 1, 2027|1 Mar 2027|Mar 1, 2027/)).toBeTruthy();
  });

  it("uses large tap choices for short option lists", () => {
    render(<CapturePanel request={{ ...request, fields: [{ id: "present", label: "Is the ATM present?", type: "single_select", required: true, options: ["Yes", "No"] }] }}/>) ;
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
    expect(screen.getByText("atm.jpg")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
    expect(screen.getByText(/Photo attached · atm.jpg/)).toBeTruthy();
    expect(screen.queryByText("artifact-secret-id")).toBeNull();
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
    render(<CapturePanel request={{ ...request, fields: [{ id: "unknown", label: "Unrecognized field", type: "biometric_scan", required: true }] }}/>) ;
    expect(screen.getByRole("alert").textContent).toMatch(/cannot safely collect/i);
    expect((screen.getByRole("button", { name: "Review and submit" }) as HTMLButtonElement).disabled).toBe(true);
  });
});
