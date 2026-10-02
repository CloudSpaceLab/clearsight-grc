package risk

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	risks       map[string]Risk
	byCode      map[string]string
	assessments map[string][]Assessment
	appetite    map[string][]AppetiteStatement
	revisions   map[string][]Risk
	events      map[string][]Event
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		risks: make(map[string]Risk), byCode: make(map[string]string),
		assessments: make(map[string][]Assessment), appetite: make(map[string][]AppetiteStatement),
		revisions: make(map[string][]Risk), events: make(map[string][]Event),
	}
}

func (r *MemoryRepository) Create(ctx context.Context, risk Risk, event Event) (Risk, error) {
	if err := ctx.Err(); err != nil {
		return Risk{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := riskKey(risk.TenantID, risk.LegalEntityID, risk.ID)
	codeKey := riskCodeKey(risk.TenantID, risk.LegalEntityID, risk.Code)
	if _, ok := r.risks[key]; ok {
		return Risk{}, ErrDuplicate
	}
	if _, ok := r.byCode[codeKey]; ok {
		return Risk{}, ErrDuplicate
	}
	r.risks[key] = cloneRisk(risk)
	r.byCode[codeKey] = risk.ID
	r.revisions[key] = []Risk{cloneRisk(risk)}
	r.events[key] = []Event{cloneEvent(event)}
	return cloneRisk(risk), nil
}

func (r *MemoryRepository) Get(ctx context.Context, scope Scope, id string) (Risk, error) {
	if err := ctx.Err(); err != nil {
		return Risk{}, err
	}
	scope, err := normalizeScope(scope)
	id = strings.TrimSpace(id)
	if err != nil || id == "" {
		return Risk{}, ErrInvalid
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.risks[riskKey(scope.TenantID, scope.LegalEntityID, id)]
	if !ok {
		return Risk{}, ErrNotFound
	}
	return cloneRisk(value), nil
}

func (r *MemoryRepository) Update(ctx context.Context, scope Scope, next Risk, expected int64, event Event) (Risk, error) {
	if err := ctx.Err(); err != nil {
		return Risk{}, err
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return Risk{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := riskKey(scope.TenantID, scope.LegalEntityID, next.ID)
	current, ok := r.risks[key]
	if !ok {
		return Risk{}, ErrNotFound
	}
	if current.Version != expected {
		return Risk{}, ErrVersionConflict
	}
	if next.TenantID != current.TenantID || next.LegalEntityID != current.LegalEntityID || next.Code != current.Code || next.Version != expected+1 {
		return Risk{}, ErrInvalid
	}
	r.risks[key] = cloneRisk(next)
	r.revisions[key] = append(r.revisions[key], cloneRisk(next))
	r.events[key] = append(r.events[key], cloneEvent(event))
	return cloneRisk(next), nil
}

func (r *MemoryRepository) AddAssessment(ctx context.Context, scope Scope, id string, expected int64, assessment Assessment, event Event) (Risk, Assessment, error) {
	if err := ctx.Err(); err != nil {
		return Risk{}, Assessment{}, err
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return Risk{}, Assessment{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := riskKey(scope.TenantID, scope.LegalEntityID, strings.TrimSpace(id))
	current, ok := r.risks[key]
	if !ok {
		return Risk{}, Assessment{}, ErrNotFound
	}
	if current.Version != expected {
		return Risk{}, Assessment{}, ErrVersionConflict
	}
	if assessment.RiskID != current.ID || assessment.RiskVersion != expected+1 || event.RiskVersion != expected+1 {
		return Risk{}, Assessment{}, ErrInvalid
	}
	current.Version++
	current.UpdatedAt = event.OccurredAt.UTC()
	r.risks[key] = cloneRisk(current)
	r.assessments[key] = append(r.assessments[key], cloneAssessment(assessment))
	r.revisions[key] = append(r.revisions[key], cloneRisk(current))
	r.events[key] = append(r.events[key], cloneEvent(event))
	return cloneRisk(current), cloneAssessment(assessment), nil
}

func (r *MemoryRepository) AddAppetite(ctx context.Context, scope Scope, id string, expected int64, statement AppetiteStatement, event Event) (Risk, AppetiteStatement, error) {
	if err := ctx.Err(); err != nil {
		return Risk{}, AppetiteStatement{}, err
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return Risk{}, AppetiteStatement{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := riskKey(scope.TenantID, scope.LegalEntityID, strings.TrimSpace(id))
	current, ok := r.risks[key]
	if !ok {
		return Risk{}, AppetiteStatement{}, ErrNotFound
	}
	if current.Version != expected {
		return Risk{}, AppetiteStatement{}, ErrVersionConflict
	}
	if statement.RiskID != current.ID || statement.RiskVersion != expected+1 || event.RiskVersion != expected+1 {
		return Risk{}, AppetiteStatement{}, ErrInvalid
	}
	current.Version++
	current.UpdatedAt = event.OccurredAt.UTC()
	r.risks[key] = cloneRisk(current)
	r.appetite[key] = append(r.appetite[key], cloneAppetite(statement))
	r.revisions[key] = append(r.revisions[key], cloneRisk(current))
	r.events[key] = append(r.events[key], cloneEvent(event))
	return cloneRisk(current), cloneAppetite(statement), nil
}

func (r *MemoryRepository) Assessments(ctx context.Context, scope Scope, id string, limit int) ([]Assessment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := riskKey(scope.TenantID, scope.LegalEntityID, strings.TrimSpace(id))
	if _, ok := r.risks[key]; !ok {
		return nil, ErrNotFound
	}
	values := cloneAssessments(r.assessments[key])
	sort.Slice(values, func(i, j int) bool {
		if values[i].RiskVersion != values[j].RiskVersion {
			return values[i].RiskVersion > values[j].RiskVersion
		}
		return values[i].ID > values[j].ID
	})
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func (r *MemoryRepository) AppetiteStatements(ctx context.Context, scope Scope, id string, limit int) ([]AppetiteStatement, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := riskKey(scope.TenantID, scope.LegalEntityID, strings.TrimSpace(id))
	if _, ok := r.risks[key]; !ok {
		return nil, ErrNotFound
	}
	values := cloneAppetites(r.appetite[key])
	sort.Slice(values, func(i, j int) bool {
		if values[i].Version != values[j].Version {
			return values[i].Version > values[j].Version
		}
		return values[i].ID > values[j].ID
	})
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func (r *MemoryRepository) List(ctx context.Context, scope Scope, filter ListFilter) (Page, error) {
	if err := ctx.Err(); err != nil {
		return Page{}, err
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return Page{}, err
	}
	cursor, err := decodeListCursor(filter.Cursor)
	if err != nil {
		return Page{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	rows := make([]Risk, 0, len(r.risks))
	for _, value := range r.risks {
		if value.TenantID != scope.TenantID || value.LegalEntityID != scope.LegalEntityID {
			continue
		}
		if filter.Status != "" && value.Status != filter.Status {
			continue
		}
		if filter.Category != "" && !strings.EqualFold(value.Category, filter.Category) {
			continue
		}
		if filter.OwnerPrincipalID != "" && value.OwnerPrincipalID != filter.OwnerPrincipalID {
			continue
		}
		if filter.Search != "" && !riskMatches(value, filter.Search) {
			continue
		}
		if filter.AppetitePosition != "" {
			latest := latestAssessment(r.assessments[riskKey(value.TenantID, value.LegalEntityID, value.ID)])
			if latest == nil || latest.AppetitePosition != filter.AppetitePosition {
				continue
			}
		}
		if !cursor.UpdatedAt.IsZero() && !(value.UpdatedAt.Before(cursor.UpdatedAt) || (value.UpdatedAt.Equal(cursor.UpdatedAt) && value.ID < cursor.ID)) {
			continue
		}
		rows = append(rows, cloneRisk(value))
	}
	sort.Slice(rows, func(i, j int) bool {
		if !rows[i].UpdatedAt.Equal(rows[j].UpdatedAt) {
			return rows[i].UpdatedAt.After(rows[j].UpdatedAt)
		}
		return rows[i].ID > rows[j].ID
	})
	page := Page{}
	pageRows := rows
	if len(rows) > filter.Limit {
		pageRows = rows[:filter.Limit]
		cursorValue, err := encodeListCursor(pageRows[len(pageRows)-1])
		if err != nil {
			return Page{}, err
		}
		page.NextCursor = cursorValue
	}
	for _, value := range pageRows {
		key := riskKey(value.TenantID, value.LegalEntityID, value.ID)
		summary := Summary{Risk: cloneRisk(value)}
		if latest := latestAssessment(r.assessments[key]); latest != nil {
			cloned := cloneAssessment(*latest)
			summary.LatestAssessment = &cloned
		}
		if active := latestAppetite(r.appetite[key], filter.AsOf); active != nil {
			cloned := cloneAppetite(*active)
			summary.ActiveAppetite = &cloned
		}
		page.Items = append(page.Items, summary)
	}
	return page, nil
}

func riskKey(tenant, entity, id string) string { return tenant + "\x00" + entity + "\x00" + id }
func riskCodeKey(tenant, entity, code string) string { return tenant + "\x00" + entity + "\x00" + strings.ToUpper(strings.TrimSpace(code)) }

func riskMatches(value Risk, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	return strings.Contains(strings.ToLower(value.Code), query) ||
		strings.Contains(strings.ToLower(value.Name), query) ||
		strings.Contains(strings.ToLower(value.Category), query) ||
		strings.Contains(strings.ToLower(value.Statement), query) ||
		strings.Contains(strings.ToLower(value.Impact), query)
}

func latestAssessment(values []Assessment) *Assessment {
	if len(values) == 0 {
		return nil
	}
	index := 0
	for i := 1; i < len(values); i++ {
		if values[i].RiskVersion > values[index].RiskVersion {
			index = i
		}
	}
	value := values[index]
	return &value
}

func latestAppetite(values []AppetiteStatement, at time.Time) *AppetiteStatement {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	var selected *AppetiteStatement
	for i := range values {
		value := values[i]
		if value.Status != AppetiteActive || !statementAppliesAt(value, at) {
			continue
		}
		if selected == nil || value.Version > selected.Version {
			copy := value
			selected = &copy
		}
	}
	return selected
}

func cloneRisk(value Risk) Risk {
	value.Scope = append([]byte(nil), value.Scope...)
	return value
}
func cloneAssessment(value Assessment) Assessment {
	value.Dimensions = append([]byte(nil), value.Dimensions...)
	value.Assumptions = append([]byte(nil), value.Assumptions...)
	value.EvidenceReferences = append([]byte(nil), value.EvidenceReferences...)
	if value.Confidence != nil {
		copy := *value.Confidence
		value.Confidence = &copy
	}
	return value
}
func cloneAppetite(value AppetiteStatement) AppetiteStatement {
	value.Rule = append([]byte(nil), value.Rule...)
	if value.EffectiveUntil != nil {
		copy := *value.EffectiveUntil
		value.EffectiveUntil = &copy
	}
	return value
}
func cloneEvent(value Event) Event {
	value.Payload = append([]byte(nil), value.Payload...)
	return value
}
func cloneAssessments(values []Assessment) []Assessment {
	out := make([]Assessment, len(values))
	for i, value := range values {
		out[i] = cloneAssessment(value)
	}
	return out
}
func cloneAppetites(values []AppetiteStatement) []AppetiteStatement {
	out := make([]AppetiteStatement, len(values))
	for i, value := range values {
		out[i] = cloneAppetite(value)
	}
	return out
}

var _ Repository = (*MemoryRepository)(nil)
