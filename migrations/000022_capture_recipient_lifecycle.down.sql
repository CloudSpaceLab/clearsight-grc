BEGIN;

DROP TABLE IF EXISTS capture_recipient_history;

ALTER TABLE capture_sessions
    DROP COLUMN IF EXISTS recipient_revision;

ALTER TABLE capture_invitations
    DROP COLUMN IF EXISTS recipient_revision;

ALTER TABLE capture_requests
    DROP CONSTRAINT IF EXISTS capture_requests_id_tenant_unique,
    DROP CONSTRAINT IF EXISTS capture_requests_recipient_state_check,
    DROP COLUMN IF EXISTS recipient_issue_reason,
    DROP COLUMN IF EXISTS recipient_revision,
    DROP COLUMN IF EXISTS recipient_state;

COMMIT;
