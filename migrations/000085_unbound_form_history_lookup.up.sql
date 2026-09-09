BEGIN;

-- Recover an exact unbound form through draft, pending, active and paused history.
-- Existing current-only indexes cannot locate an interrupted draft creation.
CREATE INDEX monitoring_form_templates_unbound_history_idx
    ON monitoring_form_templates(tenant_id,legal_entity_id,code,version,id)
    WHERE legal_entity_id IS NOT NULL AND program_id IS NULL;

COMMIT;
