# Enterprise Setup compatibility evidence — 31 October 2026

## Result and scope

**PASS: 10 Configuration scenarios and 92 People/access scenarios.** The scoped People reviewer passed all 92 required cases; Configuration passed its navigation, dialog ownership and responsive checks. The reviewer verified **107 PNGs in the original local capture bundle** against their recorded byte lengths and SHA-256 hashes. This repository bundle retains **5 affected PNGs**, all original JSON receipts, and a [107-image hash/retention manifest](image-manifest.json). **The other 102 PNGs are intentionally not included here.** The exact result is in [compatibility-review.json](compatibility-review.json).

This is compatibility evidence for the Enterprise Setup integration. The [parent Enterprise Setup review](../README.md) owns the separate original/runtime layout comparison, required viewport matrix and whole-page visual findings. These synthetic fixtures do not establish live backend authorization, four-actor execution, source-to-report reconciliation or production acceptance.

## Exact revision and build

| Identity | Value |
| --- | --- |
| Reviewed local runtime revision | `2264dfd18bc070fe6d5260422272b1533966b8dc` |
| Reviewed runtime tree | `a56a650c8e087d27c2f29ff9595a1936c6dd1361` |
| Evidence build SHA-256 | `7d92d25660e43fdf132676b7f3099ef82430bbd763422c00a12045b0a5e5acc3` |
| Named compatibility reviewer | `/root/ci_review` |
| Browser | Chromium `141.0.7390.37` |
| Node | `v24.22.1` |
| Locale / timezone | `en-NG` / `Africa/Lagos` |
| Motion preference | `reduce` |

The capture used the unchanged evidence build served at `http://127.0.0.1:4179`. Both runtime revision and build digest are recorded in the full [People manifest](people-manifest.json); the [scoped review](compatibility-review.json) checks the runtime tree, recomputed build digest and instruction hashes. The five instruction files below were also checked against the named local runtime revision. These repository links identify the files; the recorded hashes identify their reviewed contents.

| Capture or contract source | SHA-256 |
| --- | --- |
| [scripts/capture-configuration-evidence.mjs](../../../../web/scripts/capture-configuration-evidence.mjs) | `168a8006ba5a2cff74510166a903c596896b1a07d3331922feed81f6af1824e7` |
| [scripts/capture-people-access-evidence.mjs](../../../../web/scripts/capture-people-access-evidence.mjs) | `fec36b3994bc3aec2de65d3ea43a44fd1295d49405809e97d4a58aee1b3c4407` |
| [scripts/people-access-evidence.mjs](../../../../web/scripts/people-access-evidence.mjs) | `445a645f3bfd92f43672e19f37fc410ae4ce7abfaee92ec2ccb2fd8532bbd677` |
| [scripts/people-access-evidence-scenarios.mjs](../../../../web/scripts/people-access-evidence-scenarios.mjs) | `f5f4532937f73a1fede0adb6b2797d0ac236f0d09666320a9c083331ff980668` |
| [scripts/people-access-evidence-contract.mjs](../../../../web/scripts/people-access-evidence-contract.mjs) | `84d95655477864eec053c9cfaa830567c67beadeed8d0281b5a1708bbc2014bf` |

## Material checks retained

- Configuration: the real Enterprise Setup heading and six source-order tabs; selected indicator and 44 px targets; real Reporting, Provisioning and Escalation destinations; no legacy People header/rail; no document overflow; existing Configuration, People and governance destinations remain reachable.
- Positions: opening **Positions & roles** before its retained editor; cold **390 px dark** and **600 px light** header/filter containment and ordering; two consecutive identical rendered frames; Escape dismissal. The role/position owner remains an explicit sheet.
- Pending changes: explicit **Area changes** entry, independent review controls, dismissal and return to the initial page viewport without issuing approval commands.
- People/access: four existing views, local search and retained selection, actual desktop/phone Back/Forward, Organization editor and Provisioning portal closure, permission/empty/partial/recovery/missing states, composer focus and phone continuation checks. Original People layout pairing remains documented separately; this review checks continued behavior.

All 92 People case results are in [people-manifest.json](people-manifest.json). All 10 Configuration case results, measured navigation and both position-header receipts are in [configuration-review.json](configuration-review.json).

## Direct visual inspection

Reviewer **`/root/ci_review`** opened and inspected the final 390 px dark and 600 px light position-header PNGs below. Both show painted, readable heading, filter and action text. The headings sit above the filters without overlap; no blank-text regression was observed. This named inspection covers those two retained header views. The three Organization overview images are retained as affected compatibility context; the parent Enterprise review owns the whole-page visual assessment.

## Retained screenshots

| Repository PNG | Bytes | SHA-256 |
| --- | ---: | --- |
| [configuration-organization-desktop-light-1440x900.png](configuration/configuration-organization-desktop-light-1440x900.png) | 156,403 | `dce9114aa6ca7c88d0790ae68936ca4fcb6883a6cfe1c128bafde9c3f10f00b1` |
| [configuration-organization-mobile-dark-390x844.png](configuration/configuration-organization-mobile-dark-390x844.png) | 62,145 | `61b9f16e95126f6e036ec835bf5b896b6d4711b325a1f3fc2a0c5eef1ec9344a` |
| [configuration-organization-mobile-dark-390x844-position-header.png](configuration/configuration-organization-mobile-dark-390x844-position-header.png) | 40,919 | `5a37ef8495d2371789e2c93cb52057229752fcbb67c6caaf1ed235f55bdc151f` |
| [configuration-organization-compact-light-600x900.png](configuration/configuration-organization-compact-light-600x900.png) | 75,146 | `4230c5c5439fbed032eaeab18b4922a3945d9849a83c801f48ad21921a7afdd6` |
| [configuration-organization-compact-light-600x900-position-header.png](configuration/configuration-organization-compact-light-600x900-position-header.png) | 42,302 | `fa44136ae4651498d8e05e6be799966cb06c4bc44493b39db7d29a25b1c988dc` |

## Intentional omissions and binding rules

**102 PNGs are omitted:** the 7 unchanged Configuration destinations and all 95 People screenshots (92 primary captures plus 3 phone continuation captures). Their complete metadata, observations and per-image hashes remain in the full receipts and [image-manifest.json](image-manifest.json). The scoped review verified their bytes in the original local capture bundle before retention; this directory does **not** contain those PNGs and does not present links to them.

Every image-manifest record contains its original capture path, byte length, SHA-256, receipt filename and JSON Pointer to the exact original receipt record. A record with `retained: true` has a `retainedFile` that resolves within this directory. A record with `retained: false` has `retainedFile: null`; its `originalCapturePath` is capture metadata, not a repository attachment. The original full receipts are copied byte-for-byte, so image names appearing in those raw receipts retain their original meaning and do not imply repository retention.

| Unmodified original receipt | Bytes | SHA-256 |
| --- | ---: | --- |
| [configuration-review.json](configuration-review.json) | 16,217 | `02047c49d6c5f3f6631e7c3551f47a230530efa17e887a827bae535dd85eaf46` |
| [people-manifest.json](people-manifest.json) | 2,188,748 | `a8c146e4f3c07adad4be9cdaa679c808bf6fb0e55cdd2b331a690c4d39452d08` |
| [compatibility-review.json](compatibility-review.json) | 1,800 | `db00d866b78b5574c819723fafec7b8d6cff0003fcb6be10438d531bc65335ed` |

The JSON retention manifest additionally binds these receipt files and all five retained PNGs by their exact hashes. No production or whole-screen acceptance state is advanced by this compatibility bundle.
