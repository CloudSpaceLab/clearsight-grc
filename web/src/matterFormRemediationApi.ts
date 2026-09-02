import { loadContext } from "./api";
import { continuityCommand } from "./continuityCommands";
import { requestJSON } from "./http";
import type { MatterAggregate } from "./types";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type MatterFormFieldMapping = { field_id: string; missing_item: string; fact_key: string };
export type MatterFormBinding = {
  id: string; legal_entity_id: string; program_id: string; matter_id: string; matter_version_at_binding: number;
  form_template_id: string; form_template_version: number; mappings: MatterFormFieldMapping[];
  action_id?: string; verification_contract_id: string; minimum_score?: number; maximum_adverse_score?: number;
  subject_type: "MATTER"; subject_id: string; purpose: string; audience_class: "EXTERNAL"; responder_class: string;
  status: "ACTIVE"; effective_from: string; created_at: string; version: number;
};
export type MatterFormRemediationState = {
  binding: MatterFormBinding;
  request?: { id: string; title: string; status: string; deadline: string };
  response?: { id: string; revision: number; current: boolean; state: string; completed_at: string };
  application?: { id: string; response_revision_id: string; matter_version: number; applied_at: string };
  next_action: "Send form" | "Open response" | "Review evidence" | "Request correction" | "Check outcome";
};

export async function loadMatterFormRemediations(matterID: string): Promise<MatterFormRemediationState[]> {
  const context = await loadContext();
  const query = new URLSearchParams({ tenant_id: context.tenant.id, limit: "50" });
  return (await requestJSON<{ items: MatterFormRemediationState[] }>(apiBase, `/api/v1/matters/${encodeURIComponent(matterID)}/form-remediations?${query}`)).items;
}

export function createMatterFormRemediation(matterID: string, input: {
  legalEntityID: string; expectedMatterVersion: number; programID: string; formTemplateID: string; formTemplateVersion: number;
  mappings: MatterFormFieldMapping[]; actionID?: string; verificationContractID: string; minimumScore?: number; maximumAdverseScore?: number;
}): Promise<MatterFormBinding> {
  return continuityCommand(`/api/v1/matters/${encodeURIComponent(matterID)}/form-remediations`, {
    legal_entity_id: input.legalEntityID, expected_matter_version: input.expectedMatterVersion, program_id: input.programID,
    form_template_id: input.formTemplateID, form_template_version: input.formTemplateVersion, mappings: input.mappings,
    action_id: input.actionID, verification_contract_id: input.verificationContractID,
    minimum_score: input.minimumScore, maximum_adverse_score: input.maximumAdverseScore,
  });
}

export function sendMatterFormRemediation(matterID: string, bindingID: string, input: { bindingVersion: number; email: string; deadline: string; routeExpiresAt: string }): Promise<MatterFormRemediationState> {
  return continuityCommand(`/api/v1/matters/${encodeURIComponent(matterID)}/form-remediations/${encodeURIComponent(bindingID)}/send`, {
    binding_version: input.bindingVersion,
    recipient: { role: "TO", type: "EXTERNAL_AUDIENCE", address: input.email, audience_hint: input.email.replace(/^(.{2}).*(@.*)$/, "$1…$2"), contact_label: "Issue evidence contact" },
    deadline: input.deadline, route_expires_at: input.routeExpiresAt,
  });
}

export function applyMatterFormRemediation(matterID: string, bindingID: string, input: { bindingVersion: number; expectedMatterVersion: number; responseRevisionID: string; rationale: string }): Promise<{ matter: MatterAggregate; application: MatterFormRemediationState["application"] }> {
  return continuityCommand(`/api/v1/matters/${encodeURIComponent(matterID)}/form-remediations/${encodeURIComponent(bindingID)}/apply`, {
    binding_version: input.bindingVersion, expected_matter_version: input.expectedMatterVersion,
    response_revision_id: input.responseRevisionID, rationale: input.rationale,
  });
}
