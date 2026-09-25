# Vendor renewal and demo curation

**Decision:** Show one compact Program-level summary of current submitted vendor forms needing follow-up, with distinct affected vendors, expired documents, response fields and bank review findings. Keep the affected-vendor list collapsed until requested. If the scoped read fails or exceeds its bound, show a retry or limitation rather than a zero.

**Vendor workflow:** A completed assessment remains historical. A successor reassessment may select only expired fields that are present in the active form; a reviewer can adjust the selected fields. Starting the review does not email the vendor. The existing prepare/send sequence sends the selected request. An open collection request can receive a manual follow-up email containing a new secure link; delivery failure retains the controlled copy recovery.

**Demo curation:** Six explicitly identified scoring acceptance fixtures are synthetic and not part of the supplied bank samples. Archive visibility is scoped by distribution ID and demo tenant/legal entity. Form, response and assigned-request lists and execution reads omit these fixtures; exact historical response reads remain available. A restored archive receipt can be attributed once. The operator script has exact source and population guards, asserts six active archive receipts and makes no lifecycle edits.

**States to check:** no links; linked forms with no follow-up; mixed expired vendor and internal items; unavailable/over-bound read; completed assessment with and without matching expired fields; request email delivered and not delivered; mobile stacked vendor list.

No new token, motion or illustration pattern is introduced. Existing surfaces, disclosure and shared controls apply.
