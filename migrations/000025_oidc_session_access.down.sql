BEGIN;

DROP TABLE IF EXISTS web_sessions;
DROP TABLE IF EXISTS principal_identities;

ALTER TABLE principals
    DROP CONSTRAINT IF EXISTS principals_tenant_identity_unique;

COMMIT;
