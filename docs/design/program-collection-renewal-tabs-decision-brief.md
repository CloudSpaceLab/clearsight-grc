# Program collection renewal and sections decision brief

**Decision date:** 2026-09-03

**Status:** Approved for implementation

## Product job

Help a Program owner configure when a collection response becomes potentially out of date, understand who last submitted it and when, and start a governed renewal before expiry without losing the prior response or its evidence.

The Program remains the continuity record. Response age changes collection attention only; it does not silently change compliance, control effectiveness, material risk or legal status.

## Primary object and action

The primary object is one Program-linked collection record: an approved form and its form Monitoring Check shown together. The dominant setup action is **Add collection to Program**. Once configured, the record shows its current next action, such as **Start collection**, **Review renewal request** or **Resolve delivery**.

The policy requires a response-validity period in months for new collection checks. Renewal starts 30 days before expiry by default, with three reminders during that window; reminders may be configured from one to five.

## Program sections

Program identity, material version and stale-state notices remain visible above the section control. The detail workspace has six URL-addressable sections:

1. Overview
2. Requirements & controls
3. Monitoring
4. Evidence & results
5. Issues & actions
6. History

Desktop and tablet use the existing tab visual language and the accessible tabs keyboard pattern. Mobile and the 200% zoom proxy use a labelled native **Program section** selector. Only one section panel is rendered at a time; no horizontal tab strip is introduced.

## Collection record states

| State | Information and next action |
| --- | --- |
| No policy | **No expiry period set** and an authorized policy-update action; no expiry is inferred for migrated checks. |
| Current | Last respondent, exact submission time, calculated expiry and the next renewal-opening date. |
| Renewal due | Existing response remains available, while one immutable successor request is awaiting confirmation. |
| Potentially expired | Response age needs attention; show the successor or recovery action without changing risk or compliance status. |
| Awaiting response | Current request deadline and reminder progress; stop reminders after submission, cancellation, pause or retirement. |
| Delivery blocked | Safe delivery limitation and **Resolve delivery**; never label an unreceipted external request as sent. |

When a successor is created, compatible scalar answers may be carried forward with predecessor submission attribution. The respondent must review and submit them again. Files and signatures remain on the prior immutable response and are never copied as new answers.

## Sensitive-data boundary

Monitoring records, schedules, events and logs store tenant/Program/check/request identifiers, an internal principal ID or opaque external contact reference, safe contact hint, delivery status and provider receipt. They do not store raw external addresses, invitation tokens, answer content, artifact content or signatures.

External automatic resend is enabled only when the current opaque route resolves through a configured delivery adapter and returns a delivery receipt. Missing identity, authority, route or adapter fails closed and remains visible as blocked work.

## Visual and responsive proof

Rendered evidence must cover 1440×900 desktop in light and dark themes, 1024×768 tablet, 390×844 and 320×800 mobile, and the repository's 200% zoom proxy. Proof includes tab focus, selector replacement, long Program/form names, no-policy, current, renewal-due, potentially-expired, awaiting-response and delivery-blocked states.

The implementation reuses current tokens, spacing, components, focus treatment and motion preferences. It adds no navigation system, illustration style, density mode, token family or notification center.
