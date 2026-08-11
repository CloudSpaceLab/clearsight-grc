BEGIN;

ALTER TABLE governance_decisions
    DROP CONSTRAINT IF EXISTS governance_decisions_object_type_check;
ALTER TABLE governance_decisions
    ADD CONSTRAINT governance_decisions_object_type_check
    CHECK (object_type IN ('ROUTING_POLICY','DELEGATION','SEGREGATION_RULE'));

DROP INDEX IF EXISTS directory_group_role_bindings_current_unique_idx;

COMMIT;
