import { loadContext } from "./api";
import { requestJSON } from "./http";
import type {
  ActivityPage,
  ProcessingActivity,
  ProcessingActivityHistoryResponse,
  ProcessingActivityPage,
  ProcessingActivityPageWire,
  ProcessingActivityResponse,
  ProcessingActivityStatus,
  RegisterSummary,
} from "./ropaTypes";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type RopaProcessingActivityListParams = {
  status?: ProcessingActivityStatus;
  search?: string;
  cursor?: string;
  include_retired?: boolean;
};

export type RopaProcessingActivityHistoryParams = {
  after_version?: number;
  limit?: number;
};

type ProcessingActivityPagePayload = ProcessingActivityPageWire | {
  rows: ProcessingActivity[];
  next_cursor?: string;
  has_more: boolean;
};

function isAbortError(error: unknown) {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}

async function scopedRequest<T>(path: string, params: Record<string, string | number | boolean | undefined>, signal: AbortSignal | undefined, failureMessage: string): Promise<T> {
  try {
    const context = await loadContext();
    const query = new URLSearchParams({ tenant_id: context.tenant.id });
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined && value !== "") query.set(key, String(value));
    }
    const suffix = query.toString();
    return await requestJSON<T>(apiBase, `${path}?${suffix}`, signal ? { signal } : undefined);
  } catch (error) {
    if (isAbortError(error)) throw error;
    const failure = new Error(failureMessage) as Error & { kind?: string };
    if (typeof error === "object" && error !== null && "kind" in error && typeof error.kind === "string") {
      Object.defineProperty(failure, "kind", { value: error.kind, enumerable: false });
    }
    throw failure;
  }
}

function normalizeActivityPage(value: ProcessingActivityPagePayload): ActivityPage {
  const wire = value as ProcessingActivityPageWire & { rows?: ProcessingActivity[]; next_cursor?: string; has_more?: boolean };
  const rows = wire.Rows ?? wire.rows ?? [];
  const nextCursor = wire.NextCursor ?? wire.next_cursor;
  const page: ActivityPage = { rows, has_more: wire.HasMore ?? wire.has_more ?? false };
  if (nextCursor !== undefined) page.next_cursor = nextCursor;
  return page;
}

export function fetchDashboard(signal?: AbortSignal): Promise<RegisterSummary> {
  return scopedRequest<RegisterSummary>("/api/v1/ropa/dashboard", {}, signal, "Couldn’t load processing activities.");
}

export function listProcessingActivities(params: RopaProcessingActivityListParams = {}, signal?: AbortSignal): Promise<ProcessingActivityPage> {
  return scopedRequest<ProcessingActivityPagePayload>(
    "/api/v1/ropa/processing-activities",
    {
      status: params.status,
      search: params.search?.trim() || undefined,
      cursor: params.cursor,
      include_retired: params.include_retired ? "true" : undefined,
    },
    signal,
    "The processing activity register could not be loaded. Try again.",
  ).then(normalizeActivityPage);
}

export function fetchProcessingActivity(id: string, signal?: AbortSignal): Promise<ProcessingActivityResponse> {
  return scopedRequest<ProcessingActivityResponse>(
    `/api/v1/ropa/processing-activities/${encodeURIComponent(id)}`,
    {},
    signal,
    "Couldn’t load processing activity.",
  );
}

export function fetchProcessingActivityHistory(id: string, params: RopaProcessingActivityHistoryParams = {}, signal?: AbortSignal): Promise<ProcessingActivityHistoryResponse> {
  return scopedRequest<ProcessingActivityHistoryResponse>(
    `/api/v1/ropa/processing-activities/${encodeURIComponent(id)}/history`,
    { after_version: params.after_version, limit: params.limit },
    signal,
    "Couldn’t load history.",
  );
}
