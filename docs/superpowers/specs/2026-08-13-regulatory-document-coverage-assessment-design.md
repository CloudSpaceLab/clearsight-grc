# Regulatory Document Coverage Assessment Design

**Status:** Approved direction
**Date:** 13 August 2026

## Outcome

After a regulatory document is uploaded and its searchable PDF text is extracted, ClearSight automatically identifies source-backed obligation candidates, compares them with the tenant's current Programs and visible Matters, and presents an explainable estimate of institutional coverage. A reviewer can confirm or reject matches and prepare governed updates without leaving the document workflow.

The workflow reports **verified document coverage**, not legal compliance. Verified coverage means that a reviewed document obligation has a complete current chain:

`approved applicable requirement -> implemented control -> current supporting evidence`

The page also reports requirement mapping, control implementation, and evidence support separately. Open Matters explain known gaps or remediation but never increase a coverage score.

## Problem and root cause

The document-import worker currently extracts searchable PDF pages and applies a bounded keyword/sentence classifier. It creates generic proposals with exact source anchors, but it has no dependency on Continuity and therefore cannot compare proposals with Programs, requirements, controls, evidence contracts, assessments, or Matters. The UI can review extracted text, but it cannot answer whether an obligation is already covered, where the evidence chain breaks, or which governed change is appropriate.

This is an architectural absence rather than a PDF extraction defect. The comparison must be a separate, auditable stage after extraction.

## Considered approaches

### 1. Governed hybrid comparison (selected)

Normalize obligation candidates into structured fields, apply hard scope gates, rank plausible Program and requirement matches with transparent weighted signals, and derive coverage only from deterministic current Program state. Semantic similarity may rank suggestions, but it cannot independently prove coverage. Review decisions are versioned and all material changes pass through existing authority-guarded Continuity commands.

This approach is accurate enough to be useful, explainable to a reviewer, deterministic at the scoring boundary, and deployable without making an external AI service a production dependency.

### 2. Model-first interpretation

Send document sections and current Program records to a language model and use its response as the assessment. This may recognize paraphrases well, but it is nondeterministic, harder to reproduce, can leak restricted context if poorly bounded, adds cost and availability dependencies, and cannot establish that controls or evidence are current. It is unsuitable as the source of a compliance percentage.

### 3. Exact lexical or identifier matching

Compare citations, codes, and normalized token overlap only. This is inexpensive and reproducible, but regulatory obligations and internal requirements are frequently paraphrased. It would generate too many false gaps and weak suggestions if used alone.

## Terminology and truth boundaries

- **Obligation candidate:** a source-backed passage that appears to impose, prohibit, condition, or time-bound an action. It is not automatically an approved requirement.
- **Estimated coverage:** the automated assessment before all proposed mappings are reviewed. It is clearly labelled as an estimate.
- **Verified document coverage:** the percentage of eligible obligation candidates whose accepted mapping has a complete current requirement, control, and evidence chain.
- **Eligible candidate:** a distinct obligation candidate included in the denominator. Low-confidence candidates remain eligible until reviewed; uncertainty must not silently shrink the denominator.
- **Not applicable:** a reviewer decision with a recorded reason. Only this reviewed disposition removes a candidate from the denominator.
- **Matter context:** visible change, gap, finding, or remediation work related to a candidate. Matter existence does not count as implementation or evidence.

No result is labelled `compliant`, `certified`, or `legally sufficient`. The UI states the document version, assessment time, Program snapshot, and review state near every percentage.

## Architecture and component boundaries

### Document extraction

`internal/documentimport` remains responsible for storing the source, extracting bounded page text, retaining page anchors, and detecting raw proposal/candidate passages. It does not read or mutate Continuity records.

The analyzer is extended to emit a versioned structured obligation candidate while retaining the exact quote and page anchor:

- modality: mandatory, prohibitive, conditional, deadline, reporting, or uncertain;
- actor or regulated party;
- normalized required action;
- subject or object of the action;
- regulator, jurisdiction, and explicit citations;
- dates, frequencies, and thresholds where present;
- normalized topic terms;
- extraction confidence and reasons for uncertainty.

Statements that are merely descriptive, aspirational, definitions, headings, or references remain visible as extracted text but do not become eligible obligations unless reviewed into scope. Duplicate obligations are grouped using normalized content and citation, while every source anchor is retained.

### Coverage assessment

A focused `internal/documentcoverage` package owns comparison and assessment. Its public service consumes:

1. an extracted document and its structured candidates;
2. an authorized, read-only Continuity snapshot for the same tenant and legal-entity context;
3. a versioned matcher configuration; and
4. the actor performing or requesting review.

The package has isolated interfaces for candidate normalization, candidate retrieval, match ranking, deterministic chain evaluation, persistence, and suggestion generation. This keeps extraction, matching, governance mutation, and UI projection independently testable.

Coverage processing is a durable asynchronous stage triggered after successful extraction. It never blocks the upload receipt. Its states are `PENDING`, `COMPARING`, `READY`, `PARTIAL`, `FAILED`, and `STALE`.

### Continuity snapshot

The comparison snapshot contains only Program records authorized for the tenant and legal-entity context:

- active and draft Programs and their current versions;
- approved applicable requirements;
- requirement-control links and current control implementation state;
- evidence contracts and current evidence assessments;
- source and applicability metadata.

Draft Programs can be suggested as context but cannot contribute to verified coverage; retired Programs are excluded. Matter comparison is a separate actor-filtered projection resolved when an authorized user reads or reviews the assessment. Restricted Matter identifiers, titles, descriptions, actions, and decisions are never copied into the shared assessment. This keeps the Program-based percentage stable between actors while still showing each actor existing Matters they are allowed to use or extend.

## Matching and scoring

### Scope gates

A candidate cannot match a Program when explicit scope conflicts exist. Hard conflicts include tenant, legal entity, jurisdiction, regulator or regulated-party class, retired Program state, and incompatible applicability. Unknown scope lowers confidence; it does not become an invented match.

The live demo's current Nigeria privacy Program must therefore not appear to cover US Federal Reserve or UK Bank of England obligations merely because generic security terms overlap.

### Candidate retrieval and ranking

For candidates that pass scope gates, the default matcher ranks a bounded set of possible requirements with explainable signals:

- citation, instrument, section, or requirement-code agreement: 35%;
- normalized action, subject, and topic agreement: 30%;
- actor, regulator, jurisdiction, and Program-type scope: 20%;
- applicability, cadence, threshold, and date compatibility: 15%.

Each component and conflicting signal is persisted. Scores are not presented as proof. They produce three review queues:

- **strong candidate** at 0.85 or above with no hard conflict;
- **possible candidate** from 0.55 through 0.84;
- **gap candidate** below 0.55 or with no plausible current Program.

An optional future semantic reranker may reorder candidates inside these queues, but it must implement the same bounded interface, record its version, and never change the deterministic coverage-chain result.

### Coverage classification

Each eligible candidate receives exactly one primary classification:

- `VERIFIED_COVERAGE`: reviewed mapping to an approved applicable requirement with implemented control coverage and current sufficient evidence;
- `MAPPED_NO_CURRENT_EVIDENCE`: reviewed requirement and implemented control mapping, but evidence is missing, expired, failed, or insufficient;
- `MAPPED_CONTROL_GAP`: reviewed requirement mapping with no implemented linked control;
- `PARTIAL_MATCH`: a plausible mapping remains unconfirmed or incomplete;
- `GAP`: no acceptable current requirement mapping exists;
- `NEEDS_REVIEW`: extraction, scope, or match ambiguity prevents a defensible classification;
- `NOT_APPLICABLE`: reviewer-excluded with a reason and audit record.

The existing Continuity current-state projection is authoritative for control implementation and evidence sufficiency. The assessment does not invent parallel control or evidence status rules.

### Percentage calculations

Let `E` be all eligible candidates except reviewed `NOT_APPLICABLE` candidates.

- Requirement mapping = reviewed candidates mapped to approved applicable requirements / `E`.
- Control implementation = mapped candidates whose authoritative current projection has implemented linked control coverage / `E`.
- Evidence support = mapped candidates whose authoritative current projection has current sufficient evidence / `E`.
- Verified document coverage = candidates satisfying all three conditions / `E`.

Before review, estimated coverage uses strong, conflict-free proposed mappings whose current target already has the complete authoritative chain, divided by `E`. Estimated counts are visually and semantically separate from reviewed counts and never enter the verified numerator. Every metric displays numerator and denominator. If `E` is zero, the result is `Not enough information`, not 0% or 100%. Open Matters can lower confidence or explain remediation; they never enter a positive numerator.

## Suggestions and governed application

The assessment produces a single recommended next action per unresolved candidate, ordered from least to most structural change:

1. accept or correct a link to an existing requirement;
2. update an existing requirement, control link, or evidence contract;
3. create or extend a regulatory-change or control-gap Matter;
4. add a draft requirement to an existing Program; or
5. propose a new draft Program when no current Program fits the document's scope.

Suggestions contain source anchors, rationale, intended target, and a preview of the proposed fields. They never mutate Continuity automatically. Applying a suggestion invokes the existing command service and its authority, materiality, maker-checker, and optimistic-concurrency rules. A failed command leaves the suggestion intact with a recoverable explanation.

## Persistence, audit, and staleness

Persist a versioned assessment rather than only a computed response. The storage model records:

- assessment identity, tenant, legal entity, document ID, document SHA-256, and source version;
- analyzer, matcher, scoring-policy, and Continuity projection versions;
- a hash of the compared Program snapshot;
- structured candidates and all source anchors;
- ranked target identifiers, target versions, score components, and rationale;
- reviewer decisions, reasons, actor, timestamp, and prior decision;
- generated suggestions and their review/application status; and
- metric counts derived from the current accepted decisions.

Program, requirement, control, evidence, or applicability changes make an assessment stale when the snapshot hash differs. A stale page keeps the prior result for audit, labels it clearly, removes any implication that the percentage is current, and offers `Refresh comparison`. Recomparison creates a new version and preserves prior reviewer decisions where the candidate fingerprint and target version remain compatible.

## API design

- `GET /api/v1/document-imports/{id}` includes a compact coverage state and summary link.
- `GET /api/v1/document-imports/{id}/coverage` returns the actor-filtered summary, metric counts, review progress, candidates, visible Matter context, suggestions, versions, and staleness.
- `POST /api/v1/document-imports/{id}/coverage/review` records one or more explicit candidate decisions using an expected assessment version.
- `POST /api/v1/document-imports/{id}/coverage/recompare` queues a fresh assessment when failed, partial, or stale.
- `POST /api/v1/document-imports/{id}/coverage/suggestions/{suggestion_id}/apply` translates the reviewed suggestion into an existing authority-guarded Continuity command and delegates directly to that command service; it cannot bypass governance.

List payloads are bounded and cursor-paginated. Exact quotes and detailed target records are returned on demand so the default page stays fast and compact.

## User experience

### Primary flow

The import detail presents one predictable sequence:

`Upload -> Extracting -> Comparing coverage -> Review gaps -> Apply selected updates`

The upload receipt appears immediately. Extraction and comparison use named stages rather than fake percentages. A background refresh preserves the current page and announces completion without moving focus.

### Results hierarchy

The default results view has three visual layers:

1. A compact headline card showing `Estimated coverage` or `Verified document coverage`, review progress, freshness, and the document version.
2. Three small supporting metrics for requirements mapped, controls implemented, and evidence current, each with a numerator, denominator, text label, and accessible progress bar.
3. An action queue grouped as `Covered`, `Needs attention`, and `Needs review`, with `Needs attention` selected when actionable gaps exist.

The page has one primary action at a time. Before review it is `Review matches`; after review it becomes `Apply selected updates`. Secondary operations such as refresh, export, or view history remain in an overflow menu.

### Candidate review

Selecting a row opens a focused side panel on desktop and a full-width sheet on narrow screens. It shows:

- the exact quotation, page, section, and a `View in document` link;
- parsed actor, action, subject, modality, dates, and citations;
- the proposed Program and requirement;
- linked control and evidence state;
- plain-language match reasons and conflicts;
- visible Matter context; and
- one recommended decision with clearly subordinate alternatives.

Technical scoring components are collapsed under `Why this match?`. Confidence is conveyed with text and icons as well as color. Long quotes wrap and expand; they are never silently truncated.

A reviewer may accept all strong, conflict-free matches in a bounded selection, but the confirmation states exactly how many mappings will be reviewed. Gaps and ambiguous matches require individual attention. Keyboard navigation, visible focus, 44-pixel targets, `aria-live` processing updates, reduced-motion support, and WCAG AA contrast are required.

### Empty, error, and stale states

- No eligible obligations: explain why and link to extracted text; do not render a percentage.
- No Programs in scope: show 0 estimated coverage with a clear `Consider a new Program` recommendation, not a generic failure.
- Extraction unsupported or failed: preserve the stored source and explain the extraction recovery path; do not run comparison on invented text.
- Comparison partial: show completed findings, name unavailable inputs, and provide retry.
- Assessment stale: retain prior results with a stale banner and `Refresh comparison` as the primary action.
- Authority denied when applying: preserve selections and state which approval or role is required.

The visual treatment follows the existing ClearSight design system with a restrained trust-and-authority style: neutral surfaces, slate text, blue primary actions, semantic status colors, consistent Lucide icons, an 8-pixel spacing rhythm, and progressive disclosure. It avoids decorative gradients, animated metric pulses, dense multi-column dashboards, and color-only status meaning.

## Security and privacy

- All reads remain tenant-, actor-, legal-entity-, and visibility-scoped.
- Comparison never broadens the actor's Matter visibility.
- Percentage calculations depend only on Program chains authorized for the assessment context, never on hidden Matter counts.
- Stored match rationale excludes restricted Matter text.
- Source quotes are treated as untrusted content and rendered as text, never HTML.
- Candidate and response sizes remain bounded; matcher execution has time and candidate limits.
- Applying a suggestion rechecks current authority and target versions at command time.
- Every review and application action is auditable and attributable.

## Test strategy

### Unit tests

- Structured extraction retains exact quotes and positive page anchors.
- Modalities, actors, actions, dates, citations, and duplicate grouping are deterministic.
- Scope conflicts reject unrelated Programs even with high generic term overlap.
- Score components, thresholds, and explanations are reproducible.
- Percentages use visible numerators and denominators and exclude only reviewed not-applicable candidates.
- Missing or stale evidence cannot produce verified coverage.
- An open Matter never raises a metric.
- Zero eligible candidates returns `Not enough information`.

### Integration and authorization tests

- A complete approved requirement-control-evidence chain produces verified coverage after review.
- A requirement without control coverage produces `MAPPED_CONTROL_GAP`.
- A control with expired or failed evidence produces `MAPPED_NO_CURRENT_EVIDENCE`.
- Unrelated US Federal Reserve and UK Bank of England documents do not falsely match the Nigeria privacy Program.
- Restricted Matters do not leak existence, title, counts, or text to an unauthorized actor.
- Program version changes mark the stored assessment stale.
- Concurrent review updates require the expected assessment version.
- Accepted suggestions go through existing authority and maker-checker rules.

### UI tests

- Processing stages, estimated/verified labels, counts, review progress, stale state, and recovery actions are understandable without implementation terminology.
- Progressive disclosure keeps scoring detail and long quotations collapsed by default.
- Strong-match bulk review is bounded and ambiguous gaps cannot be bulk-accepted.
- Mobile layouts stack without horizontal scrolling; desktop review retains document and match context.
- Keyboard, focus, screen-reader announcements, reduced motion, and contrast pass automated and manual checks.
- Account switching still unmounts actor-scoped state before showing another account.

### Live demo acceptance

1. Upload an official Nigeria Data Protection Commission or GAID document relevant to the existing Nigeria privacy Program.
2. Confirm searchable PDF extraction, structured candidates, page anchors, and a completed comparison.
3. Confirm meaningful partial matches against existing requirements and truthful control/evidence gaps.
4. Review representative matches and confirm verified metrics recalculate from accepted mappings.
5. Prepare one existing-Program update or Matter suggestion without applying it automatically.
6. Upload the official Federal Reserve and Bank of England samples and confirm that jurisdiction/scope gates prevent false Nigeria privacy coverage.
7. Verify the workflow under at least two demo roles so restricted context and authority-dependent actions remain correct.

## Acceptance criteria

- Every assessment finding is traceable to an exact source quotation and page.
- The comparison uses current Programs, requirements, controls, evidence, and actor-visible Matter context.
- The UI distinguishes estimated from verified results and never presents a match score as legal compliance.
- Verified coverage counts only reviewed candidates with a complete authoritative chain.
- Numerators, denominators, exclusions, assessment versions, and staleness are visible and auditable.
- Unrelated regulatory documents do not create plausible false coverage through generic keyword overlap.
- Recommended updates are actionable but never mutate governance records without explicit authorized review.
- The default page is compact, responsive, accessible, and organized around the next best action.
- Live official regulatory documents demonstrate both a meaningful partial match and a truthful unrelated-document gap.

## Non-goals

- A legal opinion, certification, or guarantee of regulatory compliance.
- Automatic activation of Programs, approval of requirements, control implementation, or Matter closure.
- Making an external language model mandatory for comparison.
- OCR for image-only PDFs in this assessment increment; those documents retain the existing explicit OCR-required outcome.
- Reconstructing PDF tables, signatures, images, or visual layout.
- Replacing the existing Continuity current-state model with a parallel compliance engine.
