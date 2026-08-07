BEGIN;

DROP TRIGGER IF EXISTS continuity_lifecycle_actor_trg ON continuity_events;
DROP FUNCTION IF EXISTS sync_continuity_lifecycle_actor();

UPDATE matter_decisions SET status='PROPOSED' WHERE status='IN_REVIEW';
UPDATE matter_decisions SET status='RETURNED' WHERE status='CHALLENGED';
ALTER TABLE matter_decisions DROP CONSTRAINT IF EXISTS matter_decisions_status_check;
ALTER TABLE matter_decisions ADD CONSTRAINT matter_decisions_status_check CHECK (
    status IN ('PROPOSED','APPROVED','CONDITIONALLY_APPROVED','REJECTED','RETURNED','EXPIRED','SUPERSEDED')
);

ALTER TABLE matter_decisions
    DROP CONSTRAINT IF EXISTS matter_decisions_proposed_by_tenant_fk,
    DROP CONSTRAINT IF EXISTS matter_decisions_reviewed_by_tenant_fk,
    DROP CONSTRAINT IF EXISTS matter_decisions_challenged_by_tenant_fk,
    DROP COLUMN IF EXISTS proposed_by,
    DROP COLUMN IF EXISTS reviewed_by,
    DROP COLUMN IF EXISTS challenged_by;

ALTER TABLE response_packages
    DROP CONSTRAINT IF EXISTS response_packages_prepared_by_tenant_fk,
    DROP CONSTRAINT IF EXISTS response_packages_reviewed_by_tenant_fk,
    DROP CONSTRAINT IF EXISTS response_packages_rejected_by_tenant_fk,
    DROP CONSTRAINT IF EXISTS response_packages_withdrawn_by_tenant_fk,
    DROP CONSTRAINT IF EXISTS response_packages_transmitted_by_tenant_fk,
    DROP CONSTRAINT IF EXISTS response_packages_acknowledged_by_tenant_fk,
    DROP COLUMN IF EXISTS prepared_by,
    DROP COLUMN IF EXISTS reviewed_by,
    DROP COLUMN IF EXISTS rejected_by,
    DROP COLUMN IF EXISTS withdrawn_by,
    DROP COLUMN IF EXISTS transmitted_by,
    DROP COLUMN IF EXISTS acknowledged_by;

COMMIT;
