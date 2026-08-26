import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { loadCaptureSession, redeemCaptureInvitation } from "../captureApi";
import type { CaptureRequest } from "../types";
import { bootstrapExternalCapture, captureActiveSessionStorageKey, captureSessionStorageKey, ExternalCaptureApp } from "./ExternalCaptureApp";

vi.mock("../captureApi", () => ({ loadCaptureDraft: vi.fn().mockResolvedValue({ answers: {}, presentation_mode: "AUTOMATIC", version: 0 }), saveCaptureDraft: vi.fn(), loadCaptureSession: vi.fn(), redeemCaptureInvitation: vi.fn(), submitCaptureSession: vi.fn(), uploadCaptureSessionArtifact: vi.fn() }));

const request: CaptureRequest = {
  id: "field-request", title: "Verify ATM location after your visit", purpose: "Confirm that this ATM is present at the recorded address and provide one clear site photo.", why_you: "You were assigned to verify this location after a physical visit.", status: "READY", sensitivity: "INTERNAL", estimated_minutes: 3, deadline: "2027-08-09T12:00:00Z", known_facts: { expected_address: "12 Admiralty Way, Lekki Phase 1, Lagos" }, fields: [{ id: "present", label: "Is the ATM present?", type: "single_select", required: true, options: ["Yes", "No"] }], version: 1,
};

describe("ExternalCaptureApp", () => {
  beforeEach(() => {
    sessionStorage.clear();
    window.history.replaceState({}, "", "/");
    vi.clearAllMocks();
  });
  afterEach(() => vi.restoreAllMocks());

  it("removes the invitation token from the current history entry before capture renders", () => {
    window.history.replaceState({}, "", "/?capture_invite=full-invitation-secret&return=vendors#request");
    const historyLength = window.history.length;

    const bootstrap = bootstrapExternalCapture(window);

    expect(bootstrap.invitationToken).toBe("full-invitation-secret");
    expect(bootstrap.isExternalCapture).toBe(true);
    expect(window.location.search).toBe("?return=vendors&capture=1");
    expect(window.location.href).not.toContain("full-invitation-secret");
    expect(window.history.length).toBe(historyLength);
    render(<ExternalCaptureApp invitationToken={bootstrap.invitationToken} resumedSessionID={bootstrap.resumedSessionID}/>);
    expect(screen.getByRole("heading", { name: "Open your request" })).toBeTruthy();
  });

  it("opens a new invitation instead of resuming a different saved session", () => {
    sessionStorage.setItem(captureActiveSessionStorageKey, "prior-session");
    sessionStorage.setItem(captureSessionStorageKey("prior-session"), "prior-session-token");
    window.history.replaceState({}, "", "/?capture_invite=new-invitation");

    const bootstrap = bootstrapExternalCapture(window);

    expect(bootstrap.invitationToken).toBe("new-invitation");
    expect(bootstrap.resumedSessionID).toBeUndefined();
    expect(sessionStorage.getItem(captureActiveSessionStorageKey)).toBeNull();
    expect(sessionStorage.getItem(captureSessionStorageKey("prior-session"))).toBeNull();
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

  it("stores the returned session by session identity and resumes after the cleaned URL is refreshed", async () => {
    window.history.replaceState({}, "", "/?capture_invite=invite-secret-material");
    const bootstrap = bootstrapExternalCapture(window);
    vi.mocked(redeemCaptureInvitation).mockResolvedValue({ session_id: "session-1", session_token: "session-token-1", request_id: request.id, audience_hint: "f***@example.com", expires_at: request.deadline });
    vi.mocked(loadCaptureSession).mockResolvedValue({ session: { id: "session-1", request_id: request.id, audience_hint: "f***@example.com", expires_at: request.deadline }, request });
    const first = render(<ExternalCaptureApp invitationToken={bootstrap.invitationToken} resumedSessionID={bootstrap.resumedSessionID}/>);

    fireEvent.change(screen.getByRole("textbox", { name: "Email or phone number" }), { target: { value: "field.agent@example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Open request" }));
    expect(await screen.findByRole("heading", { name: request.title })).toBeTruthy();

    expect(sessionStorage.getItem(captureActiveSessionStorageKey)).toBe("session-1");
    expect(sessionStorage.getItem(captureSessionStorageKey("session-1"))).toBe("session-token-1");
    const storedKeys = Array.from({ length: sessionStorage.length }, (_, index) => sessionStorage.key(index) ?? "");
    expect(storedKeys).not.toContain(expect.stringContaining("invite-secret-material"));
    expect(storedKeys).not.toContain(expect.stringContaining("secret-material"));

    first.unmount();
    vi.mocked(loadCaptureSession).mockClear();
    const refreshed = bootstrapExternalCapture(window);
    expect(refreshed.invitationToken).toBeUndefined();
    expect(refreshed.resumedSessionID).toBe("session-1");
    render(<ExternalCaptureApp invitationToken={refreshed.invitationToken} resumedSessionID={refreshed.resumedSessionID}/>);

    expect(await screen.findByRole("heading", { name: request.title })).toBeTruthy();
    expect(loadCaptureSession).toHaveBeenCalledWith("session-token-1");
    expect(window.location.search).toBe("?capture=1");
  });

  it("opens the redeemed request when browser session storage is unavailable", async () => {
    vi.mocked(redeemCaptureInvitation).mockResolvedValue({ session_id: "session-1", session_token: "session-token-1", request_id: request.id, audience_hint: "f***@example.com", expires_at: request.deadline });
    vi.mocked(loadCaptureSession).mockResolvedValue({ session: { id: "session-1", request_id: request.id, audience_hint: "f***@example.com", expires_at: request.deadline }, request });
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new DOMException("Storage blocked", "SecurityError"); });
    render(<ExternalCaptureApp invitationToken="invite-1"/>);

    fireEvent.change(screen.getByRole("textbox", { name: "Email or phone number" }), { target: { value: "field.agent@example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Open request" }));

    expect(await screen.findByRole("heading", { name: request.title })).toBeTruthy();
    expect(loadCaptureSession).toHaveBeenCalledWith("session-token-1");
  });
});
