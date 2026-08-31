import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
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
});

describe("Forms workspace location state", () => {
  it("preserves a saved view after target navigation rewrites the hash", async () => {
    api.loadSavedFormViews.mockResolvedValueOnce([savedView]);
    const onTarget = vi.fn(() => window.history.replaceState(null, "", "#forms"));

    render(<FormsWorkspace targetID="template-a" onTarget={onTarget}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Active outsourcing" }));

    expect(onTarget).toHaveBeenCalledWith(undefined);
    expect(window.location.hash).toBe("#forms?search=outsourcing&status=ACTIVE&tag=third-party");
    await waitFor(() => expect(api.loadFormTemplatePage).toHaveBeenCalledWith(expect.objectContaining({ search: "outsourcing", status: "ACTIVE", tag: "third-party" }), expect.anything()));
  });

  it("clears filters and the selected target without restoring stale query state", async () => {
    window.history.replaceState(null, "", "#forms/template-missing?search=outsourcing&status=ACTIVE");
    api.loadFormTemplatePage.mockResolvedValue({ items: [] });
    const onTarget = vi.fn(() => window.history.replaceState(null, "", "#forms"));

    render(<FormsWorkspace targetID="template-missing" onTarget={onTarget}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Clear filters" }));

    expect(onTarget).toHaveBeenCalledWith(undefined);
    expect(window.location.hash).toBe("#forms");
    await waitFor(() => {
      const lastCall = api.loadFormTemplatePage.mock.calls.at(-1)?.[0];
      expect(lastCall).toMatchObject({ limit: 25 });
      expect(lastCall).not.toHaveProperty("search");
      expect(lastCall).not.toHaveProperty("status");
    });
  });
});
