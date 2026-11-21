BEGIN;
CREATE TABLE capture_response_reviews (
 id uuid PRIMARY KEY,
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 legal_entity_id uuid NOT NULL REFERENCES legal_entities(id),
 response_revision_id uuid NOT NULL REFERENCES capture_response_revisions(id),
 response_revision bigint NOT NULL CHECK(response_revision > 0),
 version bigint NOT NULL CHECK(version > 0),
 decision text NOT NULL CHECK(decision IN ('REVIEWED','CHANGES_REQUESTED')),
 comment text NOT NULL CHECK(length(comment) <= 4000 AND (decision <> 'CHANGES_REQUESTED' OR length(btrim(comment)) > 0)),
 reviewer_id uuid NOT NULL REFERENCES principals(id),
 authority_route text NOT NULL CHECK(length(btrim(authority_route)) > 0),
 reviewed_at timestamptz NOT NULL,
 supersedes_id uuid REFERENCES capture_response_reviews(id),
 UNIQUE(tenant_id,response_revision_id,version)
);
CREATE INDEX capture_response_reviews_scoped_latest ON capture_response_reviews(tenant_id,legal_entity_id,response_revision_id,version DESC);
CREATE TRIGGER capture_response_reviews_immutable BEFORE UPDATE OR DELETE ON capture_response_reviews FOR EACH ROW EXECUTE FUNCTION prevent_response_assessment_mutation();
CREATE UNIQUE INDEX capture_response_review_outbox_uq ON outbox_events(tenant_id,aggregate_id,event_type,((payload->>'version')::bigint)) WHERE aggregate_type='FORM_RESPONSE_REVIEW';
COMMIT;
