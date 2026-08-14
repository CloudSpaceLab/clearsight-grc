import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import type { ProgramAggregate } from "../types";
import { canCurrentActorTransitionProgram, transitionProgram } from "../continuityCommands";
import { ProgramLifecycleControls, programTransitionActionLabel } from "./ProgramLifecycleControls";

vi.mock("../continuityCommands", () => ({
  canCurrentActorTransitionProgram: vi.fn(),
  transitionProgram: vi.fn(),
}));

const program: ProgramAggregate = {
  state_label: "Current",
  program: {
    id: "program-1", tenant_id: "bank-1", code: "AML", name: "AML Program", type: "REGULATORY", status: "ACTIVE", owning_function: "Compliance",
    owner_principal_id: "owner-1", authority_principal_id: "actor-1", scope: {}, effective_from: "2026-01-01T00:00:00Z",
    created_at: "2026-01-01T00:00:00Z", updated_at: "2026-08-09T09:00:00Z", version: 5,
  },
  requirements: [], applicability: [], control_objectives: [], control_implementations: [], requirement_control_links: [], evidence_contracts: [], evidence_assessments: [], triggers: [],
};

beforeEach(() => vi.clearAllMocks());

it("uses action-oriented lifecycle labels", () => {
  expect(programTransitionActionLabel("ACTIVE")).toBe("Request activation");
  expect(programTransitionActionLabel("PAUSED")).toBe("Request pause");
  expect(programTransitionActionLabel("RETIRED")).toBe("Request retirement");
});

it("rejects blank and whitespace-only rationale before issuing a command", async () => {
  vi.mocked(canCurrentActorTransitionProgram).mockResolvedValue(true);
  render(<ProgramLifecycleControls aggregate={program} onUpdated={vi.fn()}/>);

  const submit = await screen.findByRole("button", { name: "Request pause" });
  expect((submit as HTMLButtonElement).disabled).toBe(true);
  fireEvent.change(screen.getByLabelText("Rationale"), { target: { value: "   " } });
  expect((submit as HTMLButtonElement).disabled).toBe(true);
  fireEvent.submit(submit.closest("form")!);
  expect(transitionProgram).not.toHaveBeenCalled();
});

it("disables the command, submits once and preserves the receipt through the status rerender", async () => {
  vi.mocked(canCurrentActorTransitionProgram).mockResolvedValue(true);
  let resolveCommand: ((value: ProgramAggregate) => void) | undefined;
  vi.mocked(transitionProgram).mockReturnValue(new Promise((resolve) => { resolveCommand = resolve; }));
  const onUpdated = vi.fn();
  const { rerender } = render(<ProgramLifecycleControls aggregate={program} onUpdated={onUpdated}/>);

  const submit = await screen.findByRole("button", { name: "Request pause" });
  fireEvent.change(screen.getByLabelText("Rationale"), { target: { value: "  Pause while ownership is corrected.  " } });
  fireEvent.click(submit);
  fireEvent.click(submit);

  expect((submit as HTMLButtonElement).disabled).toBe(true);
  expect((screen.getByRole("button", { name: "Recording…" }) as HTMLButtonElement).disabled).toBe(true);
  expect(transitionProgram).toHaveBeenCalledTimes(1);
  expect(transitionProgram).toHaveBeenCalledWith("program-1", 5, "PAUSED", "Pause while ownership is corrected.");

  const updated = { ...program, program: { ...program.program, status: "PAUSED", version: 6 } };
  resolveCommand?.(updated);
  await waitFor(() => expect(onUpdated).toHaveBeenCalledTimes(1));
  expect(await screen.findByText("Program status updated.", { exact: true })).toBeTruthy();

  rerender(<ProgramLifecycleControls aggregate={updated} onUpdated={onUpdated}/>);
  expect(await screen.findByText("Program status updated.", { exact: true })).toBeTruthy();
  expect((await screen.findByRole("button", { name: "Request activation" }) as HTMLButtonElement).disabled).toBe(true);
});
