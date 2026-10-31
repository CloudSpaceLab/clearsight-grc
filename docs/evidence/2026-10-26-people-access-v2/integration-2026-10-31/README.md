# People & access integration review — 31 October 2026

Reviewer: `/root/people_integration`. Verdict: **PASS for the scoped integration and layout review**. Final hosted CI and UI review remain required before merging [#589](https://github.com/CloudSpaceLab/clearsight-grc/pull/589).

## Integrated revision

The original People branch `c81f876f2dd4c54bc18cf3a8663912e7d4d955ea` merged current main `ff6160a5b0342ad48e32d4da105fe1e8b7c83b2b` as `6491b26dd02f1a925e1dec73423354becef753fb`. The merge preserves the later Work search, verified scope and deadline-count API changes. There were no conflicts or changes to the People runtime, CSS, fixtures or capture scripts.

The fresh browser review captures that integration revision. Commit `113b4d7645d7dd4260939c0f1b8c84869842b4c7` subsequently adds main `96cdab453458b66e097bb5d700d29b9d2f74d03e`, containing only the PostgreSQL count regression from #598. Its web tree is unchanged from the captured revision. Later documentation and evidence commits do not change runtime bytes.

The approved source remains `docs/design/references/clearsight-2026-10/source/clearsight-people-access-designs/people-access-preview.html`, SHA-256 `3300853e7959e06f76e52b484e77c71c6f0c2af876351861921e0dead0aaa7f4`. The original source, baseline and full comparison evidence remain in the [parent index](../README.md).

## Fresh validation

| Check | Result |
| --- | --- |
| Identity, governance, routing, copy and Work search integration | 87 tests in 13 files passed |
| People, fixture-boundary and shared UI evidence contracts | All five Node suites passed |
| TypeScript and evidence build | Passed |
| People browser scenarios | 92 passed, including 40 layouts, 24 selected contexts and 28 behavior/state cases |
| Configuration and secondary destinations | All 10 cases passed |
| Scoped accessibility | All 22 cases passed; zero axe violations, browser errors or undersized primary controls |
| Customer build | Passed; initial JS 161,912 B gzip, initial CSS 32,097 B gzip, largest JS chunk 570,144 B raw |

The browser scenarios cover all four source-order tabs at 1440/1280/1024/768/390 in light/dark, actual Back/Forward, keyboard selection, local filters, scope changes, unavailable/retry, partial and denied reads, missing records, bounded inventory, composers and full policy definitions. No fixture business command was issued.

## Visual review and retained images

The [review receipt](review.json) binds all 95 fresh runtime images to one build and unchanged capture inputs. **84 images match the previously reviewed PNG bytes exactly** and refer to those retained files. The remaining 11 images are retained here. The browser changed from Chromium 140 to Chromium 141; nine main frames differ by 2–132 pixels each, and two phone continuation frames differ within rendered text. The complete changed images were inspected directly, including the lower policy action and delegation controls.

The reviewer also inspected the four desktop source/runtime compositions, the People phone source/runtime pair, dark phone layouts and selected contexts, tablet Approval routes, and the 600px Organization header. Heading, source-order tabs, query controls, list/context placement, complete first records, focus targets and primary actions remain intact. No new layout defect was found. The earlier documented substitutions for real directory, role, policy and delegation data remain unchanged.

This receipt does not repeat the complete historical evidence set. It retains fresh image hashes and scenario coverage; hosted CI reruns the same permanent capture and assertion scripts on the published integration tree.

## Limits

The runtime captures use isolated synthetic transport. They do not prove live backend authorization, four-actor execution, source-to-report reconciliation or production acceptance. Accessibility color-contrast incompletes remain explicitly recorded for 17 scenarios; no blanket accessibility-conformance claim is made. **Partial V2 layout integration; whole-screen acceptance remains 0/19.**
