package authority

import "time"

type Responsibility string

const (
	ResponsibilityPerformer  Responsibility = "PERFORMER"
	ResponsibilityOwner      Responsibility = "ACCOUNTABLE_OWNER"
	ResponsibilityReviewer   Responsibility = "REVIEWER"
	ResponsibilityChallenger Responsibility = "INDEPENDENT_CHALLENGER"
	ResponsibilityAuthorizer Responsibility = "AUTHORIZER"
	ResponsibilitySignatory  Responsibility = "SIGNATORY"
	ResponsibilityEscalation Responsibility = "ESCALATION_OWNER"
)

type Principal struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Kind        string `json:"kind"`
	Role        string `json:"role"`
}

type Rule struct {
	ID             string
	TenantID       string
	LegalEntityID  string
	ObjectType     string
	ObjectID       string
	Responsibility Responsibility
	MinMateriality int
	Principal      Principal
	Priority       int
	ValidFrom      time.Time
	ValidUntil     time.Time
}

type ResolveInput struct {
	TenantID       string         `json:"tenant_id"`
	LegalEntityID  string         `json:"legal_entity_id"`
	ObjectType     string         `json:"object_type"`
	ObjectID       string         `json:"object_id"`
	Responsibility Responsibility `json:"responsibility"`
	Materiality    int            `json:"materiality"`
	At             time.Time      `json:"at,omitempty"`
}

type Resolution struct {
	Principal     Principal `json:"principal"`
	RuleID        string    `json:"rule_id"`
	PolicyVersion string    `json:"policy_version"`
	Explanation   string    `json:"explanation"`
}
