BEGIN;

-- Read indexes only: document occurrences remain immutable submitted answers.
CREATE INDEX capture_response_revisions_submission_idx
    ON capture_response_revisions(tenant_id,submission_id,legal_entity_id);
CREATE INDEX capture_submissions_document_order_idx
    ON capture_submissions(tenant_id,submitted_at DESC,id DESC) INCLUDE(request_id);
CREATE INDEX third_party_assessment_request_document_idx
    ON third_party_assessment_request_links(tenant_id,request_id,legal_entity_id);

COMMIT;
