BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM control_catalog_implementation_links LIMIT 1)
       OR EXISTS (SELECT 1 FROM control_definitions LIMIT 1) THEN
        RAISE EXCEPTION 'Retain governed control catalog records before downgrade';
    END IF;
END;
$$;

DROP TABLE IF EXISTS control_catalog_implementation_links;
DROP TABLE IF EXISTS control_definitions;
DROP INDEX IF EXISTS programs_scope_identity_idx;

COMMIT;
