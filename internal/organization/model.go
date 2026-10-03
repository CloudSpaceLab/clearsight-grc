package organization

import "time"

type ScopeKind string

const (
	ScopeKindOrganizationUnit ScopeKind = "ORGANIZATION_UNIT"
	ScopeKindBranch           ScopeKind = "BRANCH"
	ScopeKindDepartment       ScopeKind = "DEPARTMENT"
	ScopeKindFunction         ScopeKind = "FUNCTION"
	ScopeKindBusinessUnit     ScopeKind = "BUSINESS_UNIT"
	ScopeKindCriticalService  ScopeKind = "CRITICAL_SERVICE"
)

type ScopeOrigin string

const (
	ScopeOriginLegacyDepartmentPath ScopeOrigin = "LEGACY_DEPARTMENT_PATH"
	ScopeOriginManaged              ScopeOrigin = "MANAGED"
)

type ScopeStatus string

const (
	ScopeStatusActive  ScopeStatus = "ACTIVE"
	ScopeStatusRetired ScopeStatus = "RETIRED"
)

type Scope struct {
	ID             string      `json:"id"`
	TenantID       string      `json:"tenant_id,omitempty"`
	LegalEntityID  string      `json:"legal_entity_id"`
	ParentScopeID  string      `json:"parent_scope_id,omitempty"`
	Code           string      `json:"code"`
	Name           string      `json:"name"`
	Kind           ScopeKind   `json:"kind"`
	DepartmentPath []string    `json:"department_path"`
	Origin         ScopeOrigin `json:"origin"`
	Status         ScopeStatus `json:"status"`
	ValidFrom      time.Time   `json:"valid_from"`
	ValidUntil     *time.Time  `json:"valid_until,omitempty"`
	Version        int64       `json:"version"`
}

type ScopePage struct {
	Items     []Scope `json:"items"`
	Truncated bool    `json:"truncated"`
}
