BEGIN;
ALTER TABLE document_imports DROP CONSTRAINT IF EXISTS document_imports_tabular_metadata_ck;
ALTER TABLE document_imports DROP COLUMN IF EXISTS tabular_metadata;
COMMIT;
