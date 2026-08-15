BEGIN;
ALTER TABLE document_imports
  ADD COLUMN tabular_metadata jsonb NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE document_imports
  ADD CONSTRAINT document_imports_tabular_metadata_ck CHECK (
    jsonb_typeof(tabular_metadata)='object'
    AND octet_length(tabular_metadata::text) <= 2097152
  );
COMMIT;
