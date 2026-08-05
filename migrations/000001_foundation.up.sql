BEGIN;

CREATE TABLE tenants (id uuid PRIMARY KEY DEFAULT uuidv7(), slug text NOT NULL UNIQUE, name text NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp());
CREATE TABLE legal_entities (id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL REFERENCES tenants(id), code text NOT NULL, name text NOT NULL, jurisdiction text NOT NULL, valid_from timestamptz NOT NULL DEFAULT clock_timestamp(), valid_until timestamptz, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(), UNIQUE (tenant_id, code, valid_from));
CREATE INDEX legal_entities_tenant_active_idx ON legal_entities (tenant_id, code) WHERE valid_until IS NULL;

CREATE TABLE principals (id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL REFERENCES tenants(id), kind text NOT NULL CHECK (kind IN ('PERSON','TEAM','QUEUE','COMMITTEE','EXTERNAL_PARTY','SERVICE')), external_ref text, display_name text NOT NULL, status text NOT NULL DEFAULT 'ACTIVE', valid_from timestamptz NOT NULL DEFAULT clock_timestamp(), valid_until timestamptz, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp());
CREATE INDEX principals_tenant_kind_active_idx ON principals (tenant_id, kind, status) WHERE valid_until IS NULL;

CREATE TABLE responsibility_assignments (id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL REFERENCES tenants(id), legal_entity_id uuid REFERENCES legal_entities(id), principal_id uuid NOT NULL REFERENCES principals(id), responsibility text NOT NULL, object_type text NOT NULL, object_id uuid, scope jsonb NOT NULL DEFAULT '{}'::jsonb, priority integer NOT NULL DEFAULT 0, valid_from timestamptz NOT NULL, valid_until timestamptz, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(), policy_version text NOT NULL);
CREATE INDEX responsibility_resolution_idx ON responsibility_assignments (tenant_id, legal_entity_id, responsibility, object_type, object_id, priority DESC) WHERE valid_until IS NULL;
CREATE INDEX responsibility_scope_gin_idx ON responsibility_assignments USING gin (scope);

CREATE TABLE authority_grants (id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL REFERENCES tenants(id), legal_entity_id uuid REFERENCES legal_entities(id), principal_id uuid NOT NULL REFERENCES principals(id), decision_type text NOT NULL, limits jsonb NOT NULL DEFAULT '{}'::jsonb, valid_from timestamptz NOT NULL, valid_until timestamptz, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(), policy_version text NOT NULL);
CREATE INDEX authority_resolution_idx ON authority_grants (tenant_id, legal_entity_id, decision_type, principal_id) WHERE valid_until IS NULL;

CREATE TABLE workflow_instances (id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL REFERENCES tenants(id), kind text NOT NULL, subject_type text NOT NULL, subject_id uuid NOT NULL, state text NOT NULL, policy_version text NOT NULL, context_version bigint NOT NULL DEFAULT 1, due_at timestamptz, created_at timestamptz NOT NULL DEFAULT clock_timestamp(), updated_at timestamptz NOT NULL DEFAULT clock_timestamp(), version bigint NOT NULL DEFAULT 1);
CREATE INDEX workflow_queue_idx ON workflow_instances (tenant_id, state, due_at, updated_at DESC);

CREATE TABLE evidence_requests (id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL REFERENCES tenants(id), workflow_id uuid REFERENCES workflow_instances(id), purpose text NOT NULL, audience jsonb NOT NULL, sensitivity text NOT NULL, state text NOT NULL, schema_version text NOT NULL, request_schema jsonb NOT NULL, deadline timestamptz, submitted_at timestamptz, created_at timestamptz NOT NULL DEFAULT clock_timestamp(), updated_at timestamptz NOT NULL DEFAULT clock_timestamp(), version bigint NOT NULL DEFAULT 1);
CREATE INDEX evidence_requests_queue_idx ON evidence_requests (tenant_id, state, deadline);

CREATE TABLE invitation_grants (id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL REFERENCES tenants(id), request_id uuid NOT NULL REFERENCES evidence_requests(id), token_hash bytea NOT NULL UNIQUE, audience_hash bytea NOT NULL, expires_at timestamptz NOT NULL, redeemed_at timestamptz, revoked_at timestamptz, created_at timestamptz NOT NULL DEFAULT clock_timestamp());
CREATE INDEX invitation_active_idx ON invitation_grants (request_id, expires_at) WHERE redeemed_at IS NULL AND revoked_at IS NULL;

CREATE TABLE outbox_events (id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL REFERENCES tenants(id), aggregate_type text NOT NULL, aggregate_id uuid NOT NULL, event_type text NOT NULL, payload jsonb NOT NULL, occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(), available_at timestamptz NOT NULL DEFAULT clock_timestamp(), published_at timestamptz, attempts integer NOT NULL DEFAULT 0);
CREATE INDEX outbox_claim_idx ON outbox_events (available_at, id) WHERE published_at IS NULL;

CREATE TABLE audit_events (id uuid PRIMARY KEY DEFAULT uuidv7(), tenant_id uuid NOT NULL REFERENCES tenants(id), actor_id uuid, event_type text NOT NULL, subject_type text NOT NULL, subject_id uuid, purpose text NOT NULL, safe_metadata jsonb NOT NULL DEFAULT '{}'::jsonb, occurred_at timestamptz NOT NULL DEFAULT clock_timestamp());
CREATE INDEX audit_subject_time_idx ON audit_events (tenant_id, subject_type, subject_id, occurred_at DESC);

COMMIT;
