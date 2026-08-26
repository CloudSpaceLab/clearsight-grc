BEGIN;
DO $$
DECLARE program_code_collisions bigint;
DECLARE trigger_key_collisions bigint;
DECLARE matter_trigger_collisions bigint;
BEGIN
    SELECT count(*) INTO program_code_collisions FROM (
        SELECT tenant_id,code FROM programs WHERE status<>'RETIRED'
        GROUP BY tenant_id,code HAVING count(*)>1
    ) collisions;
    SELECT count(*) INTO trigger_key_collisions FROM (
        SELECT tenant_id,dedupe_key FROM program_trigger_events
        GROUP BY tenant_id,dedupe_key HAVING count(*)>1
    ) collisions;
    SELECT count(*) INTO matter_trigger_collisions FROM (
        SELECT tenant_id,trigger_key FROM matters
        WHERE trigger_key<>'' AND status NOT IN ('CLOSED','CANCELLED')
        GROUP BY tenant_id,trigger_key HAVING count(*)>1
    ) collisions;
    IF program_code_collisions+trigger_key_collisions+matter_trigger_collisions>0 THEN
        RAISE EXCEPTION 'migration 000035 is irreversible while tenant-wide uniqueness collisions exist: program_codes=%, trigger_events=%, open_matters=%',
            program_code_collisions,trigger_key_collisions,matter_trigger_collisions;
    END IF;
END $$;
DROP TRIGGER IF EXISTS programs_legal_entity_immutable ON programs;
DROP TRIGGER IF EXISTS matters_legal_entity_immutable ON matters;
DROP FUNCTION IF EXISTS prevent_legal_entity_change();
DROP TRIGGER IF EXISTS matter_links_legal_entity_match ON matter_links;
DROP FUNCTION IF EXISTS enforce_matter_program_legal_entity();
DROP INDEX IF EXISTS matters_open_trigger_idx;
CREATE UNIQUE INDEX matters_open_trigger_idx ON matters(tenant_id,trigger_key) WHERE trigger_key<>'' AND status NOT IN ('CLOSED','CANCELLED');
ALTER TABLE program_trigger_events DROP CONSTRAINT IF EXISTS program_trigger_events_program_dedupe_key;
ALTER TABLE program_trigger_events ADD CONSTRAINT program_trigger_events_tenant_id_dedupe_key_key UNIQUE (tenant_id,dedupe_key);
DROP INDEX IF EXISTS programs_entity_summary_idx;
DROP INDEX IF EXISTS matters_entity_summary_idx;
DROP INDEX IF EXISTS matters_entity_open_summary_idx;
DROP INDEX IF EXISTS matters_entity_status_summary_idx;
DROP INDEX IF EXISTS matters_entity_queue_idx;
ALTER TABLE matters DROP CONSTRAINT IF EXISTS matters_legal_entity_tenant_fk;
ALTER TABLE matters DROP COLUMN IF EXISTS legal_entity_id;
DROP INDEX IF EXISTS programs_active_code_idx;
CREATE UNIQUE INDEX programs_active_code_idx ON programs(tenant_id,code) WHERE status<>'RETIRED';
ALTER TABLE programs ALTER COLUMN legal_entity_id DROP NOT NULL;
COMMIT;
