import { loadContext } from "./api";
import { requestJSON } from "./http";
import type { EvidenceRequest } from "./types";
import type { CreateFormTemplateInput, FormTemplate, LifecycleStatus, MonitoringCheck, MonitoringResult } from "./monitoringTypes";
import type { SourceBinding } from "./sourceConfigApi";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

async function scoped<T>(path: string, init?: RequestInit): Promise<T> {
  const context = await loadContext();
  const separator = path.includes("?") ? "&" : "?";
  return requestJSON<T>(apiBase, `${path}${separator}tenant_id=${encodeURIComponent(context.tenant.id)}`, init);
}

export async function loadFormTemplates(programID?: string): Promise<FormTemplate[]> {
	const path = programID
		? `/api/v1/programs/${encodeURIComponent(programID)}/form-templates?limit=100`
		: "/api/v1/form-templates?limit=100";
	return (await scoped<{ items: FormTemplate[] }>(path)).items;
}

export function createFormTemplate(programID: string, input: CreateFormTemplateInput): Promise<FormTemplate> {
  return scoped<FormTemplate>(`/api/v1/programs/${encodeURIComponent(programID)}/form-templates`, { method: "POST", body: JSON.stringify(input) });
}

export function transitionFormTemplate(programID: string, id: string, expectedVersion: number, to: LifecycleStatus): Promise<FormTemplate> {
  return scoped<FormTemplate>(`/api/v1/programs/${encodeURIComponent(programID)}/form-templates/${encodeURIComponent(id)}/transition`, { method: "POST", body: JSON.stringify({ expected_version: expectedVersion, to }) });
}

export async function loadMonitoringChecks(programID: string): Promise<MonitoringCheck[]> {
  return (await scoped<{ items: MonitoringCheck[] }>(`/api/v1/programs/${encodeURIComponent(programID)}/monitoring-checks?limit=100`)).items;
}

export function createFormMonitoringCheck(programID: string, form: FormTemplate): Promise<MonitoringCheck> {
  return scoped<MonitoringCheck>(`/api/v1/programs/${encodeURIComponent(programID)}/monitoring-checks`, {
    method: "POST",
    body: JSON.stringify({
      code: `${form.code}-CHECK`, name: form.name, claim: form.purpose, input_kind: "FORM",
      form_template_id: form.id, form_template_version: form.version,
      thresholds: { moderate_from: 25, high_from: 50, critical_from: 75 },
      freshness_minutes: 10080, minimum_coverage: 1, failure_action: "REVIEW",
    }),
  });
}

export function createSourceMonitoringCheck(programID: string, binding: SourceBinding, input: { code: string; name: string; claim: string; field: string; expected: string }): Promise<MonitoringCheck> {
  return scoped<MonitoringCheck>(`/api/v1/programs/${encodeURIComponent(programID)}/monitoring-checks`, {
    method: "POST",
    body: JSON.stringify({
      code: input.code, name: input.name, claim: input.claim, input_kind: "SOURCE",
      binding_id: binding.binding_id, binding_version: binding.version,
      source_rules: [{ id: `${input.code}-RULE`, field: input.field, operator: "EQUALS", expected: input.expected, risk_points: 100, critical: true }],
      thresholds: { moderate_from: 25, high_from: 50, critical_from: 75 },
      freshness_minutes: 60, minimum_coverage: 1, failure_action: "RECOMMEND_MATTER",
    }),
  });
}

export function transitionMonitoringCheck(id: string, expectedVersion: number, to: LifecycleStatus): Promise<MonitoringCheck> {
  return scoped<MonitoringCheck>(`/api/v1/monitoring-checks/${encodeURIComponent(id)}/transition`, { method: "POST", body: JSON.stringify({ expected_version: expectedVersion, to }) });
}

export function startFormCollection(form: FormTemplate, input: { programID: string; periodStart: string; periodEnd: string; deadline: string }): Promise<EvidenceRequest> {
  return scoped<EvidenceRequest>(`/api/v1/programs/${encodeURIComponent(input.programID)}/form-templates/${encodeURIComponent(form.id)}/collections`, {
    method: "POST",
    body: JSON.stringify({
      form_template_version: form.version,
      period_start: input.periodStart, period_end: input.periodEnd, deadline: input.deadline,
    }),
  });
}

export async function loadMonitoringResults(checkID: string): Promise<MonitoringResult[]> {
  return (await scoped<{ items: MonitoringResult[] }>(`/api/v1/monitoring-checks/${encodeURIComponent(checkID)}/results?limit=20`)).items;
}

export function evaluateMonitoringSource(check: MonitoringCheck): Promise<MonitoringResult> {
  return scoped<MonitoringResult>(`/api/v1/monitoring-checks/${encodeURIComponent(check.id)}/evaluate-source`, { method: "POST", body: JSON.stringify({ check_version: check.version }) });
}

export function createMonitoringLinkedIssue(resultID: string): Promise<{ matter: { id: string; reference: string }; created: boolean }> {
  return scoped<{ matter: { id: string; reference: string }; created: boolean }>(`/api/v1/monitoring-results/${encodeURIComponent(resultID)}/linked-issue`, { method: "POST", body: JSON.stringify({}) });
}
