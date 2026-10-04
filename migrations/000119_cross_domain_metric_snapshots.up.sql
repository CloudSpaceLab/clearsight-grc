BEGIN;

INSERT INTO metric_definitions(
    metric_id,revision,label,unit,basis,condition_rule,aggregation_rule,
    drill_workspace,drill_filter,drill_consistency
) VALUES
    ('risks_outside_appetite','enterprise-domain-v1','Outside appetite','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','risks','outside-appetite','SOURCE_SNAPSHOT'),
    ('indicator_breaches','enterprise-domain-v1','Indicator breaches','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','risks','indicator-breach','SOURCE_SNAPSHOT'),
    ('assurance_failures','enterprise-domain-v1','Assurance failures','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','risks','assurance-failed','SOURCE_SNAPSHOT'),
    ('losses_without_intervention','enterprise-domain-v1','Losses needing intervention','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','losses','needs-intervention','SOURCE_SNAPSHOT');

CREATE TABLE domain_metric_snapshots (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    definition_revision text NOT NULL CHECK (definition_revision='enterprise-domain-v1'),
    source_revision text NOT NULL CHECK (source_revision='enterprise-domain-v1'),
    source_high_water jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(source_high_water)='object'),
    bucket_start timestamptz NOT NULL,
    generated_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(tenant_id,legal_entity_id,definition_revision,bucket_start),
    UNIQUE(id,definition_revision),
    CONSTRAINT domain_metric_snapshot_entity_fk
        FOREIGN KEY (legal_entity_id,tenant_id)
        REFERENCES legal_entities(id,tenant_id),
    CHECK (bucket_start=date_trunc('hour',bucket_start)),
    CHECK (generated_at>=bucket_start AND generated_at<bucket_start+interval '1 hour')
);

CREATE INDEX domain_metric_snapshots_retention_idx
    ON domain_metric_snapshots(generated_at,id);
CREATE INDEX domain_metric_snapshots_entity_idx
    ON domain_metric_snapshots(tenant_id,legal_entity_id,generated_at DESC,id DESC);

CREATE TABLE domain_metric_snapshot_memberships (
    source_id uuid NOT NULL,
    metric_id text NOT NULL,
    definition_revision text NOT NULL CHECK (definition_revision='enterprise-domain-v1'),
    member_id uuid NOT NULL,
    target_type text NOT NULL CHECK (target_type IN ('RISK','LOSS')),
    target_id uuid NOT NULL,
    target_title text NOT NULL,
    state text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY(source_id,metric_id,definition_revision,member_id),
    CONSTRAINT domain_metric_membership_source_fk
        FOREIGN KEY(source_id,definition_revision)
        REFERENCES domain_metric_snapshots(id,definition_revision)
        ON DELETE CASCADE,
    CONSTRAINT domain_metric_membership_definition_fk
        FOREIGN KEY(metric_id,definition_revision)
        REFERENCES metric_definitions(metric_id,revision)
);

CREATE INDEX domain_metric_snapshot_memberships_drill_idx
    ON domain_metric_snapshot_memberships(source_id,metric_id,definition_revision,member_id);

CREATE FUNCTION prevent_domain_metric_snapshot_mutation() RETURNS trigger
LANGUAGE plpgsql
AS $domain_metric_snapshot$
BEGIN
    RAISE EXCEPTION 'Domain metric snapshots and memberships are immutable';
END;
$domain_metric_snapshot$;

CREATE TRIGGER domain_metric_snapshots_immutable
    BEFORE UPDATE ON domain_metric_snapshots
    FOR EACH ROW EXECUTE FUNCTION prevent_domain_metric_snapshot_mutation();

CREATE TRIGGER domain_metric_snapshot_memberships_immutable
    BEFORE UPDATE ON domain_metric_snapshot_memberships
    FOR EACH ROW EXECUTE FUNCTION prevent_domain_metric_snapshot_mutation();

ALTER TABLE metric_observations
    DROP CONSTRAINT metric_observations_source_kind_check;
ALTER TABLE metric_observations
    ADD CONSTRAINT metric_observations_source_kind_check
    CHECK (source_kind IN ('OVERSIGHT_SNAPSHOT','DOMAIN_SNAPSHOT'));

CREATE OR REPLACE FUNCTION validate_metric_observation_source() RETURNS trigger
LANGUAGE plpgsql
AS $metric_observation_source$
DECLARE
    source_membership_revision text;
    retained_member_count bigint;
BEGIN
    IF NEW.source_kind='DOMAIN_SNAPSHOT' THEN
        IF NOT EXISTS (
            SELECT 1
            FROM domain_metric_snapshots source
            WHERE source.id=NEW.source_id
              AND source.tenant_id=NEW.tenant_id
              AND source.legal_entity_id=NEW.legal_entity_id
              AND source.definition_revision=NEW.definition_revision
        ) THEN
            RAISE EXCEPTION 'Domain metric observation source does not match tenant/legal entity';
        END IF;
        SELECT count(*)
          INTO retained_member_count
          FROM domain_metric_snapshot_memberships member
         WHERE member.source_id=NEW.source_id
           AND member.metric_id=NEW.metric_id
           AND member.definition_revision=NEW.definition_revision;
        IF retained_member_count<>NEW.value THEN
            RAISE EXCEPTION 'Domain metric observation membership count does not match value';
        END IF;
        RETURN NEW;
    END IF;

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
