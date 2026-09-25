BEGIN;
ALTER TABLE staff_assignment_notification_deliveries
    DROP CONSTRAINT staff_assignment_notification_deliveries_notification_kind_ck;
ALTER TABLE staff_assignment_notification_deliveries
    ADD CONSTRAINT staff_assignment_notification_deliveries_notification_kind_ck
    CHECK (notification_kind IN ('MATTER_OWNER_ASSIGNED','ACTION_PERFORMER_ASSIGNED','ACTION_UPDATE_REQUESTED','MATTER_COMMENT_MENTIONED'));
COMMIT;
