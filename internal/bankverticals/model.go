package bankverticals

import "time"

type Code string

const (
	JourneyNDPAContinuous     Code = "NDPA_CONTINUOUS"
	JourneyRegulatoryChange   Code = "REGULATORY_CHANGE"
	JourneyAuthorityRequest   Code = "AUTHORITY_REQUEST"
	JourneyFindingRemediation Code = "FINDING_REMEDIATION"

	ActionTargetProgram         = "PROGRAM"
	ActionTargetMatter          = "MATTER"
	ActionTargetEvidenceRequest = "EVIDENCE_REQUEST"
)

type Step struct {
	Code     string `json:"code"`
	Label    string `json:"label"`
	Complete bool   `json:"complete"`
}

type Journey struct {
	Code                    Code       `json:"code"`
	Title                   string     `json:"title"`
	Summary                 string     `json:"summary"`
	Status                  string     `json:"status"`
	StatusLabel             string     `json:"status_label"`
	NextAction              string     `json:"next_action"`
	Owner                   string     `json:"owner"`
	OwnerPrincipalID        string     `json:"owner_principal_id,omitempty"`
	ProgramID               string     `json:"program_id,omitempty"`
	MatterID                string     `json:"matter_id,omitempty"`
	EvidenceRequestID       string     `json:"evidence_request_id,omitempty"`
	ActionTargetType        string     `json:"action_target_type,omitempty"`
	ActionTargetID          string     `json:"action_target_id,omitempty"`
	ActionLabel             string     `json:"action_label,omitempty"`
	ActionAvailable         bool       `json:"action_available"`
	ActionUnavailableReason string     `json:"action_unavailable_reason,omitempty"`
	DueAt                   *time.Time `json:"due_at,omitempty"`
	CompletedSteps          int        `json:"completed_steps"`
	TotalSteps              int        `json:"total_steps"`
	Steps                   []Step     `json:"steps"`
	SourceNames             []string   `json:"source_names"`
	Sensitive               bool       `json:"sensitive"`
	Sample                  bool       `json:"sample"`
	UpdatedAt               *time.Time `json:"updated_at,omitempty"`
	AllowedPrincipalIDs     []string   `json:"-"`
}

type SeedConfig struct {
	TenantID               string
	LegalEntityID          string
	BankName               string
	ActorID                string
	OwnerPrincipalID       string
	ContributorPrincipalID string
	ReviewerPrincipalID    string
	SignatoryPrincipalID   string
	Now                    time.Time
}

func DemoSeedConfig() SeedConfig {
	return SeedConfig{
		TenantID:               "bank-demo",
		LegalEntityID:          "bank-ng",
		BankName:               "Clear Bank Nigeria",
		ActorID:                "user-demo",
		OwnerPrincipalID:       "owner-demo",
		ContributorPrincipalID: "contributor-demo",
		ReviewerPrincipalID:    "reviewer-demo",
		SignatoryPrincipalID:   "signatory-demo",
	}
}

func (j Journey) VisibleTo(principalID string) bool {
	if !j.Sensitive {
		return true
	}
	for _, allowed := range j.AllowedPrincipalIDs {
		if allowed == principalID {
			return true
		}
	}
	return false
}
