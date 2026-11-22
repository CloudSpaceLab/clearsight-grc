# Vendor relationship setup progress

Reviewer: Codex vendor-detail agent, 22 November 2026. Partial v2.15 alignment; not full-page acceptance.

The existing three-step, authority-checked relationship creation flow now exposes its current step as a labelled list. Previous steps appear complete and the current step is announced with `aria-current="step"`. The step indicator does not create navigation or submit data. Focus, back navigation and recovery remain in the existing flow.

Reference: `C:/dev/clearsight-grc/clearsight-v2.15/assets/vendor-onboarding.js`, SHA-256 `1A32FE300EC7C5DF317254D7A78BDC089C661FC9D2FDD529C3AB2322DFBEF30B`; the source contains a four-step onboarding dossier, which is broader than the supported relationship-creation command. Baseline renders are in `../2026-11-21-v210-vendor-intake/`; after renders and browser results are in this folder. The captures use synthetic data. Visual review inspected the service step at 390px after correcting the number markers; automated evidence covers 1440, 1280, 1024, 768 and 390px, matching vendors, review, edit/back retention and uncertain outcomes. Vendor and copy tests passed (31 passed, 2 skipped); TypeScript passed.

Outstanding: the reference's fuller supplier, service boundary, named respondent and independent intake review dossier is not implemented by this relationship creation flow. Its design must be connected to supported scoped backend contracts before claiming parity; the reference's prototype records and decisions are not production data.
