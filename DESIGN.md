# ClearSight interface contract

This is the fast, implementation-ready design contract for people and coding agents. Canonical product and safety semantics remain in `docs/`; this file defines how those semantics should appear and behave in the interface.

## Product and users

ClearSight is repeated-use operating software for bank executives, risk and compliance leaders, reviewers, authorizers, business owners, administrators and invited respondents. It should feel calm, exact, premium and institutional—not theatrical, playful or generic.

The interface optimizes for:

- rapid scanning of real work;
- clear ownership, authority and deadlines;
- minimum-question review and capture;
- visible source, evidence and uncertainty;
- safe interruption, resume and handoff;
- explicit outcome confirmation before closure.

## Visual direction

Use restrained institutional futurism:

- deep, low-noise dark surfaces and a clear neutral light equivalent;
- fine borders and subtle depth;
- cyan for navigation and active information;
- violet for governance/configuration context;
- amber for pending review or attention;
- green for a supported current state;
- coral for blocking gaps or failed outcomes;
- premium vector illustrations only for orientation, education, onboarding and genuine empty states.

Avoid decorative gradients, glass, glow or illustration where they compete with status, evidence or decisions.

## Theme and density tokens

Operational components consume semantic tokens rather than owning separate light/dark palettes. The canonical roles are:

```css
--canvas;
--surface;
--surface-2;
--surface-3;
--border;
--border-strong;
--text;
--text-strong;
--muted;
--cyan;
--violet;
--amber;
--green;
--coral;
--focus-ring;
```

`web/src/ui-preferences.css` owns the current dark/light mappings and the illustration token mappings. Components must not duplicate semantic colors locally when a token already represents the meaning.

Theme preference supports **System**, **Light** and **Dark**. Density supports **Comfortable** and **Compact** for repeated desktop work. Compact density may reduce desktop row/control spacing but must not reduce mobile/touch targets below the supported interaction size.

Typography uses Inter, Segoe UI Variable, Segoe UI, then system sans-serif. Headings use tight tracking; operational copy uses normal sentence case. Uppercase is limited to compact metadata labels.

Use an 8px spacing rhythm, 11–18px controls, 12–18px cards and 20–26px hero/guide radii. Shadows and blur remain subtle and never carry state.

## Structural patterns

- **Intervention Summary:** actor-scoped read projection for one human review, decision, authorization, evidence exception, escalation or outcome check. It is not new authoritative state.
- **Today:** intervention queue first; quiet status-check context follows the work rather than preceding it with a KPI wall.
- **Programs:** ongoing responsibilities, current status and reasons. Show the status reason before the complete requirement/evidence catalogue.
- **Vendors:** one legal-entity-scoped register and selected service relationship. Keep due-diligence setup, secure collection status and reviewer evidence in that relationship context; open canonical issues and changes for remediation instead of creating a second vendor dashboard or findings system.
- **Issues and changes:** bounded items needing review, decision, action, response or outcome confirmation. Show the current handoff before history.
- **Work:** review queues and focused evidence. Complete source inventories are secondary context.
- **Configure:** policy, routing, integrity and ownership.
- **Side panel:** bounded inspection or one focused action without losing list context.
- **Dedicated page:** complex or protected work requiring several sections, parallel work or a durable saved state.

Do not default every concept to a dashboard card. Choose lists, rows, details, tables, timelines or focused forms according to the operator's task.

### Progressive disclosure for governed work

Use the same reading burden across operational surfaces:

1. **Queue:** human gate, material conclusion, scope, evidence state, deadline and next action.
2. **Current handoff:** what changed or why the current state needs this actor.
3. **Review context:** evidence, contradictions, alternatives, decisions, actions and verification.
4. **Reconstruction:** complete Program/Matter, source lineage, imported material, operator receipt and immutable history.

Complete context must remain reachable, but it must not be the default reading burden. Do not relabel ordinary status data as AI or automated work; show operator/prepared-work claims only when a governed receipt exists.

## Copy

Primary screens use familiar working language. Internal codes remain available in APIs, audit views and specialist detail.

Examples:

- `EVIDENCE_INSUFFICIENT` → **Evidence incomplete**
- `DECISION_REQUIRED` → **Decision needed**
- `VERIFICATION` → **Confirming outcome**
- `MATTER` → **Issue or change** on general screens
- `CONTROL_IMPLEMENTATION` → **Safeguard** on business-owner screens

Keep useful product nouns when they already describe the job clearly. **Today**, **Programs**, **Work**, **Evidence**, **Imports** and **Configure** are practical navigation labels; do not replace them with generic phrases such as “Your work” merely to sound conversational.

Every page answers: what is shown, current state, why now, owner, next action, source and time. Never replace an unknown population with sample or persuasive numbers.

Authority copy follows the same rule: the interface may show only roles, stages, limits and explanations returned by the authority service or other canonical records. It must not invent a familiar approval chain when the runtime has not returned one.

## Inputs and capture

Ask for the smallest fact the user actually needs to supply. If the request already knows an ATM identifier, address, vendor, legal entity, date range or other context, render it as read-only context instead of asking the respondent to type it again.

Choose controls by the fact being captured:

- **Yes/no or 2–4 choices:** large tap/radio choices, especially on mobile and external capture.
- **Short names, identifiers and one-line facts:** single-line text input.
- **Explanations and exception notes:** textarea; do not use a textarea for ordinary short text.
- **Dates:** native date control unless the domain requires a richer scheduling control.
- **Numbers:** numeric input with the appropriate input mode and server-side range validation.
- **Signature/attestation:** a bounded signature control attached to explicit attestation copy; signing does not replace the factual fields being attested.
- **Files and photos:** use the upload pattern appropriate to the importance of the artifact rather than defaulting every attachment to the same large box.

### File-upload patterns

Use one shared dropzone interaction with three presentation levels:

1. **Photo evidence:** prominent drop/tap area, camera-first on supported mobile devices, immediate thumbnail preview, filename/size, and replace action. The factual confirmation remains a separate field.
2. **Document import or another file-primary task:** large dropzone with filename/type/size shown before the explicit import/submit action. Selecting or dropping a file must not silently commit it.
3. **Incidental general attachment:** compact dropzone/file row with filename/size and replace action. Do not spend a large part of the form on an optional PDF or supporting file.

Do not use a dropzone for settings or forms where the file is incidental to a different control. Do not enable multiple files unless the request contract explicitly permits multiple artifacts and defines how each is reviewed.

Image preview is appropriate for image evidence. For PDFs, Office files and other documents, show trustworthy metadata before upload; do not fabricate a document preview before extraction/rendering has actually succeeded.

### Vendor due diligence

The Vendors workspace uses one dominant action for the current assessment state: start setup, send the request, review collection status, begin bank review or record the conclusion. The selected vendor, service, accountable owner, exact form version and review deadline remain visible around that action.

External collection uses the shared request-scoped invitation and capture experience. Known vendor and service facts are prefilled or shown as context; the active form decides between Classic and Wizard presentation and renders typed controls, limits, conditional fields, uploads and attestation. Submission means the response was received, not that the evidence was accepted or the relationship was approved.

Internal review shows only the exact scoped response, answer provenance, coverage, artifact scan state, evidence classification, linked canonical findings and version-qualified provisional score. Reviewer conclusion, vendor-relationship activation and deficiency closure remain separate material outcomes. Completed assessments are read-only in the relationship workspace.

External capture should minimize normal-path typing. A field-visit verification should ordinarily be completable through known context, tap choices, required photo evidence, an optional exception note and an attestation/signature. The interface target is under four minutes for a representative simple visit; that target is not considered proven until a timed usability run confirms it.

## States and recovery

Every significant component and screen defines:

- loading;
- live/default;
- empty for an explicitly named scope;
- stale or partial data;
- unavailable source/API;
- permission denied or wrong scope;
- validation error;
- optimistic conflict;
- success/receipt;
- long content and translated text;
- keyboard, focus, 200% zoom and reduced motion.

A disabled control explains why. A visible enabled control must perform a real action.

## Responsive behavior

Responsive work is replacement, not shrinking:

- desktop may use parallel context, dense rows and side panels;
- tablet reduces simultaneous columns and preserves the next action;
- mobile converts dense rows into stacked summaries and complex side panels into full-screen flows;
- external capture prioritizes one question group, progress, save/resume and safe receipt;
- protected work may intentionally disallow offline or shared-device use.

## Motion

Motion is functional and short: panel entry, expansion, focus, progress and state change. The initiating component owns the motion. Every animation has a reduced-motion fallback and should not delay interaction. No ambient motion around material decisions or alerts.

## Illustration and icons

Illustrations use an editorial, semi-abstract vector language with restrained geometry, soft depth and no mascot personality. They support first-run guidance, empty states, education and completion. Semantic line icons identify recurring object types. Neither replaces labels or status.

Illustration geometry stays shared across themes; palette comes from semantic/theme variables. Each production illustration exposes an accessible title/description rather than a generic unlabeled SVG.

Populated default Today, Programs, Work, Evidence and Configure states do not use decorative hero illustrations. Their primary visual hierarchy comes from the human gate, status, evidence and next action.

## Design proof

Significant UI work requires:

1. a compact decision brief;
2. a before-state baseline when redesigning an existing screen;
3. at least the required state matrix;
4. rendered evidence at representative viewports;
5. one highest-impact repair and re-check;
6. design-token, input and copy review before merge.

Vitest/axe semantic checks are executable CI evidence, but jsdom does not prove visual contrast, 200% zoom, responsive replacement or theme parity. Those remain rendered-browser evidence gates.

See `docs/design/ui-delivery-workflow.md` and `docs/quality/rendered-ui-evidence.md`.
