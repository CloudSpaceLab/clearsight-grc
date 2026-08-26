import { loadContext } from "./api";
import { requestJSON } from "./http";
import type { GuideSurface, OnboardingGuide, OnboardingState } from "./types";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export function loadRoleGuide(surface: GuideSurface = "TODAY"): Promise<OnboardingGuide> {
  const params = new URLSearchParams({ surface });
  return requestJSON<OnboardingGuide>(apiBase, `/api/v1/onboarding/guide?${params.toString()}`);
}

export async function loadGuideState(guideCode: string): Promise<OnboardingState> {
  const context = await loadContext();
  const params = new URLSearchParams({
    tenant_id: context.tenant.id,
    principal_id: context.actor.id,
    guide_code: guideCode,
  });
  return requestJSON<OnboardingState>(apiBase, `/api/v1/onboarding/state?${params.toString()}`);
}

export async function saveGuideState(guideCode: string, value: Pick<OnboardingState, "current_step" | "completed" | "dismissed" | "version">): Promise<OnboardingState> {
  const context = await loadContext();
  const params = new URLSearchParams({
    tenant_id: context.tenant.id,
    principal_id: context.actor.id,
    guide_code: guideCode,
  });
  return requestJSON<OnboardingState>(apiBase, `/api/v1/onboarding/state?${params.toString()}`, {
    method: "PUT",
    body: JSON.stringify({
      current_step: value.current_step,
      completed: value.completed,
      dismissed: value.dismissed,
      expected_version: value.version,
    }),
  });
}
