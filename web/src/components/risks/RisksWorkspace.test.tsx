import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { ApiError } from "../../http";
import type { RiskAggregate, RiskPage, RiskRecord as RiskValue } from "../../riskTypes";
import { RiskRecord } from "./RiskRecord";
import { RiskRegister } from "./RiskRegister";

const risk: RiskValue = {
  id: "risk-1",
  tenant_id: "tenant-1",
  legal_entity_id: "entity-1",
  code: "NET-RES-01",
  name: "Network resilience",
  category: "Operational resilience",
  statement: "Critical network service may exceed the approved recovery tolerance.",
  cause: "Primary and recovery paths can become unavailable.",
  event: "Network service interruption",
  impact: "Customers cannot access critical services within the approved tolerance.",
  scope: { service: "Critical network", objective: "Service continuity" },
  owner_principal_id: "10000000-0000-4000-8000-000000000001",
  status: "ACTIVE",
  version: 3,
  created_at: "2026-10-02T09:00:00Z",
  updated_at: "2026-10-02T09:12:00Z",
};

const assessment = {
  id: "assessment-1",
  risk_id: risk.id,
  risk_version: 3,
  kind: "RESIDUAL" as const,
  method_code: "QUAL-5X5",
  method_version: "v1",
  dimensions: { likelihood: 4, impact: 5 },
  assumptions: {},
  evidence_references: [],
  assessed_by: "10000000-0000-4000-8000-000000000002",
  appetite_statement_id: "appetite-1",
  appetite_position: "BREACHED" as const,
  appetite_rationale: "Recovery exceeds tolerance.",
  assessed_at: "2026-10-02T09:12:00Z",
  created_at: "2026-10-02T09:12:00Z",
};

const appetite = {
  id: "appetite-1",
  risk_id: risk.id,
  risk_version: 2,
  version: 1,
  statement: "Keep disruption below 30 minutes.",
  rule: { max_minutes: 30 },
  rationale: "Protect critical customer services.",
  owner_principal_id: risk.owner_principal_id,
  authority_principal_id: "10000000-0000-4000-8000-000000000003",
  status: "ACTIVE" as const,
  effective_from: "2026-10-02T09:05:00Z",
  created_at: "2026-10-02T09:05:00Z",
};

const page: RiskPage = {
  items: [{ risk, latest_assessment: assessment, active_appetite: appetite }],
};

const aggregate: RiskAggregate = {
  risk,
  assessments: [assessment],
  appetite: [appetite],
  active_appetite: appetite,
  controls: [{
    id: "risk-control-1",
    risk_id: risk.id,
    risk_version: 3,
    catalog_link_id: "catalog-link-1",
    created_at: "2026-10-02T09:11:00Z",
  }],
  control_details: [{
    link: {
      id: "risk-control-1",
      risk_id: risk.id,
      risk_version: 3,
      catalog_link_id: "catalog-link-1",
      created_at: "2026-10-02T09:11:00Z",
    },
    definition: {
      id: "control-definition-1",
      code: "NET-01",
      name: "Network recovery testing",
      objective: "Recovery paths meet the approved service tolerance.",
      description: "",
      category: "Resilience",
      status: "ACTIVE",
      version: 1,
    },
    program_id: "program-1",
    program_name: "Network resilience program",
    implementation_id: "implementation-1",
    objective_id: "objective-1",
    implementation_name: "Quarterly recovery exercise",
    implementation_type: "OWNER_REVIEW",
    implementation_status: "IMPLEMENTED",
    implementation_version: 4,
    owner_display_name: "Jordan Ellis",
    owner_assigned: true,
    evidence: [{
      contract_id: "evidence-1",
      name: "Recovery exercise result",
      conclusion: "SUPPORTED",
      assessed_at: "2026-10-02T08:45:00Z",
    }],
  }],
  control_details_complete: true,
};

it("renders a bounded Risk register with working-language appetite state", async () => {
  const loadPage = vi.fn().mockResolvedValue(page);
  render(<RiskRegister legalEntityName="Clear Bank Nigeria" onOpenRisk={vi.fn()} loadPage={loadPage}/>);

  const table = await screen.findByRole("table", { name: "Risk register" });
  expect(within(table).getByText("Network resilience")).toBeTruthy();
  expect(within(table).getByText("Outside appetite")).toBeTruthy();
  expect(within(table).getByText("Residual")).toBeTruthy();
  expect(within(table).getByText("QUAL-5X5 · v1")).toBeTruthy();
  expect(screen.getByText("1 shown")).toBeTruthy();
  expect(screen.queryByText(risk.owner_principal_id!)).toBeNull();
});

it("opens the exact Risk from the visible row action", async () => {
  const onOpenRisk = vi.fn();
  render(<RiskRegister onOpenRisk={onOpenRisk} loadPage={vi.fn().mockResolvedValue(page)}/>);
  fireEvent.click(await screen.findByRole("button", { name: /Open risk for Network resilience/ }));
  expect(onOpenRisk).toHaveBeenCalledWith("risk-1");
});

it("sends search to the bounded register read and clears it", async () => {
  const loadPage = vi.fn().mockResolvedValue(page);
  render(<RiskRegister onOpenRisk={vi.fn()} loadPage={loadPage}/>);
  await screen.findByText("Network resilience");

  fireEvent.change(screen.getByRole("searchbox", { name: "Search risks" }), { target: { value: "network" } });
  await waitFor(() => expect(loadPage).toHaveBeenLastCalledWith(expect.objectContaining({ search: "network", limit: 25 }), expect.any(AbortSignal)));
  fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
  await waitFor(() => expect(loadPage).toHaveBeenLastCalledWith(expect.objectContaining({ search: undefined, limit: 25 }), expect.any(AbortSignal)));
});

it("uses the opaque cursor for next-page navigation", async () => {
  const secondRisk = { ...risk, id: "risk-2", code: "PAY-01", name: "Payment interruption" };
  const loadPage = vi.fn()
    .mockResolvedValueOnce({ ...page, next_cursor: "cursor-2" })
    .mockResolvedValueOnce({ items: [{ risk: secondRisk }] });
  render(<RiskRegister onOpenRisk={vi.fn()} loadPage={loadPage}/>);
  await screen.findByText("Network resilience");

  fireEvent.click(screen.getByRole("button", { name: "Load next page" }));
  expect(await screen.findByText("Payment interruption")).toBeTruthy();
  expect(loadPage).toHaveBeenLastCalledWith(expect.objectContaining({ cursor: "cursor-2" }), expect.any(AbortSignal));
});

it("shows statement, impact, scope and history without exposing principal identifiers", async () => {
  render(<RiskRecord riskID="risk-1" onBack={vi.fn()} loadRisk={vi.fn().mockResolvedValue(aggregate)}/>);

  expect(await screen.findByRole("heading", { name: "Network resilience" })).toBeTruthy();
  expect(screen.getByText(risk.statement)).toBeTruthy();
  expect(screen.getByText(risk.impact)).toBeTruthy();
  expect(screen.getByText("Critical network")).toBeTruthy();
  expect(screen.getByText("Service continuity")).toBeTruthy();
  expect(screen.getByText("Assigned")).toBeTruthy();
  expect(screen.getByRole("table", { name: "Risk assessments" })).toBeTruthy();
  expect(screen.getByRole("table", { name: "Risk appetite history" })).toBeTruthy();
  expect(screen.queryByText(risk.owner_principal_id!)).toBeNull();
  expect(screen.queryByText(assessment.assessed_by)).toBeNull();
  expect(screen.queryByText(appetite.authority_principal_id)).toBeNull();
});

it("shows linked controls from Program truth and opens the existing safeguard workspace", async () => {
  const onOpenProgramControl = vi.fn();
  render(<RiskRecord riskID="risk-1" onBack={vi.fn()} onOpenProgramControl={onOpenProgramControl} loadRisk={vi.fn().mockResolvedValue(aggregate)}/>);

  const controls = await screen.findByRole("table", { name: "Risk controls" });
  expect(within(controls).getByText("Network recovery testing")).toBeTruthy();
  expect(within(controls).getByText("Quarterly recovery exercise")).toBeTruthy();
  expect(within(controls).getByText("Implemented")).toBeTruthy();
  expect(within(controls).getByText("Jordan Ellis")).toBeTruthy();
  expect(within(controls).getByText("1 active check · 1 supported")).toBeTruthy();

  fireEvent.click(within(controls).getByRole("button", { name: /Open control for Network recovery testing/ }));
  expect(onOpenProgramControl).toHaveBeenCalledWith("program-1", "objective-1");
});

it("lets the direct Risk owner link an unlinked reusable control and reloads the record", async () => {
  const loadRisk = vi.fn().mockResolvedValue(aggregate);
  const loadControlCandidates = vi.fn().mockResolvedValue({
    complete: true,
    items: [
      {
        catalog_link_id: "catalog-link-1",
        definition: aggregate.control_details![0]!.definition,
        program_id: "program-1",
        program_name: "Network resilience program",
        implementation_id: "implementation-1",
        implementation_name: "Quarterly recovery exercise",
        implementation_status: "IMPLEMENTED",
      },
      {
        catalog_link_id: "catalog-link-2",
        definition: { ...aggregate.control_details![0]!.definition, id: "definition-2", code: "NET-02", name: "Network failover review" },
        program_id: "program-2",
        program_name: "Infrastructure assurance",
        implementation_id: "implementation-2",
        implementation_name: "Monthly failover review",
        implementation_status: "IN_PROGRESS",
      },
    ],
  });
  const linkControl = vi.fn().mockResolvedValue({
    risk: { ...risk, version: 4 },
    control: {
      id: "risk-control-2",
      risk_id: risk.id,
      risk_version: 4,
      catalog_link_id: "catalog-link-2",
      created_at: "2026-10-02T09:20:00Z",
    },
  });

  render(<RiskRecord
    riskID="risk-1"
    actorID={risk.owner_principal_id}
    onBack={vi.fn()}
    loadRisk={loadRisk}
    loadControlCandidates={loadControlCandidates}
    linkControl={linkControl}
  />);

  fireEvent.click(await screen.findByRole("button", { name: "Link control" }));
  await waitFor(() => expect(loadControlCandidates).toHaveBeenCalledTimes(1));
  expect(screen.queryByText(/Network recovery testing · Quarterly recovery exercise/)).toBeNull();
  expect((await screen.findAllByText("Network failover review · Monthly failover review · Infrastructure assurance")).length).toBeGreaterThan(0);

  fireEvent.click(screen.getByRole("button", { name: "Link control" }));
  await waitFor(() => expect(linkControl).toHaveBeenCalledWith("risk-1", 3, "catalog-link-2"));
  expect(loadRisk).toHaveBeenCalledTimes(2);
});

it("does not offer Risk control linking to a non-owner", async () => {
  render(<RiskRecord riskID="risk-1" actorID="someone-else" onBack={vi.fn()} loadRisk={vi.fn().mockResolvedValue(aggregate)}/>);
  await screen.findByRole("heading", { name: "Network resilience" });
  expect(screen.queryByRole("button", { name: "Link control" })).toBeNull();
});

it("does not present a stale assessment as the current appetite position", async () => {
  const stalePage: RiskPage = {
    items: [{ risk: { ...risk, version: 4 }, latest_assessment: assessment, active_appetite: appetite }],
  };
  render(<RiskRegister onOpenRisk={vi.fn()} loadPage={vi.fn().mockResolvedValue(stalePage)}/>);

  const register = await screen.findByRole("table", { name: "Risk register" });
  expect(within(register).getByText("Reassessment needed")).toBeTruthy();
  expect(within(register).queryByText("Outside appetite")).toBeNull();

  render(<RiskRecord riskID="risk-1" onBack={vi.fn()} loadRisk={vi.fn().mockResolvedValue({ ...aggregate, risk: { ...risk, version: 4 } })}/>);
  const currentState = await screen.findByRole("group", { name: "Current risk state" });
  expect(within(currentState).getByText("Reassessment needed")).toBeTruthy();
  expect(within(currentState).queryByText("Outside appetite")).toBeNull();
});

it("does not keep a recorded breach current after its appetite expires", async () => {
  const expiredPage: RiskPage = {
    items: [{ risk, latest_assessment: assessment }],
  };
  render(<RiskRegister onOpenRisk={vi.fn()} loadPage={vi.fn().mockResolvedValue(expiredPage)}/>);
  const register = await screen.findByRole("table", { name: "Risk register" });
  expect(within(register).getByText("No current appetite")).toBeTruthy();
  expect(within(register).queryByText("Outside appetite")).toBeNull();

  render(<RiskRecord riskID="risk-1" onBack={vi.fn()} loadRisk={vi.fn().mockResolvedValue({ ...aggregate, active_appetite: undefined })}/>);
  const currentState = await screen.findByRole("group", { name: "Current risk state" });
  expect(within(currentState).getByText("No current appetite")).toBeTruthy();
  expect(within(currentState).queryByText("Outside appetite")).toBeNull();
});

it("keeps a cross-scope or missing Risk non-disclosing", async () => {
  const loadRisk = vi.fn().mockRejectedValue(new ApiError(404, "This risk is not available in your legal entity.", "risk_not_found"));
  const onBack = vi.fn();
  render(<RiskRecord riskID="missing" onBack={onBack} loadRisk={loadRisk}/>);

  expect(await screen.findByRole("heading", { name: "Risk not found" })).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Back to risks" }));
  expect(onBack).toHaveBeenCalledTimes(1);
});
