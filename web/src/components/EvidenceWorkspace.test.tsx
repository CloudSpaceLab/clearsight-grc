import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { declareWrongCaptureRecipient, reassignCaptureRecipient } from "../captureApi";
import type { EvidenceRequest } from "../types";
import { EvidenceWorkspace } from "./EvidenceWorkspace";

const { listEvidenceRecipientCandidates } = vi.hoisted(() => ({ listEvidenceRecipientCandidates: vi.fn() }));

vi.mock("../captureApi", () => ({
  declareWrongCaptureRecipient: vi.fn(),
  reassignCaptureRecipient: vi.fn(),
}));
vi.mock("../evidenceRequestAdminApi", async (importOriginal) => ({
  ...await importOriginal<typeof import("../evidenceRequestAdminApi")>(),
  listEvidenceRecipientCandidates,
}));

beforeAll(() => {
  Object.defineProperty(Element.prototype, "scrollIntoView", { configurable: true, value: vi.fn() });
});

beforeEach(() => {
  listEvidenceRecipientCandidates.mockRejectedValue(new Error("Recipient candidates not configured"));
});

function request(overrides: Partial<EvidenceRequest> = {}): EvidenceRequest {
  return {
    id: "request-1",
    tenant_id: "bank-1",
    subject_type: "PROGRAM",
    subject_id: "program-1",
    title: "Confirm account review evidence",
    purpose: "Confirm the completed account review for this Program.",
    why_you: "The current recipient owns the account review evidence.",
    sensitivity: "INTERNAL",
    audience_type: "INTERNAL",
    recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "recipient-1", state: "ASSIGNED" },
    created_by: "requester-1",
    estimated_minutes: 3,
    deadline: "2026-08-30T12:00:00Z",
    known_facts: { Population: "Privileged accounts" },
    fields: [],
    status: "READY",
    version: 1,
    created_at: "2026-08-25T12:00:00Z",
    updated_at: "2026-08-25T12:00:00Z",
    ...overrides,
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((next) => { resolve = next; });
  return { promise, resolve };
}

function renderWorkspace(evidenceRequest: EvidenceRequest, actorPrincipalID?: string) {
  const onOpenRequest = vi.fn();
  const onRequestUpdated = vi.fn().mockReturnValue(true);
  render(
    <EvidenceWorkspace
      sources={[]}
      requests={[evidenceRequest]}
      sourceState="live"
      requestState="live"
      targetID={evidenceRequest.id}
      actorPrincipalID={actorPrincipalID}
      evidenceScopeToken={1}
      onOpenRequest={onOpenRequest}
      onRequestUpdated={onRequestUpdated}
    />,
  );
  return { onOpenRequest, onRequestUpdated };
}

describe("EvidenceWorkspace response authority", () => {
  it("does not offer the requester a response control when another person is assigned", () => {
    renderWorkspace(request(), "requester-1");

    expect(screen.queryByRole("button", { name: "Open request" })).toBeNull();
    expect(screen.getByText("Only the assigned person can respond to this request.")).toBeTruthy();
    expect(screen.getByText("You created this request. Change the recipient if the assignment is wrong.")).toBeTruthy();
    expect(screen.queryByText(/recipient-1/)).toBeNull();
  });

  it("does not offer an unrelated internal actor a response control", () => {
    renderWorkspace(request(), "reviewer-1");

    expect(screen.queryByRole("button", { name: "Open request" })).toBeNull();
    expect(screen.getByText("Only the assigned person can respond to this request.")).toBeTruthy();
    expect(screen.getByText("Ask the request creator to change the recipient if the assignment is wrong.")).toBeTruthy();
  });

  it("does not offer an internal actor a response control for an external request", () => {
    renderWorkspace(request({
      audience_type: "EXTERNAL",
      recipient: { type: "EXTERNAL_AUDIENCE", audience_hint: "c***@supplier.example", state: "ASSIGNED" },
    }), "requester-1");

    expect(screen.queryByRole("button", { name: "Open request" })).toBeNull();
    expect(screen.getByText("The external recipient responds using an invitation link; check its current status below.")).toBeTruthy();
  });

  it("uses the invitation workflow for a vendor audience", () => {
    renderWorkspace(request({
      audience_type: "VENDOR",
      recipient: { type: "EXTERNAL_AUDIENCE", audience_hint: "c***@supplier.example", state: "ASSIGNED" },
    }), "requester-1");

    expect(screen.queryByRole("button", { name: "Open request" })).toBeNull();
    expect(screen.getByText("The external recipient responds using an invitation link; check its current status below.")).toBeTruthy();
    expect(screen.getByText("c***@supplier.example")).toBeTruthy();
  });

  it("offers the response control only to the exact assigned internal recipient", () => {
    const { onOpenRequest } = renderWorkspace(request(), "recipient-1");

    fireEvent.click(screen.getByRole("button", { name: "Open request" }));
    expect(onOpenRequest).toHaveBeenCalledWith("request-1");
  });

  it("fails closed when verified actor identity is missing", () => {
    renderWorkspace(request({
      recipient: { type: "INTERNAL_PRINCIPAL", state: "ASSIGNED" },
    }), undefined);

    expect(screen.queryByRole("button", { name: "Open request" })).toBeNull();
    expect(screen.getByText("Only the assigned person can respond to this request.")).toBeTruthy();
  });

  it("fails closed when an external audience is paired with an internal recipient payload", () => {
    renderWorkspace(request({
      audience_type: "EXTERNAL",
      recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "recipient-1", state: "ASSIGNED" },
    }), "recipient-1");

    expect(screen.queryByRole("button", { name: "Open request" })).toBeNull();
    expect(screen.getByText("The external recipient responds using an invitation link; check its current status below.")).toBeTruthy();
  });

  it("fails closed when the recipient assignment state is missing", () => {
    renderWorkspace(request({
      recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "recipient-1" },
    }), "recipient-1");

    expect(screen.queryByRole("button", { name: "Open request" })).toBeNull();
  });

  it("does not announce a static denied notice as a status update", () => {
    renderWorkspace(request(), "reviewer-1");

    expect(screen.getByText("Only the assigned person can respond to this request.").closest('[role="status"]')).toBeNull();
  });

  it("publishes a returned request to the parent workspace", async () => {
    const updated = request({
      recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "recipient-1", state: "REASSIGNMENT_REQUIRED", revision: 2 },
      version: 2,
    });
    vi.mocked(declareWrongCaptureRecipient).mockResolvedValue(updated);
    const { onRequestUpdated } = renderWorkspace(request(), "recipient-1");

    fireEvent.change(screen.getByLabelText("Why should it be reassigned?"), { target: { value: "The account owner must respond." } });
    fireEvent.click(screen.getByRole("button", { name: "Return to requester" }));

    await waitFor(() => expect(onRequestUpdated).toHaveBeenCalledWith(updated, 1));
  });

  it("publishes a reassigned request to the parent workspace", async () => {
    const updated = request({
      recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "recipient-2", state: "ASSIGNED", revision: 2 },
      version: 2,
    });
    vi.mocked(reassignCaptureRecipient).mockResolvedValue(updated);
    listEvidenceRecipientCandidates.mockResolvedValue({ items: [{ principal_id: "recipient-2", display_name: "Ada Okafor" }], has_more: false });
    const { onRequestUpdated } = renderWorkspace(request(), "requester-1");

    await screen.findByRole("option", { name: "Ada Okafor" });
    fireEvent.change(screen.getByLabelText("New assigned person"), { target: { value: "recipient-2" } });
    fireEvent.change(screen.getByLabelText("Reason for change"), { target: { value: "The control owner now holds the evidence." } });
    fireEvent.click(screen.getByRole("button", { name: "Save recipient" }));

    await waitFor(() => expect(onRequestUpdated).toHaveBeenCalledWith(updated, 1));
  });

  it("ignores a lifecycle completion after the verified evidence scope changes", async () => {
    const completion = deferred<EvidenceRequest>();
    const onRequestUpdated = vi.fn().mockReturnValue(true);
    vi.mocked(declareWrongCaptureRecipient).mockReturnValue(completion.promise);
    const evidenceRequest = request();
    const view = render(
      <EvidenceWorkspace sources={[]} requests={[evidenceRequest]} sourceState="live" requestState="live" targetID={evidenceRequest.id} actorPrincipalID="recipient-1" evidenceScopeToken={1} onOpenRequest={vi.fn()} onRequestUpdated={onRequestUpdated}/>,
    );

    fireEvent.change(screen.getByLabelText("Why should it be reassigned?"), { target: { value: "The account owner must respond." } });
    fireEvent.click(screen.getByRole("button", { name: "Return to requester" }));
    await waitFor(() => expect(declareWrongCaptureRecipient).toHaveBeenCalled());
    view.rerender(
      <EvidenceWorkspace sources={[]} requests={[evidenceRequest]} sourceState="live" requestState="live" targetID={evidenceRequest.id} actorPrincipalID="recipient-1" evidenceScopeToken={2} onOpenRequest={vi.fn()} onRequestUpdated={onRequestUpdated}/>,
    );
    await act(async () => {
      completion.resolve(request({ recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "recipient-1", state: "REASSIGNMENT_REQUIRED", revision: 2 }, version: 2 }));
    });

    expect(onRequestUpdated).not.toHaveBeenCalled();
    expect(screen.queryByText("Recipient correction required.")).toBeNull();
  });
});
