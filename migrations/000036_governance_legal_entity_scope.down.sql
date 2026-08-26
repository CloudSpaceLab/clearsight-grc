BEGIN;

DO $$
DECLARE collisions integer;
BEGIN
    SELECT count(*) INTO collisions FROM (
        SELECT tenant_id,code FROM routing_policies GROUP BY tenant_id,code HAVING count(*)>1
    ) duplicate_codes;
    IF collisions > 0 THEN
        RAISE EXCEPTION 'cannot remove governance legal-entity scope: % tenant-wide routing policy code collisions', collisions;
    END IF;
END $$;

DROP TRIGGER IF EXISTS delegations_legal_entity_immutable ON delegations;
DROP TRIGGER IF EXISTS routing_policy_versions_legal_entity_immutable ON routing_policy_versions;
DROP TRIGGER IF EXISTS routing_policies_legal_entity_immutable ON routing_policies;
DROP TRIGGER IF EXISTS delegations_entity_payload ON delegations;
DROP TRIGGER IF EXISTS routing_policy_versions_entity_payload ON routing_policy_versions;
DROP FUNCTION IF EXISTS validate_governance_entity_payload();
DROP FUNCTION IF EXISTS prevent_governance_legal_entity_change();

DROP INDEX IF EXISTS delegations_entity_resolution_idx;
DROP INDEX IF EXISTS delegations_entity_inventory_idx;
DROP INDEX IF EXISTS routing_policies_entity_inventory_idx;
DROP INDEX IF EXISTS routing_policies_entity_code_uidx;

ALTER TABLE routing_policy_versions DROP CONSTRAINT IF EXISTS routing_policy_versions_policy_entity_fk;
ALTER TABLE delegations DROP CONSTRAINT IF EXISTS delegations_tenant_legal_entity_fk;
ALTER TABLE routing_policies DROP CONSTRAINT IF EXISTS routing_policies_tenant_legal_entity_fk;
ALTER TABLE routing_policies DROP CONSTRAINT IF EXISTS routing_policies_id_legal_entity_key;

ALTER TABLE routing_policy_versions DROP COLUMN IF EXISTS legal_entity_id;
ALTER TABLE delegations DROP COLUMN IF EXISTS legal_entity_id;
ALTER TABLE routing_policies DROP COLUMN IF EXISTS legal_entity_id;

ALTER TABLE routing_policies ADD CONSTRAINT routing_policies_tenant_id_code_key UNIQUE(tenant_id,code);

COMMIT;
