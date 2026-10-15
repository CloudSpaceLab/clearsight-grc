BEGIN;

-- Never silently discard published authority/source references.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM framework_catalog_clauses LIMIT 1)
       OR EXISTS (SELECT 1 FROM framework_catalog_editions LIMIT 1) THEN
        RAISE EXCEPTION 'cannot downgrade populated framework catalog';
    END IF;
END;
$$;

DROP TABLE framework_catalog_clauses;
DROP TABLE framework_catalog_editions;
DROP FUNCTION reject_framework_catalog_content_mutation();
DROP FUNCTION enforce_framework_catalog_clause_license();

COMMIT;
