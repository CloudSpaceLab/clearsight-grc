BEGIN;

-- This table is a fence primitive only. It does not yet cover authority writers.
CREATE TABLE tenant_authority_epochs (
    tenant_id uuid PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0)
);

CREATE FUNCTION create_tenant_authority_epoch() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO tenant_authority_epochs (tenant_id) VALUES (NEW.id);
    RETURN NEW;
END;
$$;

-- Create the trigger before the backfill so concurrent tenant creation is
-- fenced by the migration transaction's DDL lock on tenants.
CREATE TRIGGER tenant_authority_epoch_after_insert
AFTER INSERT ON tenants FOR EACH ROW EXECUTE FUNCTION create_tenant_authority_epoch();

INSERT INTO tenant_authority_epochs (tenant_id)
SELECT id FROM tenants ON CONFLICT (tenant_id) DO NOTHING;

COMMIT;
