BEGIN;

-- Refuse rollback while any larger capture exists; never truncate source data.
ALTER TABLE capture_requests ADD CONSTRAINT capture_requests_fields_check1
    CHECK (jsonb_array_length(fields) BETWEEN 1 AND 50);

COMMIT;
