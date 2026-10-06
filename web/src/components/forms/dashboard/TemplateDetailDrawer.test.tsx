import { render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { loadMatter } from "../../../api";
import type { FormLibraryItem } from "../../../formsTypes";
import { TemplateDetailDrawer } from "./TemplateDetailDrawer";

vi.mock("../../../api", () => ({ loadMatter: vi.fn() }));

const item = {
  template: {
    id: "form-1",
    tenant_id: "bank-1",
    legal_entity_id: "entity-1",
    origin: { type: "MATTER", id: "matter-1" },
    code: "ISSUE-CHECK",
    name: "Issue evidence check",
    purpose: "Collect current issue evidence.",
    sensitivity: "INTERNAL",
    scoring_mode: "NONE",
    presentation: { default_mode: "AUTOMATIC", allow_mode_switch: false },
    sections: [{ id: "evidence", title: "Evidence" }],
    fields: [{ id: "state", section_id: "evidence", label: "Current state", type: "short_text", required: true }],
    status: "DRAFT",
    is_current: false,
    version: 1,
    created_at: "2026-10-06T18:00:00Z",
    updated_at: "2026-10-06T18:00:00Z",
  },
  authority_available: true,
  operations: [],
} as FormLibraryItem;

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(loadMatter).mockResolvedValue({
    type_label: "Incident",
    status_label: "Assessment",
    next_action: "",
    matter: {
      id: "matter-1",
      tenant_id: "bank-1",
      legal_entity_id: "entity-1",
      reference: "MAT-82BF",
      type: "INCIDENT",
      status: "ASSESSMENT",
      priority: 3,
      title: "Settlement exception",
      summary: "Review the settlement exception.",
      scope: {},
      known_facts: {},
      missing_facts: [],
      contradictions: [],
      created_at: "2026-10-06T17:00:00Z",
      updated_at: "2026-10-06T17:00:00Z",
      version: 1,
    },
    links: [],
    decisions: [],
    actions: [],
    verification_contracts: [],
    verification_results: [],
    response_packages: [],
    closure: { ready: false, reasons: [] },
  });
});

it("resolves the originating issue only for the selected form detail", async () => {
  render(<TemplateDetailDrawer
    item={item}
    requestedID="form-1"
    busy={null}
    onClose={vi.fn()}
    onClearFilters={vi.fn()}
    onEdit={vi.fn()}
    onTransition={vi.fn()}
  />);

  expect(await screen.findByText("MAT-82BF")).toBeTruthy();
  expect(loadMatter).toHaveBeenCalledWith("matter-1");
  expect(screen.queryByText("matter-1")).toBeNull();
  expect(screen.getByText("Originating issue")).toBeTruthy();
});
