package onboarding

import "time"

type Step struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action,omitempty"`
	Target      string `json:"target,omitempty"`
}

type Guide struct {
	Code         string `json:"code"`
	Role         string `json:"role"`
	Version      int    `json:"version"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Illustration string `json:"illustration"`
	Steps        []Step `json:"steps"`
}

type State struct {
	TenantID     string    `json:"tenant_id"`
	PrincipalID  string    `json:"principal_id"`
	GuideCode    string    `json:"guide_code"`
	GuideVersion int       `json:"guide_version"`
	CurrentStep  int       `json:"current_step"`
	Completed    bool      `json:"completed"`
	Dismissed    bool      `json:"dismissed"`
	UpdatedAt    time.Time `json:"updated_at"`
	Version      int64     `json:"version"`
}

type UpdateInput struct {
	CurrentStep     int   `json:"current_step"`
	Completed       bool  `json:"completed"`
	Dismissed       bool  `json:"dismissed"`
	ExpectedVersion int64 `json:"expected_version"`
}
