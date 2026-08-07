# UI and onboarding acceptance matrix

This matrix audits the currently implemented ClearSight application surfaces against the documented banking use cases. It distinguishes a finished interaction path from a future enterprise capability.

## Acceptance rules applied to every surface

Every implemented workspace must provide:

1. a clear object, current state, reason, accountable role, due date and next valid action;
2. loading, live, empty, filtered-empty, unavailable and stale-data behavior without invented counts;
3. actor-scoped reads and no disclosure of restricted record existence;
4. direct navigation to the exact Program, issue/change or evidence request;
5. complete material facts, contradictions, requirements, actions and closure blockers without silent truncation;
6. keyboard-visible focus, semantic controls, reduced-motion support and responsive reflow;
7. plain banking language before internal domain codes;
8. explicit reference-data labels whenever stakeholder demo records are shown;
9. a first-run guide that opens the real target rather than merely describing it;
10. a restartable guide that can be skipped without permanently hiding help.

## Role-aware first-run journeys

| Role profile | First meaningful task | Implemented guide path | Acceptance status |
|---|---|---|---|
| Executive risk/compliance leader | Open a material assigned record and understand why it is not current | Today brief → exact assigned item → Program status reasons → evidence-backed interpretation | Implemented |
| Program/control/risk owner | Understand current Program state and resolve the smallest evidence gap | Programs → first Program detail → evidence queue → status semantics | Implemented |
| Reviewer/independent challenger | Inspect facts and contradictions without conflating work completion with verified outcome | Today → issue detail → evidence queue → independent outcome explanation | Implemented |
| Authorizer/signatory | Confirm why authority resolved to them before deciding | Today → approval route → full decision record → outcome-check explanation | Implemented |
| Evidence respondent | Understand why selected and answer only unresolved questions | Today → exact request → prefilled known facts → response form | Implemented for current request form; redirect/delegation remains future work |
| Configure administrator | Inspect configuration health and understand governed setup boundaries | Configure → approval route → Imports → maker-checker continuation | Implemented orientation; complete configuration builders remain future work |
| Auditor/read-only reviewer | Trace current state to source, decision and outcome history without changing state | Programs → issue detail → imported source lineage → restricted-existence explanation | Implemented orientation; dedicated immutable audit-history UI remains future work |
| Unmatched/general user | Understand the core application model safely | Today → Programs → Work → exact-record semantics | Implemented fallback |

Guide selection is server-authoritative. Signed identities and development identities expose normalized role codes. The client requests the guide matching the verified role set and stores progress independently for each guide version.

## Primary navigation and routing

| Interaction | Previous risk | Implemented correction | Evidence |
|---|---|---|---|
| Today Program action | Opened only the Programs workspace | Hash route includes the exact Program ID and opens its detail | `#programs/{id}` |
| Today issue/change action | Opened only the generic Work workspace | Hash route includes the exact Matter ID and opens its detail | `#work/matters/{id}` |
| Today evidence action | Opened only the evidence tab | Hash route includes the exact request and expands its detail | `#work/evidence/{id}` |
| Browser refresh/back | Active record could be lost | Workspace and target are reconstructed from the hash route | Route parser and hash-change listener |
| Demo-off mode | Explore/sample shortcuts could remain visible | Explore, reference routes and sample actions are omitted; operational surfaces remain | Context capability gate |
| Mobile navigation | Desktop-only sidebar created a narrow-screen dead end | Fixed bottom navigation exposes every enabled workspace | Responsive shell |

## Today

| State/use case | Expected result | Status |
|---|---|---|
| Live assigned work | Counts and cards derive from actor-scoped current work | Implemented |
| Unknown population | Shows em dash and “population not connected,” never zero-as-complete | Implemented |
| No assigned items | Calm empty state without success overclaim | Implemented |
| Service unavailable | No current counts or stale claims are shown | Implemented |
| Due-soon calculation | Includes future items due within four days; excludes overdue items | Implemented |
| Approval route | Opens explainable route panel with legal entity, responsibility and policy version | Implemented |
| Evidence shortcut | Uses a real seeded/current request only when available | Implemented |
| Exact-record action | Opens Program, Matter or evidence request encoded in the item | Implemented |

## Programs

| State/use case | Expected result | Status |
|---|---|---|
| List and pagination | Bounded summary page with explicit “more available” state | Implemented |
| Search and status filter | Search remains user-submitted; filtered empty is distinct from no data | Implemented |
| Current status | Plain label and semantic severity derive from the recorded state | Implemented |
| Status reasons | All current reasons are rendered | Implemented |
| Requirements | Full statement and source anchor are shown; no silent slice/truncation | Implemented |
| Evidence checks | Claim and minimum coverage are shown | Implemented |
| Direct target | Target Program expands and scrolls into view | Implemented |
| Detail failure | Inline retry without losing list context | Implemented |
| Full Program setup/editor | Governed create/edit/approval experience | Future productization work |

## Issues and changes

| State/use case | Expected result | Status |
|---|---|---|
| Open/filter/search | Bounded actor-visible list with distinct no-results behavior | Implemented |
| Known facts | All material facts are rendered, including structured values | Implemented |
| Missing facts | Complete list is shown | Implemented |
| Contradictions | Complete list is shown with warning semantics | Implemented |
| Decisions | Type, status, selected option and rationale are visible | Implemented |
| Actions | Full title, description and state distinguish implemented work from verified outcome | Implemented |
| External response | Purpose, audience and transmission state are visible | Implemented |
| Outcome check | Expected outcome and latest result are separate from action completion | Implemented |
| Closure | Ready state or complete blocker list is shown | Implemented |
| Direct target | Target issue expands and scrolls into view | Implemented |
| Operational mutation forms | Complete decision/action/response/verification write interfaces | Future productization work |

## Sources and evidence

| State/use case | Expected result | Status |
|---|---|---|
| Source health | Type, authority class, freshness and last success are visible | Implemented |
| Source unavailable/unknown | No claim of current evidence | Implemented |
| Request search/filter | Title, purpose, audience and why-selected are searchable | Implemented |
| Known facts | Existing information is shown before unanswered fields | Implemented |
| Exact request | Target request expands and scrolls into view | Implemented |
| Required fields | Submission remains disabled until required fields are present | Implemented |
| Submission receipt | Submission is recorded without claiming evidence sufficiency | Implemented |
| Wrong recipient/delegation | Redirect, delegate or report incorrect scope | Future productization work |
| Production scanning/storage | Malware scanning, encrypted object storage, retention and legal hold | Future production work |

## Imports

| State/use case | Expected result | Status |
|---|---|---|
| Intake | File, purpose and source type are explicit | Implemented |
| Lineage | Original digest, artifact state and extraction method remain visible | Implemented |
| Limitations | Unsupported or unscanned conditions are shown prominently | Implemented |
| Proposal review | Exact source quote, anchor, confidence and accept/reject controls | Implemented |
| Accepted proposal | Records follow-up intent only; does not create a legal conclusion | Implemented |
| Empty/unavailable | Distinct bounded states with no analysis overclaim | Implemented |
| PDF/OCR | Original retained; extraction is explicitly unsupported | Future adapter work |
| Governed conversion | Convert accepted proposal into approved Requirement/Program/Matter | Future productization work |

## Configure

| State/use case | Expected result | Status |
|---|---|---|
| Policy visibility | Active policy name, code, version and effective status | Implemented |
| Integrity findings | Missing owner, selector, delegation and route failures are surfaced | Implemented |
| Calm empty | “No blocking gaps” only after the integrity service returns live data | Implemented |
| Workflow ownership | Open task, responsibility and scope are visible | Implemented |
| Projection health | Current projection state and governed reconciliation action | Implemented |
| Full RBAC/authority/escalation builders | Draft, simulate, maker-checker approve, activate and roll back | Future productization work |
| Directory import/sync | OIDC/SAML/SCIM/LDAP/AD and controlled spreadsheet mapping | Future productization work |

## Responsive and accessibility evidence

The release gate includes:

- strict TypeScript checking;
- rendered-state tests for demo-on/demo-off and exact-record navigation;
- role-aware onboarding action and restart tests;
- existing import-state Vitest coverage;
- axe semantic checks;
- visible focus for controls and detail summaries;
- modal focus containment and Escape behavior;
- mobile bottom navigation;
- single-column reflow for cards, toolbars, details and guide actions;
- reduced-motion handling.

## Stakeholder deployment boundary

The GitHub Pages build is an explicitly labelled static stakeholder demonstration. It uses the production React components and routes with retained reference records supplied by a build-time static transport. It does not contain customer data, does not claim a live backend, and must never be used as a production bank deployment.
