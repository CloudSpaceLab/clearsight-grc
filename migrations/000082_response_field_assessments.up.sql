-- Bank decisions reference immutable submitted answers; current answers are never updated.
CREATE TABLE capture_response_assessments (
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 legal_entity_id uuid NOT NULL REFERENCES legal_entities(id),
 response_revision_id uuid NOT NULL REFERENCES capture_response_revisions(id),
 version bigint NOT NULL CHECK(version>0),
 state text NOT NULL CHECK(state IN ('NOT_REQUIRED','AWAITING_REVIEW','IN_REVIEW','ASSESSED')),
 required_count integer NOT NULL CHECK(required_count BETWEEN 0 AND 200),
 reviewed_required_count integer NOT NULL CHECK(reviewed_required_count BETWEEN 0 AND required_count),
 reviewed_count integer NOT NULL CHECK(reviewed_count BETWEEN 0 AND 200),
 score_result jsonb NOT NULL,
 decisions jsonb NOT NULL,
 created_at timestamptz NOT NULL,
 PRIMARY KEY(tenant_id,response_revision_id,version)
);
CREATE INDEX capture_response_assessments_scoped_latest ON capture_response_assessments(tenant_id,legal_entity_id,response_revision_id,version DESC);
CREATE TABLE capture_field_assessments (
 id uuid PRIMARY KEY,
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 legal_entity_id uuid NOT NULL REFERENCES legal_entities(id),
 response_revision_id uuid NOT NULL REFERENCES capture_response_revisions(id),
 assessment_version bigint NOT NULL,
 form_template_id uuid NOT NULL,
 form_template_version bigint NOT NULL,
 field_id text NOT NULL CHECK(length(field_id) BETWEEN 1 AND 80),
 field_checksum text NOT NULL CHECK(length(field_checksum)=64),
 reviewer_id uuid NOT NULL REFERENCES principals(id),
 authority_route text NOT NULL CHECK(length(authority_route)>0),
 supersedes_id uuid REFERENCES capture_field_assessments(id),
 decision jsonb NOT NULL,
 assessed_at timestamptz NOT NULL,
 UNIQUE(tenant_id,response_revision_id,assessment_version,field_id),
 FOREIGN KEY(tenant_id,response_revision_id,assessment_version) REFERENCES capture_response_assessments(tenant_id,response_revision_id,version)
);
CREATE INDEX capture_field_assessments_history ON capture_field_assessments(tenant_id,response_revision_id,field_id,assessed_at DESC,id DESC);
CREATE FUNCTION prevent_response_assessment_mutation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'Bank assessment records are immutable'; END $$;
CREATE TRIGGER capture_response_assessments_immutable BEFORE UPDATE OR DELETE ON capture_response_assessments FOR EACH ROW EXECUTE FUNCTION prevent_response_assessment_mutation();
CREATE TRIGGER capture_field_assessments_immutable BEFORE UPDATE OR DELETE ON capture_field_assessments FOR EACH ROW EXECUTE FUNCTION prevent_response_assessment_mutation();
CREATE UNIQUE INDEX capture_response_assessment_outbox_uq ON outbox_events(tenant_id,aggregate_id,event_type,((payload->>'version')::bigint)) WHERE aggregate_type='FORM_RESPONSE_ASSESSMENT';
