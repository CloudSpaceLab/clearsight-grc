# Development Standards

## Local workflow

```bash
make check
make run-api
make web-install
make run-web
```

`make check` is the minimum backend gate. Web changes also pass `npm run typecheck` and `npm run build` under `web/`.

## Change scope

Keep changes vertical and traceable to a use-case ID. Avoid framework commits that create empty generic layers. A slice includes the domain behavior, API, persistence, UI, authorization, telemetry, tests, and docs needed for one outcome.

## Go

- Run `gofmt`, `go test`, and `go vet`.
- Prefer small packages, explicit constructors, and interfaces at consumer boundaries.
- Avoid global mutable state and hidden service locators.
- Return typed domain errors and translate them once at transport boundaries.
- Bound goroutines, channels, batches, and retries.
- Use structured safe logging and correlation IDs.
- Benchmark hot paths with production-like rule counts.
- Add dependencies only when they materially improve correctness or delivery.

## HTTP

- Version public APIs under `/api/v1` and keep stable OpenAPI operation IDs.
- Enforce body limits, timeouts, cancellation, and content types.
- Use keyset pagination for populations and versions for mutable resources.
- Use `409` for stale versions, `422` for governed inability to proceed, and safe stable error codes.
- Do not reveal protected-object existence in unauthorized or invitation-failure responses.

## React and TypeScript

- Strict TypeScript is mandatory.
- Keep server-state access in typed API modules.
- Components express user purposes, not database entities.
- Avoid global state and browser replicas of authority policy.
- Lazy-load specialist workspaces; keep Today and capture small.
- Measure bundle and interaction cost before adding UI frameworks.
- Prefer accessible native controls.

## SQL

- Review plans with realistic tenant/population sizes.
- Document lock and backfill effects for each migration.
- Keep transactions short and never span remote calls.
- Avoid `SELECT *`, unbounded JSON, and deep offset pagination on high-volume paths.
- Document expected selectivity when adding indexes.
- Make backfills chunked, resumable, and throttled.

## Testing

1. domain invariants and state transitions;
2. SQL/repository integration;
3. HTTP contracts;
4. browser and accessibility journeys;
5. performance, concurrency, recovery, and security;
6. representative bank-user timing.

Fixtures must not inject the expected owner, authority, conclusion, or verification result in a way that bypasses tested behavior.

## Observability and dependencies

Instrument latency, errors, queue depth/age, retries, conflicts, fallbacks, and projection lag without recording raw evidence, answers, tokens, or protected identifiers. Runtime/dependency updates are isolated and reviewed for size, build time, licenses, vulnerabilities, behavior, and rollback.
