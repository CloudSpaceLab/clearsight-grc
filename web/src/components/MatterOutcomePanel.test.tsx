import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { MatterAggregate } from "../types";
import { MatterOutcomePanel } from "./MatterOutcomePanel";

vi.mock("../matterOperationsApi", () => ({ defineMatterOutcomeCheck: vi.fn() }));
vi.mock("../continuityCommands", () => ({ recordVerificationResult: vi.fn(), transitionMatter: vi.fn() }));

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
