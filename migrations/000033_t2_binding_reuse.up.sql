BEGIN;

ALTER TABLE capture_requests
    ADD COLUMN source_bindings jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD CONSTRAINT capture_requests_source_bindings_array CHECK (jsonb_typeof(source_bindings) = 'array');

ALTER TABLE capture_submissions
    ADD COLUMN answer_provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD CONSTRAINT capture_submissions_answer_provenance_object CHECK (jsonb_typeof(answer_provenance) = 'object');

ALTER TABLE workflow_tasks
    ADD COLUMN source_bindings jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD CONSTRAINT workflow_tasks_source_bindings_array CHECK (jsonb_typeof(source_bindings) = 'array');

COMMIT;
