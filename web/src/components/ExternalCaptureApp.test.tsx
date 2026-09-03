import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  loadFormResponseWorkspace,
  redeemFormAccess,
  saveFormResponseWorkspace,
  sendFormAccessOTP,
  startFormAccess,
  submitFormResponseWorkspace,
  verifyFormAccessOTP,
} from "../captureApi";
import { consumeCaptureInvitation, purgeLegacyCaptureSession } from "../captureInvitationBrowser";
import { ApiError } from "../http";
import type { CaptureRequest } from "../types";
import { ExternalCaptureApp } from "./ExternalCaptureApp";

vi.mock("../captureApi", () => ({
  FormWorkspaceConflictError: class FormWorkspaceConflictError extends Error {
    conflict: unknown;
    constructor(conflict: unknown) { super("workspace conflict"); this.conflict = conflict; }
  },
  loadCaptureDraft: vi.fn(),
  saveCaptureDraft: vi.fn(),
  submitInternalCaptureRequest: vi.fn(),
  uploadInternalCaptureArtifact: vi.fn(),
  startFormAccess: vi.fn(),
  sendFormAccessOTP: vi.fn(),
  verifyFormAccessOTP: vi.fn(),
  redeemFormAccess: vi.fn(),
  loadFormResponseWorkspace: vi.fn(),
  saveFormResponseWorkspace: vi.fn(),
  submitFormResponseWorkspace: vi.fn(),
  uploadCaptureSessionArtifact: vi.fn(),
  normalizeCaptureAnswer: (value: unknown) => typeof value === "string" ? { text: value } : value,
  normalizeCaptureAnswers: (answers: Record<string, unknown>) => Object.fromEntries(Object.entries(answers).map(([fieldID, value]) => [fieldID, typeof value === "string" ? { text: value } : value])),
}));

const request: CaptureRequest = {
  id: "field-request",
  title: "Verify ATM location after your visit",
  purpose: "Confirm that this ATM is present at the recorded address.",
  why_you: "You were assigned to verify this location after a physical visit.",
  audience_type: "EXTERNAL",
  status: "READY",
  sensitivity: "INTERNAL",
  estimated_minutes: 3,
  deadline: "2027-08-09T12:00:00Z",
  known_facts: { expected_address: "12 Admiralty Way, Lekki Phase 1, Lagos" },
  fields: [{ id: "present", label: "Is the ATM present?", type: "single_select", required: true, options: ["Yes", "No"] }],
  version: 1,
};

const directSession = {
  session_id: "session-1",
  session_token: "session-secret",
  distribution_id: "distribution-1",
  request_id: request.id,
  audience_hint: "f***@example.com",
  assurance: "LINK_POSSESSION" as const,
  expires_at: request.deadline,
};

const workspace = {
  workspace: {
    id: "workspace-1",
    distribution_id: "distribution-1",
    status: "OPEN" as const,
    version: 0,
    created_at: "2026-08-28T12:00:00Z",
    updated_at: "2026-08-28T12:00:00Z",
  },
  answers: {},
  presentation_mode: "AUTOMATIC" as const,
  field_sequences: {},
};

function workspacePayload(session = directSession, currentWorkspace = workspace) {
  return {
    session: {
      id: session.session_id,
      distribution_id: session.distribution_id,
      request_id: session.request_id,
      audience_hint: session.audience_hint,
      assurance: session.assurance,
      expires_at: session.expires_at,
      created_at: "2026-08-28T12:00:00Z",
    },
    request,
    workspace: currentWorkspace,
    recovery_context: {
      legal_entity_id: "entity-1",
      distribution_id: session.distribution_id,
      schema_version: 1,
      route_expires_at: session.expires_at,
    },
  };
}

describe("ExternalCaptureApp", () => {
  beforeEach(() => {
    sessionStorage.clear();
    window.history.replaceState({}, "", "/");
    vi.clearAllMocks();
    vi.mocked(loadFormResponseWorkspace).mockResolvedValue(workspacePayload());
  });

  it("consumes the invitation without leaving it in browser history or changing unrelated URL state", () => {
    window.history.replaceState({ returnTo: "request" }, "", "/capture?fixture=external&mode=compact#view=response&form_access=secret-token");

    expect(consumeCaptureInvitation(window)).toBe("secret-token");
    expect(window.location.pathname).toBe("/capture");
    expect(window.location.search).toBe("?fixture=external&mode=compact");
    expect(window.location.hash).toBe("#view=response");
    expect(window.history.state).toEqual({ returnTo: "request" });
  });

  it("consumes legacy query invitations while preserving unrelated URL state", () => {
    window.history.replaceState({}, "", "/capture?fixture=external&capture_invite=legacy-token#response");

    expect(consumeCaptureInvitation(window)).toBe("legacy-token");
    expect(window.location.search).toBe("?fixture=external");
    expect(window.location.hash).toBe("#response");
  });

  it("purges bearer material written by the legacy browser-session flow", () => {
    sessionStorage.setItem("clearsight.capture.active-session", "opaque-locator");
    sessionStorage.setItem("clearsight.capture.session.opaque-locator", "legacy-session-secret");

    purgeLegacyCaptureSession(sessionStorage);

    expect(Object.keys(sessionStorage)).toEqual([]);
  });

  it("opens a direct magic-link request without audience typing or persistent bearer storage", async () => {
    vi.mocked(startFormAccess).mockResolvedValue({ policy: "DIRECT_MAGIC_LINK", expires_at: request.deadline });
    vi.mocked(redeemFormAccess).mockResolvedValue(directSession);

    render(<ExternalCaptureApp invitationToken="route-secret"/>);

    expect(await screen.findByRole("heading", { name: request.title })).toBeTruthy();
    expect(startFormAccess).toHaveBeenCalledWith("route-secret");
    expect(redeemFormAccess).toHaveBeenCalledWith("route-secret");
    expect(loadFormResponseWorkspace).toHaveBeenCalledWith("session-secret");
    expect(screen.queryByRole("textbox", { name: /email|phone/i })).toBeNull();
    expect(Object.keys(sessionStorage)).toEqual([]);
  });

  it("uses only server-provided masked recipients for a shared-link OTP journey", async () => {
    vi.mocked(startFormAccess).mockResolvedValue({
      policy: "SHARED_LINK_EMAIL_OTP",
      expires_at: request.deadline,
      recipients: [
        { selector_id: "selector-a", hint: "a***@vendor.example", contact_label: "Accounts payable" },
        { selector_id: "selector-b", hint: "s***@vendor.example", contact_label: "Security lead" },
      ],
    });
    vi.mocked(sendFormAccessOTP).mockResolvedValue({ challenge_id: "challenge-1", hint: "s***@vendor.example", expires_at: request.deadline });
    vi.mocked(verifyFormAccessOTP).mockResolvedValue({ ...directSession, assurance: "EMAIL_VERIFIED", audience_hint: "s***@vendor.example" });

    render(<ExternalCaptureApp invitationToken="shared-route"/>);

    expect(await screen.findByRole("heading", { name: "Choose your invitation" })).toBeTruthy();
    expect(screen.getByText("a***@vendor.example")).toBeTruthy();
    expect(screen.getByText("s***@vendor.example")).toBeTruthy();
    expect(screen.queryByRole("textbox", { name: /email|phone/i })).toBeNull();

    fireEvent.click(screen.getByRole("radio", { name: /Security lead/i }));
    fireEvent.click(screen.getByRole("button", { name: "Send code" }));
    await waitFor(() => expect(sendFormAccessOTP).toHaveBeenCalledWith("shared-route", "selector-b"));

    const input = await screen.findByRole("textbox", { name: "Verification code" });
    expect(input.getAttribute("autocomplete")).toBe("one-time-code");
    fireEvent.change(input, { target: { value: "123456" } });
    fireEvent.click(screen.getByRole("button", { name: "Verify and open" }));

    await waitFor(() => expect(verifyFormAccessOTP).toHaveBeenCalledWith("shared-route", "challenge-1", "123456"));
    expect(await screen.findByRole("heading", { name: request.title })).toBeTruthy();
    expect(Object.keys(sessionStorage)).toEqual([]);
  });

  it("skips recipient selection for a direct-link OTP and sends to the sole masked recipient", async () => {
    vi.mocked(startFormAccess).mockResolvedValue({
      policy: "DIRECT_LINK_EMAIL_OTP",
      expires_at: request.deadline,
      recipients: [{ selector_id: "selector-direct", hint: "f***@example.com", contact_label: "Field agent" }],
    });
    vi.mocked(sendFormAccessOTP).mockResolvedValue({ challenge_id: "challenge-direct", hint: "f***@example.com", expires_at: request.deadline });

    render(<ExternalCaptureApp invitationToken="direct-otp-route"/>);

    expect(await screen.findByRole("heading", { name: "Enter the verification code" })).toBeTruthy();
    expect(sendFormAccessOTP).toHaveBeenCalledWith("direct-otp-route", "selector-direct");
    expect(screen.queryByRole("radio")).toBeNull();
  });

  it.each([
    ["otp_expired", 410, "Verification code expired", "Request another code to continue."],
    ["otp_attempts_exhausted", 429, "Verification attempts used", "Ask the sender for a new invitation link."],
  ])("explains the terminal %s OTP result without exposing provider detail", async (code, status, title, recovery) => {
    vi.mocked(startFormAccess).mockResolvedValue({
      policy: "DIRECT_LINK_EMAIL_OTP",
      expires_at: request.deadline,
      recipients: [{ selector_id: "selector-direct", hint: "f***@example.com", contact_label: "Field agent" }],
    });
    vi.mocked(sendFormAccessOTP).mockResolvedValue({ challenge_id: "challenge-direct", hint: "f***@example.com", expires_at: request.deadline });
    vi.mocked(verifyFormAccessOTP).mockRejectedValue(new ApiError(status, "provider diagnostic", code));

    render(<ExternalCaptureApp invitationToken="direct-otp-route"/>);

    const input = await screen.findByRole("textbox", { name: "Verification code" });
    fireEvent.change(input, { target: { value: "123456" } });
    fireEvent.click(screen.getByRole("button", { name: "Verify and open" }));

    expect(await screen.findByRole("heading", { name: title })).toBeTruthy();
    expect(screen.getByText(recovery)).toBeTruthy();
    expect(screen.queryByText("provider diagnostic")).toBeNull();
  });

  it("writes the final changed answers to the shared workspace before submission", async () => {
    vi.mocked(startFormAccess).mockResolvedValue({ policy: "DIRECT_MAGIC_LINK", expires_at: request.deadline });
    vi.mocked(redeemFormAccess).mockResolvedValue(directSession);
    vi.mocked(saveFormResponseWorkspace).mockResolvedValue({
      ...workspace,
      answers: { present: { text: "Yes" } },
      field_sequences: { present: 1 },
      workspace: { ...workspace.workspace, version: 1 },
    });
    vi.mocked(submitFormResponseWorkspace).mockResolvedValue({
      workspace: { ...workspace.workspace, status: "COMPLETED", version: 2 },
      revision: { revision: 1, current: true },
      submission: { request_id: request.id, submission_id: "submission-1", status: "SUBMITTED", submitted_at: "2026-08-28T13:00:00Z", version: 2 },
    });

    render(<ExternalCaptureApp invitationToken="route-secret"/>);

    fireEvent.click(await screen.findByRole("radio", { name: "Yes" }));
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
    fireEvent.click(await screen.findByRole("button", { name: "Submit evidence" }));

    await waitFor(() => expect(saveFormResponseWorkspace).toHaveBeenCalledWith("session-secret", {
      expected_version: 0,
      presentation_mode: "AUTOMATIC",
      edits: [{ field_id: "present", value: { text: "Yes" }, base_sequence: 0 }],
    }));
    expect(submitFormResponseWorkspace).toHaveBeenCalledWith("session-secret", { expected_version: 1 });
    expect(await screen.findByRole("heading", { name: "Submitted" })).toBeTruthy();
    expect(Object.keys(sessionStorage)).toEqual([]);
  });

  it("reloads the current workspace after a genuine submission conflict", async () => {
    vi.mocked(startFormAccess).mockResolvedValue({ policy: "DIRECT_MAGIC_LINK", expires_at: request.deadline });
    vi.mocked(redeemFormAccess).mockResolvedValue(directSession);
    vi.mocked(saveFormResponseWorkspace).mockResolvedValue({
      ...workspace,
      answers: { present: { text: "Yes" } },
      field_sequences: { present: 1 },
      workspace: { ...workspace.workspace, version: 1 },
    });
    vi.mocked(submitFormResponseWorkspace).mockRejectedValue(
      new ApiError(409, "The response workspace changed.", "workspace_conflict"),
    );

    render(<ExternalCaptureApp invitationToken="route-secret"/>);

    fireEvent.click(await screen.findByRole("radio", { name: "Yes" }));
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
    fireEvent.click(await screen.findByRole("button", { name: "Submit evidence" }));

    const reload = await screen.findByRole("button", { name: "Reload request" });
    fireEvent.click(reload);

    await waitFor(() => expect(loadFormResponseWorkspace).toHaveBeenCalledTimes(2));
    expect(await screen.findByRole("heading", { name: request.title })).toBeTruthy();
  });

  it("fails closed with generic copy when the route cannot be started", async () => {
    vi.mocked(startFormAccess).mockRejectedValue(new Error("unknown route"));

    render(<ExternalCaptureApp invitationToken="bad-route"/>);

    expect(await screen.findByRole("heading", { name: "This request is no longer available" })).toBeTruthy();
    expect(screen.getByText("This invitation could not be opened. Ask the sender for a new link.")).toBeTruthy();
    expect(screen.queryByText(/unknown route/i)).toBeNull();
  });
});
