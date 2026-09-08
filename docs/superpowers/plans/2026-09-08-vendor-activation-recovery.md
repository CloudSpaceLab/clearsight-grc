# Vendor activation recovery implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Prevent stale activation eligibility or late responses from authorizing the wrong on-screen relationship, and provide an executable refresh after a rejected activation attempt.

**Architecture:** Keep the existing activation API and current server-side policy/authority gates. Use one guarded read path for initial load and retry; invalidate pending reads/commands on relationship scope/version changes and unmount. After a command rejection discard eligibility and show Reload activation checks. The subsequent command uses the refreshed result's relationship version rather than an obsolete parent prop.

**Tech Stack:** Existing React/TypeScript, vendor API, Vitest and browser evidence; no backend contract, dependency or new cache.

## Evidence and boundaries

`VendorActivationPanel.tsx` currently leaves `eligibility.eligible` true after conflict/validation errors, while the only reload control is hidden in the unavailable state. Its inline retry read also lacks the initial effect's obsolete-response guard. `activate()` always sends the parent relationship version and can apply a late result after the selected relationship changes. These are read/recovery defects; the server's material version and authority guards remain correct and must not be weakened.

The approved next-record handoffs require access-filtered server targets and remain a separate contract task. This recovery correction does not infer or construct those links from arbitrary gate codes.

## Task 1: Regression and minimal recovery

Files: `web/src/components/VendorActivationPanel.tsx`, `VendorActivationPanel.test.tsx`; `web/src/vendorApi.ts` only if an optional AbortSignal is needed; parent `VendorsWorkspace.tsx` only to synchronize a refreshed relationship using an explicitly named read callback.

- [ ] Add tests with `ApiError(409, ...)` and `ApiError(422, ...)`: start eligible, enter a rationale, reject activation, assert no Ready for authorization or activation control and an enabled Reload activation checks button. Reload a newer eligible relationship and verify the next command uses its returned version.

```ts
vi.mocked(activateVendorRelationship).mockRejectedValueOnce(new ApiError(409, "changed"));
// Render ready, enter rationale, click Activate, await rejection.
expect(screen.queryByText("Ready for authorization")).toBeNull();
expect(screen.queryByRole("button", { name: "Activate vendor relationship" })).toBeNull();
expect(screen.getByRole("button", { name: "Reload activation checks" })).toBeTruthy();
```

- [ ] Add deferred-promise tests: initial error → retry A pending → rerender relationship B → finish B → finish A; only B may appear. Repeat with a late activation success/error, unmount and an ACTIVE relationship. Mismatched returned ID/tenant/entity is rejected. Double activation before rerender issues one command.
- [ ] Run `node node_modules/vitest/vitest.mjs run src/components/VendorActivationPanel.test.tsx` and confirm each new failure comes from the missing recovery/invalidation behavior.
- [ ] Consolidate reads around one effect-driven reload counter or guarded read function; all state writes require the current scope/generation. Clear stale eligibility before loading or exposing recovery. Preserve rationale for same-relationship retry, clear it when the relationship changes. Guard duplicate activation synchronously. Do not automatically retry a material command.
- [ ] Use the freshly loaded `eligibility.relationship.version` for `expected_version`. If a read finds the relationship already ACTIVE, update the displayed state without calling a callback whose name claims a new activation command occurred. Keep parent relationship data synchronized through a read-specific callback when required; reject late callbacks from another selection.
- [ ] Make all error copy truthful: a network error cannot assert that the server made no change. Say activation could not be confirmed and require refreshed checks. Conflict and validation recovery must not describe stale checks as current. Add the human label for Decision authority while reviewing the complete affected panel.
- [ ] Run the focused test, affected VendorsWorkspace tests, copy-quality and typecheck. Commit only verified scoped files.

## Task 2: Review and release evidence

- [ ] Add a deterministic fixture for conflict → reload and late-result isolation using the existing browser harness; exercise light/dark desktop and 390/320px. No fake actual vendor approval or scan receipt.
- [ ] Specification review then quality review; fix and re-review findings.
- [ ] Run full release gates with the navigation tranche after both pass focused checks. Record exact commands, renders, head and hosted revision under #139/#147/#200. Keep exact-record handoffs and full vendor outcome acceptance open.
