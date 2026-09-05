package httpapi

import (
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

// These DTOs are the public form-distribution contract. Keep domain aggregates
// behind this boundary so newly added Go fields cannot silently become HTTP
// response fields with Go-style names.
type formDistributionResponse struct {
	ID                  string                      `json:"id"`
	TenantID            string                      `json:"tenant_id"`
	LegalEntityID       string                      `json:"legal_entity_id"`
	FormTemplateID      string                      `json:"form_template_id"`
	FormTemplateVersion int64                       `json:"form_template_version"`
	SubjectType         string                      `json:"subject_type"`
	SubjectID           string                      `json:"subject_id"`
	Title               string                      `json:"title"`
	Purpose             string                      `json:"purpose"`
	AccessPolicy        evidence.AccessPolicy       `json:"access_policy"`
	Status              evidence.DistributionStatus `json:"status"`
	Deadline            time.Time                   `json:"deadline"`
	RouteExpiresAt      time.Time                   `json:"route_expires_at"`
	ReminderPolicy      map[string]any              `json:"reminder_policy,omitempty"`
	CreatedBy           string                      `json:"created_by"`
	Version             int64                       `json:"version"`
	CreatedAt           time.Time                   `json:"created_at"`
	UpdatedAt           time.Time                   `json:"updated_at"`
}

type distributionRecipientResponse struct {
	ID             string                              `json:"id"`
	DistributionID string                              `json:"distribution_id"`
	TenantID       string                              `json:"tenant_id"`
	LegalEntityID  string                              `json:"legal_entity_id"`
	Role           evidence.RecipientRole              `json:"role"`
	Type           evidence.RecipientType              `json:"type"`
	PrincipalID    string                              `json:"principal_id,omitempty"`
	RequestID      string                              `json:"request_id,omitempty"`
	AudienceHint   string                              `json:"audience_hint,omitempty"`
	ContactLabel   string                              `json:"contact_label,omitempty"`
	State          evidence.DistributionRecipientState `json:"state"`
	Version        int64                               `json:"version"`
	CreatedAt      time.Time                           `json:"created_at"`
	UpdatedAt      time.Time                           `json:"updated_at"`
}

type responseWorkspaceResponse struct {
	ID             string                           `json:"id"`
	TenantID       string                           `json:"tenant_id"`
	LegalEntityID  string                           `json:"legal_entity_id"`
	DistributionID string                           `json:"distribution_id"`
	Status         evidence.ResponseWorkspaceStatus `json:"status"`
	Version        int64                            `json:"version"`
	CreatedAt      time.Time                        `json:"created_at"`
	UpdatedAt      time.Time                        `json:"updated_at"`
}

type distributionBundleResponse struct {
	Distribution formDistributionResponse        `json:"distribution"`
	Recipients   []distributionRecipientResponse `json:"recipients"`
	Workspace    responseWorkspaceResponse       `json:"workspace"`
}

type distributionPageResponse struct {
	Items      []formDistributionResponse `json:"items"`
	NextCursor string                     `json:"next_cursor,omitempty"`
}

type responseRevisionResponse struct {
	ID                   string                         `json:"id"`
	Revision             int64                          `json:"revision"`
	SupersedesRevisionID string                         `json:"supersedes_revision_id,omitempty"`
	AchievedAssurance    evidence.AccessAssurance       `json:"achieved_assurance"`
	SignoffSummary       map[string]any                 `json:"signoff_summary"`
	Score                any                            `json:"score,omitempty"`
	ComplianceScore      *float64                       `json:"compliance_score,omitempty"`
	ScoredWeightCoverage float64                        `json:"scored_weight_coverage"`
	State                evidence.ResponseRevisionState `json:"state"`
	CriticalFieldResults []map[string]any               `json:"critical_field_results"`
	ScoringPolicyVersion string                         `json:"scoring_policy_version"`
	Current              bool                           `json:"current"`
	CreatedAt            time.Time                      `json:"created_at"`
}

func formDistributionDTO(value evidence.FormDistribution) formDistributionResponse {
	return formDistributionResponse{
		ID: value.ID, TenantID: value.TenantID, LegalEntityID: value.LegalEntityID,
		FormTemplateID: value.FormTemplateID, FormTemplateVersion: value.FormTemplateVersion,
		SubjectType: value.SubjectType, SubjectID: value.SubjectID, Title: value.Title, Purpose: value.Purpose,
		AccessPolicy: value.AccessPolicy, Status: value.Status, Deadline: value.Deadline,
		RouteExpiresAt: value.RouteExpiresAt, ReminderPolicy: value.ReminderPolicy, CreatedBy: value.CreatedBy,
		Version: value.Version, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func distributionRecipientDTO(value evidence.DistributionRecipient) distributionRecipientResponse {
	return distributionRecipientResponse{
		ID: value.ID, DistributionID: value.DistributionID, TenantID: value.TenantID, LegalEntityID: value.LegalEntityID,
		Role: value.Role, Type: value.Type, PrincipalID: value.PrincipalID, RequestID: value.RequestID,
		AudienceHint: value.AudienceHint, ContactLabel: value.ContactLabel, State: value.State,
		Version: value.Version, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func responseWorkspaceDTO(value evidence.ResponseWorkspace) responseWorkspaceResponse {
	return responseWorkspaceResponse{
		ID: value.ID, TenantID: value.TenantID, LegalEntityID: value.LegalEntityID,
		DistributionID: value.DistributionID, Status: value.Status, Version: value.Version,
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func distributionBundleJSON(bundle evidence.DistributionBundle) map[string]any {
	response := distributionBundleDTO(bundle)
	return map[string]any{
		"distribution": response.Distribution,
		"recipients":   response.Recipients,
		"workspace":    response.Workspace,
	}
}

func distributionBundleDTO(bundle evidence.DistributionBundle) distributionBundleResponse {
	recipients := make([]distributionRecipientResponse, 0, len(bundle.Recipients))
	for _, recipient := range bundle.Recipients {
		recipients = append(recipients, distributionRecipientDTO(recipient))
	}
	return distributionBundleResponse{
		Distribution: formDistributionDTO(bundle.Distribution),
		Recipients:   recipients,
		Workspace:    responseWorkspaceDTO(bundle.Workspace),
	}
}

func distributionPageDTO(page evidence.DistributionPage) distributionPageResponse {
	items := make([]formDistributionResponse, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, formDistributionDTO(item))
	}
	return distributionPageResponse{Items: items, NextCursor: page.NextCursor}
}

func responseRevisionJSON(value evidence.ResponseRevision) responseRevisionResponse {
	return responseRevisionResponse{
		ID: value.ID, Revision: value.Revision, SupersedesRevisionID: value.SupersedesRevisionID,
		AchievedAssurance: value.AchievedAssurance, SignoffSummary: value.SignoffSummary,
		Score: responseScoreJSON(value.Score, true), ComplianceScore: value.ComplianceScore, ScoredWeightCoverage: value.ScoredWeightCoverage,
		State: value.State, CriticalFieldResults: value.CriticalFieldResults,
		ScoringPolicyVersion: value.ScoringPolicyVersion, Current: value.Current, CreatedAt: value.CreatedAt,
	}
}
