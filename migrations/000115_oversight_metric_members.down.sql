BEGIN;

DO $oversight_metric_member_downgrade$
BEGIN
    IF EXISTS (SELECT 1 FROM oversight_metric_members) THEN
        RAISE EXCEPTION 'Refusing to discard retained oversight metric membership';
    END IF;
END;
$oversight_metric_member_downgrade$;

DROP TABLE IF EXISTS oversight_metric_members;
DROP FUNCTION IF EXISTS prevent_oversight_metric_member_mutation();

ALTER TABLE oversight_snapshots
    DROP CONSTRAINT IF EXISTS oversight_snapshots_scope_identity_unique;

COMMIT;
