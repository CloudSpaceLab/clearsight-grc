import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
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
    owner_display_name: "Amina Okafor",
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
  expect(within(strip).getByRole("article", { name: /In progress: 20/ })).toBeTruthy();
  expect(within(strip).getByText("6")).toBeTruthy();
  for (const [label, count] of [["Missing lawful basis", 4], ["Missing owner", 3], ["Missing data subjects", 5]] as const) {
    const card = within(strip).getByRole("article", { name: new RegExp(`${label}: ${count}.*Coverage incomplete`) });
    expect(card.getAttribute("data-quality")).toBe("partial");
  }
  expect(within(strip).getByText("42 checked · 2 excluded · 1 unknown")).toBeTruthy();
});

it("keeps undefined coverage values as unknown rather than zero", () => {
  render(<RopaDashboardStrip summary={{ ...summary, coverage: { population: 7 } }}/>);
  const strip = screen.getByRole("region", { name: "Processing activity status and coverage" });
  expect(within(strip).getByText("7 checked · excluded: unknown · unknown records: unknown")).toBeTruthy();
  expect(strip.textContent).not.toContain("excluded: 0");
  expect(strip.textContent).not.toContain("unknown records: 0");
});

it("does not show an empty ROPA population as healthy compliance", () => {
  render(<RopaDashboardStrip summary={{
    ...summary,
    coverage: { population: 0, excluded: 0, unknown: 0 },
    counts: { ...summary.counts, total: 0, open: 0, review_overdue: 0, missing_lawful_basis: 0, missing_owner: 0, no_data_subjects: 0 },
  }}/>);

  const cards = screen.getAllByRole("article");
  expect(cards).toHaveLength(6);
  expect(screen.getByRole("article", { name: /Processing activities: 0.*No assessed population/ })).toBeTruthy();
  expect(screen.getByRole("article", { name: /Missing lawful basis: —.*No assessed population/ })).toBeTruthy();
  expect(cards.every((card) => card.getAttribute("data-quality") === "unknown")).toBe(true);
  expect(screen.queryByText("0 missing lawful basis")).toBeNull();
});

it("labels a stale summary with its generation time and omits that warning for a current summary", () => {
  const { unmount } = render(<RopaDashboardStrip summary={{ ...summary, freshness: "STALE" }}/>);
  expect(screen.getAllByText("Stale").length).toBeGreaterThan(0);
  expect(screen.getByRole("article", { name: /Processing activities: 42.*Stale/ })).toBeTruthy();
  expect(screen.getByText("24 Sept 2026, 08:00")).toBeTruthy();
  unmount();

  render(<RopaDashboardStrip summary={summary}/>);
  expect(screen.queryByText("Stale")).toBeNull();
  expect(screen.getByText("Current")).toBeTruthy();
});

it("renders one register row per activity with working-language status", async () => {
  render(<RopaRegisterPage loadSummary={api.fetchDashboard} loadActivities={api.listProcessingActivities}/>);

  expect(await screen.findByText("Customer onboarding")).toBeTruthy();
  expect(screen.getByRole("heading", { name: "Privacy & data", level: 1 })).toBeTruthy();
  expect(screen.getByRole("tab", { name: "ROPA", selected: true })).toBeTruthy();
  expect(screen.queryByRole("region", { name: "Processing activity status and coverage" })).toBeNull();
  expect(screen.getByText("Card payment notices")).toBeTruthy();
  expect(screen.getByText("Archived payroll import")).toBeTruthy();
  expect(screen.getAllByRole("row")).toHaveLength(4);
  const table = screen.getByRole("table", { name: "Processing activity register" });
  const firstRow = within(table).getByRole("row", { name: /Customer onboarding/ });
  const secondRow = within(table).getByRole("row", { name: /Card payment notices/ });
  const thirdRow = within(table).getByRole("row", { name: /Archived payroll import/ });
  expect(within(firstRow).getByText("In progress")).toBeTruthy();
  expect(within(firstRow).getByText("Amina Okafor")).toBeTruthy();
  expect(within(firstRow).queryByText("owner-1")).toBeNull();
  expect(within(secondRow).getByText("Not started")).toBeTruthy();
  expect(within(thirdRow).getByText("Complete")).toBeTruthy();
  expect(screen.queryByText("OPEN")).toBeNull();
  expect(screen.queryByText("NEW")).toBeNull();
  expect(screen.queryByText("CLOSED")).toBeNull();
});

it("shows the empty population and the next action when no activities match", async () => {
  api.listProcessingActivities.mockResolvedValue({ rows: [], has_more: false });
  render(<RopaRegisterPage loadSummary={api.fetchDashboard} loadActivities={api.listProcessingActivities}/>);

  expect(await screen.findByText("No activities")).toBeTruthy();
  expect(screen.getByText("No current processing activities were found for this legal entity.")).toBeTruthy();
});

it("shows a retry that reloads a failed register read", async () => {
  api.listProcessingActivities.mockRejectedValueOnce(new Error("offline"));
  render(<RopaRegisterPage loadSummary={api.fetchDashboard} loadActivities={api.listProcessingActivities}/>);

  expect(await screen.findByText("Couldn’t load activities.")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Retry" }));
  expect(await screen.findByText("Customer onboarding")).toBeTruthy();
  expect(api.listProcessingActivities).toHaveBeenCalledTimes(2);
});

it("lists every closure blocker without offering an unsupported closure command", async () => {
  render(<RopaActivityPage activityID="activity-1" loadActivity={api.fetchProcessingActivity} loadHistory={api.fetchProcessingActivityHistory}/>);

  expect(await screen.findByRole("heading", { name: "Customer onboarding" })).toBeTruthy();
  const blockerPanel = screen.getByRole("region", { name: "Closure blockers" });
  expect(within(blockerPanel).getByText("Lawful basis")).toBeTruthy();
  expect(within(blockerPanel).getByText("Named owner")).toBeTruthy();
  expect(within(blockerPanel).getByText("Data subject category")).toBeTruthy();
  expect(within(blockerPanel).getByText("Completed review")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Close processing activity" })).toBeNull();
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
  expect(screen.queryByText(/Choose View details/)).toBeNull();
  expect(screen.queryByText(/Double-click/)).toBeNull();
});

it("opens the governed report workspace from the register", async () => {
  const onOpenReports = vi.fn();
  render(<RopaRegisterPage loadSummary={api.fetchDashboard} loadActivities={api.listProcessingActivities} onOpenReports={onOpenReports}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Reports" }));
  expect(onOpenReports).toHaveBeenCalledTimes(1);
});

it("loads only the selected privacy view and can recover the summary independently", async () => {
  api.fetchDashboard.mockRejectedValueOnce(new Error("offline"));
  render(<RopaRegisterPage loadSummary={api.fetchDashboard} loadActivities={api.listProcessingActivities}/>);
  expect(await screen.findByText("Customer onboarding")).toBeTruthy();
  expect(api.fetchDashboard).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("tab", { name: "Overview" }));
  expect(await screen.findByText("Summary unavailable.")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Retry" }));
  await screen.findByRole("region", { name: "Processing activity status and coverage" });
  expect(api.fetchDashboard).toHaveBeenCalledTimes(2);
  fireEvent.click(screen.getByRole("tab", { name: "ROPA" }));
  expect(await screen.findByText("Customer onboarding")).toBeTruthy();
});

it("sends search to the register read and clears the active search", async () => {
  render(<RopaRegisterPage loadSummary={api.fetchDashboard} loadActivities={api.listProcessingActivities}/>);
  await screen.findByText("Customer onboarding");

  const search = screen.getByRole("searchbox", { name: "Search processing activities" });
  fireEvent.change(search, { target: { value: "payments" } });
  await waitFor(() => expect(api.listProcessingActivities).toHaveBeenLastCalledWith(expect.objectContaining({ search: "payments" }), expect.anything()));
  fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
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
  expect(screen.getByText("Couldn’t load activity.")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Retry" }));
  expect(await screen.findByRole("heading", { name: "Customer onboarding" })).toBeTruthy();
});

it("shows the not-found state when the activity is outside the current scope", async () => {
  api.fetchProcessingActivity.mockRejectedValueOnce({ kind: "not_found", message: "The processing activity could not be loaded. Try again." });
  render(<RopaActivityPage activityID="missing-activity" loadActivity={api.fetchProcessingActivity} loadHistory={api.fetchProcessingActivityHistory}/>);

  expect(await screen.findByRole("heading", { name: "Processing activity not found" })).toBeTruthy();
  expect(screen.getByText("Return to the register.")).toBeTruthy();
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
      reviews: [{ id: "review-1", created_at: "2026-01-01T00:00:00Z", due_date: "2026-02-01", completed_at: "2026-01-31T00:00:00Z", outcome: "CONFIRMED", reviewer_principal_id: "reviewer-1", reviewer_display_name: "Chika Nwosu" }],
    },
  });
  render(<RopaActivityPage activityID="activity-1" loadActivity={api.fetchProcessingActivity} loadHistory={api.fetchProcessingActivityHistory}/>);

  expect(await screen.findByText("Sensitive by law")).toBeTruthy();
  expect(screen.getByText(/Country: GB/)).toBeTruthy();
  expect(screen.getByText(/Standard contract clauses/)).toBeTruthy();
  expect(screen.getByText("Customer platform")).toBeTruthy();
  expect(screen.getByText("Confirmed")).toBeTruthy();
  expect(screen.getByText("Review completed 31 Jan 2026")).toBeTruthy();
  expect(screen.getByText("Chika Nwosu")).toBeTruthy();
  expect(screen.queryByText(/reviewer-1/)).toBeNull();
  expect(screen.queryByText("SENSITIVE_BY_LAW")).toBeNull();
  expect(screen.queryByText("STANDARD_CONTRACT_CLAUSES")).toBeNull();
});


it("keeps the current search and page when exact-record navigation opens and closes", async () => {
  api.listProcessingActivities.mockImplementation(async (params) => ({
    rows: params?.cursor ? [rows[1]!] : [rows[0]!],
    has_more: !params?.cursor,
    next_cursor: params?.cursor ? undefined : "page-two",
  }));
  const props = { loadSummary: api.fetchDashboard, loadActivities: api.listProcessingActivities, view: "register" as const };
  const view = render(<RopaRegisterPage {...props}/>);
  await screen.findByText("Customer onboarding");
  fireEvent.change(screen.getByRole("searchbox", { name: "Search processing activities" }), { target: { value: "account" } });
  await waitFor(() => expect(api.listProcessingActivities).toHaveBeenLastCalledWith(expect.objectContaining({ search: "account", cursor: undefined }), expect.anything()));
  fireEvent.click(await screen.findByRole("button", { name: "Load next page" }));
  await screen.findByText("Card payment notices");
  const readCount = api.listProcessingActivities.mock.calls.length;
  view.rerender(<RopaRegisterPage {...props} activityID="activity-1"/>);
  const dialog = await screen.findByRole("dialog", { name: "Processing activity" });
  expect(await within(dialog).findByRole("heading", { name: "Customer onboarding", level: 2 })).toBeTruthy();
  expect(api.fetchProcessingActivity).toHaveBeenCalledWith("activity-1", expect.anything());
  expect(api.listProcessingActivities).toHaveBeenCalledTimes(readCount);
  view.rerender(<RopaRegisterPage {...props}/>);
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(screen.getByRole("searchbox", { name: "Search processing activities" })).toHaveProperty("value", "account");
  expect(screen.getByText("Card payment notices")).toBeTruthy();
  expect(api.listProcessingActivities).toHaveBeenCalledTimes(readCount);
  fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
  await waitFor(() => expect(api.listProcessingActivities).toHaveBeenLastCalledWith(expect.objectContaining({ search: undefined, cursor: undefined }), expect.anything()));
});

it("does not replace a newer search with a late response from an aborted search", async () => {
  let finishOld: ((value: { rows: ProcessingActivity[]; has_more: boolean }) => void) | undefined;
  api.listProcessingActivities.mockImplementation((params) => params?.search === "old"
    ? new Promise((resolve) => { finishOld = resolve; })
    : Promise.resolve({ rows: [rows[1]!], has_more: false }));
  render(<RopaRegisterPage/>);
  await screen.findByText("Card payment notices");
  const search = screen.getByRole("searchbox", { name: "Search processing activities" });
  fireEvent.change(search, { target: { value: "old" } });
  await waitFor(() => expect(finishOld).toBeTypeOf("function"));
  const priorSignal = api.listProcessingActivities.mock.calls.at(-1)?.[1] as AbortSignal;
  fireEvent.change(search, { target: { value: "new" } });
  await screen.findByText("Card payment notices");
  await act(async () => { finishOld?.({ rows: [rows[0]!], has_more: false }); });
  await waitFor(() => expect(priorSignal.aborted).toBe(true));
  expect(screen.queryByText("Customer onboarding")).toBeNull();
});

it("keeps denied access distinct from an empty processing register", async () => {
  api.listProcessingActivities.mockRejectedValueOnce({ kind: "forbidden" });
  render(<RopaRegisterPage/>);
  expect(await screen.findByText("Processing activities are unavailable with your current access. Contact your administrator.")).toBeTruthy();
  expect(screen.queryByText("No activities")).toBeNull();
  expect(screen.queryByRole("table")).toBeNull();
});

it("uses the exact activity read when the selected ID is absent from the current page", async () => {
  api.fetchProcessingActivity.mockRejectedValueOnce({ kind: "not_found" });
  render(<RopaRegisterPage activityID="outside-current-page"/>);
  const dialog = await screen.findByRole("dialog", { name: "Processing activity" });
  expect(await within(dialog).findByRole("heading", { name: "Processing activity not found" })).toBeTruthy();
  expect(api.fetchProcessingActivity).toHaveBeenCalledWith("outside-current-page", expect.anything());
  expect(within(dialog).queryByRole("heading", { name: "Customer onboarding" })).toBeNull();
});

it("does not present absence of recorded blockers as authority to close", async () => {
  api.fetchProcessingActivity.mockResolvedValueOnce({ ...activityResponse, closure_blockers: [] });
  render(<RopaActivityPage activityID="activity-1"/>);
  expect(await screen.findByText("No closure blockers recorded.")).toBeTruthy();
  expect(screen.queryByText("Ready to close.")).toBeNull();
  expect(screen.queryByRole("button", { name: "Close processing activity" })).toBeNull();
});

it("does not let a late activity response replace a different selected record", async () => {
  let finishOld: ((value: ProcessingActivityResponse) => void) | undefined;
  api.fetchProcessingActivity.mockImplementation((id) => id === "activity-1"
    ? new Promise((resolve) => { finishOld = resolve; })
    : Promise.resolve({ ...activityResponse, activity: rows[1]! }));
  const view = render(<RopaRegisterPage activityID="activity-1"/>);
  await waitFor(() => expect(finishOld).toBeTypeOf("function"));
  const priorSignal = api.fetchProcessingActivity.mock.calls.at(-1)?.[1] as AbortSignal;
  view.rerender(<RopaRegisterPage activityID="activity-2"/>);
  const dialog = await screen.findByRole("dialog", { name: "Processing activity" });
  await within(dialog).findByRole("heading", { name: "Card payment notices" });
  await act(async () => { finishOld?.(activityResponse); });
  await waitFor(() => expect(priorSignal.aborted).toBe(true));
  expect(within(dialog).queryByRole("heading", { name: "Customer onboarding" })).toBeNull();
  expect(within(dialog).getByRole("heading", { name: "Card payment notices" })).toBeTruthy();
});

it("keeps counted Overview drills in the summary's population including ended activities", async () => {
  render(<RopaRegisterPage/>);
  await screen.findByText("Customer onboarding");
  fireEvent.click(screen.getByRole("tab", { name: "Overview" }));
  fireEvent.click(await screen.findByRole("button", { name: /In progress: 20.*Open ROPA/ }));
  await waitFor(() => expect(api.listProcessingActivities).toHaveBeenLastCalledWith(expect.objectContaining({ status: "OPEN", include_retired: true, cursor: undefined }), expect.anything()));
  expect(screen.getByText("Includes ended activities")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Remove Activity scope filter" }));
  await waitFor(() => expect(api.listProcessingActivities).toHaveBeenLastCalledWith(expect.objectContaining({ status: "OPEN", include_retired: undefined, cursor: undefined }), expect.anything()));
});

it.each(["empty", "error"])("can return from an %s later page without losing the cursor recovery action", async (outcome) => {
  api.listProcessingActivities.mockResolvedValueOnce({ rows: [rows[0]!], has_more: true, next_cursor: "page-two" });
  if (outcome === "empty") api.listProcessingActivities.mockResolvedValueOnce({ rows: [], has_more: false });
  else api.listProcessingActivities.mockRejectedValueOnce(new Error("unavailable"));
  render(<RopaRegisterPage/>);
  fireEvent.click(await screen.findByRole("button", { name: "Load next page" }));
  if (outcome === "empty") {
    await screen.findByText("No more activities");
    expect(screen.queryByText("No processing activities have been recorded for this legal entity.")).toBeNull();
  }
  else await screen.findByText("Couldn’t load activities.");
  fireEvent.click(screen.getByRole("button", { name: "Load previous page" }));
  await screen.findByText("Customer onboarding");
  expect(api.listProcessingActivities).toHaveBeenLastCalledWith(expect.objectContaining({ cursor: undefined }), expect.anything());
});

it("shows recorded review dates without classifying deadlines in the browser's timezone", async () => {
  api.listProcessingActivities.mockResolvedValueOnce({ rows: [activity({ next_review_date: "2020-01-01" })], has_more: false });
  api.fetchProcessingActivity.mockResolvedValueOnce({ ...activityResponse, activity: activity({ next_review_date: "2020-01-01", reviews: [{ id: "pending", created_at: "2019-12-01T00:00:00Z", due_date: "2020-01-01" }] }) });
  const view = render(<RopaRegisterPage/>);
  expect(await screen.findByRole("cell", { name: "Next review: 01 Jan 2020" })).toBeTruthy();
  expect(screen.queryByText(/Overdue/)).toBeNull();
  view.rerender(<RopaRegisterPage activityID="activity-1"/>);
  const dialog = await screen.findByRole("dialog", { name: "Processing activity" });
  expect(await within(dialog).findByText("Review pending")).toBeTruthy();
  expect(within(dialog).queryByText(/Overdue|Review overdue/)).toBeNull();
});
