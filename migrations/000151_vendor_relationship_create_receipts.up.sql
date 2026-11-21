BEGIN;
CREATE TABLE third_party_relationship_create_receipts (
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE,
    actor_id uuid NOT NULL,
    idempotency_key text NOT NULL CHECK (length(idempotency_key) BETWEEN 1 AND 200),
    payload_digest text NOT NULL CHECK (length(payload_digest) = 64),
    relationship_id uuid NOT NULL REFERENCES third_party_relationships(id),
    result jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, legal_entity_id, idempotency_key)
);
COMMIT;
