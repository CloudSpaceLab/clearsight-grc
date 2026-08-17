import { describe, expect, it } from "vitest";

const interfaceSources = import.meta.glob(
  ["./*.ts", "./*.tsx", "./components/*.ts", "./components/*.tsx", "!./**/*.test.ts", "!./**/*.test.tsx"],
  { eager: true, query: "?raw", import: "default" },
) as Record<string, string>;

const productCommentary = [
  /generic dashboard/i,
  /exact (?:linked )?record/i,
  /authoritative server/i,
  /projection cycle/i,
  /bounded daily digest/i,
  /current canonical/i,
  /second directory console/i,
  /governed candidate set/i,
  /without needing to know/i,
  /clearsight resolved/i,
  /product behaviou?r/i,
  /atomically activat/i,
  /program truth/i,
  /\bis inferred\b/i,
  /ClearSight Demonstration Bank/i,
  /Demonstration Bank Nigeria/i,
  /\bDemo Bank(?: Nigeria)?\b/i,
] as const;

describe("customer-facing product language", () => {
  it("does not expose product-review or implementation commentary", () => {
    const violations: string[] = [];

    for (const [file, source] of Object.entries(interfaceSources)) {
      for (const pattern of productCommentary) {
        const match = pattern.exec(source);
        if (!match || match.index === undefined) continue;
        const line = source.slice(0, match.index).split("\n").length;
        violations.push(`${file}:${line} contains ${JSON.stringify(match[0])}`);
      }
    }

    expect(violations, violations.join("\n")).toEqual([]);
  });
});
