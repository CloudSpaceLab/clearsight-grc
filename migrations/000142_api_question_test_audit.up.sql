BEGIN;
CREATE TABLE api_question_test_attempts (
 id uuid PRIMARY KEY,
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 legal_entity_id uuid NOT NULL REFERENCES legal_entities(id),
 principal_id uuid NOT NULL REFERENCES principals(id),
 form_id uuid NOT NULL,
 form_version bigint NOT NULL CHECK (form_version > 0),
 field_id text NOT NULL CHECK (length(field_id) BETWEEN 1 AND 80),
 mode text NOT NULL CHECK (mode IN ('LIVE', 'SAMPLE')),
 authority_rule_id text NOT NULL CHECK (length(authority_rule_id) BETWEEN 1 AND 128),
 authority_policy_version text NOT NULL CHECK (length(authority_policy_version) BETWEEN 1 AND 128),
 attempted_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX api_question_test_attempts_entity_rate ON api_question_test_attempts(tenant_id, legal_entity_id, attempted_at DESC);
CREATE INDEX api_question_test_attempts_actor_rate ON api_question_test_attempts(tenant_id, legal_entity_id, principal_id, attempted_at DESC);
CREATE TABLE api_question_test_outcomes (
 attempt_id uuid PRIMARY KEY REFERENCES api_question_test_attempts(id),
 outcome text NOT NULL CHECK (outcome IN ('VALID', 'INVALID', 'UNAVAILABLE')),
 completed_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE FUNCTION prevent_api_question_test_audit_mutation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'API question test audit is immutable'; END $$;
CREATE TRIGGER api_question_test_attempts_immutable BEFORE UPDATE OR DELETE ON api_question_test_attempts FOR EACH ROW EXECUTE FUNCTION prevent_api_question_test_audit_mutation();
CREATE TRIGGER api_question_test_outcomes_immutable BEFORE UPDATE OR DELETE ON api_question_test_outcomes FOR EACH ROW EXECUTE FUNCTION prevent_api_question_test_audit_mutation();
COMMIT;
