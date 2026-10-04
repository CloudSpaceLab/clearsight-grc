import { requestJSON } from "./http";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "";

export type HomeFocusPreference = "AUTO" | "POSTURE" | "MY_WORK";
export type PortfolioLensPreference = "AUTO" | "PROGRAMS" | "RISKS" | "LOSSES" | "VENDORS" | "PROCESSING_ACTIVITIES" | "FORMS";

export type PresentationPreferences = {
  tenant_id: string;
  principal_id: string;
  home_focus: HomeFocusPreference;
  portfolio_lens: PortfolioLensPreference;
  effective_home_focus: Exclude<HomeFocusPreference, "AUTO">;
  effective_portfolio_lens: Exclude<PortfolioLensPreference, "AUTO">;
  updated_at?: string;
  version: number;
};

export function loadPresentationPreferences(signal?: AbortSignal): Promise<PresentationPreferences> {
  return requestJSON<PresentationPreferences>(
    apiBase,
    "/api/v1/preferences/presentation",
    signal ? { signal } : undefined,
  );
}

export function savePresentationPreferences(
  value: Pick<PresentationPreferences, "home_focus" | "portfolio_lens" | "version">,
): Promise<PresentationPreferences> {
  return requestJSON<PresentationPreferences>(apiBase, "/api/v1/preferences/presentation", {
    method: "PUT",
    body: JSON.stringify({
      home_focus: value.home_focus,
      portfolio_lens: value.portfolio_lens,
      expected_version: value.version,
    }),
  });
}
