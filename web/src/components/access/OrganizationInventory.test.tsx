import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { LegalEntityDataBoundary, LegalEntityDataBoundaryRevision } from "../../identityAccessApi";
import { OrganizationInventory } from "./OrganizationInventory";

function boundary(overrides: Partial<LegalEntityDataBoundary> = {}): LegalEntityDataBoundary {
  return {
    legal_entity_id: "entity-a",
    legal_entity_code: "ENTITY-A",
    legal_entity_name: "Entity A",
    jurisdiction: "NG",
    residency_region: "",
    detail_transfer_mode: "AGGREGATE_ONLY",
    allowed_destination_regions: [],
    version: 0,
    configured: false,
    ...overrides,
  };
}

function pending(overrides: Partial<LegalEntityDataBoundaryRevision> = {}): LegalEntityDataBoundaryRevision {
  return {
    id: "revision-1",
    legal_entity_id: "entity-a",
    base_version: 0,
    proposed_residency_region: "NG",
    proposed_detail_transfer_mode: "AGGREGATE_ONLY",
    proposed_destination_regions: [],
    maker_id: "maker-1",
    status: "PENDING",
    created_at: "2026-10-04T05:00:00Z",
    ...overrides,
  };
}

describe("OrganizationInventory data boundary", () => {
  it("keeps an unconfigured legal entity aggregate-only by default and proposes through maker-checker", async () => {
    const propose = vi.fn().mockResolvedValue(true);
    render(<OrganizationInventory
      positions={[]}
      people={[]}
      scopes={[]}
      mode="positions"
      dataBoundary={boundary()}
      actorPrincipalID="maker-1"
      canConfigure
      onProposeDataBoundary={propose}
    />);

    expect(screen.getByText("Aggregate only", { selector: ".cs-status-badge" })).toBeTruthy();
    expect(screen.getByText("Cross-entity record detail is blocked. Group totals remain aggregate-only.")).toBeTruthy();

    fireEvent.change(screen.getByLabelText("Residency region"), { target: { value: "NG" } });
    fireEvent.click(screen.getByRole("button", { name: "Propose change" }));

    await waitFor(() => expect(propose).toHaveBeenCalledWith({
      residency_region: "NG",
      detail_transfer_mode: "AGGREGATE_ONLY",
      allowed_destination_regions: [],
      expected_version: 0,
    }));
  });

  it("requires another administrator to approve a pending boundary", async () => {
    const approve = vi.fn().mockResolvedValue(true);
    const reject = vi.fn().mockResolvedValue(true);
    render(<OrganizationInventory
      positions={[]}
      people={[]}
      scopes={[]}
      mode="positions"
      dataBoundary={boundary({ configured: true, residency_region: "NG", version: 2 })}
      dataBoundaryRevisions={[pending({ base_version: 2, maker_id: "maker-other", proposed_residency_region: "NG-PRIMARY" })]}
      actorPrincipalID="checker-1"
      canConfigure
      onApproveDataBoundary={approve}
      onRejectDataBoundary={reject}
    />);

    expect(screen.getByText("Proposed · NG-PRIMARY")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Propose change" })).toBeNull();

    fireEvent.change(screen.getByLabelText("Decision rationale"), { target: { value: "Residency reviewed" } });
    fireEvent.click(screen.getByRole("button", { name: "Approve" }));

    await waitFor(() => expect(approve).toHaveBeenCalledWith(
      expect.objectContaining({ id: "revision-1", maker_id: "maker-other" }),
      "Residency reviewed",
    ));
  });

  it("does not show approval controls to the maker of the pending boundary", () => {
    render(<OrganizationInventory
      positions={[]}
      people={[]}
      scopes={[]}
      mode="positions"
      dataBoundary={boundary()}
      dataBoundaryRevisions={[pending({ maker_id: "maker-1" })]}
      actorPrincipalID="maker-1"
      canConfigure
    />);

    expect(screen.getByText("Awaiting independent approval")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Approve" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Reject" })).toBeNull();
  });
});
