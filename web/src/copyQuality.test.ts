import { describe, expect, it } from "vitest";

const interfaceSources = import.meta.glob(
  ["./*.ts", "./*.tsx", "./components/*.ts", "./components/*.tsx", "./components/forms/*.ts", "./components/forms/*.tsx", "!./**/*.test.ts", "!./**/*.test.tsx"],
  { eager: true, query: "?raw", import: "default" },
) as Record<string, string>;

const productCommentary = [
  /generic dashboard/i,
  /exact (?:linked )?record/i,
  /authoritative server/i,
  /server-(?:authoritative|backed|paged|rendered|side)/i,
  /sent to (?:the )?server/i,
  /protected server boundary/i,
  /append-only server records/i,
  /projection cycle/i,
  /\bprojection\b(?=\s+(?:version|status|data|result|health|record|update)\b|\s*\$\{)/i,
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
  /mapped (?:missing )?(?:information|item)/i,
  /closure remain(?:s)? separate/i,
  /binding'?s score threshold/i,
  /exact (?:final )?response/i,
  /exact (?:approved )?(?:form )?revision/i,
  /immutable (?:draft|profile|template|revision)/i,
  /revision is immutable/i,
  /bounded (?:condition|simulation|response population|subject population)/i,
  /server (?:checks|score preview)/i,
  /current (?:authority|responsibility) route/i,
  /governed (?:rollback )?(?:draft|revision)/i,
  /source-backed (?:suggestion|obligation|field proposal)/i,
  /exact version/i,
  /exact change reviewers/i,
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
