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
}));

beforeEach(() => {
  vi.clearAllMocks();
  api.loadDefinitions.mockResolvedValue([active, draft]);
  api.createDefinition.mockImplementation(async (input) => definition({
    id: "setup-created",
    code: input.code,
    name: input.name,
    description: input.description ?? "",
    dataset: input.dataset,
    scope_kind: input.scope_kind,
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
  />);
}

describe("ReportingPage saved setups", () => {
  it("shows business-facing setup fields instead of the low-level report definition model", async () => {
    renderPage();

    expect(await screen.findByRole("heading", { name: "Saved report setups" })).toBeTruthy();
    const table = screen.getByRole("table", { name: "Saved report setups" });
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
    await screen.findByRole("table", { name: "Saved report setups" });

    fireEvent.click(screen.getByRole("button", { name: "New setup" }));
    expect(screen.getByRole("heading", { name: "What should this report show?" })).toBeTruthy();
    expect(screen.getByText(/No dataset IDs, report codes, file formats or effective dates/i)).toBeTruthy();

    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "Board vendor summary" } });
    fireEvent.click(screen.getByRole("button", { name: "Save setup" }));

    await waitFor(() => expect(api.createDefinition).toHaveBeenCalledTimes(1));
    expect(api.createDefinition.mock.calls[0]?.[0]).toMatchObject({
      name: "Board vendor summary",
      dataset: "VENDORS",
      scope_kind: "LEGAL_ENTITY",
      format: "XLSX",
      filter: { kind: "group", operator: "and", children: [] },
    });
    expect(String(api.createDefinition.mock.calls[0]?.[0]?.code)).toMatch(/^VENDORS_OVERVIEW_[0-9A-F]{8}$/);
  });

  it("keeps approval as a short contextual next step instead of exposing governance internals", async () => {
    renderPage();
    const table = await screen.findByRole("table", { name: "Saved report setups" });
    fireEvent.click(within(table).getByRole("row", { name: "Outstanding work" }));

    const detail = screen.getByRole("complementary", { name: "Selected report setup" });
    expect(within(detail).getByText(/saved but not yet available/i)).toBeTruthy();
    fireEvent.click(within(detail).getByRole("button", { name: "Send for review" }));

    await waitFor(() => expect(api.transitionDefinition).toHaveBeenCalledWith(
      draft.id,
      "submit",
      { expected_version: draft.version, checksum_seen: draft.checksum },
    ));
  });
});
