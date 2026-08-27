BEGIN;

ALTER TABLE third_parties
    ADD COLUMN registered_address text,
    ADD CONSTRAINT third_parties_registered_address_length_check CHECK (
        registered_address IS NULL OR char_length(registered_address) <= 2000
    );

COMMIT;
