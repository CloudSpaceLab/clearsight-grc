# Contextual copilot — Design and production integration contract

## Implemented reference

`assets/copilot.js` is a deterministic local adapter. There is no language-model call, secret, external endpoint or document-content ingestion. The UI labels this explicitly. Source fields are treated as data, escaped at rendering and never executed. The tested workflows issued no external HTTP requests.

Supported contexts include live executive reports, published report packs, scoped risks, programs and exact assessment targets, form design, form responses, vendor engagements, ROPA/DPIA/cases, monthly KRI returns, document registers/versions, approval state and Work. Unsupported pages do not advertise the assistant. Registration/adoption wizards are not interrupted by it.

## Read contract

A context contains a page/record key, actor, scope, as-of date, distinct reporting periods, bounded facts and gaps, typed source references and exclusions. A stable digest includes source inputs and resulting facts/gaps. Source labels and routes come from the existing registry. Source totals reuse canonical report functions; no second score engine is permitted.

A published report explanation must use its captured population and versions. A monthly KRI explanation must distinguish draft/submitted/accepted. A form designer is not an assessor. A file approval is not an evidence-acceptance or compliance conclusion. Missing, unknown and stale cannot become zero or Pass.

Restricted documents are excluded, including titles. Other document context is metadata-only; original bytes and seeded `sampleText` are excluded. Production policy may impose additional restrictions by actor, purpose, jurisdiction or residency.

## Proposal contract

The only demonstrated write is a reviewed ordinary work item:

`read-model context → cited proposal → existing action form → explicit human confirmation → existing Work record`

The proposal captures its source context hash and typed sources. Confirmation rechecks that context and validates owner, date, target scope and related record. The resulting action retains reviewedBy and proposal provenance. A changed context requires regeneration. Cancel causes no mutation.

There is no approval/sign-off/send/delete/publish shortcut, hidden batch operation, automatic risk reduction, or automatic requirement satisfaction. Existing maker/checker paths remain the decision authority.

## Production adapter boundary

Before a real model is connected, implement these gates in the server, not in the HTML:

1. Authenticate actor/tenant; authorize retrieval per source and purpose; resolve allowed descendants. Pass only the minimized context and approved evidence excerpts.
2. Use an allowlisted tool catalogue with read-only tools by default. Keep secrets and original document downloads out of prompts and browser state. Apply residency, retention and vendor policy.
3. Treat retrieved text and files as untrusted input, never as instructions granting tools or authority. Bind tool calls to the originating user and scope.
4. Return a structured answer with claim/source/version references, uncertainty and exclusions. Resolve sources through authorized routes; redact inaccessible citations rather than widening scope.
5. Validate commands in the existing domain service, with source revisions, idempotency, optimistic concurrency, quotas and independent approval requirements. The model is not an identity or approval authority.
6. Test permission loss, changed scope, stale snapshots, hostile document text, duplicate submissions, model unavailability and incomplete sources. Preserve normal manual workflows when the model is disabled.
7. Provide visible pending/error/cancel/retry states, streaming cancellation and bounded response length for the eventual live adapter. These network states are not simulated as working in the local reference.

## Sources for the production security boundary

OWASP LLM Prompt Injection Prevention Cheat Sheet:
https://cheatsheetseries.owasp.org/cheatsheets/LLM_Prompt_Injection_Prevention_Cheat_Sheet.html

OWASP AI Agent Security Cheat Sheet:
https://cheatsheetseries.owasp.org/cheatsheets/AI_Agent_Security_Cheat_Sheet.html

These inform the handoff, not a claim that a live model or production security controls were implemented or certified.
