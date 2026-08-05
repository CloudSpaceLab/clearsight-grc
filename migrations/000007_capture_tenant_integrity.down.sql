BEGIN;

ALTER TABLE capture_artifacts
    DROP CONSTRAINT IF EXISTS capture_artifacts_creator_tenant_fk,
    DROP CONSTRAINT IF EXISTS capture_artifacts_submission_scope_fk,
    DROP CONSTRAINT IF EXISTS capture_artifacts_request_tenant_fk;
ALTER TABLE capture_sessions
    DROP CONSTRAINT IF EXISTS capture_sessions_invitation_scope_fk,
    DROP CONSTRAINT IF EXISTS capture_sessions_request_tenant_fk;
ALTER TABLE capture_invitations
    DROP CONSTRAINT IF EXISTS capture_invitations_creator_tenant_fk,
    DROP CONSTRAINT IF EXISTS capture_invitations_request_tenant_fk;
ALTER TABLE capture_submissions
    DROP CONSTRAINT IF EXISTS capture_submissions_session_scope_fk,
    DROP CONSTRAINT IF EXISTS capture_submissions_submitter_tenant_fk,
    DROP CONSTRAINT IF EXISTS capture_submissions_request_tenant_fk;
ALTER TABLE capture_requests
    DROP CONSTRAINT IF EXISTS capture_requests_creator_tenant_fk;
ALTER TABLE source_observations
    DROP CONSTRAINT IF EXISTS source_observations_recorder_tenant_fk,
    DROP CONSTRAINT IF EXISTS source_observations_source_tenant_fk;
ALTER TABLE evidence_sources
    DROP CONSTRAINT IF EXISTS evidence_sources_owner_tenant_fk,
    DROP CONSTRAINT IF EXISTS evidence_sources_legal_entity_tenant_fk;

ALTER TABLE capture_submissions DROP CONSTRAINT IF EXISTS capture_submissions_id_tenant_request_key;
ALTER TABLE capture_sessions DROP CONSTRAINT IF EXISTS capture_sessions_id_tenant_request_key;
ALTER TABLE capture_invitations DROP CONSTRAINT IF EXISTS capture_invitations_id_tenant_request_key;
ALTER TABLE capture_requests DROP CONSTRAINT IF EXISTS capture_requests_id_tenant_key;
ALTER TABLE evidence_sources DROP CONSTRAINT IF EXISTS evidence_sources_id_tenant_key;
ALTER TABLE legal_entities DROP CONSTRAINT IF EXISTS legal_entities_id_tenant_key;
ALTER TABLE principals DROP CONSTRAINT IF EXISTS principals_id_tenant_key;

COMMIT;
