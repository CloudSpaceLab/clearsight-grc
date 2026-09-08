# Governed Result Handoffs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Open governed document-analysis results directly in existing record workspaces.

**Architecture:** Extend the existing typed hash route with an optional Program requirement/control-objective target; reuse authorized aggregate reads and shared ActionLink. Navigation never performs a material command. Missing targets recover without guessing.

**Tech Stack:** React/TypeScript/Vitest, existing Go-backed reads unchanged.

## Task 1 — Implement the bounded handoff slice

Files: `web/src/appRouting.ts`, `App.tsx`, `AppViews.tsx`, `components/ProgramsWorkspace.tsx`, `ProgramRecordWorkspace.tsx`, `ProgramRequirementsPanel.tsx`, `ProgramSafeguardsPanel.tsx`, `DocumentProposalHandoff.tsx`, `DocumentImportWorkspace.tsx`; their existing tests. A small shared result-route helper is permitted only if both import review paths consume it. No private UI components or backend change.

- [ ] Add route and receipt regression tests before implementation. Contract example:

```ts
const target = { programID: "program/1", programSection: "requirements-controls" as const,
  programItem: { kind: "requirement" as const, id: "requirement/1" } };
expect(routeHash("programs", target, "matters"))
  .toBe("#programs/program%2F1/requirements-controls/requirement/requirement%2F1");
expect(parseRoute(routeHash("programs", target, "matters")).target).toEqual(target);
```

Also test missing/unsupported item types and old section routes. In the existing approved handoff fixture, assert Open requirement has the stored Program/result route; repeat for CONTROL_OBJECTIVE and missing result metadata. In coverage fixtures assert stored matched requirement, related Matter and applied Program/Requirement/Matter links; proposed/failed/unknown suggestions must not emit a success link.

- [ ] Run `cd web; npm test -- src/appRouting.test.ts src/components/DocumentProposalHandoff.test.tsx src/components/DocumentImportWorkspace.test.tsx src/components/ProgramRecordWorkspace.test.tsx`; capture expected assertion failures (baseline before new tests: 53 passed).
- [ ] Add `ProgramItemTarget = { kind: "requirement" | "control-objective"; id: string }` and optional `programItem` to the existing route. Parse it only for the requirements-controls section with complete supported segments. Build encoded paths from stored identifiers, not arbitrary hrefs. Preserve legacy parse results without extra undefined properties.
- [ ] Thread the optional target through existing props. Add stable, focusable record containers with accessible labels. Match against the current authorized aggregate before focusing; use an effect/ref keyed to target identity so background reload cannot steal focus. Missing targets show a safe Notice; do not render unknown IDs or select another record. Tab changes remove the item target through existing navigation.
- [ ] Replace stranded result text with shared ActionLink for supported complete receipts. For example, the requirement href is `routeHash("programs", { programID: handoff.target_program_id, programSection: "requirements-controls", programItem: { kind: "requirement", id: handoff.result_object_id } }, "matters")` only after validating those recorded fields and supported result type. Use ordinary issue/Program routes for corresponding coverage outputs. Keep failed/unknown results read-only with truthful recovery. Review the entire affected handoff copy, replacing internal implementation narration without weakening authority.
- [ ] Re-run focused tests until green, then `npm run typecheck`, `npm test -- src/copyQuality.test.ts`, `npm run check:ui-contracts`, `npm run build` and `npm run build:evidence`.
- [ ] Self-review and commit only task files. Obtain independent spec review, then independent quality review; repair findings before integration.

## Task 2 — Render, reconcile issue ownership and release

- [ ] Add or extend existing evidence fixtures for approved/missing handoffs and focused Program targets; run actual production-component renders in light/dark desktop and narrow replacements. Inspect outputs, fix/recheck highest-impact defects. Record commands/screenshots and limitations in the decision brief.
- [ ] Update `docs/implementation-plan.md` with resumed #200 scope and the historical status of the September pause. Preserve old evidence and outstanding production/user gates.
- [ ] Audit #128/#137–143/#147 against current code and their full recorded acceptance. Close only verified completion or explicitly superseded scope, with all residual work mapped to surviving owners. #200 creation alone closes nothing.
- [ ] Run full affected web suite/build/copy gates and review final diff; push scoped branch and PR linked to #200. Existing user authorization includes merge/deploy to the established non-production demo; require exact-head CI/UI gates and record deployed SHA. Do not change production bank configuration or send real-recipient mail for this navigation slice.
- [ ] Post a precise partial closeout receipt to #200; leave its broader IGX-00/09 boxes open and name the next bounded gap. Full master closure requires the remaining domain and production evidence.
