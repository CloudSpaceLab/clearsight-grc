import { describe, expect, it } from "vitest";
import { actionDeadlinePresentation, selectedDateEndOfLocalDay, storedDeadlineLocalDate } from "./dueDate";

describe("selectedDateEndOfLocalDay", () => {
  it("keeps the selected local calendar day through its final millisecond", () => {
    const result = selectedDateEndOfLocalDay("2026-09-30");

    expect(result).toBe(new Date(2026, 8, 30, 23, 59, 59, 999).toISOString());
    const local = new Date(result!);
    expect([
      local.getFullYear(),
      local.getMonth(),
      local.getDate(),
      local.getHours(),
      local.getMinutes(),
      local.getSeconds(),
      local.getMilliseconds(),
    ]).toEqual([2026, 8, 30, 23, 59, 59, 999]);
  });

  it.each(["", " ", "2026-02-30", "30-09-2026", "not-a-date"])("returns undefined for blank or invalid input %j", (value) => {
    expect(selectedDateEndOfLocalDay(value)).toBeUndefined();
  });

  it("restores the selected local calendar day from its stored instant", () => {
    const stored = new Date(2026, 8, 30, 23, 59, 59, 999).toISOString();

    expect(storedDeadlineLocalDate(stored)).toBe("2026-09-30");
    expect(storedDeadlineLocalDate("not-a-date")).toBe("");
    expect(storedDeadlineLocalDate()).toBe("");
  });
});

describe("actionDeadlinePresentation", () => {
  const now = new Date(2026, 8, 3, 12, 0, 0);

  it("names a passed open deadline and preserves the original date", () => {
    const due = new Date(2026, 7, 22, 17, 0, 0).toISOString();

    expect(actionDeadlinePresentation(due, false, now)).toEqual({
      dateTime: due,
      label: "Due 22 Aug 2026 · 12 days overdue",
      overdue: true,
    });
  });

  it("keeps a future open deadline scheduled", () => {
    const due = new Date(2026, 8, 7, 17, 0, 0).toISOString();

    expect(actionDeadlinePresentation(due, false, now)).toEqual({
      dateTime: due,
      label: "Due 7 Sep 2026",
      overdue: false,
    });
  });

  it("distinguishes a passed time today from a later deadline today", () => {
    const earlier = new Date(2026, 8, 3, 9, 0, 0).toISOString();
    const later = new Date(2026, 8, 3, 17, 0, 0).toISOString();

    expect(actionDeadlinePresentation(earlier, false, now).label).toBe("Due 3 Sep 2026 · overdue today");
    expect(actionDeadlinePresentation(later, false, now).label).toBe("Due 3 Sep 2026");
  });

  it("does not relabel terminal work as overdue", () => {
    const due = new Date(2026, 7, 22, 17, 0, 0).toISOString();

    expect(actionDeadlinePresentation(due, true, now).overdue).toBe(false);
    expect(actionDeadlinePresentation(due, true, now).label).toBe("Due 22 Aug 2026");
  });

  it.each([undefined, "", "not-a-date"])("keeps an invalid or missing deadline explicit", (due) => {
    expect(actionDeadlinePresentation(due, false, now)).toEqual({ label: "No action deadline", overdue: false });
  });
});
