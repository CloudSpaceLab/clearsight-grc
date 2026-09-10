# Matter Collaboration Workspace Design

## Decision

An issue record is an operating workspace, not a long sequence of equally weighted panels. It has a compact current handoff, a work column for the record and a separate activity column for discussion and history.

The work column uses retained tabs:

1. **Details** — finding, impact, recommendation, source facts, scope and accountable ownership.
2. **Actions** — planned work, performer, deadline, state, reassignment and update requests.
3. **Evidence** — linked form requests, submitted material and evidence gaps.
4. **Decisions** — decisions, response packages, outcome checks and closure conditions.

The activity column is available without changing tabs. It contains a comment composer, safe in-scope `@` mentions and a reverse-chronological, paged timeline. Its filter distinguishes comments from material changes without hiding either. On narrow screens, Work and Activity become peer navigation; visited tab panels retain drafts and pending state.

## Daily workflow

An owner opens the outstanding-items queue, scans the title, owner, deadline, current state and next action, and opens the selected issue. An owner can reassign a remediation action independently from changing the issue's accountable owner. A request for update is tied to one action, its current performer and a response deadline. It records an immutable request event and creates a durable delivery item. A recipient comment associated with that request marks it answered; it does not complete the action or close the issue.

Comments record the verified author, time, body, target action when supplied and zero or more permitted mentions. Mention lookup is restricted to people who already have access to the issue. A mention notifies a person but never adds visibility, responsibility or authority. The timeline exposes delivery state as Awaiting response, Answered, Overdue or Delivery failed; assignment and delivery remain separate outcomes.

## Data and authority

Comments and update requests are append-only Matter events. The canonical Matter event, event row and outbox row are committed in the existing transaction boundary. Material commands bind the verified actor, record version and legal-entity scope. Existing authority rules govern action reassignment and issue-owner reassignment. A new collaboration operation permits comments for a visible issue; a new update-request operation requires authority to manage that action.

The notification worker extends the established staff-notification delivery pattern. It resolves the active staff mailbox and access at delivery time, persists a delivery receipt keyed by the outbox event and recipient, and does not expose email addresses in the browser. Temporary delivery failures remain retryable; invalid, unavailable or rejected recipients are final recorded outcomes.

## Scalable queues and history

Outstanding vendor-linked findings use server-side keyset pagination with a default page size of 20 and a maximum of 50. The response returns only the visible page, a next cursor and the query population count when known. Filters and cursor remain in the route so opening and returning to an issue restores the same queue slice and scroll position. Timeline reads are also bounded, newest first, with an opaque cursor and a 20-item page.

## Visual and content rules

The workspace uses the existing neutral surfaces, compact metadata labels and shared tabs, status badges, focused sheets and notices. Current handoff remains the single dominant action. Queue rows reserve density for the title, source/service, accountable owner, due state, assigned action state and immediate action; long recommendations live in the issue record. No new palette, dashboard metric wall or private component contract is introduced.

## Required evidence

Rendered states cover an open issue with actions and activity in light and dark desktop layouts, plus narrow-screen Work/Activity navigation. Behavioral evidence covers paging, action reassignment, comment creation, an in-scope mention, update-request delivery receipt and answer state. The acceptance record names any unavailable mail transport as a delivery condition rather than claiming delivery.
