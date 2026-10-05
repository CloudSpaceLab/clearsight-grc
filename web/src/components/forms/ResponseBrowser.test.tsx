import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ResponseBrowser, responseDateError, sortableResponseColumns } from "./ResponseBrowser";

describe("response query controls", () => {
  it("replaces invalid date results with a recoverable error", () => {
    const onChange = vi.fn();
    const query = { completed_from: "2026-10-05T00:00:00.000Z", completed_until: "2026-10-04T23:59:59.999Z", sort: "RAW_ASC" as const, limit: 50 };
    render(<ResponseBrowser query={query} onChange={onChange} count={0} hasMore={false} loading={false} columns={null}><p>Old results</p></ResponseBrowser>);
    expect(screen.getByRole("alert").textContent).toBe(responseDateError(query));
    expect(screen.queryByText("Old results")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Clear response filters" }));
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ completed_from: undefined, completed_until: undefined }));
    expect(onChange.mock.calls[0]?.[0]).not.toHaveProperty("sort");
    expect(onChange.mock.calls[0]?.[0]).not.toHaveProperty("limit");
  });
  it("accepts both boundaries of one day and toggles stored raw-score order", () => {
    expect(responseDateError({ completed_from: "2026-10-05T00:00:00.000Z", completed_until: "2026-10-05T23:59:59.999Z" })).toBeUndefined();
    const onSort = vi.fn();
    const columns = sortableResponseColumns([{ id: "score", header: "Score", render: () => null, accessibleText: () => "" }], "RAW_DESC", onSort);
    expect(columns[0]?.sortDirection).toBe("descending");
    columns[0]?.onSort?.();
    expect(onSort).toHaveBeenCalledWith("RAW_ASC");
  });
});
