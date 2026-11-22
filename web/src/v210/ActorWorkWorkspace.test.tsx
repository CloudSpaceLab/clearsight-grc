import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { ActorWorkWorkspace } from "./ActorWorkWorkspace";
import { ApiError } from "../http";
const api = vi.hoisted(() => ({ loadActorWorkPage: vi.fn(), loadActorWorkDueCounts: vi.fn() }));
const approvalSources = vi.hoisted(() => ({ forms: vi.fn(), reviews: vi.fn() }));
vi.mock("../api", () => api);
vi.mock("../formsApi", () => ({ loadFormTemplatePage: approvalSources.forms }));
vi.mock("./programResponseReviewsApi", () => ({ loadProgramResponseReviews: approvalSources.reviews }));
const context = { tenant: { id: "bank", name: "Bank" }, legal_entity: { id: "entity", name: "Entity" }, actor: { id: "person", name: "Person" }, mode: "LIVE" };
const item = (id: string) => ({ id, title: `Review ${id}`, type: "REVIEW", state: "READY", owner: "Person", due_at: "2026-12-01T00:00:00Z" });
const page = { scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "person" }, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "NOT_AVAILABLE", as_of: "2026-11-21T10:00:00Z", items: [item("one")], has_more: true, next_cursor: "next" };
beforeEach(() => { vi.resetAllMocks(); api.loadActorWorkDueCounts.mockRejectedValue(new Error("unavailable")); });
it("shows assignment counts only from a valid actor-scoped snapshot", async () => {
  api.loadActorWorkPage.mockResolvedValue(page);
  api.loadActorWorkDueCounts.mockResolvedValue({ ...page, count_basis: "SOURCE_SNAPSHOT", total: 7, overdue: 2, due_soon: 3, undated: 1 });
  render(<ActorWorkWorkspace context={context}/>);
  const counts = await screen.findByRole("region", { name: "Workflow assignment counts" });
  expect(counts.textContent).toContain("7"); expect(counts.textContent).toContain("2"); expect(counts.textContent).toContain("3");
  expect(counts.textContent).toContain("As of");
});
it("withholds assignment counts from another actor", async () => {
  api.loadActorWorkPage.mockResolvedValue(page);
  api.loadActorWorkDueCounts.mockResolvedValue({ ...page, scope: { ...page.scope, principal_id: "other" }, count_basis: "SOURCE_SNAPSHOT", total: 89, overdue: 0, due_soon: 0, undated: 0 });
  render(<ActorWorkWorkspace context={context}/>);
  expect(await screen.findByText("Assignment counts unavailable.")).toBeTruthy();
  expect(screen.queryByText("89")).toBeNull();
});
it("clears the prior actor's assignment counts immediately when scope changes", async () => {
  api.loadActorWorkPage.mockResolvedValue(page);
  api.loadActorWorkDueCounts.mockResolvedValueOnce({ ...page, count_basis: "SOURCE_SNAPSHOT", total: 7, overdue: 2, due_soon: 3, undated: 1 }).mockReturnValueOnce(new Promise(() => {}));
  const view = render(<ActorWorkWorkspace context={context}/>);
  expect(await screen.findByText("7")).toBeTruthy();
  view.rerender(<ActorWorkWorkspace context={{ ...context, actor: { ...context.actor, id: "other" } }}/>);
  expect(screen.queryByText("7")).toBeNull();
});
it("labels due urgency against the source timestamp and leaves invalid deadlines unclassified", async () => {
  api.loadActorWorkPage.mockResolvedValue({ ...page, items: [{ ...item("overdue"), due_at: "2026-11-20T10:00:00Z" }, { ...item("soon"), due_at: "2026-11-25T10:00:00Z" }, { ...item("unknown"), due_at: "invalid" }], has_more: false, next_cursor: undefined });
  render(<ActorWorkWorkspace context={context}/>);
  const overdue = (await screen.findByText("Review overdue")).closest("tr")!;
  const soon = screen.getByText("Review soon").closest("tr")!;
  const unknown = screen.getByText("Review unknown").closest("tr")!;
  expect(overdue.textContent).toContain("Overdue"); expect(soon.textContent).toContain("Due soon");
  expect(unknown.textContent).not.toContain("Overdue"); expect(unknown.textContent).not.toContain("Due soon");
});
it.each([
  ["missing scope", { ...page, scope: undefined }],
  ["coverage", { ...page, coverage: "ALL_WORK" }],
  ["count basis", { ...page, count_basis: "LOADED" }],
])("rejects an initial work page with invalid %s", async (_label, invalid) => {
  api.loadActorWorkPage.mockResolvedValue(invalid);
  render(<ActorWorkWorkspace context={context}/>);
  expect((await screen.findByRole("alert")).textContent).toContain("Work unavailable");
  expect(screen.queryByText("Review one")).toBeNull();
  expect(screen.queryByRole("button", { name: "Load more work" })).toBeNull();
});

it("discards work after a later page belongs to another actor", async () => {
  api.loadActorWorkPage.mockResolvedValueOnce(page).mockResolvedValueOnce({ ...page, scope: { ...page.scope, principal_id: "other" }, items: [item("other")], has_more: false, next_cursor: undefined });
  render(<ActorWorkWorkspace context={context}/>);
  expect(await screen.findByText("Review one")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more work" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Work unavailable");
  expect(screen.queryByText("Review one")).toBeNull();
  expect(screen.queryByText("Review other")).toBeNull();
});

it.each([401, 403])("discards work when pagination returns HTTP %s", async (status) => {
  api.loadActorWorkPage.mockResolvedValueOnce(page).mockRejectedValueOnce(new ApiError(status, "Access changed", "work_scope_denied"));
  render(<ActorWorkWorkspace context={context}/>);
  expect(await screen.findByText("Review one")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more work" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Work unavailable");
  expect(screen.queryByText("Review one")).toBeNull();
  expect(screen.queryByRole("button", { name: "Load more work" })).toBeNull();
});

it("retains loaded work and retries the same cursor after a connection failure", async () => {
  api.loadActorWorkPage.mockResolvedValueOnce(page).mockRejectedValueOnce(new ApiError(0, "Connection lost", "connection_lost"))
    .mockResolvedValueOnce({ ...page, items: [item("two")], has_more: false, next_cursor: undefined });
  render(<ActorWorkWorkspace context={context}/>);
  expect(await screen.findByText("Review one")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more work" }));
  expect((await screen.findByRole("alert")).textContent).toContain("More work unavailable");
  expect(screen.getByText("Review one")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more work" }));
  expect(await screen.findByText("Review two")).toBeTruthy();
  expect(screen.getByText("Review one")).toBeTruthy();
  expect(api.loadActorWorkPage.mock.calls[2]).toEqual(api.loadActorWorkPage.mock.calls[1]);
});
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
