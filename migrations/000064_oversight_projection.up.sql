BEGIN;

CREATE TABLE oversight_snapshots (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    period_start timestamptz NOT NULL,
    period_end timestamptz NOT NULL,
    refresh_slot timestamptz NOT NULL,
    generated_at timestamptz NOT NULL,
    projection_version text NOT NULL,
    source_high_water jsonb NOT NULL DEFAULT '{}'::jsonb,
    coverage_population integer NOT NULL CHECK (coverage_population >= 0),
    coverage_excluded integer CHECK (coverage_excluded IS NULL OR coverage_excluded >= 0),
    coverage_unknown integer CHECK (coverage_unknown IS NULL OR coverage_unknown >= 0),
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (period_start < period_end),
    CHECK (jsonb_typeof(source_high_water) = 'object'),
    CHECK (jsonb_typeof(payload) = 'object'),
    CONSTRAINT oversight_snapshots_entity_tenant_fk
        FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    UNIQUE (tenant_id,legal_entity_id,projection_version,refresh_slot)
);

CREATE INDEX oversight_snapshots_latest_idx
    ON oversight_snapshots(tenant_id,legal_entity_id,generated_at DESC,id DESC);
CREATE INDEX oversight_snapshots_retention_idx
    ON oversight_snapshots(generated_at,id);

UPDATE role_templates
SET capabilities = (
    SELECT ARRAY(SELECT DISTINCT value FROM unnest(capabilities || ARRAY['OVERSIGHT_READ']) value ORDER BY value)
)
WHERE valid_until IS NULL AND code IN ('CRO','CCO','CISO','EXECUTIVE','GRC_ADMIN');

COMMIT;
