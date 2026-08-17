BEGIN;
DROP TABLE IF EXISTS ai_workloads;
DROP INDEX IF EXISTS automation_policies_ai_active_idx;
ALTER TABLE automation_policies DROP CONSTRAINT IF EXISTS automation_policies_id_tenant_unique;
ALTER TABLE automation_policies
    DROP CONSTRAINT IF EXISTS automation_policies_checker_tenant_fk,
    DROP CONSTRAINT IF EXISTS automation_policies_maker_tenant_fk,
    DROP COLUMN IF EXISTS record_version,
    DROP COLUMN IF EXISTS retired_at,
    DROP COLUMN IF EXISTS suspended_at,
    DROP COLUMN IF EXISTS activated_at,
    DROP COLUMN IF EXISTS approved_at,
    DROP COLUMN IF EXISTS submitted_at,
    DROP COLUMN IF EXISTS checksum,
    DROP COLUMN IF EXISTS checker_id,
    DROP COLUMN IF EXISTS maker_id,
    DROP COLUMN IF EXISTS rollout_mode,
    DROP COLUMN IF EXISTS ai_definition;
ALTER TABLE automation_policies DROP CONSTRAINT IF EXISTS automation_policies_status_check;
ALTER TABLE automation_policies ADD CONSTRAINT automation_policies_status_check CHECK(status IN ('DRAFT','ACTIVE','SUSPENDED','EXPIRED','RETIRED'));
COMMIT;
