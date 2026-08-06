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

- deep, low-noise surfaces;
- fine borders and subtle depth;
- cyan for navigation and active information;
- violet for governance/configuration context;
- amber for pending review or attention;
- green for a supported current state;
- coral for blocking gaps or failed outcomes;
- premium vector illustrations only for orientation, education, onboarding and genuine empty states.

Avoid decorative gradients, glass, glow or illustration where they compete with status, evidence or decisions.

## Current tokens

```css
--canvas: #07111d;
--surface: #0d1826;
--surface-2: #132234;
--surface-3: #17293d;
--border: #26394d;
--text: #d7e0e9;
--muted: #8fa0b2;
--cyan: #5bc6d6;
--violet: #9e92f4;
--amber: #e9b75a;
--green: #5bc996;
--coral: #f17a8d;
```

Typography uses Inter, Segoe UI Variable, Segoe UI, then system sans-serif. Headings use tight tracking; operational copy uses normal sentence case. Uppercase is limited to compact metadata labels.

Use an 8px spacing rhythm, 11–18px controls, 12–18px cards and 20–26px hero/guide radii. Shadows and blur remain subtle and never carry state.

## Structural patterns

- **Intervention Summary:** actor-scoped read projection for one human review, decision, authorization, evidence exception, escalation or outcome check. It is not new authoritative state.
- **Today:** intervention queue first; quiet continuous-check context follows the human work rather than preceding it with a KPI wall.
- **Programs:** ongoing responsibilities, current status and reasons. Show the status reason before the complete requirement/evidence catalogue.
- **Issues and changes:** bounded items needing review, decision, action, response or outcome confirmation. Show the current handoff before history.
- **Work:** review queues and focused evidence. Complete source inventories are secondary context.
- **Configure:** policy, routing, integrity and ownership.
- **Side panel:** bounded inspection or one focused action without losing list context.
- **Dedicated page:** complex or protected work requiring several sections, parallel work or a durable saved state.

Do not default every concept to a dashboard card. Choose lists, rows, details, tables, timelines or focused forms according to the operator's task.

### Progressive disclosure for governed work

Use the same reading burden across operational surfaces:

1. **Queue:** human gate, material conclusion, scope, evidence state, deadline and prepared next step.
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

Every page answers: what is shown, current state, why now, owner, next action, source and time. Never replace an unknown population with sample or persuasive numbers.

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

Populated default Today, Programs, Work, Evidence and Configure states do not use decorative hero illustrations. Their primary visual hierarchy comes from the human gate, status, evidence and next action.

## Design proof

Significant UI work requires:

1. a compact decision brief;
2. a before-state baseline when redesigning an existing screen;
3. at least the required state matrix;
4. rendered evidence at representative viewports;
5. one highest-impact repair and re-check;
6. design-token and copy review before merge.

See `docs/design/ui-delivery-workflow.md` and `docs/quality/rendered-ui-evidence.md`.
