BEGIN;

CREATE FUNCTION valid_legal_entity_data_regions(values text[]) RETURNS boolean
LANGUAGE sql
IMMUTABLE
AS $data_regions$
    SELECT cardinality(values) <= 64
       AND NOT EXISTS (
           SELECT 1
           FROM unnest(values) AS region(value)
           WHERE value !~ '^[A-Z0-9][A-Z0-9_-]{1,31}$'
       )
       AND cardinality(values) = (
           SELECT count(DISTINCT value)
           FROM unnest(values) AS region(value)
       );
$data_regions$;

CREATE TABLE legal_entity_data_boundaries (
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    residency_region text NOT NULL,
    detail_transfer_mode text NOT NULL DEFAULT 'AGGREGATE_ONLY'
        CHECK (detail_transfer_mode IN ('AGGREGATE_ONLY','ALLOWLIST')),
    allowed_destination_regions text[] NOT NULL DEFAULT ARRAY[]::text[],
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (tenant_id, legal_entity_id),
    FOREIGN KEY (tenant_id, legal_entity_id) REFERENCES legal_entities(tenant_id, id),
    CHECK (residency_region ~ '^[A-Z0-9][A-Z0-9_-]{1,31}$'),
    CHECK (valid_legal_entity_data_regions(allowed_destination_regions)),
    CHECK (
        (detail_transfer_mode='AGGREGATE_ONLY' AND cardinality(allowed_destination_regions)=0)
        OR
        (detail_transfer_mode='ALLOWLIST' AND cardinality(allowed_destination_regions)>0)
    )
);

CREATE TABLE legal_entity_data_boundary_revisions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    base_version bigint NOT NULL CHECK (base_version >= 0),
    proposed_residency_region text NOT NULL,
    proposed_detail_transfer_mode text NOT NULL
        CHECK (proposed_detail_transfer_mode IN ('AGGREGATE_ONLY','ALLOWLIST')),
    proposed_destination_regions text[] NOT NULL DEFAULT ARRAY[]::text[],
    maker_id uuid NOT NULL,
    checker_id uuid,
    status text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','APPLIED','REJECTED')),
    rationale text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    decided_at timestamptz,
    applied_at timestamptz,
    FOREIGN KEY (tenant_id, legal_entity_id) REFERENCES legal_entities(tenant_id, id),
    FOREIGN KEY (tenant_id, maker_id) REFERENCES principals(tenant_id, id),
    FOREIGN KEY (tenant_id, checker_id) REFERENCES principals(tenant_id, id),
    CHECK (checker_id IS NULL OR checker_id<>maker_id),
    CHECK (
        (status='PENDING' AND checker_id IS NULL AND rationale='' AND decided_at IS NULL AND applied_at IS NULL)
        OR
        (status='REJECTED' AND checker_id IS NOT NULL AND btrim(rationale)<>'' AND decided_at IS NOT NULL AND applied_at IS NULL)
        OR
        (status='APPLIED' AND checker_id IS NOT NULL AND btrim(rationale)<>'' AND decided_at IS NOT NULL AND applied_at IS NOT NULL)
    ),
    CHECK (proposed_residency_region ~ '^[A-Z0-9][A-Z0-9_-]{1,31}$'),
    CHECK (valid_legal_entity_data_regions(proposed_destination_regions)),
    CHECK (
        (proposed_detail_transfer_mode='AGGREGATE_ONLY' AND cardinality(proposed_destination_regions)=0)
        OR
        (proposed_detail_transfer_mode='ALLOWLIST' AND cardinality(proposed_destination_regions)>0)
    )
);

CREATE UNIQUE INDEX legal_entity_data_boundary_pending_idx
    ON legal_entity_data_boundary_revisions(tenant_id, legal_entity_id)
    WHERE status='PENDING';

CREATE INDEX legal_entity_data_boundary_revision_queue_idx
    ON legal_entity_data_boundary_revisions(tenant_id, legal_entity_id, status, created_at, id);

COMMIT;
