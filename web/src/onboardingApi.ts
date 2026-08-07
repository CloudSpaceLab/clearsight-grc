import { loadContext } from "./api";
import type { OnboardingGuide, OnboardingState } from "./types";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as { message?: string; error?: { message?: string } } | null;
    throw new Error(body?.error?.message ?? body?.message ?? `Request failed with ${response.status}`);
  }
  return (await response.json()) as T;
}

export function loadRoleGuide(): Promise<OnboardingGuide> {
  return request<OnboardingGuide>("/api/v1/onboarding/guide");
}

export async function loadGuideState(guideCode: string): Promise<OnboardingState> {
  const context = await loadContext();
  const params = new URLSearchParams({
    tenant_id: context.tenant.id,
    principal_id: context.actor.id,
    guide_code: guideCode,
  });
  return request<OnboardingState>(`/api/v1/onboarding/state?${params.toString()}`);
}

export async function saveGuideState(guideCode: string, value: Pick<OnboardingState, "current_step" | "completed" | "dismissed" | "version">): Promise<OnboardingState> {
  const context = await loadContext();
  const params = new URLSearchParams({
    tenant_id: context.tenant.id,
    principal_id: context.actor.id,
    guide_code: guideCode,
  });
  return request<OnboardingState>(`/api/v1/onboarding/state?${params.toString()}`, {
    method: "PUT",
    body: JSON.stringify({
      current_step: value.current_step,
      completed: value.completed,
      dismissed: value.dismissed,
      expected_version: value.version,
    }),
  });
}
