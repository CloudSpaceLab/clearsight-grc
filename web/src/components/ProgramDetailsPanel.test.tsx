import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { decideProgramCalculationPolicy, decideProgramFrameworkAdoption, loadFrameworkEditions, proposeProgramCalculationPolicy, proposeProgramFrameworkAdoption } from "../programOperationsApi";
import type { ProgramAggregate } from "../types";
import { ProgramDetailsPanel } from "./ProgramDetailsPanel";

vi.mock("../programOperationsApi", () => ({
  proposeProgramCalculationPolicy: vi.fn(),
  decideProgramCalculationPolicy: vi.fn(),
  loadFrameworkEditions: vi.fn(),
  proposeProgramFrameworkAdoption: vi.fn(),
  decideProgramFrameworkAdoption: vi.fn(),
}));

const program = {
  id: "program-1", tenant_id: "bank", code: "COMPLIANCE", name: "Compliance",
  type: "ASSURANCE", status: "ACTIVE", owning_function: "Risk",
  owner_principal_id: "owner", authority_principal_id: "checker",
  scope: {}, effective_from: "2026-01-01T00:00:00Z", created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z", version: 3,
};

const aggregate = { program } as ProgramAggregate;
const proposal = { command: "program.compliance-policy.propose", can_act: true, label: "Edit calculation rules", responsibility: "OWNER", reason: "" };
const approval = { command: "program.compliance-policy.approve", can_act: true, label: "Approve calculation rules", responsibility: "AUTHORIZER", reason: "" };
const rejection = { command: "program.compliance-policy.reject", can_act: true, label: "Reject calculation rules", responsibility: "AUTHORIZER", reason: "" };

describe("Program calculation rules", () => {
  beforeEach(() => vi.resetAllMocks());

  it("validates category totals and submits the existing Program command", async () => {
    const onUpdated = vi.fn();
    vi.mocked(proposeProgramCalculationPolicy).mockResolvedValue({ ...aggregate, program: { ...program, version: 4 } });
    render(<ProgramDetailsPanel aggregate={aggregate} operations={[proposal]} onUpdated={onUpdated} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", { name: "Set rules" }));
    const requirements = screen.getByLabelText("Requirements and controls (%)");
    const vendors = screen.getByLabelText("Vendor assessments (%)");
    fireEvent.change(requirements, { target: { value: "60" } });
    expect(screen.getByRole("button", { name: "Submit for approval" }).hasAttribute("disabled")).toBe(true);
    fireEvent.change(vendors, { target: { value: "40" } });
    fireEvent.change(screen.getByLabelText("Reason for change"), { target: { value: "Annual review" } });
    fireEvent.click(screen.getByRole("button", { name: "Submit for approval" }));
    await waitFor(() => expect(proposeProgramCalculationPolicy).toHaveBeenCalledWith("program-1", 3, {
      revision: "draft", minimum_coverage_bps: 10000,
      categories: [{ code: "REQUIREMENTS", weight_bps: 6000 }, { code: "VENDORS", weight_bps: 4000 }],
    }, "Annual review"));
    expect(onUpdated).toHaveBeenCalled();
  });

  it("adds optional evidence or issue factors without allowing incomplete weights", async () => {
    vi.mocked(proposeProgramCalculationPolicy).mockResolvedValue({ ...aggregate, program: { ...program, version: 4 } });
    render(<ProgramDetailsPanel aggregate={aggregate} operations={[proposal]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", { name: "Set rules" }));
    fireEvent.change(screen.getByLabelText("Add scoring factor"), { target: { value: "ISSUES" } });
    fireEvent.click(screen.getByRole("button", { name: "Add factor" }));
    expect(screen.getByLabelText("Unresolved issues (%)")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Submit for approval" }).hasAttribute("disabled")).toBe(true);
    fireEvent.change(screen.getByLabelText("Requirements and controls (%)"), { target: { value: "60" } });
    fireEvent.change(screen.getByLabelText("Unresolved issues (%)"), { target: { value: "10" } });
    fireEvent.change(screen.getByLabelText("Reason for change"), { target: { value: "Issue weighting" } });
    fireEvent.click(screen.getByRole("button", { name: "Submit for approval" }));
    await waitFor(() => expect(proposeProgramCalculationPolicy).toHaveBeenCalledWith("program-1", 3, expect.objectContaining({
      categories: [
        { code: "REQUIREMENTS", weight_bps: 6000 },
        { code: "VENDORS", weight_bps: 3000 },
        { code: "ISSUES", weight_bps: 1000 },
      ],
    }), "Issue weighting"));
  });

  it("does not expose approval to the maker", () => {
    const pending = { number: 1, checksum: "abc", maker_id: "owner", proposed_at: "2026-10-15T08:00:00Z",
      definition: { revision: "program-v1", minimum_coverage_bps: 10000, categories: [{ code: "REQUIREMENTS", weight_bps: 10000 }] } };
    render(<ProgramDetailsPanel aggregate={{ ...aggregate, program: { ...program, calculation_policy: { last_revision: 1, pending } } }}
      operations={[{ ...approval, can_act: false }, { ...rejection, can_act: false }]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    expect(screen.getByText("Awaiting approval")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Approve rules" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Reject" })).toBeNull();
  });

  it("accepts an independent decision and preserves errors", async () => {
    const pending = { number: 1, checksum: "abc", maker_id: "owner", proposed_at: "2026-10-15T08:00:00Z",
      definition: { revision: "program-v1", minimum_coverage_bps: 10000, categories: [{ code: "REQUIREMENTS", weight_bps: 10000 }] } };
    const state = { ...aggregate, program: { ...program, calculation_policy: { last_revision: 1, pending } } };
    vi.mocked(decideProgramCalculationPolicy).mockResolvedValue({ ...state, program: { ...state.program, version: 4 } });
    render(<ProgramDetailsPanel aggregate={state} operations={[approval, rejection]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.change(screen.getByLabelText("Review reason"), { target: { value: "Scope confirmed" } });
    fireEvent.click(screen.getByRole("button", { name: "Approve rules" }));
    await waitFor(() => expect(decideProgramCalculationPolicy).toHaveBeenCalledWith("program-1", 3, "approve", "Scope confirmed"));
  });
});
