BEGIN;

-- 000061 created the notification_kind CHECK inline, so its name is
-- server-generated and version-dependent: PostgreSQL 18 preserves the
-- `_check` suffix while truncating earlier characters (yielding
-- staff_assignment_notification_deliverie_notification_kind_check),
-- while older releases truncate from the end. Resolve the actual name
-- from the catalog instead of hard-coding one.
DO $$
DECLARE
    con_name text;
BEGIN
    SELECT conname INTO con_name
    FROM pg_constraint
    WHERE conrelid = 'staff_assignment_notification_deliveries'::regclass
      AND contype = 'c'
      AND pg_get_constraintdef(oid) LIKE '%notification_kind%';
    IF con_name IS NULL THEN
        RAISE EXCEPTION 'notification_kind CHECK constraint on staff_assignment_notification_deliveries not found';
    END IF;
    EXECUTE format('ALTER TABLE staff_assignment_notification_deliveries DROP CONSTRAINT %I', con_name);
END
$$;

ALTER TABLE staff_assignment_notification_deliveries
    ADD CONSTRAINT staff_assignment_notification_deliveries_notification_kind_ck
    CHECK (notification_kind IN ('MATTER_OWNER_ASSIGNED','ACTION_PERFORMER_ASSIGNED','ACTION_UPDATE_REQUESTED'));

COMMIT;