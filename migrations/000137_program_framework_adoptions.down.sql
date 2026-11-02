BEGIN;

-- This schema retains customer review and framework-reference provenance.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM framework_clause_requirement_links LIMIT 1)
        OR EXISTS (SELECT 1 FROM program_framework_adoptions LIMIT 1) THEN
        RAISE EXCEPTION 'cannot downgrade populated Program framework adoption history';
    END IF;
END;
$$;

DROP TABLE framework_clause_requirement_links;
DROP TABLE program_framework_adoptions;
DROP FUNCTION guard_framework_clause_requirement_link();
DROP INDEX framework_catalog_clauses_edition_fk_idx;
DROP INDEX framework_catalog_editions_identity_fk_idx;

COMMIT;
