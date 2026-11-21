BEGIN;
CREATE TABLE form_question_binding_proposals (
 id uuid PRIMARY KEY,
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 legal_entity_id uuid NOT NULL REFERENCES legal_entities(id),
 draft_id uuid NOT NULL REFERENCES api_question_draft_revisions(id),
 draft_version bigint NOT NULL CHECK (draft_version > 0),
 form_revision_id uuid NOT NULL REFERENCES monitoring_form_templates(revision_id),
 form_id uuid NOT NULL,
 form_version bigint NOT NULL CHECK (form_version > 0),
 field_id text NOT NULL CHECK (length(field_id) BETWEEN 1 AND 80),
 binding_id uuid NOT NULL,
 source_id uuid NOT NULL,
 binding_version bigint NOT NULL CHECK (binding_version > 0),
 version bigint NOT NULL CHECK (version > 0),
 status text NOT NULL CHECK (status='PROPOSED'),
 actor_id uuid NOT NULL REFERENCES principals(id),
 idempotency_key text NOT NULL CHECK (length(idempotency_key) BETWEEN 16 AND 128),
 request_digest bytea NOT NULL CHECK (octet_length(request_digest)=32),
 authority_rule_id text NOT NULL CHECK (length(btrim(authority_rule_id))>0),
 authority_policy_version text NOT NULL CHECK (length(btrim(authority_policy_version))>0),
 created_at timestamptz NOT NULL,
 FOREIGN KEY (binding_id,tenant_id,source_id,binding_version) REFERENCES source_bindings(binding_id,tenant_id,source_id,version),
 UNIQUE (tenant_id,legal_entity_id,draft_id,version),
 UNIQUE (tenant_id,legal_entity_id,actor_id,idempotency_key)
);
CREATE INDEX form_question_binding_proposal_scope_idx ON form_question_binding_proposals(tenant_id,legal_entity_id,draft_id,version DESC);
CREATE FUNCTION prevent_form_question_binding_proposal_mutation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'Form question binding proposals are immutable'; END $$;
CREATE TRIGGER form_question_binding_proposal_immutable BEFORE UPDATE OR DELETE ON form_question_binding_proposals FOR EACH ROW EXECUTE FUNCTION prevent_form_question_binding_proposal_mutation();
COMMIT;
