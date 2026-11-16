BEGIN;

-- Internal respondents use the same draft contract without impersonating an
-- invitation/capture session. Existing session drafts remain unchanged.
ALTER TABLE capture_response_drafts ALTER COLUMN session_id DROP NOT NULL;
ALTER TABLE capture_response_drafts ADD COLUMN principal_id uuid;
ALTER TABLE capture_response_drafts ADD CONSTRAINT capture_draft_principal_fk
 FOREIGN KEY(principal_id,tenant_id) REFERENCES principals(id,tenant_id);
ALTER TABLE capture_response_drafts ADD CONSTRAINT capture_draft_identity_check
 CHECK ((session_id IS NULL) <> (principal_id IS NULL));
CREATE UNIQUE INDEX capture_internal_draft_identity_idx
 ON capture_response_drafts(tenant_id,request_id,principal_id) WHERE principal_id IS NOT NULL;

CREATE UNIQUE INDEX capture_submission_review_scope_idx ON capture_submissions(id,tenant_id,request_id);
CREATE TABLE capture_submission_reviews (
 id uuid PRIMARY KEY DEFAULT uuidv7(),tenant_id uuid NOT NULL,legal_entity_id uuid NOT NULL,
 request_id uuid NOT NULL,submission_id uuid NOT NULL,reviewed_by uuid NOT NULL,
 status text NOT NULL CHECK(status IN ('REVIEWED','FOLLOW_UP_REQUIRED')),
 rationale text NOT NULL CHECK(char_length(btrim(rationale)) BETWEEN 1 AND 4000),
 groups jsonb NOT NULL CHECK(jsonb_typeof(groups)='array' AND jsonb_array_length(groups) BETWEEN 1 AND 20),
 version bigint NOT NULL CHECK(version>0),created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(tenant_id,request_id,version),
 FOREIGN KEY(request_id,tenant_id) REFERENCES capture_requests(id,tenant_id),
 FOREIGN KEY(submission_id,tenant_id,request_id) REFERENCES capture_submissions(id,tenant_id,request_id),
 FOREIGN KEY(legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
 FOREIGN KEY(reviewed_by,tenant_id) REFERENCES principals(id,tenant_id)
);

-- Eligibility comes from current organizational assignments, never display names.
CREATE FUNCTION framework_assessment_recipient_eligible(t uuid,e uuid,p uuid,units jsonb)
RETURNS boolean LANGUAGE sql STABLE AS $$
 WITH RECURSIVE selected AS (
  SELECT s.id FROM organization_scopes s WHERE s.tenant_id=t AND s.legal_entity_id=e
   AND units ? s.id::text AND s.status='ACTIVE' AND s.valid_from<=clock_timestamp()
   AND (s.valid_until IS NULL OR s.valid_until>clock_timestamp())
  UNION
  SELECT s.id FROM organization_scopes s JOIN selected parent ON s.parent_scope_id=parent.id
   WHERE s.tenant_id=t AND s.legal_entity_id=e AND s.status='ACTIVE'
    AND s.valid_from<=clock_timestamp() AND (s.valid_until IS NULL OR s.valid_until>clock_timestamp())
 )
 SELECT EXISTS(SELECT 1 FROM principals person WHERE person.id=p AND person.tenant_id=t
  AND person.kind='PERSON' AND person.status='ACTIVE' AND person.valid_from<=clock_timestamp()
  AND (person.valid_until IS NULL OR person.valid_until>clock_timestamp()) AND (
   EXISTS(SELECT 1 FROM org_positions position WHERE position.tenant_id=t AND position.legal_entity_id=e
    AND position.occupant_principal_id=p AND position.organization_scope_id IN (SELECT id FROM selected)
    AND position.valid_from<=clock_timestamp() AND (position.valid_until IS NULL OR position.valid_until>clock_timestamp()))
   OR EXISTS(SELECT 1 FROM scim_users u JOIN scim_sources source ON source.id=u.source_id AND source.tenant_id=u.tenant_id
    JOIN directory_group_members m ON m.scim_user_id=u.id AND m.tenant_id=u.tenant_id
    JOIN directory_groups g ON g.id=m.group_id AND g.tenant_id=m.tenant_id AND g.source_id=u.source_id
    JOIN directory_group_role_bindings b ON b.group_id=g.id AND b.tenant_id=g.tenant_id
    JOIN role_templates role ON role.id=b.role_template_id AND role.tenant_id=b.tenant_id
    WHERE u.tenant_id=t AND u.principal_id=p AND u.active AND u.deleted_at IS NULL AND source.status='ACTIVE'
     AND g.deleted_at IS NULL AND b.legal_entity_id=e AND b.organization_scope_id IN (SELECT id FROM selected)
     AND b.valid_from<=clock_timestamp() AND (b.valid_until IS NULL OR b.valid_until>clock_timestamp())
     AND role.valid_from<=clock_timestamp() AND (role.valid_until IS NULL OR role.valid_until>clock_timestamp()))
  ))
$$;

-- Recheck the adoption, owner, scope and approved requirements under database
-- locks in the request INSERT transaction. The HTTP/service checks alone would
-- otherwise race a Program edit or a requirement decision.
CREATE FUNCTION validate_adopted_assessment_request() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE program_record programs%ROWTYPE; adoption program_framework_adoptions%ROWTYPE; form monitoring_form_templates%ROWTYPE;
BEGIN
 IF NEW.origin_type <> 'FRAMEWORK_ASSESSMENT' THEN RETURN NEW; END IF;
 SELECT * INTO program_record FROM programs WHERE id::text=NEW.subject_id AND tenant_id=NEW.tenant_id
  AND legal_entity_id=NEW.legal_entity_id FOR SHARE;
 IF NOT FOUND OR NEW.subject_type<>'PROGRAM' OR program_record.status<>'ACTIVE'
  OR program_record.owner_principal_id IS DISTINCT FROM NEW.created_by
  OR program_record.version::text IS DISTINCT FROM NEW.known_facts->>'program_version'
  OR program_record.authority_principal_id::text IS DISTINCT FROM NEW.known_facts->>'reviewer'
  OR NEW.recipient_principal_id=program_record.authority_principal_id THEN
  RAISE EXCEPTION 'Program assessment scope changed' USING ERRCODE='23514';
 END IF;
 SELECT * INTO adoption FROM program_framework_adoptions WHERE id::text=NEW.known_facts->>'adoption_id'
  AND tenant_id=NEW.tenant_id AND legal_entity_id=NEW.legal_entity_id AND program_id=program_record.id FOR SHARE;
 IF NOT FOUND OR adoption.status<>'ACTIVE' OR adoption.configuration IS NULL
  OR NOT framework_assessment_recipient_eligible(NEW.tenant_id,NEW.legal_entity_id,NEW.recipient_principal_id,adoption.configuration->'respondent_scope_ids') THEN
  RAISE EXCEPTION 'Approved adoption or respondent assignment changed' USING ERRCODE='23514';
 END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(adoption.configuration->'requirements') req
  WHERE NOT EXISTS(SELECT 1 FROM program_requirements r WHERE r.tenant_id=NEW.tenant_id AND r.program_id=program_record.id
   AND r.code=adoption.template_code||'-'||(req->>'code') AND r.status='APPROVED')) THEN
  RAISE EXCEPTION 'Resolve requirement reviews before creating an assessment' USING ERRCODE='23514';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(adoption.configuration->'requirements') req
  WHERE req->>'form_template_id'=NEW.form_template_id::text AND (req->>'form_template_version')::bigint=NEW.form_template_version) THEN
  RAISE EXCEPTION 'Questionnaire is not pinned to the adoption' USING ERRCODE='23514';
 END IF;
 SELECT * INTO form FROM monitoring_form_templates WHERE tenant_id=NEW.tenant_id AND legal_entity_id=NEW.legal_entity_id
  AND id=NEW.form_template_id AND version=NEW.form_template_version FOR SHARE;
 IF NOT FOUND OR form.status NOT IN ('ACTIVE','RETIRED') OR jsonb_array_length(NEW.fields)<>jsonb_array_length(form.fields)
  OR EXISTS(SELECT 1 FROM jsonb_array_elements(NEW.fields) f WHERE NOT EXISTS(SELECT 1 FROM jsonb_array_elements(form.fields) original
   WHERE original->>'id'=f->>'id' AND original->>'type'=f->>'type' AND original->>'label'=f->>'label'
    AND original->>'section_id'=f->>'section_id' AND original->>'required'=f->>'required')) THEN
  RAISE EXCEPTION 'Approved questionnaire revision changed' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER capture_adopted_assessment_scope BEFORE INSERT ON capture_requests
 FOR EACH ROW WHEN (NEW.origin_type='FRAMEWORK_ASSESSMENT') EXECUTE FUNCTION validate_adopted_assessment_request();

CREATE FUNCTION record_adopted_assessment_creation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
 VALUES(NEW.tenant_id,'EVIDENCE_REQUEST',NEW.id,'ProgramAssessmentCreated',
  jsonb_build_object('program_id',NEW.subject_id,'adoption_id',NEW.known_facts->>'adoption_id','created_by',NEW.created_by),
  NEW.created_at,NEW.created_at,NEW.created_at);
 RETURN NEW;
END $$;
CREATE TRIGGER capture_adopted_assessment_created AFTER INSERT ON capture_requests
 FOR EACH ROW WHEN (NEW.origin_type='FRAMEWORK_ASSESSMENT') EXECUTE FUNCTION record_adopted_assessment_creation();
COMMIT;
