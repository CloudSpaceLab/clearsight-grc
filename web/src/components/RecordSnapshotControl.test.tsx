import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { RecordSnapshotControl } from "./RecordSnapshotControl";

it("loads and clearly labels an exact point-in-time record without claiming chronology", async () => {
  const load = vi.fn().mockResolvedValue({ version: 3, status: "IN_REVIEW", updatedAt: "2026-08-20T09:00:00Z" });
  render(<RecordSnapshotControl recordLabel="Program" loadSnapshot={load}/>);
  fireEvent.click(screen.getByText("View an earlier Program record"));
  expect(screen.getByLabelText("Date and time").getAttribute("type")).toBe("datetime-local");
  fireEvent.change(screen.getByLabelText("Date and time"), { target: { value: "2026-08-20T12:30" } });
  fireEvent.click(screen.getByRole("button", { name: "View earlier record" }));
  expect(await screen.findByLabelText("Program record at selected time")).toBeTruthy();
  expect(screen.getByText("In Review")).toBeTruthy();
  expect(load).toHaveBeenCalledWith(expect.stringMatching(/^2026-08-20T/));
  expect(screen.getByText(/not a list of changes/)).toBeTruthy();
});
