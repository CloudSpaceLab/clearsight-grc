BEGIN;
DROP TABLE staff_assignment_notification_deliveries;
ALTER TABLE outbox_events DROP CONSTRAINT outbox_events_id_tenant_key;
COMMIT;
