# Legacy vendor response access

Completed responses attached to the legacy `VENDOR` subject are no longer visible through ordinary response or Documents reads. The supported vendor scope is `VENDOR_RELATIONSHIP`; these identifiers are not interchangeable. Before deploying to a database that will be preserved, run the following read-only inventory with the intended tenant UUID and legal entity UUID substituted as query parameters. Do not put identifiers or protected response content in shared logs.

```sql
SELECT d.status, count(DISTINCT d.id) AS distributions,
       count(r.id) AS response_revisions
FROM capture_form_distributions d
LEFT JOIN capture_response_revisions r
  ON r.tenant_id = d.tenant_id AND r.legal_entity_id = d.legal_entity_id
 AND r.distribution_id = d.id
WHERE d.tenant_id = $1::uuid AND d.legal_entity_id = $2::uuid
  AND upper(d.subject_type) = 'VENDOR'
GROUP BY d.status ORDER BY d.status;
```

No automatic mapping is approved. To preserve historical responses, determine the original vendor relationship using authoritative tenant- and entity-scoped links, identify ambiguous or missing links, agree visibility and history treatment, and test a versioned audited migration and rollback. Do not derive access merely from matching names or reinterpret the identifier. A clean demo database avoids migrating the old records only after the exact database and reset are confirmed; it does not establish compatibility with preserved data.
