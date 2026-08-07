BEGIN;

DROP INDEX IF EXISTS workflow_instances_matter_action_subject_idx;
DROP INDEX IF EXISTS response_packages_current_order_idx;
ALTER TABLE response_packages DROP COLUMN IF EXISTS matter_version;
DROP INDEX IF EXISTS matter_decisions_current_order_idx;
ALTER TABLE matter_decisions DROP COLUMN IF EXISTS matter_version;

COMMIT;
