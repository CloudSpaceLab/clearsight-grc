BEGIN;

ALTER TABLE workflow_tasks
    DROP CONSTRAINT IF EXISTS workflow_tasks_source_bindings_array,
    DROP COLUMN IF EXISTS source_bindings;

ALTER TABLE capture_submissions
    DROP CONSTRAINT IF EXISTS capture_submissions_answer_provenance_object,
    DROP COLUMN IF EXISTS answer_provenance;

ALTER TABLE capture_requests
    DROP CONSTRAINT IF EXISTS capture_requests_source_bindings_array,
    DROP COLUMN IF EXISTS source_bindings;

COMMIT;
