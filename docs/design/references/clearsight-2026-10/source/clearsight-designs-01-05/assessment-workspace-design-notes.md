# ClearSight — Assessment & Questionnaire Workspace

Design 05 | 15 October 2026 | desktop 3760px-wide, mobile 390 CSS pixels at 2x.

## Operator flow

1. Open assigned assessment under its existing Program (ISO/IEC 27001 used as illustrative reference only).
2. Choose a section, review prior answers or source-backed suggestions, then confirm or change answers.
3. Record negative answers with a follow-up note; attach evidence separately. A submitted response does not establish compliance or evidence acceptance.
4. Import a spreadsheet; inspect the original sheet and cell mapping before choosing which answers to use. Blank cells remain unanswered.
5. Review outstanding suggestions, control gaps and missing evidence before requesting independent assessment review.

## Visual and interaction principles

- Reuse ClearSight's approved shell, typography, neutral canvas, subtle blue wave and shared button/card anatomy.
- Three-column desktop: sections, working form, contextual evidence/readiness. Tablet stacks context; mobile makes the section rail horizontal and keeps question controls full-width.
- Progress counts answers, not compliance. A negative response may be answered and still require follow-up. Source suggestion is not applied until the user confirms it.
- The evidence sidebar is functional, not a decorative AI marketing panel. Compact orb is context only and never animates perpetually.
- Distinguish draft state, completed answers, evidence availability and review approval. Use concise copy and one dominant action.
- Keep all user-controlled illustrations in a prototype-specific layer; production implementation should reuse existing Forms, Program, Response and Evidence components.

## Prototype scope

- Sample question data and mappings are local; no network, persistence or backend integration.
- Answers, sections, suggested answers, sample Excel mapping selection, review popover/drawer and three contextual suggestions are interactive.
- File chooser accepts a local file but **does not parse or upload it**; a selected filename is shown with a precise limitation.
- Submission is a preview confirmation and does not change any ClearSight record.
- The screenshot's names, statuses, deadlines, controls and source dates are design fixtures—not evidence of real compliance.

## Production acceptance contract

- Honor actor/tenant/entity/group/Program scope; users cannot adopt another owner's answer without confirmation.
- Preserve original form/response revisions, governing policies, per-field provenance, blank vs no, evidence scan status and reviewer decision.
- Excel extraction must preserve sheet, coordinate, original raw value, mapping confidence and explicit operator decision; no implicit controls passed.
- Cross-screen filter and resume state must round-trip without accidental data reclassification.
- Review step must show actual authority route and outbox receipt; denial/stale/partial/unknown states require truthful recoverable UI.
- Keyboard, focus return, actual dialog focus containment, touch targets, reduced-motion and visual regression are implementation acceptance gates.

Next screen candidate: **Vendor onboarding & compliance workspace** or **Forms Library and template builder**.
