//go:build postgres

package authority

import (
	"fmt"
	"strings"
)

// PostgresReadRouteSQL is the shared materiality-three read-route predicate.
// All SQL identifiers and route constants must be compile-time caller values,
// never request input. Actor and evaluation time remain bound parameters.
func PostgresReadRouteSQL(alias, objectIDColumn, objectType, decisionType, responsibility string, actorParam, atParam int) string {
	query := fmt.Sprintf(`EXISTS (
		WITH RECURSIVE route_defs AS (
			SELECT ear.source_rule_id AS rule_id,ear.policy_version,ear.priority,
				(CASE WHEN ear.legal_entity_ref<>'*' THEN 8 ELSE 0 END + CASE WHEN ear.object_type<>'*' THEN 4 ELSE 0 END + CASE WHEN ear.object_id<>'*' THEN 2 ELSE 0 END + CASE WHEN ear.decision_type<>'' THEN 1 ELSE 0 END) AS specificity,
				ear.selector_kind,ear.selector_ref
			FROM effective_authority_routes ear
			WHERE ear.tenant_id=w.tenant_id
			  AND (ear.legal_entity_ref='*' OR ear.legal_entity_ref=w.legal_entity_id::text OR ear.legal_entity_ref=(SELECT le.code FROM legal_entities le WHERE le.id=w.legal_entity_id AND le.tenant_id=w.tenant_id))
			  AND (ear.object_type='*' OR upper(ear.object_type)='VENDOR_RELATIONSHIP')
			  AND (ear.object_id='*' OR ear.object_id=w.relationship_id::text)
			  AND ear.responsibility='REVIEWER'
			  AND (ear.decision_type='' OR upper(ear.decision_type)='THIRDPARTY.WORK.REVIEW')
			  AND ear.min_materiality<=3 AND ear.valid_from<=$%[2]d AND (ear.valid_until IS NULL OR $%[2]d<ear.valid_until)
			UNION ALL
			SELECT 'assignment:'||ra.id::text,ra.policy_version,ra.priority,
				(CASE WHEN ra.legal_entity_id IS NOT NULL THEN 8 ELSE 0 END + 4 + CASE WHEN ra.object_id IS NOT NULL THEN 2 ELSE 0 END + CASE WHEN COALESCE(ra.decision_type,'')<>'' THEN 1 ELSE 0 END),
				CASE WHEN ra.principal_id IS NOT NULL THEN 'PRINCIPAL_ID' WHEN ra.position_id IS NOT NULL THEN 'POSITION_ID' ELSE 'ROLE_ID' END,
				COALESCE(ra.principal_id::text,ra.position_id::text,ra.role_template_id::text)
			FROM responsibility_assignments ra
			WHERE ra.tenant_id=w.tenant_id AND (ra.legal_entity_id IS NULL OR ra.legal_entity_id=w.legal_entity_id)
			  AND upper(ra.object_type)='VENDOR_RELATIONSHIP' AND (ra.object_id IS NULL OR ra.object_id=w.relationship_id)
			  AND ra.responsibility='REVIEWER' AND (COALESCE(ra.decision_type,'')='' OR upper(ra.decision_type)='THIRDPARTY.WORK.REVIEW')
			  AND ra.valid_from<=$%[2]d AND (ra.valid_until IS NULL OR $%[2]d<ra.valid_until)
		), resolved AS (
			SELECT rd.rule_id,rd.policy_version,rd.priority,rd.specificity,p.principal_id
			FROM route_defs rd
			JOIN LATERAL (
				SELECT p.id AS principal_id FROM principals p
				WHERE rd.selector_kind IN ('PRINCIPAL','TEAM','QUEUE','COMMITTEE') AND p.tenant_id=w.tenant_id
				  AND (p.id::text=rd.selector_ref OR p.external_ref=rd.selector_ref) AND p.status='ACTIVE'
				  AND p.valid_from<=$%[2]d AND (p.valid_until IS NULL OR $%[2]d<p.valid_until)
				  AND (rd.selector_kind='PRINCIPAL' OR p.kind=rd.selector_kind)
				UNION ALL
				SELECT p.id FROM principals p WHERE rd.selector_kind='PRINCIPAL_ID' AND p.id::text=rd.selector_ref AND p.tenant_id=w.tenant_id
				  AND p.status='ACTIVE' AND p.valid_from<=$%[2]d AND (p.valid_until IS NULL OR $%[2]d<p.valid_until)
				UNION ALL
				SELECT p.id FROM org_positions op JOIN principals p ON p.id=op.occupant_principal_id
				WHERE rd.selector_kind IN ('POSITION','POSITION_ID') AND op.tenant_id=w.tenant_id
				  AND ((rd.selector_kind='POSITION' AND (op.code=rd.selector_ref OR op.id::text=rd.selector_ref)) OR (rd.selector_kind='POSITION_ID' AND op.id::text=rd.selector_ref))
				  AND (op.legal_entity_id IS NULL OR op.legal_entity_id=w.legal_entity_id)
				  AND op.valid_from<=$%[2]d AND (op.valid_until IS NULL OR $%[2]d<op.valid_until)
				  AND p.status='ACTIVE' AND p.valid_from<=$%[2]d AND (p.valid_until IS NULL OR $%[2]d<p.valid_until)
				UNION ALL
				SELECT p.id FROM role_templates rt JOIN position_role_bindings prb ON prb.role_template_id=rt.id
				JOIN org_positions op ON op.id=prb.position_id JOIN principals p ON p.id=op.occupant_principal_id
				WHERE rd.selector_kind IN ('ROLE','ROLE_ID') AND rt.tenant_id=w.tenant_id
				  AND ((rd.selector_kind='ROLE' AND (rt.code=rd.selector_ref OR rt.id::text=rd.selector_ref)) OR (rd.selector_kind='ROLE_ID' AND rt.id::text=rd.selector_ref))
				  AND (op.legal_entity_id IS NULL OR op.legal_entity_id=w.legal_entity_id)
				  AND rt.valid_from<=$%[2]d AND (rt.valid_until IS NULL OR $%[2]d<rt.valid_until)
				  AND prb.valid_from<=$%[2]d AND (prb.valid_until IS NULL OR $%[2]d<prb.valid_until)
				  AND op.valid_from<=$%[2]d AND (op.valid_until IS NULL OR $%[2]d<op.valid_until)
				  AND p.status='ACTIVE' AND p.valid_from<=$%[2]d AND (p.valid_until IS NULL OR $%[2]d<p.valid_until)
			) p ON true
		), route_groups AS (
			SELECT rule_id,policy_version,priority,specificity,array_agg(DISTINCT principal_id ORDER BY principal_id) AS candidates
			FROM resolved GROUP BY rule_id,policy_version,priority,specificity
		), top_rank AS (
			SELECT priority,specificity FROM route_groups ORDER BY priority DESC,specificity DESC,rule_id LIMIT 1
		), top_groups AS (
			SELECT g.* FROM route_groups g JOIN top_rank r USING(priority,specificity)
		), unambiguous AS (
			SELECT count(*)>0 AND count(DISTINCT candidates::text)=1 AS allowed FROM top_groups
		), seeds AS (
			SELECT DISTINCT candidate.principal_id FROM top_groups g CROSS JOIN LATERAL unnest(g.candidates) candidate(principal_id)
		), chain(origin_id,principal_id,path,depth) AS (
			SELECT principal_id,principal_id,ARRAY[principal_id],0 FROM seeds
			UNION ALL
			SELECT c.origin_id,d.to_principal_id,c.path||d.to_principal_id,c.depth+1 FROM chain c JOIN delegations d ON d.from_principal_id=c.principal_id
			WHERE d.tenant_id=w.tenant_id AND d.responsibility='REVIEWER' AND d.status='ACTIVE'
			  AND d.starts_at<=$%[2]d AND $%[2]d<d.ends_at AND c.depth<8 AND NOT d.to_principal_id=ANY(c.path)
			  AND (NOT (d.scope?'legal_entity_id') OR d.scope->>'legal_entity_id' IN ('*',w.legal_entity_id::text))
			  AND (NOT (d.scope?'object_type') OR upper(d.scope->>'object_type') IN ('*','VENDOR_RELATIONSHIP'))
			  AND (NOT (d.scope?'object_id') OR d.scope->>'object_id' IN ('*',w.relationship_id::text))
			  AND (NOT (d.scope?'decision_type') OR upper(d.scope->>'decision_type') IN ('*','THIRDPARTY.WORK.REVIEW'))
		), effective AS (
			SELECT DISTINCT c.origin_id,c.principal_id FROM chain c JOIN principals p ON p.id=c.principal_id
			WHERE p.status='ACTIVE' AND p.valid_from<=$%[2]d AND (p.valid_until IS NULL OR $%[2]d<p.valid_until)
		), relevant_grants AS (
			SELECT ag.* FROM authority_grants ag WHERE ag.tenant_id=w.tenant_id AND (ag.legal_entity_id IS NULL OR ag.legal_entity_id=w.legal_entity_id)
			  AND (ag.decision_type='*' OR upper(ag.decision_type)='THIRDPARTY.WORK.REVIEW')
			  AND ag.valid_from<=$%[2]d AND (ag.valid_until IS NULL OR $%[2]d<ag.valid_until)
		), granted AS (
			SELECT ag.principal_id AS principal_id FROM relevant_grants ag WHERE ag.principal_id IS NOT NULL
			  AND COALESCE(NULLIF(ag.limits->>'min_materiality','')::integer,0)<=3 AND COALESCE(NULLIF(ag.limits->>'max_materiality','')::integer,5)>=3
			UNION
			SELECT op.occupant_principal_id FROM relevant_grants ag JOIN org_positions op ON op.id=ag.position_id
			WHERE ag.position_id IS NOT NULL AND COALESCE(NULLIF(ag.limits->>'min_materiality','')::integer,0)<=3 AND COALESCE(NULLIF(ag.limits->>'max_materiality','')::integer,5)>=3
			  AND op.valid_from<=$%[2]d AND (op.valid_until IS NULL OR $%[2]d<op.valid_until)
			UNION
			SELECT op.occupant_principal_id FROM relevant_grants ag JOIN position_role_bindings prb ON prb.role_template_id=ag.role_template_id
			JOIN org_positions op ON op.id=prb.position_id WHERE ag.role_template_id IS NOT NULL
			  AND COALESCE(NULLIF(ag.limits->>'min_materiality','')::integer,0)<=3 AND COALESCE(NULLIF(ag.limits->>'max_materiality','')::integer,5)>=3
			  AND prb.valid_from<=$%[2]d AND (prb.valid_until IS NULL OR $%[2]d<prb.valid_until)
			  AND op.valid_from<=$%[2]d AND (op.valid_until IS NULL OR $%[2]d<op.valid_until)
		), eligible AS (
			SELECT e.* FROM effective e WHERE NOT EXISTS(SELECT 1 FROM relevant_grants)
			  OR e.principal_id IN (SELECT principal_id FROM granted) OR e.origin_id IN (SELECT principal_id FROM granted)
		)
		SELECT 1 FROM eligible e CROSS JOIN unambiguous u
		WHERE u.allowed AND e.principal_id=$%[1]d::uuid
		  AND NOT EXISTS (
			SELECT 1 FROM org_positions op JOIN position_role_bindings prb ON prb.position_id=op.id
			JOIN role_templates rt ON rt.id=prb.role_template_id JOIN segregation_rules sr ON sr.tenant_id=op.tenant_id AND sr.prohibited_role_code=rt.code
			WHERE op.tenant_id=w.tenant_id AND op.occupant_principal_id=e.principal_id AND sr.responsibility='REVIEWER' AND sr.status='ACTIVE'
			  AND sr.valid_from<=$%[2]d AND (sr.valid_until IS NULL OR $%[2]d<sr.valid_until)
			  AND op.valid_from<=$%[2]d AND (op.valid_until IS NULL OR $%[2]d<op.valid_until)
			  AND prb.valid_from<=$%[2]d AND (prb.valid_until IS NULL OR $%[2]d<prb.valid_until)
		  )
	)`, actorParam, atParam)
	return strings.NewReplacer("w.", alias+".", "relationship_id", objectIDColumn, "VENDOR_RELATIONSHIP", objectType, "THIRDPARTY.WORK.REVIEW", decisionType, "REVIEWER", responsibility).Replace(query)
}
