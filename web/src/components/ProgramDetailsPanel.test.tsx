import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { decideProgramCalculationPolicy, decideProgramFrameworkAdoption, loadFrameworkEditionPage, proposeProgramCalculationPolicy, proposeProgramFrameworkAdoption } from "../programOperationsApi";
import type { ProgramAggregate } from "../types";
import { ProgramDetailsPanel } from "./ProgramDetailsPanel";

vi.mock("../programOperationsApi", () => ({
  proposeProgramCalculationPolicy: vi.fn(),
  decideProgramCalculationPolicy: vi.fn(),
  loadFrameworkEditionPage: vi.fn(),
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

  it("preserves approved expected vendors when only calculation weights change", async () => {
    const obligation = {
      key: "vendor-1:security", relationship_id: "vendor-1", form_template_id: "security",
      form_template_version: 2, review_kind: "PERIODIC" as const, weight: 1,
    };
    const definition = {
      revision: "program-v1", minimum_coverage_bps: 10000,
      categories: [{ code: "REQUIREMENTS", weight_bps: 7000 }, { code: "VENDORS", weight_bps: 3000 }],
      expected_vendor_assessments: [obligation],
    };
    const active = {
      number: 1, definition, checksum: "checksum",
      maker_id: "owner", approver_id: "reviewer", proposed_at: "2026-10-01T08:00:00Z",
      approved_at: "2026-10-02T08:00:00Z",
    };
    const state = { ...aggregate, program: {
      ...program, calculation_policy: { last_revision: 1, active },
    } };
    vi.mocked(proposeProgramCalculationPolicy).mockResolvedValue(state);
    render(<ProgramDetailsPanel aggregate={state} operations={[proposal]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", { name: "Edit rules" }));
    expect(screen.queryByRole("button", { name: "Remove Vendor assessments" })).toBeNull();
    fireEvent.change(screen.getByLabelText("Requirements and controls (%)"), { target: { value: "60" } });
    fireEvent.change(screen.getByLabelText("Vendor assessments (%)"), { target: { value: "40" } });
    fireEvent.change(screen.getByLabelText("Reason for change"), { target: { value: "Rebalance weights" } });
    fireEvent.click(screen.getByRole("button", { name: "Submit for approval" }));
    await waitFor(() => expect(proposeProgramCalculationPolicy).toHaveBeenCalledWith(
      "program-1", 3,
      expect.objectContaining({ expected_vendor_assessments: [obligation] }),
      "Rebalance weights",
    ));
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

describe("Program standards", () => {
  beforeEach(() => vi.resetAllMocks());

  const owner = { command: "program.framework.propose", can_act: true, label: "Add standard", responsibility: "OWNER", reason: "" };
  const approve = { command: "program.framework.approve", can_act: true, label: "Approve standard", responsibility: "AUTHORIZER", reason: "" };
  const reject = { command: "program.framework.reject", can_act: true, label: "Reject standard", responsibility: "AUTHORIZER", reason: "" };
  const publishedEdition = {
    id: "edition-1", publisher_code: "EXAMPLE", framework_code: "SECURITY",
    edition_code: "2026", title: "Reference security standard",
    jurisdiction: "GLOBAL", source_url: "https://example.org/standard",
    source_published_on: "2026-10-01", license_basis: "CITATIONS_ONLY",
  };

  it("loads published editions and proposes the selected edition without creating requirements", async () => {
    const onUpdated = vi.fn();
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({items: [publishedEdition], next_after_id: ""});
    vi.mocked(proposeProgramFrameworkAdoption).mockResolvedValue({
      ...aggregate, program: { ...program, version: 4 }, framework_adoptions: [{
        id: "adoption-1", edition_id: "edition-1", publisher_code: "EXAMPLE",
        framework_code: "SECURITY", edition_code: "2026", status: "PENDING",
        proposed_by: "owner", rationale: "Applies to our Program", proposed_at: "2026-10-15T10:00:00Z", version: 1,
      }],
    });
    render(<ProgramDetailsPanel aggregate={aggregate} operations={[owner]} onUpdated={onUpdated} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", { name: "Add standard" }));
    const choice = await screen.findByRole("radio", { name: /Reference security standard/ });
    fireEvent.click(choice);
    expect(screen.getByLabelText("Selected edition").textContent).toContain("Citations only");
    expect(screen.getByRole("link", {name: /Show published source/}).getAttribute("href")).toBe("https://example.org/standard");
    fireEvent.change(screen.getByLabelText("Reason for adoption"), { target: { value: "Applies to our Program" } });
    fireEvent.click(screen.getByRole("button", { name: "Submit for approval" }));
    await waitFor(() => expect(proposeProgramFrameworkAdoption).toHaveBeenCalledWith("program-1", 3, "edition-1", "Applies to our Program"));
    expect(onUpdated).toHaveBeenCalled();
    expect(loadFrameworkEditionPage).toHaveBeenCalledWith("GLOBAL", "");
  });

  it("shows an empty catalog instead of an unusable choice", async () => {
    vi.mocked(loadFrameworkEditionPage).mockResolvedValue({ items: [], next_after_id: "" });
    render(<ProgramDetailsPanel aggregate={aggregate} operations={[owner]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", { name: "Add standard" }));
    await waitFor(() => expect(screen.getByText("No published standards available for this jurisdiction.")).toBeTruthy());
    expect(screen.getByRole("button", { name: "Submit for approval" }).hasAttribute("disabled")).toBe(true);
  });

  it("allows an independent reviewer to approve a pending standard", async () => {
    const onUpdated = vi.fn();
    const adoption = {
      id: "adoption-1", edition_id: "edition-1", publisher_code: "EXAMPLE",
      framework_code: "SECURITY", edition_code: "2026", status: "PENDING" as const,
      proposed_by: "owner", rationale: "Security", proposed_at: "2026-10-15T10:00:00Z", version: 1,
    };
    vi.mocked(decideProgramFrameworkAdoption).mockResolvedValue({ ...aggregate, program: { ...program, version: 4 },
      framework_adoptions: [{ ...adoption, status: "ACTIVE", reviewed_by: "checker" }] });
    render(<ProgramDetailsPanel aggregate={{ ...aggregate, framework_adoptions: [adoption] }}
      operations={[approve, reject]} onUpdated={onUpdated} onReload={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", { name: "Review SECURITY 2026" }));
    fireEvent.change(screen.getByLabelText("Decision rationale for SECURITY 2026"), { target: { value: "Approved source and scope" } });
    fireEvent.click(screen.getByRole("button", { name: "Approve standard" }));
    await waitFor(() => expect(decideProgramFrameworkAdoption).toHaveBeenCalledWith("program-1", "adoption-1", 3, "approve", "Approved source and scope"));
    expect(onUpdated).toHaveBeenCalled();
  });
});
