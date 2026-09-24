BEGIN;

-- A processing activity may be linked to the Matter that raised the change.
-- The legal entity is part of the key so an activity cannot point at a Matter
-- from another bank entity in the same tenant.
ALTER TABLE ropa_processing_activities ADD COLUMN matter_id uuid;
ALTER TABLE ropa_processing_activities
    ADD CONSTRAINT ropa_activities_matter_tenant_fk
    FOREIGN KEY (matter_id, tenant_id, legal_entity_id)
    REFERENCES matters(id, tenant_id, legal_entity_id);
CREATE INDEX ropa_matter_idx
    ON ropa_processing_activities(tenant_id, legal_entity_id, matter_id);

CREATE TABLE report_definitions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid NOT NULL,
    code text NOT NULL CHECK (code ~ '^[A-Z0-9][A-Z0-9_-]{2,47}$'),
    name text NOT NULL CHECK (char_length(btrim(name)) BETWEEN 3 AND 120),
    description text NOT NULL DEFAULT '' CHECK (char_length(description) <= 1000),
    dataset text NOT NULL CHECK (dataset IN ('PROCESSING_ACTIVITIES','PROCESSING_ACTIVITY_EXCEPTIONS')),
    scope_kind text NOT NULL CHECK (scope_kind IN ('LEGAL_ENTITY','PROGRAM','MATTER')),
    scope_ref uuid,
    format text NOT NULL CHECK (format IN ('CSV','NDJSON')),
    filter jsonb NOT NULL DEFAULT '{"kind":"group","operator":"and","children":[]}'::jsonb
        CHECK (jsonb_typeof(filter)='object' AND octet_length(filter::text) <= 8192),
    status text NOT NULL CHECK (status IN ('DRAFT','PENDING_REVIEW','ACTIVE','RETIRED')),
    current_version integer NOT NULL DEFAULT 1 CHECK (current_version >= 1),
    checksum text NOT NULL CHECK (checksum ~ '^[0-9a-f]{64}$'),
    maker_id text NOT NULL CHECK (char_length(btrim(maker_id)) > 0),
    checker_id text,
    effective_from timestamptz,
    effective_until timestamptz,
    submitted_at timestamptz,
    approved_at timestamptz,
    retired_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    -- scope_ref is polymorphic by scope_kind. Generated keys let both parent
    -- tables be protected without accepting a same-tenant cross-entity link.
    program_scope_ref uuid GENERATED ALWAYS AS
        (CASE WHEN scope_kind = 'PROGRAM' THEN scope_ref ELSE NULL END) STORED,
    matter_scope_ref uuid GENERATED ALWAYS AS
        (CASE WHEN scope_kind = 'MATTER' THEN scope_ref ELSE NULL END) STORED,
    CONSTRAINT report_definitions_entity_tenant_fk
        FOREIGN KEY (legal_entity_id, tenant_id) REFERENCES legal_entities(id, tenant_id),
    CONSTRAINT report_definitions_program_scope_fk
        FOREIGN KEY (program_scope_ref, tenant_id, legal_entity_id)
        REFERENCES programs(id, tenant_id, legal_entity_id),
    CONSTRAINT report_definitions_matter_scope_fk
        FOREIGN KEY (matter_scope_ref, tenant_id, legal_entity_id)
        REFERENCES matters(id, tenant_id, legal_entity_id),
    CONSTRAINT report_definitions_owner_fk
        CHECK (maker_id = btrim(maker_id) AND char_length(maker_id) > 0),
    CONSTRAINT report_definitions_scope_shape_ck CHECK (
        (scope_kind='LEGAL_ENTITY' AND scope_ref IS NULL) OR
        (scope_kind IN ('PROGRAM','MATTER') AND scope_ref IS NOT NULL)),
    CONSTRAINT report_definitions_maker_checker_ck CHECK (
        checker_id IS NULL OR (checker_id = btrim(checker_id) AND checker_id <> maker_id)),
    CONSTRAINT report_definitions_effective_ck CHECK (
        effective_until IS NULL OR effective_from IS NULL OR effective_until > effective_from),
    CONSTRAINT report_definitions_status_ck CHECK (
        (status='DRAFT') OR
        (status='PENDING_REVIEW' AND submitted_at IS NOT NULL) OR
        (status='ACTIVE' AND checker_id IS NOT NULL AND approved_at IS NOT NULL AND effective_from IS NOT NULL) OR
        (status='RETIRED' AND retired_at IS NOT NULL)),
    CONSTRAINT report_definitions_scope_kind_ck CHECK (
        scope_kind <> 'MATTER' OR scope_ref IS NOT NULL),
    CONSTRAINT report_definitions_updated_at_order_ck CHECK (updated_at >= created_at),
    -- The composite key is required by immutable revisions and run receipts.
    UNIQUE (id, tenant_id, legal_entity_id)
);
CREATE UNIQUE INDEX report_definitions_scope_code_uq
    ON report_definitions(tenant_id, legal_entity_id, code) WHERE status <> 'RETIRED';
CREATE INDEX report_definitions_tenant_time_idx
    ON report_definitions(tenant_id, legal_entity_id, created_at DESC, id DESC);

CREATE TABLE report_definition_revisions (
    definition_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    version integer NOT NULL CHECK (version >= 1),
    base_version integer NOT NULL CHECK (base_version >= 0),
    dataset text NOT NULL CHECK (dataset IN ('PROCESSING_ACTIVITIES','PROCESSING_ACTIVITY_EXCEPTIONS')),
    scope_kind text NOT NULL CHECK (scope_kind IN ('LEGAL_ENTITY','PROGRAM','MATTER')),
    scope_ref uuid,
    format text NOT NULL CHECK (format IN ('CSV','NDJSON')),
    filter jsonb NOT NULL CHECK (jsonb_typeof(filter)='object' AND octet_length(filter::text) <= 8192),
    checksum text NOT NULL CHECK (checksum ~ '^[0-9a-f]{64}$'),
    maker_id text NOT NULL CHECK (char_length(btrim(maker_id)) > 0),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    approved_by text,
    approved_at timestamptz,
    decision text NOT NULL DEFAULT 'PROPOSED' CHECK (decision IN ('PROPOSED','APPROVED','REJECTED','RETIRED')),
    decision_note text NOT NULL DEFAULT '' CHECK (char_length(decision_note) <= 1000),
    PRIMARY KEY (definition_id, version),
    CONSTRAINT report_definition_revisions_entity_fk
        FOREIGN KEY (definition_id, tenant_id, legal_entity_id)
        REFERENCES report_definitions(id, tenant_id, legal_entity_id),
    CONSTRAINT report_definition_revisions_scope_shape_ck CHECK (
        (scope_kind='LEGAL_ENTITY' AND scope_ref IS NULL) OR
        (scope_kind IN ('PROGRAM','MATTER') AND scope_ref IS NOT NULL)),
    CONSTRAINT report_definition_revisions_maker_checker_ck
        CHECK (approved_by IS NULL OR (approved_by = btrim(approved_by) AND approved_by <> maker_id)),
    CONSTRAINT report_definition_revisions_decision_ck CHECK (
        (decision='PROPOSED' AND approved_by IS NULL AND approved_at IS NULL) OR
        (decision IN ('APPROVED','REJECTED') AND approved_by IS NOT NULL AND approved_at IS NOT NULL) OR
        (decision='RETIRED' AND approved_at IS NULL))
);
CREATE FUNCTION report_definition_revisions_immutable() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'report definition revisions are immutable';
END;
$$;
CREATE TRIGGER report_definition_revisions_immutable
    BEFORE UPDATE OR DELETE ON report_definition_revisions
    FOR EACH ROW EXECUTE FUNCTION report_definition_revisions_immutable();

CREATE TABLE report_runs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid NOT NULL,
    definition_id uuid NOT NULL,
    definition_version integer NOT NULL CHECK (definition_version >= 1),
    requested_by_ref text NOT NULL CHECK (char_length(btrim(requested_by_ref)) > 0),
    as_of timestamptz NOT NULL,
    filter jsonb NOT NULL CHECK (jsonb_typeof(filter)='object' AND octet_length(filter::text) <= 8192),
    dataset text NOT NULL CHECK (dataset IN ('PROCESSING_ACTIVITIES','PROCESSING_ACTIVITY_EXCEPTIONS')),
    format text NOT NULL CHECK (format IN ('CSV','NDJSON')),
    status text NOT NULL CHECK (status IN ('QUEUED','RUNNING','READY','FAILED')),
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0 AND attempt_count <= 5),
    row_count integer NOT NULL DEFAULT 0 CHECK (row_count >= 0),
    data_object_key text,
    data_sha256 text,
    manifest_object_key text,
    manifest_sha256 text,
    failure_code text,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    completed_at timestamptz,
    expires_at timestamptz NOT NULL,
    CONSTRAINT report_runs_definition_fk
        FOREIGN KEY (definition_id, tenant_id, legal_entity_id)
        REFERENCES report_definitions(id, tenant_id, legal_entity_id),
    CONSTRAINT report_runs_entity_tenant_fk
        FOREIGN KEY (legal_entity_id, tenant_id) REFERENCES legal_entities(id, tenant_id),
    CONSTRAINT report_runs_expiry_ck CHECK (expires_at > created_at),
    CONSTRAINT report_runs_generation_guard CHECK (
        (status='READY') = (data_object_key IS NOT NULL AND data_sha256 IS NOT NULL
                            AND manifest_object_key IS NOT NULL AND manifest_sha256 IS NOT NULL
                            AND completed_at IS NOT NULL)),
    CONSTRAINT report_runs_failure_ck CHECK (status <> 'FAILED' OR failure_code IS NOT NULL)
);
CREATE INDEX report_runs_tenant_time_idx ON report_runs(tenant_id, legal_entity_id, created_at DESC, id DESC);
CREATE INDEX report_runs_definition_idx ON report_runs(tenant_id, definition_id, created_at DESC, id DESC);
CREATE INDEX report_runs_expiry_idx ON report_runs(expires_at, id);
CREATE INDEX report_runs_queue_idx ON report_runs(status, created_at, id) WHERE status IN ('QUEUED','RUNNING');

-- A run may only enter READY once it holds a real artefact, and a terminal row
-- may never be re-opened. This is the database-level half of the generation guard.
CREATE FUNCTION report_runs_generation_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.status IN ('READY','FAILED') AND OLD.status IN ('READY','FAILED') THEN
    RAISE EXCEPTION 'report run % is already terminal', OLD.id;
  END IF;
  IF NEW.status = 'READY' AND (NEW.data_object_key IS NULL OR NEW.manifest_object_key IS NULL) THEN
    RAISE EXCEPTION 'report run % cannot be READY without artefacts', NEW.id;
  END IF;
  IF NEW.attempt_count > 5 THEN
    RAISE EXCEPTION 'report run % exhausted its retry budget', NEW.id;
  END IF;
  RETURN NEW;
END;
$$;
CREATE TRIGGER report_runs_generation_guard
    BEFORE UPDATE ON report_runs
    FOR EACH ROW EXECUTE FUNCTION report_runs_generation_guard();

COMMIT;
