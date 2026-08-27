import { describe, expect, it } from "vitest";
import { readWorkspaceFilters, workspaceHash } from "./workspaceFilters";

describe("workspace filters", () => {
  it("round-trips compact filter values without treating record paths as filters", () => {
    const hash = workspaceHash("#work/matters/matter-1", { q: "annual return", status: "OPEN", priority: 4, assigned_to_me: true, due: "" });
    expect(hash).toBe("#work/matters/matter-1?q=annual+return&status=OPEN&priority=4&assigned_to_me=true");
    expect(readWorkspaceFilters(hash)).toEqual({ q: "annual return", status: "OPEN", priority: "4", assigned_to_me: "true" });
  });
});
