# Framework catalog publication reviews

Migration `000143_framework_publication_reviews` defines a draft-to-review record for one exact catalog release. It stores the manifest, release and source digests, source size, responsible tenant and maker, independent reviewer and rationale, decision time, version, demo marker and published edition identity. The immutable transition guard prevents rewriting the source, skipping states, changing a recorded review or deleting history. A published record must reference the same catalog/source digests. The downgrade refuses to remove retained history.

The schema is groundwork only: no repository, authority resolver, API, outbox integration or frontend is wired yet. The `is_demo` marker alone does not isolate a global catalog; demo publication must be limited to an explicitly isolated demo database and must not be exposed as a production approval. Callers must authenticate identity, resolve current global publication authority, verify the source bytes, and store authoritative state, audit event and outbox in one transaction. The model does not establish content rights or an independent real-world review.

The up and down migrations were exercised on the copied local database, which was returned to its previous state. Its migration registry is missing several previously applied migrations; that existing discrepancy needs reconciliation before deployment. No NIST edition has been published.

<!-- schema-ownership:begin -->
| Table | Classification | Owner | Writers | Readers | Lifecycle / valid time | Retention / deletion | Executable evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `framework_catalog_publication_reviews` | explicitly reserved for a named future phase | framework catalog publication review | future independently authorized publication workflow; no runtime writer currently connected | future publication review workspace; schema guards only | schema groundwork for exact source and manifest review | retain review history; downgrade refuses retained records | migration 000143 lifecycle constraints; runtime acceptance remains pending |
<!-- schema-ownership:end -->
