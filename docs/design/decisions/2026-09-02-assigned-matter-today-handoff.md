# Assigned Matter Today handoff

Date: 2026-09-02

## Decision

When an open Matter has an accountable owner and exactly one owner-authorized next lifecycle state, project that handoff into the canonical Workflow and Today read models. Keep authorizer-only transitions such as cancellation, a required decision and closure out of the owner handoff.

## Why

An assigned issue in initial review previously appeared on the Matter record but not in the assigned owner’s Today queue unless it also had an Action, Decision, Response or ready outcome check. That left the dominant current action—confirming scope and ownership—outside the actor’s daily work surface.

## State proof

- Assigned initial review: one ready `OWNER` task targets assessment and deep-links to the exact Matter.
- Unassigned initial review: no owner task is invented; eligible oversight recovery remains the assignment path.
- Multiple valid owner outcomes: no transition is guessed; the Matter record retains the decision.
- Authorizer-only target: it is never projected under owner authority.
- Reassignment or missed delivery: the lifecycle maintainer re-resolves the current owner and converges the task.

The lifecycle Today evidence fixture includes the assigned initial-review state for responsive and theme rendering.
