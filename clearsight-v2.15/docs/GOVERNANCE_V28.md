# V2.8 — Roles, scoped access and escalation

## Delivered workspaces

- `roles`: reusable role catalogue with active assignment counts and access-review indicators.
- `role-detail`: editable 48-cell module/action matrix, maximum organizational scope, descendant control, purpose, review cadence and change rationale. The protected workspace-administrator role can be duplicated but not modified.
- `access-review`: scoped dated assignments, recertification/revocation, independently reviewed temporary direct rules, an effective-access explanation and access-register export. The audit tab retains local change receipts.
- `escalation-matrix`: configured trigger/severity/scope response policies and a timing/routing simulation.
- `escalation-policy`: full policy editor with one to six timed levels, role/owner recipients, explicit fallback, clock choice, policy owner and stopping condition.

## Access semantics

`S.accessControl` owns these fixtures and configuration drafts; the pre-existing `S.governance` document model is unchanged.

An access decision requires a known person and location, a matching active role assignment or direct rule, a valid date window, and a location inside both assignment and role boundaries. Explicit deny overrides every grant. Unset does not grant permission. The approval decision also rejects a person approving their own submission.

Role saves and new assignments check preparation/approval conflicts, including overlapping future assignments. Direct grants check all covered descendant locations and future assignment start dates inside the exception period. Direct rules expire within 90 days and need an independent person with access-management permission. Expired assignments grant nothing and cannot be silently reactivated through recertification. The last active organization-wide protected administrator cannot be revoked.

Roles and assignments remain separate: updating a role does not reassign people or silently widen their organizational locations. Drafts are saved locally; operational permissions change only when the role configuration is explicitly saved.

This is an interactive policy reference and evaluator. Existing domain actions are not secured by this browser model. Production must enforce authorization and independent decisions on the server using authenticated users, tenant boundaries, object-level permissions and version checks.

## Escalation semantics

Policies match trigger, minimum severity and a covered organizational location. `matchPolicy` selects the most specific location first, then the highest matching severity threshold; exact duplicate active selectors are rejected by the editor.

Level offsets are cumulative from the original trigger and must strictly increase. A primary role resolves to active assignments covering the event location at that level's planned send date. An absent owner or empty role resolves to the configured fallback. An empty fallback remains visibly **Unassigned** and names the policy owner; the simulation never treats it as delivered.

Calendar time operates 24/7. The business clock is Monday–Friday, 08:00–17:00 Africa/Lagos (UTC+1). No holiday calendar is configured. A business-hours trigger outside the response window begins at the next business opening. Acknowledgement stops later levels only when the policy says so; source resolution always stops later levels. Each level sends once in the intended production contract.

The simulator performs no notifications or external messages. It exports its planned times, resolved recipients, routing gaps and stopped levels. Production delivery needs source-status rechecks, idempotent delivery, retry/dead-letter handling, current role resolution, a tenant calendar and an external scheduler.

## Public inspection API

```js
CLEAR_SIGHT_GOVERNANCE.getState()
CLEAR_SIGHT_GOVERNANCE.evaluate(userId, permission, scopeId, date, {makerId})
CLEAR_SIGHT_GOVERNANCE.matchPolicy(trigger, severity, scopeId)
CLEAR_SIGHT_GOVERNANCE.simulate({policyId, scopeId, ownerId, at, severity, ackAfter, resolvedAfter})
CLEAR_SIGHT_GOVERNANCE.businessTime(startISO, hours)
```

`at` is a WAT local date-time string (`YYYY-MM-DDTHH:mm`). Empty acknowledgement/resolution delays mean those events have not occurred. Nonempty delays are calendar hours from the initial trigger, regardless of the selected policy clock.

## Integration

Load `assets/governance.css` and `assets/governance.js` after records/documents and before the final bootstrap. The module registers five routes in `CS_JOURNEYS`, associates them with Settings and adds discoverable access/escalation links to Settings. It does not perform an initial render.

## Validation

`node --check assets/governance.js`

`node tests/governance_v28_checks.js`

Fourteen model/render-contract checks passed. They cover effective scope, missing grants, expired and future access, explicit deny, self-approval, role lifecycle, last-admin protection, assignment boundaries, independent direct grants, descendant conflicts, business/weekend timing, actual recipient resolution, fallback gaps, stopping conditions, policy matching, versioned changes and complete rendered action wiring.

Chromium could not launch in the current execution environment: the browser process was denied creation of its required socket. These tests do not claim browser screenshot, keyboard, screen-reader or visual-layout validation.
