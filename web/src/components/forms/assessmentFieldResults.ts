import type { ResponseAssessmentDetail, ResponseFieldAssessment } from "../../formAssessmentApi";
import type { FormScorePredicate } from "../../monitoringTypes";

export type AutomaticFieldResult = { id: string; label: string; description: string; concern?: number; shared: boolean };
export function predicateFieldIDs(predicate: FormScorePredicate): string[] {
  return [...new Set([...(predicate.field_id ? [predicate.field_id] : []), ...(predicate.children ?? []).flatMap(predicateFieldIDs)])];
}
export function assessmentConcernThreshold(detail: ResponseAssessmentDetail) {
  const bands = detail.score_profile?.bands.filter((band) => band.band === "HIGH" || band.band === "CRITICAL");
  return bands?.length ? Math.min(...bands.map((band) => band.from)) : 50;
}
export function assessmentConcernPoints(points: number, detail: ResponseAssessmentDetail) {
  const direction = detail.assessed_score?.direction ?? detail.automatic_score?.direction ?? detail.score_profile?.direction;
  return direction === "LOW_IS_POOR" ? 100 - points : points;
}
export function automaticFieldResults(item: ResponseFieldAssessment, detail: ResponseAssessmentDetail): AutomaticFieldResult[] {
  const score = detail.automatic_score, profile = detail.score_profile;
  const results: AutomaticFieldResult[] = [];
  for (const contribution of score?.contribution_results ?? []) {
    const definition = profile?.contributions.find((value) => value.id === contribution.id);
    const fields = definition ? predicateFieldIDs(definition.predicate) : [];
    const legacy = !definition && (contribution.id === item.field.id || contribution.id === item.field.scoring?.id);
    if (!legacy && !fields.includes(item.field.id)) continue;
    const incomplete = contribution.outcome === "INDETERMINATE";
    const concern = incomplete ? undefined : assessmentConcernPoints(contribution.points, detail);
    results.push({ id: contribution.id, label: definition?.label ?? "Answer scoring rule", description: incomplete ? "Required scoring input is incomplete." : `${contribution.points} ${score?.mode === "COMPLIANCE" ? "compliance" : "risk"} points · ${concern} concern points · Weight ${contribution.weight}`, concern, shared: fields.length > 1 });
  }
  for (const rule of score?.rule_results ?? []) {
    const definition = profile?.rules?.find((value) => value.id === rule.id);
    if (!definition) continue;
    const fields = predicateFieldIDs(definition.predicate);
    if (!fields.includes(item.field.id)) continue;
    const effect = rule.effect === "FLOOR" ? `Minimum concern ${rule.value ?? definition.effect.value}` : rule.effect === "CAP" ? `Maximum concern ${rule.value ?? definition.effect.value}` : rule.effect === "DISQUALIFY" ? "Critical override" : `${rule.value ?? definition.effect.value} contribution points`;
    const value = rule.value ?? definition.effect.value;
    const concern = rule.matched ? rule.effect === "DISQUALIFY" ? 100 : rule.effect === "FLOOR" ? value : rule.effect === "CONTRIBUTION" && value !== undefined ? assessmentConcernPoints(value, detail) : undefined : undefined;
    results.push({ id: `rule:${rule.id}`, label: definition.label, description: rule.outcome === "INDETERMINATE" ? "Rule inputs are incomplete." : `${rule.matched ? "Condition met" : "Condition not met"} · ${effect}`, concern, shared: fields.length > 1 });
  }
  return results;
}
