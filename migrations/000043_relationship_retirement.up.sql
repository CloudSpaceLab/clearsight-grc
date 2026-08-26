ALTER TABLE requirement_control_links
    ADD COLUMN retired_at timestamptz,
    ADD COLUMN retired_by uuid,
    ADD COLUMN retirement_reason text NOT NULL DEFAULT '',
    ADD CONSTRAINT requirement_control_links_retired_by_tenant_fk FOREIGN KEY (retired_by,tenant_id) REFERENCES principals(id,tenant_id),
    ADD CONSTRAINT requirement_control_links_retirement_check CHECK (
        (retired_at IS NULL AND retired_by IS NULL AND retirement_reason = '') OR
        (retired_at IS NOT NULL AND retired_by IS NOT NULL AND retirement_reason <> '')
    );

DO $$
DECLARE current_unique_constraint text;
BEGIN
    SELECT conname INTO current_unique_constraint
    FROM pg_constraint
    WHERE conrelid='requirement_control_links'::regclass
      AND contype='u'
      AND pg_get_constraintdef(oid)='UNIQUE (tenant_id, program_id, requirement_id, implementation_id)';
    IF current_unique_constraint IS NULL THEN
        RAISE EXCEPTION 'requirement-control link uniqueness constraint is missing';
    END IF;
    EXECUTE format('ALTER TABLE requirement_control_links DROP CONSTRAINT %I', current_unique_constraint);
END $$;
DROP INDEX requirement_control_links_program_idx;
CREATE UNIQUE INDEX requirement_control_links_current_unique_idx
    ON requirement_control_links(tenant_id,program_id,requirement_id,implementation_id)
    WHERE retired_at IS NULL;
CREATE INDEX requirement_control_links_program_idx
    ON requirement_control_links(tenant_id,program_id,requirement_id,implementation_id)
    WHERE retired_at IS NULL;

ALTER TABLE matter_links
    ADD COLUMN retired_at timestamptz,
    ADD COLUMN retired_by uuid,
    ADD COLUMN retirement_reason text NOT NULL DEFAULT '',
    ADD CONSTRAINT matter_links_retired_by_tenant_fk FOREIGN KEY (retired_by,tenant_id) REFERENCES principals(id,tenant_id),
    ADD CONSTRAINT matter_links_retirement_check CHECK (
        (retired_at IS NULL AND retired_by IS NULL AND retirement_reason = '') OR
        (retired_at IS NOT NULL AND retired_by IS NOT NULL AND retirement_reason <> '')
    );

DROP INDEX matter_links_unique_idx;
DROP INDEX matter_links_program_idx;
CREATE UNIQUE INDEX matter_links_current_unique_idx
    ON matter_links(
        tenant_id,
        matter_id,
        COALESCE(program_id,'00000000-0000-0000-0000-000000000000'::uuid),
        COALESCE(requirement_id,'00000000-0000-0000-0000-000000000000'::uuid),
        COALESCE(control_id,'00000000-0000-0000-0000-000000000000'::uuid),
        relationship
    )
    WHERE retired_at IS NULL;
CREATE INDEX matter_links_program_idx
    ON matter_links(tenant_id,program_id,matter_id)
    WHERE program_id IS NOT NULL AND retired_at IS NULL;
CREATE INDEX matter_links_matter_current_idx
    ON matter_links(tenant_id,matter_id,created_at,id)
    WHERE retired_at IS NULL;
