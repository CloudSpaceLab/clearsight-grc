# T3 AI gateway transport acceptance

T3 is complete only when the exact publication head satisfies every gate below.

## Executable scope

- `cmd/ai-gateway` starts independently from the main API and worker.
- `internal/aigateway` owns one provider-neutral text/function request, response and semantic stream contract.
- `api/ai-gateway.openapi.json` exactly matches the isolated process route/access inventory.
- Chat Completions and Responses support strict non-streaming and truthful SSE responses.
- OpenAI and Anthropic adapters both pass completed-response, streaming, usage, refusal and function-call tests.

## Identity, bounds and confidentiality

- Missing/invalid workload bearer credentials fail before request parsing or provider access.
- Workload and metrics credentials are compared as SHA-256 digests in constant time.
- Provider secrets are resolved from environment references and never serialized into configuration output, logs, metrics or errors.
- Request bytes, message/tool counts, text, function schema/arguments, provider bodies, SSE events, timeouts and headers are bounded.
- Provider redirects are rejected.
- Logs and metrics contain no prompt, completion, refusal, tool definition, argument, result, provider body or secret.
- Unknown caller aliases collapse to the `unknown` metric label.

## Routing, failure and accounting

- Weighted first-route selection is deterministic under test and retains every configured fallback candidate.
- Retriable provider-health failures may fall back only before downstream output starts.
- Provider request rejection does not masquerade as availability failure or trigger cross-provider replay.
- A stream failure after the first committed event produces an explicit terminal error and never switches provider.
- Circuit opening, one-probe half-open state and successful recovery are tested.
- Request/token/cost/concurrency limits are enforced per workload.
- The highest candidate price is reserved before routing; trustworthy usage reconciles actual cost.
- Integer overflow, negative usage and inconsistent provider totals fail closed.
- Concurrency is released correctly when a stream crosses a minute boundary.

## Protocol truth

- OpenAI validates response object, provider response/model identity, one output choice/index, finish state, function identity/JSON and usage.
- Usage-only terminal chunks are accepted; duplicate usage, output after finish, identity drift and missing terminal usage fail.
- Anthropic validates message start, one-at-a-time ordered content blocks, delta/block type compatibility, cumulative function JSON, stop reason, usage and message stop.
- Unknown Anthropic top-level events are forward-compatible, while unknown mutations of supported blocks fail.
- Unknown finish reasons fail as invalid provider protocol rather than becoming successful `stop`; safety refusals remain distinct from truncation and content filtering.
- Provider EOF before a valid terminal event is a failed stream.

## Performance evidence

`BenchmarkWarmRouting` measures route selection/candidate construction with no provider/model latency. `BenchmarkAuthenticatedHTTP` measures bearer authentication, strict JSON translation, budget reservation, in-memory routing, fake-provider completion and response encoding with no network/model latency.

The acceptance script rejects gross regressions above:

- warm routing: 100 microseconds/op;
- authenticated in-process HTTP: 5 milliseconds/op.

These are transport regression ceilings, not production SLO claims. Pilot load testing must set real percentile and concurrency objectives using deployed infrastructure.

## Anti-bloat boundary

T3 introduces no migration or durable gateway table, no second user/RBAC/workflow/event-bus/scheduler stack, no Source Binding consumer and no policy/approval engine. Static T3 workload/route configuration is explicitly replaced—not duplicated—when T4 introduces governed lifecycle state.

## Required commands

```bash
bash scripts/verify-t3-gateway.sh
go test -race ./...
go test -tags postgres ./...
go test -p 1 -tags "postgres postgresintegration" ./internal/...
go vet ./...
cd web && npm run typecheck && npm test && npm run build
```

The exact PR head must also pass deterministic Chromium UI/UX review, even though T3 adds no customer-facing browser surface, so shared application behavior remains protected.
