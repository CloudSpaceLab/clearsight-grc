BEGIN;

CREATE TABLE rcsa_cycles (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    code text NOT NULL CHECK (code=btrim(code) AND char_length(code) BETWEEN 1 AND 128),
    name text NOT NULL CHECK (name=btrim(name) AND char_length(name) BETWEEN 1 AND 240),
    trigger_kind text NOT NULL CHECK (trigger_kind IN ('SCHEDULED','CHANGE','MANUAL')),
    first_line_owner_principal_id uuid NOT NULL,
    status text NOT NULL CHECK (status IN ('DRAFT','ASSESSMENT_OPEN','AWAITING_CHALLENGE','COMPLETED','CANCELLED')),
    population_checksum text NOT NULL CHECK (population_checksum ~ '^[0-9a-f]{64}$'),
    first_line_distribution_id uuid,
    challenge_matter_id uuid,
    version bigint NOT NULL CHECK (version>0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,code),
    UNIQUE(tenant_id,legal_entity_id,id),
    FOREIGN KEY(legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY(first_line_owner_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY(first_line_distribution_id,tenant_id,legal_entity_id)
        REFERENCES capture_form_distributions(id,tenant_id,legal_entity_id),
    FOREIGN KEY(challenge_matter_id,tenant_id,legal_entity_id)
        REFERENCES matters(id,tenant_id,legal_entity_id),
    CHECK(updated_at>=created_at)
);

CREATE INDEX rcsa_cycles_scope_status_idx
    ON rcsa_cycles(tenant_id,legal_entity_id,status,updated_at DESC,id DESC);

CREATE TABLE rcsa_cycle_risks (
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    cycle_id uuid NOT NULL,
    risk_id uuid NOT NULL,
    risk_version bigint NOT NULL CHECK (risk_version>0),
    code text NOT NULL,
    name text NOT NULL,
    category text NOT NULL DEFAULT '',
    PRIMARY KEY(tenant_id,legal_entity_id,cycle_id,risk_id),
    FOREIGN KEY(tenant_id,legal_entity_id,cycle_id)
        REFERENCES rcsa_cycles(tenant_id,legal_entity_id,id) ON DELETE CASCADE,
    FOREIGN KEY(tenant_id,legal_entity_id,risk_id,risk_version)
        REFERENCES risk_revisions(tenant_id,legal_entity_id,risk_id,risk_version)
);

CREATE UNIQUE INDEX risk_control_links_rcsa_scope_idx
    ON risk_control_links(id,tenant_id,legal_entity_id,risk_id,risk_version);

CREATE TABLE rcsa_cycle_controls (
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    cycle_id uuid NOT NULL,
    risk_id uuid NOT NULL,
    risk_version bigint NOT NULL CHECK (risk_version>0),
    risk_control_link_id uuid NOT NULL,
    catalog_link_id uuid NOT NULL,
    definition_id uuid NOT NULL,
    definition_code text NOT NULL,
    definition_name text NOT NULL,
    program_id uuid NOT NULL,
    implementation_id uuid NOT NULL,
    implementation_version bigint NOT NULL CHECK (implementation_version>0),
    implementation_name text NOT NULL,
    PRIMARY KEY(tenant_id,legal_entity_id,cycle_id,risk_control_link_id),
    FOREIGN KEY(tenant_id,legal_entity_id,cycle_id,risk_id)
        REFERENCES rcsa_cycle_risks(tenant_id,legal_entity_id,cycle_id,risk_id) ON DELETE CASCADE,
    FOREIGN KEY(risk_control_link_id,tenant_id,legal_entity_id,risk_id,risk_version)
        REFERENCES risk_control_links(id,tenant_id,legal_entity_id,risk_id,risk_version),
    FOREIGN KEY(catalog_link_id,tenant_id,legal_entity_id)
        REFERENCES control_catalog_implementation_links(id,tenant_id,legal_entity_id),
    FOREIGN KEY(definition_id,tenant_id)
        REFERENCES control_definitions(id,tenant_id),
    FOREIGN KEY(program_id,tenant_id,legal_entity_id)
        REFERENCES programs(id,tenant_id,legal_entity_id),
    FOREIGN KEY(implementation_id,tenant_id,program_id)
        REFERENCES control_implementations(id,tenant_id,program_id)
);

CREATE TABLE rcsa_cycle_revisions (
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    cycle_id uuid NOT NULL,
    cycle_version bigint NOT NULL CHECK (cycle_version>0),
    snapshot jsonb NOT NULL CHECK (jsonb_typeof(snapshot)='object'),
    recorded_at timestamptz NOT NULL,
    PRIMARY KEY(tenant_id,legal_entity_id,cycle_id,cycle_version),
    FOREIGN KEY(tenant_id,legal_entity_id,cycle_id)
        REFERENCES rcsa_cycles(tenant_id,legal_entity_id,id) ON DELETE CASCADE
);

CREATE TABLE rcsa_cycle_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    cycle_id uuid NOT NULL,
    cycle_version bigint NOT NULL CHECK (cycle_version>0),
    event_type text NOT NULL CHECK (event_type=btrim(event_type) AND event_type<>''),
    actor_id uuid,
    occurred_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,cycle_id,cycle_version),
    FOREIGN KEY(tenant_id,legal_entity_id,cycle_id)
        REFERENCES rcsa_cycles(tenant_id,legal_entity_id,id) ON DELETE CASCADE,
    FOREIGN KEY(actor_id,tenant_id) REFERENCES principals(id,tenant_id)
);

CREATE FUNCTION protect_rcsa_immutable() RETURNS trigger LANGUAGE plpgsql AS $rcsa$
BEGIN
    RAISE EXCEPTION 'RCSA history is immutable';
END;
$rcsa$;

CREATE TRIGGER rcsa_cycle_risks_immutable
    BEFORE UPDATE OR DELETE ON rcsa_cycle_risks
    FOR EACH ROW EXECUTE FUNCTION protect_rcsa_immutable();
CREATE TRIGGER rcsa_cycle_controls_immutable
    BEFORE UPDATE OR DELETE ON rcsa_cycle_controls
    FOR EACH ROW EXECUTE FUNCTION protect_rcsa_immutable();
CREATE TRIGGER rcsa_cycle_revisions_immutable
    BEFORE UPDATE OR DELETE ON rcsa_cycle_revisions
    FOR EACH ROW EXECUTE FUNCTION protect_rcsa_immutable();
CREATE TRIGGER rcsa_cycle_events_immutable
    BEFORE UPDATE OR DELETE ON rcsa_cycle_events
    FOR EACH ROW EXECUTE FUNCTION protect_rcsa_immutable();

COMMIT;
