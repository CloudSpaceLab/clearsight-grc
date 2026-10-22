import { describe, expect, it } from "vitest";

const interfaceSources = import.meta.glob(
  ["./*.ts", "./*.tsx", "./components/**/*.ts", "./components/**/*.tsx", "!./**/*.test.ts", "!./**/*.test.tsx", "!./**/*.d.ts"],
  { eager: true, query: "?raw", import: "default" },
) as Record<string, string>;

const productCommentary = [
  /\b(?:latest stored|reusable now)\b/i,
  /evidence quality is assessed separately/i,
  /a response is recorded first/i,
  /this form editor is not connected to a governed save command/i,
  /checks whether calculated program status has caught up/i,
  /review who can respond, then create or replace/i,
  /the initial request is followed by up to/i,
  /rules are evaluated from the stored form revision/i,
  /this question is always shown\. add logic only when/i,
  /activation still requires a separate approver/i,
  /no expired documents or recorded response gaps were identified/i,
  /exact retained records/i,
  /\bexact\s+(?:snapshot|retained)\s+(?:members?|detail|records?)\b/i,
  /append-only recoveries/i,
  /governed (?:indicator population|Program monitoring|responsibility policies)/i,
  /injected structurally ahead of/i,
  /from projection \$?\{/i,
  /the current snapshot contains no rows/i,
  /this control is independent of/i,
  /\b(?:awaiting|require|needs|pending) bank review\b/i,
  /\bbank (?:assessment|judgement|rubric|reviewer|decision)\b/i,
  /\bwithout mixing\b/i,
  /assigned work remains canonical/i,
  /\bbounded view\b/i,
  /<h[1-6][^>]*>\s*what (?:is|[’']s) still needed\s*<\/h[1-6]>/i,
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
  /canonical (?:object|title|statement)/i,
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
