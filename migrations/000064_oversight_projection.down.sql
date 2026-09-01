BEGIN;

UPDATE role_templates
SET capabilities = array_remove(capabilities,'OVERSIGHT_READ')
WHERE valid_until IS NULL AND code IN ('CRO','CCO','CISO','EXECUTIVE','GRC_ADMIN');

DROP TABLE IF EXISTS oversight_snapshots;

COMMIT;
