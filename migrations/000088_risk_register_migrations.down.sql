BEGIN;
DO $$ BEGIN IF EXISTS(SELECT 1 FROM risk_register_migrations) THEN RAISE EXCEPTION 'Retain risk register migration history before downgrade'; END IF; END $$;
DROP TABLE risk_register_migration_revisions;
DROP TABLE risk_register_migrations;
DROP FUNCTION protect_risk_register_revision();
COMMIT;
