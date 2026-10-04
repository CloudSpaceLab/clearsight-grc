BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM organization_position_revisions
        WHERE status IN ('SCHEDULED','FAILED')
           OR effective_from IS NOT NULL
           OR activation_attempts<>0
           OR activation_failed_at IS NOT NULL
           OR activation_error_code<>''
    ) THEN
        RAISE EXCEPTION 'organization position activation history exists; refusing to erase effective activation state';
    END IF;
END;
$$;

DROP INDEX IF EXISTS organization_position_revisions_activation_idx;
DROP INDEX IF EXISTS organization_position_revisions_pending_idx;
CREATE UNIQUE INDEX organization_position_revisions_pending_idx
    ON organization_position_revisions(tenant_id,legal_entity_id,position_id)
    WHERE status='PENDING';

ALTER TABLE organization_position_revisions
    DROP CONSTRAINT IF EXISTS organization_position_revisions_status_check;

ALTER TABLE organization_position_revisions
    ADD CONSTRAINT organization_position_revisions_status_check
    CHECK (status IN ('PENDING','APPLIED','REJECTED'));

ALTER TABLE organization_position_revisions
    DROP COLUMN IF EXISTS activation_error_code,
    DROP COLUMN IF EXISTS activation_failed_at,
    DROP COLUMN IF EXISTS activation_attempts,
    DROP COLUMN IF EXISTS effective_from;

COMMIT;
