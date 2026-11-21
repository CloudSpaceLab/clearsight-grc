BEGIN;
CREATE TABLE program_adoption_question_draft_revisions (
 id uuid PRIMARY KEY,
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 legal_entity_id uuid NOT NULL REFERENCES legal_entities(id),
 program_id uuid NOT NULL,
 program_version bigint NOT NULL CHECK (program_version > 0),
 adoption_id uuid NOT NULL,
 adoption_version bigint NOT NULL CHECK (adoption_version > 0),
 edition_id uuid NOT NULL,
 form_revision_id uuid NOT NULL REFERENCES monitoring_form_templates(revision_id),
 form_id uuid NOT NULL,
 form_version bigint NOT NULL CHECK (form_version > 0),
 field_id text NOT NULL CHECK (length(field_id) BETWEEN 1 AND 80),
 version bigint NOT NULL CHECK (version > 0),
 mode text NOT NULL CHECK (mode = 'MANUAL'),
 rationale text NOT NULL DEFAULT '' CHECK (octet_length(rationale) <= 1000),
 actor_id uuid NOT NULL REFERENCES principals(id),
 idempotency_key text NOT NULL CHECK (length(idempotency_key) BETWEEN 16 AND 128),
 request_digest bytea NOT NULL CHECK (octet_length(request_digest) = 32),
 authority_rule_id text NOT NULL CHECK (length(btrim(authority_rule_id)) > 0),
 authority_policy_version text NOT NULL CHECK (length(btrim(authority_policy_version)) > 0),
 created_at timestamptz NOT NULL,
 FOREIGN KEY (adoption_id,tenant_id,legal_entity_id,program_id,edition_id) REFERENCES program_framework_adoptions(id,tenant_id,legal_entity_id,program_id,edition_id),
 UNIQUE (tenant_id,legal_entity_id,adoption_id,form_id,form_version,field_id,version),
 UNIQUE (tenant_id,legal_entity_id,actor_id,idempotency_key)
);
CREATE INDEX program_adoption_question_draft_scope_idx ON program_adoption_question_draft_revisions(tenant_id,legal_entity_id,adoption_id,form_id,form_version,field_id,version DESC);
CREATE FUNCTION prevent_adoption_question_draft_mutation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'Adoption question draft revisions are immutable'; END $$;
CREATE TRIGGER adoption_question_draft_immutable BEFORE UPDATE OR DELETE ON program_adoption_question_draft_revisions FOR EACH ROW EXECUTE FUNCTION prevent_adoption_question_draft_mutation();
COMMIT;
