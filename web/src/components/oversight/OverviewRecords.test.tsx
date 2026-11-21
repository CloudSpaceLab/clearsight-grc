import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { OverviewRecords } from "./OverviewRecords";
import type { ProgramSummary } from "../../summaryTypes";
import type { RiskSummary } from "../../riskTypes";

it("shows scoped Program and risk records with their actual freshness and opens the chosen record", async () => {
  const loadPrograms = vi.fn().mockResolvedValue({ organization_scope_id: "ops", generated_at: "2026-11-20T10:00:00Z", items: [
    { program: { id: "p1", name: "Payments", code: "PAY", version: 3 }, requirement_count: 8, state_label: "Up to date", overall_state: "CURRENT", assessed_program_version: 2, projection_stale: true } as ProgramSummary,
  ] });
  const loadRisks = vi.fn().mockResolvedValue({ items: [
    { risk: { id: "r1", name: "Settlement outage", code: "R1", version: 2 }, latest_assessment: { risk_version: 2, appetite_position: "BREACHED", assessed_at: "2026-11-19T10:00:00Z" } } as RiskSummary,
  ] });
  const onOpenProgram = vi.fn(), onOpenRisk = vi.fn();
  render(<OverviewRecords organizationScopeID="ops" onOpenProgram={onOpenProgram} onOpenRisk={onOpenRisk} loadPrograms={loadPrograms} loadRisks={loadRisks}/>);
  const program = await screen.findByRole("region", { name: "Program assurance" });
  expect(within(program).getByText("Out of date")).toBeTruthy();
  expect(within(program).queryByText("Up to date")).toBeNull();
  expect(within(program).getByText("8 requirements")).toBeTruthy();
  expect(loadPrograms).toHaveBeenCalledWith({ organizationScopeID: "ops", limit: 4 });
  fireEvent.click(within(program).getByRole("button", { name: "Open Payments" }));
  expect(onOpenProgram).toHaveBeenCalledWith("p1");
  const risk = await screen.findByRole("button", { name: "Review Settlement outage" });
  expect(loadRisks).toHaveBeenCalledWith(expect.objectContaining({ organizationScopeID: "ops", appetitePosition: "BREACHED", status: "ACTIVE", limit: 4 }), expect.any(AbortSignal));
  fireEvent.click(risk);
  expect(onOpenRisk).toHaveBeenCalledWith("r1");
});

it("does not show a Program result for a different selected scope", async () => {
  const loadPrograms = vi.fn().mockResolvedValue({ organization_scope_id: "other", items: [{ program: { id: "p1", name: "Wrong scope" } }] });
  const loadRisks = vi.fn().mockResolvedValue({ items: [] });
  render(<OverviewRecords organizationScopeID="ops" loadPrograms={loadPrograms} loadRisks={loadRisks}/>);
  await waitFor(() => expect(screen.getByText("Program information is unavailable.")).toBeTruthy());
  expect(screen.queryByText("Wrong scope")).toBeNull();
});

it("removes prior scope records immediately while the new scope loads", async () => {
  let resolveNext: (page: unknown) => void = () => undefined;
  const loadPrograms = vi.fn().mockResolvedValueOnce({ organization_scope_id: "ops", items: [{ program: { id: "p1", name: "Operations", version: 1 }, requirement_count: 1, assessed_program_version: 0 }] })
    .mockImplementationOnce(() => new Promise((resolve) => { resolveNext = resolve; }));
  const loadRisks = vi.fn().mockResolvedValue({ items: [] });
  const { rerender } = render(<OverviewRecords organizationScopeID="ops" loadPrograms={loadPrograms} loadRisks={loadRisks}/>);
  await screen.findByText("Operations");
  rerender(<OverviewRecords organizationScopeID="technology" loadPrograms={loadPrograms} loadRisks={loadRisks}/>);
  expect(screen.queryByText("Operations")).toBeNull();
  resolveNext({ organization_scope_id: "technology", items: [] });
  await screen.findByText("No programs in this scope.");
});
