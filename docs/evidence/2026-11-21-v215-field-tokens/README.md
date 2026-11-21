# Search and vendor field token migration

The active shell search and vendor-intake inputs now use shared field component tokens mapped through existing semantic roles to primitive values. Their light appearance is unchanged. Other field families and dark theme remain outside this slice.

Before-state reference: `docs/evidence/2026-11-21-v215-vendor-intake/` and `docs/evidence/2026-11-20-v210-framework-library/`. New isolated sample captures at 1440, 768 and 390 are in this directory. Browser checks confirmed matching computed background (`rgb(255, 255, 255)`), border (`rgb(220, 231, 245)`), 44px minimum height and 8px radius. Search retains the browser's default black text and intake retains `rgb(24, 51, 94)`. Overriding the component background token changed the actual fields and restoring it restored the original value. No horizontal overflow occurred. The 390px intake image was visually inspected.

The token chain and responsive assertions pass in `web/scripts/v210-shell-contract.nodecheck.mjs`. These sample renders do not establish dark theme, authorized backend behavior or whole-screen acceptance.
