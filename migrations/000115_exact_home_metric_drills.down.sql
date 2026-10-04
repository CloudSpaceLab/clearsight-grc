BEGIN;

DO $exact_metric_drill_downgrade$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM metric_observations
        WHERE definition_revision='home-oversight-v3'
    ) THEN
        RAISE EXCEPTION 'Refusing to discard exact-drill metric definition while observations reference it';
    END IF;
END;
$exact_metric_drill_downgrade$;

DROP TABLE IF EXISTS oversight_metric_members;
DROP FUNCTION IF EXISTS prevent_oversight_metric_member_update();

ALTER TABLE oversight_snapshots
    DROP CONSTRAINT IF EXISTS oversight_snapshots_metric_membership_version_check;
ALTER TABLE oversight_snapshots
    DROP COLUMN IF EXISTS metric_membership_version;

DROP TRIGGER IF EXISTS metric_definitions_immutable ON metric_definitions;

DELETE FROM metric_definitions
WHERE revision='home-oversight-v3';

CREATE TRIGGER metric_definitions_immutable
    BEFORE UPDATE OR DELETE ON metric_definitions
    FOR EACH ROW EXECUTE FUNCTION prevent_metric_definition_mutation();

ALTER TABLE metric_definitions
    DROP CONSTRAINT metric_definitions_drill_consistency_check;
ALTER TABLE metric_definitions
    ADD CONSTRAINT metric_definitions_drill_consistency_check
    CHECK (drill_consistency IN ('CURRENT_STATE'));

COMMIT;
