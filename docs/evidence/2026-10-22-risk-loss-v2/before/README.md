# Immutable pre-tranche Risk & Losses evidence

The 40 matrix PNGs and `manifest.json` were captured from a separately built, detached worktree at commit `ba284eb24b01141666c93783c15221263a13d7b8`, tree `7d1c2880c7082861751999290430408caea94377`. The active implementation checkout could change without affecting this baseline build. `baseline-receipt.json` records the manifest checksum and all 171 built-asset checksums.

The matrix covers Risks, RCSA, Key indicators and Loss events at 1440×900, 1280×800, 1024×768, 768×900 and 390×844, in light and dark. The additional `251-risk-loss-rcsa-light-768x900-failure.png` is the original full-page diagnostic that exposed baseline overflow; it is not a 41st matrix state. No PNG has been recompressed.

Baseline findings are retained, not treated as acceptance. RCSA was 805px wide in a 768px viewport. Indicator pages were 1299px at 1280, 1152px at 1024 and 1119px at 768, in both themes. At 390px, the first Risk identity began at 1287px and the first Loss identity at 1831px, below the first viewport.

These are isolated synthetic UI fixtures, not backend authorization or operational proof. Existing RCSA, native-indicator and loss-S5 payloads are reused. The Risk collection is supplied by the test-only observation fixture so before and after use the same records. The approved operational-risk HTML remains immutable and is not imported into runtime.
