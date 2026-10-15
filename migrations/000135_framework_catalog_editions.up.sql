BEGIN;

-- Platform reference content only. Tenant Program adoption and applicable
-- obligations remain in continuity; no standard clause is a scoring unit.
CREATE TABLE framework_catalog_editions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    publisher_code text NOT NULL CHECK (publisher_code = btrim(publisher_code) AND char_length(publisher_code) BETWEEN 1 AND 64),
    framework_code text NOT NULL CHECK (framework_code = btrim(framework_code) AND char_length(framework_code) BETWEEN 1 AND 64),
    edition_code text NOT NULL CHECK (edition_code = btrim(edition_code) AND char_length(edition_code) BETWEEN 1 AND 64),
    title text NOT NULL CHECK (title = btrim(title) AND char_length(title) BETWEEN 1 AND 240),
    jurisdiction text NOT NULL CHECK (jurisdiction = btrim(jurisdiction) AND char_length(jurisdiction) BETWEEN 2 AND 64),
    source_url text NOT NULL CHECK (source_url = btrim(source_url) AND char_length(source_url) BETWEEN 9 AND 2048),
    source_sha256 text NOT NULL CHECK (source_sha256 ~ '^[0-9a-f]{64}$'),
    source_published_on date NOT NULL,
    effective_on date,
    license_basis text NOT NULL CHECK (license_basis IN ('CITATIONS_ONLY','SUMMARIES_PERMITTED')),
    license_reference text NOT NULL DEFAULT '' CHECK (license_reference = btrim(license_reference) AND char_length(license_reference) <= 500),
    release_sha256 text NOT NULL CHECK (release_sha256 ~ '^[0-9a-f]{64}$'),
    approval_reference text NOT NULL CHECK (approval_reference = btrim(approval_reference) AND char_length(approval_reference) BETWEEN 1 AND 240),
    published_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (publisher_code, framework_code, edition_code),
    UNIQUE (release_sha256),
    CHECK (license_basis = 'CITATIONS_ONLY' OR license_reference <> '')
);

CREATE TABLE framework_catalog_clauses (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    edition_id uuid NOT NULL REFERENCES framework_catalog_editions(id),
    code text NOT NULL CHECK (code = btrim(code) AND char_length(code) BETWEEN 1 AND 128),
    parent_code text CHECK (parent_code IS NULL OR (parent_code = btrim(parent_code) AND char_length(parent_code) BETWEEN 1 AND 128)),
    title text NOT NULL CHECK (title = btrim(title) AND char_length(title) BETWEEN 1 AND 500),
    source_anchor text NOT NULL CHECK (source_anchor = btrim(source_anchor) AND char_length(source_anchor) BETWEEN 1 AND 500),
    summary text NOT NULL DEFAULT '' CHECK (summary = btrim(summary) AND char_length(summary) <= 2000),
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    UNIQUE (edition_id, code),
    CONSTRAINT framework_catalog_clause_parent_fk FOREIGN KEY (edition_id, parent_code)
      REFERENCES framework_catalog_clauses (edition_id, code)
      DEFERRABLE INITIALLY DEFERRED,
    CHECK (parent_code IS NULL OR parent_code <> code)
);

CREATE INDEX framework_catalog_editions_discovery_idx
    ON framework_catalog_editions (jurisdiction, id);
CREATE INDEX framework_catalog_clauses_order_idx
    ON framework_catalog_clauses (edition_id, ordinal, code);

-- Publication facts are append-only; future withdrawal and revision decisions
-- require an independent governed lifecycle, never mutation of edition history.
CREATE FUNCTION reject_framework_catalog_content_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'published framework content is immutable';
END;
$$;
CREATE TRIGGER framework_catalog_editions_immutable
    BEFORE UPDATE OR DELETE ON framework_catalog_editions
    FOR EACH ROW EXECUTE FUNCTION reject_framework_catalog_content_mutation();
CREATE TRIGGER framework_catalog_clauses_immutable
    BEFORE UPDATE OR DELETE ON framework_catalog_clauses
    FOR EACH ROW EXECUTE FUNCTION reject_framework_catalog_content_mutation();

COMMIT;
