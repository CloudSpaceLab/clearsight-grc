BEGIN;

ALTER TABLE metric_definitions
    DROP CONSTRAINT metric_definitions_drill_consistency_check;

ALTER TABLE metric_definitions
    ADD CONSTRAINT metric_definitions_drill_consistency_check
    CHECK (drill_consistency IN ('CURRENT_STATE','SOURCE_SNAPSHOT'));

ALTER TABLE oversight_snapshots
    ADD COLUMN metric_membership_revision text
    CHECK (metric_membership_revision IS NULL OR metric_membership_revision='home-oversight-v3');

INSERT INTO metric_definitions(
    metric_id,revision,label,unit,basis,condition_rule,aggregation_rule,
    drill_workspace,drill_filter,drill_consistency
) VALUES
    ('critical_high_open','home-oversight-v3','Critical and high','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','oversight','critical-high','SOURCE_SNAPSHOT'),
    ('overdue_open','home-oversight-v3','Overdue','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','oversight','overdue','SOURCE_SNAPSHOT'),
    ('routing_gaps','home-oversight-v3','Routing gaps','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','oversight','routing-gaps','SOURCE_SNAPSHOT'),
    ('outcome_failures','home-oversight-v3','Outcome failures','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','oversight','outcome-failures','SOURCE_SNAPSHOT');

CREATE TABLE oversight_snapshot_metric_membership_sets (
    oversight_snapshot_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    definition_revision text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (oversight_snapshot_id, definition_revision),
    CONSTRAINT oversight_metric_membership_set_entity_fk
        FOREIGN KEY (legal_entity_id, tenant_id)
        REFERENCES legal_entities(id, tenant_id)
);

CREATE FUNCTION validate_oversight_metric_membership_set_source() RETURNS trigger
LANGUAGE plpgsql
AS $oversight_metric_membership_set_source$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM oversight_snapshots source
        WHERE source.id=NEW.oversight_snapshot_id
          AND source.tenant_id=NEW.tenant_id
          AND source.legal_entity_id=NEW.legal_entity_id
          AND source.metric_membership_revision=NEW.definition_revision
    ) THEN
        RAISE EXCEPTION 'Metric membership set source does not match marked oversight snapshot';
    END IF;
    RETURN NEW;
END;
$oversight_metric_membership_set_source$;

CREATE TRIGGER oversight_snapshot_metric_membership_sets_source
    BEFORE INSERT ON oversight_snapshot_metric_membership_sets
    FOR EACH ROW EXECUTE FUNCTION validate_oversight_metric_membership_set_source();

CREATE TABLE oversight_snapshot_metric_memberships (
    oversight_snapshot_id uuid NOT NULL,
    metric_id text NOT NULL,
    definition_revision text NOT NULL,
    member_id uuid NOT NULL,
    target_type text NOT NULL CHECK (target_type IN ('MATTER','PROGRAM')),
    target_id uuid NOT NULL,
    target_title text NOT NULL,
    state text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (oversight_snapshot_id, metric_id, definition_revision, member_id),
    CONSTRAINT oversight_metric_membership_definition_fk
        FOREIGN KEY (metric_id, definition_revision)
        REFERENCES metric_definitions(metric_id, revision),
    CONSTRAINT oversight_metric_membership_set_fk
        FOREIGN KEY (oversight_snapshot_id, definition_revision)
        REFERENCES oversight_snapshot_metric_membership_sets(oversight_snapshot_id, definition_revision)
);

CREATE INDEX oversight_snapshot_metric_memberships_drill_idx
    ON oversight_snapshot_metric_memberships(
        oversight_snapshot_id, metric_id, definition_revision, member_id
    );

CREATE FUNCTION prevent_oversight_metric_membership_mutation() RETURNS trigger
LANGUAGE plpgsql
AS $oversight_metric_membership$
BEGIN
    RAISE EXCEPTION 'Oversight metric memberships are immutable';
END;
$oversight_metric_membership$;

CREATE TRIGGER oversight_snapshot_metric_membership_sets_immutable
    BEFORE UPDATE OR DELETE ON oversight_snapshot_metric_membership_sets
    FOR EACH ROW EXECUTE FUNCTION prevent_oversight_metric_membership_mutation();

CREATE TRIGGER oversight_snapshot_metric_memberships_immutable
    BEFORE UPDATE OR DELETE ON oversight_snapshot_metric_memberships
    FOR EACH ROW EXECUTE FUNCTION prevent_oversight_metric_membership_mutation();


CREATE TABLE metric_runtime_membership_sets (
    source_id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    organization_scope_id uuid,
    definition_revision text NOT NULL,
    source_revision text NOT NULL,
    request_fingerprint text NOT NULL,
    generated_at timestamptz NOT NULL,
    period_start timestamptz NOT NULL,
    period_end timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    CONSTRAINT metric_runtime_membership_entity_fk
        FOREIGN KEY (legal_entity_id, tenant_id)
        REFERENCES legal_entities(id, tenant_id),
    CONSTRAINT metric_runtime_membership_scope_fk
        FOREIGN KEY (tenant_id, legal_entity_id, organization_scope_id)
        REFERENCES organization_scopes(tenant_id, legal_entity_id, id),
    CHECK (source_revision=btrim(source_revision) AND source_revision<>''),
    CHECK (request_fingerprint=btrim(request_fingerprint) AND request_fingerprint<>''),
    CHECK (period_start<=period_end),
    CHECK (expires_at>generated_at)
);

CREATE UNIQUE INDEX metric_runtime_membership_fingerprint_idx
    ON metric_runtime_membership_sets(
        tenant_id,legal_entity_id,COALESCE(organization_scope_id,'00000000-0000-0000-0000-000000000000'::uuid),
        definition_revision,request_fingerprint
    );
CREATE INDEX metric_runtime_membership_expiry_idx
    ON metric_runtime_membership_sets(expires_at);

CREATE TABLE metric_runtime_memberships (
    source_id uuid NOT NULL,
    metric_id text NOT NULL,
    definition_revision text NOT NULL,
    member_id uuid NOT NULL,
    target_type text NOT NULL CHECK (target_type IN ('MATTER','PROGRAM')),
    target_id uuid NOT NULL,
    target_title text NOT NULL,
    state text NOT NULL,
    PRIMARY KEY (source_id,metric_id,definition_revision,member_id),
    CONSTRAINT metric_runtime_membership_set_fk
        FOREIGN KEY (source_id)
        REFERENCES metric_runtime_membership_sets(source_id)
        ON DELETE CASCADE,
    CONSTRAINT metric_runtime_membership_definition_fk
        FOREIGN KEY (metric_id,definition_revision)
        REFERENCES metric_definitions(metric_id,revision)
);

CREATE INDEX metric_runtime_memberships_drill_idx
    ON metric_runtime_memberships(source_id,metric_id,definition_revision,member_id);

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

    IF NOT FOUND OR NEW.source_kind <> 'OVERSIGHT_SNAPSHOT' THEN
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
        IF retained_member_count <> NEW.value THEN
            RAISE EXCEPTION 'Exact metric observation membership count does not match value';
        END IF;
    END IF;
    RETURN NEW;
END;
$metric_observation_source$;

COMMIT;
