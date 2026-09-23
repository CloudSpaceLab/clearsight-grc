import type { ResponseFieldAssessment } from "../../formAssessmentApi";

export type SemanticRequirement = {
  id: string;
  label: string;
  response?: ResponseFieldAssessment;
  evidence?: ResponseFieldAssessment;
  answered: boolean;
  evidenceReceived: boolean;
  reviewState: "REVIEWED" | "AWAITING_REVIEW" | "NOT_REQUIRED";
};

export type SemanticServiceGroup = { id: string; name: string; requirements: SemanticRequirement[] };
export type SemanticResponseReview = {
  metrics: { requirements: number; answered: number; evidenceReceived: number; awaitingReview: number };
  services: SemanticServiceGroup[];
};

const semanticID = /^requirement_(\d+)_(response|evidence)$/;

export function buildSemanticResponseReview(fields: ResponseFieldAssessment[]): SemanticResponseReview | undefined {
  const requirements = new Map<string, SemanticRequirement & { serviceID: string; serviceName: string }>();
  for (const item of fields) {
    const match = semanticID.exec(item.field.id);
    if (!match) continue;
    const id = match[1]!;
    const current: SemanticRequirement & { serviceID: string; serviceName: string } = requirements.get(id) ?? { id, label: item.field.label, answered: false, evidenceReceived: false, reviewState: "NOT_REQUIRED", serviceID: item.field.section_id || "service", serviceName: serviceName(item.field.description) };
    if (match[2] === "response") {
      current.response = item;
      current.label = item.field.label;
      current.answered = hasAnswer(item);
      current.reviewState = item.decision ? "REVIEWED" : item.field.assessment?.mode === "MANUAL" || item.field.assessment?.mode === "AUTOMATIC_REVIEW" ? "AWAITING_REVIEW" : "NOT_REQUIRED";
    } else {
      current.evidence = item;
      current.evidenceReceived = hasEvidence(item);
    }
    requirements.set(id, current);
  }
  if (requirements.size === 0) return undefined;
  const services = new Map<string, SemanticServiceGroup>();
  for (const item of requirements.values()) {
    const service = services.get(item.serviceID) ?? { id: item.serviceID, name: item.serviceName, requirements: [] };
    service.requirements.push(item);
    services.set(item.serviceID, service);
  }
  const values = [...requirements.values()];
  return {
    metrics: { requirements: values.length, answered: values.filter((item) => item.answered).length, evidenceReceived: values.filter((item) => item.evidenceReceived).length, awaitingReview: values.filter((item) => item.reviewState === "AWAITING_REVIEW").length },
    services: [...services.values()],
  };
}

function serviceName(description?: string) {
  const marker = "Service:";
  const value = description?.split(marker)[1]?.trim();
  return value || "Vendor service";
}

function hasAnswer(item: ResponseFieldAssessment) {
  return Boolean(item.answer?.text?.trim() || item.answer?.values?.length || item.answer?.artifact_ids?.length || item.answer?.document?.artifact_id);
}

function hasEvidence(item: ResponseFieldAssessment) {
  return Boolean(item.answer?.artifact_ids?.length || item.answer?.document?.artifact_id);
}
