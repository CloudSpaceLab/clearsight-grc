package authority

import (
	"context"
	"time"
)

type Responsibility string

const (
	ResponsibilityPerformer    Responsibility = "PERFORMER"
	ResponsibilityOwner        Responsibility = "ACCOUNTABLE_OWNER"
	ResponsibilityProposer     Responsibility = "PROPOSER"
	ResponsibilityReviewer     Responsibility = "REVIEWER"
	ResponsibilityChallenger   Responsibility = "INDEPENDENT_CHALLENGER"
	ResponsibilityAuthorizer   Responsibility = "AUTHORIZER"
	ResponsibilitySignatory    Responsibility = "SIGNATORY"
	ResponsibilityTransmitter  Responsibility = "TRANSMITTER"
	ResponsibilityAcknowledger Responsibility = "ACKNOWLEDGEMENT_RECORDER"
	ResponsibilityEscalation   Responsibility = "ESCALATION_OWNER"
)

type Principal struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Kind        string `json:"kind"`
	Role        string `json:"role"`
}

type Rule struct {
	ID                  string
	TenantID            string
	LegalEntityID       string
	ObjectType          string
	ObjectID            string
	Responsibility      Responsibility
	DecisionType        string
	MinMateriality      int
	Principal           Principal
	CandidatePrincipals []Principal
	ResolutionStrategy  string
	Priority            int
	ValidFrom           time.Time
	ValidUntil          time.Time
}

type ResolveInput struct {
	TenantID       string         `json:"tenant_id"`
	LegalEntityID  string         `json:"legal_entity_id"`
	ObjectType     string         `json:"object_type"`
	ObjectID       string         `json:"object_id"`
	Responsibility Responsibility `json:"responsibility"`
	DecisionType   string         `json:"decision_type,omitempty"`
	Materiality    int            `json:"materiality"`
	At             time.Time      `json:"at,omitempty"`
}

type Resolution struct {
	Principal           Principal         `json:"principal"`
	CandidatePrincipals []Principal       `json:"candidate_principals,omitempty"`
	EffectiveOrigins    []EffectiveOrigin `json:"effective_origins,omitempty"`
	Strategy            string            `json:"strategy,omitempty"`
	RuleID              string            `json:"rule_id"`
	PolicyVersion       string            `json:"policy_version"`
	Explanation         string            `json:"explanation"`
}

// EffectiveOrigin preserves why an effective principal belongs to a route.
// Direct candidates act for themselves; an active delegate carries the route
// seed principal whose responsibility was delegated.
type EffectiveOrigin struct {
	PrincipalID       string `json:"principal_id"`
	OriginPrincipalID string `json:"origin_principal_id"`
}

func (r Resolution) AllowsPrincipal(principalID string) bool {
	if principalID == "" {
		return false
	}
	if r.Principal.ID == principalID {
		return true
	}
	for _, candidate := range r.CandidatePrincipals {
		if candidate.ID == principalID {
			return true
		}
	}
	return false
}

// AllowsPrincipalFor distinguishes a delegate of one stored authority from an
// unrelated person who happens to be eligible elsewhere in the same route.
func (r Resolution) AllowsPrincipalFor(principalID, originPrincipalID string) bool {
	if principalID == "" || originPrincipalID == "" || !r.AllowsPrincipal(principalID) {
		return false
	}
	if principalID == originPrincipalID {
		return true
	}
	for _, origin := range r.EffectiveOrigins {
		if origin.PrincipalID == principalID && origin.OriginPrincipalID == originPrincipalID {
			return true
		}
	}
	return false
}

type Candidate struct {
	Principal Principal `json:"principal"`
	RuleID    string    `json:"rule_id"`
	Priority  int       `json:"priority"`
	Eligible  bool      `json:"eligible"`
	Reason    string    `json:"reason"`
}

type Simulation struct {
	Selected      *Resolution `json:"selected,omitempty"`
	Candidates    []Candidate `json:"candidates"`
	PolicyVersion string      `json:"policy_version"`
}

type IntegrityFinding struct {
	Type           string   `json:"type"`
	Severity       string   `json:"severity"`
	Summary        string   `json:"summary"`
	RequiredAction string   `json:"required_action"`
	RuleIDs        []string `json:"rule_ids,omitempty"`
}

type PolicySummary struct {
	ID            string     `json:"id"`
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	Status        string     `json:"status"`
	Version       int        `json:"version"`
	EffectiveFrom *time.Time `json:"effective_from,omitempty"`
}

type Service interface {
	Resolve(context.Context, ResolveInput) (Resolution, error)
	Simulate(context.Context, ResolveInput) (Simulation, error)
	Integrity(context.Context, string) ([]IntegrityFinding, error)
	Policies(context.Context, string) ([]PolicySummary, error)
}
