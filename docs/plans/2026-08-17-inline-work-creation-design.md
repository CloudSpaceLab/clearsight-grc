# Inline issue and change creation

**Status:** Approved product direction  
**Date:** 2026-08-17

## Decision

Add a single-screen, inline creation flow to the **Issues and changes** workspace. The flow uses the existing authority-checked `matter.create` command and opens the new record in the current list after creation. It does not use a modal, wizard, API editor or generic schema builder.

The primary action is **New issue or change**. The form asks only for the facts needed to create a useful Draft record:

- type;
- title;
- summary;
- affected area;
- priority;
- optional due date;
- optional Program link;
- optional known information and information still needed.

The signed-in user becomes the initial accountable owner. Tenant and actor remain server-bound. Access defaults to internal, and the user cannot enter arbitrary tenant, legal-entity, owner or authority identifiers.

## Options considered

1. **Inline single-screen form — selected.** Fastest for routine work, consistent with Program setup and preserves list context.
2. **Multi-step wizard.** Useful for unusually complex cases, but adds navigation and interruption cost to the normal path.
3. **Separate form for every work type.** Can capture deeper specialist detail, but creates an inconsistent and expensive first-run experience. Type-specific follow-up can be added after the initial record exists.

## Manual work types

The form exposes business-created work only:

- Risk issue;
- Control gap;
- Regulatory change;
- Audit finding;
- Supervisory finding;
- Authority request;
- Exception;
- Incident;
- Operational loss;
- Data breach;
- Vendor issue;
- Customer concern.

System-derived types such as failed verification, conflicting evidence, KRI breach and overdue obligation are not manual choices. Those labels must remain attached to the observation or rule that produced them.

## Interaction

1. The user selects **New issue or change** from the workspace header or empty state.
2. The inline form appears before search and filters. Focus moves to the work-type field.
3. Optional Program choices load from the user's current scoped Program summary endpoint. Program loading failure does not block creation; the form explains that linking can be done later.
4. Client and server validation keep the form open and show one actionable error without clearing entered values.
5. On success, the created aggregate is inserted at the top of the current list, opened, and announced as **Issue or change created.** The creation form closes.

## Content contract

- Heading: **New issue or change**
- Supporting copy: **Record what needs attention and when it is due.**
- Title label: **Title**
- Summary label: **What happened or changed?**
- Scope label: **Affected area**
- Known-facts label: **What is already known?**
- Missing-facts label: **What information is still needed?**
- Submit action: **Create issue or change**

Copy describes the user's work. It does not discuss dashboards, records as product architecture, demo mechanics or how the interface was designed.

## Data mapping

The client maps the form to `CreateMatterInput` as follows:

- `type`: selected canonical Matter type;
- `priority`: integer 1–5;
- `title` and `summary`: trimmed user text;
- `scope`: `{ access: "INTERNAL", area: <affected area> }`;
- `known_facts`: `{ notes: <known information> }` when supplied, otherwise `{}`;
- `missing_facts`: non-empty lines from the information-needed field;
- `contradictions`: `[]`;
- `owner_principal_id`: current verified actor;
- `due_at`: local date converted to an unambiguous ISO instant at the end of that local day;
- `program_id`: optional selected scoped Program.

The command middleware rebinds tenant and actor from verified identity and performs current authority resolution before the handler executes.

## State matrix

| State | Required behavior |
| --- | --- |
| Closed | Existing workspace remains unchanged; primary creation action is visible. |
| Opening | Form appears inline and receives keyboard focus. |
| Program choices loading | Form remains usable; Program control reports loading. |
| Live | Required labels are visible; one primary submit action is present. |
| Validation error | Entered values remain; the first invalid field can be corrected. |
| Command denied/unavailable | No record is claimed; returned safe error is shown inline. |
| Success | New item appears first, opens in place, and success is announced. |
| No existing work | Empty state offers the same creation action. |
| Mobile/200% zoom | Fields become one column; actions remain at least 44px and no horizontal overflow occurs. |
| Reduced motion | No non-essential transition is required to complete creation. |

## Acceptance evidence

- Component tests prove opening, cancelling, required fields, exact command payload, Program linking, error preservation and successful handoff to the list.
- Existing exact-target and workspace tests remain green.
- Strict TypeScript, copy-quality, axe/rendered-state tests and production build pass.
- Rendered review covers desktop and mobile in light and dark presentation where supported.

