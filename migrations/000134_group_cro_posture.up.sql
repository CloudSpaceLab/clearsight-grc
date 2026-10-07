BEGIN;

ALTER TABLE group_oversight_child_facts
    ADD COLUMN domain_state text NOT NULL DEFAULT 'MISSING'
        CHECK (domain_state IN ('AVAILABLE','STALE','MISSING')),
    ADD COLUMN domain_source_id uuid,
    ADD COLUMN domain_generated_at timestamptz,
    ADD COLUMN domain_definition_revision text,
    ADD COLUMN domain_posture jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN domain_source_high_water jsonb NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE group_oversight_child_facts
    ADD CONSTRAINT group_child_domain_posture_json_check
        CHECK (jsonb_typeof(domain_posture)='object'),
    ADD CONSTRAINT group_child_domain_high_water_json_check
        CHECK (jsonb_typeof(domain_source_high_water)='object'),
    ADD CONSTRAINT group_child_domain_source_check
        CHECK (
            (domain_state='MISSING'
                AND domain_source_id IS NULL
                AND domain_generated_at IS NULL
                AND domain_definition_revision IS NULL)
            OR
            (domain_state IN ('AVAILABLE','STALE')
                AND domain_source_id IS NOT NULL
                AND domain_generated_at IS NOT NULL
                AND domain_definition_revision IS NOT NULL)
        );

CREATE INDEX group_oversight_child_domain_source_idx
    ON group_oversight_child_facts(
        tenant_id,legal_entity_id,domain_generated_at DESC
    )
    WHERE domain_source_id IS NOT NULL;

COMMIT;
