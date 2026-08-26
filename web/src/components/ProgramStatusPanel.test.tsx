import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { ProgramAggregate } from "../types";
import { ProgramStatusPanel } from "./ProgramStatusPanel";

const aggregate: ProgramAggregate = {
  state_label: "Evidence incomplete",
  program: { id: "program-1", tenant_id: "bank", code: "NDPA", name: "Privacy", type: "REGULATORY", status: "ACTIVE", owning_function: "Compliance", scope: {}, effective_from: "2026-01-01T00:00:00Z", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z", version: 4 },
  requirements: [], applicability: [], control_objectives: [], control_implementations: [], requirement_control_links: [], evidence_contracts: [], evidence_assessments: [], triggers: [],
};

it("selects the first permitted status when authority arrives after the record", () => {
  const view = render(<ProgramStatusPanel aggregate={aggregate} operations={[]} onUpdated={vi.fn()} onReload={vi.fn()}/>);

  view.rerender(<ProgramStatusPanel aggregate={aggregate} operations={[{
    command: "program.transition",
    label: "Change Program status",
    responsibility: "AUTHORIZER",
    can_act: true,
    reason: "You hold the current responsibility.",
    allowed_targets: ["PAUSED", "RETIRED"],
  }]} onUpdated={vi.fn()} onReload={vi.fn()}/>);

  expect((screen.getByLabelText("New operating status") as HTMLSelectElement).value).toBe("PAUSED");
  fireEvent.change(screen.getByLabelText("Reason for status change"), { target: { value: "Pause while evidence ownership is corrected." } });
  expect((screen.getByRole("button", { name: "Pause Program" }) as HTMLButtonElement).disabled).toBe(false);
});
