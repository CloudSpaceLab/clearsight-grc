# Authority grant compatibility inventory

Before deploying the new authority resolver, run the inventory against the target database using a database account with read access to `authority_grants`. It starts a read-only transaction and changes no records:

```sh
psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -f deploy/scripts/inventory-authority-grants.sql
```

Load the connection setting through the host's protected configuration; do not print credentials or store query output in a public artifact. The output includes tenant, scope, grant identifiers, validity, unsupported key names and issue categories. It includes expired and future grants, so review validity as well as impact. Zero rows means this check found no incompatible grant limits at the time of the query; it does not certify route correctness or later changes.

Only integer `min_materiality` and `max_materiality` values from 0 through 5 are supported. A malformed value or unsupported key makes the grant ineligible. An inverted range has no eligible materiality. An authorized policy owner must decide how to handle each affected grant. Do not strip unknown restrictions or silently replace values; use the approved policy-change and audit process. Re-run the inventory after remediation and close to deployment.
