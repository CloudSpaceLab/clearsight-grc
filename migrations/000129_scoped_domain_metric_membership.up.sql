BEGIN;

ALTER TABLE metric_runtime_membership_sets
    DROP CONSTRAINT metric_runtime_membership_sets_definition_revision_check;
ALTER TABLE metric_runtime_membership_sets
    ADD CONSTRAINT metric_runtime_membership_sets_definition_revision_check
    CHECK (definition_revision IN ('home-oversight-v3','enterprise-domain-v1'));

ALTER TABLE domain_metric_snapshot_memberships
    ADD COLUMN organization_scope_id uuid REFERENCES organization_scopes(id);
CREATE INDEX domain_metric_snapshot_memberships_scope_idx
    ON domain_metric_snapshot_memberships(source_id,metric_id,organization_scope_id);

ALTER TABLE metric_runtime_memberships
    ADD COLUMN organization_scope_id uuid REFERENCES organization_scopes(id);
CREATE INDEX metric_runtime_memberships_scope_idx
    ON metric_runtime_memberships(source_id,metric_id,organization_scope_id);

ALTER TABLE metric_runtime_memberships
    DROP CONSTRAINT metric_runtime_memberships_target_type_check;
ALTER TABLE metric_runtime_memberships
    ADD CONSTRAINT metric_runtime_memberships_target_type_check
    CHECK (target_type IN ('MATTER','PROGRAM','RISK','LOSS'));

COMMIT;
