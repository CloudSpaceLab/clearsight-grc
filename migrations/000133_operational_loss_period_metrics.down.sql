BEGIN;

DELETE FROM metric_runtime_membership_sets
WHERE definition_revision='operational-loss-period-v1';

ALTER TABLE metric_definitions DISABLE TRIGGER metric_definitions_immutable;
DELETE FROM metric_definitions
WHERE revision='operational-loss-period-v1';
ALTER TABLE metric_definitions ENABLE TRIGGER metric_definitions_immutable;

ALTER TABLE metric_runtime_membership_sets
    DROP CONSTRAINT metric_runtime_membership_sets_definition_revision_check;
ALTER TABLE metric_runtime_membership_sets
    ADD CONSTRAINT metric_runtime_membership_sets_definition_revision_check
    CHECK (definition_revision IN ('home-oversight-v3','enterprise-domain-v1'));

COMMIT;
