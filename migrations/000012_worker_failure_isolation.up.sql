BEGIN;

ALTER TABLE workflow_timers DROP CONSTRAINT IF EXISTS workflow_timers_state_check;
ALTER TABLE workflow_timers
    ADD COLUMN IF NOT EXISTS failed_at timestamptz,
    ADD CONSTRAINT workflow_timers_state_check
        CHECK (state IN ('READY','CLAIMED','FIRED','CANCELLED','FAILED'));
CREATE INDEX IF NOT EXISTS workflow_timers_failed_idx
    ON workflow_timers(failed_at DESC, id) WHERE state='FAILED';

ALTER TABLE outbox_events
    ADD COLUMN IF NOT EXISTS dead_lettered_at timestamptz;
DROP INDEX IF EXISTS outbox_retry_claim_idx;
CREATE INDEX outbox_retry_claim_idx
    ON outbox_events(COALESCE(next_attempt_at, available_at), id)
    WHERE published_at IS NULL AND dead_lettered_at IS NULL;
CREATE INDEX IF NOT EXISTS outbox_dead_letter_idx
    ON outbox_events(dead_lettered_at DESC, id) WHERE dead_lettered_at IS NOT NULL;

COMMIT;
