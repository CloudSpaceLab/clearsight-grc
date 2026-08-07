BEGIN;

DROP INDEX IF EXISTS outbox_dead_letter_idx;
DROP INDEX IF EXISTS outbox_retry_claim_idx;
UPDATE outbox_events
SET next_attempt_at=clock_timestamp()
WHERE dead_lettered_at IS NOT NULL AND published_at IS NULL;
ALTER TABLE outbox_events DROP COLUMN IF EXISTS dead_lettered_at;
CREATE INDEX outbox_retry_claim_idx
    ON outbox_events(COALESCE(next_attempt_at, available_at), id)
    WHERE published_at IS NULL;

DROP INDEX IF EXISTS workflow_timers_failed_idx;
UPDATE workflow_timers
SET state='CANCELLED', failed_at=NULL
WHERE state='FAILED';
ALTER TABLE workflow_timers DROP CONSTRAINT IF EXISTS workflow_timers_state_check;
ALTER TABLE workflow_timers DROP COLUMN IF EXISTS failed_at;
ALTER TABLE workflow_timers
    ADD CONSTRAINT workflow_timers_state_check
        CHECK (state IN ('READY','CLAIMED','FIRED','CANCELLED'));

COMMIT;
