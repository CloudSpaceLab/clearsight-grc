import { describe, expect, it } from "vitest";
import type { MatterOperation } from "../matterOperationsApi";
import type { MatterAggregate } from "../types";
import { responsibilityLabel, selectMatterHandoff } from "./matterHandoff";

const aggregate = {
  type_label: "Vendor issue",
  status_label: "Initial review",
  next_action: "Confirm scope and owner",
  matter: {
    id: "matter-1", tenant_id: "bank-1", reference: "MAT-1", type: "VENDOR_RISK", status: "ASSESSMENT",
    priority: 3, title: "Verify vendor address", summary: "Confirm the registered address.", scope: {}, known_facts: {},
    missing_facts: [], contradictions: [], owner_principal_id: "owner-1", created_at: "2026-08-31T09:00:00Z",
    updated_at: "2026-08-31T09:00:00Z", version: 2,
  },
  links: [], decisions: [], actions: [], verification_contracts: [], verification_results: [], response_packages: [],
  closure: { ready: false, reasons: [] },
} satisfies MatterAggregate;

function operation(value: Partial<MatterOperation> & Pick<MatterOperation, "command" | "responsibility">): MatterOperation {
  return {
    label: value.command,
    can_act: false,
    reason: "Current route",
    ...value,
  };
}

describe("matter handoff selection", () => {
  it("selects accountable-owner work for scope confirmation instead of an unrelated actionable authorization", () => {
    const operations = [
      operation({ command: "matter.assign", responsibility: "ACCOUNTABLE_OWNER" }),
      operation({ command: "matter.transition", responsibility: "AUTHORIZER", can_act: true, label: "Authorize issue status" }),
    ];

    expect(selectMatterHandoff(aggregate, operations)).toBe(operations[0]);
  });

  it("selects the exact named action rather than another actionable action", () => {
    const withActions: MatterAggregate = {
      ...aggregate,
      next_action: "Confirm Cloudspace registered address",
      actions: [
        { id: "action-1", title: "Collect certifications", description: "Collect current certificates.", status: "OPEN" },
        { id: "action-2", title: "Confirm Cloudspace registered address", description: "Verify the address.", status: "OPEN" },
      ],
    };
    const operations = [
      operation({ command: "matter.action.transition", subresource_id: "action-1", responsibility: "PERFORMER", can_act: true }),
      operation({ command: "matter.action.transition", subresource_id: "action-2", responsibility: "PERFORMER" }),
    ];

    expect(selectMatterHandoff(withActions, operations)).toBe(operations[1]);
  });

  it("selects the exact response package for signatory work", () => {
    const withResponse: MatterAggregate = {
      ...aggregate,
      next_action: "Review and sign regulatory response",
      response_packages: [{ id: "response-1", purpose: "Regulatory response", audience: "Regulator", status: "IN_REVIEW" }],
    };
    const operations = [
      operation({ command: "matter.transition", responsibility: "AUTHORIZER", can_act: true }),
      operation({ command: "matter.response.transition", subresource_id: "response-1", responsibility: "SIGNATORY" }),
    ];

    expect(selectMatterHandoff(withResponse, operations)).toBe(operations[1]);
  });

  it("does not invent a dominant operation when several unrelated operations are present", () => {
    const unrelated = { ...aggregate, next_action: "Resolve the missing vendor information" };
    expect(selectMatterHandoff(unrelated, [
      operation({ command: "matter.transition", responsibility: "AUTHORIZER", can_act: true }),
      operation({ command: "matter.response.add", responsibility: "PROPOSER", can_act: true }),
    ])).toBeUndefined();
  });

  it.each([
    ["ACCOUNTABLE_OWNER", "Accountable owner"],
    ["PERFORMER", "Assigned performer"],
    ["REVIEWER", "Reviewer"],
    ["AUTHORIZER", "Authorizer"],
    ["SIGNATORY", "Signatory"],
    ["TRANSMITTER", "Transmitter"],
    ["ACKNOWLEDGEMENT_RECORDER", "Acknowledgement recorder"],
  ])("presents %s as %s", (responsibility, label) => {
    expect(responsibilityLabel(responsibility)).toBe(label);
  });
});
