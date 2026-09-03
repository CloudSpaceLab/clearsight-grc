//go:build postgres

package activity

const categoryExpression = `CASE
	WHEN upper(oe.aggregate_type) LIKE 'THIRD_PARTY%' OR upper(oe.aggregate_type) LIKE 'VENDOR%' THEN 'VENDOR'
	WHEN upper(oe.aggregate_type) LIKE 'FORM%' OR upper(oe.aggregate_type) LIKE 'CAPTURE%' OR upper(oe.aggregate_type) LIKE 'EVIDENCE%' OR upper(oe.aggregate_type) LIKE 'DOCUMENT_%' THEN 'FORMS_EVIDENCE'
	WHEN upper(oe.aggregate_type) LIKE 'AI%' THEN 'AI'
	WHEN upper(oe.aggregate_type) LIKE '%POLICY%' OR upper(oe.aggregate_type)='DELEGATION' OR upper(oe.aggregate_type) LIKE 'SOURCE_%' OR upper(oe.aggregate_type) LIKE 'SCIM_%' OR upper(oe.aggregate_type) LIKE 'DIRECTORY_%' THEN 'CONFIGURATION'
	WHEN upper(oe.aggregate_type) IN ('PROGRAM','MATTER') OR upper(oe.aggregate_type) LIKE 'WORKFLOW%' OR upper(oe.aggregate_type) LIKE 'DECISION%' OR upper(oe.aggregate_type) LIKE 'ACTION%' THEN 'GRC_WORK'
	WHEN upper(oe.aggregate_type) LIKE 'PROJECTION%' OR upper(oe.aggregate_type) LIKE 'BACKGROUND_JOB%' OR upper(oe.aggregate_type) LIKE 'RUNTIME%' OR upper(oe.aggregate_type)='AUDIT_EXPORT' THEN 'SYSTEM'
	ELSE 'OTHER'
END`

const actorReferenceExpression = `COALESCE(
	NULLIF(oe.payload->>'actor_id',''),
	NULLIF(oe.payload->>'actor_principal_id',''),
	NULLIF(oe.payload->>'created_by',''),
	NULLIF(oe.payload->>'recorded_by',''),
	NULLIF(oe.payload->>'reviewed_by',''),
	NULLIF(oe.payload->>'submitted_by',''),
	NULLIF(oe.payload->>'maker_id',''),
	NULLIF(oe.payload->>'approved_by','')
)`

const actorKindExpression = `CASE
	WHEN upper(COALESCE(p.kind,'')) IN ('PERSON','TEAM','QUEUE','COMMITTEE') THEN 'INTERNAL_USER'
	WHEN upper(COALESCE(p.kind,''))='EXTERNAL_PARTY' THEN 'EXTERNAL_PARTICIPANT'
	WHEN upper(COALESCE(p.kind,''))='SERVICE' THEN 'SERVICE'
	WHEN (` + categoryExpression + `)='SYSTEM' THEN 'SYSTEM'
	ELSE 'UNKNOWN'
END`

// activitySourcesCTE federates only durable sources with an explicit ownership
// contract. Identity/access administration already records committed decisions
// in governance_decisions, so those receipts are projected here instead of
// creating another audit table or writing duplicate business history.
const activitySourcesCTE = `WITH scope AS (
	SELECT id AS tenant_id FROM tenants WHERE id::text=$1 OR slug=$1 LIMIT 1
), oe AS (
	SELECT e.id::text AS id,e.tenant_id,e.aggregate_type,e.aggregate_id::text AS aggregate_id,
	       e.event_type,e.payload,e.occurred_at,'OUTBOX_EVENT'::text AS source
	FROM outbox_events e
	WHERE e.tenant_id=(SELECT tenant_id FROM scope)
	UNION ALL
	SELECT 'decision:' || gd.id::text,gd.tenant_id,gd.object_type,gd.object_id::text,gd.rationale,
	       jsonb_strip_nulls(jsonb_build_object(
	         'actor_id',gd.actor_id::text,
	         'legal_entity_id',CASE WHEN gd.object_type='DIRECTORY_GROUP_ROLE_BINDING' THEN b.legal_entity_id::text END
	       )),gd.decided_at,'GOVERNANCE_DECISION'::text
	FROM governance_decisions gd
	LEFT JOIN directory_group_role_bindings b
	  ON gd.object_type='DIRECTORY_GROUP_ROLE_BINDING' AND b.tenant_id=gd.tenant_id AND b.id=gd.object_id
	WHERE gd.tenant_id=(SELECT tenant_id FROM scope)
	  AND (
	    (gd.object_type='SCIM_SOURCE' AND gd.rationale IN ('SCIM_SOURCE_CREATED','SCIM_SOURCE_TOKEN_ROTATED','SCIM_SOURCE_REVOKED'))
	    OR
	    (gd.object_type='DIRECTORY_GROUP_ROLE_BINDING' AND gd.rationale IN ('DIRECTORY_GROUP_ROLE_BOUND','DIRECTORY_GROUP_ROLE_RETIRED'))
	  )
)`

const activityProjection = `
	SELECT oe.id,oe.occurred_at,oe.aggregate_type,oe.aggregate_id,oe.event_type,
	       COALESCE(meta.actor_ref,''),COALESCE(p.kind,''),COALESCE(p.display_name,''),
	       COALESCE(meta.legal_entity_ref,''),COALESCE(meta.request_ref,''),COALESCE(meta.correlation_ref,''),oe.source
	FROM oe
	LEFT JOIN LATERAL (
		SELECT ` + actorReferenceExpression + ` AS actor_ref,
		       COALESCE(NULLIF(oe.payload->>'legal_entity_id',''),'') AS legal_entity_ref,
		       COALESCE(NULLIF(oe.payload->>'request_id',''),'') AS request_ref,
		       COALESCE(NULLIF(oe.payload->>'correlation_id',''),'') AS correlation_ref
	) meta ON true
	LEFT JOIN principals p ON p.tenant_id=oe.tenant_id AND p.id::text=meta.actor_ref
`
