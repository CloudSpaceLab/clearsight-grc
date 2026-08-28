import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type CommunicationAction = "INVITATION" | "REMINDER" | "DUE_SOON" | "EXPIRED" | "CHANGE_REQUESTED" | "AMENDMENT" | "COMPLETION";
export type CommunicationStatus = "DRAFT" | "PENDING_APPROVAL" | "ACTIVE" | "RETIRED";
export type CommunicationNode = { type: string; text?: string; href?: string; level?: number; items?: string[] };
export type CommunicationProfile = {
  id: string; legal_entity_id: string; version: number; default_locale: string; bank_name: string;
  support_contact?: string; brand_asset_id?: string; status: CommunicationStatus; effective_from: string;
  effective_until?: string; maker_id: string; checker_id?: string; rollback_origin_version?: number; created_at: string; updated_at: string;
};
export type CommunicationTemplate = {
  id: string; legal_entity_id: string; action: CommunicationAction; locale: string; version: number; subject_template: string;
  document: CommunicationNode[]; status: CommunicationStatus; effective_from: string; effective_until?: string;
  maker_id: string; checker_id?: string; rollback_origin_version?: number; created_at: string; updated_at: string;
};
export type CommunicationPreview = { subject: string; plain_text: string; html: string };
export type CommunicationImpact = { action: CommunicationAction; locale: string; current_version?: number; candidate_version: number; subject_changed: boolean; document_changed: boolean; effective_window_changed: boolean };

function query(values: Record<string, string | undefined>) {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(values)) if (value) params.set(key, value);
  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
}
function templatePath(template: Pick<CommunicationTemplate, "action" | "locale" | "version">) {
  return `/api/v1/forms/communications/templates/${encodeURIComponent(template.action)}/${encodeURIComponent(template.locale)}/revisions/${template.version}`;
}

export async function loadCommunicationProfiles(): Promise<CommunicationProfile[]> {
  return (await requestJSON<{ items: CommunicationProfile[] }>(apiBase, "/api/v1/forms/communications/profiles")).items ?? [];
}
export function createCommunicationProfile(input: Pick<CommunicationProfile, "default_locale" | "bank_name" | "support_contact" | "brand_asset_id" | "effective_from" | "effective_until">): Promise<CommunicationProfile> {
  return requestJSON(apiBase, "/api/v1/forms/communications/profiles", { method: "POST", body: JSON.stringify(input) });
}
export function transitionCommunicationProfile(version: number, to: CommunicationStatus): Promise<CommunicationProfile> {
  return requestJSON(apiBase, `/api/v1/forms/communications/profiles/${version}/transition`, { method: "POST", body: JSON.stringify({ expected_version: version, to }) });
}
export function rollbackCommunicationProfile(version: number): Promise<CommunicationProfile> {
  return requestJSON(apiBase, `/api/v1/forms/communications/profiles/${version}/rollback`, { method: "POST", body: "{}" });
}

export async function loadCommunicationTemplates(filters: { action?: CommunicationAction; locale?: string; status?: CommunicationStatus } = {}): Promise<CommunicationTemplate[]> {
  return (await requestJSON<{ items: CommunicationTemplate[] }>(apiBase, `/api/v1/forms/communications/templates${query(filters)}`)).items ?? [];
}
export function createCommunicationTemplate(input: Pick<CommunicationTemplate, "action" | "locale" | "subject_template" | "document" | "effective_from" | "effective_until">): Promise<CommunicationTemplate> {
  return requestJSON(apiBase, "/api/v1/forms/communications/templates", { method: "POST", body: JSON.stringify(input) });
}
export function previewCommunicationTemplate(template: Pick<CommunicationTemplate, "action" | "locale" | "version">): Promise<CommunicationPreview> {
  return requestJSON(apiBase, `${templatePath(template)}/preview`, { method: "POST", body: "{}" });
}
export function impactCommunicationTemplate(template: Pick<CommunicationTemplate, "action" | "locale" | "version">): Promise<CommunicationImpact> {
  return requestJSON(apiBase, `${templatePath(template)}/impact`, { method: "POST", body: "{}" });
}
export function testSendCommunicationTemplate(template: Pick<CommunicationTemplate, "action" | "locale" | "version">, address: string): Promise<Record<string, unknown>> {
  return requestJSON(apiBase, `${templatePath(template)}/test-send`, { method: "POST", body: JSON.stringify({ address }) });
}
export function transitionCommunicationTemplate(template: Pick<CommunicationTemplate, "action" | "locale" | "version">, to: CommunicationStatus): Promise<CommunicationTemplate> {
  return requestJSON(apiBase, `${templatePath(template)}/transition`, { method: "POST", body: JSON.stringify({ expected_version: template.version, to }) });
}
export function rollbackCommunicationTemplate(template: Pick<CommunicationTemplate, "action" | "locale" | "version">): Promise<CommunicationTemplate> {
  return requestJSON(apiBase, `${templatePath(template)}/rollback`, { method: "POST", body: "{}" });
}
