package onboarding

import "time"

type Surface string

const (
	SurfaceToday   Surface = "TODAY"
	SurfaceVendors Surface = "VENDORS"
)

type Step struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action,omitempty"`
	View        string `json:"view,omitempty"`
	Target      string `json:"target,omitempty"`
	Intent      string `json:"intent,omitempty"`
	Optional    bool   `json:"optional,omitempty"`
}

type Guide struct {
	Code         string   `json:"code"`
	Surface      Surface  `json:"surface"`
	Profile      string   `json:"profile"`
	Role         string   `json:"role"`
	RoleCodes    []string `json:"role_codes,omitempty"`
	Priority     int      `json:"priority,omitempty"`
	Version      int      `json:"version"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Illustration string   `json:"illustration"`
	Steps        []Step   `json:"steps"`
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
