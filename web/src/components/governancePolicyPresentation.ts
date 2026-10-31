type DefinitionObject = Record<string, unknown>;

export type GovernanceRuleFacts = {
  id: string;
  responsibility: string;
  legalEntityID: string;
  objectType: string;
  objectID: string;
  decisionType: string;
  minimumMateriality: number;
  priority: number;
  selector?: { kind: string; ref: string };
  lifecycle?: { type: string; state: string; subtype: string };
};

export type GovernanceDefinitionReview = {
  reviewable: boolean;
  rules: GovernanceRuleFacts[];
  fullDefinition?: string;
  hasAdditionalSettings: boolean;
};

// This checks whether the returned rule fields can be faithfully displayed.
// The server remains responsible for policy validity and command authority.
export function readGovernancePolicyDefinition(value: unknown): GovernanceDefinitionReview {
  let fullDefinition: string | undefined;
  try { fullDefinition = value == null ? undefined : JSON.stringify(value, null, 2); } catch { /* An unreadable value must not enable approval. */ }
  const unavailable: GovernanceDefinitionReview = { reviewable: false, rules: [], fullDefinition, hasAdditionalSettings: false };
  if (!fullDefinition || !isObject(value) || !Array.isArray(value.rules) || value.rules.length === 0) return unavailable;
  const rules = value.rules.map(readRule);
  if (rules.some((rule) => !rule)) return unavailable;
  const facts = rules as GovernanceRuleFacts[];
  if (new Set(facts.map((rule) => rule.id)).size !== facts.length) return unavailable;
  return {
    reviewable: true,
    rules: facts,
    fullDefinition,
    hasAdditionalSettings: Object.keys(value).some((key) => key !== "rules") || value.rules.some((rule) => isObject(rule) && (Object.keys(rule).some((key) => !RULE_FIELDS.includes(key)) || (isObject(rule.selector) && Object.keys(rule.selector).some((key) => !["kind", "ref"].includes(key))))),
  };
}

function readRule(value: unknown): GovernanceRuleFacts | undefined {
  if (!isObject(value) || RULE_STRING_FIELDS.some((field) => value[field] != null && typeof value[field] !== "string")) return undefined;
  const id = string(value.id);
  const responsibility = string(value.responsibility).toUpperCase();
  const minimumMateriality = value.min_materiality ?? 0;
  const priority = value.priority ?? 0;
  if (!id || !RESPONSIBILITIES.includes(responsibility) || typeof minimumMateriality !== "number" || !Number.isInteger(minimumMateriality) || minimumMateriality < 0 || minimumMateriality > 5 || typeof priority !== "number" || !Number.isSafeInteger(priority)) return undefined;
  if (value.selector != null && !isObject(value.selector)) return undefined;
  const selector = isObject(value.selector) ? value.selector : {};
  if ([selector.kind, selector.ref].some((item) => item != null && typeof item !== "string")) return undefined;
  const kind = string(selector.kind).toUpperCase();
  const ref = string(selector.ref);
  const lifecycleType = string(value.lifecycle_type).toUpperCase();
  const lifecycleState = string(value.lifecycle_state);
  const lifecycle = lifecycleType || lifecycleState ? { type: lifecycleType, state: lifecycleState, subtype: string(value.lifecycle_subtype) } : undefined;
  if (lifecycle ? !["DECISION", "RESPONSE"].includes(lifecycle.type) || !lifecycle.state || kind !== "" || ref !== "" : !["PRINCIPAL", "POSITION", "ROLE"].includes(kind) || !ref) return undefined;
  return {
    id, responsibility, minimumMateriality, priority,
    legalEntityID: string(value.legal_entity_id), objectType: string(value.object_type), objectID: string(value.object_id), decisionType: string(value.decision_type),
    selector: lifecycle ? undefined : { kind, ref }, lifecycle,
  };
}

function isObject(value: unknown): value is DefinitionObject { return typeof value === "object" && value !== null && !Array.isArray(value); }
function string(value: unknown) { return typeof value === "string" ? value.trim() : ""; }

const RESPONSIBILITIES = ["PERFORMER", "ACCOUNTABLE_OWNER", "PROPOSER", "REVIEWER", "INDEPENDENT_CHALLENGER", "AUTHORIZER", "SIGNATORY", "TRANSMITTER", "ACKNOWLEDGEMENT_RECORDER", "ESCALATION_OWNER"];
const RULE_STRING_FIELDS = ["id", "legal_entity_id", "object_type", "object_id", "responsibility", "decision_type", "lifecycle_type", "lifecycle_subtype", "lifecycle_state"];
const RULE_FIELDS = [...RULE_STRING_FIELDS, "min_materiality", "priority", "selector"];
