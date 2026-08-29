BEGIN;

ALTER TABLE document_imports
    DROP CONSTRAINT document_imports_extraction_status_check;

ALTER TABLE document_imports
    ADD CONSTRAINT document_imports_extraction_status_check
    CHECK (extraction_status IN ('PENDING','EXTRACTED','PARTIAL','TRUNCATED','UNSUPPORTED','FAILED'));

COMMIT;
