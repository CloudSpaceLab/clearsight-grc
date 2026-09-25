package reporting

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	DemoTenant      = "bank-demo"
	DemoLegalEntity = "bank-ng"

	DemoProgramRef = "00000000-0000-7000-8000-000000000521"
	DemoMatterRef  = "00000000-0000-7000-8000-000000000522"

	demoDefinitionOpenExceptionsID = "00000000-0000-7000-8000-000000000501"
	demoDefinitionCrossBorderID    = "00000000-0000-7000-8000-000000000502"
	demoDefinitionOverdueIssuesID  = "00000000-0000-7000-8000-000000000503"
	demoDefinitionProgramHealthID  = "00000000-0000-7000-8000-000000000504"
	demoFailedRunID                = "00000000-0000-7000-8000-000000000505"

	DemoMakerPrincipalID      = "privacy-report-maker-demo"
	DemoReviewerPrincipalID   = "privacy-report-reviewer-demo"
	DemoAuthorizerPrincipalID = "privacy-report-authorizer-demo"
	DemoPerformerPrincipalID  = "privacy-report-performer-demo"

	maxMemoryReportListRows = 500
	maxMemoryReportPageSize = ReportRunPageSize
)

// MemoryRepository is the deterministic report state boundary used by local
// development, tests and the explicitly labelled demo. It enforces the same
// version, transition, separation, terminal-run and full-scope rules as the
// PostgreSQL repository so demo behaviour cannot exceed production behaviour.
type MemoryRepository struct {
	mu          sync.Mutex
	definitions map[string]ReportDefinition
	revisions   map[string][]ReportDefinitionRevision
	runs        map[string]ReportRun
	rows        map[string][]ReportRow
	downloads   []ReportDownload
	now         func() time.Time
}

// ReportDownload is a reconstructable in-memory download receipt used by demo
// tests. Production persists the equivalent report-owned outbox event.
type ReportDownload struct {
	RunID         string
	TenantID      string
	LegalEntityID string
	PrincipalID   string
	DownloadedAt  time.Time
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		definitions: make(map[string]ReportDefinition),
		revisions:   make(map[string][]ReportDefinitionRevision),
		runs:        make(map[string]ReportRun),
		rows:        make(map[string][]ReportRow),
		now:         time.Now,
	}
}

func (r *MemoryRepository) CreateDefinition(ctx context.Context, scope ReportScope, definition ReportDefinition, revision ReportDefinitionRevision) (ReportDefinition, error) {
	if r == nil || ctx == nil {
		return ReportDefinition{}, ErrInvalid
	}
	if err := validateReportScope(scope); err != nil {
		return ReportDefinition{}, err
	}
	if definition.TenantID != scope.TenantID || definition.LegalEntityID != scope.LegalEntityID {
		return ReportDefinition{}, ErrNotFound
	}
	if err := validateDefinitionForCreate(definition); err != nil {
		return ReportDefinition{}, err
	}
	if _, err := NormalizeReportFilterForDataset(definition.Dataset, definition.Filter); err != nil {
		return ReportDefinition{}, err
	}
	if definition.Status != DefinitionDraft || definition.Version != 1 || definition.CurrentVersion != 1 ||
		definition.StoredChecksum == "" || definition.StoredChecksum != definition.Checksum() {
		return ReportDefinition{}, ErrInvalid
	}
	if err := validateDefinitionRevision(revision, definition); err != nil {
		return ReportDefinition{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.definitions[definition.ID]; exists {
		return ReportDefinition{}, ErrConflict
	}
	for _, existing := range r.definitions {
		if existing.TenantID == scope.TenantID && existing.LegalEntityID == scope.LegalEntityID &&
			existing.Code == definition.Code && existing.Status != DefinitionRetired {
			return ReportDefinition{}, ErrConflict
		}
	}
	stored := cloneReportDefinition(definition)
	stored.Effective = false
	r.definitions[stored.ID] = stored
	r.revisions[stored.ID] = []ReportDefinitionRevision{cloneReportDefinitionRevision(revision)}
	return cloneReportDefinition(stored), nil
}

func (r *MemoryRepository) GetDefinition(ctx context.Context, scope ReportScope, id string) (ReportDefinition, error) {
	if r == nil || ctx == nil {
		return ReportDefinition{}, ErrInvalid
	}
	if err := validateReportScope(scope); err != nil || strings.TrimSpace(id) == "" {
		return ReportDefinition{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	definition, ok := r.definitions[strings.TrimSpace(id)]
	if !ok || definition.TenantID != scope.TenantID || definition.LegalEntityID != scope.LegalEntityID {
		return ReportDefinition{}, ErrNotFound
	}
	definition.Effective = definitionIsEffective(definition, r.clock())
	return cloneReportDefinition(definition), nil
}

func (r *MemoryRepository) GetDefinitionByCode(ctx context.Context, scope ReportScope, code string) (ReportDefinition, error) {
	if r == nil || ctx == nil {
		return ReportDefinition{}, ErrInvalid
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if err := validateReportScope(scope); err != nil || code == "" {
		return ReportDefinition{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var matches []ReportDefinition
	for _, definition := range r.definitions {
		if definition.TenantID == scope.TenantID && definition.LegalEntityID == scope.LegalEntityID && definition.Code == code {
			matches = append(matches, definition)
		}
	}
	if len(matches) == 0 {
		return ReportDefinition{}, ErrNotFound
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Status != DefinitionRetired && matches[j].Status == DefinitionRetired {
			return true
		}
		if matches[i].Status == DefinitionRetired && matches[j].Status != DefinitionRetired {
			return false
		}
		if matches[i].CreatedAt.Equal(matches[j].CreatedAt) {
			return matches[i].ID > matches[j].ID
		}
		return matches[i].CreatedAt.After(matches[j].CreatedAt)
	})
	definition := matches[0]
	definition.Effective = definitionIsEffective(definition, r.clock())
	return cloneReportDefinition(definition), nil
}

func (r *MemoryRepository) ListDefinitions(ctx context.Context, scope ReportScope, includeRetired bool) ([]ReportDefinition, error) {
	if r == nil || ctx == nil {
		return nil, ErrInvalid
	}
	if err := validateReportScope(scope); err != nil {
		return nil, err
	}
	r.mu.Lock()
	definitions := make([]ReportDefinition, 0)
	for _, definition := range r.definitions {
		if definition.TenantID != scope.TenantID || definition.LegalEntityID != scope.LegalEntityID {
			continue
		}
		if !includeRetired && definition.Status == DefinitionRetired {
			continue
		}
		definitions = append(definitions, cloneReportDefinition(definition))
	}
	now := r.clock()
	r.mu.Unlock()
	sort.Slice(definitions, func(i, j int) bool {
		if definitions[i].CreatedAt.Equal(definitions[j].CreatedAt) {
			return definitions[i].ID > definitions[j].ID
		}
		return definitions[i].CreatedAt.After(definitions[j].CreatedAt)
	})
	if len(definitions) > maxMemoryReportListRows {
		definitions = definitions[:maxMemoryReportListRows]
	}
	for index := range definitions {
		definitions[index].Effective = definitionIsEffective(definitions[index], now)
	}
	return definitions, nil
}

func (r *MemoryRepository) ListDefinitionHistory(ctx context.Context, scope ReportScope, id string) ([]ReportDefinitionRevision, error) {
	if _, err := r.GetDefinition(ctx, scope, id); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	revisions := r.revisions[strings.TrimSpace(id)]
	result := make([]ReportDefinitionRevision, 0, len(revisions))
	for _, revision := range revisions {
		result = append(result, cloneReportDefinitionRevision(revision))
	}
	return result, nil
}

func (r *MemoryRepository) TransitionDefinition(ctx context.Context, scope ReportScope, id string, expectedVersion int64, next DefinitionStatus, decision DecisionRecord) (ReportDefinition, error) {
	if r == nil || ctx == nil {
		return ReportDefinition{}, ErrInvalid
	}
	id = strings.TrimSpace(id)
	if err := validateReportScope(scope); err != nil || id == "" || expectedVersion <= 0 ||
		strings.TrimSpace(decision.ActorID) == "" || strings.TrimSpace(decision.ChecksumSeen) == "" || decision.Timestamp.IsZero() {
		return ReportDefinition{}, ErrInvalid
	}
	decision.ActorID = strings.TrimSpace(decision.ActorID)
	decision.Action = strings.ToUpper(strings.TrimSpace(decision.Action))
	decision.Note = strings.TrimSpace(decision.Note)
	decision.Timestamp = decision.Timestamp.UTC()
	if utf8.RuneCountInString(decision.Note) > maxDecisionNote {
		return ReportDefinition{}, ErrInvalid
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.definitions[id]
	if !ok || current.TenantID != scope.TenantID || current.LegalEntityID != scope.LegalEntityID {
		return ReportDefinition{}, ErrNotFound
	}
	if current.Version != expectedVersion || decision.ChecksumSeen != current.StoredChecksum {
		return ReportDefinition{}, ErrConflict
	}
	if err := ValidateTransitionForWrite(current, next); err != nil {
		return ReportDefinition{}, err
	}
	if err := validateDefinitionDecision(current, next, decision); err != nil {
		return ReportDefinition{}, err
	}
	revisions := r.revisions[id]
	if len(revisions) == 0 {
		return ReportDefinition{}, ErrInvalid
	}
	last := len(revisions) - 1
	if revisions[last].Version != current.CurrentVersion {
		return ReportDefinition{}, ErrConflict
	}

	transitioned := applyDefinitionDecision(current, next, decision)
	transitioned.Version = expectedVersion + 1
	transitioned.UpdatedAt = decision.Timestamp
	transitioned.Effective = definitionIsEffective(transitioned, r.clock())
	revisions[last].DecisionNote = decision.Note
	switch decision.Action {
	case DecisionSubmit:
		// Submission keeps the immutable revision at PROPOSED until review.
	case DecisionReview:
		revisions[last].Decision = "REVIEWED"
		revisions[last].ReviewedBy = decision.ActorID
		revisions[last].ReviewedAt = timePtrCopy(decision.Timestamp)
	case DecisionActivate:
		revisions[last].Decision = "APPROVED"
		revisions[last].ApprovedBy = decision.ActorID
		revisions[last].ApprovedAt = timePtrCopy(decision.Timestamp)
	case DecisionReject:
		revisions[last].Decision = "REJECTED"
		if revisions[last].ReviewedBy == "" {
			revisions[last].ReviewedBy = decision.ActorID
			revisions[last].ReviewedAt = timePtrCopy(decision.Timestamp)
		}
	case DecisionRetire:
		revisions[last].Decision = "RETIRED"
	}
	r.definitions[id] = cloneReportDefinition(transitioned)
	r.revisions[id] = cloneReportDefinitionRevisions(revisions)
	return cloneReportDefinition(transitioned), nil
}

func (r *MemoryRepository) CreateRun(ctx context.Context, scope ReportScope, run ReportRun) (ReportRun, error) {
	if r == nil || ctx == nil {
		return ReportRun{}, ErrInvalid
	}
	if err := validateReportScope(scope); err != nil {
		return ReportRun{}, err
	}
	if run.TenantID != scope.TenantID || run.LegalEntityID != scope.LegalEntityID {
		return ReportRun{}, ErrNotFound
	}
	if err := validateReportRunIdentity(run); err != nil {
		return ReportRun{}, err
	}
	if _, err := NormalizeReportFilterForDataset(run.Dataset, run.Filter); err != nil {
		return ReportRun{}, err
	}
	if err := validateSourceBoundary(run.SourceBoundary); err != nil {
		return ReportRun{}, err
	}
	if run.Status != RunQueued || run.AttemptCount != 0 || run.RowCount != 0 || run.CompletedAt != nil ||
		run.DataObjectKey != "" || run.DataSHA256 != "" || run.ManifestObjectKey != "" || run.ManifestSHA256 != "" || run.FailureCode != "" ||
		!run.ExpiresAt.After(run.CreatedAt) {
		return ReportRun{}, ErrInvalid
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.runs[run.ID]; exists {
		return ReportRun{}, ErrConflict
	}
	definition, ok := r.definitions[run.DefinitionID]
	if !ok || definition.TenantID != scope.TenantID || definition.LegalEntityID != scope.LegalEntityID {
		return ReportRun{}, ErrNotFound
	}
	if definition.Status != DefinitionActive || definition.CurrentVersion != run.DefinitionVersion ||
		definition.StoredChecksum != run.DefinitionChecksum || definition.Dataset != run.Dataset ||
		definition.ScopeKind != run.ScopeKind || definition.ScopeRef != run.ScopeRef || definition.Format != run.Format {
		return ReportRun{}, ErrConflict
	}
	r.runs[run.ID] = cloneReportRun(run)
	return cloneReportRun(run), nil
}

func (r *MemoryRepository) GetRun(ctx context.Context, scope ReportScope, id string) (ReportRun, error) {
	if r == nil || ctx == nil {
		return ReportRun{}, ErrInvalid
	}
	if err := validateReportScope(scope); err != nil || strings.TrimSpace(id) == "" {
		return ReportRun{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[strings.TrimSpace(id)]
	if !ok || run.TenantID != scope.TenantID || run.LegalEntityID != scope.LegalEntityID {
		return ReportRun{}, ErrNotFound
	}
	return cloneReportRun(run), nil
}

func (r *MemoryRepository) ListRuns(ctx context.Context, scope ReportScope, definitionID string, limit int) ([]ReportRun, error) {
	if r == nil || ctx == nil {
		return nil, ErrInvalid
	}
	if err := validateReportScope(scope); err != nil {
		return nil, err
	}
	definitionID = strings.TrimSpace(definitionID)
	limit = boundedMemoryListLimit(limit, 50)
	r.mu.Lock()
	runs := make([]ReportRun, 0)
	for _, run := range r.runs {
		if run.TenantID != scope.TenantID || run.LegalEntityID != scope.LegalEntityID {
			continue
		}
		if definitionID != "" && run.DefinitionID != definitionID {
			continue
		}
		runs = append(runs, cloneReportRun(run))
	}
	r.mu.Unlock()
	sort.Slice(runs, func(i, j int) bool {
		if runs[i].CreatedAt.Equal(runs[j].CreatedAt) {
			return runs[i].ID > runs[j].ID
		}
		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})
	if len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
}

func (r *MemoryRepository) ClaimQueuedRuns(ctx context.Context, scope ReportScope, workerID string, limit int) ([]ReportRun, error) {
	if r == nil || ctx == nil {
		return nil, ErrInvalid
	}
	workerID = strings.TrimSpace(workerID)
	if err := validateReportScope(scope); err != nil || workerID == "" || utf8.RuneCountInString(workerID) > 200 || limit <= 0 {
		return nil, ErrInvalid
	}
	if limit > reportClaimBatch {
		limit = reportClaimBatch
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	queued := make([]ReportRun, 0)
	for _, run := range r.runs {
		if run.TenantID == scope.TenantID && run.LegalEntityID == scope.LegalEntityID && run.Status == RunQueued {
			queued = append(queued, run)
		}
	}
	sort.Slice(queued, func(i, j int) bool {
		if queued[i].CreatedAt.Equal(queued[j].CreatedAt) {
			return queued[i].ID < queued[j].ID
		}
		return queued[i].CreatedAt.Before(queued[j].CreatedAt)
	})
	if len(queued) > limit {
		queued = queued[:limit]
	}
	claimed := make([]ReportRun, 0, len(queued))
	for _, run := range queued {
		run.AttemptCount++
		if run.AttemptCount > MaxReportRunTries {
			run.Status = RunFailed
			run.FailureCode = FailureRetryBudgetExhausted
			run.RowCount = 0
			run.DataObjectKey = ""
			run.DataSHA256 = ""
			run.ManifestObjectKey = ""
			run.ManifestSHA256 = ""
			completed := r.clock()
			run.CompletedAt = &completed
		} else {
			run.Status = RunRunning
		}
		r.runs[run.ID] = cloneReportRun(run)
		if run.Status == RunRunning {
			claimed = append(claimed, cloneReportRun(run))
		}
	}
	return claimed, nil
}

func (r *MemoryRepository) CompleteRun(ctx context.Context, scope ReportScope, run ReportRun) (ReportRun, error) {
	if r == nil || ctx == nil {
		return ReportRun{}, ErrInvalid
	}
	if err := validateReportScope(scope); err != nil {
		return ReportRun{}, err
	}
	if run.TenantID != scope.TenantID || run.LegalEntityID != scope.LegalEntityID {
		return ReportRun{}, ErrNotFound
	}
	if err := validateReadyRunForWrite(run); err != nil {
		return ReportRun{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.runs[strings.TrimSpace(run.ID)]
	if !ok || current.TenantID != scope.TenantID || current.LegalEntityID != scope.LegalEntityID {
		return ReportRun{}, ErrNotFound
	}
	if current.Status != RunRunning || current.AttemptCount != run.AttemptCount || current.DefinitionVersion != run.DefinitionVersion ||
		current.DefinitionChecksum != run.DefinitionChecksum {
		return ReportRun{}, ErrConflict
	}
	r.runs[run.ID] = cloneReportRun(run)
	return cloneReportRun(run), nil
}

func (r *MemoryRepository) FailRun(ctx context.Context, scope ReportScope, id, failureCode string) (ReportRun, error) {
	if r == nil || ctx == nil {
		return ReportRun{}, ErrInvalid
	}
	id = strings.TrimSpace(id)
	failureCode = strings.ToLower(strings.TrimSpace(failureCode))
	if err := validateReportScope(scope); err != nil || id == "" || !memoryReportFailureCodePattern.MatchString(failureCode) {
		return ReportRun{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.runs[id]
	if !ok || current.TenantID != scope.TenantID || current.LegalEntityID != scope.LegalEntityID {
		return ReportRun{}, ErrNotFound
	}
	if current.Status != RunRunning {
		return ReportRun{}, ErrConflict
	}
	completed := r.clock()
	current.Status = RunFailed
	current.FailureCode = failureCode
	current.RowCount = 0
	current.DataObjectKey = ""
	current.DataSHA256 = ""
	current.ManifestObjectKey = ""
	current.ManifestSHA256 = ""
	current.CompletedAt = &completed
	r.runs[id] = cloneReportRun(current)
	return cloneReportRun(current), nil
}

func (r *MemoryRepository) RecordRunDownload(ctx context.Context, scope ReportScope, id, downloadedBy string) (err error) {
	if r == nil || ctx == nil {
		return ErrInvalid
	}
	downloadedBy = strings.TrimSpace(downloadedBy)
	if err := validateReportScope(scope); err != nil || downloadedBy == "" {
		return ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[strings.TrimSpace(id)]
	if !ok || run.TenantID != scope.TenantID || run.LegalEntityID != scope.LegalEntityID {
		return ErrNotFound
	}
	if run.Status != RunReady || !r.clock().Before(run.ExpiresAt) {
		return ErrNotFound
	}
	r.downloads = append(r.downloads, ReportDownload{
		RunID: run.ID, TenantID: run.TenantID, LegalEntityID: run.LegalEntityID,
		PrincipalID: downloadedBy, DownloadedAt: r.clock(),
	})
	return nil
}

func (r *MemoryRepository) CaptureSourceBoundary(ctx context.Context, scope ReportScope, definition ReportDefinition) (SourceBoundary, error) {
	if r == nil || ctx == nil {
		return SourceBoundary{}, ErrInvalid
	}
	if err := validateReportScope(scope); err != nil {
		return SourceBoundary{}, err
	}
	if definition.TenantID != scope.TenantID || definition.LegalEntityID != scope.LegalEntityID {
		return SourceBoundary{}, ErrNotFound
	}
	if !validReportDataset(definition.Dataset) || !validReportDatasetScope(definition.Dataset, definition.ScopeKind) ||
		!validReportScope(definition.ScopeKind, definition.ScopeRef) {
		return SourceBoundary{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := NormalizeReportFilterForDataset(definition.Dataset, definition.Filter); err != nil {
		return SourceBoundary{}, err
	}
	now := r.clock()
	key := "processing_activities"
	switch definition.Dataset {
	case DatasetPrograms:
		key = "programs"
	case DatasetMatters, DatasetMatterExceptions:
		key = "matters"
	case DatasetVendors:
		key = "vendor_relationships"
	}
	return SourceBoundary{
		CapturedAt: now, ProjectionVersion: "memory-report-source.v1",
		SourceHighWater: map[string]time.Time{key: now}, Population: 0, PopulationComplete: true,
	}, nil
}

func (r *MemoryRepository) ListReportRows(ctx context.Context, scope ReportScope, run ReportRun, cursor string, limit int) (ReportPage, error) {
	if r == nil || ctx == nil {
		return ReportPage{}, ErrInvalid
	}
	if err := validateReportScope(scope); err != nil {
		return ReportPage{}, err
	}
	if run.TenantID != scope.TenantID || run.LegalEntityID != scope.LegalEntityID {
		return ReportPage{}, ErrNotFound
	}
	if limit <= 0 || limit > maxMemoryReportPageSize {
		limit = maxMemoryReportPageSize
	}
	start := 0
	if cursor != "" {
		parsed, err := strconv.Atoi(cursor)
		if err != nil || parsed < 0 {
			return ReportPage{}, ErrInvalid
		}
		start = parsed
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.runs[strings.TrimSpace(run.ID)]
	if !ok || current.TenantID != scope.TenantID || current.LegalEntityID != scope.LegalEntityID {
		return ReportPage{}, ErrNotFound
	}
	rows := r.rows[current.ID]
	if start > len(rows) {
		return ReportPage{}, ErrInvalid
	}
	end := start + limit
	if end > len(rows) {
		end = len(rows)
	}
	page := ReportPage{Rows: make([]ReportRow, 0, end-start)}
	for _, row := range rows[start:end] {
		values := make(map[string]any, len(row.Values))
		for key, value := range row.Values {
			values[key] = value
		}
		page.Rows = append(page.Rows, ReportRow{ID: row.ID, Values: values})
	}
	if end < len(rows) {
		page.NextCursor = strconv.Itoa(end)
	}
	if len(page.Rows) > 0 {
		seen := make(map[string]struct{})
		for _, row := range page.Rows {
			for field := range row.Values {
				seen[field] = struct{}{}
			}
		}
		for field := range seen {
			page.Columns = append(page.Columns, field)
		}
		sort.Strings(page.Columns)
	}
	return page, nil
}

// ListQueuedRunScopes lets the polling maintainer discover only scopes with
// durable queued work. It performs no tenant-wide report read.
func (r *MemoryRepository) ListQueuedRunScopes(ctx context.Context, limit int) ([]ReportScope, error) {
	if r == nil || ctx == nil {
		return nil, ErrInvalid
	}
	if limit <= 0 || limit > maxMemoryReportListRows {
		limit = maxMemoryReportListRows
	}
	unique := make(map[string]ReportScope)
	r.mu.Lock()
	for _, run := range r.runs {
		if run.Status == RunQueued {
			scope := ReportScope{TenantID: run.TenantID, LegalEntityID: run.LegalEntityID}
			unique[scope.TenantID+"\x00"+scope.LegalEntityID] = scope
		}
	}
	r.mu.Unlock()
	scopes := make([]ReportScope, 0, len(unique))
	for _, scope := range unique {
		scopes = append(scopes, scope)
	}
	sort.Slice(scopes, func(i, j int) bool {
		if scopes[i].TenantID == scopes[j].TenantID {
			return scopes[i].LegalEntityID < scopes[j].LegalEntityID
		}
		return scopes[i].TenantID < scopes[j].TenantID
	})
	if len(scopes) > limit {
		scopes = scopes[:limit]
	}
	return scopes, nil
}

func (r *MemoryRepository) Downloads() []ReportDownload {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ReportDownload(nil), r.downloads...)
}

func (r *MemoryRepository) clock() time.Time {
	if r != nil && r.now != nil {
		return r.now().UTC()
	}
	return time.Now().UTC()
}

var memoryReportFailureCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_]{0,99}$`)

func validateDefinitionDecision(current ReportDefinition, next DefinitionStatus, decision DecisionRecord) error {
	expected := DefinitionStatus("")
	switch decision.Action {
	case DecisionSubmit:
		expected = DefinitionPendingReview
	case DecisionReview:
		expected = DefinitionReviewed
	case DecisionActivate:
		expected = DefinitionActive
	case DecisionReject, DecisionRetire:
		expected = DefinitionRetired
	default:
		return ErrInvalid
	}
	if expected != next || decision.Timestamp.Before(current.UpdatedAt) {
		return ErrInvalid
	}
	if decision.Action == DecisionSubmit && decision.ActorID != current.MakerID {
		return ErrClosureBlocked
	}
	if (decision.Action == DecisionReview || decision.Action == DecisionReject) && decision.ActorID == current.MakerID {
		return ErrClosureBlocked
	}
	if decision.Action == DecisionActivate && (decision.ActorID == current.MakerID || decision.ActorID == current.ReviewerID) {
		return ErrClosureBlocked
	}
	return nil
}

func applyDefinitionDecision(current ReportDefinition, next DefinitionStatus, decision DecisionRecord) ReportDefinition {
	result := current
	result.Status = next
	switch decision.Action {
	case DecisionSubmit:
		result.SubmittedAt = timePtrCopy(decision.Timestamp)
	case DecisionReview:
		result.ReviewerID = decision.ActorID
		result.ReviewerNote = decision.Note
	case DecisionActivate:
		result.CheckerID = decision.ActorID
		result.ApprovedAt = timePtrCopy(decision.Timestamp)
		result.EffectiveFrom = timePtrCopy(decision.Timestamp)
		if decision.EffectiveFrom != nil {
			result.EffectiveFrom = timePtrCopy(decision.EffectiveFrom.UTC())
		}
	case DecisionReject, DecisionRetire:
		result.RetiredAt = timePtrCopy(decision.Timestamp)
	}
	return result
}

func cloneReportDefinition(definition ReportDefinition) ReportDefinition {
	definition.Filter = cloneReportFilter(definition.Filter)
	return definition
}

func cloneReportDefinitionRevision(revision ReportDefinitionRevision) ReportDefinitionRevision {
	revision.Filter = cloneReportFilter(revision.Filter)
	revision.ReviewedAt = cloneTimePointer(revision.ReviewedAt)
	revision.ApprovedAt = cloneTimePointer(revision.ApprovedAt)
	return revision
}

func cloneReportDefinitionRevisions(revisions []ReportDefinitionRevision) []ReportDefinitionRevision {
	result := make([]ReportDefinitionRevision, len(revisions))
	for index := range revisions {
		result[index] = cloneReportDefinitionRevision(revisions[index])
	}
	return result
}

func cloneReportRun(run ReportRun) ReportRun {
	run.Filter = cloneReportFilter(run.Filter)
	run.CompletedAt = cloneTimePointer(run.CompletedAt)
	run.SourceBoundary.CapturedAt = run.SourceBoundary.CapturedAt.UTC()
	run.SourceBoundary.SourceHighWater = cloneReportTimeMap(run.SourceBoundary.SourceHighWater)
	return run
}

func cloneReportTimeMap(source map[string]time.Time) map[string]time.Time {
	if source == nil {
		return nil
	}
	result := make(map[string]time.Time, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := value.UTC()
	return &result
}

func boundedMemoryListLimit(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	if value > maxMemoryReportListRows {
		return maxMemoryReportListRows
	}
	return value
}

type demoDefinitionSeed struct {
	id          string
	code        string
	name        string
	description string
	dataset     ReportDataset
	scopeKind   ReportScopeKind
	scopeRef    string
	format      ReportFormat
	filter      *ReportFilterExpression
	status      DefinitionStatus
	version     int64
}

var demoDefinitionSeeds = []demoDefinitionSeed{
	{
		id: demoDefinitionOpenExceptionsID, code: "ROPA-OPEN-EXCEPTIONS",
		name:        "Processing activities with open exceptions",
		description: "Sample data: 4 open exceptions across 8 seeded activities in the Meridian Trust Bank estate, including Cloudspace OEM findings about information-security certification, VAPT, audit rights and ISO 27001/22301 evidence, plus Azure user and device access findings. Finacle Treasury, Fincore/Coligo, BVN Link Portal/Matching System and Soft Token records remain visible with their missing closure facts. This is sample reference data, not legal advice.",
		dataset:     DatasetProcessingActivityExceptions, scopeKind: ScopeLegalEntity, format: FormatCSV,
		filter: &ReportFilterExpression{Kind: "group", Operator: "and"}, status: DefinitionActive, version: 4,
	},
	{
		id: demoDefinitionCrossBorderID, code: "ROPA-CROSS-BORDER-TRANSFERS",
		name:        "Cross-border transfers in one program",
		description: "Sample data: cross-border processing activities in the seeded register. Cloudspace OEM is recorded as a domestic Nigerian processor, so it is not counted as a cross-border transfer.",
		dataset:     DatasetProcessingActivities, scopeKind: ScopeProgram, scopeRef: DemoProgramRef, format: FormatCSV,
		filter: &ReportFilterExpression{Kind: "group", Operator: "and", Children: []ReportFilterExpression{{
			Kind: "condition", Field: ReportFieldCrossBorder, Operator: "is", Value: "true",
		}}}, status: DefinitionActive, version: 4,
	},
	{
		id: demoDefinitionOverdueIssuesID, code: "ISSUES-OVERDUE-OBLIGATIONS",
		name:        "Overdue obligations in one issue or change",
		description: "Sample data: open issues and overdue obligations in the seeded Data Protection, IT risk and Cloudspace OEM remediation records.",
		dataset:     DatasetMatterExceptions, scopeKind: ScopeMatter, scopeRef: DemoMatterRef, format: FormatNDJSON,
		filter: &ReportFilterExpression{Kind: "group", Operator: "and", Children: []ReportFilterExpression{{
			Kind: "condition", Field: ReportFieldDueCondition, Operator: "is", Value: "OVERDUE",
		}}}, status: DefinitionPendingReview, version: 2,
	},
	{
		id: demoDefinitionProgramHealthID, code: "PROGRAM-HEALTH",
		name:        "Program health across the entity",
		description: "Sample data: current Program operating status and calculated attention state for the seeded Meridian Trust Bank reference estate.",
		dataset:     DatasetPrograms, scopeKind: ScopeLegalEntity, format: FormatCSV,
		filter: &ReportFilterExpression{Kind: "group", Operator: "and", Children: []ReportFilterExpression{{
			Kind: "condition", Field: ReportFieldOverallState, Operator: "is", Value: "AT_RISK",
		}}}, status: DefinitionReviewed, version: 3,
	},
}

// InstallDemo installs explicitly labelled sample report governance states and
// one bounded-stop run. Stable codes and IDs make repeat installation idempotent.
func InstallDemo(ctx context.Context, service *Service) error {
	if ctx == nil || service == nil || service.repo == nil {
		return ErrInvalid
	}
	repository, ok := service.repo.(*MemoryRepository)
	if !ok {
		return ErrInvalid
	}
	now := service.now()
	scope := ReportScope{TenantID: DemoTenant, LegalEntityID: DemoLegalEntity}

	repository.mu.Lock()
	defer repository.mu.Unlock()
	for _, seed := range demoDefinitionSeeds {
		if memoryDemoDefinitionExists(repository.definitions, scope, seed.code) {
			continue
		}
		definition, revision, err := buildDemoDefinition(seed, now)
		if err != nil {
			return err
		}
		if _, exists := repository.definitions[definition.ID]; exists {
			return ErrConflict
		}
		repository.definitions[definition.ID] = definition
		repository.revisions[definition.ID] = []ReportDefinitionRevision{revision}
	}
	if _, exists := repository.runs[demoFailedRunID]; !exists {
		definition := repository.definitions[demoDefinitionOpenExceptionsID]
		run := ReportRun{
			ID: demoFailedRunID, TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
			DefinitionID: definition.ID, DefinitionVersion: definition.CurrentVersion,
			DefinitionCode: definition.Code, DefinitionChecksum: definition.StoredChecksum,
			ScopeKind: definition.ScopeKind, RequestedByRef: DemoPerformerPrincipalID,
			AsOf: now.Add(-30 * time.Minute), Filter: cloneReportFilter(definition.Filter),
			Dataset: definition.Dataset, Format: definition.Format, Status: RunFailed, AttemptCount: 1,
			FailureCode: FailureRowLimitExceeded, CreatedAt: now.Add(-25 * time.Minute),
			ExpiresAt: now.Add(ReportRunRetention), SourceBoundary: SourceBoundary{
				CapturedAt: now.Add(-30 * time.Minute), ProjectionVersion: "sample-report-bound.v1",
				SourceHighWater: map[string]time.Time{"processing_activities": now.Add(-30 * time.Minute)},
				Population:      MaxReportRunRows + 1, PopulationComplete: true,
			},
		}
		completed := now.Add(-20 * time.Minute)
		run.CompletedAt = &completed
		repository.runs[run.ID] = run
	}
	return nil
}

func buildDemoDefinition(seed demoDefinitionSeed, now time.Time) (ReportDefinition, ReportDefinitionRevision, error) {
	normalized, err := NormalizeReportFilterForDataset(seed.dataset, cloneReportFilter(seed.filter))
	if err != nil {
		return ReportDefinition{}, ReportDefinitionRevision{}, err
	}
	if normalized == nil {
		normalized = &ReportFilterExpression{Kind: "group", Operator: "and"}
	}
	created := now.Add(-7 * 24 * time.Hour)
	definition := ReportDefinition{
		ID: seed.id, TenantID: DemoTenant, LegalEntityID: DemoLegalEntity, Code: seed.code,
		Name: seed.name, Description: seed.description, Dataset: seed.dataset, ScopeKind: seed.scopeKind,
		ScopeRef: seed.scopeRef, Format: seed.format, Filter: normalized, Status: seed.status,
		CurrentVersion: 1, MakerID: DemoMakerPrincipalID, CreatedAt: created, UpdatedAt: created,
		Version: seed.version,
	}
	revision := ReportDefinitionRevision{
		DefinitionID: definition.ID, TenantID: definition.TenantID, LegalEntityID: definition.LegalEntityID,
		Version: 1, BaseVersion: 0, Dataset: definition.Dataset, ScopeKind: definition.ScopeKind,
		ScopeRef: definition.ScopeRef, Format: definition.Format, Filter: cloneReportFilter(definition.Filter),
		Checksum: definition.Checksum(), MakerID: definition.MakerID, CreatedAt: created, Decision: "PROPOSED",
	}
	switch seed.status {
	case DefinitionPendingReview:
		submitted := now.Add(-2 * time.Hour)
		definition.SubmittedAt = &submitted
		definition.UpdatedAt = submitted
	case DefinitionReviewed:
		submitted := now.Add(-3 * time.Hour)
		reviewed := now.Add(-2 * time.Hour)
		definition.SubmittedAt = &submitted
		definition.ReviewerID = DemoReviewerPrincipalID
		definition.ReviewerNote = "Sample review completed; activation remains with the authorizer."
		definition.UpdatedAt = reviewed
		revision.Decision = "REVIEWED"
		revision.ReviewedBy = DemoReviewerPrincipalID
		revision.ReviewedAt = &reviewed
		revision.DecisionNote = definition.ReviewerNote
	case DefinitionActive:
		submitted := now.Add(-4 * time.Hour)
		reviewed := now.Add(-3 * time.Hour)
		approved := now.Add(-2 * time.Hour)
		effective := approved
		if seed.code == "ROPA-CROSS-BORDER-TRANSFERS" {
			effective = now.Add(24 * time.Hour)
		}
		definition.SubmittedAt = &submitted
		definition.ReviewerID = DemoReviewerPrincipalID
		definition.CheckerID = DemoAuthorizerPrincipalID
		definition.ReviewerNote = "Sample privacy review completed."
		definition.ApprovedAt = &approved
		definition.EffectiveFrom = &effective
		definition.UpdatedAt = approved
		revision.Decision = "APPROVED"
		revision.ReviewedBy = DemoReviewerPrincipalID
		revision.ReviewedAt = &reviewed
		revision.ApprovedBy = DemoAuthorizerPrincipalID
		revision.ApprovedAt = &approved
		revision.DecisionNote = "Sample privacy review and authorization completed."
	}
	definition.StoredChecksum = definition.Checksum()
	revision.Checksum = definition.StoredChecksum
	definition.Effective = definitionIsEffective(definition, now)
	return definition, revision, nil
}

func memoryDemoDefinitionExists(definitions map[string]ReportDefinition, scope ReportScope, code string) bool {
	for _, definition := range definitions {
		if definition.TenantID == scope.TenantID && definition.LegalEntityID == scope.LegalEntityID && definition.Code == code {
			return true
		}
	}
	return false
}

var _ Repository = (*MemoryRepository)(nil)
