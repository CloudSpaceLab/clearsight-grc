BEGIN;

CREATE TABLE document_coverage_assessments (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid REFERENCES legal_entities(id),
    document_id uuid NOT NULL REFERENCES document_imports(id) ON DELETE CASCADE,
    document_sha256 text NOT NULL CHECK (document_sha256 ~ '^[0-9a-f]{64}$'),
    status text NOT NULL CHECK (status IN ('PENDING','COMPARING','READY','PARTIAL','FAILED')),
    analyzer_version text NOT NULL,
    matcher_version text NOT NULL,
    scoring_policy_version text NOT NULL,
    program_snapshot_hash text NOT NULL CHECK (program_snapshot_hash ~ '^[0-9a-f]{64}$'),
    metrics jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metrics)='object'),
    limitations jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(limitations)='array'),
    failure_message text NOT NULL DEFAULT '',
    assessed_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    UNIQUE (id,tenant_id),
    UNIQUE (tenant_id,document_id,document_sha256,analyzer_version,matcher_version,program_snapshot_hash)
);

CREATE TABLE document_coverage_candidates (
    assessment_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    candidate_id text NOT NULL,
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    fingerprint text NOT NULL CHECK (fingerprint ~ '^[0-9a-f]{64}$'),
    eligible boolean NOT NULL,
    classification text NOT NULL CHECK (classification IN ('VERIFIED_COVERAGE','MAPPED_NO_CURRENT_EVIDENCE','MAPPED_CONTROL_GAP','PARTIAL_MATCH','GAP','NEEDS_REVIEW','NOT_APPLICABLE')),
    candidate jsonb NOT NULL CHECK (jsonb_typeof(candidate)='object'),
    PRIMARY KEY (assessment_id,candidate_id),
    UNIQUE (assessment_id,tenant_id,candidate_id),
    UNIQUE (assessment_id,ordinal),
    FOREIGN KEY (assessment_id,tenant_id) REFERENCES document_coverage_assessments(id,tenant_id) ON DELETE CASCADE
);

CREATE TABLE document_coverage_matches (
    assessment_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    candidate_id text NOT NULL,
    match_id text NOT NULL,
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    target_program_id uuid NOT NULL,
    target_requirement_id uuid NOT NULL,
    score double precision NOT NULL CHECK (score >= 0 AND score <= 1),
    match jsonb NOT NULL CHECK (jsonb_typeof(match)='object'),
    PRIMARY KEY (assessment_id,candidate_id,match_id),
    UNIQUE (assessment_id,candidate_id,ordinal),
    FOREIGN KEY (assessment_id,tenant_id,candidate_id) REFERENCES document_coverage_candidates(assessment_id,tenant_id,candidate_id) ON DELETE CASCADE
);

CREATE TABLE document_coverage_reviews (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    assessment_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    candidate_id text NOT NULL,
    decision text NOT NULL CHECK (decision IN ('ACCEPT_MATCH','REJECT_MATCH','NOT_APPLICABLE')),
    match_id text,
    reason text NOT NULL DEFAULT '',
    reviewer_id uuid NOT NULL REFERENCES principals(id),
    reviewed_at timestamptz NOT NULL,
    FOREIGN KEY (assessment_id,tenant_id,candidate_id) REFERENCES document_coverage_candidates(assessment_id,tenant_id,candidate_id) ON DELETE CASCADE
);

CREATE TABLE document_coverage_suggestions (
    assessment_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    suggestion_id text NOT NULL,
    candidate_id text NOT NULL,
    suggestion_type text NOT NULL CHECK (suggestion_type IN ('LINK_REQUIREMENT','ADD_REQUIREMENT','CREATE_MATTER','CREATE_PROGRAM')),
    status text NOT NULL CHECK (status IN ('PROPOSED','DISMISSED','APPLIED','FAILED')),
    suggestion jsonb NOT NULL CHECK (jsonb_typeof(suggestion)='object'),
    applied_type text NOT NULL DEFAULT '',
    applied_id uuid,
    failure_message text NOT NULL DEFAULT '',
    PRIMARY KEY (assessment_id,suggestion_id),
    FOREIGN KEY (assessment_id,tenant_id,candidate_id) REFERENCES document_coverage_candidates(assessment_id,tenant_id,candidate_id) ON DELETE CASCADE
);

CREATE INDEX document_coverage_assessments_current_idx
    ON document_coverage_assessments(tenant_id,document_id,created_at DESC,id DESC);
CREATE INDEX document_coverage_candidates_page_idx
    ON document_coverage_candidates(assessment_id,ordinal,candidate_id);
CREATE INDEX document_coverage_matches_target_idx
    ON document_coverage_matches(tenant_id,target_program_id,target_requirement_id);
CREATE INDEX document_coverage_reviews_history_idx
    ON document_coverage_reviews(assessment_id,candidate_id,reviewed_at,id);
CREATE INDEX document_coverage_suggestions_current_idx
    ON document_coverage_suggestions(assessment_id,status,suggestion_type,candidate_id)
    WHERE status IN ('PROPOSED','FAILED');

COMMIT;
