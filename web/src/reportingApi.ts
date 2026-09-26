import { loadContext } from "./api";
import { apiErrorKind, requestBlob, requestJSON, type ApiErrorKind } from "./http";
import type {
  ReportDefinition,
  ReportDefinitionAction,
  ReportDefinitionInput,
  ReportDefinitionRevision,
  ReportDefinitionTransitionInput,
  ReportFilterFieldResponse,
  ReportRun,
  ReportRunInput,
} from "./reportingTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";
const reportBase = "/api/v1/reports";
const legacyReportBase = "/api/v1/ropa/reports";

const readFailure = "Couldn’t load reports.";
const commandFailure = "Report action failed.";

type ReportFailure = Error & { kind?: ApiErrorKind; code?: string };

function isAbortError(error: unknown) {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}

function reportFailure(error: unknown, message: string): ReportFailure {
  if (isAbortError(error)) throw error;
  const failure = new Error(message) as ReportFailure;
  const kind = apiErrorKind(error);
  if (kind !== "unknown") Object.defineProperty(failure, "kind", { value: kind, enumerable: false });
  if (typeof error === "object" && error !== null && "code" in error && typeof error.code === "string") {
    Object.defineProperty(failure, "code", { value: error.code, enumerable: false });
  }
  return failure;
}

async function scopedPath(path: string, values: Record<string, string | number | boolean | undefined> = {}) {
  const context = await loadContext();
  const query = new URLSearchParams({ tenant_id: context.tenant.id });
  for (const [key, value] of Object.entries(values)) {
    if (value !== undefined && value !== "") query.set(key, String(value));
  }
  return `${path}?${query.toString()}`;
}

function legacyReportPath(path: string) {
  return path.startsWith(reportBase) ? legacyReportBase + path.slice(reportBase.length) : path;
}

async function reportRequest<T>(path: string, init?: RequestInit): Promise<T> {
  try {
    return await requestJSON<T>(apiBase, path, init);
  } catch (error) {
    if (apiErrorKind(error) !== "not_found") throw error;
    return requestJSON<T>(apiBase, legacyReportPath(path), init);
  }
}

async function reportBlob(path: string, init?: RequestInit): Promise<{ blob: Blob; filename?: string }> {
  try {
    return await requestBlob(apiBase, path, init);
  } catch (error) {
    if (apiErrorKind(error) !== "not_found") throw error;
    return requestBlob(apiBase, legacyReportPath(path), init);
  }
}

async function scopedRequest<T>(path: string, values: Record<string, string | number | boolean | undefined>, signal: AbortSignal | undefined, failureMessage: string): Promise<T> {
  try {
    return await reportRequest<T>(await scopedPath(path, values), signal ? { signal } : undefined);
  } catch (error) {
    throw reportFailure(error, failureMessage);
  }
}

export async function listReportFilterFields(signal?: AbortSignal): Promise<ReportFilterFieldResponse> {
  try {
    return await reportRequest<ReportFilterFieldResponse>(`${reportBase}/filter-fields`, signal ? { signal } : undefined);
  } catch (error) {
    throw reportFailure(error, "Couldn’t load report filters.");
  }
}

export async function listReportDefinitions(includeRetired = false, signal?: AbortSignal): Promise<ReportDefinition[]> {
  const response = await scopedRequest<{ items?: ReportDefinition[] }>(`${reportBase}/definitions`, { include_retired: includeRetired ? "true" : undefined }, signal, readFailure);
  return response.items ?? [];
}

export async function getReportDefinition(id: string, signal?: AbortSignal): Promise<ReportDefinition> {
  return scopedRequest<ReportDefinition>(`${reportBase}/definitions/${encodeURIComponent(id)}`, {}, signal, readFailure);
}

export async function getReportDefinitionHistory(id: string, signal?: AbortSignal): Promise<ReportDefinitionRevision[]> {
  const response = await scopedRequest<{ items?: ReportDefinitionRevision[] }>(`${reportBase}/definitions/${encodeURIComponent(id)}/history`, {}, signal, readFailure);
  return response.items ?? [];
}

export async function createReportDefinition(input: ReportDefinitionInput, signal?: AbortSignal): Promise<ReportDefinition> {
  try {
    return await reportRequest<ReportDefinition>(`${reportBase}/definitions`, {
      method: "POST",
      body: JSON.stringify(input),
      ...(signal ? { signal } : {}),
    });
  } catch (error) {
    throw reportFailure(error, commandFailure);
  }
}

export async function transitionReportDefinition(id: string, action: ReportDefinitionAction, input: ReportDefinitionTransitionInput, signal?: AbortSignal): Promise<ReportDefinition> {
  try {
    return await reportRequest<ReportDefinition>(`${reportBase}/definitions/${encodeURIComponent(id)}/${action}`, {
      method: "POST",
      body: JSON.stringify(input),
      ...(signal ? { signal } : {}),
    });
  } catch (error) {
    throw reportFailure(error, commandFailure);
  }
}

export type ReportRunPage = {
  items: ReportRun[];
  next_cursor?: string;
};

export async function listReportRunPage(params: { definitionId?: string; limit?: number; cursor?: string } = {}, signal?: AbortSignal): Promise<ReportRunPage> {
  const response = await scopedRequest<{ items?: ReportRun[]; next_cursor?: string }>(`${reportBase}/runs`, {
    definition_id: params.definitionId,
    limit: params.limit,
    cursor: params.cursor,
  }, signal, readFailure);
  return { items: response.items ?? [], next_cursor: response.next_cursor || undefined };
}

export async function listReportRuns(params: { definitionId?: string; limit?: number } = {}, signal?: AbortSignal): Promise<ReportRun[]> {
  return (await listReportRunPage(params, signal)).items;
}

export async function getReportRun(id: string, signal?: AbortSignal): Promise<ReportRun> {
  return scopedRequest<ReportRun>(`${reportBase}/runs/${encodeURIComponent(id)}`, {}, signal, readFailure);
}

export async function createReportRun(definitionId: string, expectedDefinitionVersion: number, signal?: AbortSignal): Promise<ReportRun> {
  const input: ReportRunInput = { definition_id: definitionId, expected_definition_version: expectedDefinitionVersion };
  try {
    return await reportRequest<ReportRun>(`${reportBase}/runs`, {
      method: "POST",
      body: JSON.stringify(input),
      ...(signal ? { signal } : {}),
    });
  } catch (error) {
    throw reportFailure(error, commandFailure);
  }
}

export async function downloadReportRun(id: string, signal?: AbortSignal): Promise<{ blob: Blob; filename?: string }> {
  try {
    return await reportBlob(await scopedPath(`${reportBase}/runs/${encodeURIComponent(id)}/download`), signal ? { signal } : undefined);
  } catch (error) {
    throw reportFailure(error, "Download failed.");
  }
}

// Descriptive aliases keep the client convenient for callers that use the same
// read naming as the register without changing the server route contract.
export const fetchReportFilterFields = listReportFilterFields;
export const fetchReportDefinitions = listReportDefinitions;
export const fetchReportDefinition = getReportDefinition;
export const fetchReportDefinitionHistory = getReportDefinitionHistory;
export const fetchReportRuns = listReportRuns;
export const fetchReportRunPage = listReportRunPage;
export const fetchReportRun = getReportRun;
export const submitReportDefinition = (id: string, input: ReportDefinitionTransitionInput, signal?: AbortSignal) => transitionReportDefinition(id, "submit", input, signal);
export const reviewReportDefinition = (id: string, input: ReportDefinitionTransitionInput, signal?: AbortSignal) => transitionReportDefinition(id, "review", input, signal);
export const activateReportDefinition = (id: string, input: ReportDefinitionTransitionInput, signal?: AbortSignal) => transitionReportDefinition(id, "activate", input, signal);
export const rejectReportDefinition = (id: string, input: ReportDefinitionTransitionInput, signal?: AbortSignal) => transitionReportDefinition(id, "reject", input, signal);
export const retireReportDefinition = (id: string, input: ReportDefinitionTransitionInput, signal?: AbortSignal) => transitionReportDefinition(id, "retire", input, signal);
