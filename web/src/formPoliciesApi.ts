import { requestJSON } from "./http";
import type { FormConcernBand } from "./monitoringTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type FormPolicyStatus = "DRAFT" | "PENDING_APPROVAL" | "APPROVED" | "ACTIVE" | "SUSPENDED" | "RETIRED";
export type FormPolicyRollout = "SHADOW" | "ENFORCE";
export type FormPolicyEligibility = { form_template_id: string; form_template_version: number; subject_types: string[]; current_only: boolean; minimum_coverage: number; bands?: FormConcernBand[]; raw_below?: number; raw_above?: number; adverse_at_least?: number };
export type FormPolicyMatterType = "RISK_SITUATION" | "CONTROL_GAP" | "AUDIT_FINDING" | "EXCEPTION" | "VENDOR_REVIEW" | "VENDOR_DEFICIENCY" | "FAILED_VERIFICATION" | "EVIDENCE_CONTRADICTION" | "KRI_BREACH";
export type FormPolicyAction = { type: FormPolicyMatterType; priority: number; title_template: string; summary_template: string; requested_handling: string };
export type FormPolicyBlastRadius = { per_run: number; per_day: number };
export type FormPolicyOutcome = { expected_outcome: string; check_after_minutes: number; failure_response: "ESCALATE" | "REOPEN" | "CREATE_MATTER" | "BLOCK_CLOSE" };
export type CreateFormResponsePolicyInput = {
  code: string; name: string; purpose: string; automation_policy_id: string; automation_policy_version: number;
  eligibility: FormPolicyEligibility; action: FormPolicyAction; blast_radius: FormPolicyBlastRadius;
  outcome_contract: FormPolicyOutcome; rollout: FormPolicyRollout; effective_from?: string; effective_until?: string;
};
export type FormResponsePolicy = CreateFormResponsePolicyInput & {
  id: string; tenant_id?: string; legal_entity_id?: string; action_class: "FORM_RESPONSE_CREATE_MATTER";
  status: FormPolicyStatus; maker_id: string; checker_id?: string; checksum: string; approved_simulation_id?: string;
  supersedes_policy_id?: string; rollback_of_policy_id?: string; submitted_at?: string; approved_at?: string;
  activated_at?: string; suspended_at?: string; retired_at?: string; version: number; record_version: number;
  created_at: string; updated_at: string;
};
export type FormPolicySimulation = {
  id: string; policy_id: string; policy_version: number; policy_checksum?: string; population_count: number;
  eligible_count: number; would_create_count: number; would_reuse_count: number; blast_suppressed_count: number;
  restricted_excluded_count: number; population_high_water?: string; population_checksum?: string; impact_checksum?: string;
  observed_at: string; expires_at: string;
};
export type FormScorePreviewAnswer = { text?: string; values?: string[]; artifact_ids?: string[]; document?: Record<string, string> };
export type FormScorePreview = { raw_score?: number; adverse_score?: number; coverage: number; final: boolean; band: FormConcernBand; disqualified?: boolean; contribution_results: unknown[]; rule_results: unknown[] };

export async function listFormResponsePolicies(): Promise<FormResponsePolicy[]> {
  return (await requestJSON<{ items: FormResponsePolicy[] }>(apiBase, "/api/v1/config/form-response-policies?limit=100")).items ?? [];
}
export function createFormResponsePolicy(input: CreateFormResponsePolicyInput) {
  return requestJSON<FormResponsePolicy>(apiBase, "/api/v1/config/form-response-policies", { method: "POST", body: JSON.stringify(input) });
}
export function simulateFormResponsePolicy(id: string, expectedVersion: number) {
  return requestJSON<FormPolicySimulation>(apiBase, actionPath(id, "simulate"), command(expectedVersion));
}
export function submitFormResponsePolicy(id: string, expectedVersion: number, simulationID: string) {
  return requestJSON<FormResponsePolicy>(apiBase, actionPath(id, "submit"), command(expectedVersion, { simulation_id: simulationID }));
}
export function approveFormResponsePolicy(id: string, expectedVersion: number, simulationID: string) {
  return requestJSON<FormResponsePolicy>(apiBase, actionPath(id, "approve"), command(expectedVersion, { simulation_id: simulationID }));
}
export function activateFormResponsePolicy(id: string, expectedVersion: number) {
  return requestJSON<FormResponsePolicy>(apiBase, actionPath(id, "activate"), command(expectedVersion));
}
export function suspendFormResponsePolicy(id: string, expectedVersion: number) {
  return requestJSON<FormResponsePolicy>(apiBase, actionPath(id, "suspend"), command(expectedVersion));
}
export function rollbackFormResponsePolicy(id: string, expectedVersion: number, targetPolicyID: string) {
  return requestJSON<FormResponsePolicy>(apiBase, actionPath(id, "rollback"), command(expectedVersion, { target_policy_id: targetPolicyID }));
}
export function previewFormScore(id: string, version: number, answers: Record<string, FormScorePreviewAnswer>) {
  return requestJSON<FormScorePreview>(apiBase, `/api/v1/config/form-templates/${encodeURIComponent(id)}/score-preview`, { method: "POST", body: JSON.stringify({ form_template_version: version, answers }) });
}

function actionPath(id: string, action: string) { return `/api/v1/config/form-response-policies/${encodeURIComponent(id)}/${action}`; }
function command(expectedVersion: number, extra: Record<string, string> = {}): RequestInit { return { method: "POST", body: JSON.stringify({ expected_version: expectedVersion, ...extra }) }; }
