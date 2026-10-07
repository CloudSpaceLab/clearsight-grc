package oversight

import "time"

const GroupProjectionVersion = "group-oversight-v2"

type GroupChildState string

const (
	GroupChildAvailable GroupChildState = "AVAILABLE"
	GroupChildStale     GroupChildState = "STALE"
	GroupChildMissing   GroupChildState = "MISSING"
)

type GroupDomainPosture struct {
	RisksOutsideAppetite int `json:"risks_outside_appetite"`
	IndicatorBreaches    int `json:"indicator_breaches"`
	AssuranceFailures    int `json:"assurance_failures"`
}

type GroupChildFact struct {
	LegalEntityID          string
	LegalEntityCode        string
	LegalEntityName        string
	Jurisdiction           string
	State                  GroupChildState
	ChildSnapshotID        string
	ChildGeneratedAt       *time.Time
	ChildProjectionVersion string
	Coverage               Coverage
	Counts                 Counts
	SourceHighWater        map[string]time.Time
	DomainState            GroupChildState
	DomainSourceID         string
	DomainGeneratedAt      *time.Time
	DomainDefinitionRevision string
	DomainPosture          GroupDomainPosture
	DomainSourceHighWater  map[string]time.Time
}

type GroupProjection struct {
	ID                 string
	TenantID           string
	GeneratedAt        time.Time
	RefreshSlot        time.Time
	ProjectionVersion  string
	ActiveChildCount   int
	CapturedChildCount int
	MissingChildCount  int
	StaleChildCount    int
	Children           []GroupChildFact
}

type GroupCoverage struct {
	AuthorizedChildren int  `json:"authorized_children"`
	IncludedChildren   int  `json:"included_children"`
	MissingChildren    int  `json:"missing_children"`
	StaleChildren      int  `json:"stale_children"`
	Complete           bool `json:"complete"`
}

type GroupChildSummary struct {
	LegalEntityID          string               `json:"legal_entity_id"`
	LegalEntityCode        string               `json:"legal_entity_code"`
	LegalEntityName        string               `json:"legal_entity_name"`
	Jurisdiction           string               `json:"jurisdiction,omitempty"`
	State                  GroupChildState      `json:"state"`
	ChildSnapshotID        string               `json:"child_snapshot_id,omitempty"`
	ChildGeneratedAt       *time.Time           `json:"child_generated_at,omitempty"`
	ChildProjectionVersion string               `json:"child_projection_version,omitempty"`
	Coverage               Coverage             `json:"coverage"`
	Counts                 Counts               `json:"counts"`
	SourceHighWater        map[string]time.Time `json:"source_high_water,omitempty"`
	DomainState            GroupChildState      `json:"domain_state"`
	DomainSourceID         string               `json:"domain_source_id,omitempty"`
	DomainGeneratedAt      *time.Time           `json:"domain_generated_at,omitempty"`
	DomainDefinitionRevision string             `json:"domain_definition_revision,omitempty"`
	DomainPosture          GroupDomainPosture   `json:"domain_posture"`
	DomainSourceHighWater  map[string]time.Time `json:"domain_source_high_water,omitempty"`
}

type GroupSnapshot struct {
	RevisionID        string              `json:"revision_id"`
	GeneratedAt       time.Time           `json:"generated_at"`
	ProjectionVersion string              `json:"projection_version"`
	Freshness         Freshness           `json:"freshness"`
	PostureFreshness  Freshness           `json:"posture_freshness"`
	Coverage          GroupCoverage       `json:"coverage"`
	RecordCoverage    Coverage            `json:"record_coverage"`
	PostureCoverage   GroupCoverage       `json:"posture_coverage"`
	Posture           GroupDomainPosture  `json:"posture"`
	Counts            Counts              `json:"counts"`
	Children          []GroupChildSummary `json:"children"`
}
