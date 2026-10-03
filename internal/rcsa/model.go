package rcsa

import "time"

type Scope struct {
	TenantID      string
	LegalEntityID string
}

type Cycle struct {
	ID                  string    `json:"id"`
	TenantID            string    `json:"tenant_id"`
	LegalEntityID       string    `json:"legal_entity_id"`
	Code                string    `json:"code"`
	Name                string    `json:"name"`
	FormTemplateID      string    `json:"form_template_id"`
	FormTemplateVersion int64     `json:"form_template_version"`
	PeriodStart         time.Time `json:"period_start"`
	PeriodEnd           time.Time `json:"period_end"`
	DueAt               time.Time `json:"due_at"`
	ChallengeDueAt      time.Time `json:"challenge_due_at"`
	CreatedBy           string    `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
}

type Item struct {
	ID                    string    `json:"id"`
	CycleID               string    `json:"cycle_id"`
	TenantID              string    `json:"tenant_id"`
	LegalEntityID         string    `json:"legal_entity_id"`
	RiskID                string    `json:"risk_id"`
	RiskVersion           int64     `json:"risk_version"`
	RiskCode              string    `json:"risk_code"`
	RiskName              string    `json:"risk_name"`
	RespondentPrincipalID string    `json:"respondent_principal_id"`
	CreatedAt             time.Time `json:"created_at"`
}

type ItemControl struct {
	ItemID        string `json:"item_id"`
	RiskID        string `json:"risk_id"`
	ControlLinkID string `json:"control_link_id"`
}

type DistributionLink struct {
	ItemID         string    `json:"item_id"`
	DistributionID string    `json:"distribution_id"`
	IssuedAt       time.Time `json:"issued_at"`
}

type Aggregate struct {
	Cycle         Cycle                       `json:"cycle"`
	Items         []Item                      `json:"items"`
	Controls      map[string][]ItemControl    `json:"controls"`
	Distributions map[string]DistributionLink `json:"distributions"`
}

type CreateInput struct {
	TenantID            string
	LegalEntityID       string
	Code                string
	Name                string
	FormTemplateID      string
	FormTemplateVersion int64
	PeriodStart         time.Time
	PeriodEnd           time.Time
	DueAt               time.Time
	ChallengeDueAt      time.Time
	RiskIDs             []string
	ActorID             string
}
