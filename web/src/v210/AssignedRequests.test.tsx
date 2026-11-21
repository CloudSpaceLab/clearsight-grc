import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { AssignedRequests } from "./AssignedRequests";
const api = vi.hoisted(() => ({ loadActorWorkPage: vi.fn() }));
vi.mock("../api", () => api);
vi.mock("./InternalResponse", () => ({ InternalResponse: ({ requestId }: { requestId: string }) => <p>Response: {requestId}</p> }));
const context = { tenant: { id: "bank" }, legal_entity: { id: "entity" }, actor: { id: "person" } } as never;
beforeEach(() => { vi.clearAllMocks(); api.loadActorWorkPage.mockResolvedValue({ scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "person" }, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", items: [{ id: "task", title: "Review access", action_target_type: "EVIDENCE_REQUEST", action_target_id: "request-1", due_at: "2026-12-01T12:00:00Z", state: "READY" }] }); });
it("opens the exact assigned request", async () => {
  render(<AssignedRequests context={context} allowOpening/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open Review access" }));
  expect(screen.getByText("Response: request-1")).toBeTruthy();
});
it("does not expose form opening until capture is available", async () => {
  render(<AssignedRequests context={context}/>);
  expect(await screen.findByText("Review access")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Open Review access" })).toBeNull();
  expect(screen.getByText("Form response unavailable.")).toBeTruthy();
});
it("does not display a different person's assignment receipt", async () => {
  api.loadActorWorkPage.mockResolvedValue({ scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "other" }, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", items: [{ title: "Restricted assignment", action_target_type: "EVIDENCE_REQUEST", action_target_id: "request-1" }] });
  render(<AssignedRequests context={context}/>);
  expect(await screen.findByText("Assignments unavailable.")).toBeTruthy();
  expect(screen.queryByText("Restricted assignment")).toBeNull();
});
