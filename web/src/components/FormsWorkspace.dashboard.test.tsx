import { useState } from "react";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { FormLibraryItem } from "../formsTypes";
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
  createAIFormProposal: vi.fn(),
  createAIFormRevisionProposal: vi.fn(),
  loadFormProposal: vi.fn(),
  acceptFormProposal: vi.fn(),
  rejectFormProposal: vi.fn(),
}));

vi.mock("../formsApi", () => api);

const item: FormLibraryItem = {
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
    updated_at: "2026-08-30T10:00:00Z",
    sensitivity: "INTERNAL",
    scoring_mode: "NONE",
    presentation: { default_mode: "AUTOMATIC", allow_mode_switch: true },
    sections: [{ id: "general", title: "General" }],
    fields: [{ id: "question_1", section_id: "general", label: "Registered name", type: "short_text", required: true }],
    tags: ["third-party"],
  },
  active_version: 1,
  active_status: "ACTIVE",
  authority_available: true,
  operations: [
    { command: "forms.template.revise", label: "Edit draft", responsibility: "ACCOUNTABLE_OWNER", can_act: true, reason: "You can edit this draft." },
    { command: "forms.template.transition", label: "Change form status", responsibility: "ACCOUNTABLE_OWNER", can_act: true, reason: "You can send this draft for approval.", allowed_targets: ["PENDING_APPROVAL"] },
  ],
};

beforeEach(() => {
  window.history.replaceState(null, "", "#forms");
  window.localStorage.clear();
  for (const mock of Object.values(api)) mock.mockReset();
  api.loadFormTemplatePage.mockResolvedValue({ items: [item] });
  api.loadFormTemplateRevision.mockResolvedValue(item.template);
  api.loadReusableFormTemplateRefs.mockResolvedValue([]);
  api.loadStarterTemplates.mockResolvedValue([]);
  api.loadSavedFormViews.mockResolvedValue([]);
});

describe("Forms template dashboard", () => {
  it("keeps the result surface full-width until a template is selected", async () => {
    const view = render(<FormsWorkspace/>);
    expect(await screen.findByRole("button", { name: "Open Vendor due diligence" })).toBeTruthy();
    expect(screen.queryByLabelText("Selected form template")).toBeNull();
    expect(screen.queryByText("Recently updated")).toBeNull();
    expect(screen.queryByRole("button", { name: "Cards" })).toBeNull();
    expect(view.container.querySelector(".cs-data-table")).toBeTruthy();
    expect(view.container.querySelector(".forms-library-table")).toBeNull();
    expect(screen.getByRole("checkbox", { name: "Select Vendor due diligence" }).closest(".cs-checkbox-field")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Open Vendor due diligence" }).className).toContain("cs-button");
  });

  it("opens template detail contextually and dismisses it without changing library data", async () => {
    function Harness() {
      const [target, setTarget] = useState<string>();
      const [, setRenderCount] = useState(0);
      return <><button type="button" onClick={() => setRenderCount((count) => count + 1)}>Rerender host</button><FormsWorkspace targetID={target} onTarget={setTarget}/></>;
    }

    render(<Harness/>);
    const openButton = await screen.findByRole("button", { name: "Open Vendor due diligence" });
    openButton.focus();
    fireEvent.click(openButton);
    const drawer = await screen.findByRole("dialog", { name: "Selected form template" });
    expect(drawer.textContent).toMatch(/Latest stored.*Draft.*v2/);
    expect(drawer.textContent).toMatch(/Reusable now.*Active.*v1/);
    const closeButton = screen.getByRole("button", { name: "Close form detail" });
    await waitFor(() => expect(document.activeElement).toBe(closeButton));
    const editButton = screen.getByRole("button", { name: "Edit draft" });
    editButton.focus();
    fireEvent.click(screen.getByText("Rerender host", { selector: "button" }));
    await waitFor(() => expect(document.activeElement).toBe(editButton));
    fireEvent.keyDown(drawer, { key: "Escape" });
    await waitFor(() => expect(screen.queryByLabelText("Selected form template")).toBeNull());
    expect(document.activeElement).toBe(openButton);
    expect(screen.getByRole("button", { name: "Open Vendor due diligence" })).toBeTruthy();
  });

  it("keeps every record field labelled when rows stack on narrow screens", async () => {
    render(<FormsWorkspace/>);
    const openButton = await screen.findByRole("button", { name: "Open Vendor due diligence" });
    const row = openButton.closest("tr");

    expect(row).not.toBeNull();
    expect(row?.querySelector('[data-label="State"]')?.textContent).toBe("Draft");
    expect(row?.querySelector('[data-label="Revision"]')?.textContent).toMatch(/v2.*Reusable v1/);
    expect(row?.querySelector('[data-label="Owner"]')?.textContent).toBe("Not assigned");
    expect(row?.querySelector('[data-label="Updated"]')?.textContent).toBe("Aug 30, 2026");
  });

  it("shows lifecycle actions only when the current authority read permits them", async () => {
    api.loadFormTemplatePage.mockResolvedValue({
      items: [{
        ...item,
        operations: item.operations?.map((operation) => ({ ...operation, can_act: false, reason: "Assigned to the current form owner." })),
      }],
    });
    function Harness() {
      const [target, setTarget] = useState<string>();
      return <FormsWorkspace targetID={target} onTarget={setTarget}/>;
    }
    render(<Harness/>);
    fireEvent.click(await screen.findByRole("button", { name: "Open Vendor due diligence" }));

    expect(screen.queryByRole("button", { name: "Edit draft" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Send for approval" })).toBeNull();
    expect(screen.getByText("Assigned to the current form owner.")).toBeTruthy();
  });

  it("opens a new draft revision from an active reusable form", async () => {
    const activeItem: FormLibraryItem = {
      ...item,
      template: { ...item.template, status: "ACTIVE", is_current: true, version: 3 },
      active_version: 3,
      operations: [
        { command: "forms.template.revise", label: "Create revision", responsibility: "ACCOUNTABLE_OWNER", can_act: true, reason: "You can create a new revision." },
        { command: "forms.template.transition", label: "Change form status", responsibility: "ACCOUNTABLE_OWNER", can_act: true, reason: "You can change this active revision.", allowed_targets: ["PAUSED", "RETIRED"] },
      ],
    };
    api.loadFormTemplatePage.mockResolvedValue({ items: [activeItem] });
    function Harness() {
      const [target, setTarget] = useState<string>();
      return <FormsWorkspace targetID={target} onTarget={setTarget}/>;
    }
    render(<Harness/>);

    fireEvent.click(await screen.findByRole("button", { name: "Open Vendor due diligence" }));
    fireEvent.click(screen.getByRole("button", { name: "Create revision" }));

    expect(await screen.findByText("Vendor due diligence", { selector: "strong" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Save draft" })).toBeTruthy();
  });

  it("fails closed when the authority read is unavailable even if stale operations were present", async () => {
    api.loadFormTemplatePage.mockResolvedValue({ items: [{ ...item, authority_available: false }] });
    function Harness() {
      const [target, setTarget] = useState<string>();
      return <FormsWorkspace targetID={target} onTarget={setTarget}/>;
    }
    render(<Harness/>);
    fireEvent.click(await screen.findByRole("button", { name: "Open Vendor due diligence" }));

    expect(screen.queryByRole("button", { name: "Edit draft" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Send for approval" })).toBeNull();
    expect(screen.getByText(/Current responsibilities could not be checked/)).toBeTruthy();
  });

  it("keeps saved-view controls quiet until the current query is customized", async () => {
    render(<FormsWorkspace/>);
    await screen.findByRole("button", { name: "Open Vendor due diligence" });
    expect(screen.queryByRole("button", { name: "Save view" })).toBeNull();
    fireEvent.change(screen.getByRole("searchbox"), { target: { value: "vendor" } });
    expect(await screen.findByRole("button", { name: "Save view" })).toBeTruthy();
  });

  it("adds and removes a typed filter while keeping the URL canonical", async () => {
    render(<FormsWorkspace/>);
    await screen.findByRole("button", { name: "Open Vendor due diligence" });

    fireEvent.click(screen.getByRole("button", { name: "+ Filter" }));
    const picker = screen.getByRole("dialog", { name: "Add filter" });
    fireEvent.click(within(picker).getByRole("button", { name: /Status/ }));
    fireEvent.click(within(picker).getByRole("button", { name: /Status value/ }));
    fireEvent.click(await screen.findByRole("option", { name: "Draft" }));
    fireEvent.click(within(picker).getByRole("button", { name: "Apply filter" }));

    expect(screen.getByRole("button", { name: "Remove Status filter" }).textContent).toContain("Draft");
    expect(window.location.hash).toBe("#forms?status=DRAFT");
    await waitFor(() => expect(api.loadFormTemplatePage.mock.calls.some(([query]) => query.status === "DRAFT")).toBe(true));

    fireEvent.click(screen.getByRole("button", { name: "Remove Status filter" }));
    expect(window.location.hash).toBe("#forms");
  });

  it("keeps the current table visible while a superseding query revalidates", async () => {
    render(<FormsWorkspace/>);
    await screen.findByRole("button", { name: "Open Vendor due diligence" });

    let resolveNext!: (value: { items: FormLibraryItem[] }) => void;
    const pending = new Promise<{ items: FormLibraryItem[] }>((resolve) => { resolveNext = resolve; });
    api.loadFormTemplatePage.mockImplementationOnce(() => pending);

    fireEvent.change(screen.getByRole("searchbox"), { target: { value: "no match" } });
    await waitFor(() => expect(api.loadFormTemplatePage.mock.calls.length).toBeGreaterThan(1));
    expect(screen.getByRole("button", { name: "Open Vendor due diligence" })).toBeTruthy();
    expect(screen.getByText("Updating…")).toBeTruthy();

    resolveNext({ items: [] });
    expect(await screen.findByRole("heading", { name: "No templates match “no match”" })).toBeTruthy();
  });

  it("revalidates on focus only after the visible library becomes stale", async () => {
    let now = 1_000;
    const nowSpy = vi.spyOn(Date, "now").mockImplementation(() => now);
    try {
      render(<FormsWorkspace/>);
      await screen.findByRole("button", { name: "Open Vendor due diligence" });
      const initialCalls = api.loadFormTemplatePage.mock.calls.length;

      window.dispatchEvent(new Event("focus"));
      await Promise.resolve();
      expect(api.loadFormTemplatePage.mock.calls.length).toBe(initialCalls);

      now += 31_000;
      await waitFor(() => {
        window.dispatchEvent(new Event("focus"));
        expect(api.loadFormTemplatePage.mock.calls.length).toBe(initialCalls + 1);
      });
      expect(api.loadFormTemplatePage.mock.calls.at(-1)?.[2]).toEqual({ statusFacets: true });
    } finally {
      nowSpy.mockRestore();
    }
  });

  it("does not collapse expanded pages during automatic stale revalidation", async () => {
    let now = 1_000;
    const nowSpy = vi.spyOn(Date, "now").mockImplementation(() => now);
    const additionalItems = Array.from({ length: 25 }, (_, index): FormLibraryItem => ({
      template: {
        ...item.template,
        id: `template-extra-${index + 1}`,
        code: `EXTRA-${index + 1}`,
        name: `Extra form ${index + 1}`,
      },
    }));
    api.loadFormTemplatePage.mockReset();
    api.loadFormTemplatePage
      .mockResolvedValueOnce({ items: [item], next_cursor: "next", total: 26 })
      .mockResolvedValueOnce({ items: additionalItems, total: 26 });

    try {
      render(<FormsWorkspace/>);
      await screen.findByRole("button", { name: "Open Vendor due diligence" });
      fireEvent.click(screen.getByRole("button", { name: "Load more" }));
      await screen.findByRole("button", { name: "Open Extra form 25" });
      expect(api.loadFormTemplatePage.mock.calls.length).toBe(2);

      now += 31_000;
      window.dispatchEvent(new Event("focus"));
      await Promise.resolve();
      expect(api.loadFormTemplatePage.mock.calls.length).toBe(2);
      expect(screen.getByRole("button", { name: "Open Extra form 25" })).toBeTruthy();
    } finally {
      nowSpy.mockRestore();
    }
  });
});
