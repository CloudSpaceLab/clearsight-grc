package metricview

import (
	"context"
	"errors"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
)

var (
	ErrGroupPostureInvalid = errors.New("group posture request is invalid")
	ErrGroupPostureMissing = errors.New("group posture is unavailable")
)

type GroupPostureEntity struct {
	LegalEntityID   string `json:"legal_entity_id"`
	LegalEntityCode string `json:"legal_entity_code"`
	LegalEntityName string `json:"legal_entity_name"`
	Jurisdiction    string `json:"jurisdiction,omitempty"`
}

type GroupPostureCounts struct {
	RisksOutsideAppetite int `json:"risks_outside_appetite"`
	IndicatorBreaches    int `json:"indicator_breaches"`
	AssuranceFailures    int `json:"assurance_failures"`
	LossEvents           int `json:"loss_events"`
}

type GroupPostureChild struct {
	GroupPostureEntity
	RiskState          string               `json:"risk_state"`
	Completeness       Completeness         `json:"completeness"`
	SourceID           string               `json:"source_id,omitempty"`
	SourceGeneratedAt  *time.Time           `json:"source_generated_at,omitempty"`
	SourceRevision     string               `json:"source_revision,omitempty"`
	DefinitionRevision string               `json:"definition_revision,omitempty"`
	Counts              GroupPostureCounts   `json:"counts"`
	Unknown             int                  `json:"unknown"`
	Excluded            int                  `json:"excluded"`
	Freshness           oversight.Freshness  `json:"freshness"`
}

type GroupPostureCoverage struct {
	AuthorizedChildren int  `json:"authorized_children"`
	IncludedChildren   int  `json:"included_children"`
	MissingChildren    int  `json:"missing_children"`
	StaleChildren      int  `json:"stale_children"`
	PartialChildren    int  `json:"partial_children"`
	Complete           bool `json:"complete"`
}

type GroupPostureBundle struct {
	GeneratedAt        time.Time            `json:"generated_at"`
	PeriodStart        time.Time            `json:"period_start"`
	PeriodEnd          time.Time            `json:"period_end"`
	DefinitionRevision string               `json:"definition_revision"`
	RiskCoverage       GroupPostureCoverage `json:"risk_coverage"`
	LossCoverage       GroupPostureCoverage `json:"loss_coverage"`
	Counts             GroupPostureCounts   `json:"counts"`
	Children           []GroupPostureChild  `json:"children"`
}

type GroupPostureReader interface {
	ActiveGroupEntities(context.Context, string, time.Time) ([]GroupPostureEntity, error)
	GroupPosture(
		context.Context,
		string,
		[]string,
		time.Time,
		time.Time,
		time.Time,
	) (GroupPostureBundle, error)
}
