DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM matter_links WHERE retired_at IS NOT NULL) OR
       EXISTS (SELECT 1 FROM requirement_control_links WHERE retired_at IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot roll back relationship retirement while retired material links exist';
    END IF;
END $$;

DROP INDEX matter_links_matter_current_idx;
DROP INDEX matter_links_program_idx;
DROP INDEX matter_links_current_unique_idx;
ALTER TABLE matter_links DROP CONSTRAINT matter_links_retirement_check;
ALTER TABLE matter_links
    DROP COLUMN retirement_reason,
    DROP COLUMN retired_by,
    DROP COLUMN retired_at;
CREATE UNIQUE INDEX matter_links_unique_idx
    ON matter_links(
        tenant_id,
        matter_id,
        COALESCE(program_id,'00000000-0000-0000-0000-000000000000'::uuid),
        COALESCE(requirement_id,'00000000-0000-0000-0000-000000000000'::uuid),
        COALESCE(control_id,'00000000-0000-0000-0000-000000000000'::uuid),
        relationship
    );
CREATE INDEX matter_links_program_idx
    ON matter_links(tenant_id,program_id,matter_id)
    WHERE program_id IS NOT NULL;

DROP INDEX requirement_control_links_program_idx;
DROP INDEX requirement_control_links_current_unique_idx;
ALTER TABLE requirement_control_links DROP CONSTRAINT requirement_control_links_retirement_check;
ALTER TABLE requirement_control_links
    DROP COLUMN retirement_reason,
    DROP COLUMN retired_by,
    DROP COLUMN retired_at;
ALTER TABLE requirement_control_links
    ADD CONSTRAINT requirement_control_links_scope_key
    UNIQUE (tenant_id,program_id,requirement_id,implementation_id);
CREATE INDEX requirement_control_links_program_idx
    ON requirement_control_links(tenant_id,program_id,requirement_id);
