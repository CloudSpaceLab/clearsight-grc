import axe from "axe-core";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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
    tags: ["third-party"],
  },
  active_version: 1,
  active_status: "ACTIVE",
};

beforeEach(() => {
  window.history.replaceState(null, "", "#forms");
  window.localStorage.clear();
  api.loadFormTemplatePage.mockReset();
  api.loadFormTemplatePage.mockImplementation(async (query: { search?: string }) => query.search === "outsourcing" ? { items: [] } : { items: [draftItem] });
  api.loadFormTemplateRevision.mockReset();
  api.loadFormTemplateRevision.mockResolvedValue(draftItem.template);
  api.loadReusableFormTemplateRefs.mockReset();
  api.loadReusableFormTemplateRefs.mockResolvedValue([{ id: "template-a", name: "Vendor due diligence", code: "VENDOR", version: 1 }]);
  api.loadStarterTemplates.mockReset();
  api.loadStarterTemplates.mockResolvedValue([]);
  api.loadSavedFormViews.mockReset();
  api.loadSavedFormViews.mockResolvedValue([]);
  api.createLibraryFormDraft.mockReset();
  api.createLibraryFormRevision.mockReset();
  api.instantiateStarterTemplate.mockReset();
  api.saveFormView.mockReset();
  api.deleteSavedFormView.mockReset();
  api.transitionFormTemplateRevision.mockReset();
  api.transitionFormTemplateRevision.mockResolvedValue({ ...draftItem.template, status: "PENDING_APPROVAL", version: 2 });
});

describe("Forms workspace", () => {
  it("keeps latest draft state separate from the exact reusable active revision", async () => {
    render(<FormsWorkspace targetID="template-a"/>);
    expect((await screen.findAllByText("Vendor due diligence")).length).toBeGreaterThan(0);
    const latest = screen.getByText("Latest stored").parentElement;
    const reusable = screen.getByText("Reusable now").parentElement;
    expect(latest?.textContent).toMatch(/Draft.*v2/);
    expect(reusable?.textContent).toMatch(/Active.*v1/);
  });

  it("offers a graphical recovery path when no search result matches", async () => {
    render(<FormsWorkspace initialSearch="outsourcing"/>);
    expect(await screen.findByText("No templates match “outsourcing”")).toBeTruthy();
    expect(screen.getByText("No matches")).toBeTruthy();
    expect(screen.getAllByRole("button", { name: "Create form template" }).length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: "Browse starter templates" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Clear filters" })).toBeTruthy();
  });

  it("applies and persists safe browser-local workspace styling", async () => {
    const view = render(<FormsWorkspace organizationName="Clear Bank" legalEntityName="Nigeria Bank" appearanceScope="entity-a"/>);
    expect((await screen.findAllByText("Vendor due diligence")).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByText("Style workspace"));
    fireEvent.change(screen.getByLabelText("Workspace accent color"), { target: { value: "#118844" } });
    await waitFor(() => expect((view.container.querySelector(".forms-workspace") as HTMLElement).style.getPropertyValue("--forms-accent")).toBe("#118844"));

    fireEvent.change(screen.getByLabelText("Bank or organization logo"), { target: { value: "/bank-logo.svg" } });
    fireEvent.click(screen.getByRole("button", { name: "Apply logo" }));
    expect(await screen.findByAltText("Clear Bank logo")).toBeTruthy();
    expect(window.localStorage.getItem("clearsight:forms:appearance:entity-a")).toContain("bank-logo.svg");
  });

  it("uses the full governed builder for new templates", async () => {
    api.createLibraryFormDraft.mockResolvedValueOnce({ ...draftItem.template, id: "template-new", version: 1 });
    render(<FormsWorkspace/>);
    expect((await screen.findAllByText("Vendor due diligence")).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole("button", { name: "Create form template" }));
    fireEvent.change(screen.getByLabelText("Code"), { target: { value: "OPS" } });
    fireEvent.change(screen.getByLabelText("Form name"), { target: { value: "Operations review" } });
    fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Collect current operating evidence." } });
    fireEvent.change(screen.getByLabelText("Question"), { target: { value: "Describe the current control." } });
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
    await waitFor(() => expect(api.createLibraryFormDraft).toHaveBeenCalledTimes(1));
    expect(api.createLibraryFormDraft.mock.calls[0]?.[0]).toMatchObject({
      code: "OPS",
      scoring_mode: "NONE",
      presentation: { default_mode: "AUTOMATIC", allow_mode_switch: false },
      fields: [{ label: "Describe the current control.", type: "short_text" }],
    });
  });

  it("edits a draft by creating an immutable next revision", async () => {
    api.createLibraryFormRevision.mockResolvedValueOnce({ ...draftItem.template, version: 3, purpose: "Collect current vendor evidence and attestations." });
    render(<FormsWorkspace targetID="template-a"/>);
    expect(await screen.findByRole("button", { name: "Edit draft" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Edit draft" }));
    fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Collect current vendor evidence and attestations." } });
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
    await waitFor(() => expect(api.createLibraryFormRevision).toHaveBeenCalledTimes(1));
    expect(api.createLibraryFormRevision.mock.calls[0]?.[0]).toBe("template-a");
    expect(api.createLibraryFormRevision.mock.calls[0]?.[1]).toBe(2);
    expect(api.createLibraryFormRevision.mock.calls[0]?.[2]).toMatchObject({ purpose: "Collect current vendor evidence and attestations." });
  });

  it("does not expose direct approval when a draft fails deterministic quality checks", async () => {
    const invalid: FormLibraryItem = {
      ...draftItem,
      template: {
        ...draftItem.template,
        scoring_mode: "COMPLIANCE",
        sections: [{ id: "identity", title: "Vendor identity", weight: 100 }],
        fields: [{
          id: "registered",
          section_id: "identity",
          label: "Registration verified",
          type: "yes_no",
          required: true,
          options: ["Yes", "No"],
          scoring: { weight: 80, answer_scores: { Yes: 100, No: 0 } },
        }],
      },
    };
    api.loadFormTemplatePage.mockResolvedValue({ items: [invalid] });
    render(<FormsWorkspace targetID="template-a"/>);
    const button = await screen.findByRole("button", { name: "Send for approval" });
    expect((button as HTMLButtonElement).disabled).toBe(true);
    expect(screen.getByText(/resolve approval-quality checks/i)).toBeTruthy();
  });

  it("passes semantic accessibility checks for the loaded library", async () => {
    const view = render(<FormsWorkspace targetID="template-a"/>);
    expect((await screen.findAllByText("Vendor due diligence")).length).toBeGreaterThan(0);
    const results = await axe.run(view.container, { rules: { "color-contrast": { enabled: false } } });
    expect(results.violations.map((violation) => violation.id)).toEqual([]);
  });
});
