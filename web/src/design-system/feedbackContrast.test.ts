// @ts-ignore Vitest executes this CSS source regression in Node.
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

type RGB = [number, number, number];

const primitivesCSS = readFileSync("src/design-system/tokens/primitives.css", "utf8");
const preferencesCSS = readFileSync("src/ui-preferences.css", "utf8");
const feedbackCSS = readFileSync("src/design-system/components/feedback.css", "utf8");
const monitoringCSS = readFileSync("src/monitoring.css", "utf8");

function declarations(css: string) {
  const result = new Map<string, string>();
  for (const match of css.matchAll(/(--[a-z0-9-]+)\s*:\s*([^;]+);/gi)) {
    const name = match[1];
    const value = match[2];
    if (name && value) result.set(name, value.trim());
  }
  return result;
}

function themeBlock(theme: "dark" | "light") {
  const selector = theme === "dark" ? /:root,\s*:root\[data-theme="dark"\]\s*\{([\s\S]*?)\n\}/ : /:root\[data-theme="light"\]\s*\{([\s\S]*?)\n\}/;
  const block = preferencesCSS.match(selector)?.[1];
  if (!block) throw new Error(`${theme} theme tokens were not found`);
  return declarations(block);
}

const primitives = declarations(primitivesCSS);

function parseHex(value: string): RGB {
  const match = value.match(/^#([0-9a-f]{6})$/i);
  const raw = match?.[1];
  if (!raw) throw new Error(`Unsupported colour ${value}`);
  return [Number.parseInt(raw.slice(0, 2), 16), Number.parseInt(raw.slice(2, 4), 16), Number.parseInt(raw.slice(4, 6), 16)];
}

function mix(foreground: RGB, foregroundPercent: number, background: RGB): RGB {
  const alpha = foregroundPercent / 100;
  return foreground.map((channel, index) => channel * alpha + background[index]! * (1 - alpha)) as RGB;
}

function resolve(value: string, tokens: Map<string, string>): RGB {
  if (value.startsWith("#")) return parseHex(value);
  const variable = value.match(/^var\((--[a-z0-9-]+)\)$/i);
  if (variable) {
    const name = variable[1];
    if (!name) throw new Error(`Invalid token expression ${value}`);
    const next = tokens.get(name) ?? primitives.get(name);
    if (!next) throw new Error(`Unknown token ${name}`);
    return resolve(next, tokens);
  }
  const mixed = value.match(/^color-mix\(in srgb,\s*var\((--[a-z0-9-]+)\)\s+(\d+)%,\s*var\((--[a-z0-9-]+)\)\)$/i);
  if (mixed?.[1] && mixed[2] && mixed[3]) return mix(resolve(`var(${mixed[1]})`, tokens), Number(mixed[2]), resolve(`var(${mixed[3]})`, tokens));
  throw new Error(`Unsupported token expression ${value}`);
}

function luminance([red, green, blue]: RGB) {
  const linear = [red, green, blue].map((channel) => {
    const value = channel / 255;
    return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * linear[0]! + 0.7152 * linear[1]! + 0.0722 * linear[2]!;
}

function contrast(left: RGB, right: RGB) {
  const [lighter, darker] = [luminance(left), luminance(right)].sort((a, b) => b - a) as [number, number];
  return (lighter + 0.05) / (darker + 0.05);
}

function required(tokens: Map<string, string>, name: string) {
  const value = tokens.get(name);
  if (!value) throw new Error(`Missing theme token ${name}`);
  return value;
}

describe("feedback contrast", () => {
  for (const theme of ["dark", "light"] as const) {
    it(`keeps Notice and success-label text at WCAG AA contrast in ${theme} mode`, () => {
      const tokens = themeBlock(theme);
      const text = resolve(required(tokens, "--cs-text-primary"), tokens);
      const success = resolve(required(tokens, "--cs-status-success"), tokens);
      const surface = resolve(required(tokens, "--cs-bg-surface-1"), tokens);
      const noticeBackground = mix(success, 8, surface);
      const successBackground = mix(success, 10, surface);

      expect(contrast(text, noticeBackground)).toBeGreaterThanOrEqual(4.5);
      expect(contrast(success, successBackground)).toBeGreaterThanOrEqual(4.5);
    });
  }

  it("uses semantic feedback foregrounds and removes the dark-only success class", () => {
    expect(feedbackCSS).toContain("color: var(--cs-text-primary)");
    expect(monitoringCSS).not.toContain(".inline-success");
    expect(monitoringCSS).not.toContain("#bdf5d8");
  });
});
