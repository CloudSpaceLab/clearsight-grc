package continuity

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

// SummaryQuery describes a bounded, tenant-scoped operational list read.
// Cursor values are opaque to clients and encode the last item from the
// previous page.
type SummaryQuery struct {
	Search       string
	Status       string
	ProgramID    string
	OverallState string
	Jurisdiction string
	MatterType   string
	DueCondition string
	Cursor       string
	Priority     int
	AssignedToMe bool
	asOf         time.Time
	principalID  string
	Limit        int
}

type ProgramSummary struct {
	Program                Program       `json:"program"`
	StateLabel             string        `json:"state_label"`
	OverallState           ProgramState  `json:"overall_state"`
	Reasons                []StateReason `json:"reasons"`
	ReasonsTotal           int           `json:"reasons_total"`
	ReasonsOmitted         int           `json:"reasons_omitted"`
	OpenMatterCount        int           `json:"open_matter_count"`
	RequirementCount       int           `json:"requirement_count"`
	SafeguardCount         int           `json:"safeguard_count"`
	EvidenceCheckCount     int           `json:"evidence_check_count"`
	ProgramVersion         int64         `json:"program_version"`
	AssessedProgramVersion int64         `json:"assessed_program_version"`
	ProjectionVersion      int64         `json:"projection_version"`
	ProjectionStale        bool          `json:"projection_stale"`
	StateGeneratedAt       *time.Time    `json:"state_generated_at,omitempty"`
}

type MatterSummary struct {
	Matter            Matter                   `json:"matter"`
	TypeLabel         string                   `json:"type_label"`
	StatusLabel       string                   `json:"status_label"`
	NextAction        string                   `json:"next_action"`
	ProgramCount      int                      `json:"program_count"`
	OpenActionCount   int                      `json:"open_action_count"`
	OutcomeCheckCount int                      `json:"outcome_check_count"`
	LatestOutcome     VerificationResultStatus `json:"latest_outcome,omitempty"`
	LatestOutcomeAt   *time.Time               `json:"latest_outcome_at,omitempty"`
}

type ProgramSummaryPage struct {
	Items       []ProgramSummary `json:"items"`
	NextCursor  string           `json:"next_cursor,omitempty"`
	GeneratedAt time.Time        `json:"generated_at"`
}

type MatterSummaryPage struct {
	Items       []MatterSummary `json:"items"`
	NextCursor  string          `json:"next_cursor,omitempty"`
	GeneratedAt time.Time       `json:"generated_at"`
}

// SummaryRepository is intentionally separate from Repository so custom
// repositories built against the command contract remain source-compatible.
type SummaryRepository interface {
	ListProgramSummaries(context.Context, string, SummaryQuery) (ProgramSummaryPage, error)
	ListMatterSummaries(context.Context, string, SummaryQuery) (MatterSummaryPage, error)
}

func (s *Service) ListProgramSummaries(ctx context.Context, tenant string, query SummaryQuery) (ProgramSummaryPage, error) {
	if strings.TrimSpace(tenant) == "" {
		return ProgramSummaryPage{}, fmt.Errorf("tenant_id is required")
	}
	query.Search = strings.TrimSpace(query.Search)
	query.Status = strings.ToUpper(strings.TrimSpace(query.Status))
	query.ProgramID = strings.TrimSpace(query.ProgramID)
	query.OverallState = strings.ToUpper(strings.TrimSpace(query.OverallState))
	query.Jurisdiction = strings.TrimSpace(query.Jurisdiction)
	query.asOf = s.now().UTC()
	query.Limit = boundedLimit(query.Limit)
	if err := validateProgramSummaryQuery(ctx, tenant, &query); err != nil {
		return ProgramSummaryPage{}, err
	}
	if summaries, ok := s.repo.(SummaryRepository); ok {
		return summaries.ListProgramSummaries(ctx, tenant, query)
	}
	values, err := s.repo.ListPrograms(ctx, tenant, query.Limit)
	if err != nil {
		return ProgramSummaryPage{}, err
	}
	items := make([]ProgramSummary, 0, len(values))
	for _, value := range values {
		items = append(items, summarizeProgram(value))
	}
	return ProgramSummaryPage{Items: items, GeneratedAt: s.now().UTC()}, nil
}

func (s *Service) ListMatterSummaries(ctx context.Context, tenant string, query SummaryQuery) (MatterSummaryPage, error) {
	if strings.TrimSpace(tenant) == "" {
		return MatterSummaryPage{}, fmt.Errorf("tenant_id is required")
	}
	query.Search = strings.TrimSpace(query.Search)
	query.Status = strings.ToUpper(strings.TrimSpace(query.Status))
	query.ProgramID = strings.TrimSpace(query.ProgramID)
	query.MatterType = strings.ToUpper(strings.TrimSpace(query.MatterType))
	query.DueCondition = strings.ToUpper(strings.TrimSpace(query.DueCondition))
	query.asOf = s.now().UTC()
	query.Limit = boundedLimit(query.Limit)
	if err := validateMatterSummaryQuery(ctx, tenant, &query); err != nil {
		return MatterSummaryPage{}, err
	}
	if summaries, ok := s.repo.(SummaryRepository); ok {
		return summaries.ListMatterSummaries(ctx, tenant, query)
	}
	values, err := s.repo.ListMatters(ctx, tenant, query.Status, query.Limit)
	if err != nil {
		return MatterSummaryPage{}, err
	}
	items := make([]MatterSummary, 0, len(values))
	for _, value := range values {
		if query.ProgramID != "" {
			linked := false
			for _, link := range value.Links {
				if link.ProgramID == query.ProgramID {
					linked = true
					break
				}
			}
			if !linked {
				continue
			}
		}
		items = append(items, summarizeMatter(value))
	}
	return MatterSummaryPage{Items: items, GeneratedAt: s.now().UTC()}, nil
}

func validateProgramSummaryQuery(ctx context.Context, tenant string, query *SummaryQuery) error {
	if query.Status != "" && !oneOf(query.Status, string(ProgramDraft), string(ProgramActive), string(ProgramPaused), string(ProgramRetired)) {
		return fmt.Errorf("invalid filter: Program status is not supported")
	}
	if query.OverallState != "" && !oneOf(query.OverallState, string(StateCurrent), string(StateAtRisk), string(StateGapIdentified), string(StateEvidenceInsufficient), string(StateImplementationPending), string(StateOverdue), string(StateUnderReview), string(StateNotApplicable), string(StateUnknown)) {
		return fmt.Errorf("invalid filter: overall state is not supported")
	}
	return bindSummaryPrincipal(ctx, tenant, query)
}

func validateMatterSummaryQuery(ctx context.Context, tenant string, query *SummaryQuery) error {
	if query.Status != "" && !oneOf(query.Status, "OPEN", string(MatterDraft), string(MatterInitialReview), string(MatterAssessment), string(MatterDecisionRequired), string(MatterActionsInProgress), string(MatterResponsePreparation), string(MatterVerification), string(MatterClosed), string(MatterCancelled)) {
		return fmt.Errorf("invalid filter: issue status is not supported")
	}
	if query.MatterType != "" && !oneOf(query.MatterType, string(MatterRegulatoryChange), string(MatterSupervisoryFinding), string(MatterAuthorityRequest), string(MatterRiskSituation), string(MatterControlGap), string(MatterAuditFinding), string(MatterException), string(MatterIncident), string(MatterOperationalLoss), string(MatterDataBreach), string(MatterVendorReview), string(MatterVendorDeficiency), string(MatterCustomerConcern), string(MatterOverdueObligation), string(MatterFailedVerification), string(MatterEvidenceContradiction), string(MatterKRIBreach)) {
		return fmt.Errorf("invalid filter: issue type is not supported")
	}
	if query.Priority < 0 || query.Priority > 5 {
		return fmt.Errorf("invalid filter: priority must be between 1 and 5")
	}
	if query.DueCondition != "" && !oneOf(query.DueCondition, "OVERDUE", "DUE_7_DAYS", "DUE_30_DAYS", "NO_DUE_DATE") {
		return fmt.Errorf("invalid filter: due condition is not supported")
	}
	return bindSummaryPrincipal(ctx, tenant, query)
}

func bindSummaryPrincipal(ctx context.Context, tenant string, query *SummaryQuery) error {
	if !query.AssignedToMe {
		return nil
	}
	actor, ok := identity.FromContext(ctx)
	if !ok || strings.TrimSpace(actor.PrincipalID) == "" || actor.TenantID != tenant {
		return fmt.Errorf("invalid filter: assigned-to-me requires a verified identity in this tenant")
	}
	query.principalID = actor.PrincipalID
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func summarizeProgram(aggregate ProgramAggregate) ProgramSummary {
	aggregate = decorateProgram(aggregate)
	overall := StateUnknown
	var generatedAt *time.Time
	reasons := []StateReason{}
	reasonsTotal := 0
	openMatters := 0
	assessedVersion := int64(0)
	if aggregate.CurrentState != nil {
		overall = aggregate.CurrentState.Overall
		value := aggregate.CurrentState.GeneratedAt
		generatedAt = &value
		openMatters = aggregate.CurrentState.OpenMatterCount
		assessedVersion = aggregate.CurrentState.ProgramVersion
		reasons = append(reasons, aggregate.CurrentState.Reasons...)
		reasonsTotal = len(reasons)
	}
	if len(reasons) > 6 {
		reasons = reasons[:6]
	}
	return ProgramSummary{
		Program:                aggregate.Program,
		StateLabel:             aggregate.StateLabel,
		OverallState:           overall,
		Reasons:                reasons,
		ReasonsTotal:           reasonsTotal,
		ReasonsOmitted:         max(0, reasonsTotal-len(reasons)),
		OpenMatterCount:        openMatters,
		RequirementCount:       len(aggregate.Requirements),
		SafeguardCount:         len(aggregate.ControlImplementations),
		EvidenceCheckCount:     len(aggregate.EvidenceContracts),
		ProgramVersion:         aggregate.Program.Version,
		AssessedProgramVersion: assessedVersion,
		ProjectionStale:        assessedVersion < aggregate.Program.Version,
		StateGeneratedAt:       generatedAt,
	}
}

func summarizeMatter(aggregate MatterAggregate) MatterSummary {
	aggregate = decorateMatter(aggregate)
	programs := map[string]struct{}{}
	for _, link := range aggregate.Links {
		if link.ProgramID != "" {
			programs[link.ProgramID] = struct{}{}
		}
	}
	openActions := 0
	for _, action := range aggregate.Actions {
		if action.Status != ActionImplemented && action.Status != ActionCancelled {
			openActions++
		}
	}
	activeChecks := 0
	for _, contract := range aggregate.VerificationContracts {
		if contract.Status == VerificationActive {
			activeChecks++
		}
	}
	var latest *VerificationResult
	for i := range aggregate.VerificationResults {
		value := aggregate.VerificationResults[i]
		if latest == nil || value.ObservedAt.After(latest.ObservedAt) {
			copy := value
			latest = &copy
		}
	}
	result := MatterSummary{
		Matter:            aggregate.Matter,
		TypeLabel:         aggregate.TypeLabel,
		StatusLabel:       aggregate.StatusLabel,
		NextAction:        aggregate.NextAction,
		ProgramCount:      len(programs),
		OpenActionCount:   openActions,
		OutcomeCheckCount: activeChecks,
	}
	if latest != nil {
		result.LatestOutcome = latest.Result
		value := latest.ObservedAt
		result.LatestOutcomeAt = &value
	}
	return result
}

type programSummaryCursor struct {
	Rank      int       `json:"rank"`
	UpdatedAt time.Time `json:"updated_at"`
	ID        string    `json:"id"`
}

type matterSummaryCursor struct {
	Priority  int       `json:"priority"`
	UpdatedAt time.Time `json:"updated_at"`
	ID        string    `json:"id"`
}

func encodeSummaryCursor(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeProgramSummaryCursor(value string) (programSummaryCursor, error) {
	if strings.TrimSpace(value) == "" {
		return programSummaryCursor{}, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return programSummaryCursor{}, fmt.Errorf("invalid cursor")
	}
	var cursor programSummaryCursor
	if err := json.Unmarshal(payload, &cursor); err != nil || cursor.ID == "" || cursor.UpdatedAt.IsZero() {
		return programSummaryCursor{}, fmt.Errorf("invalid cursor")
	}
	return cursor, nil
}

func decodeMatterSummaryCursor(value string) (matterSummaryCursor, error) {
	if strings.TrimSpace(value) == "" {
		return matterSummaryCursor{}, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return matterSummaryCursor{}, fmt.Errorf("invalid cursor")
	}
	var cursor matterSummaryCursor
	if err := json.Unmarshal(payload, &cursor); err != nil || cursor.ID == "" || cursor.UpdatedAt.IsZero() {
		return matterSummaryCursor{}, fmt.Errorf("invalid cursor")
	}
	return cursor, nil
}

func programStatusRank(status ProgramStatus) int {
	switch status {
	case ProgramActive:
		return 0
	case ProgramPaused:
		return 1
	case ProgramDraft:
		return 2
	default:
		return 3
	}
}
