BEGIN;

-- F2 persistence boundary. Material writes will be executed only through
-- verified Program commands with continuity events and maker-checker review.
-- Catalog clauses are references, not Program Requirements or scoring units.
CREATE UNIQUE INDEX framework_catalog_editions_identity_fk_idx
    ON framework_catalog_editions(id,publisher_code,framework_code,edition_code);
CREATE UNIQUE INDEX framework_catalog_clauses_edition_fk_idx
    ON framework_catalog_clauses(id,edition_id);

CREATE TABLE program_framework_adoptions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    program_id uuid NOT NULL,
    edition_id uuid NOT NULL,
    publisher_code text NOT NULL,
    framework_code text NOT NULL,
    edition_code text NOT NULL,
    status text NOT NULL CHECK (status IN ('PENDING','ACTIVE','REJECTED','SUPERSEDED','RETIRED')),
    proposed_by uuid NOT NULL,
    reviewed_by uuid,
    rationale text NOT NULL CHECK (char_length(btrim(rationale)) BETWEEN 1 AND 2000),
    proposed_at timestamptz NOT NULL,
    reviewed_at timestamptz,
    effective_from timestamptz,
    effective_until timestamptz,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    UNIQUE(id,tenant_id,legal_entity_id,program_id,edition_id),
    FOREIGN KEY (program_id,tenant_id,legal_entity_id)
      REFERENCES programs(id,tenant_id,legal_entity_id),
    FOREIGN KEY (edition_id,publisher_code,framework_code,edition_code)
      REFERENCES framework_catalog_editions(id,publisher_code,framework_code,edition_code),
    FOREIGN KEY (proposed_by,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (reviewed_by,tenant_id) REFERENCES principals(id,tenant_id),
    CONSTRAINT program_framework_adoption_decision_ck CHECK (
      (status='PENDING' AND reviewed_by IS NULL AND reviewed_at IS NULL AND
        effective_from IS NULL AND effective_until IS NULL)
      OR
      (status='REJECTED' AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND
        effective_from IS NULL AND effective_until IS NULL)
      OR
      (status='ACTIVE' AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND
        effective_from IS NOT NULL AND effective_until IS NULL)
      OR
      (status IN ('SUPERSEDED','RETIRED') AND reviewed_by IS NOT NULL AND
        reviewed_at IS NOT NULL AND effective_from IS NOT NULL AND
        effective_until IS NOT NULL AND effective_from < effective_until)
    ),
    CONSTRAINT program_framework_adoption_dates_ck CHECK (
      reviewed_at IS NULL OR (reviewed_at >= proposed_at AND
        (effective_from IS NULL OR effective_from >= reviewed_at))
    ),
    CONSTRAINT program_framework_adoption_independent_review_ck
      CHECK (reviewed_by IS NULL OR reviewed_by <> proposed_by)
);

CREATE UNIQUE INDEX program_framework_adoptions_pending_standard_uq
    ON program_framework_adoptions(tenant_id,legal_entity_id,program_id,publisher_code,framework_code)
    WHERE status='PENDING';
CREATE UNIQUE INDEX program_framework_adoptions_active_standard_uq
    ON program_framework_adoptions(tenant_id,legal_entity_id,program_id,publisher_code,framework_code)
    WHERE status='ACTIVE';
CREATE INDEX program_framework_adoptions_program_idx
    ON program_framework_adoptions(tenant_id,legal_entity_id,program_id,status,proposed_at DESC,id DESC);
CREATE INDEX program_framework_adoptions_edition_dependents_idx
    ON program_framework_adoptions(edition_id,status,tenant_id,legal_entity_id,program_id);

CREATE TABLE framework_clause_requirement_links (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    program_id uuid NOT NULL,
    adoption_id uuid NOT NULL,
    edition_id uuid NOT NULL,
    clause_id uuid NOT NULL,
    requirement_id uuid NOT NULL,
    mapping_kind text NOT NULL CHECK (mapping_kind IN ('EXACT','PARTIAL','SUPPORTING')),
    rationale text NOT NULL CHECK (char_length(btrim(rationale)) BETWEEN 1 AND 2000),
    mapped_by uuid NOT NULL,
    reviewed_by uuid NOT NULL,
    reviewed_at timestamptz NOT NULL,
    effective_from timestamptz NOT NULL,
    retired_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (effective_from >= reviewed_at),
    CHECK (mapped_by <> reviewed_by),
    CHECK (retired_at IS NULL OR effective_from < retired_at),
    FOREIGN KEY (adoption_id,tenant_id,legal_entity_id,program_id,edition_id)
      REFERENCES program_framework_adoptions(id,tenant_id,legal_entity_id,program_id,edition_id),
    FOREIGN KEY (clause_id,edition_id)
      REFERENCES framework_catalog_clauses(id,edition_id),
    FOREIGN KEY (requirement_id,tenant_id,program_id)
      REFERENCES program_requirements(id,tenant_id,program_id),
    FOREIGN KEY (mapped_by,tenant_id)
      REFERENCES principals(id,tenant_id),
    FOREIGN KEY (reviewed_by,tenant_id)
      REFERENCES principals(id,tenant_id)
);

-- The database is a second boundary after governed Program authorization:
-- an unapproved framework adoption or inactive Requirement cannot be mapped.
CREATE FUNCTION guard_framework_clause_requirement_link() RETURNS trigger LANGUAGE plpgsql AS $
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM program_framework_adoptions a
        JOIN program_requirements r
          ON r.id=NEW.requirement_id
         AND r.tenant_id=a.tenant_id
         AND r.program_id=a.program_id
        WHERE a.id=NEW.adoption_id
          AND a.tenant_id=NEW.tenant_id
          AND a.legal_entity_id=NEW.legal_entity_id
          AND a.program_id=NEW.program_id
          AND a.edition_id=NEW.edition_id
          AND a.status='ACTIVE'
          AND a.effective_from<=NEW.effective_from
          AND a.reviewed_at<=NEW.reviewed_at
          AND NEW.mapped_by<>a.reviewed_by
          AND r.status='APPROVED'
          AND r.effective_from<=NEW.effective_from
          AND (r.effective_until IS NULL OR NEW.effective_from<r.effective_until)
    ) THEN
        RAISE EXCEPTION 'framework clause mapping requires an active approved Program adoption and Requirement';
    END IF;
    RETURN NEW;
END;
$;
CREATE TRIGGER framework_clause_requirement_link_guard
    BEFORE INSERT ON framework_clause_requirement_links
    FOR EACH ROW EXECUTE FUNCTION guard_framework_clause_requirement_link();

CREATE UNIQUE INDEX framework_clause_requirement_links_current_uq
    ON framework_clause_requirement_links(tenant_id,legal_entity_id,adoption_id,clause_id,requirement_id)
    WHERE retired_at IS NULL;
CREATE INDEX framework_clause_requirement_links_program_idx
    ON framework_clause_requirement_links(tenant_id,legal_entity_id,program_id,adoption_id,clause_id,id)
    WHERE retired_at IS NULL;
CREATE INDEX framework_clause_requirement_links_requirement_idx
    ON framework_clause_requirement_links(tenant_id,program_id,requirement_id,effective_from DESC,id DESC);

COMMIT;
