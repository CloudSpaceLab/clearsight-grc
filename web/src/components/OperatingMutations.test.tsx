import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { MatterAggregate, ProgramAggregate, WorkflowTask } from "../types";
import { MatterWorkCommandPanel } from "./MatterWorkCommandPanel";
import { ProgramLifecycleControls } from "./ProgramLifecycleControls";
import {
  canCurrentActorTransitionProgram,
  loadActorMatterWork,
  recordMatterDecision,
  transitionProgram,
} from "../continuityCommands";

vi.mock("../continuityCommands", () => ({
  canCurrentActorTransitionProgram: vi.fn(),
  loadActorMatterWork: vi.fn(),
  recordMatterDecision: vi.fn(),
  recordVerificationResult: vi.fn(),
  transitionResponsePackage: vi.fn(),
  transitionProgram: vi.fn(),
}));

const matter: MatterAggregate = {
  type_label: "Finding",
  status_label: "Decision needed",
  next_action: "Decide",
  matter: {
    id: "matter-1", tenant_id: "bank-1", reference: "MAT-1", type: "AUDIT_FINDING", status: "DECISION_REQUIRED", priority: 4,
    title: "Resolve material finding", summary: "A governed decision is required.", scope: {}, known_facts: {}, missing_facts: [], contradictions: [],
    created_at: "2026-08-09T09:00:00Z", updated_at: "2026-08-09T09:00:00Z", version: 7,
  },
  links: [],
  decisions: [{ id: "decision-stage-1", type: "TREATMENT", status: "IN_REVIEW", rationale: "Prepared for authority review." }],
  actions: [], verification_contracts: [], verification_results: [], response_packages: [], closure: { ready: false, reasons: [] },
};

const decisionTask: WorkflowTask = {
  id: "task-1", tenant_id: "bank-1", workflow_id: "workflow-1", step_key: "decision-authorizer", responsibility: "AUTHORIZER",
  principal_id: "actor-1", title: "Decide treatment", status: "READY", version: 2,
  context: {
    type: "MATTER_WORK", matter_id: "matter-1", command_name: "matter.decision.record", subresource_type: "DECISION", subresource_id: "decision-stage-1",
    allowed_targets: "APPROVED,REJECTED", target_status: "", primary_action: "Decide", why_now: "Current policy routes this decision to you.",
  },
};

const program: ProgramAggregate = {
  state_label: "Current",
  program: {
    id: "program-1", tenant_id: "bank-1", code: "AML", name: "AML Program", type: "REGULATORY", status: "ACTIVE", owning_function: "Compliance",
    owner_principal_id: "owner-1", authority_principal_id: "actor-1", scope: {}, effective_from: "2026-01-01T00:00:00Z",
    created_at: "2026-01-01T00:00:00Z", updated_at: "2026-08-09T09:00:00Z", version: 5,
  },
  requirements: [], applicability: [], control_objectives: [], control_implementations: [], requirement_control_links: [], evidence_contracts: [], evidence_assessments: [], triggers: [],
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("governed operating mutations", () => {
  it("uses the actor Workflow packet as the source of Matter outcomes and submits the current aggregate version", async () => {
    vi.mocked(loadActorMatterWork).mockResolvedValue([decisionTask]);
    vi.mocked(recordMatterDecision).mockResolvedValue({
      ...matter,
      matter: { ...matter.matter, version: 8 },
      decisions: [...matter.decisions, { id: "decision-stage-2", type: "TREATMENT", status: "APPROVED", selected_option: "Remediate", rationale: "Approved treatment." }],
    });
    const onUpdated = vi.fn();

    render(<MatterWorkCommandPanel aggregate={matter} onUpdated={onUpdated}/>);

    expect(await screen.findByRole("heading", { name: "Decide" })).toBeTruthy();
    const outcome = screen.getByLabelText("Outcome") as HTMLSelectElement;
    expect([...outcome.options].map((option) => option.value)).toEqual(["APPROVED", "REJECTED"]);

    fireEvent.change(screen.getByLabelText("Selected option"), { target: { value: "Remediate" } });
    fireEvent.change(screen.getByLabelText("Rationale"), { target: { value: "Approved treatment." } });
    fireEvent.click(screen.getByRole("button", { name: "Decide" }));

    await waitFor(() => expect(recordMatterDecision).toHaveBeenCalledWith("matter-1", 7, expect.objectContaining({
      type: "TREATMENT", status: "APPROVED", selectedOption: "Remediate", rationale: "Approved treatment.",
    })));
    expect(onUpdated).toHaveBeenCalledWith(expect.objectContaining({ matter: expect.objectContaining({ version: 8 }) }));
  });

  it("does not invent Matter action controls when no current actor packet exists", async () => {
    vi.mocked(loadActorMatterWork).mockResolvedValue([]);
    render(<MatterWorkCommandPanel aggregate={matter} onUpdated={vi.fn()}/>);
    await waitFor(() => expect(loadActorMatterWork).toHaveBeenCalled());
    expect(screen.queryByText("Your governed action")).toBeNull();
  });

  it("keeps Program status read-only when current authority cannot be resolved to the actor", async () => {
    vi.mocked(canCurrentActorTransitionProgram).mockResolvedValue(false);
    render(<ProgramLifecycleControls aggregate={program} onUpdated={vi.fn()}/>);
    await waitFor(() => expect(canCurrentActorTransitionProgram).toHaveBeenCalledWith("program-1"));
    expect(screen.queryByRole("heading", { name: "Change operating status" })).toBeNull();
    expect(transitionProgram).not.toHaveBeenCalled();
  });

  it("submits an authorized Program status request with the current version and rationale", async () => {
    vi.mocked(canCurrentActorTransitionProgram).mockResolvedValue(true);
    vi.mocked(transitionProgram).mockResolvedValue({ ...program, program: { ...program.program, status: "PAUSED", version: 6 } });
    const onUpdated = vi.fn();
    render(<ProgramLifecycleControls aggregate={program} onUpdated={onUpdated}/>);

    expect(await screen.findByRole("heading", { name: "Change operating status" })).toBeTruthy();
    fireEvent.change(screen.getByLabelText("Requested status"), { target: { value: "PAUSED" } });
    fireEvent.change(screen.getByLabelText("Rationale"), { target: { value: "Pause while ownership is corrected." } });
    fireEvent.click(screen.getByRole("button", { name: "Request Paused" }));

    await waitFor(() => expect(transitionProgram).toHaveBeenCalledWith("program-1", 5, "PAUSED", "Pause while ownership is corrected."));
    expect(onUpdated).toHaveBeenCalledWith(expect.objectContaining({ program: expect.objectContaining({ status: "PAUSED", version: 6 }) }));
  });
});
