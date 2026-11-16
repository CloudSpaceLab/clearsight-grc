//go:build postgres

package monitoring

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"strconv"
)

func (r *PostgresRepository) AdoptedAssessmentOptions(ctx context.Context, actor Actor, programID string) ([]AdoptedAssessmentOption, error) {
	var owner, reviewer, reviewerName, status string
	var version int64
	err := r.pool.QueryRow(ctx, `SELECT p.owner_principal_id::text,p.authority_principal_id::text,reviewer.display_name,p.status,p.version
 FROM programs p JOIN tenants t ON t.id=p.tenant_id JOIN principals reviewer ON reviewer.tenant_id=p.tenant_id AND reviewer.id=p.authority_principal_id
 WHERE (t.id::text=$1 OR t.slug=$1) AND p.legal_entity_id=$2::uuid AND p.id=$3::uuid`, actor.TenantID, actor.LegalEntityID, programID).Scan(&owner, &reviewer, &reviewerName, &status, &version)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	if actor.PrincipalID != owner && actor.PrincipalID != reviewer {
		return nil, ErrNotFound
	}
	rows, err := r.pool.Query(ctx, `SELECT a.id::text,a.template_code,a.configuration,
 NOT EXISTS(SELECT 1 FROM jsonb_array_elements(a.configuration->'requirements') req WHERE NOT EXISTS(
 SELECT 1 FROM program_requirements r WHERE r.tenant_id=a.tenant_id AND r.program_id=a.program_id AND r.code=a.template_code||'-'||(req->>'code') AND r.status='APPROVED'))
 FROM program_framework_adoptions a JOIN tenants t ON t.id=a.tenant_id
 WHERE (t.id::text=$1 OR t.slug=$1) AND a.legal_entity_id=$2::uuid AND a.program_id=$3::uuid AND a.status='ACTIVE' AND a.template_code IS NOT NULL ORDER BY a.id`, actor.TenantID, actor.LegalEntityID, programID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	type adoption struct {
		ID, Code             string
		Config               []byte
		RequirementsApproved bool
	}
	var adoptions []adoption
	for rows.Next() {
		var a adoption
		if err = rows.Scan(&a.ID, &a.Code, &a.Config, &a.RequirementsApproved); err != nil {
			rows.Close()
			return nil, err
		}
		adoptions = append(adoptions, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	out := []AdoptedAssessmentOption{}
	for _, a := range adoptions {
		var config struct {
			Requirements []struct {
				FormID  string `json:"form_template_id"`
				Version int64  `json:"form_template_version"`
			}
			Units []string `json:"respondent_scope_ids"`
		}
		if err = json.Unmarshal(a.Config, &config); err != nil {
			return nil, ErrInvalid
		}
		units, _ := json.Marshal(config.Units)
		people, err := r.pool.Query(ctx, `SELECT person.id::text,person.display_name FROM principals person JOIN tenants t ON t.id=person.tenant_id
   WHERE (t.id::text=$1 OR t.slug=$1) AND person.id<>$3::uuid AND framework_assessment_recipient_eligible(person.tenant_id,$2::uuid,person.id,$4::jsonb)
   ORDER BY person.display_name,person.id LIMIT 101`, actor.TenantID, actor.LegalEntityID, reviewer, string(units))
		if err != nil {
			return nil, err
		}
		respondents := []AssessmentRespondent{}
		for people.Next() {
			var p AssessmentRespondent
			if err = people.Scan(&p.ID, &p.Name); err != nil {
				people.Close()
				return nil, err
			}
			respondents = append(respondents, p)
		}
		err = people.Err()
		people.Close()
		if err != nil {
			return nil, err
		}
		if len(respondents) > 100 {
			return nil, ErrInvalid
		}
		seen := map[string]bool{}
		for _, req := range config.Requirements {
			key := req.FormID + ":" + strconv.FormatInt(req.Version, 10)
			if seen[key] {
				continue
			}
			seen[key] = true
			form, e := r.ReusableFormRevision(ctx, actor.TenantID, actor.LegalEntityID, req.FormID, req.Version)
			if e != nil && e != pgx.ErrNoRows {
				return nil, e
			}
			if e != nil {
				continue
			}
			option := AdoptedAssessmentOption{AdoptionID: a.ID, TemplateCode: a.Code, FormID: form.ID, FormVersion: form.Version, Name: form.Name, ProgramVersion: version, ReviewerID: reviewer, ReviewerName: reviewerName, Respondents: respondents, Ready: true}
			switch {
			case actor.PrincipalID != owner:
				option.Blocker = "The assigned Program Owner creates the assessment."
			case status != "ACTIVE":
				option.Blocker = "Review and activate the Program before creating an assessment."
			case !a.RequirementsApproved:
				option.Blocker = "Resolve the Program's draft requirement reviews before creating an assessment."
			case form.Status != LifecycleActive && form.Status != LifecycleRetired:
				option.Blocker = "The adopted questionnaire revision is not approved."
			case len(respondents) == 0:
				option.Blocker = "No active people are assigned to the adopted respondent units. Update organizational assignments first."
			}
			option.Ready = option.Blocker == ""
			out = append(out, option)
		}
	}
	return out, nil
}
