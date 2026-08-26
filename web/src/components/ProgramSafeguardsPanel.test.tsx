import { render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { ProgramAggregate } from "../types";
import { ProgramSafeguardsPanel } from "./ProgramSafeguardsPanel";

it("shows the stored safeguard owner label without exposing its principal ID", () => {
  const aggregate = {
    program: { id: "program-1", version: 4 },
    requirements: [],
    control_objectives: [{ id: "objective-1", code: "OBJ-1", name: "Reliable filing", outcome: "Returns are filed on time.", status: "ACTIVE" }],
    control_implementations: [{ id: "safeguard-1", objective_id: "objective-1", name: "Annual return checklist", description: "Confirm every filing section.", owner_principal_id: "owner-private", status: "IMPLEMENTED" }],
    requirement_control_links: [],
  } as unknown as ProgramAggregate;
  render(<ProgramSafeguardsPanel aggregate={aggregate} operations={[]} responsibleParties={[{ scope: "SAFEGUARD", subresource_id: "safeguard-1", responsibility: "PERFORMER", display_name: "Ada Okafor", kind: "PERSON" }]} onUpdated={vi.fn()} onReload={vi.fn()}/>);

  expect(screen.getByText(/Ada Okafor · Implemented/)).toBeTruthy();
  expect(screen.queryByText(/owner-private/)).toBeNull();
});
