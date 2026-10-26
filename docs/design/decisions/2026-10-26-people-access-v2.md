# People & access — original HTML layout adaptation

Date: 26 October 2026. Tracking: #462. Status: implementation in progress; partial v2 layout integration. Whole-screen acceptance remains 0/19.

## Authority and starting point

Layout authority: `docs/design/references/clearsight-2026-10/source/clearsight-people-access-designs/people-access-preview.html`, SHA-256 `3300853e7959e06f76e52b484e77c71c6f0c2af876351861921e0dead0aaa7f4`, with the immutable companion CSS, JavaScript and notes. Shared navigation follows the 21 October decision; shared tokens and controls follow `DESIGN.md`.

Starting runtime: main `ea84283f913755be6dc1b8e366282469745b3e9a`, tree `28f0b24ef1da098b46bfd345c466cecd79051675`. Tabs #587 and Privacy #588 are merged, all exact-main checks passed and the demo deployment verified this SHA. Existing Configuration puts an additional 230px navigation rail and repeated headings before organization/provisioning cards; its People inventory is truncated again to eight rows. Governance policies and delegations appear in tall consecutive inventories below routing checks.

## Page composition and ownership

Adapt the full People & access heading, four local tabs, compact summary strip, searchable directory, selected-person context, role list/detail, routing-policy list/detail and delegation list/detail. The source's two-column list/context composition becomes stacked at 1024px; the compact four-fact strip becomes two columns. Retain shared keyboard, touch, focus and dark-theme behavior. Review 1440, 1280, 1024, 768 and 390px.

| Destination | Existing owner and retained behavior |
| --- | --- |
| People — `#configure/access` | `IdentityAccessPanel`, authorized access overview, bounded people, profile/source state, recorded current-entity positions |
| Roles — `#configure/access/roles` | Same access overview, real role definitions/capabilities and recorded mappings; role membership remains separate from authority |
| Approval routes — `#configure/authority` | `GovernanceAdminPanel` and existing versioned policy draft/submit/approve/return/retire commands |
| Delegations — `#configure/authority/delegations` | Same governance owner, authority-returned candidates, independent checker and effective dates |
| Organization settings | Existing positions, reporting lines, directory provisioning/mappings and escalation configuration; secondary destinations under the same workspace |
| Routing checks — `#configure/authority/checks` | Existing integrity and affected-work checks, reachable from Approval routes |

Settings stays in the common footer. The specialist workspace uses a Settings return action instead of the extra Configuration rail. `#people/:id` remains the distinct employee work/activity record. Selection and browser history must not replace the mounted directory or lose its filters. Scope changes remount the workspace by verified tenant, legal entity and actor; reads wait for verified runtime context.

## Data and command boundaries

- `/api/v1/access/overview?limit=50` returns limited tenant-wide people/groups and entity-scoped positions/bindings. Label returned-record counts and local search coverage; do not call this the complete current-entity directory. Do not infer group membership or complete effective access from role templates or positions.
- Principal profile state and provisioning-source state are separate. Neither alone proves sign-in eligibility or current command authority. `user_name` is a username, not necessarily an email address. A missing source code is labelled "Source not returned" because it does not prove local origin.
- Policy/delegation lists request 51 records, display at most 50 and disclose additional records without inventing a total or pagination. Stored lifecycle/effective dates are not proof of a currently resolved approval path.
- The existing policy list already returns the current revision's `definition` and `checksum`. Preserve both and lead selected-policy context with actual rule facts: responsibility, selector or lifecycle condition, entity/object/decision scope, materiality and priority. Full returned JSON remains inspectable, including additional settings. Missing or unreadable definitions cannot enable approval. Preserve versions, makers/checkers, effective windows and latest decisions, with exact times and an explicit timezone in the inspector. Original sequential approval-stage illustrations are not fabricated from stored rule order; actual resolution still belongs to the authority service.
- Classify access denial separately from successful empty results. Clear denied populations; transient failures may retain explicitly stale same-scope records while all affected material commands stay disabled.
- Existing draft, candidate search, maker-checker, expected-version and rationale contracts remain authoritative. No identity, authority, routing, invitation or audit store is added.
- Invitation delivery/acceptance, suspension, impersonation, last-active history, role-holder totals, source route-gap counts and unsupported simulation controls remain excluded. Use supported configuration actions in their appropriate region.

## Verification and completion gates

Freeze a before build before runtime edits. Retain original and runtime same-screen pairs for all four tabs at all five widths, plus light/dark runtime states. Assert actual heading/tab/control/list/detail geometry and first useful records, not DOM order alone. Cover keyboard selection, narrow-screen detail access, tab/history retention, query reset, no results, missing selection, loading/retry, denied/partial/read-only states, bounded inventory, stale-response scope changes and no page horizontal overflow.

Run affected identity/governance/routing/copy tests, fixture-boundary/type/build/bundle gates and the existing hosted UI/UX suite. A named independent agent must inspect the exact runtime and original images; document source/font and supported-data adaptations. Commit and push in small chunks. Final release receipts belong to the PR and evidence index. Local fixtures and visual review do not establish four-actor execution, live backend authorization, source-to-report reconciliation or production acceptance.
