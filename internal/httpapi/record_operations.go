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

type matterOperationsResponse struct {
	MatterID           string            `json:"matter_id"`
	MatterVersion      int64             `json:"matter_version"`
	AuthorityAvailable bool              `json:"authority_available"`
	Operations         []RecordOperation `json:"operations"`
	GeneratedAt        time.Time         `json:"generated_at"`
}
