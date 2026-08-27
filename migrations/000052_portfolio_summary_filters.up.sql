BEGIN;

CREATE INDEX programs_portfolio_filter_idx
    ON programs(tenant_id,legal_entity_id,status,lower(btrim(jurisdiction)),updated_at DESC,id DESC);

CREATE INDEX matters_portfolio_filter_idx
    ON matters(tenant_id,legal_entity_id,matter_type,status,priority DESC,updated_at DESC,id DESC);

CREATE INDEX matters_owner_portfolio_idx
    ON matters(tenant_id,legal_entity_id,owner_principal_id,status,priority DESC,updated_at DESC,id DESC)
    WHERE owner_principal_id IS NOT NULL;

CREATE INDEX matters_due_portfolio_idx
    ON matters(tenant_id,legal_entity_id,due_at,priority DESC,updated_at DESC,id DESC)
    WHERE due_at IS NOT NULL AND status NOT IN ('CLOSED','CANCELLED');

COMMIT;
