# Vendor relationship detail: v2.15 alignment

Reviewer: Codex vendor-detail agent, 22 November 2026. This is a partial layout review, not production acceptance.

## Decision

Lead with the service and supplier, followed by four recorded relationship facts and the existing due diligence workflow. Relationship facts are supplied by the scoped relationship API; an absent value is labelled `Not recorded`. Keep the scoped Documents destination and existing review, activation and permission checks. Do not substitute prototype counts, sample data or actions.

The v2.15 source is `C:/dev/clearsight-grc/clearsight-v2.15/vendor-engagement.html` (SHA-256 `06CB337EC976856F4AC301DC14C925780BF657EA6799C6C43A54CC8196ABDD85`), with `assets/vendor-onboarding.js` (`1A32FE300EC7C5DF317254D7A78BDC089C661FC9D2FDD529C3AB2322DFBEF30B`) and `assets/workflow-completion.js` (`9CDD45C81B5B1B69C9274CD5CC028B841542168C4DFD4D0A13436384F9849670`). `reference-1440.png` is the local synthetic reference service ENG-03. The runtime images use isolated sample data, not prototype values. Baselines are `../2026-11-22-vendor-start-review/before-{1440,768,390}.png`.

## Responsive and state review

The fact cards form four columns at desktop and two at smaller widths. Long service names wrap. The due diligence and Documents destinations remain labelled. `vendor-detail-{1440,1280,1024,768,390}.png` captures the submitted assessment and pending document state; the screenshot script checks key facts, an assessment panel, and horizontal overflow. Existing vendor tests cover loading, missing assessment, denied access, review decisions and retry states. The review was based on these local sample renders and automated tests; real-bank data, touch devices and a named human acceptance review were not checked.

The source also shows assurance-program summaries, related services and a dedicated onboarding/lifecycle dossier. The current workspace does not yet reproduce that complete anatomy. These cannot be represented by invented counts or unsupported actions and remain follow-up work. The screenshot comparison is not a claim of full-page parity or production readiness.
