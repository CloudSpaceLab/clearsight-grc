BEGIN;

CREATE FUNCTION organization_scope_search_path(path text[]) RETURNS text
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
AS $scope$
    SELECT COALESCE(string_agg(value, ' ' ORDER BY ordinality), '')
    FROM unnest(path) WITH ORDINALITY AS item(value, ordinality);
$scope$;

ALTER TABLE organization_scopes
    ADD COLUMN search_document tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple'::regconfig, code), 'A') ||
        setweight(to_tsvector('simple'::regconfig, name), 'A') ||
        setweight(to_tsvector('simple'::regconfig, organization_scope_search_path(department_path)), 'B')
    ) STORED;

CREATE INDEX organization_scopes_search_idx
    ON organization_scopes USING gin(search_document)
    WHERE status='ACTIVE' AND valid_until IS NULL;

COMMIT;
