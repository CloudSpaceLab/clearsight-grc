BEGIN;
CREATE INDEX program_activity_schedule_event_idx ON outbox_events(tenant_id,aggregate_id,occurred_at DESC,id DESC) WHERE aggregate_type='PROGRAM_ASSESSMENT_SCHEDULE';
CREATE INDEX program_activity_review_request_idx ON capture_submission_reviews(tenant_id,request_id,created_at DESC,id DESC);
COMMIT;
