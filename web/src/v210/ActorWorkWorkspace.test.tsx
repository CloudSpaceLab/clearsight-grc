import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { ActorWorkWorkspace } from "./ActorWorkWorkspace";
const api = vi.hoisted(() => ({ loadActorWorkPage: vi.fn() }));
const approvalSources = vi.hoisted(() => ({ forms: vi.fn(), reviews: vi.fn() }));
vi.mock("../api", () => api);
vi.mock("../formsApi", () => ({ loadFormTemplatePage: approvalSources.forms }));
vi.mock("./programResponseReviewsApi", () => ({ loadProgramResponseReviews: approvalSources.reviews }));
const context = { tenant: { id: "bank", name: "Bank" }, legal_entity: { id: "entity", name: "Entity" }, actor: { id: "person", name: "Person" }, mode: "LIVE" };
const item = (id: string) => ({ id, title: `Review ${id}`, type: "REVIEW", state: "READY", owner: "Person", due_at: "2026-12-01T00:00:00Z" });
const page = { scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "person" }, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "NOT_AVAILABLE", as_of: "2026-11-21T10:00:00Z", items: [item("one")], has_more: true, next_cursor: "next" };
beforeEach(() => { vi.resetAllMocks(); });
it("brings form approvals and assigned Program reviews together", async () => {
  api.loadActorWorkPage.mockResolvedValue(page);
  approvalSources.forms.mockResolvedValue({ items: [] });
  approvalSources.reviews.mockResolvedValue({ scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "person" }, coverage: "PROGRAM_RESPONSE_REVIEWS", items: [{ id: "revision-1", title: "Access response", revision: 1, current: true, state: "FINAL", subject_type: "PROGRAM", submitted_by_display_name: "Ada Okafor" }] });
  render(<ActorWorkWorkspace context={context}/>);
  fireEvent.click(screen.getByRole("button", { name: "Approvals & reviews" }));
  expect(await screen.findByRole("link", { name: "Review Access response" })).toBeTruthy();
  expect(screen.getByRole("heading", { name: "Form approvals" })).toBeTruthy();
});
it("loads and continues the actor-scoped work list with a stable receipt", async () => {
  api.loadActorWorkPage.mockResolvedValueOnce(page).mockResolvedValueOnce({ ...page, items: [item("two")], has_more: false, next_cursor: undefined });
  render(<ActorWorkWorkspace context={context}/>);
  expect(await screen.findByText("Review one")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more work" }));
  expect(await screen.findByText("Review two")).toBeTruthy();
  expect(api.loadActorWorkPage).toHaveBeenLastCalledWith("", "next");
  expect(screen.queryByRole("button", { name: /Open form|Approve|Complete/i })).toBeNull();
});
it.each(["tenant_id", "legal_entity_id", "principal_id"])("rejects a mismatched %s receipt", async (key) => {
  api.loadActorWorkPage.mockResolvedValue({ ...page, scope: { ...page.scope, [key]: "other" } });
  render(<ActorWorkWorkspace context={context}/>);
  expect(await screen.findByRole("alert")).toBeTruthy();
  expect(screen.queryByText("Review one")).toBeNull();
});
it("clears loaded work when a later receipt changes snapshot or scope", async () => {
  api.loadActorWorkPage.mockResolvedValueOnce(page).mockResolvedValueOnce({ ...page, as_of: "2026-11-21T10:01:00Z", items: [item("two")], has_more: false, next_cursor: undefined });
  render(<ActorWorkWorkspace context={context}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Load more work" }));
  expect(await screen.findByRole("alert")).toBeTruthy();
  expect(screen.queryByText("Review one")).toBeNull();
  expect(screen.queryByText("Review two")).toBeNull();
});
it("clears work immediately when the actor changes", async () => {
  api.loadActorWorkPage.mockResolvedValueOnce(page).mockReturnValueOnce(new Promise(() => {}));
  const view = render(<ActorWorkWorkspace context={context}/>);
  expect(await screen.findByText("Review one")).toBeTruthy();
  view.rerender(<ActorWorkWorkspace context={{ ...context, actor: { ...context.actor, id: "other" } }}/>);
  expect(screen.queryByText("Review one")).toBeNull();
});
it("ignores a late response from the previous actor", async () => {
  let resolvePrevious!: (value: typeof page) => void;
  api.loadActorWorkPage.mockReturnValueOnce(new Promise((resolve) => { resolvePrevious = resolve; })).mockReturnValueOnce(new Promise(() => {}));
  const view = render(<ActorWorkWorkspace context={context}/>);
  view.rerender(<ActorWorkWorkspace context={{ ...context, actor: { ...context.actor, id: "other" } }}/>);
  resolvePrevious(page);
  await waitFor(() => expect(api.loadActorWorkPage).toHaveBeenCalledTimes(2));
  expect(screen.queryByText("Review one")).toBeNull();
});
it("does not query without an exact legal entity", async () => {
  render(<ActorWorkWorkspace context={{ ...context, legal_entity: { ...context.legal_entity, id: "*" } }}/>);
  expect(await screen.findByRole("alert")).toBeTruthy();
  expect(api.loadActorWorkPage).not.toHaveBeenCalled();
});
it("searches on the server and describes empty search results", async () => {
  api.loadActorWorkPage.mockResolvedValueOnce(page).mockResolvedValueOnce({ ...page, items: [], has_more: false, next_cursor: undefined });
  render(<ActorWorkWorkspace context={context}/>);
  await screen.findByText("Review one");
  fireEvent.change(screen.getByLabelText("Search work"), { target: { value: "supplier" } });
  fireEvent.submit(screen.getByRole("search"));
  await waitFor(() => expect(api.loadActorWorkPage).toHaveBeenLastCalledWith("supplier"));
  expect(await screen.findByText("No work matches the search.")).toBeTruthy();
});

it("links a Matter assignment to its scoped issue without creating links for other work", async () => {
  api.loadActorWorkPage.mockResolvedValue({ ...page, items: [
    { ...item("one"), action_target_type: "MATTER", action_target_id: "matter/one" },
    { ...item("two"), action_target_type: "PROGRAM", action_target_id: "program-1" },
  ], has_more: false, next_cursor: undefined });
  render(<ActorWorkWorkspace context={context}/>);
  expect((await screen.findByRole("link", { name: "Review one" })).getAttribute("href")).toBe("#/work/issues/matter%2Fone");
  expect(screen.queryByRole("link", { name: "Review two" })).toBeNull();
});
