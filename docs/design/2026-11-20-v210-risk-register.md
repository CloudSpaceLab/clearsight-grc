# V2.10 risk register

This partial risk-register slice uses the existing authorized paginated risk endpoint for search and exact-record reads. It displays the stored appetite position and marks an assessment for an earlier risk version. The list does not infer a risk score from missing assessments. Detail retains the current risk version and stored assessment history; unsupported commands are not exposed.

Reference: `clearsight-v2.10/risks.html` and `risk-workspace.html`; the existing backend contract is documented in `docs/superpowers/plans/2026-11-20-v29-risk-register-implementation.md`. Sample register and detail renders at 1440, 1280, 1024, 768 and 390 pixels are in `docs/evidence/2026-11-20-v210-risk-register/`; no horizontal overflow was observed. These sample views do not prove complete source-layout parity, authorized production data, dark theme, risk editing, RCSA, KRI or loss workflows. Production cutover remains blocked.
