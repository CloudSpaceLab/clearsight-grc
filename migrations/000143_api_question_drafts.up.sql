BEGIN;
CREATE TABLE api_question_draft_revisions (
 id uuid PRIMARY KEY,
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 legal_entity_id uuid NOT NULL REFERENCES legal_entities(id),
 form_revision_id uuid NOT NULL REFERENCES monitoring_form_templates(revision_id),
 form_id uuid NOT NULL,
 form_version bigint NOT NULL CHECK (form_version > 0),
 field_id text NOT NULL CHECK (length(field_id) BETWEEN 1 AND 80),
 version bigint NOT NULL CHECK (version > 0),
 mode text NOT NULL CHECK (mode = 'API'),
 endpoint text NOT NULL DEFAULT '',
 result_type text NOT NULL DEFAULT '',
 actor_id uuid NOT NULL REFERENCES principals(id),
 idempotency_key text NOT NULL CHECK (length(idempotency_key) BETWEEN 16 AND 128),
 request_digest bytea NOT NULL CHECK (octet_length(request_digest) = 32),
 authority_rule_id text NOT NULL CHECK (length(btrim(authority_rule_id)) > 0),
 authority_policy_version text NOT NULL CHECK (length(btrim(authority_policy_version)) > 0),
 created_at timestamptz NOT NULL,
 CHECK (length(endpoint) BETWEEN 1 AND 2048 AND result_type IN ('BOOLEAN','NUMBER')),
 UNIQUE (tenant_id, legal_entity_id, form_id, form_version, field_id, version),
 UNIQUE (tenant_id, legal_entity_id, actor_id, idempotency_key)
);
CREATE INDEX api_question_draft_scope_idx ON api_question_draft_revisions(tenant_id, legal_entity_id, form_id, form_version, field_id, version DESC);
CREATE FUNCTION prevent_api_question_draft_mutation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'API question draft revisions are immutable'; END $$;
CREATE TRIGGER api_question_draft_immutable BEFORE UPDATE OR DELETE ON api_question_draft_revisions FOR EACH ROW EXECUTE FUNCTION prevent_api_question_draft_mutation();
COMMIT;
