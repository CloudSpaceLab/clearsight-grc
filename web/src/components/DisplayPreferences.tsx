import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import {
  loadPresentationPreferences,
  savePresentationPreferences,
  type HomeFocusPreference,
  type PortfolioLensPreference,
  type PresentationPreferences,
} from "../presentationPreferencesApi";

type ThemePreference = "system" | "light" | "dark";
type DensityPreference = "comfortable" | "compact";
type PresentationState = "loading" | "live" | "saving" | "unavailable";

type DisplayPreferencesValue = {
  theme: ThemePreference;
  density: DensityPreference;
  presentation?: PresentationPreferences;
  presentationState: PresentationState;
  setTheme: (value: ThemePreference) => void;
  setDensity: (value: DensityPreference) => void;
  setHomeFocus: (value: HomeFocusPreference) => void;
  setPortfolioLens: (value: PortfolioLensPreference) => void;
};

const THEME_KEY = "clearsight.theme";
const DENSITY_KEY = "clearsight.density";
const DisplayPreferencesContext = createContext<DisplayPreferencesValue | null>(null);

function readTheme(): ThemePreference {
  if (typeof window === "undefined") return "system";
  try {
    const value = window.localStorage.getItem(THEME_KEY);
    if (value === "system" || value === "light" || value === "dark") return value;
  } catch {
    // Storage can be unavailable in privacy-restricted environments.
  }
  return "system";
}

function readDensity(): DensityPreference {
  if (typeof window === "undefined") return "comfortable";
  try {
    const value = window.localStorage.getItem(DENSITY_KEY);
    if (value === "comfortable" || value === "compact") return value;
  } catch {
    // Storage can be unavailable in privacy-restricted environments.
  }
  return "comfortable";
}

function systemTheme(): "light" | "dark" {
  return typeof window !== "undefined" && typeof window.matchMedia === "function" && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function applyTheme(preference: ThemePreference) {
  if (typeof document === "undefined") return;
  const resolved = preference === "system" ? systemTheme() : preference;
  document.documentElement.dataset.theme = resolved;
  document.documentElement.dataset.themePreference = preference;
  document.documentElement.style.colorScheme = resolved;
}

function applyDensity(preference: DensityPreference) {
  if (typeof document === "undefined") return;
  document.documentElement.dataset.density = preference;
}

function writePreference(key: string, value: string) {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(key, value);
  } catch {
    // Preference persistence is optional; the active session still updates.
  }
}

applyTheme(readTheme());
applyDensity(readDensity());

export function DisplayPreferencesRoot({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<ThemePreference>(readTheme);
  const [density, setDensity] = useState<DensityPreference>(readDensity);
  const [presentation, setPresentation] = useState<PresentationPreferences>();
  const [presentationState, setPresentationState] = useState<PresentationState>("loading");

  useEffect(() => {
    applyTheme(theme);
    writePreference(THEME_KEY, theme);

    if (theme !== "system" || typeof window.matchMedia !== "function") return;
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const update = () => applyTheme("system");
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, [theme]);

  useEffect(() => {
    applyDensity(density);
    writePreference(DENSITY_KEY, density);
  }, [density]);

  useEffect(() => {
    const controller = new AbortController();
    setPresentationState("loading");
    void loadPresentationPreferences(controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setPresentation(value);
      setPresentationState("live");
    }).catch(() => {
      if (!controller.signal.aborted) setPresentationState("unavailable");
    });
    return () => controller.abort();
  }, []);

  function updatePresentation(next: Pick<PresentationPreferences, "home_focus" | "portfolio_lens">) {
    if (!presentation || presentationState === "saving") return;
    setPresentationState("saving");
    void savePresentationPreferences({ ...next, version: presentation.version }).then((value) => {
      setPresentation(value);
      setPresentationState("live");
    }).catch(() => {
      void loadPresentationPreferences().then((value) => {
        setPresentation(value);
        setPresentationState("live");
      }).catch(() => setPresentationState("unavailable"));
    });
  }

  return <DisplayPreferencesContext.Provider value={{
    theme,
    density,
    presentation,
    presentationState,
    setTheme,
    setDensity,
    setHomeFocus: (value) => updatePresentation({ home_focus: value, portfolio_lens: presentation?.portfolio_lens ?? "AUTO" }),
    setPortfolioLens: (value) => updatePresentation({ home_focus: presentation?.home_focus ?? "AUTO", portfolio_lens: value }),
  }}>{children}</DisplayPreferencesContext.Provider>;
}

export function useDisplayPreferences() {
  return useContext(DisplayPreferencesContext);
}

export function DisplayPreferencesMenu() {
  const preferences = useDisplayPreferences();
  if (!preferences) return null;
  const {
    theme,
    density,
    presentation,
    presentationState,
    setTheme,
    setDensity,
    setHomeFocus,
    setPortfolioLens,
  } = preferences;

  return <details className="display-preferences">
    <summary aria-label="Display preferences">Display</summary>
    <div className="display-preferences-popover">
      <div className="display-preference-group" role="group" aria-label="Theme">
        <span>Theme</span>
        <div>
          {(["system", "light", "dark"] as const).map((value) => <button key={value} type="button" aria-pressed={theme === value} onClick={() => setTheme(value)}>{label(value)}</button>)}
        </div>
      </div>
      <div className="display-preference-group" role="group" aria-label="Density">
        <span>Density</span>
        <div>
          {(["comfortable", "compact"] as const).map((value) => <button key={value} type="button" aria-pressed={density === value} onClick={() => setDensity(value)}>{label(value)}</button>)}
        </div>
      </div>
      {presentation && <div className="display-preference-group">
        <label htmlFor="display-home-focus">Home focus</label>
        <select id="display-home-focus" value={presentation.home_focus} disabled={presentationState === "saving"} onChange={(event) => setHomeFocus(event.target.value as HomeFocusPreference)}>
          <option value="AUTO">Role default · {homeFocusLabel(presentation.effective_home_focus)}</option>
          <option value="POSTURE">Posture first</option>
          <option value="MY_WORK">My work first</option>
        </select>
      </div>}
      {presentation && <div className="display-preference-group">
        <label htmlFor="display-portfolio-lens">Portfolio start</label>
        <select id="display-portfolio-lens" value={presentation.portfolio_lens} disabled={presentationState === "saving"} onChange={(event) => setPortfolioLens(event.target.value as PortfolioLensPreference)}>
          <option value="AUTO">Role default · {portfolioLensLabel(presentation.effective_portfolio_lens)}</option>
          <option value="PROGRAMS">Programs</option>
          <option value="RISKS">Risks</option>
          <option value="LOSSES">Losses</option>
          <option value="VENDORS">Vendors</option>
          <option value="PROCESSING_ACTIVITIES">Processing activities</option>
          <option value="FORMS">Forms</option>
        </select>
      </div>}
      {presentationState === "unavailable" && <small>Workspace defaults unavailable.</small>}
    </div>
  </details>;
}

function homeFocusLabel(value: Exclude<HomeFocusPreference, "AUTO">) {
  return value === "POSTURE" ? "Posture first" : "My work first";
}

function portfolioLensLabel(value: Exclude<PortfolioLensPreference, "AUTO">) {
  switch (value) {
    case "RISKS": return "Risks";
    case "LOSSES": return "Losses";
    case "VENDORS": return "Vendors";
    case "PROCESSING_ACTIVITIES": return "Processing activities";
    case "FORMS": return "Forms";
    default: return "Programs";
  }
}

function label(value: string) {
  return value.charAt(0).toUpperCase() + value.slice(1);
}
