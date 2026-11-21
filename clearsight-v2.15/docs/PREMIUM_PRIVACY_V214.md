# V2.14 privacy and monitoring presentation

## Scope and integration

Load `assets/premium-privacy.css` after the V2.13 premium styles and `assets/premium-privacy.js` after the existing presentation adapters, before bootstrap. This module requires the existing `render`, `current` and canonical route handlers. It does not render on load.

The adapter adds its own `data-pv214` marker to `#main-content` after the existing render chain completes. It removes that marker on unowned routes and leaves `data-pf213`, `data-pp214`, `data-cs-route` and other adapters intact. The public presentation API is `CLEAR_SIGHT_PREMIUM_PRIVACY`, with `version`, `family(route)`, `refresh()` and `presentationOnly`.

| Route | Presentation coverage |
| --- | --- |
| `privacy` | Overview, processing register, DPIA register, sharing, cases and saved drafts; legible decisions and state filters |
| `activity` | Record context, published-versus-proposed banner, data flow, purposes/retention, screening, review and history |
| `activity-wizard` | Six canonical stages, pinned draft location, completeness links, table editing and final review |
| `activity-version` | Read-only accepted snapshot, source metadata and historical content |
| `dpia` | Assessment reasoning, privacy harm, initial/residual scores, measure state, decision checks and history |
| `dpia-wizard` | Five canonical stages, source revision, risks, measures, ownership and final recommendation |
| `privacy-case` | Rights request and incident preparation, returned comments, submission state, decision and chronology |
| `case-intake` | Both three-stage intake journeys and partial drafts |
| `integrations` | Actual local-check quality, latest observation, execution time, individual/run-all actions and history |
| `data-health` | Local consistency results, counts, storage limitations and canonical source contracts |

## Presentation decisions

The older privacy layouts used smaller chronology and source labels than the premium workspaces. The new styles use the shared semantic palette and spacing, with opaque reading surfaces, clearer titles and comfortable paragraph widths. Facts, supporting content and independent decision forms have distinct hierarchy without introducing another summary or new workflow step.

Privacy editing tables keep their native table and input structure. Their horizontal scroll regions now have an accessible name and keyboard focus target. Field names, paths, control types, labels, values, disabled states and events are retained. Existing accessible names, roles and tab order are respected.

Step counters use the fixed solid accent for white numerals. Completed counters use the semantic teal surface, correcting the previous pale fallback in dark appearance. The register's existing state bar retains the same values and table-equivalent counts, with Active teal, In review blue, Draft muted and Changes requested amber. These are record states; no regulatory-compliance outcome is inferred.

Data flows become vertical at narrower widths, retaining source order and directional arrows. Review panels become one column before long references and rationale fields become cramped. On mobile and coarse-pointer devices, controls have 44px targets and editing fields use 16px text. The draft footer becomes static on those devices and short viewports so it does not obscure the current field; its real save indicator remains visible. Compact spacing applies only to scanning surfaces at `min-width:1024px` with `pointer:fine`.

Integration cards show only the canonical `Not run` or recorded quality, observation and execution timestamp. No connectivity status, real-time signal, credential flow or monitoring outcome is invented. Data checks retain the distinction between local reference consistency and completeness or truth of organizational information.

The print styles preserve readable source facts and chronology, return table layouts to tables, and remove editor/navigation controls. They do not open disclosures, mutate records or regenerate historical snapshots. Print pagination has not been validated in a browser.

## Authorized terminology corrections

Two exact renderer literals were updated, with no general replacement of user content:

- `assets/journeys.js`, `jDpiaView`: the tab label is **Actions & documents**. Its `actions` key and route remain unchanged.
- `assets/documents.js`, `docDataHealthView`: the exact contextual-review row is **Assessment documents**. The preceding **Documents** row still describes document identity and versions. Backing evidence/document types and references are unchanged.

## Preserved contracts

The new JavaScript does not replace route handlers or domain commands, and does not read or write `S`, storage, permissions, source objects or draft fields. It only marks the current route and supplies missing names/roles for existing action groups, review forms and editing-table scroll regions.

Activity, DPIA and case submission/review gates, actual-actor checks, scope selection, source-revision checks, history, immutable accepted versions and pending changes remain canonical. The adapter does not approve, submit, run checks, modify notifications or infer readiness. Privacy task discovery and the distinction between saving case progress and explicitly submitting for review are retained.

## Verification

`node tests/premium_privacy_v214_checks.js` passes **7/7 presentation contracts** using the actual application source and controlled DOM nodes. Coverage includes 28 read/capture-entry views, all 17 stages across activity/DPIA/rights-request/incident drafts, exact output and saved values, existing command identities, repeated semantic enhancement, marker cleanup/composition, immutable source state, exact Documents labels and actual fixture observation/quality/time.

`node tests/privacy_interactions_v211_checks.js` passes **10/10 existing interaction regressions**, including draft discovery and continuation, validation links, separate save/submit intent, returned comments and one-source-check behavior.

`node --check assets/premium-privacy.js` passes. No browser or browser workaround was used. The source tests establish markup/state/command preservation; they do not establish pixel layout, native keyboard scrolling, assistive-technology behavior or PDF pagination. Result files are `tests/premium-privacy-v214-results.json` and `tests/privacy-interactions-v211-results.json`.
