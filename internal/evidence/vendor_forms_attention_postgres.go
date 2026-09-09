//go:build postgres

package evidence

// The same field facts feed the page and aggregate. Expiry belongs to current
// evidence, never the invitation or request deadline. Review joins are exact
// tenant/entity/assessment/request/artifact joins, as in document inventory.
func vendorDocumentFactsSQL(allowUnscanned bool) string {
	artifactStatus := "held_artifact.status='AVAILABLE'"
	if allowUnscanned {
		artifactStatus = "held_artifact.status IN ('AVAILABLE','STORED_UNSCANNED')"
	}
	return ` LEFT JOIN LATERAL (
 SELECT COALESCE(jsonb_object_agg(fact.field_id,jsonb_build_object('expires_on',fact.expires_on,'source',fact.source,'rejected',fact.rejected,'expired',fact.expired,'current',fact.current)),'{}'::jsonb) AS facts,
 COALESCE(bool_or(fact.current AND (fact.expired OR (fact.expires_on ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' AND pg_input_is_valid(fact.expires_on,'date') AND fact.expires_on<to_char($4::timestamptz AT TIME ZONE 'UTC','YYYY-MM-DD')))),false) AS expired,
 count(*) FILTER(WHERE fact.current AND fact.expires_on ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' AND pg_input_is_valid(fact.expires_on,'date'))>0
 AND COALESCE(bool_and(fact.current AND fact.expires_on ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' AND pg_input_is_valid(fact.expires_on,'date')),false) AS known
 FROM (
 SELECT field->>'id' AS field_id,
 CASE WHEN held.use_held THEN COALESCE(LEAST(COALESCE(source_review.expires_on::text,NULLIF(field->'collection_resolution'->'source'->>'expires_on','')),NULLIF(field->'collection_resolution'->'document'->>'expires_on','')),'') ELSE COALESCE(review.expires_on::text,submission.answers->(field->>'id')->'document'->>'expires_on','') END AS expires_on,
 CASE WHEN (CASE WHEN held.use_held THEN COALESCE(source_review.status,field->'collection_resolution'->'source'->'review'->>'status',field->'collection_resolution'->>'bank_review_state','') ELSE COALESCE(review.status,'') END) IN ('VALIDATED','REJECTED','EXPIRED') THEN 'REVIEW' ELSE 'RESPONSE' END AS source,
 CASE WHEN held.use_held THEN COALESCE(source_review.status,field->'collection_resolution'->>'bank_review_state','')='REJECTED' ELSE COALESCE(review.status,'')='REJECTED' END AS rejected,
 CASE WHEN held.use_held THEN COALESCE(source_review.status,field->'collection_resolution'->'source'->'review'->>'status','')='EXPIRED' ELSE COALESCE(review.status,'')='EXPIRED' END AS expired,
 (` + documentCurrentSQL() + `) AND CASE WHEN held.use_held THEN held.valid_source ELSE submission.answers->(field->>'id')->'document' IS NOT NULL END AS current
 FROM jsonb_array_elements(req.fields) field
 LEFT JOIN capture_artifacts artifact ON artifact.id=CASE WHEN pg_input_is_valid(submission.answers->(field->>'id')->'document'->>'artifact_id','uuid') THEN (submission.answers->(field->>'id')->'document'->>'artifact_id')::uuid END AND artifact.tenant_id=req.tenant_id AND artifact.submission_id=submission.id
 LEFT JOIN third_party_documents review ON review.tenant_id=req.tenant_id AND review.legal_entity_id=req.legal_entity_id AND review.assessment_id=assessment.id AND review.request_id=req.id AND review.artifact_id=artifact.id
 LEFT JOIN third_party_documents source_review ON source_review.tenant_id=req.tenant_id AND source_review.legal_entity_id=req.legal_entity_id AND source_review.assessment_id=NULLIF(field->'collection_resolution'->'source'->>'assessment_id','')::uuid AND source_review.request_id=(field->'collection_resolution'->'source'->>'request_id')::uuid AND source_review.artifact_id=(field->'collection_resolution'->'source'->>'artifact_id')::uuid
 LEFT JOIN LATERAL (SELECT
 COALESCE(field->'collection_resolution'->'source'->>'current','false')='true'
 AND ` + collectionSourceCurrentSQL("field->'collection_resolution'", "req.tenant_id", "req.legal_entity_id") + `
 AND EXISTS(SELECT 1 FROM capture_artifacts held_artifact WHERE held_artifact.tenant_id=req.tenant_id AND held_artifact.id=(field->'collection_resolution'->'source'->>'artifact_id')::uuid AND held_artifact.request_id=(field->'collection_resolution'->>'source_artifact_request_id')::uuid AND held_artifact.sha256=field->'collection_resolution'->'source'->>'sha256' AND held_artifact.size_bytes=(field->'collection_resolution'->'source'->>'size_bytes')::bigint AND ` + artifactStatus + `) AS valid_source,
 submission.answers->(field->>'id')->'document' IS NULL AND COALESCE(field->'collection_resolution'->>'id','')<>'' AS use_held
 ) held ON true
 WHERE field->>'type'='vendor_document' AND ` + vendorFieldVisibleSQL() + `
 ) fact
 ) document_facts ON true `
}

// Visibility mirrors the contract's field and section conditions. Conditions
// only compare submitted values; no label or answer-language inference occurs.
func vendorFieldVisibleSQL() string {
	return `NOT EXISTS (
 SELECT 1 FROM (SELECT field->'condition' AS condition UNION ALL SELECT section->'condition' FROM jsonb_array_elements(req.sections) section WHERE section->>'id'=field->>'section_id') conditions
 CROSS JOIN LATERAL (SELECT submission.answers->(condition->>'field_id') AS answer) observed
 CROSS JOIN LATERAL (SELECT EXISTS(SELECT 1 FROM jsonb_array_elements_text(CASE WHEN COALESCE(btrim(answer->>'text'),'')<>'' THEN jsonb_build_array(btrim(answer->>'text')) ELSE COALESCE(answer->'values','[]'::jsonb) END) actual WHERE btrim(actual)=ANY(ARRAY(SELECT jsonb_array_elements_text(COALESCE(condition->'values','[]'::jsonb))))) AS matched) comparison
 WHERE condition IS NOT NULL AND condition<>'null'::jsonb AND NOT CASE condition->>'operator'
 WHEN 'ANSWERED' THEN COALESCE(btrim(answer->>'text'),'')<>'' OR COALESCE(jsonb_array_length(answer->'values'),0)>0 OR COALESCE(jsonb_array_length(answer->'artifact_ids'),0)>0 OR COALESCE(answer->'document','null'::jsonb)<>'null'::jsonb
 WHEN 'EQUALS' THEN matched WHEN 'IN' THEN matched WHEN 'NOT_EQUALS' THEN NOT matched WHEN 'NOT_IN' THEN NOT matched ELSE false END
 )`
}
