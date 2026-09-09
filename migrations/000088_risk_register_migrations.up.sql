BEGIN;
CREATE TABLE risk_register_migrations (
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 legal_entity_id uuid NOT NULL REFERENCES legal_entities(id),
 source_sha256 text NOT NULL CHECK (length(source_sha256)=64),
 document_id uuid NOT NULL REFERENCES document_imports(id),
 source_version bigint NOT NULL,
 version bigint NOT NULL CHECK(version>0),
 status text NOT NULL CHECK(status IN ('DRAFT','IMPORTED')),
 selection jsonb NOT NULL,
 receipts jsonb NOT NULL DEFAULT '[]'::jsonb,
 updated_by uuid NOT NULL REFERENCES principals(id),
 updated_at timestamptz NOT NULL,
 PRIMARY KEY(tenant_id,legal_entity_id,source_sha256)
);
CREATE TABLE risk_register_migration_revisions (
 tenant_id uuid NOT NULL,
 legal_entity_id uuid NOT NULL,
 source_sha256 text NOT NULL,
 version bigint NOT NULL,
 snapshot jsonb NOT NULL,
 PRIMARY KEY(tenant_id,legal_entity_id,source_sha256,version),
 FOREIGN KEY(tenant_id,legal_entity_id,source_sha256) REFERENCES risk_register_migrations(tenant_id,legal_entity_id,source_sha256)
);
CREATE FUNCTION protect_risk_register_revision() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'Risk register migration history is immutable'; END; $$;
CREATE TRIGGER risk_register_revision_immutable BEFORE UPDATE OR DELETE ON risk_register_migration_revisions FOR EACH ROW EXECUTE FUNCTION protect_risk_register_revision();
COMMIT;
