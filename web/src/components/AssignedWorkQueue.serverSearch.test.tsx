import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ActorWorkWorkspace } from "../v210/ActorWorkWorkspace";

const api = vi.hoisted(() => ({ loadActorWorkPage: vi.fn() }));
vi.mock("../api", () => api);
vi.mock("../formsApi", () => ({ loadFormTemplatePage: vi.fn() }));
vi.mock("../v210/programResponseReviewsApi", () => ({ loadProgramResponseReviews: vi.fn() }));

const context = { tenant: { id: "bank", name: "Bank" }, legal_entity: { id: "entity", name: "Entity" }, actor: { id: "person", name: "Person" }, mode: "LIVE" };
const item = (id: string, title: string) => ({ id, title, type: "REVIEW", state: "READY", owner: "Person", due_at: "" });
const page = { scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "person" }, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "NOT_AVAILABLE", as_of: "2026-11-21T10:00:00Z", items: [item("initial", "Original work")], has_more: false };

beforeEach(() => vi.resetAllMocks());

describe("actor-scoped work search", () => {
  it("searches the server and continues the same search with its cursor", async () => {
    api.loadActorWorkPage.mockResolvedValueOnce(page)
      .mockResolvedValueOnce({ ...page, items: [item("one", "Review retained document")], has_more: true, next_cursor: "next" })
      .mockResolvedValueOnce({ ...page, items: [item("two", "Review another document")] });
    render(<ActorWorkWorkspace context={context}/>);
    await screen.findByText("Original work");
    const field = screen.getByLabelText("Search work");
    fireEvent.change(field, { target: { value: "retained" } });
    fireEvent.submit(screen.getByRole("search"));
    expect(await screen.findByText("Review retained document")).toBeTruthy();
    expect(screen.queryByText("Original work")).toBeNull();
    expect(api.loadActorWorkPage).toHaveBeenLastCalledWith("retained");
    fireEvent.click(screen.getByRole("button", { name: "Load more work" }));
    expect(await screen.findByText("Review another document")).toBeTruthy();
    expect(api.loadActorWorkPage).toHaveBeenLastCalledWith("retained", "next");
    expect(screen.queryByRole("button", { name: "Load more work" })).toBeNull();
  });

  it("clears old search results when the entity changes and ignores a late page", async () => {
    let resolveOld!: (value: typeof page) => void;
    api.loadActorWorkPage.mockResolvedValueOnce(page).mockReturnValueOnce(new Promise((resolve) => { resolveOld = resolve; })).mockReturnValueOnce(new Promise(() => {}));
    const view = render(<ActorWorkWorkspace context={context}/>);
    await screen.findByText("Original work");
    fireEvent.change(screen.getByLabelText("Search work"), { target: { value: "retained" } });
    fireEvent.submit(screen.getByRole("search"));
    await waitFor(() => expect(api.loadActorWorkPage).toHaveBeenCalledWith("retained"));
    view.rerender(<ActorWorkWorkspace context={{ ...context, legal_entity: { ...context.legal_entity, id: "other" } }}/>);
    await act(async () => resolveOld(page));
    expect(screen.queryByText("Original work")).toBeNull();
    expect(screen.getByRole("status").textContent).toContain("Loading work");
  });

  it("rejects an empty page that claims more results", async () => {
    api.loadActorWorkPage.mockResolvedValueOnce(page)
      .mockResolvedValueOnce({ ...page, items: [], has_more: true, next_cursor: "next" });
    render(<ActorWorkWorkspace context={context}/>);
    await screen.findByText("Original work");
    fireEvent.change(screen.getByLabelText("Search work"), { target: { value: "retained" } });
    fireEvent.submit(screen.getByRole("search"));
    expect((await screen.findByRole("alert")).textContent).toContain("Work unavailable.");
    expect(screen.queryByText("No work matches the search.")).toBeNull();
    expect(screen.queryByRole("button", { name: "Load more work" })).toBeNull();
  });

  it("does not query or expose records for an unscoped entity", async () => {
    render(<ActorWorkWorkspace context={{ ...context, legal_entity: { ...context.legal_entity, id: "*" } }}/>);
    expect(await screen.findByRole("alert")).toBeTruthy();
    fireEvent.change(screen.getByLabelText("Search work"), { target: { value: "retained" } });
    fireEvent.submit(screen.getByRole("search"));
    expect(api.loadActorWorkPage).not.toHaveBeenCalled();
  });
});
