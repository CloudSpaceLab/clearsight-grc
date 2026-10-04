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

CREATE OR REPLACE FUNCTION validate_metric_observation_source() RETURNS trigger
LANGUAGE plpgsql
AS $metric_observation_source$
BEGIN
    IF NEW.source_kind <> 'OVERSIGHT_SNAPSHOT' OR NOT EXISTS (
        SELECT 1
        FROM oversight_snapshots source
        WHERE source.id=NEW.source_id
          AND source.tenant_id=NEW.tenant_id
          AND source.legal_entity_id=NEW.legal_entity_id
    ) THEN
        RAISE EXCEPTION 'Metric observation source does not match tenant/legal entity';
    END IF;
    RETURN NEW;
END;
$metric_observation_source$;

DROP TABLE IF EXISTS metric_runtime_memberships;
DROP TABLE IF EXISTS metric_runtime_membership_sets;
DROP TABLE IF EXISTS oversight_snapshot_metric_memberships;
DROP TABLE IF EXISTS oversight_snapshot_metric_membership_sets;
DROP FUNCTION IF EXISTS validate_oversight_metric_membership_set_source();
DROP FUNCTION IF EXISTS prevent_oversight_metric_membership_mutation();

ALTER TABLE oversight_snapshots DROP COLUMN IF EXISTS metric_membership_revision;

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
