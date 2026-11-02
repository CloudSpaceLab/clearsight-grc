# People & access — source layout and integration evidence

26 October 2026 · Tracking [#462](https://github.com/CloudSpaceLab/clearsight-grc/issues/462) · [Implementation and release record #589](https://github.com/CloudSpaceLab/clearsight-grc/pull/589).

**Partial v2 layout integration. Whole-screen acceptance remains 0/19.** This evidence compares the original HTML with the implemented People, Roles, Approval routes and Delegations views. It verifies the recorded presentation and read-only interactions with isolated synthetic transport. Four-actor execution, live backend authorization, source-to-report reconciliation and production acceptance require their own evidence.

The [31 October integration review](integration-2026-10-31/README.md) incorporates current main through #598, preserves the later Work APIs/search, and repeats the 92 People, 10 Configuration and 22 scoped accessibility cases. It supersedes the branch-integration checkpoint only; the original source and baseline evidence below remain unchanged.

## Authority and provenance

The layout authority is [`people-access-preview.html`](../../design/references/clearsight-2026-10/source/clearsight-people-access-designs/people-access-preview.html), SHA-256 `3300853e7959e06f76e52b484e77c71c6f0c2af876351861921e0dead0aaa7f4`. The companion HTML, CSS, JavaScript, notes and archive manifest remain unchanged. The [decision](../../design/decisions/2026-10-26-people-access-v2.md) maps the source anatomy to existing runtime owners and records supported-data substitutions.

The before build is main `ea84283f913755be6dc1b8e366282469745b3e9a`, tree `28f0b24ef1da098b46bfd345c466cecd79051675`. This includes the released shared-tabs and Privacy tranches. The [baseline inputs](baseline-inputs/README.md) retain the verbatim fixture, entry point and capture script used with the unchanged baseline customer components.

The final published runtime/scripts revision is `1bde850512b9eb4045b23f306322a29ed0859292`, equivalent to local `2b2edf2ff1c3f4892dbe2c388973ae8a0cb86d3a`, tree `b8ddfceddb8b767bd060362a2028c8a90de784e2`. Source and image hashes, build digest and capture-input digest are recorded in [verification-summary.json](verification-summary.json). The [integrity receipt](after/integrity-review.json) verifies 322 image, build-asset and capture-script byte/hash pairs, including 173 build assets. Every final runtime image is bound to the same immutable build by the [runtime manifest](after/manifest.json). Git publication used equivalent verified trees; local and published commit identifiers differ because their commit metadata differs. Subsequent commits retain evidence and documentation only. PR #589 records the final reviewed branch head, exact hosted gates, merge and exact-main demo-release receipts.

## Retained evidence

| Evidence | Coverage | Receipt |
| --- | --- | --- |
| Before runtime | 30 baseline frames across directory, organization and governance at all five widths in light/dark | [Before manifest](before/manifest.json) |
| Original HTML | 32 light-theme frames: four views at five widths plus selected context at three stacked widths | [Source manifest](approved-reference/manifest.json) |
| Final runtime | 92 states, 95 PNGs including three phone continuation frames | [Runtime manifest](after/manifest.json) |
| Source/runtime bindings | 64 pairs covering light/dark runtime layouts and stacked context | [Source comparison index](approved-reference/README.md) |
| Accessibility | 22 scoped cases, zero axe violations and zero undersized targets | [Raw accessibility report](after/accessibility.json) |
| Configuration | 10 cases / 12 PNGs: cold Organization entry at 1440/600/390, retained overview, routing checks and specialist destinations | [Configuration report](after/configuration-review.json) |
| Customer build | Existing bundle budgets and fixture exclusion checked on a fresh output directory | [Build receipt](after/customer-build.json) |
| Named independent review | People/Role/root owner and governance scopes reviewed separately | [People/Role review](reviews/people-role-review.json), [governance review](reviews/governance-review.json) |
| Rejected checkpoints | Source mismatch, semantics, secondary navigation, unavailable summary and capture-paint findings | [Review findings](review-findings/) |

The runtime matrix contains 40 layouts, 24 stacked selected contexts and 28 behavior/state cases. It covers browser Back/Forward, queryless selection, retained filters, empty and missing records, restricted and partial reads, recovery, read-only commands, a 50-profile result, governance composers and full policy definitions. The original design is light-only; its pairings with dark runtime frames check anatomy, not an unavailable original dark palette.

## Source composition and measured result

All four source tabs are retained in order: **People / Roles / Approval routes / Delegations**. One heading and a compact four-fact strip precede selectable records and their context. At 1024px the list and context stack in source order. Selecting a record on a narrow screen scrolls and focuses its context. The original common-menu variants are adapted through the [21 October shared navigation decision](../../design/decisions/2026-10-21-v2-navigation-adaptation.md).

The additional Configuration rail and repeated headings are removed from this specialist workspace. Settings remains in the shared footer, with a Settings return link. Organization, reporting lines, provisioning, escalation and routing checks keep their existing destinations. Their navigation uses the shared WorkspaceSwitcher stylesheet from the shared component layer on cold entry; its labelled selector replaces the desktop row at 760px.

The table measures the first People name and the complete first selectable record. Coordinates are CSS pixels at the recorded viewport, with the same browser font environment.

| Width × height | Before name top | Original name top | Final name top | Final complete row |
| --- | ---: | ---: | ---: | --- |
| 1440 × 900 | 1040.91 | 604.50 | 608.05 | 593–669 |
| 1280 × 800 | 1040.91 | 601.50 | 666.05 | 646–742 |
| 1024 × 768 | 1125.67 | 732.50 | 699.05 | 684–760 |
| 768 × 900 | 1437.17 | 721.50 | 663.05 | 648–724 |
| 390 × 844 | 1726.89 | 686.50 | 728.55 | 704.5–814 |

The complete first record fits at every approved width. At 390px the baseline document extended to 553px; final cases fit their viewport. People search fills its assigned toolbar row at 1280/1024/768/390. The 1024px heading and wave are both 78px tall and end at y202, clear of the tabs. After scrolling desktop content by 500px, the selected-person context begins at y96, below the context bar ending at y68.

These are source-shaped adaptations rather than pixel equivalence. The original is rendered with the available font fallback. Runtime keeps shared semantic colors, accessible controls, 44px touch targets and the common authenticated shell. Its 1280px search/filter arrangement wraps earlier than the original to fit shared controls. Stored status/quality labels and actual content differ from illustrative prototype data. Phone unavailable or stale governance summaries stack label and quality so uncertainty remains readable; the ready-state geometry remains compact.

## Existing data and command ownership

People search now includes every returned profile, removing the previous eight-row display truncation. The API remains `/api/v1/access/overview?limit=50`; people/groups are tenant-wide while positions and mappings keep current-entity scope. Role definitions remain bounded by the existing API. Counts describe the returned list, and search/filter options operate on that population. The UI does not imply complete role membership, effective access or sign-in eligibility. Missing provenance reads **Source not returned**; a username is not assumed to be an email address.

Policies and delegations request 51 records and display at most 50, using the additional record only to disclose continuation. Selected policy context leads with actual current-version rule facts: responsibility, selector or lifecycle, entity/object/decision scope, minimum materiality and priority. Full returned JSON and its checksum remain inspectable. Approval is disabled when the policy definition cannot be reviewed. Delegation object, decision and materiality restrictions, exact UTC effective windows, makers/checkers and version history retain their actual contracts.

Verified tenant/entity/actor changes remount the owner. Retained views recheck on return, cancel stale reads and ignore late responses from inactive visits. Denial clears the population; successful empty results remain distinct; transient failures may retain explicitly previous same-scope records with affected commands disabled. Existing expected-version, candidate selection, maker/checker and server authorization contracts remain unchanged.

Navigation closes policy/delegation and Organization editor portals, including Add position, Add area and provisioning-source editors. Returning with browser Forward leaves those editors closed. One-time provisioning results cannot leak into an inactive view. No new identity, authorization, invitation, routing or audit store was introduced, and no unsupported invitation, suspension, last-active or effective-authority totals were copied from the prototype.

## Findings corrected before acceptance of the scoped review

The retained history distinguishes automated checks from direct visual review. A passing capture count was not treated as approval.

- The first review corrected the old full-page hierarchy, truncated directory, control ownership, hidden selected context and retained-query behavior.
- Checkpoint `712cd431` was rejected because the shared search field retained a 420px cap and the decorative wave exceeded the tablet heading. The public search-width contract and contained wave geometry now have explicit assertions.
- Checkpoint `75cb676d` was superseded after the loaded summary needed an accessible group role and the Delegations metric label split on phones. The corrected label is Records; the summary role removes the ARIA finding.
- Checkpoint `d2cbf333` was rejected by independent review despite passing its existing automated checks: Organization cold entry depended on a stylesheet loaded by unrelated screens, and unavailable governance labels broke into fragments on phones. Shared stylesheet ownership and state-specific use of the existing metric geometry token correct both.
- The added cold-entry phone check found a zero-width Positions & roles heading beside its full-width filters. A 600px probe exposed the same collision and fragmented table actions above the initial 560px fix. The owning header now stacks at the existing 900px grid breakpoint, and identity tables reflow into labelled records at 680px. The visibility check and intermediate-width case remain in the final gate.
- A policy continuation screenshot intermittently painted blank below its rationale even though DOM geometry, hit testing and styles were valid. Reproduction isolated a post-scroll browser paint race. The final capture settles paint, checks consecutive image stability and verifies content inside the action text region. The [rejected frame and diagnosis](review-findings/d2cbf333-superseded.json) are retained.

The separate [Organization range review](after/organization-range/organization-range-review.json) retains header and table images at 390/600/768. It verifies readable title/filter order, whole action labels, 44px controls, labelled cards at 390/600 and table cells at 768, with no horizontal clipping.

## Review scope and accessibility limitations

The two named reviewers independently checked the other agent’s feature owner: `/root/people_review` covers People/Role/root ownership and secondary navigation; `/root/people_runtime` covers governance views and composers. Their JSON receipts list exact revisions, inspected filenames/hashes, original and baseline comparisons, byte-identical carry-forwards and limitations. Root also inspected source/runtime comparisons and the specific corrected frames. The governance receipt discloses that its reviewer authored adjacent shared People CSS; that shared layout also received root review.

The final scoped accessibility report has zero axe violations and zero controls below the required target geometry. Raw color-contrast uncertainty remains preserved: 17 cases / 38 axe incomplete nodes, all color-contrast only. Supplemental rendered readings retain 1,413 PASS, 157 INCOMPLETE, 110 REVIEW and 20 INACTIVE. It includes decorative select chevrons, clipped nonselected labels in the scrollable tab strip, decorative-wave background overlap and surfaces involving backdrop filters. Reviewers classify those readings individually; opaque essential text and field boundaries are inspected separately. Disabled actions retain their inactive classification. This is not a blanket WCAG-conformance claim.

The recovery PNG is taken after successful Retry; the preceding 503 is established by transport and DOM evidence. Missing-selection scenarios verify the unknown query, cleared context and Return recovery; the People missing inspector lies below the long list in the retained screenshot, and Role context is only partly visible. Those frames do not prove every offscreen missing-state style. All recorded command counts remain zero: reviewing lists, candidates and disabled controls does not establish live command authorization or real approval execution.

## Validation and release

Affected identity, governance, routing, history and customer-copy regressions were run alongside typecheck, the shared UI contract gate, fixture isolation and a fresh customer build. The full local web suite passed 2,219 tests before the last secondary-owner guards; subsequent focused suites cover those guards and the final layout changes. The exact final hosted suite and UI artifact are authoritative for the final PR head and are linked in [#589](https://github.com/CloudSpaceLab/clearsight-grc/pull/589).

The final fresh customer build meets the existing budgets:

| Budget | Measured | Limit |
| --- | ---: | ---: |
| Initial JavaScript, gzip | 161,876 bytes | 196,608 bytes |
| Initial CSS, gzip | 32,097 bytes | 32,768 bytes |
| Largest JavaScript chunk, raw | 566,368 bytes | 614,400 bytes |

People evidence markers are absent from customer JavaScript and source maps. The build uses a fresh output directory and contains one ConfigureWorkspace JavaScript chunk, avoiding retained chunks from earlier builds.

The before and final fixtures deliberately differ: final evidence adds actual policy definitions/checksums, a bounded candidate response and a precise same-day pending delegation window. The core population remains comparable. Both fixture hashes and the separate baseline inputs are retained; no claim of byte-identical fixture data is made.

Implementation, source comparison and demo release remain separate from whole-screen operational acceptance. The production cutover block and 0/19 register remain unchanged.
