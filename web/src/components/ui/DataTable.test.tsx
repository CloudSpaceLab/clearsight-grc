import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Button, DataTable, Notice, type DataColumn } from "./index";

type Row = { id: string; title: string; status: string; recipients: number };
const rows: readonly Row[] = [
  { id: "distribution-1", title: "Vendor annual review", status: "Open", recipients: 3 },
  { id: "distribution-2", title: "PCI-DSS certification", status: "Completed", recipients: 1 },
];
const columns: readonly DataColumn<Row>[] = [
  { id: "title", header: "Distribution", render: (row) => row.title, accessibleText: (row) => row.title },
  { id: "status", header: "Status", kind: "status", render: (row) => row.status, accessibleText: (row) => row.status },
  { id: "recipients", header: "Recipients", kind: "number", render: (row) => row.recipients, accessibleText: (row) => `${row.recipients} recipients` },
];

describe("DataTable", () => {
  it("renders labelled cells and a complete accessible row name", () => {
    render(<DataTable ariaLabel="Sent-form distributions" rows={rows} rowKey={(row) => row.id} rowName={(row) => `${row.title}, ${row.status}, ${row.recipients} recipients`} columns={columns}/>);
    const row = screen.getByRole("row", { name: "Vendor annual review, Open, 3 recipients" });
    expect(row).toBeTruthy();
    expect(Array.from(row.querySelectorAll("td")).map((cell) => cell.getAttribute("data-label"))).toEqual(["Distribution", "Status", "Recipients"]);
  });

  it("marks the selected row without changing its accessible name", () => {
    render(<DataTable ariaLabel="Sent-form distributions" rows={rows} rowKey={(row) => row.id} rowName={(row) => row.title} columns={columns} selectedKey="distribution-2"/>);
    expect(screen.getByRole("row", { name: "PCI-DSS certification" }).getAttribute("aria-selected")).toBe("true");
  });

  it("announces loading and exposes bounded pagination actions", () => {
    const next = vi.fn();
    render(<DataTable ariaLabel="Sent-form distributions" rows={rows} rowKey={(row) => row.id} rowName={(row) => row.title} columns={columns} isLoading pagination={{ label: "Sent-form pages", nextLabel: "Load next page", onNext: next }}/>);
    expect(screen.getByRole("table").getAttribute("aria-busy")).toBe("true");
    expect(screen.getByRole("status", { name: "Loading sent-form distributions" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Load next page" }));
    expect(next).toHaveBeenCalledTimes(1);
  });

  it("allows the owning feature to replace an unavailable table with recovery", () => {
    const retry = vi.fn();
    render(<Notice tone="error"><strong>Sent forms could not be loaded.</strong> The current query could not be completed. <Button onPress={retry}>Try again</Button></Notice>);
    expect(screen.queryByRole("table")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(retry).toHaveBeenCalledTimes(1);
  });
});
