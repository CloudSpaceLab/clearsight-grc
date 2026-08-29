BEGIN;

UPDATE document_imports
SET extraction_status = 'EXTRACTED'
WHERE extraction_status IN ('PARTIAL','TRUNCATED');

ALTER TABLE document_imports
    DROP CONSTRAINT document_imports_extraction_status_check;

ALTER TABLE document_imports
    ADD CONSTRAINT document_imports_extraction_status_check
    CHECK (extraction_status IN ('PENDING','EXTRACTED','UNSUPPORTED','FAILED'));

COMMIT;
