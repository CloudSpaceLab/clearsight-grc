BEGIN;

CREATE INDEX monitoring_form_templates_reusable_idx
    ON monitoring_form_templates(tenant_id,legal_entity_id,code,version DESC,id)
    WHERE legal_entity_id IS NOT NULL AND status='ACTIVE' AND is_current;

COMMIT;
