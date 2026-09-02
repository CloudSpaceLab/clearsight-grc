BEGIN;

-- Exact Automation Policy reads expose lifecycle timestamps. Add them in a
-- follow-up migration so environments that already applied migration 65 retain
-- checksum continuity.
ALTER TABLE automation_policies
    ADD COLUMN created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    ADD CONSTRAINT automation_policies_updated_after_created_ck CHECK (updated_at >= created_at);

COMMIT;
