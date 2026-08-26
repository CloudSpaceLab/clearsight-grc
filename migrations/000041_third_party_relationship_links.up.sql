BEGIN;

CREATE TABLE third_party_relationship_program_links (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    relationship_id uuid NOT NULL,
    program_id uuid NOT NULL,
    purpose_code text NOT NULL CHECK (purpose_code ~ '^[A-Z][A-Z0-9_]{1,63}$'),
    purpose_label text NOT NULL CHECK (purpose_label=btrim(purpose_label) AND char_length(purpose_label) BETWEEN 1 AND 160),
    state text NOT NULL CHECK (state IN ('ACTIVE','ENDED')),
    created_by_principal_id uuid NOT NULL,
    ended_by_principal_id uuid,
    end_reason text NOT NULL DEFAULT '' CHECK (end_reason=btrim(end_reason) AND char_length(end_reason) <= 1000),
    version bigint NOT NULL CHECK (version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at),
    ended_at timestamptz,
    UNIQUE (id,tenant_id,legal_entity_id),
    FOREIGN KEY (relationship_id,tenant_id,legal_entity_id) REFERENCES third_party_relationships(id,tenant_id,legal_entity_id),
    FOREIGN KEY (program_id,tenant_id) REFERENCES programs(id,tenant_id),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (created_by_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (ended_by_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK ((state='ACTIVE' AND ended_at IS NULL AND ended_by_principal_id IS NULL AND end_reason='') OR
           (state='ENDED' AND ended_at IS NOT NULL AND ended_by_principal_id IS NOT NULL AND end_reason<>''))
);
CREATE UNIQUE INDEX third_party_relationship_program_links_active_idx
    ON third_party_relationship_program_links(tenant_id,legal_entity_id,relationship_id,program_id)
    WHERE state='ACTIVE';
CREATE INDEX third_party_relationship_program_links_relationship_idx
    ON third_party_relationship_program_links(tenant_id,legal_entity_id,relationship_id,state,updated_at DESC,id DESC);
CREATE INDEX third_party_relationship_program_links_target_idx
    ON third_party_relationship_program_links(tenant_id,legal_entity_id,program_id,state,updated_at DESC,id DESC);

CREATE TABLE third_party_relationship_matter_links (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    relationship_id uuid NOT NULL,
    matter_id uuid NOT NULL,
    purpose_code text NOT NULL CHECK (purpose_code ~ '^[A-Z][A-Z0-9_]{1,63}$'),
    purpose_label text NOT NULL CHECK (purpose_label=btrim(purpose_label) AND char_length(purpose_label) BETWEEN 1 AND 160),
    state text NOT NULL CHECK (state IN ('ACTIVE','ENDED')),
    created_by_principal_id uuid NOT NULL,
    ended_by_principal_id uuid,
    end_reason text NOT NULL DEFAULT '' CHECK (end_reason=btrim(end_reason) AND char_length(end_reason) <= 1000),
    version bigint NOT NULL CHECK (version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at),
    ended_at timestamptz,
    UNIQUE (id,tenant_id,legal_entity_id),
    FOREIGN KEY (relationship_id,tenant_id,legal_entity_id) REFERENCES third_party_relationships(id,tenant_id,legal_entity_id),
    FOREIGN KEY (matter_id,tenant_id) REFERENCES matters(id,tenant_id),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (created_by_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (ended_by_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK ((state='ACTIVE' AND ended_at IS NULL AND ended_by_principal_id IS NULL AND end_reason='') OR
           (state='ENDED' AND ended_at IS NOT NULL AND ended_by_principal_id IS NOT NULL AND end_reason<>''))
);
CREATE UNIQUE INDEX third_party_relationship_matter_links_active_idx
    ON third_party_relationship_matter_links(tenant_id,legal_entity_id,relationship_id,matter_id)
    WHERE state='ACTIVE';
CREATE INDEX third_party_relationship_matter_links_relationship_idx
    ON third_party_relationship_matter_links(tenant_id,legal_entity_id,relationship_id,state,updated_at DESC,id DESC);
CREATE INDEX third_party_relationship_matter_links_target_idx
    ON third_party_relationship_matter_links(tenant_id,legal_entity_id,matter_id,state,updated_at DESC,id DESC);

CREATE TABLE third_party_relationship_link_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    link_id uuid NOT NULL,
    relationship_id uuid NOT NULL,
    target_type text NOT NULL CHECK (target_type IN ('PROGRAM','MATTER')),
    target_id uuid NOT NULL,
    link_version bigint NOT NULL CHECK (link_version > 0),
    actor_principal_id uuid NOT NULL,
    event_type text NOT NULL CHECK (event_type IN ('VendorRelationshipLinked','VendorRelationshipLinkEnded')),
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload)='object'),
    occurred_at timestamptz NOT NULL,
    FOREIGN KEY (relationship_id,tenant_id,legal_entity_id) REFERENCES third_party_relationships(id,tenant_id,legal_entity_id),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (actor_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    UNIQUE (tenant_id,link_id,link_version)
);
CREATE INDEX third_party_relationship_link_events_history_idx
    ON third_party_relationship_link_events(tenant_id,legal_entity_id,link_id,link_version,id);

COMMIT;
