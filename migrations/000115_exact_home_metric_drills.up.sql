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
    snapshot_id uuid NOT NULL REFERENCES oversight_snapshots(id) ON DELETE CASCADE,
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
    CONSTRAINT oversight_metric_member_shape_check CHECK (
        (member_type='MATTER' AND subject_type='MATTER' AND member_id=subject_id)
        OR member_type='WORKFLOW_TASK'
    )
);

CREATE INDEX oversight_metric_members_drill_idx
    ON oversight_metric_members(snapshot_id, metric_id, member_type, member_id);

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
