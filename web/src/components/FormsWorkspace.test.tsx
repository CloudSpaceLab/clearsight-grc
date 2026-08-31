import axe from "axe-core";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { FormLibraryItem, StarterTemplate } from "../formsTypes";
import { appearanceStorageKey } from "./forms/formsAppearance";
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

const starter: StarterTemplate = {
  code: "VENDOR-SECURITY",
  catalog_version: 3,
  published_on: "2026-08-01",
  reference_label: "Reviewed vendor security pattern",
  template: {
    ...draftItem.template,
    id: "starter-vendor-security",
    code: "VENDOR-SECURITY",
    name: "Vendor security review",
    purpose: "Review core security safeguards before onboarding a critical provider.",
    sections: [{ id: "access", title: "Access controls" }, { id: "evidence", title: "Evidence" }],
    fields: [
      { id: "mfa", section_id: "access", label: "Is MFA enforced for privileged access?", type: "yes_no", required: true },
      { id: "pam", section_id: "access", label: "Describe privileged access monitoring.", type: "long_text", required: true },
      { id: "evidence", section_id: "evidence", label: "Upload supporting evidence.", type: "file", required: false },
    ],
    tags: ["vendor", "security"],
  },
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
  api.createAIFormProposal.mockReset();
  api.createAIFormRevisionProposal.mockReset();
  api.loadFormProposal.mockReset();
  api.acceptFormProposal.mockReset();
  api.rejectFormProposal.mockReset();
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

  it("offers one creation entry point when no search result matches", async () => {
    render(<FormsWorkspace initialSearch="outsourcing"/>);
    expect(await screen.findByText("No templates match “outsourcing”")).toBeTruthy();
    expect(screen.getByText("No matches")).toBeTruthy();
    expect(screen.getAllByRole("button", { name: "+ New form" }).length).toBeGreaterThan(0);
    expect(screen.queryByRole("button", { name: "Browse starter templates" })).toBeNull();
    expect(screen.getByRole("button", { name: "Clear filters" })).toBeTruthy();
  });

  it("loads configured workspace branding without exposing styling in the Forms task flow", async () => {
    window.localStorage.setItem(appearanceStorageKey("entity-a"), JSON.stringify({ accentColor: "#118844", logoURL: "/bank-logo.svg" }));
    const view = render(<FormsWorkspace organizationName="Clear Bank" legalEntityName="Nigeria Bank" appearanceScope="entity-a"/>);
    expect((await screen.findAllByText("Vendor due diligence")).length).toBeGreaterThan(0);
    await waitFor(() => expect((view.container.querySelector(".forms-workspace") as HTMLElement).style.getPropertyValue("--forms-accent")).toBe("#118844"));
    expect(await screen.findByAltText("Clear Bank logo")).toBeTruthy();
    expect(screen.queryByText("Style workspace")).toBeNull();
    expect(screen.getByText("Create, send and govern information requests.")).toBeTruthy();
  });

  it("opens Blank, Template, AI and Import from one New form launcher", async () => {
    api.loadStarterTemplates.mockResolvedValueOnce([starter]);
    render(<FormsWorkspace/>);
    await screen.findAllByText("Vendor due diligence");
    expect(screen.queryByRole("button", { name: "Draft with AI" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "+ New form" }));
    const dialog = await screen.findByRole("dialog", { name: "New form" });
    const launcher = within(dialog);
    for (const method of ["Blank form", "Draft with AI", "From template", "Import"]) {
      expect(launcher.getByRole("button", { name: new RegExp(`^${method}\\b`) })).toBeTruthy();
    }
    expect(launcher.getByText("Vendor security review")).toBeTruthy();
    expect(launcher.getByLabelText("Vendor security review preview")).toBeTruthy();
    expect(screen.queryByText("Style workspace")).toBeNull();
  });

  it("uses the full governed builder for a blank form", async () => {
    api.createLibraryFormDraft.mockResolvedValueOnce({ ...draftItem.template, id: "template-new", version: 1 });
    render(<FormsWorkspace/>);
    expect((await screen.findAllByText("Vendor due diligence")).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole("button", { name: "+ New form" }));
    const launcher = within(await screen.findByRole("dialog", { name: "New form" }));
    fireEvent.click(launcher.getByRole("button", { name: /^Blank form\b/ }));
    fireEvent.click(screen.getByRole("button", { name: "Overview" }));
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

  it("creates an ordinary governed draft from a starter template", async () => {
    api.loadStarterTemplates.mockResolvedValueOnce([starter]);
    api.instantiateStarterTemplate.mockResolvedValueOnce({ ...starter.template, id: "template-from-starter", version: 1, status: "DRAFT" });
    render(<FormsWorkspace/>);
    await screen.findAllByText("Vendor due diligence");
    fireEvent.click(screen.getByRole("button", { name: "+ New form" }));
    const launcher = within(await screen.findByRole("dialog", { name: "New form" }));
    fireEvent.click(launcher.getByRole("button", { name: "Use Vendor security review template" }));
    await waitFor(() => expect(api.instantiateStarterTemplate).toHaveBeenCalledWith("VENDOR-SECURITY"));
    expect(await screen.findByText("Vendor security review")).toBeTruthy();
  });

  it("opens governed AI authoring without replacing the manual builder", async () => {
    api.createAIFormProposal.mockResolvedValueOnce({
      id: "ai-proposal", source_kind: "AI", status: "REVIEW_REQUIRED",
      proposed_contract: { scoring_mode: "NONE", presentation: { default_mode: "AUTOMATIC", allow_mode_switch: true }, sections: [{ id: "general", title: "General" }], fields: [{ id: "ownership", section_id: "general", label: "Current ownership", type: "long_text", required: false }] },
      field_changes: [{ id: "change-ownership", kind: "ADD_FIELD", field: { id: "ownership", section_id: "general", label: "Current ownership", type: "long_text", required: false }, anchor: {}, confidence: .82 }], unresolved_items: [],
      provenance: { proposal_version: "FORM_AI_PROPOSAL_V1", source_document_id: "", source_sha256: "", source_version: 0, extraction_status: "NOT_APPLICABLE" },
      created_by: "author-1", created_at: "2026-08-29T08:00:00Z", updated_at: "2026-08-29T08:00:01Z", version: 2,
    });
    render(<FormsWorkspace/>);
    await screen.findAllByText("Vendor due diligence");
    fireEvent.click(screen.getByRole("button", { name: "+ New form" }));
    const launcher = within(await screen.findByRole("dialog", { name: "New form" }));
    fireEvent.click(launcher.getByRole("button", { name: /^Draft with AI\b/ }));
    expect(screen.getByRole("button", { name: "Open manual builder" })).toBeTruthy();
    fireEvent.change(screen.getByRole("textbox", { name: "What should this form collect or change?" }), { target: { value: "Collect current ownership details." } });
    fireEvent.click(screen.getByRole("button", { name: "Generate field proposal" }));
    expect(await screen.findByRole("heading", { name: "Review proposed form fields" })).toBeTruthy();
  });

  it("routes Import through the existing governed import workspace", async () => {
    render(<FormsWorkspace/>);
    await screen.findAllByText("Vendor due diligence");
    fireEvent.click(screen.getByRole("button", { name: "+ New form" }));
    const launcher = within(await screen.findByRole("dialog", { name: "New form" }));
    fireEvent.click(launcher.getByRole("button", { name: /^Import\b/ }));
    expect(window.location.hash).toBe("#imports");
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
