# AI gateway transport

## Status

T3 established the separate `cmd/ai-gateway` process and provider-neutral `internal/aigateway` transport. T4 adds deterministic governed enforcement before that transport while preserving the raw-model-traffic isolation boundary.

Production workload authentication now comes from active, maker-checker-approved `ai_workloads` revisions bound to an exact active `automation_policies` revision. The gateway resolves configured Source Bindings, evaluates stable actions/reasons/obligations, and only then enters the unchanged T3 routing, budget and provider path. T4 stores no prompts, completions, tool arguments, source rows or gateway decision receipts.

## Process boundary

```text
workload bearer credential
        │
        ▼
cmd/ai-gateway
  ├─ strict Chat/Responses ingress
  ├─ governed workload + exact Automation Policy snapshot
  ├─ Source Binding fact resolution
  ├─ deterministic decision / reason / obligation kernel
  ├─ request/token/cost/concurrency budget reservation
  ├─ weighted route selection and circuit state
  ├─ OpenAI adapter
  ├─ Anthropic adapter
  ├─ canonical response / stream events
  └─ content-free logs and metrics
        │
        ▼
provider HTTPS API
```

The main Go API remains the configuration authority. `internal/aigovernance` reuses the existing Automation Policy owner, Source Binding catalog, verified identity and `governance_decisions`; it owns only AI workload registration and runtime snapshot loading. `internal/aigateway` remains a deterministic provider-neutral kernel and transport package and does not depend on product workflows or durable repositories.

## Executable routes

`internal/aigateway/routes.go` is the gateway process route/access inventory. `api/ai-gateway.openapi.json` is mechanically checked against it.

| Route | Access | Purpose |
| --- | --- | --- |
| `GET /health/live` | public health | process liveness only |
| `GET /health/ready` | public health | every configured alias has an available route |
| `GET /metrics` | independent metrics bearer | content-free counters and timings; absent when disabled |
| `GET /v1/models` | workload bearer | aliases allowed for that workload |
| `POST /v1/chat/completions` | workload bearer | OpenAI-compatible text/function completion or SSE |
| `POST /v1/responses` | workload bearer | OpenAI Responses-compatible text/function result or typed SSE lifecycle |

This inventory does not grant access to the main ClearSight API and does not replace `internal/httpapi/route_registry.go` for the main process.

## Canonical transport contract

The canonical request contains only:

- request ID and protocol;
- governed model alias;
- ordered system/developer/user/assistant/tool text messages;
- function definitions and function-call/result messages;
- provider-neutral tool choice;
- output-token, temperature, top-p and stop limits;
- streaming and usage flags.

T3 rejects image, audio, file, computer-use, hosted tool, background, stored-response, response-format and participant-name options instead of silently discarding or provider-locking them. T4 accepts a bounded string metadata map as caller assertions. Assertions are available for restrictive/routing rules and Source Binding lookup keys, but cannot directly authorize an `ALLOW` rule.

The canonical completed response contains text, refusal, function calls, finish reason and usage. The canonical stream contains only text/refusal/tool deltas, one finish event, usage and a terminal done event. Unknown provider finish reasons are surfaced as protocol failures; they are never rewritten as a successful stop. Provider safety refusals remain distinct from truncated or content-filtered output.

## Streaming and fallback truth

Provider adapters validate provider identity, model identity, event ordering, output indices, function identity, cumulative arguments, terminal state and usage before exposing canonical events.

Fallback is permitted only before the gateway commits downstream SSE bytes. The gateway reads one valid semantic event from a route before starting the client stream. Once output has started, a provider failure produces an explicit terminal stream error; the gateway never splices output from a different provider into the same response.

For Chat Completions, success terminates with `[DONE]`. For Responses, completed output emits `response.completed`; valid truncation or content filtering emits `response.incomplete`; transport or protocol failure emits an `error` event followed by `response.failed`. A provider EOF without its required terminal protocol is a failure, not completion.

## Routing and provider health

A model alias owns one or more weighted routes. The weighted route is the first candidate; every remaining route is considered in stable order for pre-output fallback.

Each route has a small in-memory circuit breaker:

- transport failures, provider overload/rate limits, timeouts and invalid provider protocol count as provider-health failures;
- request-specific provider rejection does not open the circuit and does not fall through to another provider as if it were an availability failure;
- the first request after the open interval is the only half-open probe;
- only a fully completed response/stream closes the circuit.

Circuit state is process-local operational state. T3 makes no cross-instance claim; ordinary load-balancing may send the next request to another healthy instance.

## Authentication and budgets

In production, AI workload credentials are represented only by SHA-256 digests in the governed `ai_workloads` revision. Incoming digests are compared in constant time against an immutable in-memory snapshot refreshed from PostgreSQL. Static configured workload digests remain development/test compatibility only. The optional metrics credential remains configuration-owned. Provider credentials are referenced by environment-variable name and resolved only into process memory.

Each workload has an allowed-alias set and per-minute limits for requests, conservatively reserved tokens and micro-US-dollar cost, plus a concurrent-request ceiling. Reservations use the highest configured candidate price so fallback cannot bypass the cost guard. Actual provider usage reconciles the reservation when trustworthy usage is returned. Concurrency is released against the original reservation even when a stream crosses a minute boundary.

These limits are transport controls, not T4 governed policy or approval decisions.


## Governed decisions and rollout

Policies are immutable `automation_policies` revisions with canonical SHA-256 checksums. Their deterministic definitions contain bounded Binding requirements, rules, conditions, actions, obligations and request mutations. Supported actions are:

- `ALLOW` — continue unchanged;
- `DENY` — stop before provider selection;
- `MODIFY` — apply only bounded model/output/tool mutations and revalidate;
- `ROUTE` — select an existing route inside the governed alias;
- `REQUIRE_APPROVAL` — stop until T5 supplies an exact approved execution grant;
- `SHADOW` — record the proposed result in content-free telemetry/headers while continuing unchanged.

Activation is maker-checker governed. An `ENFORCE` revision can activate only after the exact same canonical checksum was previously activated in `SHADOW`. Suspending the policy is the runtime kill switch: new snapshots no longer expose workloads bound to it.

## Source Binding resolution

Each policy references exact `{binding_id, binding_version}` values. T4 supports:

- `METADATA` from verified workload registration;
- `LIVE_LOOKUP` through the active tenant-scoped Source Binding;
- `ADAPTER_CACHE` through an explicit cache provider or a Binding whose adapter is the bank-approved local index;
- `EXTERNAL_CONTROL` through an explicit provider or REST/other Binding representing the bank control;
- `ASYNC` as explicit `PENDING`, never fabricated current truth.

Facts retain `CURRENT`, `STALE`, `PARTIAL`, `UNAVAILABLE`, `UNKNOWN` and `PENDING`. Required-fact handling produces stable reason codes and obligations before normal rules. Stale/partial values cannot match as current values. T4 does not persist source rows, fact values or a mandatory cache format.

## Governed configuration API

The main API exposes `CONFIG_READ`/`CONFIG_WRITE` routes under `/api/v1/ai-governance` for policy and workload revision creation, review, approval, activation, suspension and retirement. Tenant, maker and transition actor are bound to verified identity. Workload creation returns the raw credential once with `Cache-Control: no-store`; later reads expose no digest or token.

## Provider adapters

### OpenAI

The OpenAI adapter uses Chat Completions as the provider transport for both public ingress protocols. It supports text, refusals, function calls, cached-token accounting, non-streaming results and usage-bearing SSE. Redirects are rejected and response/event sizes are bounded.

### Anthropic

The Anthropic adapter uses Messages. System/developer text is translated to the system instruction; role runs are normalized to Anthropic's alternating message shape; assistant function calls become `tool_use`; tool results become `tool_result`. Streaming validates the documented message/content-block lifecycle, cumulative usage and partial JSON function arguments. Thinking/signature content is consumed only for protocol validation and is not exposed.

`tool_choice: none` is represented by omitting tools, because Anthropic does not define a `none` tool-choice object.

## Confidentiality and observability

Ordinary logs contain only bounded identifiers, alias, route/provider ID, stream flag, safe error code, duration, TTFT, token counts and calculated cost. Metrics use only configured aliases/providers and stable outcome codes; an unknown caller-supplied alias is labelled `unknown` to prevent unbounded metric cardinality.

The gateway does not log or persist:

- Authorization headers or provider secrets;
- messages, instructions or response text;
- refusals;
- function definitions, function arguments or tool results;
- provider response bodies or provider error messages;
- raw SSE payloads.

## Configuration and deployment

Set exactly one of:

- `CLEARSIGHT_AI_GATEWAY_CONFIG_FILE`;
- `CLEARSIGHT_AI_GATEWAY_CONFIG_JSON`.

`deploy/ai-gateway.config.example.json` uses `DATABASE` governance and contains no workload credentials. Create/approve/activate policies and workloads through the main API, retain the one-time workload token securely, populate provider secret environment variables and replace placeholder provider model IDs before use.

Production provider URLs must be fixed HTTPS origins without credentials, query strings, fragments or path prefixes. Development/test may use loopback HTTP only. Redirects are never followed.

`Dockerfile.ai-gateway` builds a non-root distroless image. The process exposes port 8090 by convention and supports graceful termination without waiting for a provider beyond the configured request deadline.

## Remaining T5 boundary

T4 deliberately does not implement:

- durable gateway decision receipts or sampled allow aggregates;
- governed response inspection/redaction receipts;
- approval/execution grants for `REQUIRE_APPROVAL`;
- binding a grant to an existing Matter Decision and exact action hash;
- provider configuration UI or durable provider-route state;
- raw-content audit logging.

Those are T5 responsibilities and must reuse existing Matter, Decision, evidence, authority and workflow models rather than adding parallel approval or audit stacks.
