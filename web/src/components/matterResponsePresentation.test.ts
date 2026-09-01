import { describe, expect, it } from "vitest";
import type { MatterOperation } from "../matterOperationsApi";
import { matterResponsePresentation, matterStatusPresentation } from "./matterResponsePresentation";

function operation(responsibility: string): MatterOperation {
  return { command: "matter.response.transition", label: "Update response status", responsibility, can_act: true, reason: "Current route" };
}

describe("matterResponsePresentation", () => {
  it.each([
    ["REVIEWER", "Review response", "Record review"],
    ["SIGNATORY", "Review and sign response", "Record sign-off"],
    ["TRANSMITTER", "Record transmission", "Record transmission"],
    ["ACKNOWLEDGEMENT_RECORDER", "Record acknowledgement", "Record acknowledgement"],
  ])("keeps %s work distinct", (responsibility, action, submit) => {
    expect(matterResponsePresentation(operation(responsibility))).toMatchObject({ action, submit });
  });
});

describe("matterStatusPresentation", () => {
  it("keeps authorization separate from ordinary lifecycle handling", () => {
    expect(matterStatusPresentation({ ...operation("AUTHORIZER"), command: "matter.transition" })).toEqual({
      action: "Authorize issue status",
      sheet: "Authorize issue status",
      submit: "Record authorization",
      rationaleLabel: "Authorization basis",
      consequence: "Records the authorized issue state and basis. It does not complete assigned work or confirm an outcome that has not been independently checked.",
    });
    expect(matterStatusPresentation({ ...operation("ACCOUNTABLE_OWNER"), command: "matter.transition", label: "Change issue status" }).submit).toBe("Confirm issue status");
  });
});
