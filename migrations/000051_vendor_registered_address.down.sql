BEGIN;

ALTER TABLE third_parties
    DROP CONSTRAINT IF EXISTS third_parties_registered_address_length_check,
    DROP COLUMN IF EXISTS registered_address;

COMMIT;
