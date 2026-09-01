import { describe, expect, it } from "vitest";
import type { MatterOperation } from "../matterOperationsApi";
import type { MatterAggregate } from "../types";
import { matterOperationControlID, responsibilityLabel, selectMatterHandoff } from "./matterHandoff";

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
    ["DRAFT", "Start initial review", operation({ command: "matter.transition", responsibility: "ACCOUNTABLE_OWNER", allowed_targets: ["INITIAL_REVIEW"] }), operation({ command: "matter.transition", responsibility: "AUTHORIZER", allowed_targets: ["CANCELLED"] })],
    ["ASSESSMENT", "Review impact and options", operation({ command: "matter.context.change", responsibility: "ACCOUNTABLE_OWNER" }), operation({ command: "matter.transition", responsibility: "AUTHORIZER", allowed_targets: ["DECISION_REQUIRED", "CANCELLED"] })],
    ["DECISION_REQUIRED", "Record decision", operation({ command: "matter.decision.record", responsibility: "PROPOSER", allowed_targets: ["PROPOSED"] }), operation({ command: "matter.transition", responsibility: "ACCOUNTABLE_OWNER", allowed_targets: ["ACTION_IN_PROGRESS"] })],
    ["ACTION_IN_PROGRESS", "Complete assigned work", operation({ command: "matter.action.transition", subresource_id: "action-1", responsibility: "PERFORMER" }), operation({ command: "matter.context.change", responsibility: "ACCOUNTABLE_OWNER" })],
    ["RESPONSE_PREPARATION", "Prepare response", operation({ command: "matter.response.add", responsibility: "PROPOSER" }), operation({ command: "matter.response.transition", subresource_id: "response-1", responsibility: "REVIEWER" })],
    ["VERIFICATION", "Confirm whether the outcome was achieved", operation({ command: "matter.outcome.record", subresource_id: "contract-1", responsibility: "REVIEWER" }), operation({ command: "matter.context.change", responsibility: "ACCOUNTABLE_OWNER" })],
  ])("maps the canonical %s next action from the real API operation shape", (status, nextAction, expected, unrelated) => {
    const canonical: MatterAggregate = {
      ...aggregate,
      next_action: nextAction,
      matter: { ...aggregate.matter, status },
      actions: status === "ACTION_IN_PROGRESS" ? [{ id: "action-1", title: "Verify address", description: "Check the registered address.", status: "OPEN" }] : [],
      verification_contracts: status === "VERIFICATION" ? [{ id: "contract-1", expected_outcome: "The registered address is verified.", status: "ACTIVE", observation_period_minutes: 0 }] : [],
      response_packages: status === "RESPONSE_PREPARATION" ? [{ id: "response-1", purpose: "Existing response", audience: "Regulator", status: "DRAFT" }] : [],
    };

    expect(selectMatterHandoff(canonical, [expected, unrelated])).toBe(expected);
  });

  it("prefers one exact named record over a shorter containment match", () => {
    const exact = { ...aggregate, next_action: "Review vendor response", actions: [
      { id: "action-short", title: "Vendor response", description: "General response work.", status: "OPEN" },
      { id: "action-exact", title: "Review vendor response", description: "Review the response.", status: "OPEN" },
    ] };
    const expected = operation({ command: "matter.action.transition", subresource_id: "action-exact", responsibility: "PERFORMER" });
    expect(selectMatterHandoff(exact, [
      operation({ command: "matter.action.transition", subresource_id: "action-short", responsibility: "PERFORMER" }),
      expected,
    ])).toBe(expected);
  });

  it("keeps overlapping fuzzy record names ambiguous across work types", () => {
    const ambiguous: MatterAggregate = {
      ...aggregate,
      next_action: "Complete vendor response review",
      actions: [{ id: "action-1", title: "Vendor response", description: "Review response.", status: "OPEN" }],
      response_packages: [{ id: "response-1", purpose: "Response review", audience: "Regulator", status: "DRAFT" }],
    };
    expect(selectMatterHandoff(ambiguous, [
      operation({ command: "matter.action.transition", subresource_id: "action-1", responsibility: "PERFORMER" }),
      operation({ command: "matter.response.transition", subresource_id: "response-1", responsibility: "REVIEWER" }),
    ])).toBeUndefined();
  });

  it("does not choose between two active actions when the stored next action is generic", () => {
    const twoActions: MatterAggregate = {
      ...aggregate,
      next_action: "Complete assigned work",
      matter: { ...aggregate.matter, status: "ACTION_IN_PROGRESS" },
      actions: [
        { id: "action-1", title: "Verify address", description: "Check the registered address.", status: "OPEN" },
        { id: "action-2", title: "Collect certificate", description: "Collect current certification.", status: "IN_PROGRESS" },
      ],
    };
    expect(selectMatterHandoff(twoActions, [
      operation({ command: "matter.action.transition", subresource_id: "action-1", responsibility: "PERFORMER" }),
      operation({ command: "matter.action.transition", subresource_id: "action-2", responsibility: "PERFORMER" }),
    ])).toBeUndefined();
  });

  it("gives same-command authority routes distinct control targets", () => {
    const owner = operation({ command: "matter.transition", responsibility: "ACCOUNTABLE_OWNER" });
    const authorizer = operation({ command: "matter.transition", responsibility: "AUTHORIZER" });
    expect(matterOperationControlID(owner)).not.toBe(matterOperationControlID(authorizer));
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
