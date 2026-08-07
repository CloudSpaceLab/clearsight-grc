BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM workflow_instances
        GROUP BY tenant_id, kind, subject_type, subject_id
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'workflow_instances contains duplicate projection identities';
    END IF;
END $$;

CREATE UNIQUE INDEX workflow_instance_projection_identity_idx
    ON workflow_instances(tenant_id, kind, subject_type, subject_id);

COMMIT;
