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

it("does not present a stale assessment as the current appetite position", async () => {
  const stalePage: RiskPage = {
    items: [{ risk: { ...risk, version: 4 }, latest_assessment: assessment, active_appetite: appetite }],
  };
  render(<RiskRegister onOpenRisk={vi.fn()} loadPage={vi.fn().mockResolvedValue(stalePage)}/>);

  expect(await screen.findByText("Reassessment needed")).toBeTruthy();
  expect(screen.queryByText("Outside appetite")).toBeNull();

  render(<RiskRecord riskID="risk-1" onBack={vi.fn()} loadRisk={vi.fn().mockResolvedValue({ ...aggregate, risk: { ...risk, version: 4 } })}/>);
  expect((await screen.findAllByText("Reassessment needed")).length).toBeGreaterThan(0);
});

it("keeps a cross-scope or missing Risk non-disclosing", async () => {
  const loadRisk = vi.fn().mockRejectedValue(new ApiError(404, "This risk is not available in your legal entity.", "risk_not_found"));
  const onBack = vi.fn();
  render(<RiskRecord riskID="missing" onBack={onBack} loadRisk={loadRisk}/>);

  expect(await screen.findByRole("heading", { name: "Risk not found" })).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Back to risks" }));
  expect(onBack).toHaveBeenCalledTimes(1);
});
