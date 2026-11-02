# Frozen People & access baseline inputs

The 30 images in `../before/` show the existing customer layout at main `ea84283f913755be6dc1b8e366282469745b3e9a` (tree `28f0b24ef1da098b46bfd345c466cecd79051675`) before this tranche. The Organization, Directory and Authority destinations were captured at 1440×900, 1280×800, 1024×768, 768×900 and 390×844 in light and dark themes.

These three files are verbatim historical capture inputs, retained for provenance. They are not customer entry points. The archived web source was built with the isolated `people-v2-current` evidence transport; its runtime components were unchanged. The capture script contains the temporary paths and local server port used for that run. To reproduce in a new checkout, archive the pinned revision, place the two source snapshots at `web/src/`, install the locked dependencies and pinned Playwright 1.55.0/Chromium runtime, then adapt only the scratch paths/preview URL in a copy of the capture script. The run used `en-NG`, `Africa/Lagos`, comfortable density and reduced motion.

| Input | SHA-256 |
| --- | --- |
| `evidenceMain.tsx` | `0d79695148be0f443b87f2c72e7587c0651506abd6631b3161225cb69e4ccbd7` |
| `peopleAccessEvidence.ts` | `54bf1da6258144b4e8b524d50fd53499147571ea19818b789062bac085c3cb60` |
| `capture-baseline.mjs` | `9d3d341d78e4d6ee0580c4a40ae15795662d0b3a8a5f1cbabffd97c0c937ccfe` |

The [baseline manifest](../before/manifest.json) pins all image hashes, read observations, geometry, build assets and input hashes. Build digest: `8a6b34ca9f23620233e6ba0b64f1337135d8c5b072c2950279f855259c2f35e0`. Input digest: `98027d3f997db615f369db5a75c781fcc3267279ffb1df9807831a50ff483812`.

The baseline transport returns 12 synthetic people; the old Directory renders eight, while the first person is below the initial viewport at every checked width. At 390px the document is 553px wide. These are fixture-bound layout observations, not live directory or authorization results.

The final evidence fixture extends the same core synthetic populations with actual-contract policy definitions/checksums and precise same-day pending-delegation times so the newly exposed details can be reviewed. Those added fields were not rendered by the baseline; the final fixture is deliberately not described as byte-identical to this snapshot.
