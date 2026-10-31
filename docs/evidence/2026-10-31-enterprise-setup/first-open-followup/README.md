# Enterprise Setup: preserve the first sheet opening

**The runtime regression, full local web suite, and fresh CI-browser capture pass.** This directory records the evidence available before the final PR branch update and hosted validation. See [PR #599](https://github.com/CloudSpaceLab/clearsight-grc/pull/599) for current release status. Whole-screen acceptance remains **0/19**.

This is a separate runtime correction from the earlier [Select/Escape capture synchronization](../hosted-followup/README.md). Both earlier failures and their original evidence remain unchanged.

## Failure and correction

[CI run 41203345467](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/41203345467) failed one web test at head `c1d5185e769cf2c555b6fc83bafd1beeedd61d7f`. Both job logs printed synthetic checkout `8b3712b67036f3d74da9f09bcc24737ab8b058eb`, whose tree `23a20a10a165a84b9f3be111a358d21a1374b054` matched that head. The web job passed 276/277 files and 2,298/2,299 tests; the first progressive workflow test could not find Ada Okafor after opening Positions & roles. The failure DOM contained the Organization tree without the sheet. The backend job passed. [The exact CI record](ci-failure.json) retains job identities, timestamps, counts and the limits of that checkpoint.

The local A/B probe established a real ordering defect: the two initial passive panel-reset effects could clear the first opening as the asynchronously loaded inventory became available. The same one-click probe retained the sheet when only those resets became layout effects. The correction preserves selected-area and permission-revocation resets while completing them before the controls become interactive. It changes the inventory component and adds two focused regressions; it adds no retry click, delay or timeout increase. The exact event ordering in the earlier hosted failure was not instrumented.

[The regression record](runtime-regression-verification.json) preserves the A/B observations, 19 passing focused tests and passing typecheck. Its positive results are a compact record of actual execution responses, not reconstructed raw logs. [The retained negative-control log](old-effects-regression-red.log) shows the new single-click test failing when only the two prior passive resets are restored. Permission revocation, selected-area changes, read-only views and absence of commands are covered by the focused tests.

## Exact fresh-capture identity

| Identity | Value |
| --- | --- |
| Local runtime commit | `aa666af542fdcba2f2883a58821434f3fbc18740` |
| Equivalent remote commit object | `3a32ab85e0cb0615903f76fcac1c08362db306c8` |
| Shared tree | `9e670cd7477ddd25b0e5c9e1d01961ae3ecfbacc` |
| Browser | Chromium `140.0.7339.16`, matching the CI pin |
| Build digest before and after capture | `40e12a493b9e595206607cba4729db53c7d685dc39cd5169c330fc268542649c` |
| Receipt SHA-256 | `f9d493fcf9dec86f7d07883e45636fdd825dbfd85ab67ea23b703c7ac10999eb` |

The GitHub Git commit API independently confirmed the remote object's tree and parent `c1d5185e769cf2c555b6fc83bafd1beeedd61d7f`. That object was not yet the PR branch head at this review. [Provenance](provenance.json) binds every retained file, the immutable original HTML, the three frozen inputs and the preceding capture.

## Verification retained here

| Check | Result and evidence |
| --- | --- |
| Full web suite at the committed runtime | **277 files / 2,301 tests passed**, 104.67s; [unaltered log](full-web-tests.log) |
| Customer and evidence builds | PASS; [customer log](validation/customer-build.log), [evidence log](validation/evidence-build.log) |
| Four Node gates | PASS: forms evidence scenarios, UI flow manifest, runtime fixture boundary and UI contracts; [log](validation/node-gates.log) |
| Bundle budgets | PASS: initial JS 161,940 gzip bytes; initial CSS 32,126 gzip bytes; largest JS chunk 573,982 raw bytes; [budget](validation/budget.json) |
| Fresh Enterprise matrix | **10 layouts / 11 behaviors / 42 PNGs**, all bytes retained; [receipt](capture/enterprise-setup/receipt.json), [capture log](validation/capture.log) |
| Frozen-input replay | PASS with no failures; [archived review](archived-review.json) |
| Image comparison | **41/42 byte-identical** to the preceding corrected CI-browser capture; [all-image comparison](validation/image-comparison.json) |
| Named visual review | PASS within the stated scope; [review](named-review.md), [independent comparison record](validation/visual-review.json) |

The [validation summary](validation/validation-summary.json) is retained unaltered, including its original scratch paths. The durable equivalents of those logs are linked above and bound by identical hashes in provenance. The capture's fixture, capture script and reviewer are exact commit snapshots under [inputs](inputs/). Replaying from that directory reads the frozen files rather than the current working source:

```sh
cd docs/evidence/2026-10-31-enterprise-setup/first-open-followup/inputs
node --input-type=module -e 'import fs from "node:fs/promises"; import {reviewEnterpriseSetupEvidence} from "./scripts/review-enterprise-setup-evidence.mjs"; const r=JSON.parse(await fs.readFile("../capture/enterprise-setup/receipt.json","utf8")); const v=await reviewEnterpriseSetupEvidence("../capture",{revision:r.revision,digest:r.build.digest}); console.log(JSON.stringify(v,null,2)); if(v.failures.length)process.exitCode=1;'
```

The fixture makes no material commands. This evidence does not establish live authorization, four-actor execution, complete WCAG conformance or production acceptance. The full hosted generic flows, final checkout/artifact identity and exact-SHA deployment require their own final-head records.
