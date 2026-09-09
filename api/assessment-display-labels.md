# Assessment display labels

Assessment reads may enrich recorded principal IDs with optional display names after the underlying read or command has passed its existing identity, legal-entity and authority checks.

| Response | Optional field |
| --- | --- |
| `GET` and `POST /api/v1/forms/responses/{revision_id}/assessment` | `fields[].decision.reviewer_display_name` |
| Vendor assessment review read and document-review response | `application_receipt.actor_display_name` |
| Vendor assessment response application | `receipt.actor_display_name` and `review.application_receipt.actor_display_name` |

Names resolve the stored `reviewer_id` or `actor_principal_id`, using the verified request tenant and the assessment's exact legal entity. The resolver must return the same tenant, entity and principal. Reads deduplicate IDs and stop at the directory's maximum principal batch size; they do not load a directory population. Missing scope, missing directory service, an unavailable principal or a mismatched resolution omits the name. Optional enrichment failure does not turn a committed command into an error.

These fields identify the principal who recorded the decision or application. They do not identify a pending assignee and are not immutable snapshots of the person's historical name. Persisted decisions, principal IDs, audit events and authority evaluation remain unchanged. Clients retain the recorded ID in history and show a missing-name state when enrichment is absent. Clients must not substitute a current owner or reviewer for a historical actor. Pending review remains **Awaiting review** unless another authorized contract supplies the current responsible party.

`api/runtime.openapi.json` remains the executable route/access-class contract; it does not declare these detailed response schemas.
