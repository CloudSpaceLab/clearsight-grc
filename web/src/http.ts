export type ApiErrorKind = "unauthorized" | "forbidden" | "not_found" | "conflict" | "validation" | "unavailable" | "unknown";

type ErrorEnvelope = { message?: string; error?: { code?: string; message?: string } };

export class ApiError extends Error {
  readonly status: number;
  readonly code?: string;
  readonly kind: ApiErrorKind;

  constructor(status: number, message: string, code?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.kind = kindFromStatus(status);
  }
}

export async function parseJSON<T>(response: Response): Promise<T> {
  if (!response.ok) throw await responseError(response);
  return await response.json() as T;
}

export async function requestJSON<T>(apiBase: string, path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  if (!(init?.body instanceof FormData) && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  return parseJSON<T>(await fetch(`${apiBase}${path}`, { ...init, credentials: init?.credentials ?? "include", headers }));
}

export async function requestVoid(apiBase: string, path: string, init?: RequestInit): Promise<void> {
  const headers = new Headers(init?.headers);
  if (!(init?.body instanceof FormData) && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  const response = await fetch(`${apiBase}${path}`, { ...init, credentials: init?.credentials ?? "include", headers });
  if (!response.ok) throw await responseError(response);
}

export async function requestBlob(apiBase: string, path: string, init?: RequestInit): Promise<{ blob: Blob; filename?: string }> {
  const response = await fetch(`${apiBase}${path}`, { ...init, credentials: init?.credentials ?? "include" });
  if (!response.ok) throw await responseError(response);
  return {
    blob: await response.blob(),
    filename: responseFilename(response.headers.get("Content-Disposition")),
  };
}

export function apiErrorKind(error: unknown): ApiErrorKind {
  return error instanceof ApiError ? error.kind : "unknown";
}

async function responseError(response: Response): Promise<ApiError> {
  const body = await response.json().catch(() => null) as ErrorEnvelope | null;
  return new ApiError(
    response.status,
    body?.error?.message ?? body?.message ?? `Request failed with ${response.status}`,
    body?.error?.code,
  );
}

function responseFilename(contentDisposition: string | null): string | undefined {
  if (!contentDisposition) return undefined;
  const encoded = /filename\*=UTF-8''([^;]+)/i.exec(contentDisposition)?.[1];
  if (encoded) {
    try { return decodeURIComponent(encoded); } catch { return undefined; }
  }
  const quoted = /filename="([^"]+)"/i.exec(contentDisposition)?.[1];
  return quoted?.trim() || undefined;
}

function kindFromStatus(status: number): ApiErrorKind {
  if (status === 401) return "unauthorized";
  if (status === 403) return "forbidden";
  if (status === 404) return "not_found";
  if (status === 409 || status === 412) return "conflict";
  if (status === 400 || status === 413 || status === 415 || status === 422) return "validation";
  if (status === 429 || status >= 500) return "unavailable";
  return "unknown";
}
