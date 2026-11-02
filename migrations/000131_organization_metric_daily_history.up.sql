BEGIN;

CREATE TABLE organization_metric_daily_sources (
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    definition_revision text NOT NULL CHECK (definition_revision='enterprise-domain-v1'),
    bucket_date date NOT NULL,
    source_id uuid NOT NULL,
    source_revision text NOT NULL,
    source_generated_at timestamptz NOT NULL,
    source_high_water jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(source_high_water)='object'),
    finalized_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY(tenant_id,legal_entity_id,definition_revision,bucket_date),
    FOREIGN KEY(legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    CHECK (
        source_generated_at>=(bucket_date::timestamp AT TIME ZONE 'UTC')
        AND source_generated_at<((bucket_date+1)::timestamp AT TIME ZONE 'UTC')
    )
);

CREATE INDEX organization_metric_daily_sources_time_idx
    ON organization_metric_daily_sources(
        tenant_id,legal_entity_id,bucket_date DESC,source_generated_at DESC
    );

CREATE TABLE organization_metric_daily_buckets (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    definition_revision text NOT NULL CHECK (definition_revision='enterprise-domain-v1'),
    bucket_date date NOT NULL,
    metric_id text NOT NULL CHECK (
        metric_id IN ('risks_outside_appetite','indicator_breaches','assurance_failures')
    ),
    organization_scope_id uuid,
    lineage_event_id uuid,
    value bigint NOT NULL CHECK (value>=0),
    recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY(
        tenant_id,legal_entity_id,definition_revision,bucket_date
    ) REFERENCES organization_metric_daily_sources(
        tenant_id,legal_entity_id,definition_revision,bucket_date
    ) ON DELETE CASCADE,
    FOREIGN KEY(metric_id,definition_revision)
        REFERENCES metric_definitions(metric_id,revision),
    FOREIGN KEY(tenant_id,legal_entity_id,organization_scope_id)
        REFERENCES organization_scopes(tenant_id,legal_entity_id,id),
    FOREIGN KEY(lineage_event_id)
        REFERENCES organization_scope_lineage_events(id),
    CHECK (
        (organization_scope_id IS NULL AND lineage_event_id IS NULL)
        OR (organization_scope_id IS NOT NULL AND lineage_event_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX organization_metric_daily_bucket_identity_uq
    ON organization_metric_daily_buckets(
        tenant_id,legal_entity_id,definition_revision,bucket_date,metric_id,organization_scope_id
    ) NULLS NOT DISTINCT;

CREATE INDEX organization_metric_daily_bucket_scope_idx
    ON organization_metric_daily_buckets(
        tenant_id,legal_entity_id,organization_scope_id,bucket_date DESC,metric_id
    );

CREATE FUNCTION prevent_organization_metric_daily_mutation() RETURNS trigger
LANGUAGE plpgsql
AS $daily_metric_immutable$
BEGIN
    RAISE EXCEPTION 'Organization metric daily history is immutable';
END;
$daily_metric_immutable$;

CREATE TRIGGER organization_metric_daily_sources_immutable
    BEFORE UPDATE OR DELETE ON organization_metric_daily_sources
    FOR EACH ROW EXECUTE FUNCTION prevent_organization_metric_daily_mutation();

CREATE TRIGGER organization_metric_daily_buckets_immutable
    BEFORE UPDATE OR DELETE ON organization_metric_daily_buckets
    FOR EACH ROW EXECUTE FUNCTION prevent_organization_metric_daily_mutation();

COMMIT;
