import type { AttentionItem, AuthorityResolution, CaptureRequest } from "./types";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers }
  });
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as { message?: string } | null;
    throw new Error(body?.message ?? `Request failed with ${response.status}`);
  }
  return (await response.json()) as T;
}

export async function loadToday(): Promise<AttentionItem[]> {
  const body = await request<{ items: AttentionItem[] }>("/api/v1/today");
  return body.items;
}

export function resolveAuthority(): Promise<AuthorityResolution> {
  return request<AuthorityResolution>("/api/v1/authority/resolve", {
    method: "POST",
    body: JSON.stringify({
      tenant_id: "bank-demo",
      legal_entity_id: "bank-ng",
      object_type: "MATTER",
      object_id: "matter-demo",
      responsibility: "AUTHORIZER",
      materiality: 5
    })
  });
}

export function loadCaptureRequest(): Promise<CaptureRequest> {
  return request<CaptureRequest>("/api/v1/requests/req_branch_generator");
}

export function submitCaptureRequest(id: string, version: number, answers: Record<string, string>) {
  return request<{ request_id: string; status: string; submitted_at: string }>(`/api/v1/requests/${id}/submit`, {
    method: "POST",
    body: JSON.stringify({ version, answers })
  });
}
