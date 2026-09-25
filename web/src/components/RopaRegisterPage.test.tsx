import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { RopaActivityPage } from "./RopaActivityPage";
import { RopaDashboardStrip } from "./RopaDashboardStrip";
import { RopaRegisterPage } from "./RopaRegisterPage";
import type { ProcessingActivity, ProcessingActivityResponse, RegisterSummary } from "../ropaTypes";

const api = vi.hoisted(() => ({
  fetchDashboard: vi.fn(),
  listProcessingActivities: vi.fn(),
  fetchProcessingActivity: vi.fn(),
  fetchProcessingActivityHistory: vi.fn(),
}));

vi.mock("../ropaApi", () => api);

const summary: RegisterSummary = {
  generated_at: "2026-09-24T08:00:00Z",
  projection_version: "ropa-v1",
  freshness: "CURRENT",
  source_high_water: "2026-09-24T07:58:00Z",
  coverage: { population: 42, excluded: 2, unknown: 1 },
  counts: {
    total: 42,
    new: 8,
    open: 20,
    closed: 14,
    review_overdue: 6,
    missing_lawful_basis: 4,
    missing_owner: 3,
    no_data_subjects: 5,
    retired: 0,
  },
};

function activity(overrides: Partial<ProcessingActivity> = {}): ProcessingActivity {
  return {
    id: "activity-1",
    tenant_id: "tenant-1",
    legal_entity_id: "entity-1",
    code: "PA-001",
    name: "Customer onboarding",
    description: "Collects identity information when a customer opens an account.",
    status: "OPEN",
    purpose: "Open and maintain customer accounts",
    lawful_basis: "Contract",
    controller: "Fidelity Bank",
    processor: "Internal operations",
    automated_decision_making: false,
    data_subject_categories: "Customers",
    personal_data_categories: "Name; date of birth",
    security_measures: "Encryption at rest",
    retention_period: "Seven years after account closure",
    start_date: "2024-01-01",
    next_review_date: "2026-10-01",
    owner_principal_id: "owner-1",
    version: 4,
    created_at: "2024-01-01T08:00:00Z",
    updated_at: "2026-09-20T08:00:00Z",
    ...overrides,
  };
}

const rows = [
  activity(),
  activity({ id: "activity-2", code: "PA-002", name: "Card payment notices", status: "NEW", lawful_basis: "", owner_principal_id: undefined, next_review_date: undefined }),
  activity({ id: "activity-3", code: "PA-003", name: "Archived payroll import", status: "CLOSED", next_review_date: "2026-09-01" }),
];

const activityResponse: ProcessingActivityResponse = {
  state_label: "In progress",
  activity: rows[0]!,
  closure_blockers: ["lawful basis", "named owner", "data subject category", "completed review"],
};

beforeEach(() => {
  vi.clearAllMocks();
  api.fetchDashboard.mockResolvedValue(summary);
  api.listProcessingActivities.mockResolvedValue({ rows, has_more: false });
  api.fetchProcessingActivity.mockResolvedValue(activityResponse);
  api.fetchProcessingActivityHistory.mockResolvedValue({ events: [], has_more: false });
});

it("renders the stored register counts and exception signals", () => {
  render(<RopaDashboardStrip summary={summary}/>);

  const strip = screen.getByRole("region", { name: "Processing activity status and coverage" });
  expect(within(strip).getByText("42")).toBeTruthy();
  expect(within(strip).getByText("20")).toBeTruthy();
  expect(within(strip).getByText("6")).toBeTruthy();
  expect(within(strip).getByText("4 missing lawful basis")).toBeTruthy();
  expect(within(strip).getByText("3 missing owner")).toBeTruthy();
  expect(within(strip).getByText("5 missing data subjects")).toBeTruthy();
  expect(within(strip).getByText("42 activities checked · 2 excluded · 1 unknown")).toBeTruthy();
});

it("keeps undefined coverage values as unknown rather than zero", () => {
  render(<RopaDashboardStrip summary={{ ...summary, coverage: { population: 7 } }}/>);
  const strip = screen.getByRole("region", { name: "Processing activity status and coverage" });
  expect(within(strip).getByText("7 activities checked · excluded: unknown · unknown records: unknown")).toBeTruthy();
  expect(strip.textContent).not.toContain("excluded: 0");
  expect(strip.textContent).not.toContain("unknown records: 0");
});

it("labels a stale summary with its generation time and omits that warning for a current summary", () => {
  const { unmount } = render(<RopaDashboardStrip summary={{ ...summary, freshness: "STALE" }}/>);
  expect(screen.getByText("Stale register summary")).toBeTruthy();
  expect(screen.getByText("24 Sept 2026, 08:00")).toBeTruthy();
  unmount();

  render(<RopaDashboardStrip summary={summary}/>);
  expect(screen.queryByText("Stale register summary")).toBeNull();
  expect(screen.getByText("Current register summary")).toBeTruthy();
});

it("renders one register row per activity with working-language status", async () => {
  render(<RopaRegisterPage loadSummary={api.fetchDashboard} loadActivities={api.listProcessingActivities}/>);

  expect(await screen.findByText("Customer onboarding")).toBeTruthy();
  expect(screen.getByText("Card payment notices")).toBeTruthy();
  expect(screen.getByText("Archived payroll import")).toBeTruthy();
  expect(screen.getAllByRole("row")).toHaveLength(4);
  const table = screen.getByRole("table", { name: "Processing activity register" });
  const firstRow = within(table).getByRole("row", { name: /Customer onboarding/ });
  const secondRow = within(table).getByRole("row", { name: /Card payment notices/ });
  const thirdRow = within(table).getByRole("row", { name: /Archived payroll import/ });
  expect(within(firstRow).getByText("In progress")).toBeTruthy();
  expect(within(secondRow).getByText("Not started")).toBeTruthy();
  expect(within(thirdRow).getByText("Complete")).toBeTruthy();
  expect(screen.queryByText("OPEN")).toBeNull();
  expect(screen.queryByText("NEW")).toBeNull();
  expect(screen.queryByText("CLOSED")).toBeNull();
});

it("shows the empty population and the next action when no activities match", async () => {
  api.listProcessingActivities.mockResolvedValue({ rows: [], has_more: false });
  render(<RopaRegisterPage loadSummary={api.fetchDashboard} loadActivities={api.listProcessingActivities}/>);

  expect(await screen.findByText("No processing activities recorded in this legal entity yet")).toBeTruthy();
  expect(screen.getByText(/next valid action/i)).toBeTruthy();
  expect(screen.getByText(/add the first processing activity/i)).toBeTruthy();
});

it("shows a retry that reloads a failed register read", async () => {
  api.listProcessingActivities.mockRejectedValueOnce(new Error("offline"));
  render(<RopaRegisterPage loadSummary={api.fetchDashboard} loadActivities={api.listProcessingActivities}/>);

  expect(await screen.findByText(/processing activity register could not be loaded/i)).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: /retry register/i }));
  expect(await screen.findByText("Customer onboarding")).toBeTruthy();
  expect(api.listProcessingActivities).toHaveBeenCalledTimes(2);
});

it("lists every closure blocker and explains why closing is unavailable", async () => {
  render(<RopaActivityPage activityID="activity-1" loadActivity={api.fetchProcessingActivity} loadHistory={api.fetchProcessingActivityHistory}/>);

  expect(await screen.findByRole("heading", { name: "Customer onboarding" })).toBeTruthy();
  const blockerPanel = screen.getByRole("region", { name: "Complete these facts before closing" });
  expect(within(blockerPanel).getByText("Lawful basis")).toBeTruthy();
  expect(within(blockerPanel).getByText("Named owner")).toBeTruthy();
  expect(within(blockerPanel).getByText("Data subject category")).toBeTruthy();
  expect(within(blockerPanel).getByText("Completed review")).toBeTruthy();
  expect(within(blockerPanel).getByText(/cannot be closed/i)).toBeTruthy();
  expect(screen.getByRole("button", { name: "Close processing activity" })).toHaveProperty("disabled", true);
});

it("opens the selected activity from the register row action", async () => {
  const onOpenActivity = vi.fn();
  render(<RopaRegisterPage loadSummary={api.fetchDashboard} loadActivities={api.listProcessingActivities} onOpenActivity={onOpenActivity}/>);
  const row = await screen.findByRole("row", { name: /Customer onboarding/ });
  fireEvent.doubleClick(row);
  expect(onOpenActivity).toHaveBeenCalledWith("activity-1");
});

it("shows a visible control that opens each activity's details", async () => {
  // A row action reachable only by double-click or a keyboard shortcut cannot be
  // discovered, and double-tap is unreliable on a touch screen, so every row
  // must carry a control that names the record it opens.
  const onOpenActivity = vi.fn();
  render(<RopaRegisterPage loadSummary={api.fetchDashboard} loadActivities={api.listProcessingActivities} onOpenActivity={onOpenActivity}/>);
  const control = await screen.findByRole("button", { name: /View details for Customer onboarding/ });
  fireEvent.click(control);
  expect(onOpenActivity).toHaveBeenCalledWith("activity-1");
});

it("does not describe the interaction instead of naming the action", async () => {
  render(<RopaRegisterPage loadSummary={api.fetchDashboard} loadActivities={api.listProcessingActivities} onOpenActivity={vi.fn()}/>);
  await screen.findByRole("row", { name: /Customer onboarding/ });
  expect(screen.getByText(/Choose View details/)).toBeTruthy();
  expect(screen.queryByText(/Double-click/)).toBeNull();
});

it("opens the governed report workspace from the register", async () => {
  const onOpenReports = vi.fn();
  render(<RopaRegisterPage loadSummary={api.fetchDashboard} loadActivities={api.listProcessingActivities} onOpenReports={onOpenReports}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open reports" }));
  expect(onOpenReports).toHaveBeenCalledTimes(1);
});

it("retries a summary failure without hiding the register read", async () => {
  api.fetchDashboard.mockRejectedValueOnce(new Error("offline"));
  render(<RopaRegisterPage loadSummary={api.fetchDashboard} loadActivities={api.listProcessingActivities}/>);

  expect(await screen.findByText(/register summary could not be loaded/i)).toBeTruthy();
  expect(screen.getByText("Customer onboarding")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: /retry summary/i }));
  await waitFor(() => expect(api.fetchDashboard).toHaveBeenCalledTimes(2));
});

it("sends search to the register read and clears the active search", async () => {
  render(<RopaRegisterPage loadSummary={api.fetchDashboard} loadActivities={api.listProcessingActivities}/>);
  await screen.findByText("Customer onboarding");

  const search = screen.getByRole("searchbox", { name: "Search processing activities" });
  fireEvent.change(search, { target: { value: "payments" } });
  await waitFor(() => expect(api.listProcessingActivities).toHaveBeenLastCalledWith(expect.objectContaining({ search: "payments" }), expect.anything()));
  fireEvent.click(screen.getByRole("button", { name: "Clear register filters" }));
  await waitFor(() => expect(api.listProcessingActivities).toHaveBeenLastCalledWith(expect.objectContaining({ search: undefined }), expect.anything()));
});

it("loads the next register page with the opaque cursor and can return to the first page", async () => {
  api.listProcessingActivities
    .mockResolvedValueOnce({ rows: [rows[0]!], has_more: true, next_cursor: "cursor-2" })
    .mockResolvedValueOnce({ rows: [rows[1]!], has_more: false });
  render(<RopaRegisterPage loadSummary={api.fetchDashboard} loadActivities={api.listProcessingActivities}/>);
  await screen.findByText("Customer onboarding");

  fireEvent.click(screen.getByRole("button", { name: "Load next page" }));
  expect(await screen.findByText("Card payment notices")).toBeTruthy();
  expect(api.listProcessingActivities).toHaveBeenLastCalledWith(expect.objectContaining({ cursor: "cursor-2" }), expect.anything());
  fireEvent.click(screen.getByRole("button", { name: "Load previous page" }));
  expect(await screen.findByText("Customer onboarding")).toBeTruthy();
  expect(api.listProcessingActivities).toHaveBeenLastCalledWith(expect.objectContaining({ cursor: undefined }), expect.anything());
});

it("shows a recoverable error when the activity read fails", async () => {
  api.fetchProcessingActivity.mockRejectedValueOnce(new Error("offline"));
  render(<RopaActivityPage activityID="activity-1" loadActivity={api.fetchProcessingActivity} loadHistory={api.fetchProcessingActivityHistory}/>);

  expect(await screen.findByRole("heading", { name: "Processing activity unavailable" })).toBeTruthy();
  expect(screen.getByText(/could not be loaded/i)).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Retry activity" }));
  expect(await screen.findByRole("heading", { name: "Customer onboarding" })).toBeTruthy();
});

it("shows the not-found state when the activity is outside the current scope", async () => {
  api.fetchProcessingActivity.mockRejectedValueOnce({ kind: "not_found", message: "The processing activity could not be loaded. Try again." });
  render(<RopaActivityPage activityID="missing-activity" loadActivity={api.fetchProcessingActivity} loadHistory={api.fetchProcessingActivityHistory}/>);

  expect(await screen.findByRole("heading", { name: "Processing activity not found" })).toBeTruthy();
  expect(screen.getByText(/not available in the current legal entity/i)).toBeTruthy();
});

it("shows stored category sensitivity, cross-border safeguards, systems and review outcomes", async () => {
  api.fetchProcessingActivity.mockResolvedValueOnce({
    state_label: "In progress",
    closure_blockers: [],
    activity: {
      ...rows[0]!,
      data_categories: [{ category: "Biometric data", sensitivity: "SENSITIVE_BY_LAW" }],
      recipients: [{ recipient: "Cloud processor", recipient_kind: "EXTERNAL", country_code: "GB", is_cross_border: true, transfer_basis: "STANDARD_CONTRACT_CLAUSES" }],
      systems: [{ system_name: "Customer platform", system_kind: "APPLICATION" }],
      reviews: [{ id: "review-1", created_at: "2026-01-01T00:00:00Z", due_date: "2026-02-01", completed_at: "2026-01-31T00:00:00Z", outcome: "CONFIRMED", reviewer_principal_id: "reviewer-1" }],
    },
  });
  render(<RopaActivityPage activityID="activity-1" loadActivity={api.fetchProcessingActivity} loadHistory={api.fetchProcessingActivityHistory}/>);

  expect(await screen.findByText("Sensitive by law")).toBeTruthy();
  expect(screen.getByText(/Country: GB/)).toBeTruthy();
  expect(screen.getByText(/Standard contract clauses/)).toBeTruthy();
  expect(screen.getByText("Customer platform")).toBeTruthy();
  expect(screen.getByText("Confirmed")).toBeTruthy();
  expect(screen.getByText("Review completed 31 Jan 2026")).toBeTruthy();
  expect(screen.queryByText("SENSITIVE_BY_LAW")).toBeNull();
  expect(screen.queryByText("STANDARD_CONTRACT_CLAUSES")).toBeNull();
});
