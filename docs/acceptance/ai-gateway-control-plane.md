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

The gateway operations endpoint `GET /health/config` exposes only desired/applied revision metadata and stable error codes. It is protected by the independent operations/metrics bearer credential.

The ClearSight API may be configured with `CLEARSIGHT_AI_GATEWAY_OPERATIONS_URL` and `CLEARSIGHT_AI_GATEWAY_OPERATIONS_TOKEN`. This is a server-to-server bridge only: Configure receives the projected runtime status together with the governed revision list; the operations credential is never exposed to the browser. Missing bridge configuration is represented as **not connected**, while a configured but unreachable gateway is represented as **unavailable/degraded**.

## Configure UX

`Configure → AI governance → Organization AI proxy` provides:

- environment selection;
- governed provider connection metadata;
- fixed provider origin and adapter kind;
- opaque secret reference, region and enabled/suspended state;
- logical aliases with weighted/fallback upstream routes and cost metadata;
- change reason and revision history;
- maker/checker submit, approve, activate, suspend and retire lifecycle;
- desired database authority and actual gateway applied-state distinction.

The UI must never describe database activation alone as proof that a gateway process has applied the revision.

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
- Configure typecheck, rendered accessibility tests, production build and deterministic Chromium review.
