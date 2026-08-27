export type FormsAppearance = {
  accentColor: string;
  logoURL?: string;
};

export const defaultFormsAccent = "#3867FF";

export function appearanceStorageKey(scope: string) {
  const normalized = scope.trim() || "default";
  return `clearsight:forms:appearance:${normalized}`;
}

export function normalizeAccentColor(value?: string) {
  const trimmed = value?.trim();
  if (!trimmed) return defaultFormsAccent;
  if (/^#[0-9a-f]{6}$/i.test(trimmed)) return trimmed.toUpperCase();
  if (/^#[0-9a-f]{3}$/i.test(trimmed)) {
    const [r, g, b] = trimmed.slice(1).split("");
    return `#${r}${r}${g}${g}${b}${b}`.toUpperCase();
  }
  return defaultFormsAccent;
}

export function contrastTextForAccent(value: string) {
  const color = normalizeAccentColor(value).slice(1);
  const channels = [0, 2, 4].map((offset) => Number.parseInt(color.slice(offset, offset + 2), 16) / 255);
  const [red, green, blue] = channels.map((channel) => channel <= 0.03928 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4);
  const luminance = 0.2126 * red! + 0.7152 * green! + 0.0722 * blue!;
  return luminance > 0.45 ? "#111827" : "#FFFFFF";
}

export function normalizeLogoURL(value: string | undefined, baseURL: string) {
  const trimmed = value?.trim();
  if (!trimmed || trimmed.length > 2048) return undefined;
  try {
    const base = new URL(baseURL);
    const resolved = new URL(trimmed, base);
    if (resolved.protocol === "https:") return resolved.href;
    if (resolved.origin === base.origin && (resolved.protocol === "http:" || resolved.protocol === "https:")) return resolved.href;
  } catch {
    return undefined;
  }
  return undefined;
}

export function loadFormsAppearance(storage: Pick<Storage, "getItem">, scope: string, baseURL: string): FormsAppearance {
  try {
    const raw = storage.getItem(appearanceStorageKey(scope));
    if (!raw) return { accentColor: defaultFormsAccent };
    const parsed = JSON.parse(raw) as Partial<FormsAppearance>;
    return {
      accentColor: normalizeAccentColor(parsed.accentColor),
      logoURL: normalizeLogoURL(parsed.logoURL, baseURL),
    };
  } catch {
    return { accentColor: defaultFormsAccent };
  }
}

export function saveFormsAppearance(storage: Pick<Storage, "setItem">, scope: string, appearance: FormsAppearance, baseURL: string) {
  const normalized: FormsAppearance = {
    accentColor: normalizeAccentColor(appearance.accentColor),
    logoURL: normalizeLogoURL(appearance.logoURL, baseURL),
  };
  storage.setItem(appearanceStorageKey(scope), JSON.stringify(normalized));
  return normalized;
}
