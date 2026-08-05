# ADR 0010 — Verified command actors and separately versioned Program status

**Status:** Accepted  
**Date:** 2026-08-05

## Context

Program and issue commands previously accepted actor identifiers from request bodies. Calculated Program status was also written as a Program aggregate event, advancing the same version used for material commands. Some compound operations crossed multiple repository calls.

This created four risks:

1. actor fields could be forged by a caller;
2. a command could run without rechecking current authority;
3. calculated status could be confused with a material Program change;
4. a post-commit status-refresh failure could make a successful command appear to have failed.

## Decision

1. Production requests use a short-lived gateway-signed identity envelope. Actor fields in command bodies are overwritten from verified request context.
2. Material Program and issue commands resolve current authority before execution. Production uses enforced mode; development may use audit mode.
3. Compound trigger/issue/link operations and issue/initial-link operations are committed in one transaction.
4. Material commands create a deduplicated Program-status update job in the same transaction.
5. Program status snapshots use an independent projection version and record the Program command version assessed.
6. A bounded worker calculates status, writes the snapshot, emits a `PROGRAM_STATE` event and completes the job.
7. Operators receive health, reconciliation and manual-rebuild controls using plain working language.

## Consequences

- successful commands no longer depend on synchronous status calculation;
- stale status is explicit and measurable;
- command history and calculated history remain independently reconstructable;
- production requires gateway secret management and current routing data;
- callers must not depend on a newly calculated status in the immediate command response;
- status workers and lag become monitored operational dependencies.

## Rejected alternatives

### Trust body actor IDs

Rejected because attribution would be caller-controlled.

### Calculate status synchronously after commit

Rejected because a calculation failure can produce a false command failure and adds latency.

### Advance Program version for every calculated snapshot

Rejected because calculated state is not a material user command and creates unnecessary optimistic-lock conflicts.

### Use an unbounded background scan

Rejected because it hides lag, increases database load and is difficult to recover safely.
