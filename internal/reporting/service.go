package reporting

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

const (
	DecisionPropose  = "PROPOSE"
	DecisionSubmit   = "SUBMIT"
	DecisionReview   = "REVIEW"
	DecisionActivate = "ACTIVATE"
	DecisionReject   = "REJECT"
	DecisionRetire   = "RETIRE"

	FailureRowLimitExceeded       = "row_limit_exceeded"
	FailureByteLimitExceeded      = "byte_limit_exceeded"
	FailureRetryBudgetExhausted   = "retry_budget_exhausted"
	FailureSourceBoundaryMismatch = "source_boundary_mismatch"

	reportClaimBatch = 100
	maxDecisionNote  = 1000
)

var reportCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{2,47}$`)

// AuthorityChecker is the existing authority service's read contract. A
// production service must be supplied: nil, an unavailable route, or an actor
// outside the current effective candidate set fails closed.
type AuthorityChecker interface {
	Resolve(context.Context, authority.ResolveInput) (authority.Resolution, error)
}

// Service owns every report command and asynchronous run operation. Human
// command actors always come from identity.WithActor; actor fields in input
// structs are ignored.
type Service struct {
	repo      Repository
	objects   evidence.ObjectStore
	authority AuthorityChecker
	Now       func() time.Time
	WorkerID  string
}

func NewService(repository Repository, objects evidence.ObjectStore, authorityChecker AuthorityChecker) *Service {
	return &Service{repo: repository, objects: objects, authority: authorityChecker, Now: time.Now}
}

type ProposeInput struct {
	Code           string                  `json:"code"`
	Name           string                  `json:"name"`
	Description    string                  `json:"description"`
	Dataset        ReportDataset           `json:"dataset"`
	ScopeKind      ReportScopeKind         `json:"scope_kind"`
	ScopeRef       string                  `json:"scope_ref,omitempty"`
	Format         ReportFormat            `json:"format"`
	Filter         *ReportFilterExpression `json:"filter,omitempty"`
	EffectiveFrom  *time.Time              `json:"effective_from,omitempty"`
	EffectiveUntil *time.Time              `json:"effective_until,omitempty"`
	// MakerID is accepted only to make the request contract explicit. Propose
	// always overwrites it from the verified context.
	MakerID string `json:"maker_id,omitempty"`
}

type DefinitionTransitionInput struct {
	Scope           ReportScope `json:"-"`
	DefinitionID    string      `json:"definition_id"`
	ExpectedVersion int64       `json:"expected_version"`
	ChecksumSeen    string      `json:"checksum_seen"`
	Note            string      `json:"note,omitempty"`
	EffectiveFrom   *time.Time  `json:"effective_from,omitempty"`
}

type CreateRunInput struct {
	Scope                     ReportScope `json:"-"`
	DefinitionID              string      `json:"definition_id"`
	ExpectedDefinitionVersion int         `json:"expected_definition_version"`
	// RequestedByRef is accepted for wire compatibility but always overwritten
	// from the verified actor.
	RequestedByRef string `json:"requested_by_ref,omitempty"`
}

// ReportRow is one bounded dataset row. Values are rendered only after the
// repository has applied tenant, legal-entity, visibility, dataset, and filter
// scope inside the page query.
type ReportRow struct {
	ID     string         `json:"id"`
	Values map[string]any `json:"values"`
}

type ReportPage struct {
	Rows       []ReportRow
	NextCursor string
	// Columns is the stable ordered CSV schema. It is optional for row maps so
	// a test or source adapter can provide a bounded first page; once rows are
	// present the service derives and freezes a deterministic schema.
	Columns []string
}

// ValidateTransitionForWrite is the sole definition state machine. Both the
// service and every repository TransitionDefinition implementation must call it
// after locking and checking the expected command version.
func ValidateTransitionForWrite(current ReportDefinition, next DefinitionStatus) error {
	if !validDefinitionStatus(current.Status) || !validDefinitionStatus(next) || current.Status == next {
		return ErrInvalid
	}
	switch current.Status {
	case DefinitionDraft:
		if next != DefinitionPendingReview && next != DefinitionRetired {
			return ErrInvalid
		}
	case DefinitionPendingReview:
		if next != DefinitionReviewed && next != DefinitionRetired {
			return ErrInvalid
		}
	case DefinitionReviewed:
		if next != DefinitionActive && next != DefinitionRetired {
			return ErrInvalid
		}
	case DefinitionActive:
		if next != DefinitionRetired {
			return ErrInvalid
		}
	case DefinitionRetired:
		return ErrInvalid
	}
	return nil
}

func (s *Service) Propose(ctx context.Context, input ProposeInput) (ReportDefinition, error) {
	actor, scope, err := s.actorScope(ctx)
	if err != nil {
		return ReportDefinition{}, err
	}
	if s.repo == nil {
		return ReportDefinition{}, ErrInvalid
	}
	definitionID, err := id.NewUUIDv7()
	if err != nil {
		return ReportDefinition{}, err
	}
	if err := s.authorized(ctx, actor, scope, "REPORT_DEFINITION", definitionID, authority.ResponsibilityProposer, "report.definition.propose", 4); err != nil {
		return ReportDefinition{}, err
	}

	now := s.now()
	filter, err := normalizedStoredFilter(input.Filter)
	if err != nil {
		return ReportDefinition{}, err
	}
	definition := ReportDefinition{
		ID: definitionID, TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
		Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description), Dataset: input.Dataset,
		ScopeKind: input.ScopeKind, ScopeRef: strings.TrimSpace(input.ScopeRef),
		Format: input.Format, Filter: filter, Status: DefinitionDraft, CurrentVersion: 1,
		MakerID: actor.PrincipalID, EffectiveFrom: normalizedTimeValue(input.EffectiveFrom),
		EffectiveUntil: normalizedTimeValue(input.EffectiveUntil), CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	if err := validateDefinitionForCreate(definition); err != nil {
		return ReportDefinition{}, err
	}
	definition.StoredChecksum = definition.Checksum()

	existing, lookupErr := s.repo.GetDefinitionByCode(ctx, scope, definition.Code)
	switch {
	case lookupErr == nil && existing.Status != DefinitionRetired:
		return ReportDefinition{}, ErrConflict
	case lookupErr != nil && !errors.Is(lookupErr, ErrNotFound):
		return ReportDefinition{}, lookupErr
	}
	revision := ReportDefinitionRevision{
		DefinitionID: definition.ID, TenantID: definition.TenantID, LegalEntityID: definition.LegalEntityID,
		Version: 1, BaseVersion: 0, Dataset: definition.Dataset, ScopeKind: definition.ScopeKind,
		ScopeRef: definition.ScopeRef, Format: definition.Format, Filter: cloneReportFilter(definition.Filter),
		Checksum: definition.StoredChecksum, MakerID: actor.PrincipalID, CreatedAt: now,
		Decision: "PROPOSED",
	}
	created, err := s.repo.CreateDefinition(ctx, scope, definition, revision)
	if err != nil {
		return ReportDefinition{}, err
	}
	created.Effective = definitionIsEffective(created, now)
	return created, nil
}

func (s *Service) Submit(ctx context.Context, input DefinitionTransitionInput) (ReportDefinition, error) {
	return s.transitionDefinition(ctx, input, DefinitionPendingReview, authority.ResponsibilityProposer, DecisionSubmit, "report.definition.submit", 4)
}

func (s *Service) Review(ctx context.Context, input DefinitionTransitionInput) (ReportDefinition, error) {
	return s.transitionDefinition(ctx, input, DefinitionReviewed, authority.ResponsibilityReviewer, DecisionReview, "report.definition.review", 4)
}

func (s *Service) Activate(ctx context.Context, input DefinitionTransitionInput) (ReportDefinition, error) {
	return s.transitionDefinition(ctx, input, DefinitionActive, authority.ResponsibilityAuthorizer, DecisionActivate, "report.definition.activate", 5)
}

func (s *Service) Reject(ctx context.Context, input DefinitionTransitionInput) (ReportDefinition, error) {
	return s.transitionDefinition(ctx, input, DefinitionRetired, authority.ResponsibilityReviewer, DecisionReject, "report.definition.reject", 4)
}

func (s *Service) Retire(ctx context.Context, input DefinitionTransitionInput) (ReportDefinition, error) {
	return s.transitionDefinition(ctx, input, DefinitionRetired, authority.ResponsibilityAuthorizer, DecisionRetire, "report.definition.retire", 5)
}

func (s *Service) transitionDefinition(ctx context.Context, input DefinitionTransitionInput, next DefinitionStatus, responsibility authority.Responsibility, action, decisionType string, materiality int) (ReportDefinition, error) {
	actor, scope, err := s.scopedActor(ctx, input.Scope)
	if err != nil {
		return ReportDefinition{}, err
	}
	definitionID := strings.TrimSpace(input.DefinitionID)
	if s.repo == nil || definitionID == "" || input.ExpectedVersion <= 0 || utf8.RuneCountInString(input.Note) > maxDecisionNote {
		return ReportDefinition{}, ErrInvalid
	}
	if err := s.authorized(ctx, actor, scope, "REPORT_DEFINITION", definitionID, responsibility, decisionType, materiality); err != nil {
		return ReportDefinition{}, err
	}
	current, err := s.repo.GetDefinition(ctx, scope, definitionID)
	if err != nil {
		return ReportDefinition{}, err
	}
	if current.TenantID != scope.TenantID || current.LegalEntityID != scope.LegalEntityID {
		return ReportDefinition{}, ErrNotFound
	}
	if current.Version != input.ExpectedVersion {
		return ReportDefinition{}, ErrConflict
	}
	if input.ChecksumSeen == "" || input.ChecksumSeen != current.StoredChecksum || input.ChecksumSeen != current.Checksum() {
		return ReportDefinition{}, ErrConflict
	}

	if action == DecisionActivate && strings.TrimSpace(current.ReviewerID) == "" {
		return ReportDefinition{}, ErrClosureBlocked
	}
	if action == DecisionSubmit && actor.PrincipalID != current.MakerID {
		return ReportDefinition{}, ErrClosureBlocked
	}
	if (action == DecisionReview || action == DecisionReject) && actor.PrincipalID == current.MakerID {
		return ReportDefinition{}, ErrClosureBlocked
	}
	if (action == DecisionActivate || (action == DecisionRetire && current.ReviewerID != "")) &&
		(actor.PrincipalID == current.MakerID || actor.PrincipalID == current.ReviewerID) {
		return ReportDefinition{}, ErrClosureBlocked
	}
	if err := ValidateTransitionForWrite(current, next); err != nil {
		return ReportDefinition{}, err
	}

	now := s.now()
	decision := DecisionRecord{
		ActorID: actor.PrincipalID, Action: action, Note: strings.TrimSpace(input.Note),
		ChecksumSeen: input.ChecksumSeen, Timestamp: now,
	}
	if action == DecisionActivate {
		effectiveFrom := now
		if input.EffectiveFrom != nil {
			effectiveFrom = input.EffectiveFrom.UTC()
		}
		decision.EffectiveFrom = &effectiveFrom
	}
	transitioned, err := s.repo.TransitionDefinition(ctx, scope, definitionID, input.ExpectedVersion, next, decision)
	if err != nil {
		return ReportDefinition{}, err
	}
	transitioned.Effective = definitionIsEffective(transitioned, now)
	return transitioned, nil
}

func (s *Service) CreateRun(ctx context.Context, input CreateRunInput) (ReportRun, error) {
	actor, scope, err := s.scopedActor(ctx, input.Scope)
	if err != nil {
		return ReportRun{}, err
	}
	definitionID := strings.TrimSpace(input.DefinitionID)
	if s.repo == nil || s.objects == nil || definitionID == "" || input.ExpectedDefinitionVersion <= 0 {
		return ReportRun{}, ErrInvalid
	}
	runID, err := id.NewUUIDv7()
	if err != nil {
		return ReportRun{}, err
	}
	if err := s.authorized(ctx, actor, scope, "REPORT_RUN", runID, authority.ResponsibilityPerformer, "report.run.create", 3); err != nil {
		return ReportRun{}, err
	}
	definition, err := s.repo.GetDefinition(ctx, scope, definitionID)
	if err != nil {
		return ReportRun{}, err
	}
	if definition.TenantID != scope.TenantID || definition.LegalEntityID != scope.LegalEntityID {
		return ReportRun{}, ErrNotFound
	}
	if input.ExpectedDefinitionVersion != definition.CurrentVersion || definition.StoredChecksum == "" || definition.StoredChecksum != definition.Checksum() {
		return ReportRun{}, ErrConflict
	}
	now := s.now()
	if definition.Status != DefinitionActive || strings.TrimSpace(definition.ReviewerID) == "" || strings.TrimSpace(definition.CheckerID) == "" || definition.ApprovedAt == nil || !definitionIsEffective(definition, now) {
		return ReportRun{}, ErrClosureBlocked
	}
	filter, err := normalizedStoredFilter(definition.Filter)
	if err != nil {
		return ReportRun{}, err
	}
	boundary, err := s.repo.CaptureSourceBoundary(ctx, scope, definition)
	if err != nil {
		return ReportRun{}, err
	}
	boundary.CapturedAt = now
	if err := validateSourceBoundary(boundary); err != nil {
		return ReportRun{}, err
	}
	run := ReportRun{
		ID: runID, TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
		DefinitionID: definition.ID, DefinitionVersion: definition.CurrentVersion,
		DefinitionCode: definition.Code, DefinitionChecksum: definition.StoredChecksum,
		ScopeKind: definition.ScopeKind, ScopeRef: definition.ScopeRef,
		RequestedByRef: actor.PrincipalID, AsOf: now, SourceBoundary: boundary,
		Filter: filter, Dataset: definition.Dataset, Format: definition.Format, Status: RunQueued,
		CreatedAt: now, ExpiresAt: now.Add(ReportRunRetention),
	}
	return s.repo.CreateRun(ctx, scope, run)
}

func (s *Service) ExecuteRun(ctx context.Context, requested ReportRun) (ReportRun, error) {
	if s.repo == nil || s.objects == nil || strings.TrimSpace(s.WorkerID) == "" {
		return ReportRun{}, ErrInvalid
	}
	scope := ReportScope{TenantID: requested.TenantID, LegalEntityID: requested.LegalEntityID}
	if err := validateReportRunIdentity(requested); err != nil {
		return ReportRun{}, err
	}
	claimedRuns, err := s.repo.ClaimQueuedRuns(ctx, scope, strings.TrimSpace(s.WorkerID), reportClaimBatch)
	if err != nil {
		return ReportRun{}, err
	}
	var run ReportRun
	found := false
	for _, candidate := range claimedRuns {
		if candidate.ID == requested.ID {
			run = candidate
			found = true
			break
		}
	}
	if !found {
		return ReportRun{}, ErrConflict
	}
	if err := validateClaimedRun(run); err != nil {
		return ReportRun{}, err
	}
	if run.AttemptCount > MaxReportRunTries {
		return run, s.failRun(ctx, run, FailureRetryBudgetExhausted, ErrConflict)
	}
	if err := validateSourceBoundary(run.SourceBoundary); err != nil {
		return run, err
	}

	data, rowCount, err := s.renderRun(ctx, run)
	if err != nil {
		var limitError *reportLimitError
		if errors.As(err, &limitError) {
			return run, s.failRun(ctx, run, limitError.Code, err)
		}
		var boundaryError *sourceBoundaryError
		if errors.As(err, &boundaryError) {
			return run, s.failRun(ctx, run, FailureSourceBoundaryMismatch, err)
		}
		return run, err
	}

	dataDigest := sha256.Sum256(data)
	dataChecksum := hex.EncodeToString(dataDigest[:])
	dataKey, manifestKey := reportObjectKeys(run)
	dataInfo, err := s.objects.Put(ctx, dataKey, bytes.NewReader(data), MaxReportRunBytes)
	if err != nil {
		_ = s.objects.Delete(ctx, dataKey)
		return run, err
	}
	if dataInfo.Key != dataKey || dataInfo.SizeBytes != int64(len(data)) || dataInfo.SHA256 != dataChecksum {
		_ = s.objects.Delete(ctx, dataKey)
		return run, ErrArtifactIntegrity
	}

	manifest := Manifest{
		Schema: reportManifestSchema, GeneratedAt: s.now(), AsOf: run.AsOf, Source: run.SourceBoundary,
		DefinitionCode: run.DefinitionCode, DefinitionVersion: run.DefinitionVersion,
		DefinitionChecksum: run.DefinitionChecksum, Dataset: run.Dataset,
		ScopeKind: run.ScopeKind, ScopeRef: run.ScopeRef, RowCount: rowCount,
		PopulationComplete: run.SourceBoundary.PopulationComplete, Filter: cloneReportFilter(run.Filter),
		Coverage:   ManifestCoverage{Population: run.SourceBoundary.Population},
		DataSHA256: dataChecksum, RetentionUntil: run.ExpiresAt,
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		_ = s.objects.Delete(ctx, dataKey)
		return run, err
	}
	manifestDigest := sha256.Sum256(manifestData)
	manifestChecksum := hex.EncodeToString(manifestDigest[:])
	manifestInfo, err := s.objects.Put(ctx, manifestKey, bytes.NewReader(manifestData), MaxReportRunBytes)
	if err != nil {
		_ = s.objects.Delete(ctx, dataKey)
		_ = s.objects.Delete(ctx, manifestKey)
		return run, err
	}
	if manifestInfo.Key != manifestKey || manifestInfo.SizeBytes != int64(len(manifestData)) || manifestInfo.SHA256 != manifestChecksum {
		_ = s.objects.Delete(ctx, dataKey)
		_ = s.objects.Delete(ctx, manifestKey)
		return run, ErrArtifactIntegrity
	}

	completedAt := s.now()
	run.Status = RunReady
	run.RowCount = rowCount
	run.DataObjectKey = dataKey
	run.DataSHA256 = dataChecksum
	run.ManifestObjectKey = manifestKey
	run.ManifestSHA256 = manifestChecksum
	run.CompletedAt = &completedAt
	completed, err := s.repo.CompleteRun(ctx, scope, run)
	if err != nil {
		// Both complete objects are present, so a reclaimed execution can
		// deterministically replace them. Do not delete one and publish a
		// partial package after an ambiguous completion result.
		return run, err
	}
	return completed, nil
}

func (s *Service) renderRun(ctx context.Context, run ReportRun) ([]byte, int, error) {
	buffer := &boundedReportBuffer{max: MaxReportRunBytes}
	var csvWriter *csv.Writer
	var jsonEncoder *json.Encoder
	if run.Format == FormatCSV {
		csvWriter = csv.NewWriter(buffer)
	} else if run.Format == FormatNDJSON {
		jsonEncoder = json.NewEncoder(buffer)
		jsonEncoder.SetEscapeHTML(false)
	} else {
		return nil, 0, ErrInvalid
	}

	rowCount := 0
	cursor := ""
	columnsWritten := false
	var columns []string
	for {
		page, err := s.repo.ListReportRows(ctx, ReportScope{TenantID: run.TenantID, LegalEntityID: run.LegalEntityID}, run, cursor, ReportRunPageSize)
		if err != nil {
			return nil, 0, err
		}
		if len(page.Rows) > ReportRunPageSize {
			return nil, 0, ErrInvalid
		}
		if csvWriter != nil && len(page.Rows) > 0 && !columnsWritten {
			columns = reportColumns(page)
			if err := csvWriter.Write(columns); err != nil {
				return nil, 0, normalizeRenderError(err)
			}
			columnsWritten = true
		}
		for _, row := range page.Rows {
			if rowCount >= MaxReportRunRows {
				return nil, 0, &reportLimitError{Code: FailureRowLimitExceeded, Limit: "rows"}
			}
			if csvWriter != nil {
				values := make([]string, len(columns))
				for index, column := range columns {
					if column == "id" {
						values[index] = row.ID
						continue
					}
					values[index] = reportValueString(row.Values[column])
				}
				if err := csvWriter.Write(spreadsheetSafeReportRow(values)); err != nil {
					return nil, 0, normalizeRenderError(err)
				}
			} else if err := jsonEncoder.Encode(row); err != nil {
				return nil, 0, normalizeRenderError(err)
			}
			rowCount++
		}
		if csvWriter != nil {
			csvWriter.Flush()
			if err := csvWriter.Error(); err != nil {
				return nil, 0, normalizeRenderError(err)
			}
		}
		if page.NextCursor == "" {
			break
		}
		if rowCount >= MaxReportRunRows {
			return nil, 0, &reportLimitError{Code: FailureRowLimitExceeded, Limit: "rows"}
		}
		if page.NextCursor == cursor {
			return nil, 0, ErrInvalid
		}
		cursor = page.NextCursor
	}
	if run.SourceBoundary.PopulationComplete && rowCount != run.SourceBoundary.Population {
		return nil, 0, &sourceBoundaryError{Expected: run.SourceBoundary.Population, Actual: rowCount}
	}
	return append([]byte(nil), buffer.Bytes()...), rowCount, nil
}

func (s *Service) Open(ctx context.Context, scope ReportScope, runID string) (ReportRun, io.ReadCloser, error) {
	actor, verifiedScope, err := s.scopedActor(ctx, scope)
	if err != nil {
		return ReportRun{}, nil, err
	}
	runID = strings.TrimSpace(runID)
	if s.repo == nil || s.objects == nil || runID == "" {
		return ReportRun{}, nil, ErrInvalid
	}
	if err := s.authorized(ctx, actor, verifiedScope, "REPORT_RUN", runID, authority.ResponsibilityPerformer, "report.run.download", 3); err != nil {
		return ReportRun{}, nil, err
	}
	run, err := s.repo.GetRun(ctx, verifiedScope, runID)
	if err != nil {
		return ReportRun{}, nil, err
	}
	if run.TenantID != verifiedScope.TenantID || run.LegalEntityID != verifiedScope.LegalEntityID ||
		run.Status != RunReady || run.DataObjectKey == "" || run.DataSHA256 == "" ||
		!s.now().Before(run.ExpiresAt) {
		return ReportRun{}, nil, ErrNotFound
	}
	reader, err := s.objects.Open(ctx, run.DataObjectKey)
	if err != nil {
		return ReportRun{}, nil, errors.Join(ErrNotFound, err)
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, MaxReportRunBytes+1))
	if err != nil || int64(len(data)) > MaxReportRunBytes {
		return ReportRun{}, nil, ErrNotFound
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != run.DataSHA256 {
		return ReportRun{}, nil, ErrNotFound
	}
	if err := s.repo.RecordRunDownload(ctx, verifiedScope, run.ID, actor.PrincipalID); err != nil {
		return ReportRun{}, nil, err
	}
	return run, io.NopCloser(bytes.NewReader(data)), nil
}

func (s *Service) failRun(ctx context.Context, run ReportRun, code string, cause error) error {
	failed, err := s.repo.FailRun(ctx, ReportScope{TenantID: run.TenantID, LegalEntityID: run.LegalEntityID}, run.ID, code)
	if err != nil {
		return errors.Join(cause, err)
	}
	_ = failed
	return cause
}

func (s *Service) actorScope(ctx context.Context) (identity.Actor, ReportScope, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return identity.Actor{}, ReportScope{}, err
	}
	if err := actor.Valid(s.now()); err != nil || strings.TrimSpace(actor.PrincipalID) == "" || actor.LegalEntityID == "*" {
		if err != nil {
			return identity.Actor{}, ReportScope{}, err
		}
		return identity.Actor{}, ReportScope{}, ErrInvalid
	}
	if strings.EqualFold(strings.TrimSpace(actor.Kind), "SERVICE") {
		return identity.Actor{}, ReportScope{}, ErrClosureBlocked
	}
	scope := ReportScope{TenantID: strings.TrimSpace(actor.TenantID), LegalEntityID: strings.TrimSpace(actor.LegalEntityID)}
	if err := validateReportScope(scope); err != nil {
		return identity.Actor{}, ReportScope{}, err
	}
	return actor, scope, nil
}

func (s *Service) scopedActor(ctx context.Context, requested ReportScope) (identity.Actor, ReportScope, error) {
	actor, scope, err := s.actorScope(ctx)
	if err != nil {
		return identity.Actor{}, ReportScope{}, err
	}
	requested.TenantID = strings.TrimSpace(requested.TenantID)
	requested.LegalEntityID = strings.TrimSpace(requested.LegalEntityID)
	if err := validateReportScope(requested); err != nil || requested != scope {
		return identity.Actor{}, ReportScope{}, ErrInvalid
	}
	return actor, scope, nil
}

func (s *Service) authorized(ctx context.Context, actor identity.Actor, scope ReportScope, objectType, objectID string, responsibility authority.Responsibility, decisionType string, materiality int) error {
	if s.authority == nil {
		return errors.Join(ErrClosureBlocked, errors.New("current report authority is unavailable"))
	}
	resolution, err := s.authority.Resolve(ctx, authority.ResolveInput{
		TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
		ObjectType: objectType, ObjectID: objectID, Responsibility: responsibility,
		DecisionType: decisionType, Materiality: materiality, At: s.now(),
	})
	if err != nil {
		return err
	}
	if strings.TrimSpace(resolution.PolicyVersion) == "" || !resolution.AllowsPrincipal(actor.PrincipalID) {
		return ErrClosureBlocked
	}
	return nil
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func validateDefinitionForCreate(definition ReportDefinition) error {
	if !reportCodePattern.MatchString(definition.Code) ||
		utf8.RuneCountInString(definition.Name) < 3 || utf8.RuneCountInString(definition.Name) > 120 ||
		utf8.RuneCountInString(definition.Description) > 1000 ||
		!validReportDataset(definition.Dataset) || !validReportScope(definition.ScopeKind, definition.ScopeRef) ||
		(definition.Format != FormatCSV && definition.Format != FormatNDJSON) ||
		strings.TrimSpace(definition.MakerID) == "" || definition.Filter == nil {
		return ErrInvalid
	}
	if definition.EffectiveFrom != nil && definition.EffectiveUntil != nil && !definition.EffectiveUntil.After(*definition.EffectiveFrom) {
		return ErrInvalid
	}
	return nil
}

func validReportDataset(dataset ReportDataset) bool {
	return dataset == DatasetProcessingActivities || dataset == DatasetProcessingActivityExceptions ||
		dataset == DatasetPrograms || dataset == DatasetMatterExceptions
}

func validReportScope(kind ReportScopeKind, reference string) bool {
	reference = strings.TrimSpace(reference)
	if kind == ScopeLegalEntity {
		return reference == ""
	}
	return (kind == ScopeProgram || kind == ScopeMatter) && isUUID(reference)
}

func normalizedStoredFilter(expression *ReportFilterExpression) (*ReportFilterExpression, error) {
	normalized, err := NormalizeReportFilter(expression)
	if err != nil {
		return nil, err
	}
	if normalized == nil {
		return &ReportFilterExpression{Kind: "group", Operator: "and", Children: []ReportFilterExpression{}}, nil
	}
	return cloneReportFilter(normalized), nil
}

func cloneReportFilter(expression *ReportFilterExpression) *ReportFilterExpression {
	if expression == nil {
		return nil
	}
	raw, err := json.Marshal(expression)
	if err != nil {
		return nil
	}
	var cloned ReportFilterExpression
	if err := json.Unmarshal(raw, &cloned); err != nil {
		return nil
	}
	return &cloned
}

func definitionIsEffective(definition ReportDefinition, now time.Time) bool {
	if definition.Status != DefinitionActive || definition.EffectiveFrom == nil || definition.ApprovedAt == nil || definition.ReviewerID == "" {
		return false
	}
	if now.Before(definition.EffectiveFrom.UTC()) {
		return false
	}
	return definition.EffectiveUntil == nil || now.Before(definition.EffectiveUntil.UTC())
}

func validateReportScope(scope ReportScope) error {
	if !isUUID(strings.TrimSpace(scope.TenantID)) || !isUUID(strings.TrimSpace(scope.LegalEntityID)) {
		return ErrInvalid
	}
	return nil
}

func validateSourceBoundary(boundary SourceBoundary) error {
	if boundary.CapturedAt.IsZero() || strings.TrimSpace(boundary.ProjectionVersion) == "" ||
		boundary.SourceHighWater == nil || boundary.Population < 0 {
		return ErrInvalid
	}
	return nil
}

func validateReportRunIdentity(run ReportRun) error {
	if err := validateReportScope(ReportScope{TenantID: run.TenantID, LegalEntityID: run.LegalEntityID}); err != nil {
		return err
	}
	if !isUUID(strings.TrimSpace(run.ID)) || !isUUID(strings.TrimSpace(run.DefinitionID)) ||
		run.DefinitionVersion <= 0 || !reportCodePattern.MatchString(run.DefinitionCode) ||
		len(run.DefinitionChecksum) != 64 || !validReportScope(run.ScopeKind, run.ScopeRef) {
		return ErrInvalid
	}
	return nil
}

func validateClaimedRun(run ReportRun) error {
	if err := validateReportRunIdentity(run); err != nil {
		return err
	}
	if run.Status != RunRunning || run.AttemptCount <= 0 {
		return ErrConflict
	}
	return nil
}

type reportLimitError struct {
	Code  string
	Limit string
}

func (e *reportLimitError) Error() string {
	return fmt.Sprintf("report %s limit exceeded", e.Limit)
}

func (e *reportLimitError) Unwrap() error { return ErrReportTooLarge }

type sourceBoundaryError struct {
	Expected int
	Actual   int
}

func (e *sourceBoundaryError) Error() string {
	return fmt.Sprintf("report source population changed: expected %d rows, read %d", e.Expected, e.Actual)
}

type boundedReportBuffer struct {
	bytes.Buffer
	max int64
}

func (b *boundedReportBuffer) Write(value []byte) (int, error) {
	remaining := b.max - int64(b.Len())
	if int64(len(value)) > remaining {
		written := 0
		if remaining > 0 {
			written, _ = b.Buffer.Write(value[:remaining])
		}
		return written, &reportLimitError{Code: FailureByteLimitExceeded, Limit: "byte"}
	}
	return b.Buffer.Write(value)
}

func normalizeRenderError(err error) error {
	var limitError *reportLimitError
	if errors.As(err, &limitError) {
		return limitError
	}
	return err
}

func reportColumns(page ReportPage) []string {
	seen := make(map[string]struct{})
	columns := make([]string, 0, len(page.Columns)+len(page.Rows)+1)
	appendColumn := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		columns = append(columns, value)
	}
	appendColumn("id")
	for _, column := range page.Columns {
		appendColumn(column)
	}
	if len(page.Rows) > 0 {
		keys := make([]string, 0, len(page.Rows[0].Values))
		for key := range page.Rows[0].Values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			appendColumn(key)
		}
	}
	return columns
}

func reportValueString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case time.Time:
		return typed.UTC().Format(time.RFC3339Nano)
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

func spreadsheetSafeReportRow(values []string) []string {
	result := make([]string, len(values))
	for index, value := range values {
		for position, character := range value {
			if position == 0 && (character == '\t' || character == '\r' || character == '\n') {
				result[index] = "'" + value
				break
			}
			if unicode.IsSpace(character) {
				continue
			}
			if character == '=' || character == '+' || character == '-' || character == '@' {
				result[index] = "'" + value
			}
			break
		}
		if result[index] == "" {
			result[index] = value
		}
	}
	return result
}

func reportObjectKeys(run ReportRun) (string, string) {
	base := fmt.Sprintf("reports/%s/%s/%s", run.TenantID, run.LegalEntityID, run.ID)
	return base + "/report" + reportExtension(run.Format), base + "/manifest.json"
}

func reportExtension(format ReportFormat) string {
	if format == FormatNDJSON {
		return ".ndjson"
	}
	return ".csv"
}

func normalizedTimeValue(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := value.UTC()
	return &result
}
