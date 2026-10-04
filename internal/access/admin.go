package access

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/organization"
)

var (
	ErrAdminNotFound     = errors.New("identity access object not found")
	ErrAdminConflict     = errors.New("identity access object conflicts with current state")
	ErrAdminInvalid      = errors.New("identity access input is invalid")
	ErrAdminMakerChecker = errors.New("a different administrator must approve this change")
)

type SCIMSourceSummary struct {
	ID               string     `json:"id"`
	Code             string     `json:"code"`
	Status           string     `json:"status"`
	IdentityIssuer   string     `json:"identity_issuer,omitempty"`
	SubjectAttribute string     `json:"subject_attribute"`
	ActiveUsers      int        `json:"active_users"`
	ActiveGroups     int        `json:"active_groups"`
	LastActivityAt   *time.Time `json:"last_activity_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type PersonSummary struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
	UserName    string `json:"user_name,omitempty"`
	SourceCode  string `json:"source_code,omitempty"`
	SourceState string `json:"source_state,omitempty"`
}

type GroupSummary struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	ExternalID  string `json:"external_id,omitempty"`
	SourceCode  string `json:"source_code"`
	SourceState string `json:"source_state"`
	MemberCount int    `json:"member_count"`
}

type WorkspaceRoleMaterialReferences struct {
	Responsibilities int `json:"responsibilities"`
	AuthorityGrants   int `json:"authority_grants"`
	RoutingPolicies   int `json:"routing_policies"`
	SegregationRules  int `json:"segregation_rules"`
}

type RoleTemplateSummary struct {
	ID                 string                          `json:"id"`
	Code               string                          `json:"code"`
	Name               string                          `json:"name"`
	Capabilities       []string                        `json:"capabilities"`
	WorkspaceEditable  bool                            `json:"workspace_editable"`
	WorkspaceLockReason string                          `json:"workspace_lock_reason,omitempty"`
	MaterialReferences WorkspaceRoleMaterialReferences `json:"material_references"`
}

type LegalEntitySummary struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type GroupRoleBindingSummary struct {
	ID                  string     `json:"id"`
	GroupID             string     `json:"group_id"`
	GroupName           string     `json:"group_name"`
	RoleTemplateID      string     `json:"role_template_id"`
	RoleCode            string     `json:"role_code"`
	LegalEntityID       string     `json:"legal_entity_id"`
	LegalEntity         string     `json:"legal_entity"`
	DepartmentPath      []string   `json:"department_path"`
	OrganizationScopeID string     `json:"organization_scope_id,omitempty"`
	ValidFrom           time.Time  `json:"valid_from"`
	ValidUntil          *time.Time `json:"valid_until,omitempty"`
}

type PositionSummary struct {
	ID                  string     `json:"id"`
	Code                string     `json:"code"`
	Title               string     `json:"title"`
	FunctionName        string     `json:"function_name,omitempty"`
	DepartmentPath      []string   `json:"department_path"`
	OrganizationScopeID string     `json:"organization_scope_id,omitempty"`
	ParentPositionID    string     `json:"parent_position_id,omitempty"`
	ParentPositionCode  string     `json:"parent_position_code,omitempty"`
	ParentPositionTitle string     `json:"parent_position_title,omitempty"`
	OccupantPrincipalID string     `json:"occupant_principal_id,omitempty"`
	OccupantName        string     `json:"occupant_name,omitempty"`
	OccupantStatus      string     `json:"occupant_status,omitempty"`
	RoleCodes           []string   `json:"role_codes"`
	ValidFrom           time.Time  `json:"valid_from"`
	ValidUntil          *time.Time `json:"valid_until,omitempty"`
	Version             int64      `json:"version"`
}

type OrganizationPositionRoleOperation string

const (
	OrganizationPositionRoleAdd    OrganizationPositionRoleOperation = "ADD"
	OrganizationPositionRoleRetire OrganizationPositionRoleOperation = "RETIRE"
)

type OrganizationPositionRoleRevisionSummary struct {
	ID                  string                            `json:"id"`
	PositionID          string                            `json:"position_id"`
	RoleTemplateID      string                            `json:"role_template_id"`
	RoleTemplateVersion int64                             `json:"role_template_version"`
	Operation           OrganizationPositionRoleOperation `json:"operation"`
	BasePositionVersion int64                             `json:"base_position_version"`
	RoleCode            string                            `json:"role_code"`
	RoleName            string                            `json:"role_name"`
	Capabilities        []string                          `json:"capabilities"`
	MakerID             string                            `json:"maker_id"`
	CheckerID           string                            `json:"checker_id,omitempty"`
	Status              string                            `json:"status"`
	Rationale           string                            `json:"rationale,omitempty"`
	CreatedAt           time.Time                         `json:"created_at"`
	DecidedAt           *time.Time                        `json:"decided_at,omitempty"`
	AppliedAt           *time.Time                        `json:"applied_at,omitempty"`
}

type ProposeOrganizationPositionRoleInput struct {
	TenantID                string                            `json:"tenant_id"`
	LegalEntityID           string                            `json:"legal_entity_id"`
	PositionID              string                            `json:"position_id"`
	RoleTemplateID          string                            `json:"role_template_id"`
	Operation               OrganizationPositionRoleOperation `json:"operation"`
	ExpectedPositionVersion int64                             `json:"expected_position_version"`
	ActorID                 string                            `json:"-"`
}

type DecideOrganizationPositionRoleInput struct {
	TenantID      string `json:"tenant_id"`
	LegalEntityID string `json:"legal_entity_id"`
	RevisionID    string `json:"revision_id"`
	ActorID       string `json:"-"`
	Rationale     string `json:"rationale"`
}

type OrganizationPositionOperation string

const (
	OrganizationPositionCreate OrganizationPositionOperation = "CREATE"
	OrganizationPositionUpdate OrganizationPositionOperation = "UPDATE"
	OrganizationPositionRetire OrganizationPositionOperation = "RETIRE"
)

type OrganizationPositionState struct {
	Code                string `json:"code"`
	Title               string `json:"title"`
	FunctionName        string `json:"function_name,omitempty"`
	OrganizationScopeID string `json:"organization_scope_id,omitempty"`
	ParentPositionID    string `json:"parent_position_id,omitempty"`
	OccupantPrincipalID string `json:"occupant_principal_id,omitempty"`
}

type OrganizationPositionImpact struct {
	ChildPositions            int `json:"child_positions"`
	ResponsibilityAssignments int `json:"responsibility_assignments"`
	AuthorityGrants           int `json:"authority_grants"`
	ActiveRoleBindings        int `json:"active_role_bindings"`
}

type OrganizationPositionRevisionSummary struct {
	ID          string                        `json:"id"`
	PositionID  string                        `json:"position_id"`
	Operation   OrganizationPositionOperation `json:"operation"`
	BaseVersion int64                         `json:"base_version"`
	Base        OrganizationPositionState     `json:"base"`
	Proposed    OrganizationPositionState     `json:"proposed"`
	MakerID     string                        `json:"maker_id"`
	CheckerID   string                        `json:"checker_id,omitempty"`
	Status      string                        `json:"status"`
	Rationale   string                        `json:"rationale,omitempty"`
	Impact      OrganizationPositionImpact    `json:"impact"`
	CreatedAt   time.Time                     `json:"created_at"`
	DecidedAt   *time.Time                    `json:"decided_at,omitempty"`
	AppliedAt   *time.Time                    `json:"applied_at,omitempty"`
}

type ProposeOrganizationPositionInput struct {
	TenantID            string                        `json:"tenant_id"`
	LegalEntityID       string                        `json:"legal_entity_id"`
	PositionID          string                        `json:"position_id,omitempty"`
	Operation           OrganizationPositionOperation `json:"operation"`
	Code                string                        `json:"code,omitempty"`
	Title               string                        `json:"title,omitempty"`
	FunctionName        string                        `json:"function_name,omitempty"`
	OrganizationScopeID string                        `json:"organization_scope_id,omitempty"`
	ParentPositionID    string                        `json:"parent_position_id,omitempty"`
	OccupantPrincipalID string                        `json:"occupant_principal_id,omitempty"`
	ExpectedVersion     int64                         `json:"expected_version,omitempty"`
	ActorID             string                        `json:"-"`
}

type DecideOrganizationPositionInput struct {
	TenantID      string `json:"tenant_id"`
	LegalEntityID string `json:"legal_entity_id"`
	RevisionID    string `json:"revision_id"`
	ActorID       string `json:"-"`
	Rationale     string `json:"rationale"`
}

type OrganizationScopeOperation string

const (
	OrganizationScopeCreate OrganizationScopeOperation = "CREATE"
	OrganizationScopeUpdate OrganizationScopeOperation = "UPDATE"
	OrganizationScopeMove   OrganizationScopeOperation = "MOVE"
	OrganizationScopeRetire OrganizationScopeOperation = "RETIRE"
)

type OrganizationScopeImpact struct {
	ChildScopes    int `json:"child_scopes"`
	Positions      int `json:"positions"`
	AccessMappings int `json:"access_mappings"`
	OpenMatters    int `json:"open_matters"`
}

type OrganizationScopeRevisionSummary struct {
	ID                    string                     `json:"id"`
	ScopeID               string                     `json:"scope_id"`
	Operation             OrganizationScopeOperation `json:"operation"`
	BaseVersion           int64                      `json:"base_version"`
	ProposedParentScopeID string                     `json:"proposed_parent_scope_id,omitempty"`
	ProposedCode          string                     `json:"proposed_code,omitempty"`
	ProposedName          string                     `json:"proposed_name,omitempty"`
	ProposedKind          organization.ScopeKind     `json:"proposed_kind,omitempty"`
	MakerID               string                     `json:"maker_id"`
	CheckerID             string                     `json:"checker_id,omitempty"`
	Status                string                     `json:"status"`
	Rationale             string                     `json:"rationale,omitempty"`
	Impact                OrganizationScopeImpact    `json:"impact"`
	CreatedAt             time.Time                  `json:"created_at"`
	DecidedAt             *time.Time                 `json:"decided_at,omitempty"`
	AppliedAt             *time.Time                 `json:"applied_at,omitempty"`
}

type ProposeOrganizationScopeInput struct {
	TenantID        string                     `json:"tenant_id"`
	LegalEntityID   string                     `json:"legal_entity_id"`
	ScopeID         string                     `json:"scope_id,omitempty"`
	Operation       OrganizationScopeOperation `json:"operation"`
	ParentScopeID   string                     `json:"parent_scope_id,omitempty"`
	Code            string                     `json:"code,omitempty"`
	Name            string                     `json:"name,omitempty"`
	Kind            organization.ScopeKind     `json:"kind,omitempty"`
	ExpectedVersion int64                      `json:"expected_version,omitempty"`
	ActorID         string                     `json:"-"`
}

type DecideOrganizationScopeInput struct {
	TenantID      string `json:"tenant_id"`
	LegalEntityID string `json:"legal_entity_id"`
	RevisionID    string `json:"revision_id"`
	ActorID       string `json:"-"`
	Rationale     string `json:"rationale"`
}

type EscalationRuntimeStatus struct {
	PendingTimers  int `json:"pending_timers"`
	EscalatedTasks int `json:"escalated_tasks"`
	Unresolved24h  int `json:"unresolved_24h"`
	FailedTimers   int `json:"failed_timers"`
}

type AdminOverview struct {
	Sources                       []SCIMSourceSummary                   `json:"sources"`
	People                        []PersonSummary                       `json:"people"`
	Groups                        []GroupSummary                        `json:"groups"`
	Roles                         []RoleTemplateSummary                 `json:"roles"`
	LegalEntities                 []LegalEntitySummary                  `json:"legal_entities"`
	Bindings                      []GroupRoleBindingSummary             `json:"bindings"`
	Positions                     []PositionSummary                     `json:"positions"`
	OrganizationScopes            []organization.Scope                  `json:"organization_scopes"`
	OrganizationScopesTruncated   bool                                  `json:"organization_scopes_truncated"`
	OrganizationScopeRevisions    []OrganizationScopeRevisionSummary    `json:"organization_scope_revisions"`
	OrganizationPositionRevisions     []OrganizationPositionRevisionSummary     `json:"organization_position_revisions"`
	OrganizationPositionRoleRevisions []OrganizationPositionRoleRevisionSummary `json:"organization_position_role_revisions"`
	DataBoundary                  LegalEntityDataBoundary               `json:"data_boundary"`
	DataBoundaryRevisions         []LegalEntityDataBoundaryRevision     `json:"data_boundary_revisions"`
	Escalation                    EscalationRuntimeStatus               `json:"escalation"`
}

// OperationalStatus is the bounded exception projection used by actor-facing
// administration queues. It deliberately excludes people, group membership,
// role bindings and other configuration detail that is not needed to decide
// whether an administrator must act.
type OperationalStatus struct {
	SourceExceptions []SCIMSourceSummary     `json:"source_exceptions"`
	Escalation       EscalationRuntimeStatus `json:"escalation"`
}

type CreateSCIMSourceInput struct {
	TenantID         string `json:"tenant_id"`
	Code             string `json:"code"`
	IdentityIssuer   string `json:"identity_issuer,omitempty"`
	SubjectAttribute string `json:"subject_attribute"`
	ActorID          string `json:"-"`
}

type CreateGroupRoleBindingInput struct {
	TenantID       string   `json:"tenant_id"`
	GroupID        string   `json:"group_id"`
	RoleTemplateID string   `json:"role_template_id"`
	LegalEntityID  string   `json:"legal_entity_id"`
	DepartmentPath []string `json:"department_path"`
	ActorID        string   `json:"-"`
}

type Administrator interface {
	Overview(context.Context, string, string, int) (AdminOverview, error)
	CreateSCIMSource(context.Context, CreateSCIMSourceInput, []byte) (SCIMSourceSummary, error)
	RotateSCIMSourceToken(context.Context, string, string, string, []byte) error
	RevokeSCIMSource(context.Context, string, string, string) error
	CreateGroupRoleBinding(context.Context, CreateGroupRoleBindingInput) (GroupRoleBindingSummary, error)
	RetireGroupRoleBinding(context.Context, string, string, string) error
	ProposeOrganizationScope(context.Context, ProposeOrganizationScopeInput) (OrganizationScopeRevisionSummary, error)
	ApproveOrganizationScope(context.Context, DecideOrganizationScopeInput) error
	RejectOrganizationScope(context.Context, DecideOrganizationScopeInput) error
	ProposeOrganizationPosition(context.Context, ProposeOrganizationPositionInput) (OrganizationPositionRevisionSummary, error)
	ApproveOrganizationPosition(context.Context, DecideOrganizationPositionInput) error
	RejectOrganizationPosition(context.Context, DecideOrganizationPositionInput) error
	ProposeOrganizationPositionRole(context.Context, ProposeOrganizationPositionRoleInput) (OrganizationPositionRoleRevisionSummary, error)
	ApproveOrganizationPositionRole(context.Context, DecideOrganizationPositionRoleInput) error
	RejectOrganizationPositionRole(context.Context, DecideOrganizationPositionRoleInput) error
}

func NewProvisioningToken() (string, [32]byte, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", [32]byte{}, err
	}
	token := "cs_scim_" + base64.RawURLEncoding.EncodeToString(raw[:])
	return token, sha256.Sum256([]byte(token)), nil
}

func normalizeSCIMSourceInput(input CreateSCIMSourceInput) (CreateSCIMSourceInput, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.IdentityIssuer = strings.TrimSpace(input.IdentityIssuer)
	input.SubjectAttribute = strings.TrimSpace(input.SubjectAttribute)
	input.ActorID = strings.TrimSpace(input.ActorID)
	if input.SubjectAttribute == "" {
		input.SubjectAttribute = "externalId"
	}
	if input.TenantID == "" || input.ActorID == "" || input.Code == "" || len(input.Code) > 80 {
		return CreateSCIMSourceInput{}, ErrAdminInvalid
	}
	for _, r := range input.Code {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return CreateSCIMSourceInput{}, ErrAdminInvalid
		}
	}
	if input.SubjectAttribute != "externalId" && input.SubjectAttribute != "userName" {
		return CreateSCIMSourceInput{}, ErrAdminInvalid
	}
	if input.IdentityIssuer != "" {
		issuer, err := url.Parse(input.IdentityIssuer)
		if err != nil || issuer.Scheme != "https" || issuer.Host == "" || issuer.User != nil || issuer.Fragment != "" || len(input.IdentityIssuer) > 2048 {
			return CreateSCIMSourceInput{}, ErrAdminInvalid
		}
	}
	return input, nil
}
