BEGIN;

CREATE UNIQUE INDEX programs_scope_identity_idx
    ON programs(id,tenant_id,legal_entity_id);

CREATE TABLE control_definitions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    code text NOT NULL CHECK (code=btrim(code) AND char_length(code) BETWEEN 1 AND 128),
    name text NOT NULL CHECK (name=btrim(name) AND char_length(name) BETWEEN 1 AND 240),
    objective text NOT NULL CHECK (objective=btrim(objective) AND char_length(objective) BETWEEN 1 AND 2000),
    description text NOT NULL DEFAULT '' CHECK (description=btrim(description) AND char_length(description)<=4000),
    category text NOT NULL DEFAULT '' CHECK (category=btrim(category) AND char_length(category)<=160),
    status text NOT NULL CHECK (status IN ('ACTIVE','RETIRED')),
    version bigint NOT NULL CHECK (version>0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE(tenant_id,code),
    UNIQUE(id,tenant_id),
    CHECK(updated_at>=created_at)
);

CREATE INDEX control_definitions_status_idx
    ON control_definitions(tenant_id,status,lower(category),code,id);

CREATE TABLE control_catalog_implementation_links (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    definition_id uuid NOT NULL,
    program_id uuid NOT NULL,
    implementation_id uuid NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,implementation_id),
    UNIQUE(tenant_id,definition_id,implementation_id),
    UNIQUE(id,tenant_id,legal_entity_id),
    FOREIGN KEY(definition_id,tenant_id)
        REFERENCES control_definitions(id,tenant_id),
    FOREIGN KEY(legal_entity_id,tenant_id)
        REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY(program_id,tenant_id,legal_entity_id)
        REFERENCES programs(id,tenant_id,legal_entity_id),
    FOREIGN KEY(implementation_id,tenant_id,program_id)
        REFERENCES control_implementations(id,tenant_id,program_id)
);

CREATE INDEX control_catalog_definition_links_idx
    ON control_catalog_implementation_links(tenant_id,definition_id,created_at,id);
CREATE INDEX control_catalog_entity_links_idx
    ON control_catalog_implementation_links(tenant_id,legal_entity_id,created_at,id);

CREATE TABLE risk_control_links (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    risk_id uuid NOT NULL,
    risk_version bigint NOT NULL CHECK (risk_version>1),
    catalog_link_id uuid NOT NULL,
    linked_by uuid,
    created_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,risk_id,catalog_link_id),
    UNIQUE(tenant_id,legal_entity_id,risk_id,risk_version),
    FOREIGN KEY(tenant_id,legal_entity_id,risk_id)
        REFERENCES risks(tenant_id,legal_entity_id,id),
    FOREIGN KEY(catalog_link_id,tenant_id,legal_entity_id)
        REFERENCES control_catalog_implementation_links(id,tenant_id,legal_entity_id),
    FOREIGN KEY(linked_by,tenant_id)
        REFERENCES principals(id,tenant_id)
);

CREATE INDEX risk_control_links_risk_idx
    ON risk_control_links(tenant_id,legal_entity_id,risk_id,risk_version DESC,id DESC);

CREATE FUNCTION protect_control_catalog_immutable() RETURNS trigger LANGUAGE plpgsql AS $
BEGIN
    RAISE EXCEPTION 'control catalog history is immutable';
END;
$;

CREATE TRIGGER control_definitions_immutable
    BEFORE UPDATE OR DELETE ON control_definitions
    FOR EACH ROW EXECUTE FUNCTION protect_control_catalog_immutable();
CREATE TRIGGER control_catalog_links_immutable
    BEFORE UPDATE OR DELETE ON control_catalog_implementation_links
    FOR EACH ROW EXECUTE FUNCTION protect_control_catalog_immutable();
CREATE TRIGGER risk_control_links_immutable
    BEFORE UPDATE OR DELETE ON risk_control_links
    FOR EACH ROW EXECUTE FUNCTION protect_risk_immutable();

COMMIT;
