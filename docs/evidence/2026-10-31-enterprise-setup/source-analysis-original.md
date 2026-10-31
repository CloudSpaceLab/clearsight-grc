# Enterprise Setup → Organization source and baseline review

Reviewer: GPT-6 Astra Pro, People integration agent. Date: 31 October 2026.

## Main finding

The approved Organization page leads with a searchable hierarchy, a compact structure/coverage column, and selected-unit context. The existing runtime leads with a data-boundary mutation form inside People & access and then stacks area and position tables. Its first area and position records are below the first viewport at every required width. A whole-page adaptation must change the page identity, local navigation, hierarchy/list anatomy, contextual selection, and responsive replacement together.

## Immutable source identity

Directory: `docs/design/references/clearsight-2026-10/source/clearsight-enterprise-setup-designs/`

| File | SHA-256 |
| --- | --- |
| enterprise-setup-preview.html | cc7542e2a3047338b1d2fbbbe33756adbc9ee8c2084fb5920834da8aeedc04c7 |
| enterprise-setup.html | 5182fcd83331fafd42c7711fa6eafdbebb5d255dec51b6b73d42217e8ebdfd3a |
| enterprise-setup.css | 2a467e326ce02fe7725183d0a4906cdc8280664909faf5b89885d124a78ee59a |
| enterprise-setup.js | 9c7d5ba79279f7d8f29258b8ec7cca915937ffc169f4f383849461a9e2a07aeb |
| enterprise-setup-design-notes.md | 850af313dfdab1006bbb891e86c9486391757ae1fc90ad3869d42f9d4169106a |

The self-contained preview is the rendered authority. No source file was changed. External requests were blocked for original renders; the environment supplies fallback fonts. The original is light-only.

## Original anatomy

1. Enterprise setup heading, breadcrumb, decorative wave and heading actions.
2. Compact setup-progress banner.
3. One local surface menu: Organization, Groups, People & access, Jurisdictions, Frameworks, Data imports.
4. Primary organization grid: hierarchy card with query and Add control; adjacent Scope coverage and Structure cards.
5. Lower primary cards for Groups and organization import/source actions.
6. Current selection context card with type, code, parent, owner and member facts; explicit Edit unit / Manage groups actions. A separate Next actions card follows it.

The original starts with Technology selected. Selecting a unit updates its context. Parent selection also toggles expansion in the prototype, but production can separate expansion from selection using existing shared controls. This preserves explicit selection, keyboard access and a reliable phone context transition.

## Measured responsive replacement

Dimensions below are CSS pixels from the immutable original rendered in Chromium 141.0.7390.37.

| Viewport | Hierarchy | Summary column | Selected context | Main replacement |
| --- | ---: | ---: | ---: | --- |
| 1440 × 900 | 565 | 275 | 290 beside primary | Three content columns; 14px inner gap and 18px context gap |
| 1280 × 800 | 688 | 300 | 493 below primary | Context and Next actions become two lower columns |
| 1024 × 768 | 618 | 270 | 443 below primary | Same primary pair; shared source menu collapses to icon rail |
| 768 × 900 | 646 | Below hierarchy | 315 below primary | Primary hierarchy/summary stack; lower and context cards retain two columns |
| 390 × 844 | 307 | Below hierarchy | 307 below primary | All cards stack; local tabs scroll edge-to-edge |

Source CSS breakpoints are 1550, 1300, 1024, 780, 690 and 560px. Context moves below at ≤1300; hierarchy/summary stack at ≤780; all lower/context cards stack at ≤560. The global menu follows the later shared navigation decision rather than the prototype’s obsolete destination list.

The original first hierarchy row begins at y=482 for 1440/1280/1024, y=476 for 768, and y=592 for 390. Ten/eight/seven/ten/six rows respectively intersect the first viewport. Source hierarchy rows are 43px tall; implementation must retain the required 44px shared interaction minimum.

## Baseline runtime findings

Route: `#configure/access/positions`.

| Width | First area row y | First position row y | Area/position rows initially visible |
| --- | ---: | ---: | --- |
| 1440 | 1042 | 1510 | 0 / 0 |
| 1280 | 1042 | 1606 | 0 / 0 |
| 1024 | 1024 | 1588 | 0 / 0 |
| 768 | 1028 | 1733 | 0 / 0 |
| 390 | 1203 | 2247 | 0 / 0 |

Current content is governed and useful but needs rearrangement. Existing scope and position managers already provide version checks, impact previews, independent approval, move/retire restrictions and focused editors. Those contracts should remain the command owners. Move the large data-boundary and position-management surfaces behind contextual focused sheets and keep their state/permission semantics intact.

## Recommended data adaptation

- Use the original hierarchy/search/context composition with existing scoped organization records. Keep legal entity distinct from managed area records. Resolve parent links safely; a missing parent or cycle must not silently remove an otherwise returned row.
- Use loaded structure counts and pending-change review in the original summary/banner regions. Never reproduce prototype readiness totals, ownership percentages, region counts, owners or member totals absent from the returned contracts.
- Display partial/truncated populations explicitly. Existing overview reads use `limit=50`; organization scopes can expose a separate truncation flag. Search applies to returned scopes and must not imply a complete server-side search.
- Preserve the source six-tab order. People and Frameworks can hand off to existing routes. Unsupported cross-team Groups and jurisdiction setup require honest unavailable states; directory role mappings are not equivalent to the prototype’s reusable organization groups.
- Retain the shared application shell and Settings current state. Avoid duplicate People tabs around the Enterprise Setup page.
- Replace unsupported Invite/import-hierarchy actions with actual supported destinations or unavailable states. Existing document import must not be presented as automatic hierarchy ingestion.
- Put selected-area facts and Edit/Move/Retire controls in the context card. Imported `ORGANIZATION_UNIT` rows require an approved managed-type change before move/retire. Pending changes retain independent-maker checks.
- On widths ≤1300, selection should move focus to the context card and reveal it without making document-wide horizontal scrolling necessary. Back/Forward and `unit=<id>` must restore selected state.

## Preserved evidence and limits

Root: `/tmp/clearsight-organization-before-20261031/`.

- `before-manifest.json`: 15 original/default-runtime scenarios and 54 hashed PNGs. Original at all five widths; runtime in both themes, including full-page, areas and positions captures. Zero captured page errors and zero document overflow.
- `source/organization-light-{width}.png`: original first viewport at 1440/1280/1024/768/390. Matching `-full.png` captures preserve the whole composition; `-context.png` captures preserve lower context at ≤1280.
- `runtime-before/organization-{light|dark}-{width}.png`: old runtime first viewport. `-full`, `-areas` and `-positions` variants preserve its full content and editors’ location.
- `runtime-build/` and `baseline-build.json`: isolated copy of the existing evidence build, 173 hashed files. Repository HEAD at copy was c40f4edd7ad4107fc2b36ebfd65af87bd5d5502b, tree 90106281c8a5446963ab582799c07e5b3c351ff0.
- `baseline-source-map-verification.json`: embedded source-map content for OrganizationInventory, OrganizationScopeManager, OrganizationPositionManager, IdentityAccessPanel and PeopleAccessSection exactly matches c40f. This is specific source verification, not an invented build stamp.
- `fixture-matched/enterprise-setup/receipt.json`: permanent injected fixture rendered against the unchanged copied build, with 10 runtime layouts, five original renders and 19 hashed PNGs. The same fixture can be used for AFTER comparison.

These are isolated synthetic UI fixtures, not live actor/authorization evidence. The source itself is illustrative. The scoped review does not advance whole-screen production acceptance or the 0/19 register.
