package httpapi

import (
	"context"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

// Display labels belong to the authorized read, not the immutable decision.
type assessmentDecisionRead struct {
	evidence.FieldAssessmentDecision
	ReviewerDisplayName string `json:"reviewer_display_name,omitempty"`
}
type assessmentFieldRead struct {
	evidence.ResponseAssessmentField
	Decision *assessmentDecisionRead `json:"decision,omitempty"`
}
type responseAssessmentRead struct {
	evidence.ResponseAssessment
	Fields []assessmentFieldRead `json:"fields"`
}
type responseApplicationReceiptRead struct {
	thirdparty.ResponseApplicationReceipt
	ActorDisplayName string `json:"actor_display_name,omitempty"`
}
type vendorAssessmentReviewRead struct {
	thirdparty.AssessmentReviewView
	ApplicationReceipt *responseApplicationReceiptRead `json:"application_receipt,omitempty"`
}
type vendorAssessmentApplicationRead struct {
	Receipt responseApplicationReceiptRead `json:"receipt"`
	Review  vendorAssessmentReviewRead     `json:"review"`
}

func (a *API) responseAssessmentWithLabels(ctx context.Context, actor identity.Actor, entity string, value evidence.ResponseAssessment) responseAssessmentRead {
	result := responseAssessmentRead{ResponseAssessment: value, Fields: make([]assessmentFieldRead, 0, len(value.Fields))}
	ids := make([]string, 0, len(value.Fields))
	for _, field := range value.Fields {
		if field.Decision != nil {
			ids = append(ids, field.Decision.ReviewerID)
		}
	}
	labels := a.exactAssessmentLabels(ctx, actor, entity, ids)
	for _, field := range value.Fields {
		read := assessmentFieldRead{ResponseAssessmentField: field}
		if field.Decision != nil {
			read.Decision = &assessmentDecisionRead{FieldAssessmentDecision: *field.Decision, ReviewerDisplayName: labels[field.Decision.ReviewerID]}
		}
		result.Fields = append(result.Fields, read)
	}
	return result
}

func (a *API) vendorAssessmentReviewWithLabels(ctx context.Context, actor thirdparty.Actor, value thirdparty.AssessmentReviewView) vendorAssessmentReviewRead {
	result := vendorAssessmentReviewRead{AssessmentReviewView: value}
	if value.ApplicationReceipt != nil {
		labels := a.exactAssessmentLabels(ctx, identity.Actor{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID}, value.Assessment.LegalEntityID, []string{value.ApplicationReceipt.ActorPrincipalID})
		result.ApplicationReceipt = &responseApplicationReceiptRead{ResponseApplicationReceipt: *value.ApplicationReceipt, ActorDisplayName: labels[value.ApplicationReceipt.ActorPrincipalID]}
	}
	return result
}

// Call only after the domain read/command authorizes the assessment. Resolve
// stored actor IDs in that record's exact entity, never a current role/owner.
// Failure affects only optional labels, including after a committed command.
func (a *API) exactAssessmentLabels(ctx context.Context, actor identity.Actor, entity string, ids []string) map[string]string {
	labels := map[string]string{}
	if a == nil || a.deps.Access == nil || actor.TenantID == "" || actor.PrincipalID == "" || entity == "" || entity == "*" || (actor.LegalEntityID != entity && actor.LegalEntityID != "*") {
		return labels
	}
	unique := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if strings.TrimSpace(id) == "" || seen[id] {
			continue
		}
		if len(unique) >= access.MaxPrincipalBatchSize {
			break
		}
		seen[id] = true
		unique = append(unique, id)
	}
	accept := func(id string, resolved access.Resolution, err error) {
		if err == nil && resolved.TenantID == actor.TenantID && resolved.LegalEntityID == entity && resolved.PrincipalID == id && strings.TrimSpace(resolved.DisplayName) != "" {
			labels[id] = resolved.DisplayName
		}
	}
	if len(unique) == 0 {
		return labels
	}
	if batch, ok := a.deps.Access.(access.BatchPrincipalResolver); ok {
		outcomes, err := batch.ResolvePrincipals(ctx, actor.TenantID, entity, unique)
		if err != nil || len(outcomes) != len(unique) {
			return labels
		}
		for i, outcome := range outcomes {
			accept(unique[i], outcome.Resolution, outcome.Err)
		}
		return labels
	}
	for _, id := range unique {
		resolved, err := a.deps.Access.ResolvePrincipal(ctx, actor.TenantID, id, entity)
		accept(id, resolved, err)
	}
	return labels
}
