import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadCaptureSession, redeemCaptureInvitation, submitCaptureSession } from "../captureApi";
import { consumeCaptureInvitation, EXTERNAL_CAPTURE_SESSION_KEY } from "../captureInvitationBrowser";
import { ApiError } from "../http";
import type { CaptureRequest } from "../types";
import { ExternalCaptureApp } from "./ExternalCaptureApp";

vi.mock("../captureApi", () => ({ loadCaptureSession: vi.fn(), redeemCaptureInvitation: vi.fn(), submitCaptureSession: vi.fn(), uploadCaptureSessionArtifact: vi.fn() }));

const request: CaptureRequest = {
  id: "field-request", title: "Verify ATM location after your visit", purpose: "Confirm that this ATM is present at the recorded address and provide one clear site photo.", why_you: "You were assigned to verify this location after a physical visit.", status: "READY", sensitivity: "INTERNAL", estimated_minutes: 3, deadline: "2027-08-09T12:00:00Z", known_facts: { expected_address: "12 Admiralty Way, Lekki Phase 1, Lagos" }, fields: [{ id: "present", label: "Is the ATM present?", type: "single_select", required: true, options: ["Yes", "No"] }], version: 1,
};

describe("ExternalCaptureApp", () => {
  beforeEach(() => {
    sessionStorage.clear();
    window.history.replaceState({}, "", "/");
    vi.clearAllMocks();
  });

  it("consumes the invitation without leaving it in browser history or changing unrelated URL state", () => {
    window.history.replaceState({ returnTo: "request" }, "", "/capture?fixture=external&capture_invite=secret-token&mode=compact#response");

    expect(consumeCaptureInvitation(window)).toBe("secret-token");
    expect(window.location.pathname).toBe("/capture");
    expect(window.location.search).toBe("?fixture=external&mode=compact");
    expect(window.location.hash).toBe("#response");
    expect(window.history.state).toEqual({ returnTo: "request" });
  });

  it("resumes from one fixed tab-scoped key that contains no invitation token characters", async () => {
    sessionStorage.setItem(EXTERNAL_CAPTURE_SESSION_KEY, "session-token");
    vi.mocked(loadCaptureSession).mockResolvedValue({ session: { id: "session-1", request_id: request.id, audience_hint: "f***@example.com", expires_at: request.deadline }, request });

    render(<ExternalCaptureApp invitationToken=""/>);

    expect(await screen.findByRole("heading", { name: request.title })).toBeTruthy();
    expect(loadCaptureSession).toHaveBeenCalledWith("session-token");
    expect(Object.keys(sessionStorage)).toEqual([EXTERNAL_CAPTURE_SESSION_KEY]);
    expect(EXTERNAL_CAPTURE_SESSION_KEY).not.toContain("invite");
  });

  it("redeems the invite with the addressed identity and opens only the linked request", async () => {
    vi.mocked(redeemCaptureInvitation).mockResolvedValue({ session_id: "session-1", session_token: "token-1", request_id: request.id, audience_hint: "f***@example.com", expires_at: request.deadline });
    vi.mocked(loadCaptureSession).mockResolvedValue({ session: { id: "session-1", request_id: request.id, audience_hint: "f***@example.com", expires_at: request.deadline }, request });
    render(<ExternalCaptureApp invitationToken="invite-1"/>);

    expect(screen.getByRole("heading", { name: "Open your request" })).toBeTruthy();
    fireEvent.change(screen.getByRole("textbox", { name: "Email or phone number" }), { target: { value: "field.agent@example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Open request" }));

    await waitFor(() => expect(redeemCaptureInvitation).toHaveBeenCalledWith("invite-1", "field.agent@example.com"));
    expect(await screen.findByRole("heading", { name: request.title })).toBeTruthy();
    expect(screen.getByText("12 Admiralty Way, Lekki Phase 1, Lagos")).toBeTruthy();
    expect(screen.queryByRole("textbox", { name: /address/i })).toBeNull();
    expect(screen.queryByText(/do not need a ClearSight account/i)).toBeNull();
  });

  it("clears the stored external session after a successful submission", async () => {
    vi.mocked(redeemCaptureInvitation).mockResolvedValue({ session_id: "session-1", session_token: "session-token", request_id: request.id, audience_hint: "f***@example.com", expires_at: request.deadline });
    vi.mocked(loadCaptureSession).mockResolvedValue({ session: { id: "session-1", request_id: request.id, audience_hint: "f***@example.com", expires_at: request.deadline }, request });
    vi.mocked(submitCaptureSession).mockResolvedValue({ request_id: request.id, status: "SUBMITTED", submitted_at: "2026-08-26T10:00:00Z" });
    render(<ExternalCaptureApp invitationToken="secret-invitation"/>);

    fireEvent.change(screen.getByRole("textbox", { name: "Email or phone number" }), { target: { value: "field.agent@example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Open request" }));
    fireEvent.click(await screen.findByRole("radio", { name: "Yes" }));
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
    fireEvent.click(screen.getByRole("button", { name: "Submit evidence" }));

    await waitFor(() => expect(submitCaptureSession).toHaveBeenCalledWith("session-token", request.version, { present: "Yes" }));
    expect(await screen.findByRole("heading", { name: "Submitted" })).toBeTruthy();
    expect(sessionStorage.getItem(EXTERNAL_CAPTURE_SESSION_KEY)).toBeNull();
  });

  it.each([
    new ApiError(401, "Session unavailable", "session_unavailable"),
    new ApiError(409, "Request closed", "request_closed"),
  ])("clears the stored external session when submission reports revocation or expiry", async (failure) => {
    sessionStorage.setItem(EXTERNAL_CAPTURE_SESSION_KEY, "ended-session");
    vi.mocked(loadCaptureSession).mockResolvedValue({ session: { id: "session-1", request_id: request.id, audience_hint: "f***@example.com", expires_at: request.deadline }, request });
    vi.mocked(submitCaptureSession).mockRejectedValue(failure);
    render(<ExternalCaptureApp invitationToken=""/>);

    fireEvent.click(await screen.findByRole("radio", { name: "Yes" }));
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
    fireEvent.click(screen.getByRole("button", { name: "Submit evidence" }));

    await waitFor(() => expect(submitCaptureSession).toHaveBeenCalled());
    await waitFor(() => expect(sessionStorage.getItem(EXTERNAL_CAPTURE_SESSION_KEY)).toBeNull());
  });

  it("clears a revoked or otherwise unrecoverable saved session", async () => {
    sessionStorage.setItem(EXTERNAL_CAPTURE_SESSION_KEY, "revoked-session");
    vi.mocked(loadCaptureSession).mockRejectedValue(new ApiError(401, "Session expired", "session_expired"));

    render(<ExternalCaptureApp invitationToken=""/>);

    await waitFor(() => expect(loadCaptureSession).toHaveBeenCalledWith("revoked-session"));
    await waitFor(() => expect(sessionStorage.getItem(EXTERNAL_CAPTURE_SESSION_KEY)).toBeNull());
  });

  it("clears a saved session when the loaded request is already expired", async () => {
    sessionStorage.setItem(EXTERNAL_CAPTURE_SESSION_KEY, "expired-session");
    vi.mocked(loadCaptureSession).mockResolvedValue({
      session: { id: "session-1", request_id: request.id, audience_hint: "f***@example.com", expires_at: "2026-08-25T10:00:00Z" },
      request: { ...request, status: "EXPIRED", deadline: "2026-08-25T10:00:00Z" },
    });

    render(<ExternalCaptureApp invitationToken=""/>);

    expect(await screen.findByRole("heading", { name: "This request has expired" })).toBeTruthy();
    expect(sessionStorage.getItem(EXTERNAL_CAPTURE_SESSION_KEY)).toBeNull();
  });
});
