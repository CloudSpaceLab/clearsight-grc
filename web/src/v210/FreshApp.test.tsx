import { act, fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import axe from "axe-core";
import { FreshApp } from "./FreshApp";
import { ApiError } from "../http";

const api = vi.hoisted(() => ({ loadContext: vi.fn(), loadSessionStatus: vi.fn(), loadDemoAccounts: vi.fn(), loginDemo: vi.fn(), loadProgramSummaries: vi.fn(), loadProgram: vi.fn(), loadMatter: vi.fn(), loadActorWorkDueCounts: vi.fn() }));
const metrics = vi.hoisted(() => ({ loadHomeMetrics: vi.fn() }));
const oversight = vi.hoisted(() => ({ loadOversight: vi.fn() }));
const forms = vi.hoisted(() => ({ loadFormTemplatePage: vi.fn(), loadFormTemplateItem: vi.fn(), transitionFormTemplateRevision: vi.fn() }));
const documents = vi.hoisted(() => ({ loadDocuments: vi.fn(), documentContentURL: vi.fn(), documentEligibility: vi.fn() }));
vi.mock("../api", () => api);
vi.mock("../metricApi", () => metrics);
vi.mock("../oversightApi", () => oversight);
vi.mock("../formsApi", () => forms);
vi.mock("../submittedDocumentApi", () => documents);

it("removes form records when access is denied while loading another page", async () => {
  forms.loadFormTemplatePage.mockReset();
  forms.loadFormTemplatePage.mockResolvedValueOnce({ items: [{ template: { id: "private", name: "Private form", status: "ACTIVE", version: 1 } }], next_cursor: "next" });
  forms.loadFormTemplatePage.mockRejectedValueOnce(new ApiError(403, "Access denied"));
  window.location.hash = "#/forms";
  render(<FreshApp/>);
  expect(await screen.findByText("Private form")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Forms unavailable.");
  expect(screen.queryByText("Private form")).toBeNull();
});

it("retains form records and retries after a temporary pagination failure", async () => {
  forms.loadFormTemplatePage.mockReset();
  forms.loadFormTemplatePage.mockResolvedValueOnce({ items: [{ template: { id: "first", name: "First form", status: "ACTIVE", version: 1 } }], next_cursor: "next" });
  forms.loadFormTemplatePage.mockRejectedValueOnce(new ApiError(503, "Unavailable"));
  forms.loadFormTemplatePage.mockResolvedValueOnce({ items: [{ template: { id: "second", name: "Second form", status: "ACTIVE", version: 1 } }] });
  window.location.hash = "#/forms";
  render(<FreshApp/>);
  expect(await screen.findByText("First form")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  expect((await screen.findByRole("alert")).textContent).toContain("More forms unavailable.");
  expect(screen.getByText("First form")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  expect(await screen.findByText("Second form")).toBeTruthy();
  expect(forms.loadFormTemplatePage).toHaveBeenLastCalledWith({ search: "", limit: 20, cursor: "next" });
});

it("removes program records when access is denied while loading another page", async () => {
  api.loadProgramSummaries.mockReset();
  api.loadProgramSummaries.mockResolvedValueOnce({ items: [{ program: { id: "private", name: "Private program", code: "P-1" }, requirement_count: 0, open_matter_count: 0, state_label: "Review due" }], next_cursor: "next" });
  api.loadProgramSummaries.mockRejectedValueOnce(new ApiError(403, "Access denied"));
  window.location.hash = "#/programs";
  render(<FreshApp/>);
  expect(await screen.findByText("Private program")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Programs unavailable.");
  expect(screen.queryByText("Private program")).toBeNull();
});

it("hides filters when program summaries are unavailable and lets the user clear a failed filter", async () => {
  api.loadProgramSummaries.mockReset();
  api.loadProgramSummaries.mockRejectedValueOnce(new ApiError(503, "Unavailable"));
  api.loadProgramSummaries.mockResolvedValueOnce({ items: [] });
  window.location.hash = "#/programs?q=access&overall_state=CURRENT";
  render(<FreshApp/>);
  expect((await screen.findByRole("alert")).textContent).toContain("Programs unavailable.");
  expect(screen.queryByLabelText("Assessed status")).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
  expect(await screen.findByText("0 loaded")).toBeTruthy();
  expect(api.loadProgramSummaries).toHaveBeenLastCalledWith({ q: "", limit: 20 });
  expect(window.location.hash).toBe("#/programs");
});

it("does not append a forms page that repeats its cursor", async () => {
  forms.loadFormTemplatePage.mockReset();
  const row = { template: { id: "first", name: "First form", status: "ACTIVE", version: 1 } };
  forms.loadFormTemplatePage.mockResolvedValueOnce({ items: [row], next_cursor: "next" }).mockResolvedValueOnce({ items: [row], next_cursor: "next" });
  window.location.hash = "#/forms";
  render(<FreshApp/>);
  expect(await screen.findByText("First form")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  expect((await screen.findByRole("alert")).textContent).toContain("More forms unavailable.");
  expect(screen.getAllByText("First form")).toHaveLength(1);
});

it("requests a forms page only once when load more is activated twice", async () => {
  forms.loadFormTemplatePage.mockReset();
  forms.loadFormTemplatePage.mockResolvedValueOnce({ items: [{ template: { id: "first", name: "First form", status: "ACTIVE", version: 1 } }], next_cursor: "next" });
  let finish!: (value: { items: never[] }) => void;
  forms.loadFormTemplatePage.mockReturnValueOnce(new Promise((resolve) => { finish = resolve; }));
  window.location.hash = "#/forms";
  render(<FreshApp/>);
  expect(await screen.findByText("First form")).toBeTruthy();
  const button = screen.getByRole("button", { name: "Load more" });
  act(() => { button.dispatchEvent(new MouseEvent("click", { bubbles: true })); button.dispatchEvent(new MouseEvent("click", { bubbles: true })); });
  expect(forms.loadFormTemplatePage).toHaveBeenCalledTimes(2);
  await act(async () => { finish({ items: [] }); });
});

it("does not append a program page that repeats its cursor", async () => {
  api.loadProgramSummaries.mockReset();
  const row = { program: { id: "first", name: "First program", code: "P1" }, requirement_count: 0, open_matter_count: 0, state_label: "Review due" };
  api.loadProgramSummaries.mockResolvedValueOnce({ items: [row], next_cursor: "next" }).mockResolvedValueOnce({ items: [row], next_cursor: "next" });
  window.location.hash = "#/programs";
  render(<FreshApp/>);
  expect(await screen.findByText("First program")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  expect((await screen.findByRole("alert")).textContent).toContain("More programs unavailable.");
  expect(screen.getAllByText("First program")).toHaveLength(1);
});

beforeEach(() => {
  vi.clearAllMocks(); window.location.hash = "";
  api.loadSessionStatus.mockResolvedValue({ authenticated: true, demo_login_available: false });
  api.loadContext.mockResolvedValue({ tenant: { id: "bank", name: "Meridian Bank" }, legal_entity: { id: "entity", name: "Meridian Bank" }, actor: { id: "person", name: "Ada Okafor" }, capabilities: {} });
  api.loadActorWorkDueCounts.mockResolvedValue({ scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "person" }, coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "SOURCE_SNAPSHOT", as_of: "2026-11-20T10:00:00Z", total: 0, overdue: 0, due_soon: 0, undated: 0 });
  metrics.loadHomeMetrics.mockResolvedValue({ scope_kind: "LEGAL_ENTITY", scope_id: "entity", source_revision: "rev-1", reporting_period: { start_date: "2026-10-21", end_date: "2026-11-20", mode: "CURRENT_WINDOW", max_days: 365, historical_end_supported: false }, items: [{ id: "overdue_open", label: "Overdue", value: 3, unit: "COUNT", condition: "ATTENTION", freshness: "CURRENT", completeness: "COMPLETE" }], posture_as_of: "2026-11-20T10:00:00Z" });
  oversight.loadOversight.mockResolvedValue({ projection_version: "rev-1", reporting_period: { start_date: "2026-10-21", end_date: "2026-11-20", mode: "CURRENT_WINDOW", max_days: 365, historical_end_supported: false }, interventions: [{ target_id: "issue", title: "Review access exceptions", owner_name: "Ada Okafor", due_at: "2026-11-30T10:00:00Z", state: "OPEN", priority: 4 }] });
  api.loadProgramSummaries.mockResolvedValue({ items: [{ program: { id: "program-1", code: "P-01", name: "Bank assurance", type: "COMPLIANCE", status: "ACTIVE", owning_function: "Risk" }, state_label: "Review due", projection_stale: false, requirement_count: 8, open_matter_count: 2 }], generated_at: "2026-11-20T10:00:00Z" });
  api.loadProgram.mockResolvedValue({ program: { id: "program-1", name: "Bank assurance", code: "P-01", status: "ACTIVE", owning_function: "Risk" }, requirements: [{ id: "req-1", code: "R-01", title: "Review access", statement: "Review privileged access." }], state_label: "Review due" });
  forms.loadFormTemplatePage.mockResolvedValue({ items: [{ template: { id: "form-1", name: "Vendor review", code: "F-01", purpose: "Review service controls.", status: "ACTIVE", version: 2, fields: [{ id: "q-1", label: "Is access reviewed?", type: "yes_no" }] } }] });
  forms.loadFormTemplateItem.mockResolvedValue({ template: { id: "form-1", name: "Vendor review", code: "F-01", purpose: "Review service controls.", status: "ACTIVE", version: 2, fields: [{ id: "q-1", label: "Is access reviewed?", type: "yes_no" }] } });
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "document-1", artifact_id: "artifact", submission_id: "submission", field_id: "field", file_name: "Access review.pdf", form_title: "Access assessment", artifact_status: "AVAILABLE", submitted_at: "2026-11-20T10:00:00Z", current: true }] });
  documents.documentContentURL.mockReturnValue("/api/v1/forms/documents/submission/field/artifact/content?download=true");
  documents.documentEligibility.mockReturnValue("available");
});

it("shows stale program status and unknown counts without claiming current assurance", async () => {
  window.location.hash = "#/programs";
  api.loadProgramSummaries.mockResolvedValue({ items: [{ program: { id: "stale", name: "Access program", code: "P-1", version: 5 }, state_label: "Up to date", overall_state: "CURRENT", program_version: 5, assessed_program_version: 4, projection_version: 2, projection_stale: false, state_generated_at: "2026-11-20T10:00:00Z" }] });
  render(<FreshApp/>);
  const row = (await screen.findByRole("link", { name: "Access program" })).closest("tr")!;
  expect(within(row).getByText("Out of date")).toBeTruthy();
  expect(screen.getByRole("columnheader", { name: "Visible open issues" })).toBeTruthy();
  expect(within(row).getByText("Visible open issues")).toBeTruthy();
  expect(within(row).getAllByText("Unknown")).toHaveLength(2);
  expect(within(row).queryByText("Up to date")).toBeNull();
  expect(within(row).getByText(/Status as of/).querySelector("time")?.getAttribute("datetime")).toBe("2026-11-20T10:00:00Z");
});

it("keeps the actor-visible issue count when the assessment status is unknown", async () => {
  window.location.hash = "#/programs";
  api.loadProgramSummaries.mockResolvedValue({ items: [{ program: { id: "p1", name: "Access program", code: "P-1", version: 2 }, state_label: "Up to date", overall_state: "CURRENT", program_version: 2, requirement_count: 8, open_matter_count: 0 }] });
  render(<FreshApp/>);
  const row = (await screen.findByRole("link", { name: "Access program" })).closest("tr")!;
  expect(within(row).getByText("Unknown")).toBeTruthy();
  expect(within(row).getByText("0")).toBeTruthy();
  expect(within(row).getByText("Visible open issues")).toBeTruthy();
  expect(within(row).queryByText("Up to date")).toBeNull();
});

it("does not present a loaded page as an enterprise Program total", async () => {
  window.location.hash = "#/programs";
  api.loadProgramSummaries.mockResolvedValue({ items: [{ program: { id: "p1", name: "Access program", code: "P-1", version: 2 }, requirement_count: 8, open_matter_count: 0 }], next_cursor: "more" });
  render(<FreshApp/>);
  expect(await screen.findByRole("link", { name: "Access program" })).toBeTruthy();
  expect(screen.getByRole("button", { name: "Load more" })).toBeTruthy();
  expect(screen.getByText("1 loaded · More available")).toBeTruthy();
  expect(screen.queryByText(/total programs/i)).toBeNull();
});

it("applies the assessed status to the server query and retains it on pagination", async () => {
  window.location.hash = "#/programs";
  api.loadProgramSummaries.mockResolvedValue({ items: [{ program: { id: "p1", name: "Access program", code: "P-1", version: 2 }, requirement_count: 8, open_matter_count: 0 }], next_cursor: "more" });
  render(<FreshApp/>);
  await screen.findByRole("link", { name: "Access program" });
  fireEvent.change(screen.getByLabelText("Search programs"), { target: { value: "access" } });
  fireEvent.change(screen.getByLabelText("Assessed status"), { target: { value: "EVIDENCE_INSUFFICIENT" } });
  fireEvent.click(screen.getByRole("button", { name: "Apply filters" }));
  await vi.waitFor(() => expect(api.loadProgramSummaries).toHaveBeenLastCalledWith({ q: "access", overallState: "EVIDENCE_INSUFFICIENT", limit: 20 }));
  expect(window.location.hash).toContain("overall_state=EVIDENCE_INSUFFICIENT");
  expect(window.location.hash).toContain("q=access");
  api.loadProgramSummaries.mockResolvedValueOnce({ items: [] });
  fireEvent.click(await screen.findByRole("button", { name: "Load more" }));
  await vi.waitFor(() => expect(api.loadProgramSummaries).toHaveBeenLastCalledWith({ q: "access", overallState: "EVIDENCE_INSUFFICIENT", limit: 20, cursor: "more" }));
});

it("preserves the applied Program filters when opening and returning from a record", async () => {
  window.location.hash = "#/programs?q=bank&overall_state=AT_RISK";
  render(<FreshApp/>);
  const link = await screen.findByRole("link", { name: "Bank assurance" });
  expect(link.getAttribute("href")).toBe("#/programs/program-1?q=bank&overall_state=AT_RISK");
  fireEvent.click(link);
  expect(await screen.findByRole("heading", { name: "Bank assurance" })).toBeTruthy();
  const back = screen.getByRole("link", { name: "← Programs" });
  expect(back.getAttribute("href")).toBe("#/programs?q=bank&overall_state=AT_RISK");
  fireEvent.click(back);
  expect((await screen.findByLabelText("Assessed status") as HTMLSelectElement).value).toBe("AT_RISK");
  expect((screen.getByLabelText("Search programs") as HTMLInputElement).value).toBe("bank");
});

it("does not append an old page after the assessed status changes", async () => {
  window.location.hash = "#/programs";
  const row = (id: string) => ({ program: { id, name: `${id} program`, code: id }, requirement_count: 0, open_matter_count: 0 });
  let finish!: (value: unknown) => void;
  api.loadProgramSummaries.mockResolvedValueOnce({ items: [row("first")], next_cursor: "more" }).mockReturnValueOnce(new Promise((resolve) => { finish = resolve; })).mockResolvedValueOnce({ items: [row("filtered")] });
  render(<FreshApp/>);
  await screen.findByRole("link", { name: "first program" });
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  fireEvent.change(screen.getByLabelText("Assessed status"), { target: { value: "AT_RISK" } });
  fireEvent.click(screen.getByRole("button", { name: "Apply filters" }));
  await screen.findByRole("link", { name: "filtered program" });
  await act(async () => finish({ items: [row("old")] }));
  expect(screen.queryByRole("link", { name: "old program" })).toBeNull();
  expect(screen.getByText("1 loaded")).toBeTruthy();
});

it("counts each loaded Program once when pages overlap", async () => {
  window.location.hash = "#/programs";
  const row = (id: string) => ({ program: { id, name: `${id} program`, code: id }, requirement_count: 0, open_matter_count: 0 });
  api.loadProgramSummaries.mockResolvedValueOnce({ items: [row("first")], next_cursor: "more" }).mockResolvedValueOnce({ items: [row("first"), row("second")] });
  render(<FreshApp/>);
  await screen.findByRole("link", { name: "first program" });
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  await screen.findByRole("link", { name: "second program" });
  expect(screen.getAllByRole("link", { name: "first program" })).toHaveLength(1);
  expect(screen.getByText("2 loaded")).toBeTruthy();
});

it("opens an exact issue from an assigned work link", async () => {
  window.location.hash = "#/work/issues/issue%2Fone";
  api.loadMatter.mockResolvedValue({ matter: { id: "issue/one", tenant_id: "bank", legal_entity_id: "entity", title: "Review issue", summary: "Owner access needs review.", reference: "ISS-1", status: "DECISION_REQUIRED", version: 1 }, scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "person", record_tenant_id: "bank", record_legal_entity_id: "entity" }, status_label: "Decision needed" });
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Review issue" })).toBeTruthy();
  expect(api.loadMatter).toHaveBeenCalledWith("issue/one");
  expect(screen.getByText("Decision unavailable.")).toBeTruthy();
});

it("rejects unsupported work paths", async () => {
  window.location.hash = "#/work/issues/one/extra";
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Record unavailable" })).toBeTruthy();
  expect(api.loadMatter).not.toHaveBeenCalled();
});

it("requires a fresh review when the form version has changed", async () => {
  window.location.hash = "#/forms/form-1?approval_version=2";
  forms.loadFormTemplateItem.mockResolvedValue({ template: { id: "form-1", name: "Vendor review", status: "PENDING_APPROVAL", version: 3, fields: [] }, authority_available: true, operations: [{ command: "forms.template.transition", can_act: true, allowed_targets: ["ACTIVE", "REJECTED"] }] });
  render(<FreshApp/>);
  expect(await screen.findByText("Form version changed. Review the latest version before deciding.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Approve form" })).toBeNull();
  expect(screen.queryByRole("button", { name: "Reject form" })).toBeNull();
  expect(screen.getByRole("link", { name: "Review latest form" }).getAttribute("href")).toBe("#/forms/form-1");
});

it("does not offer approval when a linked form is no longer pending", async () => {
  window.location.hash = "#/forms/form-1?approval_version=2";
  forms.loadFormTemplateItem.mockResolvedValue({ template: { id: "form-1", name: "Vendor review", status: "ACTIVE", version: 2, fields: [] }, authority_available: true, operations: [{ command: "forms.template.transition", can_act: true, allowed_targets: ["RETIRED"] }] });
  render(<FreshApp/>);
  expect(await screen.findByText("This form is no longer awaiting approval.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Retire form" })).toBeNull();
});

it.each(["0", "-1", "1.5", "abc", "9007199254740992", "2&approval_version=3"])("rejects invalid approval version %s", async (version) => {
  window.location.hash = `#/forms/form-1?approval_version=${version}`;
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Record unavailable" })).toBeTruthy();
  expect(forms.loadFormTemplateItem).not.toHaveBeenCalled();
});

it("does not show a decision when the returned form does not match the link", async () => {
  window.location.hash = "#/forms/form-1?approval_version=2";
  forms.loadFormTemplateItem.mockResolvedValue({ template: { id: "other", name: "Other form", status: "PENDING_APPROVAL", version: 2, fields: [] }, authority_available: true, operations: [{ command: "forms.template.transition", can_act: true, allowed_targets: ["ACTIVE"] }] });
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Form unavailable" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Approve form" })).toBeNull();
});

it("decodes an encoded form ID and preserves its approval version", async () => {
  window.location.hash = "#/forms/form%2Fone?approval_version=2";
  forms.loadFormTemplateItem.mockResolvedValue({ template: { id: "form/one", name: "Vendor review", status: "PENDING_APPROVAL", version: 2, fields: [] }, authority_available: true, operations: [{ command: "forms.template.transition", can_act: true, allowed_targets: ["ACTIVE"] }] });
  render(<FreshApp/>);
  expect(await screen.findByRole("button", { name: "Approve form" })).toBeTruthy();
  expect(forms.loadFormTemplateItem).toHaveBeenCalledWith("form/one");
});

it("opens the same pending version with its authorized decision", async () => {
  window.location.hash = "#/forms/form-1?approval_version=2";
  forms.loadFormTemplateItem.mockResolvedValue({ template: { id: "form-1", name: "Vendor review", status: "PENDING_APPROVAL", version: 2, fields: [] }, authority_available: true, operations: [{ command: "forms.template.transition", can_act: true, allowed_targets: ["ACTIVE", "REJECTED"] }] });
  render(<FreshApp/>);
  expect(await screen.findByRole("button", { name: "Approve form" })).toBeTruthy();
});

it("confirms an approval after the updated form is loaded", async () => {
  window.location.hash = "#/forms/form-1?approval_version=2";
  const pending = { id: "form-1", name: "Vendor review", status: "PENDING_APPROVAL", version: 2, fields: [] };
  const approved = { ...pending, status: "ACTIVE", version: 3 };
  forms.loadFormTemplateItem.mockResolvedValueOnce({ template: pending, authority_available: true, operations: [{ command: "forms.template.transition", can_act: true, allowed_targets: ["ACTIVE", "REJECTED"] }] }).mockResolvedValueOnce({ template: approved });
  forms.transitionFormTemplateRevision.mockResolvedValueOnce(approved);
  render(<FreshApp/>);
  fireEvent.click(await screen.findByRole("button", { name: "Approve form" }));
  expect(await screen.findByText("Form approved.")).toBeTruthy();
  expect(screen.queryByText("Form version changed. Review the latest version before deciding.")).toBeNull();
  expect(screen.queryByRole("button", { name: "Approve form" })).toBeNull();
});

it("retains a verified rejection when refreshing the form fails", async () => {
  window.location.hash = "#/forms/form-1?approval_version=2";
  const pending = { id: "form-1", name: "Vendor review", status: "PENDING_APPROVAL", version: 2, fields: [] };
  const rejected = { ...pending, status: "REJECTED", version: 3 };
  forms.loadFormTemplateItem.mockResolvedValueOnce({ template: pending, authority_available: true, operations: [{ command: "forms.template.transition", can_act: true, allowed_targets: ["ACTIVE", "REJECTED"] }] }).mockRejectedValueOnce(new Error("unavailable"));
  forms.transitionFormTemplateRevision.mockResolvedValueOnce(rejected);
  render(<FreshApp/>);
  fireEvent.click(await screen.findByRole("button", { name: "Reject form" }));
  expect(await screen.findByText("Form rejected.")).toBeTruthy();
  expect(screen.getByRole("heading", { name: "Form unavailable" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Reject form" })).toBeNull();
});

it("does not attribute a later form change to the confirmed decision", async () => {
  window.location.hash = "#/forms/form-1?approval_version=2";
  const pending = { id: "form-1", name: "Vendor review", status: "PENDING_APPROVAL", version: 2, fields: [] };
  const approved = { ...pending, status: "ACTIVE", version: 3 };
  forms.loadFormTemplateItem.mockResolvedValueOnce({ template: pending, authority_available: true, operations: [{ command: "forms.template.transition", can_act: true, allowed_targets: ["ACTIVE"] }] }).mockResolvedValueOnce({ template: { ...approved, status: "RETIRED", version: 4 } });
  forms.transitionFormTemplateRevision.mockResolvedValueOnce(approved);
  render(<FreshApp/>);
  fireEvent.click(await screen.findByRole("button", { name: "Approve form" }));
  expect(await screen.findByText("Form version changed. Review the latest version before deciding.")).toBeTruthy();
  expect(screen.queryByText("Form approved.")).toBeNull();
});

it("renders a fresh shell and real Home data", async () => {
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Overview" })).toBeTruthy();
  expect(screen.getAllByText("Meridian Bank").length).toBeGreaterThan(0);
  expect(await screen.findByText("Review access exceptions")).toBeTruthy();
  expect(screen.getByText("3")).toBeTruthy();
  expect(screen.getByRole("navigation", { name: "Primary navigation" })).toBeTruthy();
});

it("shows unavailable data rather than sample counts", async () => {
  metrics.loadHomeMetrics.mockRejectedValue(new Error("unavailable"));
  oversight.loadOversight.mockRejectedValue(new Error("unavailable"));
  render(<FreshApp/>);
  expect(await screen.findByText("Home data unavailable.")).toBeTruthy();
  expect(screen.queryByText("3")).toBeNull();
});

it("does not show a number when metric coverage is unknown", async () => {
  metrics.loadHomeMetrics.mockResolvedValue({ scope_kind: "LEGAL_ENTITY", scope_id: "entity", source_revision: "rev-1", reporting_period: { start_date: "2026-10-21", end_date: "2026-11-20", mode: "CURRENT_WINDOW", max_days: 365, historical_end_supported: false }, items: [{ id: "unknown", label: "Coverage", value: 0, freshness: "STALE", completeness: "UNKNOWN" }], posture_as_of: "2026-11-20T10:00:00Z" });
  render(<FreshApp/>);
  expect(await screen.findByText("Outdated · Coverage unknown")).toBeTruthy();
  expect(screen.queryByText("0")).toBeNull();
});

it("does not combine Home data from a different scope", async () => {
  metrics.loadHomeMetrics.mockResolvedValue({ scope_kind: "LEGAL_ENTITY", scope_id: "other-entity", reporting_period: { start_date: "2026-10-21", end_date: "2026-11-20", mode: "CURRENT_WINDOW", max_days: 365, historical_end_supported: false }, items: [] });
  render(<FreshApp/>);
  expect(await screen.findByText("Home data unavailable.")).toBeTruthy();
  expect(screen.queryByText("Review access exceptions")).toBeNull();
});

it("closes mobile navigation with Escape", async () => {
  render(<FreshApp/>);
  const menu = await screen.findByRole("button", { name: "Open navigation" });
  fireEvent.click(menu);
  expect(menu.getAttribute("aria-expanded")).toBe("true");
  fireEvent.keyDown(document, { key: "Escape" });
  expect(menu.getAttribute("aria-expanded")).toBe("false");
  expect(document.activeElement).toBe(menu);
});

it("contains keyboard focus in the mobile navigation and restores it on close", async () => {
  render(<FreshApp/>);
  const menu = await screen.findByRole("button", { name: "Open navigation" });
  fireEvent.click(menu);
  const dialog = screen.getByRole("dialog", { name: "Navigation" });
  const close = screen.getByRole("button", { name: "Close navigation" });
  expect(dialog.getAttribute("aria-modal")).toBe("true");
  expect(document.activeElement).toBe(close);
  expect(document.getElementById("v-main")?.hasAttribute("inert")).toBe(true);
  expect(document.querySelector(".v-topbar")?.hasAttribute("inert")).toBe(true);
  expect(screen.getByText("Skip to content").hasAttribute("inert")).toBe(true);
  fireEvent.keyDown(document, { key: "Tab", shiftKey: true });
  expect(document.activeElement?.textContent).toContain("Work & Approvals");
  fireEvent.keyDown(document, { key: "Tab" });
  expect(document.activeElement).toBe(close);
  fireEvent.click(close);
  expect(document.activeElement).toBe(menu);
  expect(document.getElementById("v-main")?.hasAttribute("inert")).toBe(false);
  expect(document.querySelector(".v-topbar")?.hasAttribute("inert")).toBe(false);
  expect(screen.getByText("Skip to content").hasAttribute("inert")).toBe(false);
});

it("focuses the destination after choosing a mobile navigation link", async () => {
  render(<FreshApp/>);
  const menu = await screen.findByRole("button", { name: "Open navigation" });
  fireEvent.click(menu);
  fireEvent.click(screen.getByRole("link", { name: "Programs" }));
  await screen.findByRole("heading", { name: "Compliance programs" });
  await vi.waitFor(() => expect(document.activeElement).toBe(document.getElementById("v-main")));
  expect(menu.getAttribute("aria-expanded")).toBe("false");
});

it("unlocks the page when the navigation closes on desktop resize", async () => {
  render(<FreshApp/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open navigation" }));
  expect(document.body.style.overflow).toBe("hidden");
  fireEvent(window, new Event("resize"));
  await vi.waitFor(() => expect(screen.queryByRole("dialog", { name: "Navigation" })).toBeNull());
  expect(document.body.style.overflow).toBe("");
  expect(document.getElementById("v-main")?.hasAttribute("inert")).toBe(false);
  expect(document.activeElement).toBe(document.getElementById("v-main"));
});

it("removes the navigation focus trap when the session ends", async () => {
  render(<FreshApp/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open navigation" }));
  expect(document.body.style.overflow).toBe("hidden");
  fireEvent(window, new Event("clearsight:session-ended"));
  expect(await screen.findByRole("heading", { name: "Sign in to ClearSight" })).toBeTruthy();
  expect(document.body.style.overflow).toBe("");
  const signIn = screen.getByRole("link", { name: "Sign in" });
  signIn.focus();
  fireEvent.keyDown(document, { key: "Tab", shiftKey: true });
  expect(document.activeElement).toBe(signIn);
});

it("has no detectable accessibility violations in the Home shell", async () => {
  const { container } = render(<FreshApp/>);
  await screen.findByText("Review access exceptions");
  const result = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  expect(result.violations).toEqual([]);
});

it("opens the selected Program using its exact authorized record", async () => {
  render(<FreshApp/>);
  fireEvent.click(await screen.findByRole("link", { name: "Programs" }));
  expect(await screen.findByRole("heading", { name: "Compliance programs" })).toBeTruthy();
  fireEvent.click(await screen.findByRole("link", { name: "Bank assurance" }));
  expect(await screen.findByRole("heading", { name: "Bank assurance" })).toBeTruthy();
  expect(await screen.findByText("R-01 · Review access")).toBeTruthy();
  expect(api.loadProgram).toHaveBeenCalledWith("program-1");
});

it("opens the selected form from the new library", async () => {
  render(<FreshApp/>);
  fireEvent.click(await screen.findByRole("link", { name: "Forms" }));
  expect(await screen.findByRole("heading", { name: "Forms & assessments" })).toBeTruthy();
  fireEvent.click(await screen.findByRole("link", { name: "Vendor review" }));
  expect(await screen.findByRole("heading", { name: "Vendor review" })).toBeTruthy();
  expect(screen.getByText("Is access reviewed?")).toBeTruthy();
  expect(forms.loadFormTemplateItem).toHaveBeenCalledWith("form-1");
});

it("offers API response format tests only for authorized draft questions", async () => {
  forms.loadFormTemplateItem.mockResolvedValue({ template: { id: "form-1", name: "Access review", status: "DRAFT", version: 2, fields: [{ id: "q-1", label: "Is access reviewed?", type: "yes_no" }, { id: "q-2", label: "Comments", type: "long_text" }] }, authority_available: true, operations: [{ command: "forms.template.revise", can_act: true }] });
  window.location.hash = "#/forms/form-1";
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Access review" })).toBeTruthy();
  expect(screen.getAllByRole("button", { name: "Prepare API question" })).toHaveLength(1);
});

it("does not offer API format tests without revise authority", async () => {
  forms.loadFormTemplateItem.mockResolvedValue({ template: { id: "form-1", name: "Access review", status: "DRAFT", version: 2, fields: [{ id: "q-1", label: "Is access reviewed?", type: "yes_no" }] }, authority_available: false, operations: [{ command: "forms.template.revise", can_act: true }] });
  window.location.hash = "#/forms/form-1";
  render(<FreshApp/>);
  expect(await screen.findByRole("heading", { name: "Access review" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Prepare API question" })).toBeNull();
});

it("loads documents from the scoped library and applies the selected view", async () => {
  render(<FreshApp/>);
  fireEvent.click(await screen.findByRole("link", { name: "Documents" }));
  expect(await screen.findByRole("heading", { name: "Documents" })).toBeTruthy();
  expect(await screen.findByText("Access review.pdf")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Needs refresh" }));
  await vi.waitFor(() => expect(documents.loadDocuments).toHaveBeenCalledWith(expect.objectContaining({ view: "NEEDS_REFRESH" }), expect.any(AbortSignal)));
});


it("labels Home as Dashboard for a verified administrative role", async () => {
  api.loadContext.mockResolvedValue({ tenant: { id: "bank", name: "Meridian Bank" }, legal_entity: { id: "entity", name: "Meridian Bank" }, actor: { id: "person", name: "Ada Okafor", role_codes: ["SYSTEM_ADMIN"] }, capabilities: {} });
  render(<FreshApp/>);
  expect(await screen.findByRole("link", { name: "Dashboard" })).toBeTruthy();
  expect(await screen.findByRole("heading", { name: "Dashboard" })).toBeTruthy();
});
