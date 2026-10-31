# Enterprise Setup → Organization evidence

**Status: final scoped synthetic UI gate and named visual review PASS. Whole-screen production acceptance remains unclaimed.**

Reviewer: GPT-6 Astra Pro, `/root/people_integration`. Date: 31 October 2026.

This tranche adapts `#configure/access/positions` to the approved Enterprise Setup Organization composition. The preserved BEFORE runtime uses the earlier People & access heading, organization switcher, data-boundary form, and stacked area/position managers. The BEFORE records establish the baseline; they are not a passing V2 cutover receipt.

## Final scoped result

The final gate passes **10 layouts across five widths and two themes, 11 behavior scenarios, and 42 retained PNGs**. The [named review](named-review.md) records the source/before/after inspection, the corrected phone and partial-list disclosures, and the accessibility limits. [Final provenance](final-provenance.json) binds every image, source, input snapshot and review checkpoint.

| Item | Final identity |
| --- | --- |
| Local reviewed revision | `613f07f705c383201f576504f7debf7b84592151` |
| Published equivalent revision | `74603165763e69d00b8dc2cb8e804225053f6a68` |
| Exact tree | `c9adf95492efe0942d3d5803d9184ddb19ae6183` |
| Build digest, unchanged during capture | `28e6d798572e8d37da39a977b639694fb68842515f73b4011517ff51cfe3d4b1` |
| Final receipt SHA-256 | `3ac8caa8a93ba6eb2e5bf8e6440e6a7c75f338835be5f468b39a82ba348fafd0` |
| Canonical image manifest SHA-256 | `160f61e07bd372c868a79b47a8005b366d742f2380f95882b7845afda9a65ad2` |
| PNG count / bytes | 42 / 4,652,719 |

The complete [receipt](after/enterprise-setup/receipt.json), [gate review](after/gate-review.json), [image manifest](after/image-manifest.json), and [archived replay](after/archived-review.json) are retained. Every original image filename resolves. All nine source images are byte-identical to the BEFORE source images; keeping both sets of filenames preserves replay, while Git stores identical blob content once.

| Width | AFTER light | AFTER dark | Lower context light | Lower context dark |
| --- | --- | --- | --- | --- |
| 1440 | [Light](after/enterprise-setup/layout-light-1440.png) | [Dark](after/enterprise-setup/layout-dark-1440.png) | Beside hierarchy | Beside hierarchy |
| 1280 | [Light](after/enterprise-setup/layout-light-1280.png) | [Dark](after/enterprise-setup/layout-dark-1280.png) | [Context](after/enterprise-setup/layout-light-1280-context.png) | [Context](after/enterprise-setup/layout-dark-1280-context.png) |
| 1024 | [Light](after/enterprise-setup/layout-light-1024.png) | [Dark](after/enterprise-setup/layout-dark-1024.png) | [Context](after/enterprise-setup/layout-light-1024-context.png) | [Context](after/enterprise-setup/layout-dark-1024-context.png) |
| 768 | [Light](after/enterprise-setup/layout-light-768.png) | [Dark](after/enterprise-setup/layout-dark-768.png) | [Context](after/enterprise-setup/layout-light-768-context.png) | [Context](after/enterprise-setup/layout-dark-768-context.png) |
| 390 | [Light](after/enterprise-setup/layout-light-390.png) | [Dark](after/enterprise-setup/layout-dark-390.png) | [Context](after/enterprise-setup/layout-light-390-context.png) | [Context](after/enterprise-setup/layout-dark-390-context.png) |

Full-page AFTER renders: [1440 light](after/enterprise-setup/layout-light-1440-full.png), [1440 dark](after/enterprise-setup/layout-dark-1440-full.png), [390 light](after/enterprise-setup/layout-light-390-full.png), and [390 dark](after/enterprise-setup/layout-dark-390-full.png). The [truncated-list frame](after/enterprise-setup/truncated.png) shows the omitted-area qualifier inside the phone viewport.

### Validation checkpoints

[Compatibility evidence](compatibility/README.md) retains the 10 Configuration and 92 People scenarios from runtime `2264dfd18bc070fe6d5260422272b1533966b8dc`. [The earlier validation receipt](validation/receipt.json) records the 277-file/2,299-test suite at that checkpoint. These historical results are not relabeled as final-runtime runs. [The final-runtime supplement](validation/final-runtime-supplement.json) separately records the truncation correction's 15 focused tests, customer/evidence builds, budgets, and four Node gates at `613f07f7`. Final hosted CI/UI and release verification remain separate gates owned by the parent rollout.

### Replay the archived gate

The final snapshots preserve their `src/` and `scripts/` paths, so replay uses the exact archived reviewer and inputs rather than later active files. No browser, build assets or installed web dependencies are needed for this receipt replay:

```sh
cd docs/evidence/2026-10-31-enterprise-setup/final-inputs
node --input-type=module - <<'NODE'
import { reviewEnterpriseSetupEvidence } from './scripts/review-enterprise-setup-evidence.mjs';
const result = await reviewEnterpriseSetupEvidence('../after', {
  revision: '613f07f705c383201f576504f7debf7b84592151',
  digest: '28e6d798572e8d37da39a977b639694fb68842515f73b4011517ff51cfe3d4b1'
});
if (result.failures.length) throw new Error(JSON.stringify(result));
console.log(result.detail);
NODE
```

## Baseline identity

| Item | Identity |
| --- | --- |
| Initial main HEAD at build preservation | `c40f4edd7ad4107fc2b36ebfd65af87bd5d5502b` |
| Initial main tree | `90106281c8a5446963ab582799c07e5b3c351ff0` |
| Preserved build digest | `3bfc7fcf130c22bcde0ece385a909632d872cb5c47f7f74867e3c41651548c51` |
| BEFORE receipt SHA-256 | `5fce358fc1b7b061255966c2c587ce7e364044c4b7df315f8018193a266809fc` |
| Runtime matrix | 1440×900, 1280×800, 1024×768, 768×900, 390×844; light and dark |
| Retained images | 23: 10 runtime viewport images, 4 runtime full-page images, 5 original viewport images, 4 original lower-context images |
| Browser | Chromium 141.0.7390.37 |

The digest is SHA-256 of the JSON serialization of the recursively sorted `{path: {bytes, sha256}}` asset map retained in [the receipt](before/receipt.json). [Baseline provenance](baseline-provenance.json) also identifies the archived inputs and immutable source files. All 23 copied images and all three archived inputs were verified against the receipt's byte counts and hashes before copying.

The existing evidence build was copied before tranche edits. The recorded main HEAD is the repository state at copy time, **not an embedded build stamp**. [The build inventory](baseline-build.json) retains hashes of all 173 assets without storing the assets here. [The source-map comparison](baseline-source-map-verification.json) records exact matches to that main revision for OrganizationInventory, OrganizationScopeManager, OrganizationPositionManager, IdentityAccessPanel and PeopleAccessSection. This is specific source verification, not a claim about every bundled source.

## Approved source and paired viewport renders

The rendered authority is [enterprise-setup-preview.html](../../design/references/clearsight-2026-10/source/clearsight-enterprise-setup-designs/enterprise-setup-preview.html), SHA-256 `cc7542e2a3047338b1d2fbbbe33756adbc9ee8c2084fb5920834da8aeedc04c7`. The original HTML, CSS, JavaScript and notes remain unchanged. Their complete hash set is retained in the provenance file and [original source analysis](source-analysis-original.md).

Source images are retained beside each receipt so all relative image references remain valid. Their matching content hashes are recorded in final provenance. The original has only a light theme; no synthetic dark source was produced.

| Width | Original | BEFORE light | BEFORE dark | Original lower context |
| --- | --- | --- | --- | --- |
| 1440 | [Source](before/source-organization-1440.png) | [Light](before/layout-light-1440.png) | [Dark](before/layout-dark-1440.png) | Visible beside the hierarchy |
| 1280 | [Source](before/source-organization-1280.png) | [Light](before/layout-light-1280.png) | [Dark](before/layout-dark-1280.png) | [Context](before/source-organization-1280-context.png) |
| 1024 | [Source](before/source-organization-1024.png) | [Light](before/layout-light-1024.png) | [Dark](before/layout-dark-1024.png) | [Context](before/source-organization-1024-context.png) |
| 768 | [Source](before/source-organization-768.png) | [Light](before/layout-light-768.png) | [Dark](before/layout-dark-768.png) | [Context](before/source-organization-768-context.png) |
| 390 | [Source](before/source-organization-390.png) | [Light](before/layout-light-390.png) | [Dark](before/layout-dark-390.png) | [Context](before/source-organization-390-context.png) |

Full-page BEFORE renders retain the older forms and lists below the viewport: [1440 light](before/layout-light-1440-full.png), [1440 dark](before/layout-dark-1440-full.png), [390 light](before/layout-light-390-full.png), and [390 dark](before/layout-dark-390-full.png).

## Historical capture inputs

The [fixture](baseline-inputs/staticDemoEnterpriseSetupRuntime.js), [capture script](baseline-inputs/capture-enterprise-setup-evidence.mjs), and [reviewer](baseline-inputs/review-enterprise-setup-evidence.mjs) are the exact snapshots identified by the BEFORE receipt. They are evidence inputs, not customer runtime imports. Later improvements to the active gate do not retroactively alter this baseline.

The original analysis is preserved verbatim. Its final paragraph describes the earlier 19-image fixture capture and temporary exploratory paths; the retained receipt supersedes that image count with **23** after four source context images were added. The earlier exploratory capture contained 54 images and was used for the source measurements and baseline findings. Those temporary exploratory images and the runtime build assets are not duplicated here.

## Scope and limits

- The original preview contains illustrative people, ownership, readiness and coverage values. They guide composition, not runtime facts or available operations.
- External requests were blocked for original renders; environment fallback fonts were used.
- The isolated injected fixture provides synthetic organization records. It does not establish backend identity, authorization, material command results, or source-to-report reconciliation.
- The BEFORE receipt intentionally has `baseline: true` and no passing behavior scenarios. It cannot pass the current AFTER reviewer.
- The final ten axe scans have zero violations and retain 28 color-contrast incomplete nodes. The [targeted contrast follow-up](after/contrast-review-ed46dd.json) found no concrete low-contrast blocker for the actual header handoff label and hierarchy carets; its computed CSS and sampled hit-test method is not full WCAG acceptance. Its earlier exact revision is preserved, and all 31 CSS assets match the final build.
- These records do not advance the 0/19 whole-screen acceptance register or remove the production cutover block. The final result is a named, scoped synthetic layout and interaction review.
