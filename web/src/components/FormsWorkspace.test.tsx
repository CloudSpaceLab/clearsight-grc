import axe from "axe-core";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { FormsWorkspace } from "./FormsWorkspace";
import type { FormLibraryItem } from "../formsTypes";

const api = vi.hoisted(() => ({
  loadFormTemplatePage: vi.fn(),
  loadStarterTemplates: vi.fn(),
  loadSavedFormViews: vi.fn(),
  createLibraryFormDraft: vi.fn(),
  instantiateStarterTemplate: vi.fn(),
  saveFormView: vi.fn(),
  deleteSavedFormView: vi.fn(),
  transitionFormTemplateRevision: vi.fn(),
}));

vi.mock("../formsApi", () => api);

const draftItem: FormLibraryItem = {
  template: {
    id: "template-a", tenant_id: "bank-a", legal_entity_id: "entity-a", code: "VENDOR", name: "Vendor due diligence", purpose: "Collect current vendor evidence.",
    status: "DRAFT", is_current: false, version: 2, created_at: "2026-08-27T09:00:00Z", updated_at: "2026-08-27T10:00:00Z",
    sensitivity: "INTERNAL", scoring_mode: "NONE", presentation: { default_mode: "AUTOMATIC", allow_mode_switch: true }, sections: [{ id: "general", title: "General" }],
    fields: [{ id: "question_1", section_id: "general", label: "Registered name", type: "short_text", required: true }],
    tags: ["third-party"],
  },
  active_version: 1,
  active_status: "ACTIVE",
};

beforeEach(() => {
  window.history.replaceState(null, "", "#forms");
  api.loadFormTemplatePage.mockReset();
  api.loadFormTemplatePage.mockImplementation(async (query: { search?: string }) => query.search === "outsourcing" ? { items: [] } : { items: [draftItem] });
  api.loadStarterTemplates.mockReset();
  api.loadStarterTemplates.mockResolvedValue([]);
  api.loadSavedFormViews.mockReset();
  api.loadSavedFormViews.mockResolvedValue([]);
  api.createLibraryFormDraft.mockReset();
  api.instantiateStarterTemplate.mockReset();
  api.saveFormView.mockReset();
  api.deleteSavedFormView.mockReset();
  api.transitionFormTemplateRevision.mockReset();
  api.transitionFormTemplateRevision.mockResolvedValue({ ...draftItem.template, status: "PENDING_APPROVAL", version: 2 });
});

describe("Forms workspace", () => {
  it("keeps latest draft state separate from the exact reusable active revision", async () => {
    render(<FormsWorkspace targetID="template-a"/>);
    expect(await screen.findByText("Vendor due diligence")).toBeTruthy();
    const latest = screen.getByText("Latest stored").parentElement;
    const reusable = screen.getByText("Reusable now").parentElement;
    expect(latest?.textContent).toMatch(/Draft.*v2/);
    expect(reusable?.textContent).toMatch(/Active.*v1/);
  });

  it("states the bounded legal-entity population when no search result matches", async () => {
    render(<FormsWorkspace initialSearch="outsourcing"/>);
    expect(await screen.findByText("No form templates match ‘outsourcing’ in this legal entity.")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Create form template" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Use a starter template" })).toBeEnabled();
  });

  it("creates a valid governed draft rather than exposing an enabled no-op", async () => {
    api.createLibraryFormDraft.mockResolvedValueOnce({ ...draftItem.template, id: "template-new", version: 1 });
    render(<FormsWorkspace/>);
    await screen.findByText("Vendor due diligence");
    fireEvent.click(screen.getByRole("button", { name: "Create form template" }));
    fireEvent.change(screen.getByLabelText("Code"), { target: { value: "OPS" } });
    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "Operations review" } });
    fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Collect current operating evidence." } });
    fireEvent.change(screen.getByLabelText("First question"), { target: { value: "Describe the current control." } });
    fireEvent.click(screen.getByRole("button", { name: "Create draft" }));
    await waitFor(() => expect(api.createLibraryFormDraft).toHaveBeenCalledTimes(1));
    expect(api.createLibraryFormDraft.mock.calls[0]?.[0]).toMatchObject({ code: "OPS", scoring_mode: "NONE", fields: [{ label: "Describe the current control." }] });
  });

  it("passes semantic accessibility checks for the loaded library", async () => {
    const view = render(<FormsWorkspace targetID="template-a"/>);
    await screen.findByText("Vendor due diligence");
    const results = await axe.run(view.container, { rules: { "color-contrast": { enabled: false } } });
    expect(results.violations.map((violation) => violation.id)).toEqual([]);
  });
});
