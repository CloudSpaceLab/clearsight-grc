BEGIN;

CREATE TABLE group_oversight_runs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    refresh_slot timestamptz NOT NULL,
    generated_at timestamptz NOT NULL,
    projection_version text NOT NULL,
    active_child_count integer NOT NULL CHECK (active_child_count >= 0),
    captured_child_count integer NOT NULL CHECK (captured_child_count >= 0),
    missing_child_count integer NOT NULL CHECK (missing_child_count >= 0),
    stale_child_count integer NOT NULL CHECK (stale_child_count >= 0),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, projection_version, refresh_slot),
    CHECK (captured_child_count + missing_child_count = active_child_count),
    CHECK (stale_child_count <= captured_child_count)
);

CREATE INDEX group_oversight_runs_latest_idx
    ON group_oversight_runs(tenant_id, generated_at DESC, id DESC);

CREATE TABLE group_oversight_child_facts (
    run_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    legal_entity_code text NOT NULL,
    legal_entity_name text NOT NULL,
    jurisdiction text NOT NULL DEFAULT '',
    state text NOT NULL CHECK (state IN ('AVAILABLE','STALE','MISSING')),
    child_snapshot_id uuid,
    child_generated_at timestamptz,
    child_projection_version text,
    coverage_population integer,
    coverage_excluded integer,
    coverage_unknown integer,
    counts jsonb NOT NULL DEFAULT '{}'::jsonb,
    source_high_water jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (run_id, legal_entity_id),
    CONSTRAINT group_oversight_child_run_fk
        FOREIGN KEY (tenant_id, run_id)
        REFERENCES group_oversight_runs(tenant_id, id)
        ON DELETE CASCADE,
    CONSTRAINT group_oversight_child_entity_fk
        FOREIGN KEY (legal_entity_id, tenant_id)
        REFERENCES legal_entities(id, tenant_id),
    CHECK (jsonb_typeof(counts) = 'object'),
    CHECK (jsonb_typeof(source_high_water) = 'object'),
    CHECK (
        (state='MISSING' AND child_snapshot_id IS NULL AND child_generated_at IS NULL AND child_projection_version IS NULL
            AND coverage_population IS NULL AND coverage_excluded IS NULL AND coverage_unknown IS NULL)
        OR
        (state IN ('AVAILABLE','STALE') AND child_snapshot_id IS NOT NULL AND child_generated_at IS NOT NULL
            AND child_projection_version IS NOT NULL AND coverage_population IS NOT NULL AND coverage_population >= 0
            AND (coverage_excluded IS NULL OR coverage_excluded >= 0)
            AND (coverage_unknown IS NULL OR coverage_unknown >= 0))
    )
);

CREATE INDEX group_oversight_child_entity_idx
    ON group_oversight_child_facts(tenant_id, legal_entity_id, run_id);

CREATE FUNCTION prevent_group_oversight_projection_mutation() RETURNS trigger
LANGUAGE plpgsql
AS $group_oversight$
BEGIN
    RAISE EXCEPTION 'Group oversight projections are immutable';
END;
$group_oversight$;

CREATE TRIGGER group_oversight_runs_immutable
    BEFORE UPDATE ON group_oversight_runs
    FOR EACH ROW EXECUTE FUNCTION prevent_group_oversight_projection_mutation();

CREATE TRIGGER group_oversight_child_facts_immutable
    BEFORE UPDATE ON group_oversight_child_facts
    FOR EACH ROW EXECUTE FUNCTION prevent_group_oversight_projection_mutation();

COMMIT;
