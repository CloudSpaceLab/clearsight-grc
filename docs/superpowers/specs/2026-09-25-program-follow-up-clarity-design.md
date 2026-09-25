# Program follow-up clarity

## Goal

Make the reason for a Program follow-up immediately understandable and route the user to the smallest valid request for the affected record.

## Operating model

The Program overview groups overdue or incomplete collection work into three mutually exclusive populations:

1. **Vendor response needs updating** — an answer supplied by a third party is no longer current.
2. **Vendor document needs replacing** — a supporting document is missing, expired, or no longer valid.
3. **Internal update needed** — a bank-owned answer, decision, or review is absent or overdue.

The summary names both the number of affected records and vendors. It does not label an answer as a document or use the generic word “evidence” when a more specific business object is known. Each population opens a filtered detail list.

For a completed vendor form, a reviewer requests only the affected fields. The request uses the existing versioned clarification endpoint and creates a successor capture containing only those questions. In the interface this is called **Request updated fields**; “clarification” remains an internal/API term.

For an open vendor request, the reviewer can send a manual reminder. The reminder uses the existing distribution communication route, records delivery state, and never exposes recipient details or secure tokens.

Internal comments keep the current append-only activity record. Mentioned employees receive a staff notification email through the same durable outbox pattern already used for reassignment and status-update requests. The recipient, action link, and notification kind remain protected operational data.

## Experience

The Program position keeps one dominant action. Beneath the status reason it shows a compact follow-up summary, for example: “3 vendor records across 2 vendors need updated responses; 1 vendor document must be replaced; 2 internal reviews are overdue.” It uses three labelled counts rather than KPI cards.

Response review makes provenance visible in the field row: **Vendor answer**, **Vendor document**, or **Internal review**. A stale state includes its expiry/assessment date and a single contextual action. The targeted request drawer preselects the affected fields and explains the requested outcome and deadline.

Comments show recipients before posting. A successful post confirms that mentioned colleagues were notified; notification failure remains visible in the activity timeline without losing the stored comment.

## Boundaries

- Program status calculation remains authoritative and unchanged.
- Submission, attachment, review decision, and verified outcome remain distinct.
- A manual reminder is available only for a currently open external distribution. A completed response receives a targeted successor request instead.
- No client-supplied actor, tenant, recipient address, or authority route is trusted.
- Existing history remains immutable and reconstructable.

## Verification

- Component tests prove the summary uses distinct answer, document, and internal labels and counts unique vendors.
- Vendor review tests prove selected fields produce a targeted update request.
- Workflow tests prove a mention creates a durable notification addressed to the mentioned internal principal and does not duplicate delivery.
- Render the Program and vendor response states at desktop and mobile widths before release.
