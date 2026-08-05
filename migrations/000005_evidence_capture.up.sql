BEGIN;

CREATE TABLE evidence_sources (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid REFERENCES legal_entities(id),
    code text NOT NULL,
    name text NOT NULL,
    source_type text NOT NULL CHECK (source_type IN ('REGULATORY','SYSTEM','DOCUMENT','HUMAN','VENDOR')),
    authority_class text NOT NULL,
    owner_principal_id uuid REFERENCES principals(id),
    endpoint text NOT NULL DEFAULT '',
    expected_freshness_minutes integer NOT NULL CHECK (expected_freshness_minutes BETWEEN 1 AND 525600),
    last_observed_at timestamptz,
    last_success_at timestamptz,
    health text NOT NULL DEFAULT 'UNKNOWN' CHECK (health IN ('UNKNOWN','CURRENT','DEGRADED','STALE','UNAVAILABLE')),
    status text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','PAUSED','RETIRED')),
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE UNIQUE INDEX evidence_sources_active_code_idx ON evidence_sources(tenant_id,code) WHERE status<>'RETIRED';
CREATE INDEX evidence_sources_health_idx ON evidence_sources(tenant_id,status,health,last_success_at);
CREATE INDEX evidence_sources_owner_idx ON evidence_sources(tenant_id,owner_principal_id) WHERE status='ACTIVE';

CREATE TABLE source_observations (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    source_id uuid NOT NULL REFERENCES evidence_sources(id),
    observed_at timestamptz NOT NULL,
    success boolean NOT NULL,
    unavailable boolean NOT NULL DEFAULT false,
    latency_ms integer NOT NULL DEFAULT 0 CHECK (latency_ms >= 0),
    detail text NOT NULL DEFAULT '',
    recorded_by uuid REFERENCES principals(id),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (NOT (success AND unavailable))
);
CREATE INDEX source_observations_history_idx ON source_observations(tenant_id,source_id,observed_at DESC,id DESC);

CREATE TABLE capture_requests (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    subject_type text NOT NULL,
    subject_id text NOT NULL,
    title text NOT NULL,
    purpose text NOT NULL,
    why_you text NOT NULL,
    sensitivity text NOT NULL,
    audience_type text NOT NULL CHECK (audience_type IN ('INTERNAL','EXTERNAL','CUSTOMER','VENDOR','AUTHORITY')),
    estimated_minutes integer NOT NULL CHECK (estimated_minutes BETWEEN 1 AND 60),
    deadline timestamptz NOT NULL,
    known_facts jsonb NOT NULL DEFAULT '{}'::jsonb,
    fields jsonb NOT NULL,
    status text NOT NULL DEFAULT 'READY' CHECK (status IN ('DRAFT','READY','IN_PROGRESS','SUBMITTED','CANCELLED','EXPIRED')),
    created_by uuid REFERENCES principals(id),
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (jsonb_typeof(known_facts)='object'),
    CHECK (jsonb_typeof(fields)='array'),
    CHECK (jsonb_array_length(fields) BETWEEN 1 AND 50)
);
CREATE INDEX capture_requests_queue_idx ON capture_requests(tenant_id,status,deadline,id);
CREATE INDEX capture_requests_subject_idx ON capture_requests(tenant_id,subject_type,subject_id,created_at DESC);

CREATE TABLE capture_submissions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    request_id uuid NOT NULL REFERENCES capture_requests(id),
    session_id uuid,
    submitted_by uuid REFERENCES principals(id),
    channel text NOT NULL CHECK (channel IN ('INTERNAL','MAGIC_LINK','API')),
    answers jsonb NOT NULL,
    submitted_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (jsonb_typeof(answers)='object')
);
CREATE INDEX capture_submissions_request_idx ON capture_submissions(tenant_id,request_id,submitted_at DESC,id);

CREATE TABLE capture_invitations (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    request_id uuid NOT NULL REFERENCES capture_requests(id),
    token_hash bytea NOT NULL,
    audience_hash bytea NOT NULL,
    audience_hint text NOT NULL,
    purpose text NOT NULL,
    expires_at timestamptz NOT NULL,
    max_redemptions integer NOT NULL DEFAULT 1 CHECK (max_redemptions BETWEEN 1 AND 5),
    redemptions integer NOT NULL DEFAULT 0 CHECK (redemptions >= 0),
    last_redeemed_at timestamptz,
    revoked_at timestamptz,
    created_by uuid REFERENCES principals(id),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(token_hash),
    CHECK (redemptions <= max_redemptions),
    CHECK (expires_at > created_at)
);
CREATE INDEX capture_invitations_request_idx ON capture_invitations(tenant_id,request_id,expires_at DESC);
CREATE INDEX capture_invitations_expiry_idx ON capture_invitations(expires_at) WHERE revoked_at IS NULL AND redemptions < max_redemptions;

CREATE TABLE capture_sessions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    request_id uuid NOT NULL REFERENCES capture_requests(id),
    invitation_id uuid NOT NULL REFERENCES capture_invitations(id),
    token_hash bytea NOT NULL,
    audience_hint text NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    last_seen_at timestamptz,
    UNIQUE(token_hash),
    CHECK (expires_at > created_at)
);
CREATE INDEX capture_sessions_request_idx ON capture_sessions(tenant_id,request_id,expires_at DESC);
CREATE INDEX capture_sessions_expiry_idx ON capture_sessions(expires_at) WHERE revoked_at IS NULL;

ALTER TABLE capture_submissions
    ADD CONSTRAINT capture_submissions_session_fk FOREIGN KEY(session_id) REFERENCES capture_sessions(id);

CREATE TABLE capture_artifacts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    request_id uuid NOT NULL REFERENCES capture_requests(id),
    submission_id uuid REFERENCES capture_submissions(id),
    file_name text NOT NULL,
    media_type text NOT NULL,
    size_bytes bigint NOT NULL CHECK (size_bytes >= 0),
    sha256 text NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    storage_key text NOT NULL,
    status text NOT NULL DEFAULT 'STORED_UNSCANNED' CHECK (status IN ('STORED_UNSCANNED','AVAILABLE','QUARANTINED','DELETED')),
    created_by uuid REFERENCES principals(id),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    inspected_at timestamptz,
    inspection_reference text NOT NULL DEFAULT '',
    UNIQUE(tenant_id,storage_key)
);
CREATE INDEX capture_artifacts_request_idx ON capture_artifacts(tenant_id,request_id,created_at DESC,id);
CREATE INDEX capture_artifacts_status_idx ON capture_artifacts(tenant_id,status,created_at);

COMMIT;
