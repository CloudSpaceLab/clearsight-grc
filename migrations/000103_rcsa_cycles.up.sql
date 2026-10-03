BEGIN;

CREATE UNIQUE INDEX risk_control_links_rcsa_scope_idx
    ON risk_control_links(id,tenant_id,legal_entity_id,risk_id);

CREATE TABLE rcsa_cycles (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid NOT NULL,
    code text NOT NULL CHECK (code=btrim(code) AND char_length(code) BETWEEN 1 AND 128),
    name text NOT NULL CHECK (name=btrim(name) AND char_length(name) BETWEEN 1 AND 240),
    form_template_id uuid NOT NULL,
    form_template_version bigint NOT NULL CHECK (form_template_version>0),
    period_start timestamptz NOT NULL,
    period_end timestamptz NOT NULL,
    due_at timestamptz NOT NULL,
    challenge_due_at timestamptz NOT NULL,
    created_by uuid NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,code),
    UNIQUE(id,tenant_id,legal_entity_id),
    FOREIGN KEY(legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY(tenant_id,form_template_id,form_template_version,legal_entity_id)
        REFERENCES monitoring_form_templates(tenant_id,id,version,legal_entity_id),
    FOREIGN KEY(created_by,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK(period_end>period_start),
    CHECK(due_at>=period_end),
    CHECK(challenge_due_at>=due_at)
);

CREATE INDEX rcsa_cycles_scope_period_idx
    ON rcsa_cycles(tenant_id,legal_entity_id,period_end DESC,id DESC);

CREATE TABLE rcsa_cycle_items (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    cycle_id uuid NOT NULL,
    risk_id uuid NOT NULL,
    risk_version bigint NOT NULL CHECK (risk_version>0),
    risk_code text NOT NULL CHECK (risk_code=btrim(risk_code) AND char_length(risk_code) BETWEEN 1 AND 128),
    risk_name text NOT NULL CHECK (risk_name=btrim(risk_name) AND char_length(risk_name) BETWEEN 1 AND 240),
    respondent_principal_id uuid NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,cycle_id,risk_id),
    UNIQUE(id,tenant_id,legal_entity_id),
    UNIQUE(id,tenant_id,legal_entity_id,risk_id),
    FOREIGN KEY(cycle_id,tenant_id,legal_entity_id)
        REFERENCES rcsa_cycles(id,tenant_id,legal_entity_id),
    FOREIGN KEY(tenant_id,legal_entity_id,risk_id,risk_version)
        REFERENCES risk_revisions(tenant_id,legal_entity_id,risk_id,risk_version),
    FOREIGN KEY(respondent_principal_id,tenant_id)
        REFERENCES principals(id,tenant_id)
);

CREATE INDEX rcsa_cycle_items_cycle_idx
    ON rcsa_cycle_items(tenant_id,legal_entity_id,cycle_id,risk_code,id);
CREATE INDEX rcsa_cycle_items_respondent_idx
    ON rcsa_cycle_items(tenant_id,legal_entity_id,respondent_principal_id,cycle_id,id);

CREATE TABLE rcsa_cycle_item_controls (
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    item_id uuid NOT NULL,
    risk_id uuid NOT NULL,
    control_link_id uuid NOT NULL,
    PRIMARY KEY(tenant_id,legal_entity_id,item_id,control_link_id),
    FOREIGN KEY(item_id,tenant_id,legal_entity_id,risk_id)
        REFERENCES rcsa_cycle_items(id,tenant_id,legal_entity_id,risk_id),
    FOREIGN KEY(control_link_id,tenant_id,legal_entity_id,risk_id)
        REFERENCES risk_control_links(id,tenant_id,legal_entity_id,risk_id)
);

CREATE TABLE rcsa_cycle_item_distributions (
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    item_id uuid NOT NULL,
    distribution_id uuid NOT NULL,
    issued_at timestamptz NOT NULL,
    PRIMARY KEY(tenant_id,legal_entity_id,item_id),
    UNIQUE(tenant_id,legal_entity_id,distribution_id),
    FOREIGN KEY(item_id,tenant_id,legal_entity_id)
        REFERENCES rcsa_cycle_items(id,tenant_id,legal_entity_id),
    FOREIGN KEY(distribution_id,tenant_id,legal_entity_id)
        REFERENCES capture_form_distributions(id,tenant_id,legal_entity_id)
);

CREATE FUNCTION protect_rcsa_immutable() RETURNS trigger LANGUAGE plpgsql AS $rcsa$
BEGIN
    RAISE EXCEPTION 'RCSA cycle history is immutable';
END;
$rcsa$;

CREATE TRIGGER rcsa_cycles_immutable
    BEFORE UPDATE OR DELETE ON rcsa_cycles
    FOR EACH ROW EXECUTE FUNCTION protect_rcsa_immutable();
CREATE TRIGGER rcsa_cycle_items_immutable
    BEFORE UPDATE OR DELETE ON rcsa_cycle_items
    FOR EACH ROW EXECUTE FUNCTION protect_rcsa_immutable();
CREATE TRIGGER rcsa_cycle_item_controls_immutable
    BEFORE UPDATE OR DELETE ON rcsa_cycle_item_controls
    FOR EACH ROW EXECUTE FUNCTION protect_rcsa_immutable();
CREATE TRIGGER rcsa_cycle_item_distributions_immutable
    BEFORE UPDATE OR DELETE ON rcsa_cycle_item_distributions
    FOR EACH ROW EXECUTE FUNCTION protect_rcsa_immutable();

COMMIT;
