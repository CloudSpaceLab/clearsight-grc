BEGIN;

CREATE UNIQUE INDEX monitoring_checks_indicator_scope_idx
    ON monitoring_checks(tenant_id,id,version,program_id);

CREATE TABLE risk_indicator_links (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    risk_id uuid NOT NULL,
    risk_version bigint NOT NULL CHECK (risk_version>1),
    program_id uuid NOT NULL,
    monitoring_check_id uuid NOT NULL,
    monitoring_check_version bigint NOT NULL CHECK (monitoring_check_version>0),
    kind text NOT NULL CHECK (kind IN ('KRI','KCI')),
    measurement text NOT NULL CHECK (measurement='MONITORING_RISK_SCORE'),
    linked_by uuid,
    created_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,risk_id,risk_version),
    UNIQUE(tenant_id,legal_entity_id,risk_id,monitoring_check_id,monitoring_check_version),
    FOREIGN KEY(tenant_id,legal_entity_id,risk_id)
        REFERENCES risks(tenant_id,legal_entity_id,id),
    FOREIGN KEY(program_id,tenant_id,legal_entity_id)
        REFERENCES programs(id,tenant_id,legal_entity_id),
    FOREIGN KEY(tenant_id,monitoring_check_id,monitoring_check_version,program_id)
        REFERENCES monitoring_checks(tenant_id,id,version,program_id),
    FOREIGN KEY(linked_by,tenant_id)
        REFERENCES principals(id,tenant_id)
);

CREATE INDEX risk_indicator_links_risk_idx
    ON risk_indicator_links(tenant_id,legal_entity_id,risk_id,risk_version DESC,id DESC);

CREATE TRIGGER risk_indicator_links_immutable
    BEFORE UPDATE OR DELETE ON risk_indicator_links
    FOR EACH ROW EXECUTE FUNCTION protect_risk_immutable();

COMMIT;
