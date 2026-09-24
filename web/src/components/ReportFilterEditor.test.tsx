import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ReportFilterEditor, validateReportFilter } from "./ReportFilterEditor";
import type { ReportFilterExpression, ReportFilterFieldDefinition } from "../reportingTypes";

const fields: ReportFilterFieldDefinition[] = [
  { field: "status", label: "Processing activity status", dataset: "PROCESSING_ACTIVITIES", operators: ["is"], indexed: true },
  { field: "name", label: "Activity name contains", dataset: "PROCESSING_ACTIVITIES", operators: ["contains"], indexed: false },
  { field: "status", label: "Program status", dataset: "PROGRAMS", operators: ["is"], indexed: true },
  { field: "overall_state", label: "Calculated Program state", dataset: "PROGRAMS", operators: ["is"], indexed: true },
  { field: "status", label: "Issue or change status", dataset: "MATTER_EXCEPTIONS", operators: ["is"], indexed: true },
];

function condition(field = "status", value = "OPEN"): ReportFilterExpression {
  return { kind: "group", operator: "and", children: [{ kind: "condition", field, operator: "is", value }] };
}

describe("ReportFilterEditor", () => {
  it("offers only the fields published for the selected dataset", () => {
    render(<ReportFilterEditor fields={fields} dataset="PROCESSING_ACTIVITIES" value={condition()} onChange={vi.fn()} />);

    expect(screen.getAllByText("Processing activity status").length).toBeGreaterThan(0);
    expect(screen.queryByText("Calculated Program state")).toBeNull();
    expect(screen.queryByText("Issue or change status")).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: /Change filter field/ }));
    expect(screen.getByRole("option", { name: /Processing activity status/ })).toBeTruthy();
    expect(screen.getByRole("option", { name: /Activity name contains/ })).toBeTruthy();
    expect(screen.queryByRole("option", { name: /Calculated Program state/ })).toBeNull();
  });

  it("explains that a non-indexed field is slower before an operator uses it", () => {
    render(<ReportFilterEditor fields={fields} dataset="PROCESSING_ACTIVITIES" value={condition("name", "customer")} onChange={vi.fn()} />);
    expect(screen.getByText(/not indexed/i)).toBeTruthy();
    expect(screen.getByText(/may take longer to run/i)).toBeTruthy();
  });

  it("refuses a field from another dataset and names the datasets it belongs to", () => {
    const onChange = vi.fn();
    render(<ReportFilterEditor fields={fields} dataset="PROCESSING_ACTIVITIES" value={condition("overall_state", "AT_RISK")} onChange={onChange} />);

    expect(screen.getByRole("alert").textContent).toMatch(/Calculated Program state/);
    expect(screen.getByRole("alert").textContent).toMatch(/Programs/);
    expect(screen.getByRole("alert").textContent).toMatch(/not available for this report dataset/i);
    expect(onChange).not.toHaveBeenCalled();
  });

  it("rejects an empty group with the condition the operator must add", () => {
    render(<ReportFilterEditor fields={fields} dataset="PROCESSING_ACTIVITIES" value={{ kind: "group", operator: "and", children: [] }} onChange={vi.fn()} />);

    expect(screen.getByRole("alert").textContent).toMatch(/add at least one condition/i);
    expect(screen.getByRole("button", { name: "Save filter" })).toHaveProperty("disabled", true);
  });

  it("enforces the published field and node limits through the exported validation", () => {
    expect(validateReportFilter({ kind: "condition", field: "secret_column", operator: "is", value: "x" }, fields, "PROCESSING_ACTIVITIES")).toMatch(/not published|not available/i);
    const deep: ReportFilterExpression = { kind: "group", operator: "and", children: [{ kind: "group", operator: "and", children: [{ kind: "group", operator: "and", children: [{ kind: "condition", field: "status", operator: "is", value: "OPEN" }] }] }] };
    expect(validateReportFilter(deep, fields, "PROCESSING_ACTIVITIES")).toMatch(/levels|nodes/i);
  });
});
