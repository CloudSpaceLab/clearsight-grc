BEGIN;
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM third_party_relationship_create_receipts) THEN
        RAISE EXCEPTION 'vendor creation receipts must be preserved for safe retries';
    END IF;
END $$;
DROP TABLE third_party_relationship_create_receipts;
COMMIT;
