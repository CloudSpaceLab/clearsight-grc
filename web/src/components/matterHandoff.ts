import type { MatterOperation } from "../matterOperationsApi";
import type { MatterAggregate } from "../types";

function normalize(value: string | undefined) {
  return (value ?? "").trim().toLowerCase().replace(/[^a-z0-9]+/g, " ").trim();
}

function textMatches(nextAction: string, value: string | undefined) {
  const candidate = normalize(value);
  return candidate.length >= 4 && (nextAction.includes(candidate) || candidate.includes(nextAction));
}

function operationFor(operations: MatterOperation[], command: string, subresourceID?: string) {
  return operations.find((operation) => operation.command === command
    && (subresourceID === undefined || operation.subresource_id === subresourceID));
}

/**
 * Selects the operation that implements the aggregate's recorded next work.
 * Authority only controls whether that operation can be performed; it must not
 * change which responsibility is presented as the current handoff.
 */
export function selectMatterHandoff(aggregate: MatterAggregate, operations: MatterOperation[]) {
  const nextAction = normalize(aggregate.next_action);

  const action = aggregate.actions.find((candidate) => textMatches(nextAction, candidate.title));
  if (action) {
    const selected = operationFor(operations, "matter.action.transition", action.id)
      ?? operationFor(operations, "matter.action.update", action.id)
      ?? operationFor(operations, "matter.action.assign", action.id);
    if (selected) return selected;
  }

  const response = aggregate.response_packages.find((candidate) => textMatches(nextAction, candidate.purpose));
  if (response) {
    const selected = operationFor(operations, "matter.response.transition", response.id);
    if (selected) return selected;
  }

  const contract = aggregate.verification_contracts.find((candidate) => textMatches(nextAction, candidate.expected_outcome));
  if (contract) {
    const selected = operationFor(operations, "matter.outcome.record", contract.id);
    if (selected) return selected;
  }

  const preferences: string[] = [];
  if (nextAction.includes("scope") && nextAction.includes("owner")) {
    preferences.push("matter.assign", "matter.details.update");
  } else if (nextAction.includes("sign") || nextAction.includes("transmi") || nextAction.includes("acknowledg") || nextAction.includes("response")) {
    preferences.push("matter.response.transition", "matter.response.add");
  } else if (nextAction.includes("decision") || nextAction.includes("decide")) {
    preferences.push("matter.decision.record");
  } else if (nextAction.includes("outcome") || nextAction.includes("verification") || nextAction.includes("verify")) {
    preferences.push("matter.outcome.record", "matter.outcome.define");
  }

  for (const command of preferences) {
    const selected = operationFor(operations, command);
    if (selected) return selected;
  }

  return operations.length === 1 ? operations[0] : undefined;
}

export function responsibilityLabel(value: string | undefined) {
  switch (normalize(value).replaceAll(" ", "_")) {
    case "accountable_owner": return "Accountable owner";
    case "performer": return "Assigned performer";
    case "reviewer": return "Reviewer";
    case "authorizer": return "Authorizer";
    case "signatory": return "Signatory";
    case "transmitter": return "Transmitter";
    case "acknowledgement_recorder": return "Acknowledgement recorder";
    case "independent_challenger": return "Independent challenger";
    case "proposer": return "Proposer";
    default: return "Responsible person";
  }
}

