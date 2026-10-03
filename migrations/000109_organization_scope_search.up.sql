BEGIN;

ALTER TABLE organization_scopes
    ADD COLUMN search_document tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple'::regconfig, code), 'A') ||
        setweight(to_tsvector('simple'::regconfig, name), 'A') ||
        setweight(to_tsvector('simple'::regconfig, array_to_string(department_path, ' ')), 'B')
    ) STORED;

CREATE INDEX organization_scopes_search_idx
    ON organization_scopes USING gin(search_document)
    WHERE status='ACTIVE' AND valid_until IS NULL;

COMMIT;
