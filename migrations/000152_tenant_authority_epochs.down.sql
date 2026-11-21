BEGIN;

DROP TRIGGER tenant_authority_epoch_after_insert ON tenants;
DROP FUNCTION create_tenant_authority_epoch();
DROP TABLE tenant_authority_epochs;

COMMIT;
