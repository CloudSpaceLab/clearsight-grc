# First-opening correction: named scoped review

**PASS for local runtime regression verification and the retained Enterprise capture.** Final-head hosted validation was still pending at this review; [PR #599](https://github.com/CloudSpaceLab/clearsight-grc/pull/599) records subsequent status.

Reviewer: **GPT-6 Astra Pro `/root/ci_review`**, 2026-10-31. Independent visual comparison: **GPT-6 Astra Pro `/root/people_integration`**, recorded in [visual-review.json](validation/visual-review.json). Root also directly inspected the sole changed frame.

## Binding and replay

This review binds local runtime `aa666af542fdcba2f2883a58821434f3fbc18740`, equivalent remote object `3a32ab85e0cb0615903f76fcac1c08362db306c8`, shared tree `9e670cd7477ddd25b0e5c9e1d01961ae3ecfbacc`, Chromium `140.0.7339.16`, and build `40e12a493b9e595206607cba4729db53c7d685dc39cd5169c330fc268542649c`. [Provenance](provenance.json) records the complete hashes and retained files. The remote object existed but had not advanced the PR branch at this point.

I verified all 42 PNGs against their byte counts and SHA-256 values, verified the three input snapshots against the receipt and exact commit blobs, and replayed the frozen reviewer against the retained capture: **10 layouts, 11 behavior scenarios, 42 images, no failures**. The approved HTML hash remains `cc7542e2a3047338b1d2fbbbe33756adbc9ee8c2084fb5920834da8aeedc04c7`. The build digest is unchanged across this capture, and differs from the preceding build because this follow-up changes runtime code.

## Comparison and direct inspection

[All-image comparison](validation/image-comparison.json) verifies that **41/42 images are byte-identical** to the [preceding corrected capture](../hosted-followup/corrected-capture/enterprise-setup/receipt.json), which used the same Chromium version. Those identical images retain their [prior named review](../hosted-followup/corrected-named-review.md) binding.

I directly opened the changed [1440px dark overview](capture/enterprise-setup/layout-dark-1440.png): the selected Technology row and context, six source-order tabs, hierarchy density, text and controls remain readable, with no new overlap or missing content. The independent pair comparison found only **37 changed pixels**, each differing by at most **one RGB level**, within `[1059,18,1369,56]` in existing top-shell controls. No Enterprise Setup body pixels changed. This is consistent with the direct inspection and does not imply a behavioral proof from pixels.

The replay preserves the five-width light/dark anatomy and geometry checks, first entity and first actual area visibility, source-matched context replacement, no horizontal overflow, manual keyboard tab activation and minimum 44px controls. The 500-area limitation remains within the 390px viewport at y=544.109375–562.109375. Before the imported editor's single Escape, the receipt still observes `expanded: "false"`, `focused: true`, `listboxes: 0` and `popovers: 0`. Recorded browser errors and material commands remain zero.

## Runtime proof and limits

The [A/B and focused test record](runtime-regression-verification.json), [old-effects negative control](old-effects-regression-red.log) and [full 277-file / 2,301-test suite](full-web-tests.log) support the first-opening correction separately from screenshot equivalence. The local A/B demonstrates a reproducible dropped-open ordering defect. It does not reconstruct the exact timing of the earlier hosted CI failure.

No additional visible regression was found within this review scope. Existing accessibility incompletes and targeted-contrast limitations remain; no full WCAG claim is made. Synthetic fixtures do not prove backend authorization, real commands, independent actors or production acceptance. Whole-screen acceptance remains **0/19**, and the earlier failed CI/UI checkpoints retain their original conclusions.
