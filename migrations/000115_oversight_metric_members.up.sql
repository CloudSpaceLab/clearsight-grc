BEGIN;

ALTER TABLE oversight_snapshots
    ADD CONSTRAINT oversight_snapshots_scope_identity_unique
    UNIQUE (id,tenant_id,legal_entity_id);

CREATE TABLE oversight_metric_members (
    source_snapshot_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    metric_id text NOT NULL CHECK (metric_id IN (
        'critical_high_open','overdue_open','routing_gaps','outcome_failures'
    )),
    target_type text NOT NULL CHECK (target_type IN ('MATTER','WORKFLOW_TASK')),
    target_id uuid NOT NULL,
    label text NOT NULL CHECK (btrim(label)<>'' AND length(label)<=500),
    subject_type text NOT NULL CHECK (subject_type IN ('MATTER','PROGRAM')),
    subject_id uuid NOT NULL,
    captured_at timestamptz NOT NULL,
    PRIMARY KEY (source_snapshot_id,metric_id,target_type,target_id),
    CONSTRAINT oversight_metric_member_source_fk
        FOREIGN KEY (source_snapshot_id,tenant_id,legal_entity_id)
        REFERENCES oversight_snapshots(id,tenant_id,legal_entity_id)
        ON DELETE CASCADE
);

CREATE INDEX oversight_metric_members_read_idx
    ON oversight_metric_members(
        tenant_id,legal_entity_id,source_snapshot_id,metric_id,
        target_type,target_id
    );

CREATE FUNCTION prevent_oversight_metric_member_mutation() RETURNS trigger
LANGUAGE plpgsql
AS $oversight_metric_member$
BEGIN
    RAISE EXCEPTION 'Oversight metric membership is immutable';
END;
$oversight_metric_member$;

CREATE TRIGGER oversight_metric_members_immutable
    BEFORE UPDATE OR DELETE ON oversight_metric_members
    FOR EACH ROW EXECUTE FUNCTION prevent_oversight_metric_member_mutation();

COMMIT;
