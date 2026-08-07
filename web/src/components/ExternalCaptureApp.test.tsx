import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadCaptureSession, redeemCaptureInvitation } from "../captureApi";
import type { CaptureRequest } from "../types";
import { ExternalCaptureApp } from "./ExternalCaptureApp";

vi.mock("../captureApi", () => ({ loadCaptureSession: vi.fn(), redeemCaptureInvitation: vi.fn(), submitCaptureSession: vi.fn(), uploadCaptureSessionArtifact: vi.fn() }));

const request: CaptureRequest = {
  id: "field-request", title: "Verify ATM location after your visit", purpose: "Confirm that this ATM is present at the recorded address and provide one clear site photo.", why_you: "You were assigned to verify this location after a physical visit.", status: "READY", sensitivity: "INTERNAL", estimated_minutes: 3, deadline: "2027-08-09T12:00:00Z", known_facts: { expected_address: "12 Admiralty Way, Lekki Phase 1, Lagos" }, fields: [{ id: "present", label: "Is the ATM present?", type: "single_select", required: true, options: ["Yes", "No"] }], version: 1,
};

describe("ExternalCaptureApp", () => {
  beforeEach(() => sessionStorage.clear());

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
});
