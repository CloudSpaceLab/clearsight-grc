import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReportDefinition } from "../reportingTypes";
import { ReportingPage, type ReportingPageProps } from "./ReportingPage";

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
  onDefinitionChange: vi.fn(),
  onGenerate: vi.fn(),
}));

beforeEach(() => {
  vi.resetAllMocks();
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

function renderPage(props: Partial<ReportingPageProps> = {}) {
  return render(<ReportingPage
    embedded
    organizationName="Clear Bank"
    legalEntityName="Clear Bank Nigeria"
    loadDefinitions={api.loadDefinitions}
    createDefinition={api.createDefinition}
    transitionDefinition={api.transitionDefinition}
    loadMatters={api.loadMatters}
    onDefinitionChange={api.onDefinitionChange}
    onGenerate={api.onGenerate}
    {...props}
  />);
}

async function choose(field: string, option: string | RegExp) {
  const trigger = screen.getByRole("button", { name: new RegExp(field) });
  await waitFor(() => expect(trigger).toHaveProperty("disabled", false));
  fireEvent.click(trigger);
  fireEvent.click(await screen.findByRole("option", { name: option }));
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: Error) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}

describe("ReportingPage builder", () => {
  it("starts with business-facing setup fields and lets the operator select a saved setup", async () => {
    renderPage();

    expect(screen.getByLabelText(/^Name/)).toHaveProperty("value", "");
    expect(screen.getByRole("button", { name: /Area/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /Report type/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Save setup" })).toBeTruthy();
    await waitFor(() => expect(api.loadDefinitions).toHaveBeenCalledWith(true, expect.any(AbortSignal)));
    await choose("Saved setup", /^Outstanding work/);
    expect(screen.getByRole("button", { name: "Send for review" })).toBeTruthy();
    expect(screen.queryByLabelText(/dataset|scope identifier/i)).toBeNull();
    expect(screen.queryByText(draft.id)).toBeNull();
    expect(api.createDefinition).not.toHaveBeenCalled();
  });

  it("creates the supported overview payload and hands the returned draft back to its workspace", async () => {
    renderPage();
    fireEvent.change(screen.getByLabelText(/^Name/), { target: { value: "Board vendor summary" } });
    const save = screen.getByRole("button", { name: "Save setup" });
    await waitFor(() => expect(save).toHaveProperty("disabled", false));
    fireEvent.click(save);

    await waitFor(() => expect(api.createDefinition).toHaveBeenCalledTimes(1));
    expect(api.createDefinition.mock.calls[0]?.[0]).toEqual({
      code: expect.stringMatching(/^BOARD_VENDOR_SUMMARY_[0-9A-F]{8}$/),
      name: "Board vendor summary",
      description: "Current vendors overview for the legal entity.",
      dataset: "VENDORS",
      scope_kind: "LEGAL_ENTITY",
      format: "XLSX",
      filter: { kind: "group", operator: "and", children: [] },
    });
    const created = await api.createDefinition.mock.results[0]?.value;
    expect(api.onDefinitionChange).toHaveBeenCalledExactlyOnceWith(created);
    expect(screen.getByRole("button", { name: "Send for review" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Generate report" })).toBeNull();
  });

  it("creates a governed board brief setup by named Issue without showing its raw ID", async () => {
    renderPage();
    fireEvent.change(screen.getByLabelText(/^Name/), { target: { value: "Settlement board brief" } });
    await choose("Area", /^Board brief/);

    await waitFor(() => expect(api.loadMatters).toHaveBeenCalledWith({ q: "", limit: 20 }));
    fireEvent.click(screen.getByRole("button", { name: "Save setup" }));
    expect(await screen.findByText("Choose an issue for the board brief.")).toBeTruthy();
    expect(api.createDefinition).not.toHaveBeenCalled();
    expect(screen.queryByText("matter-82bf")).toBeNull();

    await choose("Choose an issue", "MAT-82BF — Settlement exception");
    fireEvent.click(screen.getByRole("button", { name: "Save setup" }));

    await waitFor(() => expect(api.createDefinition).toHaveBeenCalledTimes(1));
    expect(api.createDefinition.mock.calls[0]?.[0]).toEqual({
      code: expect.stringMatching(/^SETTLEMENT_BOARD_BRIEF_[0-9A-F]{8}$/),
      name: "Settlement board brief",
      description: "Governed executive brief for one issue or change.",
      dataset: "MATTER_BOARD_BRIEF",
      scope_kind: "MATTER",
      scope_ref: "matter-82bf",
      format: "PDF",
      filter: { kind: "group", operator: "and", children: [] },
    });
  });

  it("uses each returned version and checksum through review and activation before generating", async () => {
    const submitted = definition({ ...draft, status: "PENDING_REVIEW", version: 7, checksum: "b".repeat(64) });
    const reviewed = definition({ ...submitted, status: "REVIEWED", version: 11, checksum: "c".repeat(64), reviewer_id: "reviewer-2" });
    const activated = definition({ ...reviewed, status: "ACTIVE", effective: true, version: 15, current_version: 3, checksum: "d".repeat(64), checker_id: "checker-2" });
    api.transitionDefinition.mockResolvedValueOnce(submitted).mockResolvedValueOnce(reviewed).mockResolvedValueOnce(activated);
    renderPage();
    await choose("Saved setup", /^Outstanding work/);

    fireEvent.click(screen.getByRole("button", { name: "Send for review" }));
    fireEvent.click(await screen.findByRole("button", { name: "Record review" }));
    fireEvent.click(await screen.findByRole("button", { name: "Activate setup" }));
    const generate = await screen.findByRole("button", { name: "Generate report" });

    expect(api.transitionDefinition.mock.calls).toEqual([
      [draft.id, "submit", { expected_version: draft.version, checksum_seen: draft.checksum }],
      [draft.id, "review", { expected_version: submitted.version, checksum_seen: submitted.checksum }],
      [draft.id, "activate", { expected_version: reviewed.version, checksum_seen: reviewed.checksum }],
    ]);
    expect(api.onDefinitionChange.mock.calls).toEqual([[submitted], [reviewed], [activated]]);
    fireEvent.click(generate);
    expect(api.onGenerate).toHaveBeenCalledExactlyOnceWith(activated);
  });

  it("prevents duplicate saves and preserves the entered setup after a failed command", async () => {
    const pending = deferred<ReportDefinition>();
    api.createDefinition.mockReturnValueOnce(pending.promise);
    renderPage();
    fireEvent.change(screen.getByLabelText(/^Name/), { target: { value: "Vendor issues for review" } });
    await choose("Report type", /^Exceptions & outstanding/);
    const save = screen.getByRole("button", { name: "Save setup" });
    act(() => {
      fireEvent.click(save);
      fireEvent.click(save);
    });
    expect(api.createDefinition).toHaveBeenCalledTimes(1);
    expect(save).toHaveProperty("disabled", true);
    expect(screen.getByLabelText(/^Name/)).toHaveProperty("disabled", true);

    await act(async () => pending.reject(new Error("Setup authority is unavailable.")));
    expect(await screen.findByText("Setup authority is unavailable.")).toBeTruthy();
    expect(screen.getByLabelText(/^Name/)).toHaveProperty("value", "Vendor issues for review");
    expect(screen.getByRole("button", { name: /Report type/ }).textContent).toContain("Exceptions & outstanding");
    expect(api.onDefinitionChange).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Save setup" }));
    await waitFor(() => expect(api.onDefinitionChange).toHaveBeenCalledTimes(1));
    expect(api.createDefinition).toHaveBeenCalledTimes(2);
    expect(api.createDefinition.mock.calls[1]?.[0]).toMatchObject({
      name: "Vendor issues for review", dataset: "VENDORS",
      filter: { kind: "group", operator: "or", children: expect.arrayContaining([{ kind: "condition", field: "status", operator: "is", value: "UNDER_REVIEW" }]) },
    });
  });

  it("does not advance or duplicate a transition while pending or after a version conflict", async () => {
    const pending = deferred<ReportDefinition>();
    api.transitionDefinition.mockReturnValueOnce(pending.promise);
    renderPage();
    await choose("Saved setup", /^Outstanding work/);
    const submit = screen.getByRole("button", { name: "Send for review" });
    act(() => {
      fireEvent.click(submit);
      fireEvent.click(submit);
    });
    expect(api.transitionDefinition).toHaveBeenCalledTimes(1);
    expect(submit).toHaveProperty("disabled", true);
    expect(screen.getByRole("button", { name: /Saved setup/ })).toHaveProperty("disabled", true);
    expect(screen.queryByRole("button", { name: "Record review" })).toBeNull();

    await act(async () => pending.reject(new Error("The setup has changed. Refresh before retrying.")));
    expect(await screen.findByText("The setup has changed. Refresh before retrying.")).toBeTruthy();
    expect(api.onDefinitionChange).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Send for review" })).toHaveProperty("disabled", false);
    expect(screen.queryByRole("button", { name: "Record review" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Generate report" })).toBeNull();
  });

  it("blocks generation for an active setup that is not effective", async () => {
    api.loadDefinitions.mockResolvedValue([{ ...active, effective: false }]);
    renderPage();
    await choose("Saved setup", /^Monthly vendor overview/);
    const generate = screen.getByRole("button", { name: "Generate report" });
    expect(generate).toHaveProperty("disabled", true);
    fireEvent.click(generate);
    expect(api.onGenerate).not.toHaveBeenCalled();
  });
});
