BEGIN;

CREATE INDEX IF NOT EXISTS org_positions_active_parent_idx
    ON org_positions(tenant_id, legal_entity_id, parent_position_id, code)
    WHERE valid_until IS NULL;

COMMIT;
