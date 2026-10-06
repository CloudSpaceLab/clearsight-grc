BEGIN;

DELETE FROM metric_runtime_membership_sets
WHERE definition_revision='enterprise-domain-v1';

ALTER TABLE metric_runtime_memberships
    DROP CONSTRAINT metric_runtime_memberships_target_type_check;
ALTER TABLE metric_runtime_memberships
    ADD CONSTRAINT metric_runtime_memberships_target_type_check
    CHECK (target_type IN ('MATTER','PROGRAM'));

ALTER TABLE metric_runtime_membership_sets
    DROP CONSTRAINT metric_runtime_membership_sets_definition_revision_check;
ALTER TABLE metric_runtime_membership_sets
    ADD CONSTRAINT metric_runtime_membership_sets_definition_revision_check
    CHECK (definition_revision='home-oversight-v3');

COMMIT;
