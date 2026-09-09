import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { FormLibraryItem, SavedFormView } from "../formsTypes";
import { FormsWorkspace } from "./FormsWorkspace";

const api = vi.hoisted(() => ({
  loadFormTemplatePage: vi.fn(),
  loadFormTemplateRevision: vi.fn(),
  loadReusableFormTemplateRefs: vi.fn(),
  loadStarterTemplates: vi.fn(),
  loadSavedFormViews: vi.fn(),
  createLibraryFormDraft: vi.fn(),
  createLibraryFormRevision: vi.fn(),
  instantiateStarterTemplate: vi.fn(),
  saveFormView: vi.fn(),
  deleteSavedFormView: vi.fn(),
  transitionFormTemplateRevision: vi.fn(),
}));

vi.mock("../formsApi", () => api);

const documents = vi.hoisted(() => ({ loadDocuments: vi.fn() }));
vi.mock("../submittedDocumentApi", () => documents);

const draftItem: FormLibraryItem = {
  template: {
    id: "template-a",
    tenant_id: "bank-a",
    legal_entity_id: "entity-a",
    code: "VENDOR",
    name: "Vendor due diligence",
    purpose: "Collect current vendor evidence.",
    status: "DRAFT",
    is_current: false,
    version: 2,
    created_at: "2026-08-27T09:00:00Z",
    updated_at: "2026-08-27T10:00:00Z",
    sensitivity: "INTERNAL",
    scoring_mode: "NONE",
    presentation: { default_mode: "AUTOMATIC", allow_mode_switch: true },
    sections: [{ id: "general", title: "General" }],
    fields: [{ id: "question_1", section_id: "general", label: "Registered name", type: "short_text", required: true }],
  },
  active_version: 1,
  active_status: "ACTIVE",
};

const savedView: SavedFormView = {
  id: "view-active-outsourcing",
  name: "Active outsourcing",
  filter: { search: "outsourcing", status: "ACTIVE", tag: "third-party" },
  created_at: "2026-08-27T09:00:00Z",
  updated_at: "2026-08-27T09:00:00Z",
};

beforeEach(() => {
  window.history.replaceState(null, "", "#forms");
  Object.values(api).forEach((mock) => mock.mockReset());
  api.loadFormTemplatePage.mockResolvedValue({ items: [draftItem] });
  api.loadFormTemplateRevision.mockResolvedValue(draftItem.template);
  api.loadReusableFormTemplateRefs.mockResolvedValue([]);
  api.loadStarterTemplates.mockResolvedValue([]);
  api.loadSavedFormViews.mockResolvedValue([]);
  documents.loadDocuments.mockReset().mockResolvedValue({ items: [] });
});

afterEach(() => vi.restoreAllMocks());

describe("Forms workspace location state", () => {
  it("opens Imports directly without a second launcher and preserves the prior Forms URL", async () => {
    window.history.replaceState(null, "", "#forms?search=vendor");
    render(<FormsWorkspace/>);
    await screen.findAllByText("Vendor due diligence");
    fireEvent.click(screen.getByRole("tab", { name: "Imports" }));
    expect(window.location.hash).toBe("#imports");
    expect(screen.queryByRole("button", { name: "Open Imports" })).toBeNull();
  });

  it.each(["hashchange", "popstate"])("lets a pending library read settle after section-only %s navigation", async (event) => {
    let finish!: (page: { items: FormLibraryItem[] }) => void;
    api.loadFormTemplatePage.mockImplementationOnce(() => new Promise((resolve) => { finish = resolve; }));
    render(<FormsWorkspace/>);
    await waitFor(() => expect(api.loadFormTemplatePage).toHaveBeenCalledTimes(1));

    fireEvent.click(screen.getByRole("tab", { name: "Imports" }));
    window.history.replaceState(null, "", "#forms");
    fireEvent(window, new Event(event));
    await act(async () => finish({ items: [draftItem] }));

    expect(await screen.findByRole("button", { name: "Details for Vendor due diligence" })).toBeTruthy();
    expect(screen.queryByText("Loading form templates…")).toBeNull();
    expect(api.loadFormTemplatePage).toHaveBeenCalledTimes(1);
  });

  it("keeps a pending advanced-filter read when section history repeats the same expression", async () => {
    const expression = encodeURIComponent(JSON.stringify({ kind: "condition", field: "status", operator: "is", value: "ACTIVE" }));
    const hash = `#forms?filter=${expression}`;
    window.history.replaceState(null, "", hash);
    let finish!: (page: { items: FormLibraryItem[] }) => void;
    api.loadFormTemplatePage.mockImplementationOnce(() => new Promise((resolve) => { finish = resolve; }));
    render(<FormsWorkspace/>);
    await waitFor(() => expect(api.loadFormTemplatePage).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByRole("tab", { name: "Imports" }));
    window.history.replaceState(null, "", hash);
    fireEvent(window, new Event("popstate"));
    fireEvent(window, new Event("hashchange"));
    await act(async () => finish({ items: [draftItem] }));

    expect(await screen.findByRole("button", { name: "Details for Vendor due diligence" })).toBeTruthy();
    expect(api.loadFormTemplatePage.mock.calls[0]?.[1].aborted).toBe(false);
    expect(api.loadFormTemplatePage).toHaveBeenCalledTimes(1);
  });

  it("rejects an obsolete library read when history changes the actual template query", async () => {
    let finishOld!: (page: { items: FormLibraryItem[] }) => void;
    const current = { ...draftItem, template: { ...draftItem.template, id: "current", name: "Current template" } };
    api.loadFormTemplatePage.mockImplementationOnce(() => new Promise((resolve) => { finishOld = resolve; }))
      .mockResolvedValue({ items: [current] });
    render(<FormsWorkspace/>);
    await waitFor(() => expect(api.loadFormTemplatePage).toHaveBeenCalledTimes(1));
    window.history.replaceState(null, "", "#forms?search=current");
    fireEvent(window, new Event("popstate"));
    fireEvent(window, new Event("hashchange"));
    expect(api.loadFormTemplatePage.mock.calls[0]?.[1].aborted).toBe(true);
    await screen.findByRole("button", { name: "Details for Current template" });
    await act(async () => finishOld({ items: [draftItem] }));
    expect(screen.queryByRole("button", { name: "Details for Vendor due diligence" })).toBeNull();
    expect(screen.getByRole("button", { name: "Details for Current template" })).toBeTruthy();
  });

  it("rejects an obsolete additional page when history changes the actual template query", async () => {
    let finishPage!: (page: { items: FormLibraryItem[] }) => void;
    const old = { ...draftItem, template: { ...draftItem.template, id: "old-page", name: "Old additional template" } };
    const current = { ...draftItem, template: { ...draftItem.template, id: "current", name: "Current template" } };
    api.loadFormTemplatePage.mockResolvedValueOnce({ items: [draftItem], next_cursor: "next" })
      .mockImplementationOnce(() => new Promise((resolve) => { finishPage = resolve; }))
      .mockResolvedValue({ items: [current] });
    render(<FormsWorkspace/>);
    fireEvent.click(await screen.findByRole("button", { name: "Load more" }));
    await waitFor(() => expect(api.loadFormTemplatePage).toHaveBeenCalledTimes(2));
    window.history.replaceState(null, "", "#forms?search=current");
    fireEvent(window, new Event("popstate"));
    await screen.findByRole("button", { name: "Details for Current template" });
    await act(async () => finishPage({ items: [old] }));
    expect(screen.queryByRole("button", { name: "Details for Old additional template" })).toBeNull();
    expect(screen.getByRole("button", { name: "Details for Current template" })).toBeTruthy();
  });

  it("opens Documents from a direct section URL and re-fetches after remount", async () => {
    window.history.replaceState(null, "", "#forms?section=documents");
    const first = render(<FormsWorkspace/>);
    expect(screen.getByRole("tab", { name: "Documents" }).getAttribute("aria-selected")).toBe("true");
    const panel = screen.getByRole("tabpanel", { name: "Documents" });
    expect(within(panel).getByRole("heading", { name: "Documents" })).toBeTruthy();
    await screen.findByText("No matching documents");
    expect(documents.loadDocuments).toHaveBeenCalledTimes(1);

    first.unmount();
    render(<FormsWorkspace/>);
    expect(screen.getByRole("tab", { name: "Documents" }).getAttribute("aria-selected")).toBe("true");
    await screen.findByText("No matching documents");
    expect(documents.loadDocuments).toHaveBeenCalledTimes(2);
  });

  it.each(["#forms", "#forms?section=unknown", "#forms?section=", "#forms/documents"])(
    "defaults to Templates without reinterpreting the path in %s", async (hash) => {
      window.history.replaceState(null, "", hash);
      render(<FormsWorkspace targetID={hash === "#forms/documents" ? "documents" : undefined}/>);
      expect(screen.getByRole("tab", { name: "Templates", hidden: true }).getAttribute("aria-selected")).toBe("true");
      await screen.findAllByText("Vendor due diligence");
      expect(window.location.hash).toBe(hash);
    },
  );

  it("pushes one entry per different section while retaining the template target and filters", async () => {
    const query = "search=vendor&status=ACTIVE&owner=owner-a&program=program-a&use=REVIEW&tag=third-party&sort=UPDATED_ASC&limit=50";
    window.history.replaceState(null, "", `#forms/template-a?${query}`);
    render(<FormsWorkspace targetID="template-a"/>);
    await screen.findAllByText("Vendor due diligence");
    const push = vi.spyOn(window.history, "pushState");
    const replace = vi.spyOn(window.history, "replaceState");

    fireEvent.click(screen.getByRole("tab", { name: "Documents", hidden: true }));
    expect(window.location.hash).toBe(`#forms/template-a?${query}&section=documents`);
    expect(push).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole("tab", { name: "Documents" }));
    expect(push).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole("tab", { name: "Templates" }));
    expect(window.location.hash).toBe(`#forms/template-a?${query}`);
    expect(push).toHaveBeenCalledTimes(2);
    expect(replace).not.toHaveBeenCalled();
  });

  it.each(["hashchange", "popstate"])("restores sections on %s without adding history", async (event) => {
    render(<FormsWorkspace/>);
    await screen.findAllByText("Vendor due diligence");
    const push = vi.spyOn(window.history, "pushState");
    window.history.replaceState(null, "", "#forms?section=documents");
    fireEvent(window, new Event(event));
    expect(screen.getByRole("tabpanel", { name: "Documents" })).toBeTruthy();
    await screen.findByText("No matching documents");

    window.history.replaceState(null, "", "#forms?section=imports");
    fireEvent(window, new Event(event));
    expect(screen.getByRole("tabpanel", { name: "Imports" })).toBeTruthy();
    window.history.replaceState(null, "", "#forms?section=documents");
    fireEvent(window, new Event(event));
    await screen.findByText("No matching documents");
    expect(documents.loadDocuments).toHaveBeenCalledTimes(2);
    expect(push).not.toHaveBeenCalled();

    window.history.replaceState(null, "", "#forms?section=unrecognized");
    fireEvent(window, new Event(event));
    expect(screen.getByRole("tabpanel", { name: "Templates" })).toBeTruthy();
  });

  it.each(["click", "history"])("keeps an editor on repeated same-section events and clears it after %s section changes", async (navigation) => {
    render(<FormsWorkspace/>);
    await screen.findAllByText("Vendor due diligence");
    fireEvent.click(screen.getByRole("button", { name: "Create form" }));
    fireEvent.click(within(await screen.findByRole("dialog", { name: "New form" })).getByRole("button", { name: /^Blank form\b/ }));
    fireEvent.change(screen.getByLabelText("Question"), { target: { value: "Unsaved control question" } });
    fireEvent(window, new Event("popstate"));
    fireEvent(window, new Event("hashchange"));
    fireEvent.click(screen.getByRole("tab", { name: "Templates" }));
    expect((screen.getByLabelText("Question") as HTMLInputElement).value).toBe("Unsaved control question");

    if (navigation === "click") fireEvent.click(screen.getByRole("tab", { name: "Imports" }));
    else {
      window.history.replaceState(null, "", "#forms?section=imports");
      fireEvent(window, new Event("popstate"));
    }
    expect(screen.getByRole("tabpanel", { name: "Imports" })).toBeTruthy();
    if (navigation === "click") fireEvent.click(screen.getByRole("tab", { name: "Templates" }));
    else {
      window.history.replaceState(null, "", "#forms");
      fireEvent(window, new Event("popstate"));
    }
    expect(screen.queryByLabelText("Question")).toBeNull();
    expect(screen.getByRole("button", { name: "Create form" })).toBeTruthy();
  });

  it.each(["launcher", "AI"])("clears transient %s state when history changes section", async (surface) => {
    render(<FormsWorkspace/>);
    await screen.findAllByText("Vendor due diligence");
    fireEvent.click(screen.getByRole("button", { name: "Create form" }));
    const launcher = await screen.findByRole("dialog", { name: "New form" });
    if (surface === "AI") fireEvent.click(within(launcher).getByRole("button", { name: /^Draft with AI\b/ }));
    window.history.replaceState(null, "", "#forms?section=imports");
    fireEvent(window, new Event("popstate"));
    expect(screen.queryByRole("dialog", { name: "New form" })).toBeNull();
    window.history.replaceState(null, "", "#forms");
    fireEvent(window, new Event("hashchange"));
    expect(screen.queryByRole("button", { name: "Open manual builder" })).toBeNull();
    expect(screen.queryByRole("dialog", { name: "New form" })).toBeNull();
    expect(screen.getByRole("button", { name: "Create form" })).toBeTruthy();
  });

  it("clears a template load error after a history section change", async () => {
    api.loadFormTemplatePage.mockRejectedValueOnce(new Error("Template library read failed."));
    render(<FormsWorkspace/>);
    await screen.findByText("Template library read failed.");
    window.history.replaceState(null, "", "#forms?section=imports");
    fireEvent(window, new Event("popstate"));
    expect(screen.queryByText("Template library read failed.")).toBeNull();
    expect(screen.getByRole("tabpanel", { name: "Imports" })).toBeTruthy();
  });

  it("applies a saved view without discarding the selected target", async () => {
    api.loadSavedFormViews.mockResolvedValueOnce([savedView]);
    const onTarget = vi.fn(() => window.history.replaceState(null, "", "#forms"));

    render(<FormsWorkspace targetID="template-a" onTarget={onTarget}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Active outsourcing", hidden: true }));

    expect(onTarget).not.toHaveBeenCalled();
    expect(window.location.hash).toBe("#forms/template-a?search=outsourcing&status=ACTIVE&tag=third-party");
    await waitFor(() => expect(api.loadFormTemplatePage).toHaveBeenCalledWith(
      expect.objectContaining({ search: "outsourcing", status: "ACTIVE", tag: "third-party" }),
      expect.anything(),
      { statusFacets: true },
    ));
  });

  it("clears filters while preserving the selected target and removing stale query state", async () => {
    window.history.replaceState(null, "", "#forms/template-missing?search=outsourcing&status=ACTIVE");
    api.loadFormTemplatePage
      .mockResolvedValueOnce({ items: [] })
      .mockResolvedValue({ items: [{ ...draftItem, template: { ...draftItem.template, id: "template-missing" } }] });
    const onTarget = vi.fn(() => window.history.replaceState(null, "", "#forms"));

    render(<FormsWorkspace targetID="template-missing" onTarget={onTarget}/>);
    const clearFilters = await screen.findByRole("button", { name: "Clear filters" });
    await waitFor(() => expect(screen.queryByText("Loading form templates…")).toBeNull());
    fireEvent.click(clearFilters);

    expect(onTarget).not.toHaveBeenCalled();
    expect(window.location.hash).toBe("#forms/template-missing");
    await waitFor(() => {
      const lastCall = api.loadFormTemplatePage.mock.calls.at(-1)?.[0];
      expect(lastCall).toMatchObject({ limit: 25 });
      expect(lastCall).not.toHaveProperty("search");
      expect(lastCall).not.toHaveProperty("status");
    });
    expect(await screen.findByRole("heading", { name: "Vendor due diligence" })).toBeTruthy();
  });
});
