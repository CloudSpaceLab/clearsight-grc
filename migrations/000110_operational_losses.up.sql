BEGIN;

CREATE TABLE operational_losses (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid NOT NULL,
    organization_scope_id uuid,
    code text NOT NULL CHECK (code=btrim(code) AND char_length(code) BETWEEN 1 AND 128),
    title text NOT NULL CHECK (title=btrim(title) AND char_length(title) BETWEEN 1 AND 240),
    event_type text NOT NULL CHECK (event_type IN (
        'INTERNAL_FRAUD','EXTERNAL_FRAUD','EMPLOYMENT_PRACTICES',
        'CLIENT_PRODUCTS_BUSINESS_PRACTICES','DAMAGE_TO_PHYSICAL_ASSETS',
        'BUSINESS_DISRUPTION_SYSTEM_FAILURES','EXECUTION_DELIVERY_PROCESS_MANAGEMENT','OTHER'
    )),
    cause text NOT NULL CHECK (cause=btrim(cause) AND char_length(cause) BETWEEN 1 AND 2000),
    description text NOT NULL DEFAULT '' CHECK (description=btrim(description) AND char_length(description)<=4000),
    gross_amount_minor bigint NOT NULL CHECK (gross_amount_minor>0),
    currency text NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    occurred_at timestamptz NOT NULL,
    discovered_at timestamptz NOT NULL,
    risk_id uuid,
    matter_id uuid,
    owner_principal_id uuid NOT NULL,
    status text NOT NULL CHECK (status IN ('ACTIVE','VOIDED')),
    version bigint NOT NULL CHECK (version>0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,code),
    UNIQUE(tenant_id,legal_entity_id,id),
    FOREIGN KEY(legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY(tenant_id,legal_entity_id,organization_scope_id)
        REFERENCES organization_scopes(tenant_id,legal_entity_id,id),
    FOREIGN KEY(tenant_id,legal_entity_id,risk_id)
        REFERENCES risks(tenant_id,legal_entity_id,id),
    FOREIGN KEY(matter_id,tenant_id,legal_entity_id)
        REFERENCES matters(id,tenant_id,legal_entity_id),
    FOREIGN KEY(owner_principal_id,tenant_id)
        REFERENCES principals(id,tenant_id),
    CHECK(discovered_at>=occurred_at),
    CHECK(updated_at>=created_at)
);

CREATE INDEX operational_losses_scope_updated_idx
    ON operational_losses(tenant_id,legal_entity_id,updated_at DESC,id DESC);
CREATE INDEX operational_losses_scope_event_idx
    ON operational_losses(tenant_id,legal_entity_id,event_type,occurred_at DESC,id DESC);
CREATE INDEX operational_losses_scope_org_idx
    ON operational_losses(tenant_id,legal_entity_id,organization_scope_id,occurred_at DESC,id DESC)
    WHERE organization_scope_id IS NOT NULL;
CREATE INDEX operational_losses_scope_risk_idx
    ON operational_losses(tenant_id,legal_entity_id,risk_id,occurred_at DESC,id DESC)
    WHERE risk_id IS NOT NULL;
CREATE INDEX operational_losses_scope_matter_idx
    ON operational_losses(tenant_id,legal_entity_id,matter_id,occurred_at DESC,id DESC)
    WHERE matter_id IS NOT NULL;

CREATE TABLE operational_loss_revisions (
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    loss_id uuid NOT NULL,
    loss_version bigint NOT NULL CHECK (loss_version>0),
    snapshot jsonb NOT NULL CHECK (jsonb_typeof(snapshot)='object'),
    recorded_at timestamptz NOT NULL,
    PRIMARY KEY(tenant_id,legal_entity_id,loss_id,loss_version),
    FOREIGN KEY(tenant_id,legal_entity_id,loss_id)
        REFERENCES operational_losses(tenant_id,legal_entity_id,id)
);

CREATE TABLE operational_loss_recoveries (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    loss_id uuid NOT NULL,
    loss_version bigint NOT NULL CHECK (loss_version>1),
    kind text NOT NULL CHECK (kind IN ('RECOVERY','REVERSAL')),
    amount_minor bigint NOT NULL CHECK (amount_minor>0),
    currency text NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    reference text NOT NULL DEFAULT '' CHECK (reference=btrim(reference) AND char_length(reference)<=500),
    recovered_at timestamptz NOT NULL,
    actor_id uuid,
    created_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,loss_id,loss_version),
    FOREIGN KEY(tenant_id,legal_entity_id,loss_id)
        REFERENCES operational_losses(tenant_id,legal_entity_id,id),
    FOREIGN KEY(actor_id,tenant_id) REFERENCES principals(id,tenant_id)
);

CREATE INDEX operational_loss_recoveries_history_idx
    ON operational_loss_recoveries(tenant_id,legal_entity_id,loss_id,recovered_at DESC,id DESC);

CREATE TABLE operational_loss_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    loss_id uuid NOT NULL,
    loss_version bigint NOT NULL CHECK (loss_version>0),
    event_type text NOT NULL CHECK (event_type=btrim(event_type) AND event_type<>''),
    actor_id uuid,
    occurred_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,loss_id,loss_version),
    FOREIGN KEY(tenant_id,legal_entity_id,loss_id)
        REFERENCES operational_losses(tenant_id,legal_entity_id,id),
    FOREIGN KEY(actor_id,tenant_id) REFERENCES principals(id,tenant_id)
);

CREATE INDEX operational_loss_events_history_idx
    ON operational_loss_events(tenant_id,legal_entity_id,loss_id,loss_version DESC);

CREATE FUNCTION protect_operational_loss_immutable() RETURNS trigger LANGUAGE plpgsql AS $loss$
BEGIN
    RAISE EXCEPTION 'operational loss history is immutable';
END;
$loss$;

CREATE TRIGGER operational_loss_revisions_immutable
    BEFORE UPDATE OR DELETE ON operational_loss_revisions
    FOR EACH ROW EXECUTE FUNCTION protect_operational_loss_immutable();
CREATE TRIGGER operational_loss_recoveries_immutable
    BEFORE UPDATE OR DELETE ON operational_loss_recoveries
    FOR EACH ROW EXECUTE FUNCTION protect_operational_loss_immutable();
CREATE TRIGGER operational_loss_events_immutable
    BEFORE UPDATE OR DELETE ON operational_loss_events
    FOR EACH ROW EXECUTE FUNCTION protect_operational_loss_immutable();

COMMIT;
