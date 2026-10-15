# Governed compliance framework library and Program adoption — implementation plan

Date: 2026-10-15
Tracking: https://github.com/CloudSpaceLab/clearsight-grc/issues/437
Status: plan only; no runtime or customer content published
Reviewed alongside: #433 calculation plan (PR #434), T0 link correction (PR #435), weighted evaluator (PR #436)
Adjacent: #270, #140, #344, #346, #265, #267, #57, #425

## 1. Outcome and release boundary

Allow a GRC administrator or Program owner to find an available compliance standard, inspect its edition and scope, adopt it into a new or existing Program, reconcile clauses with existing approved Program Requirements, and reuse approved controls/forms/evidence. Maintain the exact edition and clause lineage through local applicability, review, supersession, audit and change impact. Provide separately named standard-clause coverage, and later framework-attributed contributions from the canonical #433 calculation, without duplicating a scoring unit or presenting an unsupported compliance claim.

Ship a small proven content set before extending to a 30–40-framework commercial library. The number of published standards is not the number of installed/activated Programs, assessable requirements, or independently licensed full-text editions. Each content entry states content completeness and rights; never market a stub/reference as a complete production-ready framework.

No new generic GRC module, second Program, another Requirements store, independent control implementation/evidence engine, questionnaire engine, workflow/approval/scheduler, scoring engine, generic rule language, chart package, or reporting engine. No auto-creating a Program per publication or clause. No automatic regulator/legal conclusion, automatic interpretation of ambiguous applicability, or silent tenant activation. Preserve offline operation without AI or source connectivity.

## 2. Source audit and concrete contracts

Audited main at 2957586cb211e911a3336be4d3570820d21eeb15. Re-check latest main and PR diff/status immediately before each implementation tranche; the present plan describes checked code, not future merges.

| Existing owner | Verified foundation | Contract boundary for #437 |
| --- | --- | --- |
| internal/continuity/model.go, program_edits.go, requirement_coverage.go | Program: tenant/entity/organization scope, owner/authority, jurisdiction, version; approved time-scoped Requirements with source ID/anchor and independent applicability; ControlImplementation, RequirementControlLink and evidence. | Canonical Program Requirement remains the only assessed obligation identity; published clauses are external references, not scored records. Add reviewed adoption/link metadata, not a replacement Program aggregate. |
| internal/continuity/state.go and current_state_refresh.go | Ten-dimensional categorical state + event/projection worker; status is not a numeric rate. | Never override current/at-risk/gap state with an arbitrary framework grade. Map validity must use the same effective-time predicate as #433. |
| internal/controlcatalog/model.go, service.go; migration 000100 | Tenant control definitions linked to actual entity-scoped Program implementations; promote/link API; immutable catalog rows. | A reusable control definition is not a shared current operating implementation or a published-standard mapping. Existing service only creates version 1; do not assume catalog revision/retirement commands already exist. |
| internal/documentcoverage | Document candidates, obligation matching, source anchors, reviewed suggestions into existing Programs. | Reuse extraction/proposal and anchored review for edition delta; do not equate matching confidence with legal applicability or a signed publication. |
| internal/evidence, internal/formcontract | Governed scored forms, response revisions, bank assessment, vendor relationships and evidence contracts. | Never store answer/test copies in catalog or implicitly credit a linked form as a passed clause. A new request only when existing evidence cannot satisfy current purpose/scope/age. |
| migration 000060_form_starter_catalog, form authoring | DB-backed versioned catalog seed precedent. | Use analogous immutable, rerunnable publisher/seed provenance, not form templates as a pseudo-framework. |
| migration 000134_form_subject_links | Generic shared-form links currently constrain subject_type to MATTER. | Do not claim framework/Program Requirement links already supported by this table; extend only if operating journey requires governed form association, in coordination with #140. |
| internal/metricview, reporting | Retained metric observations/trends/drill/report pipelines. | No framework-specific metric warehouse; consume #433 snapshot if and when it supports trustworthy framework attribution. |
| internal/governance, authority, organization | Verified actor scope; maker-checker and legal-entity boundary. | Publication authority and tenant adoption authority remain distinct; no source author edits a customer's compliance state. |

No catalog publication workflow or versioned standard-clause-identity relation was confirmed in main. The product regulatory-intelligence specification describes desired Authority Source and provision handling, not proof that a full authoritative publication registry is operational. Sourceaccess Connection/View/Binding describes connected technical sources; do not recast it as a legal-regulatory source catalog.

## 3. Explicit ownership agreement with #433

### Catalog owns reference identity and relationships

- Edition identity: publisher/issuer, stable standard code, jurisdiction, release/version, dates, rights, authoritative URL/citation, content checksum, withdraw/supersede status.
- Clause identities and hierarchy: stable logical clause identity across editions when provable, edition-specific content and parent, source coordinates/reference. Clause prose only when legally distributable.
- Reviewed mappings: clause edition to exact current Program Requirement identity/revision/effective interval and explicit mapping status/extent; optional curated recommended control objectives.
- Tenant adoption: Program, scope, edition pin, publication visibility, reviewer, activation/retirement, source/right compliance, upgrade proposal/history.
- Content release pipeline: signed/approved catalog release manifests, idempotent install, change-detection, impact preview, rollback of activation without rewriting adopted facts.

### #433 exclusively owns calculation

- Eligibility population for Program Requirements and expected vendor/internal assessments; weights and category allocation; unknown/missing/overdue semantics; score caps; critical gates.
- Bank-assessed contribution provenance, numerator/denominator, quality and status; exact retained snapshot, privileged/actor-safe contribution drill; invalidation and trends.
- Percent/ratio metric registration, group compatibility, Portfolio/Insights/Reports.

A published clause SHALL NOT be inserted as an additional ProgramComplianceInput. A clause mapped to one or several Requirements is a reporting attribution over the #433 canonical units. Multiple clauses mapped to the same Program Requirement do not multiply its effective weight. Cross-standard reuse may legitimately show the same assessed control as evidence in two standard-specific views, but cannot multiply the Program score. Distinct independently applicable obligations remain separate Program Requirements only after explicit governed review and are therefore distinct scoring units.

A catalog may suggest a default calculation profile only as an immutable, approved preset reference. Selecting a preset creates a Program-scoped proposed #433 policy revision; publisher updates cannot change approved Program weights. Framework clause importance/mandatory flag is descriptive data and must not silently become numeric weight, a disqualifier, or automatic legal conclusion.

### Resolve circular acceptance gates

- #433 T0 and T2 can validate two obligations using one existing assessment/control through source-linked Program Requirements before #437 F2 lands. This is a scoring-truth fixture, not a claim that the standards library exists.
- #437 F0/F1 can ship independently of #433 numeric calculations. F2 must use reconciled effective link semantics from #435 or its accepted replacement.
- The production end-to-end 'two published standards / one shared assessment' gate belongs to #437 F5 after both contracts ship; it must not block completion of independent #433 evaluator/schema work.
- A framework-specific numeric score is forbidden until same-snapshot #433 member attribution, accessible evidence, minimum coverage, mapping validity and compatible per-standard population are proven. Before that display clause coverage and current Program status separately.

## 4. Canonical data model and schema budget

Model these logical identities; physical table design is decided at F0 after migration/replay/index review. Persist actual many-to-many mapped identities with typed scoped foreign keys, not unbounded serialized arrays.

1. Published edition — immutable release keyed by publisher + standard code + edition + content hash. Fields include official source identity/URI, citation, jurisdiction, publish/effective/withdrawn dates, classification (normative/reference/preview), completeness, redistribution entitlement, approver, release checksum, supersedes edition and manifest/provenance. Status and availability transitions are separately audited, never edit published clause facts in place.
2. Edition clause — stable clause code, edition-specific identity, optional parent/sequence, source anchor, permitted summary/full text, requirement nature (where determined), checksum and provenance. Identity cannot depend on spreadsheet row number, display sort order or AI-generated wording. The same clause code under a different edition is a different immutable revision.
3. Program-edition binding — exact tenant + legal entity + Program + edition, optional authorized organization scope and applicability period, adoption lifecycle, maker/checker, target active version, revision and audit events. One Program may adopt multiple editions; prevent two simultaneously active editions of the same standard within the same incompatible scope unless expressly supported and marked as transition. No source-publication write may automatically alter a tenant binding.
4. Clause–Program Requirement link — exact Program Requirement and edition clause, reviewed mapping kind (EXACT / PARTIAL / SUPPORTING), scope, effective-from/until, reviewer, origin and rationale; activate/retire/supersede via versioned authority. Links cannot cross tenant or entity and cannot point to an unrelated Program binding, retired requirement or unavailable edition. Multiple clauses can refer to one Program Requirement; multiple Requirements can cover one clause with explicit combined-coverage semantics. A PARTIAL/SUPPORTING link never means the clause is fully proved.

Control crosswalk: phase 1 resolves existing RequirementControlLink -> Program ControlImplementation -> controlcatalog Definition; derive with bounded scoped reads and same validity period. Global normative clause -> generic recommended control objective mapping is a later optional curated relation ONLY after demonstrating a useful non-duplicative reuse case. Never link a global published clause directly to another tenant's ControlImplementation.

Physical migration candidates: edition, clause, Program-edition binding, reviewed clause-Requirement mapping, plus append-only review/events only where current existing Program event/authority machinery cannot safely represent them. This is an intentional durable catalog requirement, not prohibited by #433's 'no additional tables by default' constraint, which applies to that issue's calculation snapshot/member architecture. Favor normalized FKs, unique active/published identity, keyset indexes, query-time legal scope and exact historical membership. The implementation PR must justify each table/column/index and update durable-schema-ownership.md with owner, lifecycle, consumers, retention, migration/down-protection, and executable tests.

### Effective-time and historical consistency

Catalog release identity, binding revision, Program Requirement revision, applicability decision and control mapping validity must all be resolved as-of the same timestamp. Treat superseded Requirements with a retained source mapping as historically addressable; do not rewrite or delete old Program linkage when an edition is upgraded. Invalidate only dependent current Program views on approved adoption/remapping; store deterministic source/binding high-water alongside #433 calculation snapshots once integrated. Stale/unavailable content and source withdrawal must not rewrite past compliant/failed evidence.

## 5. Applicability, populations and anti-duplication truth table

- UNREVIEWED clause: in the catalog and pinned Program, no approved applicability conclusion. Count as 'Decision needed', not 'Not applicable' or 'Met'. Do not inject into #433 eligible mass until represented by an approved current Program Requirement and policy population.
- APPLICABLE clause + approved mapped Requirement: clause view references existing coverage/status; if the control/test is unassessed, mark UNKNOWN. Clause metric and Program score use separate named denominators.
- PARTIALLY APPLICABLE clause: preserve the chosen in-scope subject subset and unresolved remainder, subject to governed review; never infer full coverage from one mapped passing control.
- APPROVED NOT APPLICABLE: explicit rationale, scope, authority and effective time; distinguish true exclusion from missing review. Program source applicability remains in continuity, not independently decided by a catalog link.
- UNMAPPED applicable clause: explicit gap and work item/recommended next action; no fabricated green, no auto-zero score unless the approved #433 policy already establishes the corresponding current obligation.
- SUPERSEDED/RETIRED clause or control link: historical attribution before retirement remains reconstructable; current joins exclude it at effective time. #435 reconciliation is a prerequisite.
- ONE Requirement mapped to TWO clauses: one #433 contribution with two views; clause rollup shows that its evidence is reused and does not present a sum of duplicate contribution points as the Program weighted total.
- ONE control to TWO separate Requirements: one actual test/request, two obligation contexts only if independently applicable; #433 must dedupe origin/evidence and avoid a second issue penalty while preserving legitimate separate requirement eligibility.
- REUSED THIRD PARTY across two Programs: use one authorized response origin when revision/scope/purpose match, with Program-specific expected population; no global vendor compliance score copied across Programs.
- CONTENT RIGHTS WITHDRAWN: hide/redact prohibited catalog text, preserve permissible citation and historical foreign-key identity; notify authorized administrators, do not delete immutable bank decisions or retroactively change calculations.

Required displayed metrics:
(a) catalog clauses in selected published edition;
(b) applicable / N/A / unresolved / unmapped clause counts for selected Program and scope;
(c) requirement coverage from canonical continuity;
(d) #433 compliance score + assessment coverage, only if published by #433 and correctly attributable.
Do not equate (a) or (b) with 'compliance percentage'. If one Requirement covers multiple clauses, clause coverage is an annotated view, not a separately calculated weighted compliance index unless a governed denominator/mapping rule is approved.

## 6. Edition lifecycle and content operations

### Publish and install
- Trusted content curator imports authoritative source or manifest through existing document import/analysis where applicable, preserving exact file hash, issuer, date, license, retrieval time, coordinates and human review receipt.
- Reject fabricated official-source anchors, duplicate or conflicting edition IDs/checksums, missing publication authority, unlicensed full text, malformed clause tree/cycles, dangling crosswalk codes, and ambiguous supersedes relation. Never silently publish an inferred clause.
- Revision review is maker-checker where material; active content edition is immutable. Publisher installs/reruns reconcile by external stable IDs and content checksum rather than resetting tenant edits; no automatic activation.
- Seed quality tiers: REFERENCE_ONLY (metadata/citation), LICENSED_CONTENT (rights verified), OPERATION_READY (curated requirements, sample applicability, reviewed mappings, reference scenario tested). Labels reflect evidence; do not market all tiers as ready-to-activate complete programs.
- A centrally shipped edition is shared metadata; a tenant's adoption, copied/approved Program obligations and evidence remain tenant-owned. Do not expose tenant operational data in global catalog reads.

### Upgrade
- Publication of edition 2 creates a candidate change set, not an edit to edition 1.
- Reuse documentcoverage grounded matching to propose UNCHANGED, MOVED, AMENDED, NEW, RETIRED and UNRESOLVED clauses with exact source anchors, provenance and human review.
- Preview impacted Programs and requirements using bounded reverse indexes: clause mapping, control/test reuse, due evidence, Program score policy, Scope and any linked Matters. Changes to standard wording do not automatically change customer's source requirement or #433 weights.
- A Program owner selects an edition upgrade, reviews re-use/new/retired mappings and new applicability, submits to independent authorized approval, then activates effective-dated version. Existing Program Requirement mutations use canonical add/supersede/applicability commands. For incompatible material changes, gate activation on recorded resolution or explicit unknowns.
- Update may invalidate relevant current #433 source high-water after activation, but prior as-of snapshots stay pinned to edition 1 and original policy. Use existing Program event/outbox/projection jobs; no new global recompute loop.
- Failed/partial upgrades leave old active version and return a resumable review/repair state. Idempotency keys prevent duplicate Program Requirements or multiple activation events.

## 7. UX and end-to-end journeys

Do not add a new top-level 'Standards' application in the first release. Use existing Program creation, Requirements & controls, Program Overview, Configure/reference-content administration and document proposals. Reuse existing shared SelectField, DataTable, FocusedSheet, Notice, status and metric components; no standalone invented UI primitives.

New Program: existing setup begins with 'Use a standard' / 'Start blank' / 'Import requirements'. Choose standard, edition, legal entity, area, owner. Preview count of applicable/unreviewed clauses and candidate existing controls, confirm. Creation still uses canonical Program command. Publish proposal/requirements with correct review state; do not say that adopting the standard makes a Program compliant. Time objective: <3 minutes hands-on for routine adoption setup, excluding required approval wait (to be measured, not claimed).

Existing Program: 'Add standard' in current Program setup; inspect source/edition and mapping impact; link existing approved requirements first. Only create missing requirements after explicit review. One current Program can have several standards, and one required form can serve both if authority, purpose and evidence checks permit.

Requirements & controls: filter by standard/edition, clause, applicability, missing maps and evidence status; review inline source reference with link to publication if permitted. Current Program Requirement is the editable operating record. 'Link existing control' and 'Link existing form' are default actions over governed candidate choices, not 'Create linked form'. If generic form_subject_links requires PROGRAM/REQUIREMENT expansion, coordinate schema/authority with #140; do not overload Matter-only FK.

Program Overview: reuse existing status/score/coverage strip; include concise 'Standards: 2', important unmapped obligations and direct actionable path. Avoid duplicated forms/tables or a second compliance score for each standard. If #433 inactive: show standard mapping/coverage without a numeric score.

Administration: use a controlled catalog publisher/view, with edition, source, rights, release approval and upgrade impact; keep complexity off routine Program owners. Maintain concise, operational, action-oriented copy under #425.

## 8. Implementation tranches and dependency gates

### F0 — signed integration contract, no schema or UI

- Pin code/PR state; reconcile #433 T0 eligibility, Program requirement UUID/version semantics, source anchors, approval authorities, time boundaries, #435 effective links and #436 input key contract.
- Inspect exact migrations/replay and confirm proposed table/ownership budget, FK feasibility, actor-scope strategy, planned version-snapshot representation, reverse mapping and performance/index/retention assumptions.
- Prove sample two editions/clauses mapped to one Requirement and control without altering #433 weighted mass; prove explicit mixed applicability with unknown/missing states.
- Review source publication licensing, official-source traceability, completeness tiers and minimum releasable content set.
Gate: approved data-contract diagram, join/identity truth table, zero duplicate scoring inputs, exact F2 interface, nonempty fixture, confirmed owners. F0 may proceed alongside #433 T0/T1.

### F1 — immutable reference content

- Typed catalog edition/clause read/validation + idempotent release installer (following Form starters migration 000060 precedent, not its form schema).
- Exact approved source manifest, checksum verification, status and rights, bounded search/keyset by domain, country, edition, publisher and available/retired.
- Reference-only edition sample and one operation-ready permitted sample; tenant isolation for future bindings.
Gate: rerun produces byte-identical manifest/current editions and no tenant effect; malformed/unauthorized source refused, replay and migration checks pass. No scoring integration.

### F2 — reviewed Program adoption and mapping

- Program-edition binding and reviewed clause↔Program Requirement mapping through authorized Program commands/events, dates, scope and immutable history.
- Candidate matcher uses documentcoverage as proposals; deterministic reuse of existing approved Requirements, controlcatalog links and current evidence. Canonical add/supersede for unmatched obligations, never direct SQL status spoofing.
- Multi-standard Program and no-collisions/regression with #435 status/coverage semantics.
Gate: 2 editions and 1 Program share existing Requirement/control/test without duplicates; different entity cannot activate/view; conflicting concurrent adoption and failed approval leave prior active state.

### F3 — operational UX and first content tranche

- Add 'Use a standard' to Program setup and 'Add standard' to an existing Program. Unified review for unmatched/ambiguous clauses in existing Requirements & controls; honest source and count states.
- Reuse current approved Forms/Monitoring and vendor collection paths, #140 if linked remediation required; avoid a new engagement.
- Publish a small, legally usable bank-relevant content pack only after source verification and scenario test; flag non-operational metadata references clearly.
Gate: customer role walkthrough at desktop/mobile/light/dark, real action routes, no recreated questionnaires, <3-minute setup target measured; no extra global menu.

### F4 — governed edition revision and adoption upgrade

- Versioned delta comparison from authoritative documents, conflict/reviewer resolution, bounded affected Program preview and guarded approve/activate/retire; old edition retained.
- Existing outbox/projection invalidation only for affected Programs; no whole-fleet rescoring on publication.
Gate: v1 pinned until v2 approved; old as-of view unchanged; missing/ambiguous mapping visibly requires review; rollback of activation returns prior effective mapping without deleting events.

### F5 — #433 score attribution, ratios and reporting (hard dependency)

- Consume ONLY finalized #433 T2/T3 typed Program snapshot + contribution/member contract, respecting #433 actor visibility and deduplication; attach clause/edition references to eligible Program Requirement contribution identity.
- Summarize clause coverage and, if mathematically supportable, selected-framework assessed mass/quality using an approved explicit calculation policy and same retained source-revision basis. Do not invent score percentages by averaging clause compliance or by counting controls.
- Use #344/#346 native ratio path, existing Portfolio/Insights/Reports and scope-compatible trend comparison; privileged aggregate and detailed drill remain distinct.
Gate: one source assessment displayed under two standards but original Program numerator/denominator unchanged; revoked access cannot reveal hidden Matter/vendor score movement; source/policy/edition revisions match, current/as-of/output parity.

### F6 — production content/release evidence and bounded expansion

- Cross-version, multi-tenant/entity/group, 0-applicable, partially applicable, missing, duplicate, license-withdrawn, source-changed, malformed-content, stale, revoked-reader, concurrent upgrade, backfill/replay and disaster recovery tests.
- Benchmark catalog browsing, Program/edition listing, clause drill, reverse dependent-Program lookup, upgrade impact and projected metrics at representative bank/MTN cardinality. Initial test fixture should include at least 40 catalog editions, 20,000 clauses, 10,000 Programs and large mapping fanout; record p95/EXPLAIN and index behavior rather than claim blanket scalability.
- User acceptance: CRO/CCO finds worse standard/program/area and exact owned gap, Program owner adopts/matches/fixes, reviewer approves, vendor supplies only assigned evidence; ≥90% unassisted completion target and supported comparator notes where Archer accessible.
- Ship first validated catalog tranche, then add standards only through tested signed manifests. Track published/operational/tenant-activated counts separately, with documented date and licensing. No automatic backfill of existing Programs.
Gate: same commit passes Go PostgreSQL/tenant-history/authority/performance, frontend type/check/copy/render/accessibility, hosted end-to-end, audit/download; known gaps explicitly recorded.

## 9. API, performance and security

Prefer existing Program operation discovery and command routes. Only add a scoped bounded catalog listing/detail (editions, clauses, permitted citations), an adoption preview/commit route, and reviewer/upgrade action extensions where existing command/read contracts cannot serve them. Every material command: verified actor, current tenant/entity/scope, expected revision, role/route check, transactional durable event and outbox. No browser-computed compliance or client-page denominators.

Protected content: a published copyrighted body is not globally retrievable merely because its metadata exists. Enforce license/entitlement and tenant redaction at the API/document boundary; preserve audit references when display rights lapse. Tenant-specific interpretation and extracted uploaded legal documents remain isolated; do not send to external AI without existing source/AI policy authorization.

Use keyset pagination, stable natural-code/revision indexes, unique active mappings by scope/effective interval, bounded joins/queries, release checksum caching. Avoid repeated per-clause Program aggregate loads (N+1), unbounded cross-tenant fanout, in-memory authorization-after-pagination, or global dependent-Program scans on a publication event. Projection failure labels current score STALE/UNAVAILABLE; never claim activation succeeded if material transaction failed, and never undo committed activation because a derived metric failed later.

## 10. Explicit anti-regression tests

1. Two frameworks, one canonical Requirement, one ControlImplementation, one assessment: Program scoring unit appears exactly once.
2. Two distinct independently applicable Requirements share one control test: genuine requirement population stays two, evidence origin one, no duplicate Matter penalty.
3. One applicable clause unmapped, one partial, one N/A and one unknown: counts and next actions distinguish all four; no false 100%.
4. v2 source published but not adopted: current tenant stays on v1; v1 as-of status/policy/score/reference available.
5. Requirement superseded, link retired, or mapping future-dated: point-in-time joins agree with #435 predicates and current coverage.
6. Tenant A standard or local evidence cannot leak to Tenant B; entity scope cannot be widened through Group reading or catalog mapping.
7. Maker cannot self-approve material activation, authority revocation enforced, conflicting version write rejected.
8. Bad source rights or unsupported full-text license returns safe metadata/limited citation, not unauthorized content.
9. Missing or pending vendor assessment changes expected coverage only via #433 defined population, not catalog clause count.
10. All read surfaces with same actor/scope/revision reconcile; if actor cannot view contributor, do not expose a protected weighted difference.
11. Catalog installation twice, failed worker replay and amended content with same purported edition ID cannot duplicate/rewrite immutable records.
12. Adopted existing Program retains its manual requirements, edits, evidence and prior approved #433 policy on failed/rolled-back standard upgrade.

## 11. PR and execution strategy

- PR F0: review/contract documentation and acceptance fixtures, no schema; coordinate with #433 T0 without blocking #434/#435/#436.
- PR F1: catalog schema, seed/publisher and bounded reads; tests independently.
- PR F2a: Program binding commands/events; PR F2b: reviewed clause↔Requirement mapping with canonical reuse.
- PR F3a: existing Program setup and existing Program adoption; PR F3b: requirements/control linking and end-to-end UX.
- PR F4: edition diff/upgrade; split matcher and transition if necessary.
- PR F5: score attribution only after #433 T2/T3 and ratio contracts are merged; follow with reporting parity.
- PR F6: vetted content packs, validation and release evidence independently from runtime rollout.

No changes to open #433 drafts from this planning PR. Before implementation check latest branch heads, open PRs and shared tests; rebase small tranches on main and merge only after exact-head gates. A separate issue owns this catalog work (#437); #433 keeps scoring ownership. Cross-link the issues and add the short interface clarification to #433, rather than merging plans or changing its unreviewed implementation branch.
