# Governed form authoring schema ownership

This fragment extends the executable durable-schema ownership register for governed document-to-form authoring proposals. The repository test loads it together with the core register and rejects duplicate or incomplete table ownership.

<!-- schema-ownership:begin -->
| Table | Classification | Owner | Writers | Readers | Lifecycle / valid time | Retention / deletion | Executable evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `form_template_proposals` | active authoritative state | monitoring / governed Forms authoring | form proposal service and PostgreSQL proposal acceptance transaction | Forms proposal review API, proposal worker and audit/reconstruction consumers | exact source document/version/hash and optional base form revision pinned at generation; generating → review required → accepted/rejected/failed with optimistic versioning and exact accepted change IDs | retain with source import, resulting form revision and authoring audit history so accepted/rejected proposal provenance remains reconstructable | migration `000057_form_template_proposals`; deterministic proposal, worker, HTTP and PostgreSQL acceptance tests |
<!-- schema-ownership:end -->
