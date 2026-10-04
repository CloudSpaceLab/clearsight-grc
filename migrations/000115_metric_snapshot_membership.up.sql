BEGIN;

ALTER TABLE metric_definitions
    DROP CONSTRAINT metric_definitions_drill_consistency_check;

ALTER TABLE metric_definitions
    ADD CONSTRAINT metric_definitions_drill_consistency_check
    CHECK (drill_consistency IN ('CURRENT_STATE','SOURCE_SNAPSHOT'));

INSERT INTO metric_definitions(
    metric_id,revision,label,unit,basis,condition_rule,aggregation_rule,
    drill_workspace,drill_filter,drill_consistency
) VALUES
    ('critical_high_open','home-oversight-v3','Critical and high','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','oversight','critical-high','SOURCE_SNAPSHOT'),
    ('overdue_open','home-oversight-v3','Overdue','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','oversight','overdue','SOURCE_SNAPSHOT'),
    ('routing_gaps','home-oversight-v3','Routing gaps','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','oversight','routing-gaps','SOURCE_SNAPSHOT'),
    ('outcome_failures','home-oversight-v3','Outcome failures','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','oversight','outcome-failures','SOURCE_SNAPSHOT');

CREATE TABLE oversight_snapshot_metric_memberships (
    oversight_snapshot_id uuid NOT NULL REFERENCES oversight_snapshots(id) ON DELETE CASCADE,
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
        REFERENCES metric_definitions(metric_id, revision)
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

CREATE TRIGGER oversight_snapshot_metric_memberships_immutable
    BEFORE UPDATE OR DELETE ON oversight_snapshot_metric_memberships
    FOR EACH ROW EXECUTE FUNCTION prevent_oversight_metric_membership_mutation();

COMMIT;
