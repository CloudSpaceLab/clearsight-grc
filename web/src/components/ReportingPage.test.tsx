import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReportDefinition } from "../reportingTypes";
import { ReportingPage } from "./ReportingPage";

function definition(overrides: Partial<ReportDefinition> = {}): ReportDefinition {
  return {
    id: "setup-active",
    tenant_id: "tenant-1",
    legal_entity_id: "entity-1",
    code: "VENDORS_OVERVIEW_12345678",
    name: "Monthly vendor overview",
    description: "Current vendors overview for the legal entity.",
    dataset: "VENDORS",
    scope_kind: "LEGAL_ENTITY",
    format: "XLSX",
    filter: { kind: "group", operator: "and", children: [] },
    status: "ACTIVE",
    current_version: 1,
    effective: true,
    checksum: "a".repeat(64),
    maker_id: "maker-1",
    reviewer_id: "reviewer-1",
    checker_id: "checker-1",
    effective_from: "2026-09-25T08:00:00Z",
    approved_at: "2026-09-25T08:00:00Z",
    created_at: "2026-09-24T08:00:00Z",
    updated_at: "2026-09-25T08:00:00Z",
    version: 4,
    ...overrides,
  };
}

const active = definition();
const draft = definition({
  id: "setup-draft",
  code: "WORK_ATTENTION_12345678",
  name: "Outstanding work",
  dataset: "MATTER_EXCEPTIONS",
  status: "DRAFT",
  effective: false,
  reviewer_id: undefined,
  checker_id: undefined,
  effective_from: undefined,
  approved_at: undefined,
  version: 1,
});

const api = vi.hoisted(() => ({
  loadDefinitions: vi.fn(),
  createDefinition: vi.fn(),
  transitionDefinition: vi.fn(),
  loadMatters: vi.fn(),
}));

beforeEach(() => {
  vi.clearAllMocks();
  api.loadDefinitions.mockResolvedValue([active, draft]);
  api.loadMatters.mockResolvedValue({
    items: [{
      matter: {
        id: "matter-82bf", tenant_id: "tenant-1", legal_entity_id: "entity-1",
        reference: "MAT-82BF", type: "INCIDENT", status: "ASSESSMENT", priority: 4,
        title: "Settlement exception", summary: "Review current settlement controls.", scope: {},
        known_facts: {}, missing_facts: [], contradictions: [], created_at: "2026-10-06T10:00:00Z",
        updated_at: "2026-10-06T10:00:00Z", version: 3,
      },
      type_label: "Incident", status_label: "Assessment", next_action: "Review current settlement controls.",
      program_count: 1, open_action_count: 2, outcome_check_count: 0,
    }],
    generated_at: "2026-10-06T18:00:00Z",
  });
  api.createDefinition.mockImplementation(async (input) => definition({
    id: "setup-created",
    code: input.code,
    name: input.name,
    description: input.description ?? "",
    dataset: input.dataset,
    scope_kind: input.scope_kind,
    scope_ref: input.scope_ref,
    format: input.format,
    filter: input.filter,
    status: "DRAFT",
    effective: false,
    reviewer_id: undefined,
    checker_id: undefined,
    effective_from: undefined,
    approved_at: undefined,
    version: 1,
  }));
  api.transitionDefinition.mockImplementation(async (_id, action) => definition({
    ...draft,
    status: action === "submit" ? "PENDING_REVIEW" : draft.status,
    version: 2,
  }));
});

function renderPage() {
  return render(<ReportingPage
    embedded
    organizationName="Clear Bank"
    legalEntityName="Clear Bank Nigeria"
    loadDefinitions={api.loadDefinitions}
    createDefinition={api.createDefinition}
    transitionDefinition={api.transitionDefinition}
    loadMatters={api.loadMatters}
  />);
}

describe("ReportingPage saved setups", () => {
  it("shows business-facing setup fields instead of the low-level report definition model", async () => {
    renderPage();

    expect(await screen.findByRole("heading", { name: "Saved setups" })).toBeTruthy();
    const table = screen.getByRole("table", { name: "Saved setups" });
    expect(within(table).getByRole("row", { name: "Monthly vendor overview" })).toBeTruthy();
    expect(within(table).getByRole("row", { name: "Outstanding work" })).toBeTruthy();

    expect(screen.getAllByText("Vendors").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Overview").length).toBeGreaterThan(0);
    expect(screen.getByText("Exceptions & outstanding")).toBeTruthy();
    expect(screen.queryByText(/dataset/i)).toBeNull();
    expect(screen.queryByText(/scope identifier/i)).toBeNull();
    expect(screen.queryByText(/file format/i)).toBeNull();
  });

  it("creates a reusable overview with only a name and intent visible to the operator", async () => {
    renderPage();
    await screen.findByRole("table", { name: "Saved setups" });

    fireEvent.click(screen.getByRole("button", { name: "New setup" }));
    expect(screen.getByRole("heading", { name: "New report setup" })).toBeTruthy();
    
    fireEvent.change(screen.getByLabelText(/^Name/), { target: { value: "Board vendor summary" } });
    fireEvent.click(screen.getByRole("button", { name: "Save setup" }));

    await waitFor(() => expect(api.createDefinition).toHaveBeenCalledTimes(1));
    expect(api.createDefinition.mock.calls[0]?.[0]).toMatchObject({
      name: "Board vendor summary",
      dataset: "VENDORS",
      scope_kind: "LEGAL_ENTITY",
      format: "XLSX",
      filter: { kind: "group", operator: "and", children: [] },
    });
    expect(String(api.createDefinition.mock.calls[0]?.[0]?.code)).toMatch(/^BOARD_VENDOR_SUMMARY_[0-9A-F]{8}$/);
  });

  it("creates a governed board brief setup by named Issue without showing its raw ID", async () => {
    renderPage();
    await screen.findByRole("table", { name: "Saved setups" });

    fireEvent.click(screen.getByRole("button", { name: "New setup" }));
    fireEvent.change(screen.getByLabelText(/^Name/), { target: { value: "Settlement board brief" } });

    fireEvent.click(screen.getByRole("button", { name: /Vendors/ }));
    fireEvent.click(await screen.findByRole("option", { name: /Board brief/ }));

    await waitFor(() => expect(api.loadMatters).toHaveBeenCalledWith({ q: "", limit: 20 }));
    expect(await screen.findByText("MAT-82BF — Settlement exception")).toBeTruthy();
    expect(screen.queryByText("matter-82bf")).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: /Choose an issue/ }));
    fireEvent.click(await screen.findByRole("option", { name: "MAT-82BF — Settlement exception" }));
    fireEvent.click(screen.getByRole("button", { name: "Save setup" }));

    await waitFor(() => expect(api.createDefinition).toHaveBeenCalledTimes(1));
    expect(api.createDefinition.mock.calls[0]?.[0]).toMatchObject({
      dataset: "MATTER_BOARD_BRIEF",
      scope_kind: "MATTER",
      scope_ref: "matter-82bf",
      format: "PDF",
    });
  });

  it("keeps approval as a short contextual next step instead of exposing governance internals", async () => {
    renderPage();
    const table = await screen.findByRole("table", { name: "Saved setups" });
    fireEvent.click(within(table).getByRole("row", { name: "Outstanding work" }));

    const detail = screen.getByRole("complementary", { name: "Selected report setup" });
    expect(within(detail).getByText("Send for review.")).toBeTruthy();
    fireEvent.click(within(detail).getByRole("button", { name: "Send for review" }));

    await waitFor(() => expect(api.transitionDefinition).toHaveBeenCalledWith(
      draft.id,
      "submit",
      { expected_version: draft.version, checksum_seen: draft.checksum },
    ));
  });
});
