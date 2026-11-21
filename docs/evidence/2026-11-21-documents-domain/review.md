# Documents and vendor document review — 21 November 2026

Reviewer: Codex agent `/root/domain_workflows`. The static evidence build and sample fixtures were inspected, not a connected bank environment. Captures were made before a concurrent, unrelated API-question wizard evidence rebuild; they are not a release or source-parity approval.

`web/scripts/v210-domain-evidence.mjs` passed its five viewport and horizontal-overflow checks (1440, 1280, 1024, 768 and 390 pixels). The Documents grid, selected details and sample image preview element were rendered. At 1440 pixels the details panel is beside the list; at 768 and 390 pixels it follows the list. I inspected the 1440, 768 and 390 captures. Controls remain visible without horizontal overflow. The sample image itself appears blank, so these captures do not verify legible image content. They do not exercise the newly added denied-access recovery path, which is covered by the component regression test.

Preview investigation: `web/src/v210/evidence.ts` serves a hard-coded 1×1 PNG for this sample document. A separate Chrome browser check opened the document and decoded the preview image with `complete=true`, `naturalWidth=1` and `naturalHeight=1`. The blank appearance is therefore expected for this fixture; the check does not establish that a realistic document renders correctly.

`web/scripts/v210-form-choices-vendor-evidence.mjs` passed the same five widths and verified that the document link action is disabled when no eligible document is available. I inspected the 1440, 768 and 390 vendor captures: the requested-document section and disabled action fit without horizontal overflow. The selection section appears below the first viewport, with substantial vertical spacing; this fixture does not exercise a successful link, a mismatched response revision, real authority, or a live backend.

These light-theme, sample-data captures do not establish accessibility, dark-theme, production authorization, live preview fidelity, or full v2.15 screen acceptance. Screenshots and machine-readable results are in this directory and `../2026-11-21-vendor-document-domain/`.
