import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { EvidenceRequestAdminContainer } from "./EvidenceRequestAdminContainer";

const { listEvidenceInvitationMetadata, issueEvidenceInvitation, replaceEvidenceInvitation, revokeEvidenceInvitation, revokeEvidenceSession } = vi.hoisted(() => ({
  listEvidenceInvitationMetadata: vi.fn(),
  issueEvidenceInvitation: vi.fn(),
  replaceEvidenceInvitation: vi.fn(),
  revokeEvidenceInvitation: vi.fn(),
  revokeEvidenceSession: vi.fn(),
}));

vi.mock("../evidenceRequestAdminApi", () => ({ listEvidenceInvitationMetadata, issueEvidenceInvitation, replaceEvidenceInvitation, revokeEvidenceInvitation, revokeEvidenceSession }));

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((next) => { resolve = next; });
  return { promise, resolve };
}

beforeEach(() => {
  vi.clearAllMocks();
  listEvidenceInvitationMetadata.mockResolvedValue([{ id: "invitation-1", request_id: "request-1", audience_hint: "m***@example.com", purpose: "Provide the return", expires_at: "2099-09-01T12:00:00Z", max_redemptions: 1, redemptions: 0, created_at: "2026-08-26T12:00:00Z" }]);
  revokeEvidenceInvitation.mockResolvedValue(undefined);
});

it("loads sanitized requester metadata and refreshes after revocation", async () => {
  render(<EvidenceRequestAdminContainer requestID="request-1" requestTitle="Annual return evidence"/>);

  expect(await screen.findByText("m***@example.com")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Revoke invitation for m***@example.com" }));

  await waitFor(() => expect(revokeEvidenceInvitation).toHaveBeenCalledWith("request-1", "invitation-1"));
  await waitFor(() => expect(listEvidenceInvitationMetadata).toHaveBeenCalledTimes(2));
  expect(screen.queryByText(/invitation-1|request-1/)).toBeNull();
});

it("keeps a created invitation available when the follow-up inventory refresh fails", async () => {
  listEvidenceInvitationMetadata.mockResolvedValueOnce([]).mockRejectedValueOnce(new Error("refresh unavailable"));
  issueEvidenceInvitation.mockResolvedValue({ invitation_id: "new-invitation", token: "one-time-token", audience_hint: "m***@example.com", expires_at: "2099-09-01T12:00:00Z" });
  render(<EvidenceRequestAdminContainer requestID="request-1" requestTitle="Annual return evidence"/>);

  await screen.findByText("No invitations have been issued for this evidence request.");
  fireEvent.change(screen.getByRole("textbox", { name: "Recipient email or approved audience" }), { target: { value: "manager@example.com" } });
  fireEvent.change(screen.getByRole("textbox", { name: "Invitation purpose" }), { target: { value: "Provide the return" } });
  fireEvent.click(screen.getByRole("button", { name: "Create invitation" }));

  expect(await screen.findByDisplayValue(/capture_invite=one-time-token/)).toBeTruthy();
  expect(screen.getByText("Issue time awaiting invitation history refresh")).toBeTruthy();
  expect((await screen.findByRole("alert")).textContent).toMatch(/Previously loaded records remain available, but changes are disabled/i);
  expect(screen.queryByText(/invitation could not be created/i)).toBeNull();
});

it("does not expose an old request token or records after the request scope changes", async () => {
  listEvidenceInvitationMetadata.mockResolvedValue([]);
  const command = deferred<{ invitation_id: string; token: string; audience_hint: string; expires_at: string }>();
  issueEvidenceInvitation.mockReturnValue(command.promise);
  const view = render(<EvidenceRequestAdminContainer requestID="request-1" requestTitle="First request"/>);

  await screen.findByText("No invitations have been issued for this evidence request.");
  fireEvent.change(screen.getByRole("textbox", { name: "Recipient email or approved audience" }), { target: { value: "first@example.com" } });
  fireEvent.change(screen.getByRole("textbox", { name: "Invitation purpose" }), { target: { value: "First request evidence" } });
  fireEvent.click(screen.getByRole("button", { name: "Create invitation" }));
  await waitFor(() => expect(issueEvidenceInvitation).toHaveBeenCalledWith("request-1", expect.anything()));

  view.rerender(<EvidenceRequestAdminContainer requestID="request-2" requestTitle="Second request"/>);
  await screen.findByRole("heading", { name: "Second request" });
  await act(async () => command.resolve({ invitation_id: "old-invitation", token: "old-one-time-token", audience_hint: "f***@example.com", expires_at: "2099-09-01T12:00:00Z" }));

  expect(screen.queryByDisplayValue(/old-one-time-token/)).toBeNull();
  expect(screen.queryByText("f***@example.com")).toBeNull();
});
