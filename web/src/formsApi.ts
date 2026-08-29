import { requestJSON, requestVoid } from "./http";
import type { LifecycleStatus } from "./monitoringTypes";
import type {
  CreateLibraryFormInput,
  FormTemplate,
  FormTemplatePage,
  FormTemplateQuery,
  InstantiateStarterTemplateInput,
  ReusableFormTemplateRef,
  SavedFormView,
  SavedFormViewFilter,
  StarterTemplate,
  FormTemplateProposal,
  RequestAIFormProposalInput,
} from "./formsTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

function formQuery(query: FormTemplateQuery = {}) {
  const params = new URLSearchParams();
  const values: Array<[string, string | number | undefined]> = [
    ["search", query.search?.trim() || undefined], ["status", query.status], ["owner", query.owner?.trim() || undefined],
    ["program", query.program?.trim() || undefined], ["use", query.use?.trim() || undefined], ["tag", query.tag?.trim() || undefined],
    ["cursor", query.cursor], ["limit", query.limit],
  ];
  for (const [key, value] of values) if (value !== undefined && value !== "") params.set(key, String(value));
  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
}

export function loadFormTemplatePage(query: FormTemplateQuery = {}): Promise<FormTemplatePage> {
  return requestJSON<FormTemplatePage>(apiBase, `/api/v1/forms/templates${formQuery(query)}`);
}
export function loadFormTemplateRevision(id: string, version: number): Promise<FormTemplate> {
  return requestJSON<FormTemplate>(apiBase, `/api/v1/forms/templates/${encodeURIComponent(id)}/revisions/${version}`);
}
export function createLibraryFormDraft(input: CreateLibraryFormInput): Promise<FormTemplate> {
  return requestJSON<FormTemplate>(apiBase, "/api/v1/forms/templates", { method: "POST", body: JSON.stringify(input) });
}
export function createLibraryFormRevision(id: string, expectedVersion: number, form: CreateLibraryFormInput): Promise<FormTemplate> {
  return requestJSON<FormTemplate>(apiBase, `/api/v1/forms/templates/${encodeURIComponent(id)}/revisions`, { method: "POST", body: JSON.stringify({ expected_version: expectedVersion, form }) });
}
export function transitionFormTemplateRevision(id: string, expectedVersion: number, to: LifecycleStatus): Promise<FormTemplate> {
  return requestJSON<FormTemplate>(apiBase, `/api/v1/forms/templates/${encodeURIComponent(id)}/transition`, { method: "POST", body: JSON.stringify({ expected_version: expectedVersion, to }) });
}
export async function loadStarterTemplates(): Promise<StarterTemplate[]> {
  return (await requestJSON<{ items: StarterTemplate[] }>(apiBase, "/api/v1/forms/starter-templates")).items;
}
export function instantiateStarterTemplate(code: string, input: InstantiateStarterTemplateInput = {}): Promise<FormTemplate> {
  return requestJSON<FormTemplate>(apiBase, `/api/v1/forms/starter-templates/${encodeURIComponent(code)}/instantiate`, { method: "POST", body: JSON.stringify(input) });
}
export async function loadReusableFormTemplateRefs(search?: string): Promise<ReusableFormTemplateRef[]> {
  const page = await loadFormTemplatePage({ search: search?.trim() || undefined, limit: 50 });
  return page.items.flatMap((item) => item.active_version && item.active_status === "ACTIVE"
    ? [{ id: item.template.id, name: item.template.name, code: item.template.code, version: item.active_version }]
    : []);
}
export async function loadSavedFormViews(): Promise<SavedFormView[]> {
  return (await requestJSON<{ items: SavedFormView[] }>(apiBase, "/api/v1/forms/saved-views")).items;
}
export function saveFormView(name: string, query: FormTemplateQuery, id?: string): Promise<SavedFormView> {
  const filter: SavedFormViewFilter = { search: query.search?.trim() || undefined, status: query.status, owner_principal_id: query.owner?.trim() || undefined, program_id: query.program?.trim() || undefined, use: query.use?.trim() || undefined, tag: query.tag?.trim() || undefined, limit: query.limit };
  return requestJSON<SavedFormView>(apiBase, "/api/v1/forms/saved-views", { method: "POST", body: JSON.stringify({ id, name, filter }) });
}
export function deleteSavedFormView(id: string): Promise<void> {
  return requestVoid(apiBase, `/api/v1/forms/saved-views/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function loadFormProposal(id: string): Promise<FormTemplateProposal> {
  return requestJSON<FormTemplateProposal>(apiBase, `/api/v1/forms/proposals/${encodeURIComponent(id)}`);
}

export function acceptFormProposal(id: string, expectedVersion: number, changeIDs: string[]): Promise<FormTemplateProposal> {
  return requestJSON<FormTemplateProposal>(apiBase, `/api/v1/forms/proposals/${encodeURIComponent(id)}/accept`, {
    method: "POST",
    body: JSON.stringify({ expected_version: expectedVersion, change_ids: changeIDs }),
  });
}

export function rejectFormProposal(id: string, expectedVersion: number): Promise<FormTemplateProposal> {
  return requestJSON<FormTemplateProposal>(apiBase, `/api/v1/forms/proposals/${encodeURIComponent(id)}/reject`, {
    method: "POST",
    body: JSON.stringify({ expected_version: expectedVersion }),
  });
}

export function createAIFormProposal(input: RequestAIFormProposalInput): Promise<FormTemplateProposal> {
  return requestJSON<FormTemplateProposal>(apiBase, "/api/v1/forms/proposals/ai", { method: "POST", body: JSON.stringify(input) });
}

export function createAIFormRevisionProposal(id: string, version: number, input: RequestAIFormProposalInput): Promise<FormTemplateProposal> {
  return requestJSON<FormTemplateProposal>(apiBase, `/api/v1/forms/templates/${encodeURIComponent(id)}/revisions/${version}/ai-proposals`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}
