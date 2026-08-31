import type { MatterOperation } from "../matterOperationsApi";
import type { MatterAggregate } from "../types";

function normalize(value: string | undefined) {
  return (value ?? "").trim().toLowerCase().replace(/[^a-z0-9]+/g, " ").trim();
}

function textMatches(nextAction: string, value: string | undefined) {
  const candidate = normalize(value);
  return candidate.length >= 4 && (nextAction.includes(candidate) || candidate.includes(nextAction));
}

function mentions(nextAction: string, concepts: string[]) {
  const words = new Set(nextAction.split(" "));
  return concepts.some((concept) => words.has(concept));
}

function operationFor(operations: MatterOperation[], command: string, subresourceID?: string) {
  return operations.find((operation) => operation.command === command
    && (subresourceID === undefined || operation.subresource_id === subresourceID));
}

function uniqueOperation(operations: MatterOperation[], command: string, subresourceIDs?: string[]) {
  const matches = operations.filter((operation) => operation.command === command
    && (subresourceIDs === undefined || (operation.subresource_id !== undefined && subresourceIDs.includes(operation.subresource_id))));
  return matches.length === 1 ? matches[0] : undefined;
}

export function matterOperationControlID(operation: Pick<MatterOperation, "command" | "subresource_id" | "responsibility">) {
  return `matter-control-${operation.command}-${operation.subresource_id ?? "record"}-${operation.responsibility}`;
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
  if (mentions(nextAction, ["scope"]) && mentions(nextAction, ["owner"])) {
    preferences.push("matter.assign", "matter.details.update");
  } else if (mentions(nextAction, ["sign", "signed", "signing", "signature", "transmit", "transmission", "acknowledge", "acknowledgement", "response"])) {
    preferences.push("matter.response.transition", "matter.response.add");
  } else if (mentions(nextAction, ["decision", "decide"])) {
    preferences.push("matter.decision.record");
  } else if (mentions(nextAction, ["outcome", "verification", "verify", "verified"])) {
    preferences.push("matter.outcome.record", "matter.outcome.define");
  }

  if (preferences.length === 0) {
    switch (aggregate.matter.status) {
      case "DRAFT":
        return uniqueOperation(operations, "matter.transition");
      case "INITIAL_REVIEW":
        return uniqueOperation(operations, "matter.assign") ?? uniqueOperation(operations, "matter.details.update");
      case "ASSESSMENT":
      case "DECISION_REQUIRED": {
        const initialDecision = operations.filter((operation) => operation.command === "matter.decision.record" && !operation.subresource_id);
        return initialDecision.length === 1 ? initialDecision[0] : uniqueOperation(operations, "matter.decision.record");
      }
      case "ACTION_IN_PROGRESS": {
        const activeActionIDs = aggregate.actions.filter((candidate) => !["IMPLEMENTED", "CANCELLED"].includes(candidate.status)).map((candidate) => candidate.id);
        return uniqueOperation(operations, "matter.action.transition", activeActionIDs);
      }
      case "RESPONSE_PREPARATION": {
        const responseIDs = aggregate.response_packages.map((candidate) => candidate.id);
        return uniqueOperation(operations, "matter.response.transition", responseIDs) ?? uniqueOperation(operations, "matter.response.add");
      }
      case "VERIFICATION": {
        const activeContractIDs = aggregate.verification_contracts.filter((candidate) => candidate.status === "ACTIVE").map((candidate) => candidate.id);
        return uniqueOperation(operations, "matter.outcome.record", activeContractIDs);
      }
    }
  }

  for (const command of preferences) {
    const selected = uniqueOperation(operations, command);
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
