import { loadContext } from "../api";
import { requestJSON } from "../http";

export type VendorAction = "start" | "send" | "review" | "complete" | "document" | "activate" | "setup" | "reissue" | "clarification";
const actions: Record<VendorAction, { object_types: string[]; responsibility: string; decision_type: string; materiality?: number }> = {
  start: { object_types: ["VENDOR_RELATIONSHIP", "THIRD_PARTY_RELATIONSHIP"], responsibility: "ACCOUNTABLE_OWNER", decision_type: "thirdparty.assessment.start" },
  send: { object_types: ["THIRD_PARTY_ASSESSMENT"], responsibility: "ACCOUNTABLE_OWNER", decision_type: "thirdparty.assessment.send_request" },
  review: { object_types: ["THIRD_PARTY_ASSESSMENT"], responsibility: "REVIEWER", decision_type: "thirdparty.assessment.review" },
  complete: { object_types: ["THIRD_PARTY_ASSESSMENT"], responsibility: "REVIEWER", decision_type: "thirdparty.assessment.complete" },
  document: { object_types: ["THIRD_PARTY_ASSESSMENT"], responsibility: "REVIEWER", decision_type: "thirdparty.assessment.document.review" },
  activate: { object_types: ["VENDOR_RELATIONSHIP"], responsibility: "AUTHORIZER", decision_type: "thirdparty.relationship.activate", materiality: 4 },
  setup: { object_types: ["THIRD_PARTY_ASSESSMENT"], responsibility: "ACCOUNTABLE_OWNER", decision_type: "thirdparty.assessment.setup.retry" },
  reissue: { object_types: ["THIRD_PARTY_ASSESSMENT"], responsibility: "ACCOUNTABLE_OWNER", decision_type: "thirdparty.assessment.reissue_request" },
  clarification: { object_types: ["THIRD_PARTY_ASSESSMENT"], responsibility: "REVIEWER", decision_type: "thirdparty.assessment.clarification.request" },
};

export async function checkVendorAuthority(action: VendorAction, objectID: string): Promise<boolean> {
  const context = await loadContext();
  if (!context.actor.id || context.actor.kind?.toUpperCase() === "SERVICE" || !context.legal_entity.id || context.legal_entity.id === "*") return false;
  const spec = actions[action];
  const routes = await Promise.all(spec.object_types.map((object_type) => requestJSON<{ principal: { id: string }; candidate_principals?: { id: string }[] }>(import.meta.env.VITE_API_BASE_URL ?? "", "/api/v1/authority/resolve", { method: "POST", body: JSON.stringify({ object_type, responsibility: spec.responsibility, decision_type: spec.decision_type, object_id: objectID, tenant_id: context.tenant.id, legal_entity_id: context.legal_entity.id, materiality: spec.materiality ?? 3 }) })));
  return routes.every((route) => route.principal?.id === context.actor.id || Boolean(route.candidate_principals?.some((person) => person.id === context.actor.id)));
}
