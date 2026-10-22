import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { ApiError } from "../../http";
import type { SystemActivityEvent, SystemActivityPage } from "../../operationsTypes";
import type { loadSystemActivity } from "../../systemActivityApi";
import { NotificationActivity } from "./NotificationActivity";

const now = Date.parse("2026-10-21T12:00:00Z");
const props = { canRead: true, legalEntityID: "bank-ng", legalEntityName: "Clear Bank Nigeria" };

beforeEach(() => { vi.spyOn(Date, "now").mockReturnValue(now); });

function event(id: string, values: Partial<SystemActivityEvent> = {}): SystemActivityEvent {
  return {
    event_id: id,
    occurred_at: "2026-10-21T10:15:00Z",
    category: "GRC_WORK",
    event_type: "PROGRAM_REVIEWED",
    action: `Program review ${id}`,
    outcome: "SUCCEEDED",
    actor_kind: "INTERNAL_USER",
    actor_id: "reviewer-1",
    actor_display_name: "Ada Okafor",
    legal_entity_id: "bank-ng",
    object_type: "PROGRAM",
    object_id: "program-42",
    source: "OUTBOX_EVENT",
    ...values,
  };
}

function page(items: SystemActivityEvent[], next_cursor?: string): SystemActivityPage {
  return { items, next_cursor, as_of: "2026-10-21T12:00:00Z" };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej; });
  return { promise, resolve, reject };
}

async function selectPeriod(label: string) {
  fireEvent.click(screen.getByLabelText("Activity period"));
  fireEvent.click(await screen.findByRole("option", { name: label }));
}

it.each([
  { canRead: false, legalEntityID: "bank-ng", heading: "Activity access is restricted" },
  { canRead: true, legalEntityID: undefined, heading: "Choose a legal entity" },
  { canRead: true, legalEntityID: "*", heading: "Choose a legal entity" },
])("makes no history request when access or an individual entity is missing: $heading", async ({ canRead, legalEntityID, heading }) => {
  const load = vi.fn<typeof loadSystemActivity>().mockResolvedValue(page([event("hidden")]));
  render(<NotificationActivity {...props} canRead={canRead} legalEntityID={legalEntityID} load={load}/>);

  expect(screen.getByRole("heading", { name: heading })).toBeTruthy();
  await act(async () => { await Promise.resolve(); });
  expect(load).not.toHaveBeenCalled();
  expect(screen.queryByLabelText("Activity period")).toBeNull();
  expect(screen.queryByText("Program review hidden")).toBeNull();
});

it("queries one entity and renders dated actor, record and outcome facts without inventing versions", async () => {
  const load = vi.fn<typeof loadSystemActivity>().mockResolvedValue(page([
    event("new", { outcome: "DENIED", action: "Program approval denied" }),
    event("older", { occurred_at: "2026-10-20T08:00:00Z", actor_display_name: undefined, actor_id: undefined, actor_kind: "UNKNOWN", outcome: "PENDING", action: "Review requested" }),
  ]));
  const { container } = render(<NotificationActivity {...props} load={load}/>);

  await screen.findByRole("heading", { name: "Program approval denied" });
  expect(load).toHaveBeenCalledExactlyOnceWith({
    legalEntityID: "bank-ng", limit: 25,
    from: "2026-09-21T12:00:00.000Z", to: "2026-10-21T12:00:00.000Z",
  });
  expect(screen.getByText("2 events loaded")).toBeTruthy();
  expect(screen.getByRole("heading", { name: "21 October 2026" })).toBeTruthy();
  expect(screen.getByRole("heading", { name: "20 October 2026" })).toBeTruthy();
  const recent = within(screen.getByRole("list", { name: "Activity on 21 October 2026" }));
  expect(recent.getByText("Ada Okafor")).toBeTruthy();
  expect(recent.getByText("Denied")).toBeTruthy();
  expect(recent.getByText("Program · program-42")).toBeTruthy();
  expect(recent.getByText("Event new · Outbox event")).toBeTruthy();
  expect(recent.getByText("10:15 UTC").getAttribute("datetime")).toBe("2026-10-21T10:15:00Z");
  expect(screen.getByText("Actor not recorded")).toBeTruthy();
  expect(screen.getByText("Pending")).toBeTruthy();
  const context = within(screen.getByRole("complementary", { name: "Activity context" }));
  expect(context.getByText("Selected scope")).toBeTruthy();
  expect(context.getByText("Clear Bank Nigeria")).toBeTruthy();
  expect(context.getByText(/Record versions are not recorded in this history/)).toBeTruthy();
  const records = container.querySelectorAll(".notification-activity__record");
  expect(records).toHaveLength(2);
  for (const record of records) {
    const row = within(record as HTMLElement);
    expect(row.getByText("Program · program-42")).toBeTruthy();
    expect(row.queryByText("Clear Bank Nigeria")).toBeNull();
    expect(row.queryByText("Scope")).toBeNull();
    expect(row.queryByText("Version")).toBeNull();
    expect(row.queryByText("Not recorded")).toBeNull();
  }
  expect(screen.queryByRole("button", { name: /export/i })).toBeNull();
});

it("sends each period to the server and ignores a response from the former period", async () => {
  const initial = deferred<SystemActivityPage>();
  const load = vi.fn<typeof loadSystemActivity>()
    .mockReturnValueOnce(initial.promise)
    .mockResolvedValueOnce(page([event("week")]))
    .mockResolvedValueOnce(page([event("quarter")]));
  render(<NotificationActivity {...props} load={load}/>);

  await selectPeriod("Last 7 days");
  await screen.findByText("Program review week");
  expect(load).toHaveBeenNthCalledWith(2, {
    legalEntityID: "bank-ng", limit: 25,
    from: "2026-10-14T12:00:00.000Z", to: "2026-10-21T12:00:00.000Z",
  });
  await act(async () => { initial.resolve(page([event("stale month")])); });
  expect(screen.queryByText("Program review stale month")).toBeNull();
  expect(screen.getByText("Program review week")).toBeTruthy();

  await selectPeriod("Last 90 days");
  await screen.findByText("Program review quarter");
  expect(load).toHaveBeenLastCalledWith({
    legalEntityID: "bank-ng", limit: 25,
    from: "2026-07-23T12:00:00.000Z", to: "2026-10-21T12:00:00.000Z",
  });
  expect(screen.queryByText("Program review week")).toBeNull();
});

it("keeps the fixed date window and opaque cursor through a paging failure and retry", async () => {
  const older = deferred<SystemActivityPage>();
  const load = vi.fn<typeof loadSystemActivity>()
    .mockResolvedValueOnce(page([event("first")], "opaque:cursor/+="))
    .mockReturnValueOnce(older.promise)
    .mockResolvedValueOnce(page([event("first"), event("second", { occurred_at: "2026-10-20T08:00:00Z" })]));
  render(<NotificationActivity {...props} load={load}/>);
  await screen.findByText("Program review first");
  const firstQuery = load.mock.calls[0]?.[0];
  if (!firstQuery) throw new Error("Expected the initial scoped activity query");
  vi.mocked(Date.now).mockReturnValue(now + 60 * 60 * 1000);

  const loadOlder = screen.getByRole("button", { name: "Load older activity" });
  fireEvent.click(loadOlder);
  fireEvent.click(loadOlder);
  expect(load).toHaveBeenCalledTimes(2);
  expect(load).toHaveBeenLastCalledWith({ ...firstQuery, cursor: "opaque:cursor/+=" });
  await act(async () => { older.reject(new ApiError(503, "Unavailable")); });
  expect(screen.getByText("Program review first")).toBeTruthy();
  expect(screen.getByRole("alert").textContent).toContain("events already loaded are still shown");

  fireEvent.click(screen.getByRole("button", { name: "Load older activity" }));
  await screen.findByText("Program review second");
  expect(load).toHaveBeenCalledTimes(3);
  expect(load).toHaveBeenLastCalledWith({ ...firstQuery, cursor: "opaque:cursor/+=" });
  expect(screen.getAllByText("Program review first")).toHaveLength(1);
  expect(screen.getByText("2 events loaded")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Load older activity" })).toBeNull();
  expect(screen.queryByRole("alert")).toBeNull();
});

it("immediately clears the old entity and discards its pending older page after a scope change", async () => {
  const older = deferred<SystemActivityPage>();
  const nextEntity = deferred<SystemActivityPage>();
  const load = vi.fn<typeof loadSystemActivity>()
    .mockResolvedValueOnce(page([event("Nigeria first")], "ng-cursor"))
    .mockReturnValueOnce(older.promise)
    .mockReturnValueOnce(nextEntity.promise);
  const { rerender } = render(<NotificationActivity {...props} load={load}/>);
  await screen.findByText("Program review Nigeria first");
  fireEvent.click(screen.getByRole("button", { name: "Load older activity" }));

  rerender(<NotificationActivity {...props} legalEntityID="bank-gh" legalEntityName="Clear Bank Ghana" load={load}/>);
  expect(screen.queryByText("Program review Nigeria first")).toBeNull();
  expect(load).toHaveBeenLastCalledWith(expect.objectContaining({ legalEntityID: "bank-gh", limit: 25 }));
  expect(load.mock.calls[2]?.[0]?.cursor).toBeUndefined();
  await act(async () => { older.resolve(page([event("Nigeria stale")])); });
  expect(screen.queryByText("Program review Nigeria stale")).toBeNull();
  await act(async () => { nextEntity.resolve(page([event("Ghana", { legal_entity_id: "bank-gh" })])); });
  expect(screen.getByText("Program review Ghana")).toBeTruthy();
  expect(screen.queryByText("Program review Nigeria first")).toBeNull();
});

it("removes loaded history and ignores a pending request as soon as read permission is revoked", async () => {
  const older = deferred<SystemActivityPage>();
  const load = vi.fn<typeof loadSystemActivity>()
    .mockResolvedValueOnce(page([event("visible")], "next"))
    .mockReturnValueOnce(older.promise);
  const { rerender } = render(<NotificationActivity {...props} load={load}/>);
  await screen.findByText("Program review visible");
  fireEvent.click(screen.getByRole("button", { name: "Load older activity" }));

  rerender(<NotificationActivity {...props} canRead={false} load={load}/>);
  expect(screen.getByRole("heading", { name: "Activity access is restricted" })).toBeTruthy();
  expect(screen.queryByText("Program review visible")).toBeNull();
  await act(async () => { older.resolve(page([event("must remain hidden")])); });
  expect(screen.queryByText("Program review must remain hidden")).toBeNull();
  expect(load).toHaveBeenCalledTimes(2);
});

it("clears previously loaded history when the server denies a later page", async () => {
  const load = vi.fn<typeof loadSystemActivity>()
    .mockResolvedValueOnce(page([event("previous")], "next"))
    .mockRejectedValueOnce(new ApiError(403, "Access revoked"));
  render(<NotificationActivity {...props} load={load}/>);
  await screen.findByText("Program review previous");
  fireEvent.click(screen.getByRole("button", { name: "Load older activity" }));

  await screen.findByRole("heading", { name: "Activity access is restricted" });
  expect(screen.queryByText("Program review previous")).toBeNull();
  expect(screen.queryByRole("button", { name: "Load older activity" })).toBeNull();
});

it.each(["bank-gh", undefined])("rejects the whole page if a returned event has the wrong or absent entity: %s", async (legal_entity_id) => {
  const load = vi.fn<typeof loadSystemActivity>().mockResolvedValue(page([
    event("in scope"), event("unsafe", { legal_entity_id }),
  ]));
  render(<NotificationActivity {...props} load={load}/>);

  await screen.findByRole("heading", { name: "Activity could not be loaded" });
  expect(screen.getByText(/could not be confirmed for the selected legal entity/)).toBeTruthy();
  expect(screen.queryByText("Program review in scope")).toBeNull();
  expect(screen.queryByText("Program review unsafe")).toBeNull();
  expect(screen.queryByText("0 events loaded")).toBeNull();
});

it("keeps an unavailable history distinct from a successful empty period and supports retry", async () => {
  const load = vi.fn<typeof loadSystemActivity>()
    .mockRejectedValueOnce(new ApiError(503, "Unavailable"))
    .mockResolvedValueOnce(page([]));
  render(<NotificationActivity {...props} load={load}/>);

  await screen.findByRole("heading", { name: "Activity could not be loaded" });
  expect(screen.queryByText("0 events loaded")).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Retry activity" }));
  await screen.findByRole("heading", { name: "No activity in this period" });
  expect(screen.getByText("0 events loaded")).toBeTruthy();
  expect(load).toHaveBeenCalledTimes(2);
  expect(load.mock.calls.every(([query]) => query?.legalEntityID === "bank-ng" && query.limit === 25)).toBe(true);
});

it("refreshes the current period without accepting an older in-flight response", async () => {
  const initial = deferred<SystemActivityPage>();
  const load = vi.fn<typeof loadSystemActivity>()
    .mockReturnValueOnce(initial.promise)
    .mockResolvedValueOnce(page([event("fresh")]));
  render(<NotificationActivity {...props} load={load}/>);
  vi.mocked(Date.now).mockReturnValue(now + 60_000);
  fireEvent.click(screen.getByRole("button", { name: "Refresh activity" }));

  await screen.findByText("Program review fresh");
  expect(load).toHaveBeenLastCalledWith(expect.objectContaining({ to: "2026-10-21T12:01:00.000Z" }));
  await act(async () => { initial.resolve(page([event("outdated")])); });
  await waitFor(() => expect(screen.queryByText("Program review outdated")).toBeNull());
  expect(screen.getByText("Program review fresh")).toBeTruthy();
});
