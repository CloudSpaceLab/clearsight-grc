BEGIN;
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM demo_record_archives) THEN
        RAISE EXCEPTION 'Cannot remove retained demo archive history';
    END IF;
END $$;
DROP TABLE demo_record_archives;
DROP FUNCTION protect_demo_record_archive_history();
ALTER TABLE third_party_relationships DROP CONSTRAINT third_party_relationships_archive_scope_unique;
ALTER TABLE matters DROP CONSTRAINT matters_archive_scope_unique;
COMMIT;
