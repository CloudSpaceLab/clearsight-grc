BEGIN;

ALTER TABLE principals
    ADD CONSTRAINT principals_id_tenant_key UNIQUE (id, tenant_id);
ALTER TABLE legal_entities
    ADD CONSTRAINT legal_entities_id_tenant_key UNIQUE (id, tenant_id);
ALTER TABLE evidence_sources
    ADD CONSTRAINT evidence_sources_id_tenant_key UNIQUE (id, tenant_id);
ALTER TABLE capture_requests
    ADD CONSTRAINT capture_requests_id_tenant_key UNIQUE (id, tenant_id);
ALTER TABLE capture_invitations
    ADD CONSTRAINT capture_invitations_id_tenant_request_key UNIQUE (id, tenant_id, request_id);
ALTER TABLE capture_sessions
    ADD CONSTRAINT capture_sessions_id_tenant_request_key UNIQUE (id, tenant_id, request_id);
ALTER TABLE capture_submissions
    ADD CONSTRAINT capture_submissions_id_tenant_request_key UNIQUE (id, tenant_id, request_id);

ALTER TABLE evidence_sources
    ADD CONSTRAINT evidence_sources_legal_entity_tenant_fk
        FOREIGN KEY (legal_entity_id, tenant_id) REFERENCES legal_entities(id, tenant_id),
    ADD CONSTRAINT evidence_sources_owner_tenant_fk
        FOREIGN KEY (owner_principal_id, tenant_id) REFERENCES principals(id, tenant_id);

ALTER TABLE source_observations
    ADD CONSTRAINT source_observations_source_tenant_fk
        FOREIGN KEY (source_id, tenant_id) REFERENCES evidence_sources(id, tenant_id),
    ADD CONSTRAINT source_observations_recorder_tenant_fk
        FOREIGN KEY (recorded_by, tenant_id) REFERENCES principals(id, tenant_id);

ALTER TABLE capture_requests
    ADD CONSTRAINT capture_requests_creator_tenant_fk
        FOREIGN KEY (created_by, tenant_id) REFERENCES principals(id, tenant_id);

ALTER TABLE capture_submissions
    ADD CONSTRAINT capture_submissions_request_tenant_fk
        FOREIGN KEY (request_id, tenant_id) REFERENCES capture_requests(id, tenant_id),
    ADD CONSTRAINT capture_submissions_submitter_tenant_fk
        FOREIGN KEY (submitted_by, tenant_id) REFERENCES principals(id, tenant_id),
    ADD CONSTRAINT capture_submissions_session_scope_fk
        FOREIGN KEY (session_id, tenant_id, request_id) REFERENCES capture_sessions(id, tenant_id, request_id);

ALTER TABLE capture_invitations
    ADD CONSTRAINT capture_invitations_request_tenant_fk
        FOREIGN KEY (request_id, tenant_id) REFERENCES capture_requests(id, tenant_id),
    ADD CONSTRAINT capture_invitations_creator_tenant_fk
        FOREIGN KEY (created_by, tenant_id) REFERENCES principals(id, tenant_id);

ALTER TABLE capture_sessions
    ADD CONSTRAINT capture_sessions_request_tenant_fk
        FOREIGN KEY (request_id, tenant_id) REFERENCES capture_requests(id, tenant_id),
    ADD CONSTRAINT capture_sessions_invitation_scope_fk
        FOREIGN KEY (invitation_id, tenant_id, request_id) REFERENCES capture_invitations(id, tenant_id, request_id);

ALTER TABLE capture_artifacts
    ADD CONSTRAINT capture_artifacts_request_tenant_fk
        FOREIGN KEY (request_id, tenant_id) REFERENCES capture_requests(id, tenant_id),
    ADD CONSTRAINT capture_artifacts_submission_scope_fk
        FOREIGN KEY (submission_id, tenant_id, request_id) REFERENCES capture_submissions(id, tenant_id, request_id),
    ADD CONSTRAINT capture_artifacts_creator_tenant_fk
        FOREIGN KEY (created_by, tenant_id) REFERENCES principals(id, tenant_id);

COMMIT;
