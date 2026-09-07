# AI gateway control-plane acceptance

This contract closes the system-administrator provider/routing portion of issue #172. It is additive to the T3 transport and T4/T5 AI-governance acceptance contracts; it does not weaken either boundary.

## Authority and tenancy

- Provider/model transport state is versioned in `ai_gateway_config_revisions` by tenant and environment.
- Production has at most one `ACTIVE` revision per tenant/environment.
- Activation atomically supersedes the prior active revision.
- Makers cannot approve or activate their own submitted revision.
- Applications select logical model aliases; callers cannot nominate an arbitrary upstream provider.
- `CLEARSIGHT_AI_GATEWAY_TRANSPORT_MODE=DATABASE` is explicit opt-in. `STATIC` remains the compatibility default and the two modes never act as competing request-routing authorities.

## Secret boundary

- Durable provider definitions contain only `secret_ref`, never provider credential material.
- The currently supported reference form is `env:<NAME>` and is validated before persistence so a credential pasted into the reference field is rejected.
- Secret values are resolved only inside the gateway process when a candidate snapshot is applied.
- Browser/API responses, decision receipts, logs and analytics do not contain resolved provider credentials.
- Provider origins are fixed; production routes require HTTPS.

## Runtime apply contract

For each tenant/environment the gateway applies a candidate as:

1. fetch exact active revision;
2. validate tenant/environment/revision/checksum;
3. validate provider/model definition;
4. resolve opaque secret references inside the gateway process;
5. construct provider adapters and the complete router;
6. atomically replace the in-memory snapshot only after all prior steps succeed.

A failed refresh or candidate application cannot evict the prior known-good router. Suspended providers are ineligible and a logical alias with no enabled route fails candidate application.

The gateway operations endpoint `GET /health/config` exposes only desired/applied revision metadata, emergency-control apply state and stable error codes. It is protected by the independent operations/metrics bearer credential.

The ClearSight API may be configured with `CLEARSIGHT_AI_GATEWAY_OPERATIONS_URL` and `CLEARSIGHT_AI_GATEWAY_OPERATIONS_TOKEN`. This is a server-to-server bridge only: Configure receives the projected runtime status together with the governed revision list; the operations credential is never exposed to the browser. Missing bridge configuration is represented as **not connected**, while a configured but unreachable gateway is represented as **unavailable/degraded**.

## Stable application proxy

`CLEARSIGHT_AI_GATEWAY_PUBLIC_BASE_URL` is optional deployment metadata for the application-facing governed proxy. It is deliberately separate from the internal operations URL and credential.

- production public proxy URLs require HTTPS;
- loopback HTTP is allowed only in local development;
- embedded URL credentials, query strings and fragments are rejected;
- the API projects only the configured public base URL plus routes whose executable gateway access class is `WORKLOAD_AUTHENTICATED`;
- health and metrics endpoints are never projected as application ingress;
- the ingress list comes from `aigateway.GatewayRoutes()` so Configure cannot drift from the executable gateway contract;
- publishing the base URL does not imply that the gateway runtime or an upstream provider is healthy. Desired/applied runtime state remains a separate truth signal.

Current workload ingress is `/v1/models`, `/v1/chat/completions` and `/v1/responses` according to the executable gateway route registry.

## Emergency outbound control

Emergency freeze is a separate authority from transport revision suspension. Suspending a transport revision cannot be treated as a kill switch because a gateway may still retain a prior known-good router.

- `ai_gateway_emergency_controls` stores one optimistic-versioned state per tenant/environment with only frozen state, reason, actor, timestamp and record version; its governance-owned schema is migration `000079_ai_governance_gateway_emergency_control`;
- `POST /api/v1/ai-governance/gateway-emergency-control` is `CONFIG_WRITE`, derives tenant and actor from authenticated server identity and requires a non-empty bounded reason;
- freeze/unfreeze persists the control and appends `AI_GATEWAY_OUTBOUND_FROZEN` / `AI_GATEWAY_OUTBOUND_UNFROZEN` to the canonical outbox in the same PostgreSQL transaction;
- gateway database transport checks the emergency state before cached known-good routing can be used and refreshes it on a bounded fast cadence of at most one second;
- a frozen state returns the explicit non-provider error `outbound_ai_frozen` before a new provider call is opened;
- after the emergency cache expires, inability to read or validate emergency state fails closed as `emergency_control_unavailable` rather than silently allowing outbound AI;
- unfreeze does not replace or rebuild the known-good transport snapshot; it re-enables request routing only after the gateway has observed the newer control revision;
- `GET /health/config` reports `emergency_supported`, `emergency_revision` and `outbound_frozen` so Configure distinguishes desired state from gateway-applied state;
- the control does not claim to cancel provider calls that were already in flight when a freeze was issued.

## Configure UX

`Configure → AI governance → Organization AI proxy` provides:

- environment selection;
- stable public proxy base URL and executable workload ingress capability inventory;
- governed provider connection metadata;
- fixed provider origin and adapter kind;
- opaque secret reference, region and enabled/suspended state;
- logical aliases with weighted/fallback upstream routes and cost metadata;
- change reason and revision history;
- maker/checker submit, approve, activate, suspend and retire lifecycle;
- desired database authority and actual gateway applied-state distinction;
- emergency freeze/unfreeze as a separate incident control requiring an explicit reason and acknowledgement;
- propagation state until the gateway reports the exact emergency-control revision, never a false claim that outbound AI has already stopped.

The UI must never describe database activation alone, publication of the base URL alone, or an emergency write that the gateway has not yet observed as proof of runtime state.

## Required regression proof

Repository CI must continue to prove:

- runtime/OpenAPI route parity;
- migration up/down and durable-schema ownership parity;
- no raw prompt/response/source-payload/provider-secret fields in AI-governance migrations;
- definition checksum stability;
- maker/checker separation;
- atomic active-revision supersession;
- database transport bootstrap without static provider secret resolution;
- known-good snapshot retention after failed refresh/apply;
- successful atomic swap to a later valid revision;
- caller-controlled unknown model aliases do not enter telemetry label cardinality;
- operations client rejects redirects and mismatched tenant/environment status;
- public proxy configuration rejects unsafe URLs and does not expose metrics/health routes as application capabilities;
- emergency freeze overrides a cached known-good router and emergency-control read failure fails closed;
- stale emergency writes are rejected and no-op freeze/unfreeze transitions are rejected;
- PostgreSQL freeze/unfreeze state and its canonical outbox audit event commit atomically;
- Configure typecheck, rendered accessibility tests, production build and deterministic Chromium review.
