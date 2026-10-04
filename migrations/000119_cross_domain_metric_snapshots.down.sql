BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM metric_observations WHERE definition_revision='enterprise-domain-v1'
    ) OR EXISTS (
        SELECT 1 FROM metric_observation_daily_rollups WHERE definition_revision='enterprise-domain-v1'
    ) THEN
        RAISE EXCEPTION 'cannot roll back cross-domain metrics while retained observations exist';
    END IF;
END $$;

DROP TABLE IF EXISTS domain_metric_snapshot_memberships;
DROP TABLE IF EXISTS domain_metric_snapshots;
DROP FUNCTION IF EXISTS prevent_domain_metric_snapshot_mutation();

ALTER TABLE metric_observations
    DROP CONSTRAINT metric_observations_source_kind_check;
ALTER TABLE metric_observations
    ADD CONSTRAINT metric_observations_source_kind_check
    CHECK (source_kind='OVERSIGHT_SNAPSHOT');

ALTER TABLE metric_definitions DISABLE TRIGGER metric_definitions_immutable;
DELETE FROM metric_definitions WHERE revision='enterprise-domain-v1';
ALTER TABLE metric_definitions ENABLE TRIGGER metric_definitions_immutable;

-- Restore the source validator owned by 000116.
CREATE OR REPLACE FUNCTION validate_metric_observation_source() RETURNS trigger
LANGUAGE plpgsql
AS $metric_observation_source$
DECLARE
    source_membership_revision text;
    retained_member_count bigint;
BEGIN
    SELECT source.metric_membership_revision
      INTO source_membership_revision
      FROM oversight_snapshots source
     WHERE source.id=NEW.source_id
       AND source.tenant_id=NEW.tenant_id
       AND source.legal_entity_id=NEW.legal_entity_id;

    IF NOT FOUND OR NEW.source_kind<>'OVERSIGHT_SNAPSHOT' THEN
        RAISE EXCEPTION 'Metric observation source does not match tenant/legal entity';
    END IF;

    IF NEW.definition_revision='home-oversight-v3' THEN
        IF source_membership_revision IS DISTINCT FROM 'home-oversight-v3' OR NOT EXISTS (
            SELECT 1
            FROM oversight_snapshot_metric_membership_sets membership_set
            WHERE membership_set.oversight_snapshot_id=NEW.source_id
              AND membership_set.tenant_id=NEW.tenant_id
              AND membership_set.legal_entity_id=NEW.legal_entity_id
              AND membership_set.definition_revision=NEW.definition_revision
        ) THEN
            RAISE EXCEPTION 'Exact metric observation source has no retained membership revision';
        END IF;
        SELECT count(*)
          INTO retained_member_count
          FROM oversight_snapshot_metric_memberships member
         WHERE member.oversight_snapshot_id=NEW.source_id
           AND member.metric_id=NEW.metric_id
           AND member.definition_revision=NEW.definition_revision;
        IF retained_member_count<>NEW.value THEN
            RAISE EXCEPTION 'Exact metric observation membership count does not match value';
        END IF;
    END IF;
    RETURN NEW;
END;
$metric_observation_source$;

COMMIT;
