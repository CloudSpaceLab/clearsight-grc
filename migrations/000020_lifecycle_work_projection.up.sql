BEGIN;

CREATE UNIQUE INDEX workflow_instances_matter_response_subject_idx
    ON workflow_instances(tenant_id, kind, subject_type, subject_id)
    WHERE kind='MATTER_RESPONSE';

COMMIT;
