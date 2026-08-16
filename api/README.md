# API contract ownership

ClearSight deliberately has one executable route/access inventory.

## Executable contract

`internal/httpapi/route_registry.go` is the canonical runtime route inventory. `runtime.openapi.json` is the mechanically verified projection of that registry and records the route, HTTP method, access mode and permission contract used by the running server.

A route is not executable merely because it appears in a design document or a domain-specific schema file. Authorization truth comes from the route registry and the command/access guards used by the registered handler.

## Isolated AI gateway contract

`cmd/ai-gateway` is a separate process and does not register routes in the main API. Its executable route/access inventory is `internal/aigateway/routes.go`; `ai-gateway.openapi.json` is mechanically checked against that inventory. The contract is limited to workload-authenticated OpenAI-compatible model transport, separate metrics authentication and public liveness/readiness. It does not grant ClearSight application permissions or override `runtime.openapi.json`.

## Domain descriptive specifications

- `bank-journeys.openapi.yaml` describes the bank-reference journey payload surface.
- `document-imports.openapi.yaml` describes governed document-import payloads and review states.

These files may document domain shapes and examples. They do **not** create routes, grant access or override `runtime.openapi.json`.

## Removed broad duplicate

The former `openapi.yaml` was a manually maintained broad catalogue that duplicated the executable route list. By P2 it had drifted enough to advertise retired legacy capture aliases and direct Workflow Task mutations that the runtime deliberately no longer exposes.

Maintaining two route catalogues creates false capability and authorization claims, so P2 removes the broad duplicate rather than adding another synchronization layer. New executable routes must be added to `internal/httpapi/route_registry.go` and pass the existing runtime-contract parity tests.

The web application currently uses explicit TypeScript API adapters/types rather than an OpenAPI-generated client. If generated clients are introduced later, generate them from an explicitly owned contract rather than treating descriptive domain files as executable authorization truth.
