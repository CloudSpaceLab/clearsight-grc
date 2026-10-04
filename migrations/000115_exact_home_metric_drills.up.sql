BEGIN;

ALTER TABLE metric_definitions
    DROP CONSTRAINT metric_definitions_drill_consistency_check;

ALTER TABLE metric_definitions
    ADD CONSTRAINT metric_definitions_drill_consistency_check
    CHECK (drill_consistency IN ('CURRENT_STATE','SNAPSHOT_EXACT'));

INSERT INTO metric_definitions(
    metric_id,revision,label,unit,basis,condition_rule,aggregation_rule,
    drill_workspace,drill_filter,drill_consistency
) VALUES
    ('critical_high_open','home-oversight-v3','Critical and high','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','oversight','critical-high','SNAPSHOT_EXACT'),
    ('overdue_open','home-oversight-v3','Overdue','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','oversight','overdue','SNAPSHOT_EXACT'),
    ('routing_gaps','home-oversight-v3','Routing gaps','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','oversight','routing-gaps','SNAPSHOT_EXACT'),
    ('outcome_failures','home-oversight-v3','Outcome failures','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','oversight','outcome-failures','SNAPSHOT_EXACT');

ALTER TABLE oversight_snapshots
    ADD COLUMN metric_membership_version text;

ALTER TABLE oversight_snapshots
    ADD CONSTRAINT oversight_snapshots_metric_membership_version_check
    CHECK (metric_membership_version IS NULL OR metric_membership_version='home-membership-v1');

CREATE TABLE oversight_metric_members (
    snapshot_id uuid NOT NULL,
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    source_generated_at timestamptz NOT NULL,
    metric_id text NOT NULL CHECK (metric_id IN ('critical_high_open','overdue_open','routing_gaps','outcome_failures')),
    member_type text NOT NULL CHECK (member_type IN ('MATTER','WORKFLOW_TASK')),
    member_id uuid NOT NULL,
    subject_type text NOT NULL CHECK (subject_type IN ('MATTER','PROGRAM')),
    subject_id uuid NOT NULL,
    reference text NOT NULL DEFAULT '',
    title text NOT NULL,
    state text NOT NULL,
    priority integer CHECK (priority IS NULL OR priority BETWEEN 1 AND 5),
    due_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (snapshot_id, metric_id, member_type, member_id),
    CONSTRAINT oversight_metric_member_entity_fk
        FOREIGN KEY (legal_entity_id, tenant_id)
        REFERENCES legal_entities(id, tenant_id),
    CONSTRAINT oversight_metric_member_shape_check CHECK (
        (member_type='MATTER' AND subject_type='MATTER' AND member_id=subject_id)
        OR member_type='WORKFLOW_TASK'
    )
);

CREATE INDEX oversight_metric_members_drill_idx
    ON oversight_metric_members(tenant_id, legal_entity_id, snapshot_id, metric_id, member_type, member_id);

CREATE INDEX oversight_metric_members_retention_idx
    ON oversight_metric_members(source_generated_at, snapshot_id);

CREATE FUNCTION validate_oversight_metric_member_source() RETURNS trigger
LANGUAGE plpgsql
AS $oversight_metric_member_source$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM oversight_snapshots source
        WHERE source.id=NEW.snapshot_id
          AND source.tenant_id=NEW.tenant_id
          AND source.legal_entity_id=NEW.legal_entity_id
          AND source.generated_at=NEW.source_generated_at
          AND source.metric_membership_version='home-membership-v1'
    ) THEN
        RAISE EXCEPTION 'Oversight metric membership source does not match tenant/legal entity';
    END IF;
    RETURN NEW;
END;
$oversight_metric_member_source$;

CREATE TRIGGER oversight_metric_members_validate_source
    BEFORE INSERT ON oversight_metric_members
    FOR EACH ROW EXECUTE FUNCTION validate_oversight_metric_member_source();

CREATE FUNCTION prevent_oversight_metric_member_update() RETURNS trigger
LANGUAGE plpgsql
AS $oversight_metric_member$
BEGIN
    RAISE EXCEPTION 'Oversight metric membership is immutable';
END;
$oversight_metric_member$;

CREATE TRIGGER oversight_metric_members_immutable
    BEFORE UPDATE ON oversight_metric_members
    FOR EACH ROW EXECUTE FUNCTION prevent_oversight_metric_member_update();

COMMIT;
