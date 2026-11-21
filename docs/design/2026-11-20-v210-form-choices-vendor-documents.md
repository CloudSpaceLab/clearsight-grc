# V2.10 form choices and vendor documents

The form draft editor supports single- and multiple-choice questions with 2–50 distinct options. An uncertain save retains the entered options for comparison before retrying. Vendor document linking uses the current assessment and request versions and leaves document review pending.

Evidence captured with the isolated sample transport on 20 November 2026: `form-{1440,1280,1024,768,390}.png` and `vendor-{1440,1280,1024,768,390}.png` in `docs/evidence/2026-11-20-v210-form-choices/`. The captures check the choice input and vendor document empty state; no horizontal overflow was detected at those widths. They do not establish backend authorization, a successful document link, dark-theme parity, or whole-screen acceptance. The vendor link action remained disabled when no eligible documents were returned.

Internal response workspace routes remain disabled pending concurrency coverage for all scope and directory eligibility changes. API-question tests do not configure an approved integration or execute an assessment.

On 21 November, vendor document selection and rationale were locked during an in-flight link command, and pagination was made single-flight with stale-result rejection after a field is reopened. Synthetic pending-command renders at 1440, 768 and 390 pixels are retained in `docs/evidence/2026-11-21-v215-vendor-collection-lock/`. The fixture modifies the sample document's channel and expiry to make it eligible and holds the link request open; it does not prove a successful or authorized production link. Dark-mode, real-backend and full-screen acceptance remain open.
