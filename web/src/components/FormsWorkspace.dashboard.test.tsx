import { useState } from "react";
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
    render(<FormsWorkspace/>);
    expect(await screen.findByRole("button", { name: "Open Vendor due diligence" })).toBeTruthy();
    expect(screen.queryByLabelText("Selected form template")).toBeNull();
    expect(screen.queryByText("Recently updated")).toBeNull();
    expect(screen.queryByRole("button", { name: "Cards" })).toBeNull();
  });

  it("opens template detail contextually and dismisses it without changing library data", async () => {
    function Harness() {
      const [target, setTarget] = useState<string>();
      return <FormsWorkspace targetID={target} onTarget={setTarget}/>;
    }

    render(<Harness/>);
    fireEvent.click(await screen.findByRole("button", { name: "Open Vendor due diligence" }));
    const drawer = await screen.findByLabelText("Selected form template");
    expect(drawer.textContent).toMatch(/Latest stored.*Draft.*v2/);
    expect(drawer.textContent).toMatch(/Reusable now.*Active.*v1/);
    expect(screen.getByRole("button", { name: "Close form detail" })).toBeTruthy();
    fireEvent.keyDown(window, { key: "Escape" });
    await waitFor(() => expect(screen.queryByLabelText("Selected form template")).toBeNull());
    expect(screen.getByRole("button", { name: "Open Vendor due diligence" })).toBeTruthy();
  });

  it("keeps saved-view controls quiet until the current query is customized", async () => {
    render(<FormsWorkspace/>);
    await screen.findByRole("button", { name: "Open Vendor due diligence" });
    expect(screen.queryByRole("button", { name: "Save view" })).toBeNull();
    fireEvent.change(screen.getByRole("searchbox"), { target: { value: "vendor" } });
    expect(await screen.findByRole("button", { name: "Save view" })).toBeTruthy();
  });
});
