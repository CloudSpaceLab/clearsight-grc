//go:build postgres

package evidence

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"time"
)

// All branches constrain tenant, legal entity and current record permission before
// filtering or pagination. Payloads are deliberately projected, never returned raw.
const programActivitySQL = `WITH target AS (
 SELECT p.* FROM programs p JOIN tenants t ON t.id=p.tenant_id
 WHERE (t.id::text=$1 OR t.slug=$1) AND p.legal_entity_id::text=$2 AND p.id::text=$3
), scoped AS (
 SELECT er.* FROM capture_requests er JOIN tenants t ON t.id=er.tenant_id WHERE ` + programAssessmentReadScope + `
), issues AS (
 SELECT DISTINCT m.* FROM target p JOIN matter_links ml ON ml.program_id=p.id AND ml.tenant_id=p.tenant_id AND ml.retired_at IS NULL
 JOIN matters m ON m.id=ml.matter_id AND m.tenant_id=p.tenant_id AND m.legal_entity_id=p.legal_entity_id
 WHERE m.owner_principal_id::text=$4 OR EXISTS(SELECT 1 FROM matter_actions a WHERE a.tenant_id=m.tenant_id AND a.matter_id=m.id AND a.owner_principal_id::text=$4)
 OR CASE WHEN NOT(m.scope ? 'access') THEN true
 WHEN jsonb_typeof(m.scope->'access')<>'string' THEN false
 WHEN upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL') THEN true
 WHEN upper(btrim(m.scope->>'access'))='RESTRICTED' THEN CASE WHEN jsonb_typeof(m.scope->'allowed_principal_ids')='array' THEN
 NOT EXISTS(SELECT 1 FROM jsonb_array_elements(m.scope->'allowed_principal_ids') x WHERE jsonb_typeof(x)<>'string') AND
 EXISTS(SELECT 1 FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') x WHERE btrim(x)=$4) ELSE false END ELSE false END
), events(id,category,record_id,request_id,submission_id,artifact_id,field_id,title,event,actor,at,details) AS (
 SELECT 'program:'||e.id::text,'PROGRAM',p.id::text,'','','','',p.name,e.event_type,coalesce(e.actor_id::text,''),e.occurred_at,
  jsonb_strip_nulls(jsonb_build_object('version',e.aggregate_version,'status',e.payload->>'status','rationale',e.payload->>'rationale',
 'name',coalesce(e.payload#>>'{program,name}',e.payload->>'name'),'previous_name',e.payload#>>'{previous,name}',
 'accountable_function',coalesce(e.payload#>>'{program,owning_function}',e.payload->>'owning_function'),'previous_accountable_function',e.payload#>>'{previous,owning_function}',
 'jurisdiction',coalesce(e.payload#>>'{program,jurisdiction}',e.payload->>'jurisdiction'),'previous_jurisdiction',e.payload#>>'{previous,jurisdiction}',
 'owner_reference',e.payload->>'owner_principal_id','previous_owner_reference',e.payload->>'previous_owner_principal_id',
 'approval_authority_reference',e.payload->>'authority_principal_id','previous_approval_authority_reference',e.payload->>'previous_authority_principal_id',
 'record_title',e.payload->>'title','record_reference',e.payload->>'id'))
 FROM target p JOIN continuity_events e ON e.tenant_id=p.tenant_id AND e.aggregate_type='PROGRAM' AND e.aggregate_id=p.id
 WHERE $4 IN (p.owner_principal_id::text,p.authority_principal_id::text)
 UNION ALL
 SELECT 'program-review:'||review.id::text,'PROGRAM',p.id::text,'','','','',p.name,'PROGRAM_REVIEW_ACCEPTED',review.principal_id::text,review.accepted_at,
 jsonb_build_object('program_version',review.program_version,'projection_version',review.projection_version)
 FROM target p JOIN program_review_checkpoints review ON review.tenant_id=p.tenant_id AND review.program_id=p.id
 WHERE $4 IN (p.owner_principal_id::text,p.authority_principal_id::text)
 UNION ALL
 SELECT 'questionnaire-link:'||event.id::text,'PROGRAM',p.id::text,'','','','',form.name,'QUESTIONNAIRE_'||event.event_type,event.actor_id::text,event.occurred_at,
 jsonb_strip_nulls(jsonb_build_object('form_version',link.form_template_version,'form_reference',link.form_template_id,'reason',event.payload->>'reason'))
 FROM target p JOIN framework_adoption_questionnaires link ON link.program_id=p.id AND link.tenant_id=p.tenant_id AND link.legal_entity_id=p.legal_entity_id
 JOIN framework_adoption_questionnaire_events event ON event.link_id=link.id AND event.tenant_id=p.tenant_id AND event.legal_entity_id=p.legal_entity_id
 JOIN monitoring_form_templates form ON form.id=link.form_template_id AND form.version=link.form_template_version AND form.tenant_id=p.tenant_id AND form.legal_entity_id=p.legal_entity_id
 WHERE $4 IN (p.owner_principal_id::text,p.authority_principal_id::text)
 UNION ALL
 SELECT 'issue:'||e.id::text,'ISSUE',m.id::text,'','','','',m.title,e.event_type,coalesce(e.actor_id::text,''),e.occurred_at,
 jsonb_strip_nulls(jsonb_build_object('version',e.aggregate_version,'status',e.payload->>'status','rationale',e.payload->>'rationale','outcome',coalesce(e.payload->>'outcome',e.payload->>'result'),'closure_reason',e.payload->>'closure_reason','action',CASE WHEN e.event_type LIKE 'ACTION_%' THEN e.payload->>'title' END))
 FROM issues m JOIN continuity_events e ON e.tenant_id=m.tenant_id AND e.aggregate_type='MATTER' AND e.aggregate_id=m.id
 UNION ALL
 SELECT 'schedule:'||e.id::text,'SCHEDULE',s.id::text,coalesce(e.payload->>'request_id',''),'','','','Assessment schedule',e.event_type,coalesce(e.payload->>'actor_id',''),e.occurred_at,
 jsonb_build_object('version',e.payload->'version','state',e.payload->>'state','reason',e.payload->>'reason','cycle_index',e.payload->'cycle_index')
 FROM target p JOIN program_assessment_schedules s ON s.tenant_id=p.tenant_id AND s.legal_entity_id=p.legal_entity_id AND s.program_id=p.id
 JOIN outbox_events e ON e.tenant_id=s.tenant_id AND e.aggregate_type='PROGRAM_ASSESSMENT_SCHEDULE' AND e.aggregate_id=s.id
 WHERE $4 IN (p.owner_principal_id::text,p.authority_principal_id::text)
 UNION ALL
 SELECT 'document-request:'||er.id::text,'DOCUMENT',er.id::text,er.id::text,'','','',er.title,'DOCUMENT_REQUEST_CREATED',coalesce(er.created_by::text,''),er.created_at,jsonb_build_object('request_kind','DOCUMENT_REQUEST') FROM scoped er WHERE er.purpose='Supporting documentation for the selected Program'
 UNION ALL
 SELECT 'request:'||er.id::text,'ASSESSMENT',er.id::text,er.id::text,'','','',er.title,'ASSESSMENT_CREATED',coalesce(er.created_by::text,''),er.created_at,'{}'::jsonb FROM scoped er WHERE er.origin_type='FRAMEWORK_ASSESSMENT'
 UNION ALL
 SELECT 'submission:'||s.id::text,'ASSESSMENT',er.id::text,er.id::text,s.id::text,'','',er.title,
 CASE WHEN s.assessment_revision>1 THEN 'RESPONSE_RESUBMITTED' ELSE 'RESPONSE_SUBMITTED' END,coalesce(s.submitted_by::text,''),s.submitted_at,jsonb_build_object('revision',s.assessment_revision,'request_kind',CASE WHEN er.purpose='Supporting documentation for the selected Program' THEN 'DOCUMENT_REQUEST' ELSE 'ASSESSMENT' END)
 FROM scoped er JOIN capture_submissions s ON s.tenant_id=er.tenant_id AND s.request_id=er.id WHERE er.purpose<>'Supporting documentation for the selected Program'
 UNION ALL
 SELECT 'review:'||r.id::text,'ASSESSMENT',er.id::text,er.id::text,r.submission_id::text,'','',er.title,r.status,coalesce(r.reviewed_by::text,''),r.created_at,jsonb_build_object('review_id',r.id,'rationale',r.rationale)
 FROM scoped er JOIN capture_submission_reviews r ON r.tenant_id=er.tenant_id AND r.request_id=er.id WHERE er.purpose<>'Supporting documentation for the selected Program'
 UNION ALL
 SELECT 'upload:'||a.id::text,'DOCUMENT',a.id::text,er.id::text,coalesce(binding.submission_id,''),a.id::text,coalesce(binding.field_id,''),a.file_name,'DOCUMENT_UPLOADED',coalesce(a.created_by::text,''),a.created_at,jsonb_build_object('request_kind',CASE WHEN er.purpose='Supporting documentation for the selected Program' THEN 'DOCUMENT_REQUEST' ELSE 'ASSESSMENT' END)
 FROM scoped er JOIN capture_artifacts a ON a.tenant_id=er.tenant_id AND a.request_id=er.id
 LEFT JOIN LATERAL (SELECT s.id::text AS submission_id,answer.key AS field_id FROM capture_submissions s CROSS JOIN LATERAL jsonb_each(s.answers) answer
 WHERE s.tenant_id=er.tenant_id AND s.request_id=er.id AND ((answer.value->'artifact_ids') ? a.id::text OR answer.value#>>'{document,artifact_id}'=a.id::text) ORDER BY s.submitted_at,s.id,answer.key LIMIT 1) binding ON true
 WHERE a.created_by::text=$4 OR binding.submission_id IS NOT NULL
 UNION ALL
 SELECT 'document:'||s.id::text||':'||answer.key||':'||a.id::text,'DOCUMENT',a.id::text,er.id::text,s.id::text,a.id::text,answer.key,a.file_name,'DOCUMENT_SUBMITTED',coalesce(s.submitted_by::text,''),s.submitted_at,jsonb_build_object('revision',s.assessment_revision,'request_kind',CASE WHEN er.purpose='Supporting documentation for the selected Program' THEN 'DOCUMENT_REQUEST' ELSE 'ASSESSMENT' END)
 FROM scoped er JOIN capture_submissions s ON s.tenant_id=er.tenant_id AND s.request_id=er.id
 CROSS JOIN LATERAL jsonb_each(s.answers) answer
 CROSS JOIN LATERAL (SELECT jsonb_array_elements_text(CASE WHEN jsonb_typeof(answer.value->'artifact_ids')='array' THEN answer.value->'artifact_ids' ELSE '[]'::jsonb END) AS value UNION SELECT answer.value#>>'{document,artifact_id}' WHERE nullif(answer.value#>>'{document,artifact_id}','') IS NOT NULL) aid
 JOIN capture_artifacts a ON a.id::text=aid.value AND a.tenant_id=er.tenant_id AND a.request_id=er.id
 UNION ALL
 SELECT 'document-review:'||r.id::text||':'||answer.key||':'||a.id::text,'DOCUMENT',a.id::text,er.id::text,r.submission_id::text,a.id::text,answer.key,a.file_name,
 'DOCUMENT_'||coalesce(feedback.value->>'outcome',decision.value->>'outcome',r.status),coalesce(r.reviewed_by::text,''),r.created_at,
 jsonb_build_object('request_kind',CASE WHEN er.purpose='Supporting documentation for the selected Program' THEN 'DOCUMENT_REQUEST' ELSE 'ASSESSMENT' END,'review_id',r.id,'rationale',coalesce(feedback.value->>'rationale',decision.value->>'rationale',r.rationale))
 FROM scoped er JOIN capture_submission_reviews r ON r.tenant_id=er.tenant_id AND r.request_id=er.id
 JOIN capture_submissions s ON s.id=r.submission_id AND s.tenant_id=er.tenant_id AND s.request_id=er.id
 CROSS JOIN LATERAL jsonb_each(s.answers) answer
 CROSS JOIN LATERAL (SELECT jsonb_array_elements_text(CASE WHEN jsonb_typeof(answer.value->'artifact_ids')='array' THEN answer.value->'artifact_ids' ELSE '[]'::jsonb END) AS value UNION SELECT answer.value#>>'{document,artifact_id}' WHERE nullif(answer.value#>>'{document,artifact_id}','') IS NOT NULL) aid
 JOIN capture_artifacts a ON a.id::text=aid.value AND a.tenant_id=er.tenant_id AND a.request_id=er.id
 CROSS JOIN LATERAL jsonb_array_elements(r.groups) decision
 LEFT JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(decision.value->'questions')='array' THEN decision.value->'questions' ELSE '[]'::jsonb END) feedback ON feedback.value->>'field_id'=answer.key
 WHERE (decision.value->'artifact_ids') ? a.id::text
), named AS (
 SELECT e.*,coalesce(person.display_name,CASE WHEN e.actor='' THEN 'System' ELSE 'Recorded actor' END) AS actor_name
 FROM events e LEFT JOIN principals person ON person.id::text=e.actor AND person.tenant_id=(SELECT tenant_id FROM target)
)
SELECT id,category,record_id,request_id,submission_id,artifact_id,field_id,title,event,actor,actor_name,at,details FROM named
WHERE at<=$5 AND ($13='' OR event=$13) AND ($6='' OR category=$6) AND ($7='' OR strpos(lower(concat_ws(' ',title,event,replace(event,'_',' '),actor_name,actor,record_id)),lower($7))>0)
AND ($8='' OR at>=NULLIF($8,'')::date AT TIME ZONE 'UTC') AND ($9='' OR at<(NULLIF($9,'')::date+1) AT TIME ZONE 'UTC')
AND ($10='' OR (at,id)<($11::timestamptz,$10))
ORDER BY at DESC,id DESC LIMIT $12`

func (r *PostgresRepository) ProgramActivity(ctx context.Context, actor identity.Actor, program string, q ProgramActivityQuery, now time.Time) (ProgramActivityPage, error) {
	q, c, err := activityQuery(actor, program, q, now)
	result := ProgramActivityPage{Items: []ProgramActivityEvent{}, Through: c.Through}
	if err != nil {
		return result, err
	}
	var before any
	if !c.At.IsZero() {
		before = c.At
	}
	rows, err := r.pool.Query(ctx, programActivitySQL, actor.TenantID, actor.LegalEntityID, program, actor.PrincipalID, c.Through, q.Category, q.Search, q.From, q.To, c.ID, before, q.Limit+1, q.EventType)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var e ProgramActivityEvent
		var details []byte
		if err = rows.Scan(&e.ID, &e.Category, &e.RecordID, &e.RequestID, &e.SubmissionID, &e.ArtifactID, &e.FieldID, &e.Title, &e.Event, &e.Actor, &e.ActorName, &e.At, &details); err != nil {
			return result, err
		}
		if err = json.Unmarshal(details, &e.Details); err != nil {
			return result, err
		}
		result.Items = append(result.Items, e)
	}
	if err = rows.Err(); err != nil {
		return result, err
	}
	if len(result.Items) > q.Limit {
		result.Items = result.Items[:q.Limit]
		last := result.Items[len(result.Items)-1]
		c.At, c.ID = last.At, last.ID
		raw, _ := json.Marshal(c)
		result.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	return result, nil
}
