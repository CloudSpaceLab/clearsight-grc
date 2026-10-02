BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM risk_control_links LIMIT 1)
       OR EXISTS (SELECT 1 FROM control_catalog_implementation_links LIMIT 1)
       OR EXISTS (SELECT 1 FROM control_definitions LIMIT 1) THEN
        RAISE EXCEPTION 'Retain governed control catalog records before downgrade';
    END IF;
END;
$$;

DROP TRIGGER IF EXISTS risk_control_links_immutable ON risk_control_links;
DROP TRIGGER IF EXISTS control_catalog_links_immutable ON control_catalog_implementation_links;
DROP TRIGGER IF EXISTS control_definitions_immutable ON control_definitions;
DROP FUNCTION IF EXISTS protect_control_catalog_immutable();
DROP TABLE IF EXISTS risk_control_links;
DROP TABLE IF EXISTS control_catalog_implementation_links;
DROP TABLE IF EXISTS control_definitions;
DROP INDEX IF EXISTS programs_scope_identity_idx;

COMMIT;
