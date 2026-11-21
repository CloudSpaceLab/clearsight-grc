BEGIN;
CREATE TABLE program_adoption_question_approvals (
 id uuid PRIMARY KEY,
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 legal_entity_id uuid NOT NULL REFERENCES legal_entities(id),
 draft_id uuid NOT NULL REFERENCES program_adoption_question_draft_revisions(id),
 draft_version bigint NOT NULL CHECK (draft_version > 0),
 program_id uuid NOT NULL,
 adoption_id uuid NOT NULL,
 adoption_version bigint NOT NULL CHECK (adoption_version > 0),
 reviewer_id uuid NOT NULL REFERENCES principals(id),
 rationale text NOT NULL CHECK (octet_length(rationale) BETWEEN 1 AND 1000 AND length(btrim(rationale)) > 0),
 idempotency_key text NOT NULL CHECK (length(idempotency_key) BETWEEN 16 AND 128),
 request_digest bytea NOT NULL CHECK (octet_length(request_digest) = 32),
 authority_rule_id text NOT NULL CHECK (length(btrim(authority_rule_id)) > 0),
 authority_policy_version text NOT NULL CHECK (length(btrim(authority_policy_version)) > 0),
 reviewed_at timestamptz NOT NULL,
 UNIQUE (id,tenant_id,legal_entity_id),
 UNIQUE (tenant_id,legal_entity_id,draft_id),
 UNIQUE (tenant_id,legal_entity_id,reviewer_id,idempotency_key)
);
CREATE INDEX adoption_question_approval_scope_idx ON program_adoption_question_approvals(tenant_id,legal_entity_id,adoption_id,draft_id);
CREATE FUNCTION prevent_adoption_question_approval_mutation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'Adoption question approvals are immutable'; END $$;
CREATE TRIGGER adoption_question_approval_immutable BEFORE UPDATE OR DELETE ON program_adoption_question_approvals FOR EACH ROW EXECUTE FUNCTION prevent_adoption_question_approval_mutation();
COMMIT;
