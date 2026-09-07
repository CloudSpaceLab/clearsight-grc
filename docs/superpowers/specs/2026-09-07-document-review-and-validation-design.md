# Focused document review and validation rules

Date: 7 September 2026. Status: proposed extension awaiting review of the interactive concept. This is not an implemented validation capability.

Extends the approved [shared document management proposal](../../reviews/2026-09-07-form-and-vendor-document-management.md) and its [implementation plan](../plans/2026-09-07-shared-document-management.md). Existing secure inspection work can proceed; new validation behavior must follow approval of this extension.

## User requirements

- A premium document-access and reading experience, including blur and focus features.
- Powerful search and advanced filters shared by Forms and Vendors.
- Sophisticated configurable rulesets that automatically check submitted documents and reduce manual verification effort.
- AI-assisted validation must be easy to configure and understand.
- Signatures remain required only by applicable source-form signature fields.

## Recommended approach and alternatives

**Recommended: typed rules with AI-assisted evidence extraction.** Users assemble named checks, test them on selected samples, and publish an approved ruleset. Deterministic checks handle dates, identifiers, amounts and exact matches; AI can extract facts and evaluate clauses with citations. Approved issuer/source integrations add independent verification. This is reconstructable, works in degraded conditions, and fits existing governance.

**Prompt-only validation** makes an initial prototype quick but leaves applicability, failure handling, repeatability and authority ambiguous. Free text may help draft a rule; it must not become executable policy without a validated typed representation and review.

**A separate document-automation platform** adds storage, workflow and configuration surfaces beyond this need. Reuse Evidence/Capture, document extraction, the AI gateway, source access and existing configuration/automation controls instead.

## Focused interface

### Documents

The same authorized list appears in Forms and in a selected vendor relationship. Vendor context narrows by stored relationships, never filenames or email matching. Search and applied filters remain when the viewer opens, closes or moves to the next result. Counts identify the checked population and use the same server scope.

- Filename, document type, source form/question, vendor or subject, respondent, submission time, reviewer, expiry, security state and validation outcome.
- Quick views: Needs review, Failed checks, Replacement requested, Awaiting inspection and Expiring soon.
- Advanced filters: entity/authorized subject, form and revision, document type, submitter/uploader, reviewer, submission date range, expiry range or unknown, ruleset/version, rule outcome, source-check status, replacement state and current/history.
- Typed all/any groups, inclusive range semantics, a readable applied-filter summary, clear-all, saved personal views and explicit shared-view permissions. Bound group depth, condition count and query time.
- Metadata/reference search first. Extracted-content search appears only for authorized, safely processed content and discloses incomplete extraction. Never promise full-document search when only filenames or partial text are indexed. Use existing PostgreSQL capabilities before adding infrastructure.
- Optional natural-language filter assistance produces editable filter chips and a scope preview; it does not broaden authorization or save a view silently.

### Viewer

A wide focused sheet uses a subtle blurred backdrop, quiet borders, document-first typography and a compact review panel. Blur affects only background competition; it is not redaction or access control. Document text, source excerpts, errors and decisions remain sharp. Reuse the existing token and overlay contracts; no parallel visual system.

- Desktop: document canvas plus Checks / Details / History. Focus mode hides the review panel and expands the reading area. Visible zoom/page controls, previous/next document and close preserve keyboard access.
- Mobile/reflow: full-height surface with Preview and Checks & details sections, independent intentional scrolling and visible actions. Never shrink two desktop columns into illegibility.
- Both themes, reduced motion, reduced transparency fallback, forced colors, focus containment/restoration and 200% reflow are required.
- Supported PDF/image preview only after inspection. Unsupported types show permitted metadata/download; pending/quarantined/disposed bytes stay unavailable.
- Open a check's citation at its page/region. If extraction cannot identify the source, show that limitation rather than a fabricated highlight.

The prototype uses a hand-authored HTML insurance excerpt to demonstrate layout; it is not a PDF renderer, genuine insurance evidence or a real extraction result.

## Ruleset definition

A ruleset is scoped, versioned configuration with owner, reviewer/authorizer route, document purposes/types, applicability, effective dates and a bounded set of checks. Its revision and dependencies are immutable once used.

Rule families:

1. **File and extraction prerequisites:** clean security receipt, permitted file type, readable pages, language/support limits, complete required extraction. Unsupported input is unknown, not success.
2. **Structured checks:** required values, date/freshness ranges, issuer/registration identifiers, coverage/amount thresholds with currency/unit semantics and exact or explicitly normalized held-record comparisons.
3. **Content checks:** required clauses, prohibited exclusions, document scope and contradictions. AI-assisted checks return cited assertions, counterevidence and uncertainty. A confidence score alone cannot satisfy a material check.
4. **Cross-record checks:** agreement with selected authoritative vendor/form/previous-document facts, including the compared record version. Changed source facts invalidate currentness without rewriting the historical run.
5. **Source verification:** approved issuer/register confirmation with purpose-bound connection, source receipt and timestamp. Unavailable source or stale result requires review.
6. **Integrity indicators:** reuse/digest comparisons and supported signature/certificate verification, each with clear limitations. File metadata or an AI visual assessment alone cannot prove authenticity.

Each rule has a stable ID, applicability, typed operator, expected result, severity, evidence requirement, dependency versions, unknown handling and a human-readable next action. Allow bounded nested all/any groups. Reject arbitrary code, unbounded expressions and free-form network targets.

Start with editable templates for insurance, registration evidence and security assurance. Natural language may draft a ruleset, but the user must inspect its actual conditions and test outcome. Only source-form signature fields govern respondent signature requests; a ruleset cannot silently add one.

## Results and decisions

Store separate states for security, extraction, rule evaluation, reviewer acceptance, expiry and parent workflow conclusion.

- Rule states: Passed, Failed, Needs review, Not applicable and Not run. Not run includes blocked, missing-provider and exhausted-retry conditions with reasons.
- A result names the exact artifact digest, submission/field occurrence, document purpose, ruleset revision, extraction version, compared source versions, AI/provider/model configuration where used, evaluation time and citations.
- “Passed configured checks” is not “Authentic,” “Legitimate,” “Compliant” or vendor approval. Authentication claims require a defined supported source/method and a bounded claim about what was verified.
- Aggregate results disclose coverage. Missing required checks, contradictions and unknowns cannot be averaged into a green score.
- Material source, file, ruleset or evaluator changes make a previous result stale. Preserve that result as history and queue an eligible new run; never relabel it current.
- Human acceptance is a separate authorized version-aware decision. An override records rationale and the unresolved findings; it cannot override quarantine or bypass authority/hold requirements.

## Governed automation

Submission plus clean inspection can trigger applicable checks through the existing durable worker. Idempotency binds document occurrence/digest, ruleset version and evaluation dependencies. Bound pages, bytes, group depth, concurrent jobs, retries, tenant quota and provider spend; cancellation/revocation is rechecked before external processing and material writes.

An Automation Policy specifies purpose, action class, eligibility, blast radius, compensation, monitoring, kill switch, expiry and outcome contract. Initial automation records findings and routes review only. Automatic acceptance, rejection, replacement messages or disposal are separate action classes requiring explicit approval; they are not implied by configuring checks.

Ruleset publication follows draft → sample simulation → impact preview → maker-checker approval → effective activation. Preserve rollback and run history. A no-AI path supports deterministic checks and manual evidence review; unavailable AI must not hold unrelated deterministic work hostage.

## Security and integration boundaries

- Evidence/Capture owns artifacts, immutable submission membership and protected content delivery.
- Reuse bounded extraction where its security/maturity contract applies; do not enable an unscanned import-analysis fallback for capture documents.
- AI runs through existing approved gateway/provider/data-use controls. Sending protected documents to a new external provider requires the appropriate configured authorization.
- Treat document text, OCR, links and embedded instructions as untrusted data. No document can alter the rules, choose a provider, grant access, cause unrestricted fetching or instruct the evaluator to approve it.
- Retrieval and search filter tenant/entity/purpose/record visibility before result limits. Indexes and extracted text have the same retention, deletion and authorization requirements as original documents.
- Prefer a narrow validation component with typed contracts over embedding policy logic in viewers, providers or third-party assessment services. Existing domain review commands remain authoritative.

## Extended issue tracker

| ID | Priority | Delivery / acceptance condition | Status |
| --- | --- | --- | --- |
| DOC-08 | P1 | Focused viewer, subtle backdrop, reading mode, theme/mobile/a11y evidence | Interactive concept; runtime integration pending |
| DOC-09 | P1 | Bounded authorized advanced search, saved views, explicit content coverage | Interactive filter sample; backend contract pending |
| DOC-10 | P1 | Typed versioned rule schema, templates, nested groups and deterministic evaluator | Proposed |
| DOC-11 | P1 | Citation-preserving safe extraction and constrained AI checks via approved gateway | Proposed; provider/processing maturity must be verified |
| DOC-12 | P1 | Ruleset simulation, impact, maker-checker publication/effectivity/rollback | Proposed; reuse configuration controls |
| DOC-13 | P1 | Durable policy-governed execution, dependency freshness, rerun and recovery | Proposed; no auto-acceptance implied |
| DOC-14 | P1 | Issuer/source checks with actual provider receipts and unavailable paths | Proposed; source availability is a prerequisite |
| DOC-15 | P1 | Adversarial evaluation, false-pass/false-fail tests, accessibility and release proof | Proposed |

## Acceptance and evaluation additions

1. A reviewer finds a submitted file from either surface with identical authorized metadata and actions; draft uploads stay excluded.
2. Saved filters, selection and source scope survive open, next/previous and return. Guessed IDs and cross-entity content searches disclose nothing.
3. Blur/focus and citation navigation work by keyboard, in both themes, at narrow widths and reflow, without hiding actions or errors.
4. Two different rulesets can be applied to one document for different purposes without overwriting each other's results or acceptance.
5. Date boundaries, time zones, normalization, currency and missing values have deterministic tests.
6. An altered entity, expired policy or missing required clause fails the appropriate rule; inability to read a clause remains unknown.
7. Prompt-injection text and hostile URLs in a document cannot modify policy, expose other records or trigger external requests.
8. Issuer outage, provider outage, rate limit, partial OCR, unsupported language and corrupted content have visible recoverable outcomes.
9. Repeated delivery/job execution yields one result per eligible dependency version; lease expiry and worker crashes recover safely.
10. A stale ruleset/source/document result cannot be presented as current; historical acceptance remains reconstructable.
11. Reviewer overrides require authority/rationale; no checks add signatures or silently approve parent workflows.
12. Before production, measure false-pass and false-fail rates on labelled representative/adversarial samples by document type and language. Agree acceptance thresholds for each intended use; an attractive mockup is not evaluator validation.

## Prototype and verification receipt

- Tracked interactive concept: [document review concept](../../design/prototypes/2026-09-07-document-review-concept.html). The current companion copy is `.superpowers/brainstorm/document-review-20260907/content/document-workspace-v3.html`, served at `http://127.0.0.1:5191` while the companion is running. Generated companion sessions are ignored; the self-contained concept is retained in documentation.
- Uses only fictional/sample records and session-local simulated actions. No real vendor mutation, email delivery or AI processing.
- Browser checks exercised filters, opening/closing, focus mode, mobile panes and sample rule simulation at dark 1440px, light 1440px, dark 390px and light 720px widths. No page errors or horizontal overflow in those checks.
- Visual inspection found and corrected theme-token inheritance and mobile viewer action-height problems; images were rerendered. Prototype behavior is not production workflow acceptance.
- Automated axe WCAG 2 A/AA and WCAG 2.1 AA checks reported no violations on the dark desktop document list, viewer and rules builder. This is a limited automated check, not an accessibility certification.
- Native browser prompt dialogs are temporary prototype action placeholders; production actions use the existing focused form components with authority, reasons, deadlines and failure recovery.
- Proposed sequencing: finish the approved secure document foundation, approve this extension, then implement typed deterministic checks and governed configuration, followed by AI/source adapters only where safe processing and provider evidence are ready. Avoid building a general-purpose rules platform.
