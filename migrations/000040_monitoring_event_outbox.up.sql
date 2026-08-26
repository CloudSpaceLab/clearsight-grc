BEGIN;

ALTER TABLE monitoring_form_templates
    ADD COLUMN legal_entity_id uuid,
    ADD COLUMN program_id uuid;

WITH exact_scope AS (
    SELECT mc.tenant_id,mc.form_template_id,mc.form_template_version,
           min(mc.program_id::text)::uuid AS program_id
    FROM monitoring_checks mc
    WHERE mc.form_template_id IS NOT NULL
    GROUP BY mc.tenant_id,mc.form_template_id,mc.form_template_version
    HAVING count(DISTINCT mc.program_id)=1
)
UPDATE monitoring_form_templates f
SET program_id=scope.program_id,
    legal_entity_id=p.legal_entity_id
FROM exact_scope scope
JOIN programs p ON p.tenant_id=scope.tenant_id AND p.id=scope.program_id
WHERE f.tenant_id=scope.tenant_id
  AND f.id=scope.form_template_id
  AND f.version=scope.form_template_version;

ALTER TABLE programs
    ADD CONSTRAINT programs_id_tenant_entity_key UNIQUE(id,tenant_id,legal_entity_id);

ALTER TABLE monitoring_form_templates
    ADD CONSTRAINT monitoring_form_templates_scope_pair_ck
        CHECK ((legal_entity_id IS NULL AND program_id IS NULL) OR (legal_entity_id IS NOT NULL AND program_id IS NOT NULL)),
    ADD CONSTRAINT monitoring_form_templates_program_entity_fk
        FOREIGN KEY(program_id,tenant_id,legal_entity_id) REFERENCES programs(id,tenant_id,legal_entity_id),
    ADD CONSTRAINT monitoring_form_templates_scope_version_key
        UNIQUE(tenant_id,id,version,program_id);

DROP INDEX monitoring_form_templates_current_code_idx;
CREATE UNIQUE INDEX monitoring_form_templates_current_code_idx
    ON monitoring_form_templates(tenant_id,program_id,code)
    WHERE is_current AND program_id IS NOT NULL;
CREATE UNIQUE INDEX monitoring_form_templates_legacy_current_code_idx
    ON monitoring_form_templates(tenant_id,code)
    WHERE is_current AND program_id IS NULL;
CREATE INDEX monitoring_form_templates_program_idx
    ON monitoring_form_templates(tenant_id,legal_entity_id,program_id,code,version DESC,id)
    WHERE program_id IS NOT NULL;

ALTER TABLE monitoring_checks
    ADD CONSTRAINT monitoring_checks_form_program_fk
        FOREIGN KEY(tenant_id,form_template_id,form_template_version,program_id)
        REFERENCES monitoring_form_templates(tenant_id,id,version,program_id)
        NOT VALID;

CREATE TABLE monitoring_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    aggregate_type text NOT NULL CHECK (aggregate_type IN ('MONITORING_FORM','MONITORING_CHECK','MONITORING_RESULT')),
    aggregate_id uuid NOT NULL,
    aggregate_version bigint NOT NULL CHECK (aggregate_version > 0),
    event_type text NOT NULL,
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload)='object'),
    actor_id uuid,
    occurred_at timestamptz NOT NULL,
    UNIQUE(tenant_id,aggregate_type,aggregate_id,aggregate_version),
    CONSTRAINT monitoring_events_actor_tenant_fk FOREIGN KEY(actor_id,tenant_id) REFERENCES principals(id,tenant_id)
);

CREATE INDEX monitoring_events_history_idx
    ON monitoring_events(tenant_id,aggregate_type,aggregate_id,aggregate_version);

CREATE UNIQUE INDEX monitoring_outbox_event_uq
    ON outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,(COALESCE(payload->>'version','1')))
    WHERE aggregate_type IN ('MONITORING_FORM','MONITORING_CHECK','MONITORING_RESULT');

COMMIT;
