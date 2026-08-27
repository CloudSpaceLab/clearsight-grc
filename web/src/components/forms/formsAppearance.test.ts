import { describe, expect, it } from "vitest";
import { appearanceStorageKey, defaultFormsAccent, loadFormsAppearance, normalizeAccentColor, normalizeLogoURL, saveFormsAppearance } from "./formsAppearance";

describe("Forms appearance", () => {
  it("normalizes safe accent colors and rejects arbitrary values", () => {
    expect(normalizeAccentColor("#3a7")).toBe("#33AA77");
    expect(normalizeAccentColor("#2244cc")).toBe("#2244CC");
    expect(normalizeAccentColor("url(javascript:alert(1))")).toBe(defaultFormsAccent);
  });

  it("accepts https and same-origin logos but rejects insecure third-party URLs", () => {
    const base = "https://clearsight.example/forms";
    expect(normalizeLogoURL("/bank/logo.svg", base)).toBe("https://clearsight.example/bank/logo.svg");
    expect(normalizeLogoURL("https://assets.bank.example/logo.png", base)).toBe("https://assets.bank.example/logo.png");
    expect(normalizeLogoURL("http://tracker.example/logo.png", base)).toBeUndefined();
    expect(normalizeLogoURL("javascript:alert(1)", base)).toBeUndefined();
  });

  it("stores only normalized browser-local presentation preferences", () => {
    const values = new Map<string, string>();
    const storage = {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => { values.set(key, value); },
    };
    const base = "https://clearsight.example/forms";
    const stored = saveFormsAppearance(storage, "entity-a", { accentColor: "#3a7", logoURL: "/logo.svg" }, base);
    expect(stored).toEqual({ accentColor: "#33AA77", logoURL: "https://clearsight.example/logo.svg" });
    expect(values.has(appearanceStorageKey("entity-a"))).toBe(true);
    expect(loadFormsAppearance(storage, "entity-a", base)).toEqual(stored);
  });
});
