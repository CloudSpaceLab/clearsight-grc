BEGIN;

CREATE TABLE risks (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid NOT NULL,
    code text NOT NULL CHECK (code=btrim(code) AND char_length(code) BETWEEN 1 AND 128),
    name text NOT NULL CHECK (name=btrim(name) AND char_length(name) BETWEEN 1 AND 240),
    category text NOT NULL DEFAULT '' CHECK (category=btrim(category) AND char_length(category)<=160),
    statement text NOT NULL CHECK (statement=btrim(statement) AND char_length(statement) BETWEEN 1 AND 2000),
    cause text NOT NULL DEFAULT '' CHECK (cause=btrim(cause) AND char_length(cause)<=2000),
    event text NOT NULL DEFAULT '' CHECK (event=btrim(event) AND char_length(event)<=2000),
    impact text NOT NULL CHECK (impact=btrim(impact) AND char_length(impact) BETWEEN 1 AND 2000),
    scope jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(scope)='object'),
    owner_principal_id uuid,
    status text NOT NULL CHECK (status IN ('DRAFT','ACTIVE','RETIRED')),
    version bigint NOT NULL CHECK (version>0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,code),
    UNIQUE(tenant_id,legal_entity_id,id),
    FOREIGN KEY(legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY(owner_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK(updated_at>=created_at)
);

CREATE INDEX risks_scope_updated_idx
    ON risks(tenant_id,legal_entity_id,updated_at DESC,id DESC);
CREATE INDEX risks_scope_status_updated_idx
    ON risks(tenant_id,legal_entity_id,status,updated_at DESC,id DESC);
CREATE INDEX risks_scope_owner_updated_idx
    ON risks(tenant_id,legal_entity_id,owner_principal_id,updated_at DESC,id DESC);
CREATE INDEX risks_scope_category_updated_idx
    ON risks(tenant_id,legal_entity_id,lower(category),updated_at DESC,id DESC);

CREATE TABLE risk_revisions (
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    risk_id uuid NOT NULL,
    risk_version bigint NOT NULL CHECK (risk_version>0),
    snapshot jsonb NOT NULL CHECK (jsonb_typeof(snapshot)='object'),
    recorded_at timestamptz NOT NULL,
    PRIMARY KEY(tenant_id,legal_entity_id,risk_id,risk_version),
    FOREIGN KEY(tenant_id,legal_entity_id,risk_id) REFERENCES risks(tenant_id,legal_entity_id,id) ON DELETE CASCADE
);

CREATE TABLE risk_appetite_statements (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    risk_id uuid NOT NULL,
    risk_version bigint NOT NULL CHECK (risk_version>1),
    version bigint NOT NULL CHECK (version>0),
    statement text NOT NULL CHECK (statement=btrim(statement) AND char_length(statement) BETWEEN 1 AND 3000),
    rule jsonb NOT NULL CHECK (jsonb_typeof(rule)='object'),
    rationale text NOT NULL DEFAULT '' CHECK (rationale=btrim(rationale) AND char_length(rationale)<=3000),
    owner_principal_id uuid,
    authority_principal_id uuid,
    status text NOT NULL CHECK (status IN ('ACTIVE','RETIRED')),
    effective_from timestamptz NOT NULL,
    effective_until timestamptz,
    created_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,risk_id,version),
    UNIQUE(tenant_id,legal_entity_id,risk_id,risk_version),
    UNIQUE(tenant_id,legal_entity_id,risk_id,id),
    FOREIGN KEY(tenant_id,legal_entity_id,risk_id) REFERENCES risks(tenant_id,legal_entity_id,id) ON DELETE CASCADE,
    FOREIGN KEY(owner_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY(authority_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK(effective_until IS NULL OR effective_until>effective_from)
);

CREATE INDEX risk_appetite_current_idx
    ON risk_appetite_statements(tenant_id,legal_entity_id,risk_id,version DESC,effective_from DESC);

CREATE TABLE risk_assessments (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    risk_id uuid NOT NULL,
    risk_version bigint NOT NULL CHECK (risk_version>1),
    assessment_kind text NOT NULL CHECK (assessment_kind IN ('INHERENT','CURRENT','RESIDUAL','TARGET','STRESSED','ACCEPTED')),
    method_code text NOT NULL CHECK (method_code=btrim(method_code) AND char_length(method_code) BETWEEN 1 AND 128),
    method_version text NOT NULL CHECK (method_version=btrim(method_version) AND char_length(method_version) BETWEEN 1 AND 128),
    dimensions jsonb NOT NULL CHECK (jsonb_typeof(dimensions)='object'),
    assumptions jsonb NOT NULL CHECK (jsonb_typeof(assumptions)='object'),
    evidence_references jsonb NOT NULL CHECK (jsonb_typeof(evidence_references)='array'),
    confidence double precision CHECK (confidence>=0 AND confidence<=1),
    assessed_by uuid,
    appetite_statement_id uuid,
    appetite_position text NOT NULL CHECK (appetite_position IN ('WITHIN','APPROACHING','BREACHED','UNKNOWN')),
    appetite_rationale text NOT NULL DEFAULT '' CHECK (appetite_rationale=btrim(appetite_rationale) AND char_length(appetite_rationale)<=3000),
    assessed_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,risk_id,risk_version),
    FOREIGN KEY(tenant_id,legal_entity_id,risk_id) REFERENCES risks(tenant_id,legal_entity_id,id) ON DELETE CASCADE,
    FOREIGN KEY(tenant_id,legal_entity_id,risk_id,appetite_statement_id)
        REFERENCES risk_appetite_statements(tenant_id,legal_entity_id,risk_id,id),
    FOREIGN KEY(assessed_by,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK((appetite_statement_id IS NULL AND appetite_position='UNKNOWN') OR appetite_statement_id IS NOT NULL)
);

CREATE INDEX risk_assessments_latest_idx
    ON risk_assessments(tenant_id,legal_entity_id,risk_id,risk_version DESC,id DESC);
CREATE INDEX risk_assessments_position_idx
    ON risk_assessments(tenant_id,legal_entity_id,appetite_position,risk_version DESC,risk_id);

CREATE TABLE risk_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    risk_id uuid NOT NULL,
    risk_version bigint NOT NULL CHECK (risk_version>0),
    event_type text NOT NULL CHECK (event_type=btrim(event_type) AND event_type<>''),
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload)='object'),
    actor_id uuid,
    occurred_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,risk_id,risk_version),
    FOREIGN KEY(tenant_id,legal_entity_id,risk_id) REFERENCES risks(tenant_id,legal_entity_id,id) ON DELETE CASCADE,
    FOREIGN KEY(actor_id,tenant_id) REFERENCES principals(id,tenant_id)
);

CREATE INDEX risk_events_history_idx
    ON risk_events(tenant_id,legal_entity_id,risk_id,risk_version DESC);

CREATE FUNCTION protect_risk_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'risk history is immutable';
END;
$$;

CREATE TRIGGER risk_revisions_immutable
    BEFORE UPDATE OR DELETE ON risk_revisions
    FOR EACH ROW EXECUTE FUNCTION protect_risk_immutable();
CREATE TRIGGER risk_appetite_immutable
    BEFORE UPDATE OR DELETE ON risk_appetite_statements
    FOR EACH ROW EXECUTE FUNCTION protect_risk_immutable();
CREATE TRIGGER risk_assessments_immutable
    BEFORE UPDATE OR DELETE ON risk_assessments
    FOR EACH ROW EXECUTE FUNCTION protect_risk_immutable();
CREATE TRIGGER risk_events_immutable
    BEFORE UPDATE OR DELETE ON risk_events
    FOR EACH ROW EXECUTE FUNCTION protect_risk_immutable();

COMMIT;
