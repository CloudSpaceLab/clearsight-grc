BEGIN;

DO $metric_membership_downgrade$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM metric_observations
        WHERE definition_revision='home-oversight-v3'
    ) THEN
        RAISE EXCEPTION 'Refusing to discard retained v3 metric observations';
    END IF;
END;
$metric_membership_downgrade$;

DROP TABLE IF EXISTS oversight_snapshot_metric_memberships;
DROP FUNCTION IF EXISTS prevent_oversight_metric_membership_mutation();

DROP TRIGGER IF EXISTS metric_definitions_immutable ON metric_definitions;
DELETE FROM metric_definitions WHERE revision='home-oversight-v3';

ALTER TABLE metric_definitions
    DROP CONSTRAINT metric_definitions_drill_consistency_check;
ALTER TABLE metric_definitions
    ADD CONSTRAINT metric_definitions_drill_consistency_check
    CHECK (drill_consistency IN ('CURRENT_STATE'));

CREATE TRIGGER metric_definitions_immutable
    BEFORE UPDATE OR DELETE ON metric_definitions
    FOR EACH ROW EXECUTE FUNCTION prevent_metric_definition_mutation();

COMMIT;
