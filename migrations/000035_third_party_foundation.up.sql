BEGIN;

CREATE TABLE third_parties (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_name text NOT NULL CHECK (legal_name=btrim(legal_name) AND char_length(legal_name) BETWEEN 1 AND 512),
    trading_name text NOT NULL DEFAULT '' CHECK (trading_name=btrim(trading_name) AND char_length(trading_name) <= 512),
    registration_ref text NOT NULL DEFAULT '' CHECK (registration_ref=btrim(registration_ref) AND char_length(registration_ref) <= 256),
    jurisdiction text NOT NULL DEFAULT '' CHECK (jurisdiction=btrim(jurisdiction) AND char_length(jurisdiction) <= 256),
    source_id text NOT NULL DEFAULT '' CHECK (source_id=btrim(source_id) AND char_length(source_id) <= 256),
    external_ref text NOT NULL DEFAULT '' CHECK (external_ref=btrim(external_ref) AND char_length(external_ref) <= 512),
    status text NOT NULL CHECK (status IN ('ACTIVE','INACTIVE')),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at),
    version bigint NOT NULL CHECK (version > 0),
    UNIQUE (id, tenant_id),
    CHECK ((source_id='' AND external_ref='') OR (source_id<>'' AND external_ref<>''))
);
CREATE UNIQUE INDEX third_parties_source_identity_idx
    ON third_parties(tenant_id,source_id,external_ref)
    WHERE source_id<>'' AND external_ref<>'';
CREATE INDEX third_parties_name_idx ON third_parties(tenant_id,lower(legal_name),id);

CREATE TABLE third_party_relationships (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    vendor_id uuid NOT NULL,
    service_name text NOT NULL CHECK (service_name=btrim(service_name) AND char_length(service_name) BETWEEN 1 AND 512),
    business_owner_principal_id uuid NOT NULL,
    criticality text NOT NULL CHECK (criticality IN ('STANDARD','IMPORTANT','CRITICAL')),
    privacy_role text NOT NULL CHECK (privacy_role IN ('NONE','PROCESSOR','JOINT_CONTROLLER')),
    status text NOT NULL CHECK (status IN ('PROPOSED','UNDER_REVIEW','ACTIVE','RESTRICTED','SUSPENDED','EXITING','TERMINATED')),
    effective_from timestamptz,
    renewal_at timestamptz,
    source_id text NOT NULL DEFAULT '' CHECK (source_id=btrim(source_id) AND char_length(source_id) <= 256),
    external_ref text NOT NULL DEFAULT '' CHECK (external_ref=btrim(external_ref) AND char_length(external_ref) <= 512),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at),
    version bigint NOT NULL CHECK (version > 0),
    UNIQUE (id, tenant_id),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (vendor_id,tenant_id) REFERENCES third_parties(id,tenant_id),
    FOREIGN KEY (business_owner_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK (renewal_at IS NULL OR effective_from IS NULL OR renewal_at >= effective_from),
    CHECK ((source_id='' AND external_ref='') OR (source_id<>'' AND external_ref<>''))
);
CREATE INDEX third_party_relationships_scoped_list_idx
    ON third_party_relationships(tenant_id,legal_entity_id,updated_at DESC,id DESC);
CREATE INDEX third_party_relationships_vendor_idx
    ON third_party_relationships(tenant_id,vendor_id,updated_at DESC,id DESC);
CREATE INDEX third_party_relationships_owner_idx
    ON third_party_relationships(tenant_id,legal_entity_id,business_owner_principal_id,status,updated_at DESC);

CREATE TABLE third_party_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    aggregate_type text NOT NULL CHECK (aggregate_type='VENDOR_RELATIONSHIP'),
    aggregate_id uuid NOT NULL,
    aggregate_version bigint NOT NULL CHECK (aggregate_version > 0),
    actor_principal_id uuid NOT NULL,
    event_type text NOT NULL CHECK (event_type IN ('VendorRelationshipCreated','VendorRelationshipUpdated')),
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload)='object'),
    occurred_at timestamptz NOT NULL,
    FOREIGN KEY (aggregate_id,tenant_id) REFERENCES third_party_relationships(id,tenant_id),
    FOREIGN KEY (actor_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    UNIQUE (tenant_id,aggregate_id,aggregate_version)
);
CREATE INDEX third_party_events_history_idx
    ON third_party_events(tenant_id,aggregate_id,aggregate_version,id);

COMMIT;
