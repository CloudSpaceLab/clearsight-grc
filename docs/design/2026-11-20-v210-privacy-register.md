# V2.10 processing activities

This partial Privacy slice uses the existing scoped processing-activity register and exact record reads. Search and pagination use the authorized source. The detail identifies the returned activity, version, owner, purpose, lawful basis and recorded processing information; missing values remain not recorded. Exact returned IDs are checked before display.

Reference: `clearsight-v2.10/privacy.html` and `processing-activity.html`. Backend scope and limitations are described in `docs/superpowers/plans/2026-11-20-v29-ropa-read-implementation.md`. Sample register and detail renders at 1440, 1280, 1024, 768 and 390 pixels are in `docs/evidence/2026-11-20-v210-privacy/`; no horizontal overflow was observed. These sample views do not prove complete source-layout parity, production authorization, dark-theme coverage, privacy case handling, impact assessments, editing or whole-screen acceptance. Production cutover remains blocked.

A subsequent authorization review found that the existing processing-activity read routes do not establish actor-level privacy visibility within a legal entity. The new workspace is held unavailable pending a privacy-specific server policy and tests; the sample renders predate that hold.
