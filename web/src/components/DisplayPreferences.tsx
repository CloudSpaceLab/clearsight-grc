import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

type ThemePreference = "system" | "light" | "dark";
type DensityPreference = "comfortable" | "compact";
type DisplayPreferencesValue = {
  theme: ThemePreference;
  density: DensityPreference;
  setTheme: (value: ThemePreference) => void;
  setDensity: (value: DensityPreference) => void;
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

// Apply persisted/system preferences before React mounts to minimize theme flash.
applyTheme(readTheme());
applyDensity(readDensity());

export function DisplayPreferencesRoot({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<ThemePreference>(readTheme);
  const [density, setDensity] = useState<DensityPreference>(readDensity);

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

  return <DisplayPreferencesContext.Provider value={{ theme, density, setTheme, setDensity }}>{children}</DisplayPreferencesContext.Provider>;
}

export function DisplayPreferencesMenu() {
  const preferences = useContext(DisplayPreferencesContext);
  if (!preferences) return null;
  const { theme, density, setTheme, setDensity } = preferences;

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
    </div>
  </details>;
}

function label(value: string) {
  return value.charAt(0).toUpperCase() + value.slice(1);
}
