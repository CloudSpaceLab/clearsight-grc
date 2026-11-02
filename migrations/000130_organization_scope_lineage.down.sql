BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM organization_scope_lineage_events
        WHERE event_kind<>'BACKFILL'
    ) THEN
        RAISE EXCEPTION 'Organization scope lineage history exists; refusing to erase it';
    END IF;
END;
$$;

DROP TRIGGER organization_scopes_capture_lineage ON organization_scopes;
DROP FUNCTION capture_organization_scope_lineage();
DROP TRIGGER organization_scope_lineage_immutable ON organization_scope_lineage_events;
DROP FUNCTION prevent_organization_scope_lineage_mutation();
DROP TABLE organization_scope_lineage_events;

COMMIT;
