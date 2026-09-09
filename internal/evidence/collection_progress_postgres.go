//go:build postgres

package evidence

// collectionNoVendorActionSQL is deliberately limited to unconditional
// required document collections. Unknown applicability never suppresses a
// vendor reminder or disappears from the outstanding population. Both inputs
// are fixed SQL identifiers supplied by repository code, never request values.
func collectionNoVendorActionSQL(request, now string) string {
	return `(
	 EXISTS (SELECT 1 FROM jsonb_array_elements(` + request + `.fields) cf WHERE cf->>'required'='true')
	 AND NOT EXISTS (
	  SELECT 1 FROM jsonb_array_elements(` + request + `.fields) cf
	  WHERE cf->>'required'='true' AND NOT (
	   cf->>'type'='vendor_document'
	   AND COALESCE(cf->'condition','null'::jsonb)='null'::jsonb
	   AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements(` + request + `.sections) cs WHERE cs->>'id'=cf->>'section_id' AND COALESCE(cs->'condition','null'::jsonb)<>'null'::jsonb)
	   AND COALESCE(cf->'collection_resolution'->>'id','')<>''
	   AND COALESCE((cf->'collection_resolution'->>'version')::bigint,0)>0
	   AND COALESCE(cf->'collection_resolution'->'source'->>'current','false')='true'
	   AND ` + collectionSourceCurrentSQL("cf->'collection_resolution'", request+".tenant_id", request+".legal_entity_id") + `
	   AND COALESCE(cf->'collection_resolution'->>'bank_review_state','') IN ('PENDING','VALIDATED')
	   AND COALESCE(cf->'collection_resolution'->'source'->'review'->>'status','') NOT IN ('REJECTED','EXPIRED')
	   AND NOT EXISTS (SELECT 1 FROM (VALUES (cf->'collection_resolution'->'source'->>'expires_on'),(cf->'collection_resolution'->'document'->>'expires_on')) expiry(value)
	     WHERE COALESCE(value,'')<>'' AND (value !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR NOT pg_input_is_valid(value,'date') OR value<to_char(` + now + `::timestamptz AT TIME ZONE 'UTC','YYYY-MM-DD')))
	   AND NOT EXISTS (SELECT 1 FROM third_party_documents source_review
	     WHERE source_review.tenant_id=` + request + `.tenant_id AND source_review.legal_entity_id=` + request + `.legal_entity_id
	       AND source_review.assessment_id=NULLIF(cf->'collection_resolution'->'source'->>'assessment_id','')::uuid
	       AND source_review.request_id=(cf->'collection_resolution'->'source'->>'request_id')::uuid
	       AND source_review.artifact_id=(cf->'collection_resolution'->'source'->>'artifact_id')::uuid
	       AND (source_review.status IN ('REJECTED','EXPIRED') OR source_review.expires_on<(` + now + `::timestamptz AT TIME ZONE 'UTC')::date))
	   AND EXISTS (SELECT 1 FROM capture_artifacts ca
	     WHERE ca.id=(cf->'collection_resolution'->'source'->>'artifact_id')::uuid
	       AND ca.tenant_id=` + request + `.tenant_id
	       AND ca.request_id=(cf->'collection_resolution'->>'source_artifact_request_id')::uuid
	       AND ca.status='AVAILABLE'
	       AND ca.sha256=cf->'collection_resolution'->'source'->>'sha256'
	       AND ca.size_bytes=(cf->'collection_resolution'->'source'->>'size_bytes')::bigint)
	  ) IS TRUE
	 )
	)`
}
