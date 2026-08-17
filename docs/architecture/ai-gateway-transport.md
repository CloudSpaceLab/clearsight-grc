# AI gateway transport

## Status

T3 adds a separate, stateless `cmd/ai-gateway` process and the provider-neutral `internal/aigateway` transport package. It is an isolated model-traffic edge, not another ClearSight API module and not yet the governed enforcement layer.

The gateway accepts authenticated workload traffic, translates a bounded common text/function contract to pilot providers, returns OpenAI-compatible Chat Completions or Responses payloads, and emits content-free operational telemetry. It has no durable tables and stores no prompts, completions, tool arguments or policy decisions.

## Process boundary

```text
workload bearer credential
        │
        ▼
cmd/ai-gateway
  ├─ strict Chat/Responses ingress
  ├─ static T3 workload and alias bootstrap
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

The main Go API remains the authority for Programs, issues and changes, evidence, workflow and configuration. T4 will replace the static workload bootstrap with governed workload/policy state and will resolve reusable Source Bindings. T3 deliberately does not import `internal/sourceaccess`, create approval state or make governance decisions.

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

T3 rejects image, audio, file, computer-use, hosted tool, background, stored-response, response-format, participant-name, arbitrary user tag and retained metadata options instead of silently discarding or provider-locking them.

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

T3 workload credentials and the optional metrics credential are represented only by SHA-256 digests in configuration. Incoming digests are compared in constant time. Provider credentials are referenced by environment-variable name and resolved only into process memory.

Each workload has an allowed-alias set and per-minute limits for requests, conservatively reserved tokens and micro-US-dollar cost, plus a concurrent-request ceiling. Reservations use the highest configured candidate price so fallback cannot bypass the cost guard. Actual provider usage reconciles the reservation when trustworthy usage is returned. Concurrency is released against the original reservation even when a stream crosses a minute boundary.

These limits are transport controls, not T4 governed policy or approval decisions.

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

`deploy/ai-gateway.config.example.json` is fail-closed: its all-zero credential digests cannot authenticate a known example key. Generate real digests outside the JSON, populate provider secret environment variables and replace placeholder provider model IDs before use.

Production provider URLs must be fixed HTTPS origins without credentials, query strings, fragments or path prefixes. Development/test may use loopback HTTP only. Redirects are never followed.

`Dockerfile.ai-gateway` builds a non-root distroless image. The process exposes port 8090 by convention and supports graceful termination without waiting for a provider beyond the configured request deadline.

## Deliberate T3 exclusions

T3 does not implement:

- governed AI workload registration or maker-checker activation;
- Automation Policy extension or deterministic governance decisions;
- source-aware classification or Source Binding resolution;
- prompt mutation/redaction obligations;
- durable decision receipts;
- approval/execution grants;
- provider configuration UI or durable provider route state;
- raw-content audit logging.

Those are T4/T5 responsibilities and must reuse existing governance, evidence, source, workflow and authority models rather than adding parallel stacks.
