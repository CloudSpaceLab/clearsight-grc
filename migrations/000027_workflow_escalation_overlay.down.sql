BEGIN;

DROP TRIGGER IF EXISTS workflow_tasks_preserve_escalation_trg ON workflow_tasks;
DROP FUNCTION IF EXISTS preserve_active_workflow_escalation();

COMMIT;
