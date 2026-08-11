BEGIN;

CREATE OR REPLACE FUNCTION preserve_active_workflow_escalation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.status = 'ESCALATED'
       AND COALESCE(OLD.context->>'escalation_active','') = 'true'
       AND COALESCE(NEW.context->>'escalation_active','') <> 'true'
       AND COALESCE(OLD.context->>'work_requirement_key','') = COALESCE(NEW.context->>'work_requirement_key','')
       AND OLD.due_at IS NOT DISTINCT FROM NEW.due_at
       AND COALESCE(OLD.context->>'escalation_policy_version','') = COALESCE(NEW.context->>'authority_policy_version','')
       AND COALESCE(OLD.context->>'decision_type','') = COALESCE(NEW.context->>'decision_type','')
       AND COALESCE(OLD.context->>'materiality','') = COALESCE(NEW.context->>'materiality','')
       AND COALESCE(OLD.context->>'command_name','') = COALESCE(NEW.context->>'command_name','')
       AND COALESCE(OLD.context->>'target_status','') = COALESCE(NEW.context->>'target_status','')
       AND COALESCE(OLD.context->>'allowed_targets','') = COALESCE(NEW.context->>'allowed_targets','')
       AND COALESCE(OLD.context->>'sequence_policy_version','') = COALESCE(NEW.context->>'sequence_policy_version','')
    THEN
        NEW.responsibility := OLD.responsibility;
        NEW.principal_id := OLD.principal_id;
        NEW.status := OLD.status;
        NEW.context := OLD.context || NEW.context;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER workflow_tasks_preserve_escalation_trg
BEFORE UPDATE ON workflow_tasks
FOR EACH ROW EXECUTE FUNCTION preserve_active_workflow_escalation();

COMMIT;
