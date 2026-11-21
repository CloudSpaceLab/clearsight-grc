# V2.14 — Operational risk presentation

## Scope

This change extends the shared premium design to operational registers and their adjacent inspectors. It uses the existing semantic palette, spacing, radii, typography and density preferences. The presentation adapter is confined to `assets/premium-risk.js` and `assets/premium-risk.css`. A final authorized terminology cleanup is recorded separately below.

The source audit found fixed light colours in the residual-risk matrix, very small fact and source text, a horizontally scrolling causal statement, a seven-control loss filter row, and inconsistent register, cycle and catalog spacing. The update addresses those reading and navigation problems without introducing another operational model.

## Route coverage

| Route | Presentation changes |
| --- | --- |
| `risks` | Clear separation of current risk exposure and period loss measures; readable branch rows; theme-aware matrix cells, numeric counts and distinct axis text; consistent register metadata. |
| `risk` | Fact strip, statement, causal explanation and assurance context use a consistent reading rhythm. Cause, event and impact stack at narrow widths. Existing next actions retain their source explanation. |
| `risk-edit` | Comfortable draft fields, long-text spacing, change-comparison reading region and normal-flow actions. |
| `rcsa` | Quiet cycle cards with clearly differentiated counts, population context, progress and review entry. The existing submitted and independently approved descriptions remain separate. |
| `rcsa-review` | Readable population selector, first-line assessment fields and saved/proposed comparison. The source and reviewer decision remain separate. |
| `indicators` | Clear names, metadata and aligned numeric observations within the existing responsive register. |
| `indicator` | Dated history has readable labels and values, a named keyboard-accessible scrolling region, and intact source/rationale rows. |
| `losses` | Grouped filters wrap into three, two or one columns as space reduces. Accepted gross, verified recovery and unrecovered balances keep distinct labels and readable amounts. |
| `controls` | Consistent implementation register, record metadata, draft rows and empty states. |
| `control` | Implementation facts, mappings, source rows, test history and independent conclusions use a generous reading layout. |
| `control-editor` | Draft-state copy, requirement/risk choices, implementation fields and source comparison use the same hierarchy as the detail view. |
| `control-test` | Existing review presentation is retained; source facts, current/proposed effectiveness, draft fields and actions receive the operational reading treatment. |
| `kri-returns` | Quiet reporting-cycle entry, readable population register, concise lifecycle/source information and wrapped cycle controls. |
| `kri-library` | More legible question definitions and interpretation columns across the existing 51-question library. |
| `vendors` | Consistent service register and onboarding catalog entry; theme-aware dossier-step cards and metadata. |

KRI capture, loss intake/detail, vendor intake/detail, assessment response, Documents and the isolated external recipient package remain under their existing presentation modules. This adapter does not claim those routes or alter their HTML.

## Implementation boundary

`pr214Refresh()` sets `#main-content.dataset.pr214` to the exact assigned route and removes it on other routes. A single render wrapper calls it after the canonical render. There is no startup render, additional store, operational event listener or copied record.

The only semantic additions are missing accessible group/region names and keyboard entry for existing matrix, source-comparison and observation regions. Existing `role`, `aria-label`, `aria-labelledby` and `tabindex` values are preserved. Source fields, IDs, names, draft keys, `data-*` actions, links and route query parameters are untouched.

The read-only integration API is:

```js
window.CLEAR_SIGHT_PREMIUM_RISK
// { version: '2.14', presentationOnly: true, routes, refresh }
```

Load the JavaScript after the existing render chain and the stylesheet after the shared foundation and V2.13 form styles. Both the API and route list are frozen. The module coexists with the existing `data-pf213` family on `control-test`.

## Data meaning and responsive rules

The matrix retains all 25 original buttons, likelihood/impact coordinates, counts, score text, filter actions and score bands: 1–4, 5–9 and 10–25. Theme colours derive from the existing teal, amber and red tokens. The update does not infer trends or change appetite thresholds.

Dated indicator bars retain their exact inline heights, observation dates and values. Undated legacy values remain excluded by the canonical source view. Loss amounts, financial inclusion, recovery verification and accepted/proposed review states are unchanged.

Ordinary registers retain the foundation’s mobile card layout. Only existing source-comparison tables keep a horizontally readable table with headers. The matrix retains at least 44 px cells and can scroll within its named region. Wide causal statements, inspector columns and RCSA layouts stack as space reduces.

Compact spacing applies only at widths of at least 1024 px with a fine pointer. Phone and coarse-pointer native fields use 16 px text and at least 44 px controls. Action groups stay in document flow; the module adds no fixed or sticky bar. Forced-colour borders and print overflow rules are included.

## Verification

`node tests/premium_risk_v214_checks.js` passes **8 of 8** focused checks, covering **15 routes and 18 route variants**. Results are saved in `tests/premium-risk-v214-results.json`.

The checks use the integrated source runtime and verify route markers/reset, unchanged HTML and domain/storage state, actual draft fields and registered commands, matrix values/bands, dated observation scales, financial summary labels, denied/out-of-scope/missing-record states, and preservation of existing accessible attributes. They also check stylesheet scope, mobile/coarse-pointer rules and the absence of introduced fixed/sticky positioning.

Two targeted inherited suites also pass with this module loaded: operational interactions **16/16**, and V2.13 premium forms **6/6**. Their source domains and recipient HTML equivalence remain intact.

The runtime harness substitutes the DOM shell. These are source and interaction checks, not browser layout, device or screen-reader verification. No browser was opened or attempted.

## Bounded document-label cleanup

A final live-output audit found a small number of labels still presenting evidence as a separate file workspace or answer type. The following exact public literals were changed in their canonical source functions with root authorization:

| Canonical source | Previous literal | Replacement |
| --- | --- | --- |
| `workflow-completion.js`, `rcsaRespondent` | “Drafts are retained as you type. A submission pins the control definition and evidence version.” | “Drafts are retained as you type. A submission pins the control definition and document version.” |
| Same function, picker label | “Supporting evidence” | “Supporting document” |
| Same function, picker placeholder | “Choose evidence” | “Choose a document” |
| Same function, empty-state guidance | “No current evidence maps to this implementation. Add and review evidence, then resume this saved draft.” | “No current documents map to this implementation. Link and review a document, then resume this saved draft.” |
| Same function, workspace link | “Evidence workspace” | “Documents” |
| `app.js`, locked `rcsaRespondent` submission label | “Evidence” | “Supporting document” |
| `app.js`, `inspectQuestion` | “Default evidence” | “Default document requirement” |
| `forms.js`, `FA_TYPES.evidence` | “Evidence reference” | “Document reference” |
| `experience.js`, active builder `CX_TYPES.evidence` | “Evidence reference” | “Document reference” |
| `loss-register.js`, `l29Summary` | “Source evidence” | “Source document” |
| `loss-register.js`, `l29EvidenceLinks` | “Evidence [version id] is unavailable to the current reviewer.” | “Document version [version id] is unavailable to the current reviewer.” |

The backing `evidence` type, field names, compatibility routes, source IDs and stored histories remain unchanged. Seeded or user-authored question text, retained rationales, working-paper references and contextual explanations about supporting evidence were not rewritten. One consolidated source assertion covers the live RCSA picker, submitted/approved RCSA source labels, canonical question inspector, active form-builder type label and conditional loss-document output, including preservation of canonical keys and source text.
