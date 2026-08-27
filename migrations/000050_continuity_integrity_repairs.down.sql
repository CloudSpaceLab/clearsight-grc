BEGIN;
ALTER TABLE continuity_events
    DROP CONSTRAINT continuity_events_aggregate_type_check;
ALTER TABLE continuity_events
    ADD CONSTRAINT continuity_events_aggregate_type_check
    CHECK (aggregate_type IN ('PROGRAM','MATTER','PROGRAM_STATE'));

DROP INDEX IF EXISTS matters_open_trigger_idx;
CREATE UNIQUE INDEX matters_open_trigger_idx
    ON matters(tenant_id,trigger_key)
    WHERE trigger_key<>'' AND status NOT IN ('CLOSED','CANCELLED');
COMMIT;
