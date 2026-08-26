import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import type { MatterAggregate } from "../types";
import { loadEvidenceSources } from "../api";
import { defineMatterOutcomeCheck } from "../matterOperationsApi";
import { MatterOutcomePanel } from "./MatterOutcomePanel";

vi.mock("../matterOperationsApi", () => ({ defineMatterOutcomeCheck: vi.fn() }));
vi.mock("../continuityCommands", () => ({ recordVerificationResult: vi.fn(), transitionMatter: vi.fn() }));
vi.mock("../api", () => ({ loadEvidenceSources: vi.fn() }));

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(loadEvidenceSources).mockResolvedValue([
    { id: "source-1", tenant_id: "bank", legal_entity_id: "entity-1", code: "CORE-BAL", name: "Core banking balance report", type: "SYSTEM", authority_class: "AUTHORITATIVE", expected_freshness_minutes: 60, health: "HEALTHY", status: "ACTIVE", version: 2 },
    { id: "source-other", tenant_id: "bank", legal_entity_id: "entity-2", code: "OTHER", name: "Other entity report", type: "SYSTEM", authority_class: "AUTHORITATIVE", expected_freshness_minutes: 60, health: "HEALTHY", status: "ACTIVE", version: 1 },
  ]);
});

it("captures the complete outcome contract in business fields with an eligible reviewer", async () => {
  const aggregate = { matter: { id: "matter-1", legal_entity_id: "entity-1", version: 7 }, actions: [{ id: "action-1", title: "Restore account posting" }], verification_contracts: [], verification_results: [], closure: { ready: false, reasons: [] } } as unknown as MatterAggregate;
  const operations = [{ command: "matter.outcome.define", label: "Define an outcome check", responsibility: "REVIEWER", can_act: true, reason: "", candidates: [{ id: "reviewer-1", display_name: "Ada Okafor", kind: "PERSON", role: "Internal Audit reviewer" }] }];
  vi.mocked(defineMatterOutcomeCheck).mockResolvedValue({ ...aggregate, matter: { ...aggregate.matter, version: 8 } });
  render(<MatterOutcomePanel aggregate={aggregate} operations={operations} onUpdated={vi.fn()} onReload={vi.fn()}/>);

  fireEvent.click(screen.getByRole("button", { name: "Define outcome check" }));
  await screen.findByRole("option", { name: "Core banking balance report" });
  expect(screen.queryByRole("option", { name: "Other entity report" })).toBeNull();
  expect(screen.getByLabelText("Expected outcome")).toBeTruthy();
  expect(screen.getByLabelText("Scope covered")).toBeTruthy();
  expect(screen.getByLabelText("How the outcome will be measured")).toBeTruthy();
  expect(screen.getByLabelText("Current baseline")).toBeTruthy();
  expect(screen.getByLabelText("Success threshold")).toBeTruthy();
  expect(screen.getByLabelText("Registered measurement source (optional)")).toBeTruthy();
  expect(screen.getByLabelText("Observation period (days)")).toBeTruthy();
  expect(screen.getByLabelText("Independent reviewer")).toBeTruthy();
  expect(screen.getByLabelText("If the outcome is not achieved")).toBeTruthy();
  expect(screen.queryByText("reviewer-1")).toBeNull();
  expect(screen.queryByText("source-1")).toBeNull();

  fireEvent.change(screen.getByLabelText("Expected outcome"), { target: { value: "Posting remains available after restoration." } });
  fireEvent.change(screen.getByLabelText("Scope covered"), { target: { value: "Retail current accounts processed by the Lagos core banking service." } });
  fireEvent.change(screen.getByLabelText("How the outcome will be measured"), { target: { value: "Review successful and failed postings in the daily availability report." } });
  fireEvent.change(screen.getByLabelText("Current baseline"), { target: { value: "Posting is unavailable for all retail current accounts." } });
  fireEvent.change(screen.getByLabelText("Success threshold"), { target: { value: "At least 99.9% of postings complete without an availability error." } });
  fireEvent.change(screen.getByLabelText("Registered measurement source (optional)"), { target: { value: "source-1" } });
  fireEvent.change(screen.getByLabelText("Observation period (days)"), { target: { value: "2" } });
  fireEvent.change(screen.getByLabelText("Independent reviewer"), { target: { value: "reviewer-1" } });
  fireEvent.change(screen.getByLabelText("If the outcome is not achieved"), { target: { value: "REOPEN" } });
  fireEvent.click(screen.getByRole("button", { name: "Save outcome check" }));

  await waitFor(() => expect(defineMatterOutcomeCheck).toHaveBeenCalledWith("matter-1", 7, {
    actionID: "action-1",
    expectedOutcome: "Posting remains available after restoration.",
    baseline: { description: "Posting is unavailable for all retail current accounts." },
    scope: { description: "Retail current accounts processed by the Lagos core banking service.", measurement_method: "Review successful and failed postings in the daily availability report." },
    threshold: { success_condition: "At least 99.9% of postings complete without an availability error." },
    measurementSourceID: "source-1",
    observationPeriodMinutes: 2880,
    reviewerCandidateID: "reviewer-1",
    failureResponse: "REOPEN",
  }));
});

it("shows every stored outcome contract term with safe source and reviewer labels", async () => {
  const aggregate = { matter: { id: "matter-1", legal_entity_id: "entity-1", version: 9 }, actions: [], verification_contracts: [{ id: "contract-1", expected_outcome: "Posting remains available", baseline: { description: "Service was unavailable" }, scope: { description: "Retail current accounts", measurement_method: "Review the daily availability report" }, measurement_source_id: "source-1", threshold: { success_condition: "99.9% successful postings" }, observation_period_minutes: 2880, authority_principal_id: "reviewer-1", failure_response: "BLOCK_CLOSE", status: "ACTIVE" }], verification_results: [], closure: { ready: false, reasons: [] } } as unknown as MatterAggregate;
  const operations = [{ command: "matter.outcome.define", label: "Define", responsibility: "REVIEWER", can_act: false, reason: "Read only", candidates: [{ id: "reviewer-1", display_name: "Ada Okafor", kind: "PERSON", role: "Internal Audit reviewer" }] }];
  render(<MatterOutcomePanel aggregate={aggregate} operations={operations} onUpdated={vi.fn()} onReload={vi.fn()}/>);

  expect(await screen.findByText("Core banking balance report")).toBeTruthy();
  expect(screen.getByText("Retail current accounts")).toBeTruthy();
  expect(screen.getByText("Service was unavailable")).toBeTruthy();
  expect(screen.getByText("Review the daily availability report")).toBeTruthy();
  expect(screen.getByText("99.9% successful postings")).toBeTruthy();
  expect(screen.getByText("2 days")).toBeTruthy();
  expect(screen.getByText("Ada Okafor")).toBeTruthy();
  expect(screen.queryByText("source-1")).toBeNull();
  expect(screen.queryByText("reviewer-1")).toBeNull();
});

it("renders bounded legacy scalar contract terms as human key and value labels", () => {
  const aggregate = { matter: { id: "matter-1", version: 3 }, actions: [], verification_contracts: [{ id: "contract-legacy", expected_outcome: "No privileged account lacks approval", baseline: { unresolved: 4 }, scope: { accounts: 4 }, threshold: { unresolved: 0 }, observation_period_minutes: 0, failure_response: "REOPEN", status: "ACTIVE" }], verification_results: [], closure: { ready: false, reasons: [] } } as unknown as MatterAggregate;
  render(<MatterOutcomePanel aggregate={aggregate} operations={[]} onUpdated={vi.fn()} onReload={vi.fn()}/>);
  expect(screen.getByText("Accounts: 4")).toBeTruthy();
  expect(screen.getByText("Unresolved: 4")).toBeTruthy();
  expect(screen.getByText("Unresolved: 0")).toBeTruthy();
  expect(screen.queryByText(/\{"unresolved"/)).toBeNull();
});

it("allows a manual measurement method when registered sources are unavailable", async () => {
  vi.mocked(loadEvidenceSources).mockRejectedValue(new Error("catalog unavailable"));
  const aggregate = { matter: { id: "matter-1", version: 7 }, actions: [], verification_contracts: [], verification_results: [], closure: { ready: false, reasons: [] } } as unknown as MatterAggregate;
  const operations = [{ command: "matter.outcome.define", label: "Define an outcome check", responsibility: "REVIEWER", can_act: true, reason: "", candidates: [{ id: "reviewer-1", display_name: "Ada Okafor", kind: "PERSON", role: "Reviewer" }] }];
  vi.mocked(defineMatterOutcomeCheck).mockResolvedValue({ ...aggregate, matter: { ...aggregate.matter, version: 8 } });
  render(<MatterOutcomePanel aggregate={aggregate} operations={operations} onUpdated={vi.fn()} onReload={vi.fn()}/>);
  fireEvent.click(screen.getByRole("button", { name: "Define outcome check" }));
  expect(await screen.findByText(/Registered evidence sources could not be loaded/)).toBeTruthy();
  fireEvent.change(screen.getByLabelText("Expected outcome"), { target: { value: "Posting remains available." } });
  fireEvent.change(screen.getByLabelText("Scope covered"), { target: { value: "Retail accounts." } });
  fireEvent.change(screen.getByLabelText("How the outcome will be measured"), { target: { value: "An independent reviewer samples the daily posting report." } });
  fireEvent.change(screen.getByLabelText("Current baseline"), { target: { value: "Posting is unavailable." } });
  fireEvent.change(screen.getByLabelText("Success threshold"), { target: { value: "All sampled postings succeed." } });
  fireEvent.change(screen.getByLabelText("If the outcome is not achieved"), { target: { value: "BLOCK_CLOSE" } });
  const save = screen.getByRole("button", { name: "Save outcome check" });
  expect((save as HTMLButtonElement).disabled).toBe(false);
  fireEvent.click(save);
  await waitFor(() => expect(defineMatterOutcomeCheck).toHaveBeenCalledWith("matter-1", 7, expect.objectContaining({
    measurementSourceID: undefined,
    scope: { description: "Retail accounts.", measurement_method: "An independent reviewer samples the daily posting report." },
  })));
});

it("shows all results within a bounded labelled outcome history", () => {
  const results = Array.from({ length: 21 }, (_, index) => ({ id: `result-${index}`, contract_id: "contract-1", result: index ? "PASS" : "FAIL", rationale: `Conclusion ${index}`, reviewer_principal_id: index ? "hidden-reviewer" : "reviewer-1", observed_at: new Date(Date.UTC(2026, 7, 25 - index)).toISOString() }));
  const aggregate = { matter: { id: "matter-1", version: 9 }, actions: [], verification_contracts: [{ id: "contract-1", expected_outcome: "Accounts are approved", observation_period_minutes: 0, failure_response: "BLOCK_CLOSE", status: "ACTIVE" }], verification_results: results, closure: { ready: false, reasons: [] } } as unknown as MatterAggregate;
  const operations = [{ command: "matter.outcome.record", subresource_id: "contract-1", label: "Record result", responsibility: "REVIEWER", can_act: false, reason: "This result is read only.", assigned_to: { id: "reviewer-1", display_name: "Ada Okafor", kind: "PERSON", role: "Reviewer" }, candidates: [{ id: "reviewer-1", display_name: "Ada Okafor", kind: "PERSON", role: "Reviewer" }] }];
  render(<MatterOutcomePanel aggregate={aggregate} operations={operations} onUpdated={vi.fn()} onReload={vi.fn()}/>);
  fireEvent.click(screen.getByText("View outcome result history (21)"));
  expect(screen.getByText(/Showing 20 of 21 stored results for issue version 9/)).toBeTruthy();
  expect(screen.getByText(/Recorded 2026-08-25 by Ada Okafor/)).toBeTruthy();
  expect(screen.getByText(/1 additional results/)).toBeTruthy();
  expect(screen.queryByText(/hidden-reviewer/)).toBeNull();
});
