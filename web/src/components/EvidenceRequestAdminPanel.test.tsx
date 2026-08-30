import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import {
  EvidenceRequestAdminPanel,
  type EvidenceInvitationAdminItem,
  type EvidenceRequestAdminPanelProps,
} from "./EvidenceRequestAdminPanel";

it("issues an invitation from labelled recipient, purpose, and expiry controls and shows the token once", async () => {
  const issueInvitation = vi.fn().mockResolvedValue({
    invitation_id: "invitation-new",
    token: "sample-token",
    audience_hint: "a***@supplier.example",
    expires_at: "2026-09-02T12:00:00Z",
  });
  const storageWrite = vi.spyOn(Storage.prototype, "setItem");
  renderPanel({ issueInvitation });

  fireEvent.change(screen.getByRole("combobox", { name: "Recipient email or approved audience" }), { target: { value: "assurance@supplier.example" } });
  fireEvent.change(screen.getByRole("textbox", { name: "Invitation purpose" }), { target: { value: "Provide the quarter-end access review" } });
  fireEvent.change(screen.getByRole("combobox", { name: "Invitation expiry" }), { target: { value: "1440" } });
  fireEvent.click(screen.getByRole("button", { name: "Create invitation" }));

  await waitFor(() => expect(issueInvitation).toHaveBeenCalledWith({
    audience: "assurance@supplier.example",
    purpose: "Provide the quarter-end access review",
    ttlMinutes: 1440,
  }));
  expect(screen.getByRole("status").textContent).toMatch(/shown once.*not saved/i);
  expect(screen.getByDisplayValue(/#form_access=sample-token/)).toBeTruthy();
  expect(storageWrite).not.toHaveBeenCalled();

  fireEvent.click(screen.getByRole("button", { name: "Hide invitation link" }));
  expect(screen.queryByDisplayValue(/#form_access=sample-token/)).toBeNull();
});

it("accepts the approved external address when only a masked recipient is stored", async () => {
  const issueInvitation = vi.fn().mockResolvedValue({ invitation_id: "new", token: "token", audience_hint: "m***@example.com", expires_at: "2026-09-02T12:00:00Z" });
  renderPanel({ recipients: [], activeSessions: undefined, issueInvitation });

  fireEvent.change(screen.getByRole("textbox", { name: "Recipient email or approved audience" }), { target: { value: "manager@example.com" } });
  fireEvent.change(screen.getByRole("textbox", { name: "Invitation purpose" }), { target: { value: "Provide the requested evidence" } });
  fireEvent.click(screen.getByRole("button", { name: "Create invitation" }));

  await waitFor(() => expect(issueInvitation).toHaveBeenCalledWith({ audience: "manager@example.com", purpose: "Provide the requested evidence", ttlMinutes: 1440 }));
  expect(screen.queryByRole("heading", { name: "Active external sessions" })).toBeNull();
});

it("clears clipboard notices when the one-time link is hidden", async () => {
  const priorClipboard = navigator.clipboard;
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText: vi.fn().mockResolvedValue(undefined) } });
  try {
    renderPanel();
    fireEvent.change(screen.getByRole("combobox", { name: "Recipient email or approved audience" }), { target: { value: "assurance@supplier.example" } });
    fireEvent.change(screen.getByRole("textbox", { name: "Invitation purpose" }), { target: { value: "Provide the requested evidence" } });
    fireEvent.click(screen.getByRole("button", { name: "Create invitation" }));
    await screen.findByDisplayValue(/#form_access=token/);

    fireEvent.click(screen.getByRole("button", { name: "Copy invitation link" }));
    expect(await screen.findByText(/Invitation link copied/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Hide invitation link" }));

    expect(screen.queryByText(/Invitation link copied/)).toBeNull();
  } finally {
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: priorClipboard });
  }
});

it("shows sanitized metadata, replaces the active invitation as the dominant action, and never displays internal IDs", async () => {
  const replaceInvitation = vi.fn().mockResolvedValue({
    invitation_id: "replacement-id",
    token: "replacement-token",
    audience_hint: "a***@supplier.example",
    expires_at: "2099-09-03T12:00:00Z",
  });
  const revokeInvitation = vi.fn().mockResolvedValue(undefined);
  renderPanel({ invitations: [invitation()], replaceInvitation, revokeInvitation });

  expect(screen.getByRole("heading", { name: "Quarter-end privileged access review" })).toBeTruthy();
  expect(screen.getByText("a***@supplier.example")).toBeTruthy();
  expect(screen.getByText("Provide the quarter-end access review", { selector: "dd" })).toBeTruthy();
  expect(screen.getByText("Active", { selector: "strong" })).toBeTruthy();
  expect(screen.queryByText("invitation-internal-42")).toBeNull();
  expect(screen.queryByText(/request-internal-17|\{.*audience_hint/i)).toBeNull();
  expect(screen.getAllByRole("button", { name: "Replace invitation" })).toHaveLength(1);
  expect(document.querySelectorAll(".primary-button")).toHaveLength(1);

  fireEvent.click(screen.getByRole("button", { name: "Replace invitation" }));
  await waitFor(() => expect(replaceInvitation).toHaveBeenCalledWith("invitation-internal-42", {
    audience: "assurance@supplier.example",
    purpose: "Provide the quarter-end access review",
    ttlMinutes: 1440,
  }));

  fireEvent.click(screen.getByRole("button", { name: "Revoke invitation for a***@supplier.example" }));
  await waitFor(() => expect(revokeInvitation).toHaveBeenCalledWith("invitation-internal-42"));
});

it("revokes a labelled active external session without exposing its internal identifier", async () => {
  const revokeSession = vi.fn().mockResolvedValue(undefined);
  renderPanel({
    activeSessions: [{ id: "session-internal-9", audienceHint: "a***@supplier.example", expiresAt: "2099-08-28T12:00:00Z", startedAt: "2026-08-26T12:00:00Z" }],
    revokeSession,
  });

  expect(screen.getByRole("heading", { name: "Active external sessions" })).toBeTruthy();
  expect(screen.queryByText("session-internal-9")).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "End session for a***@supplier.example" }));
  await waitFor(() => expect(revokeSession).toHaveBeenCalledWith("session-internal-9"));
});

it("preserves bounded loaded records and disables every change when requester administration is degraded", () => {
  const invitations = Array.from({ length: 51 }, (_, index) => invitation({ id: `invitation-${index}`, audienceHint: `recipient-${index}@supplier.example`, revokedAt: "2026-08-26T13:00:00Z" }));
  renderPanel({
    invitations,
    loadState: "degraded",
    degradedReason: "Current requester authority could not be confirmed.",
    activeSessions: [{ id: "session-1", audienceHint: "session@supplier.example", expiresAt: "2099-08-28T12:00:00Z", startedAt: "2026-08-26T12:00:00Z" }],
  });

  expect(screen.getByText(/Current requester authority could not be confirmed.*loaded invitation records remain available.*changes are disabled/i)).toBeTruthy();
  expect(screen.getByText("Showing the first 50 invitations. More records are available.")).toBeTruthy();
  expect(screen.getByText("recipient-49@supplier.example")).toBeTruthy();
  expect(screen.queryByText("recipient-50@supplier.example")).toBeNull();
  for (const control of screen.getAllByRole("button")) expect((control as HTMLButtonElement).disabled).toBe(true);
  expect((screen.getByRole("combobox", { name: "Recipient email or approved audience" }) as HTMLSelectElement).disabled).toBe(true);
});

it("does not claim an empty population when the current administration read is unavailable", () => {
  renderPanel({ invitations: [], activeSessions: [], loadState: "unavailable" });

  expect(screen.getByText(/invitation history could not be loaded.*invitation changes are unavailable/i)).toBeTruthy();
  expect(screen.getByText(/active external sessions could not be loaded.*session changes are unavailable/i)).toBeTruthy();
  expect(screen.queryByText(/No invitations have been issued/)).toBeNull();
  expect(screen.queryByText(/No active external sessions/)).toBeNull();
  expect((screen.getByRole("button", { name: "Create invitation" }) as HTMLButtonElement).disabled).toBe(true);
});

function renderPanel(overrides: Partial<EvidenceRequestAdminPanelProps> = {}) {
  const props: EvidenceRequestAdminPanelProps = {
    requestTitle: "Quarter-end privileged access review",
    recipients: [{ label: "Acme assurance mailbox", audience: "assurance@supplier.example" }],
    invitations: [],
    activeSessions: [],
    canManage: true,
    loadState: "ready",
    issueInvitation: vi.fn().mockResolvedValue({ invitation_id: "new", token: "token", audience_hint: "a***@supplier.example", expires_at: "2026-09-02T12:00:00Z" }),
    replaceInvitation: vi.fn().mockResolvedValue({ invitation_id: "replacement", token: "token", audience_hint: "a***@supplier.example", expires_at: "2026-09-02T12:00:00Z" }),
    revokeInvitation: vi.fn().mockResolvedValue(undefined),
    revokeSession: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
  return render(<EvidenceRequestAdminPanel {...props}/>);
}

function invitation(overrides: Partial<EvidenceInvitationAdminItem> = {}): EvidenceInvitationAdminItem {
  return {
    id: "invitation-internal-42",
    audienceHint: "a***@supplier.example",
    purpose: "Provide the quarter-end access review",
    expiresAt: "2099-09-01T12:00:00Z",
    maxRedemptions: 1,
    redemptions: 0,
    issuedAt: "2026-08-26T12:00:00Z",
    ...overrides,
  };
}
