BEGIN;

ALTER TABLE organization_position_revisions
    ADD COLUMN effective_from timestamptz,
    ADD COLUMN activation_attempts integer NOT NULL DEFAULT 0 CHECK (activation_attempts>=0),
    ADD COLUMN activation_failed_at timestamptz,
    ADD COLUMN activation_error_code text NOT NULL DEFAULT '';

ALTER TABLE organization_position_revisions
    DROP CONSTRAINT IF EXISTS organization_position_revisions_status_check;

ALTER TABLE organization_position_revisions
    ADD CONSTRAINT organization_position_revisions_status_check
    CHECK (status IN ('PENDING','SCHEDULED','APPLIED','REJECTED','FAILED'));

DROP INDEX IF EXISTS organization_position_revisions_pending_idx;
CREATE UNIQUE INDEX organization_position_revisions_pending_idx
    ON organization_position_revisions(tenant_id,legal_entity_id,position_id)
    WHERE status IN ('PENDING','SCHEDULED');

CREATE INDEX organization_position_revisions_activation_idx
    ON organization_position_revisions(effective_from,id)
    WHERE status='SCHEDULED';

COMMIT;
