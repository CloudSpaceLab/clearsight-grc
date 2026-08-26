package httpapi

import (
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
)

type RecordOperation struct {
	Command        string                `json:"command"`
	SubresourceID  string                `json:"subresource_id,omitempty"`
	Label          string                `json:"label"`
	Responsibility string                `json:"responsibility"`
	CanAct         bool                  `json:"can_act"`
	AssignedTo     *authority.Principal  `json:"assigned_to,omitempty"`
	Candidates     []authority.Principal `json:"candidates,omitempty"`
	Reason         string                `json:"reason"`
	AllowedTargets []string              `json:"allowed_targets,omitempty"`
}

// RecordResponsibleParty preserves a safe human label for responsibility that
// belonged to a stored record. It intentionally omits the principal identifier:
// commands use current authority resolution, while this value is read-only
// reconstruction context.
type RecordResponsibleParty struct {
	Scope          string `json:"scope"`
	SubresourceID  string `json:"subresource_id,omitempty"`
	Responsibility string `json:"responsibility"`
	DisplayName    string `json:"display_name"`
	Kind           string `json:"kind,omitempty"`
}

type matterOperationsResponse struct {
	MatterID                     string                   `json:"matter_id"`
	MatterVersion                int64                    `json:"matter_version"`
	AuthorityAvailable           bool                     `json:"authority_available"`
	Operations                   []RecordOperation        `json:"operations"`
	ResponsibleParties           []RecordResponsibleParty `json:"responsible_parties,omitempty"`
	ResponsibilityLabelsComplete bool                     `json:"responsibility_labels_complete"`
	GeneratedAt                  time.Time                `json:"generated_at"`
}

type programOperationsResponse struct {
	ProgramID                    string                   `json:"program_id"`
	ProgramVersion               int64                    `json:"program_version"`
	AuthorityAvailable           bool                     `json:"authority_available"`
	Operations                   []RecordOperation        `json:"operations"`
	ResponsibleParties           []RecordResponsibleParty `json:"responsible_parties,omitempty"`
	ResponsibilityLabelsComplete bool                     `json:"responsibility_labels_complete"`
	GeneratedAt                  time.Time                `json:"generated_at"`
}
