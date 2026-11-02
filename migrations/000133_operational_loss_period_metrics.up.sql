BEGIN;

INSERT INTO metric_definitions(
    metric_id,revision,label,unit,basis,condition_rule,aggregation_rule,
    drill_workspace,drill_filter,drill_consistency
) VALUES
    ('operational_loss_events','operational-loss-period-v1','Loss events','COUNT','PERIOD_FLOW','NO_CONDITION','SUM_DISJOINT_COUNTS','losses','period-events','SOURCE_SNAPSHOT'),
    ('operational_loss_net','operational-loss-period-v1','Net operational loss','MONEY','PERIOD_FLOW','NO_CONDITION','SUM_SAME_CURRENCY','losses','period-net','SOURCE_SNAPSHOT');

ALTER TABLE metric_runtime_membership_sets
    DROP CONSTRAINT metric_runtime_membership_sets_definition_revision_check;
ALTER TABLE metric_runtime_membership_sets
    ADD CONSTRAINT metric_runtime_membership_sets_definition_revision_check
    CHECK (definition_revision IN ('home-oversight-v3','enterprise-domain-v1','operational-loss-period-v1'));

COMMIT;
