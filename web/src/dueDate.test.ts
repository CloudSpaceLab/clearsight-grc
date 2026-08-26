import { describe, expect, it } from "vitest";
import { selectedDateEndOfLocalDay, storedDeadlineLocalDate } from "./dueDate";

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
