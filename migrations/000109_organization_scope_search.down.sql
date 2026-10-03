BEGIN;

DROP INDEX IF EXISTS organization_scopes_search_idx;
ALTER TABLE organization_scopes DROP COLUMN IF EXISTS search_document;

COMMIT;
