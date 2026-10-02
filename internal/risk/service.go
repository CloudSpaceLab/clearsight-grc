package risk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	EventRiskCreated       = "RiskCreated"
	EventRiskUpdated       = "RiskUpdated"
	EventRiskAssessed      = "RiskAssessed"
	EventAppetiteActivated = "RiskAppetiteActivated"
)

type Service struct {
	repository Repository
	Now        func() time.Time
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Risk, error) {
	if s == nil || s.repository == nil {
		return Risk{}, ErrInvalid
	}
	scope, err := normalizeScope(Scope{TenantID: input.TenantID, LegalEntityID: input.LegalEntityID})
	if err != nil {
		return Risk{}, err
	}
	now := s.now()
	risk := Risk{
		TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
		Code: strings.ToUpper(strings.TrimSpace(input.Code)),
		Name: strings.TrimSpace(input.Name), Category: strings.TrimSpace(input.Category),
		Statement: strings.TrimSpace(input.Statement), Cause: strings.TrimSpace(input.Cause),
		Event: strings.TrimSpace(input.Event), Impact: strings.TrimSpace(input.Impact),
		Scope: normalizedJSON(input.Scope), OwnerPrincipalID: strings.TrimSpace(input.OwnerPrincipalID),
		Status: StatusDraft, Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := validateRisk(risk); err != nil {
		return Risk{}, err
	}
	risk.ID, err = newID()
	if err != nil {
		return Risk{}, err
	}
	event, err := riskEvent(risk, EventRiskCreated, input.ActorID, risk, now)
	if err != nil {
		return Risk{}, err
	}
	return s.repository.Create(ctx, risk, event)
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (Risk, error) {
	if s == nil || s.repository == nil {
		return Risk{}, ErrInvalid
	}
	scope, err := normalizeScope(Scope{TenantID: input.TenantID, LegalEntityID: input.LegalEntityID})
	if err != nil {
		return Risk{}, err
	}
	current, err := s.repository.Get(ctx, scope, strings.TrimSpace(input.RiskID))
	if err != nil {
		return Risk{}, err
	}
	if input.ExpectedVersion <= 0 || current.Version != input.ExpectedVersion {
		return Risk{}, ErrVersionConflict
	}
	next := current
	next.Name = strings.TrimSpace(input.Name)
	next.Category = strings.TrimSpace(input.Category)
	next.Statement = strings.TrimSpace(input.Statement)
	next.Cause = strings.TrimSpace(input.Cause)
	next.Event = strings.TrimSpace(input.Event)
	next.Impact = strings.TrimSpace(input.Impact)
	next.Scope = normalizedJSON(input.Scope)
	next.OwnerPrincipalID = strings.TrimSpace(input.OwnerPrincipalID)
	next.Status = input.Status
	next.Version = current.Version + 1
	now := s.now()
	next.UpdatedAt = now
	if err := validateRisk(next); err != nil {
		return Risk{}, err
	}
	event, err := riskEvent(next, EventRiskUpdated, input.ActorID, next, now)
	if err != nil {
		return Risk{}, err
	}
	return s.repository.Update(ctx, scope, next, input.ExpectedVersion, event)
}

func (s *Service) AddAssessment(ctx context.Context, input AssessmentInput) (Risk, Assessment, error) {
	if s == nil || s.repository == nil {
		return Risk{}, Assessment{}, ErrInvalid
	}
	scope, err := normalizeScope(Scope{TenantID: input.TenantID, LegalEntityID: input.LegalEntityID})
	if err != nil {
		return Risk{}, Assessment{}, err
	}
	current, err := s.repository.Get(ctx, scope, strings.TrimSpace(input.RiskID))
	if err != nil {
		return Risk{}, Assessment{}, err
	}
	if input.ExpectedRiskVersion <= 0 || current.Version != input.ExpectedRiskVersion {
		return Risk{}, Assessment{}, ErrVersionConflict
	}
	now := s.now()
	assessment := Assessment{
		RiskID: current.ID, RiskVersion: current.Version + 1,
		Kind: input.Kind, MethodCode: strings.TrimSpace(input.MethodCode), MethodVersion: strings.TrimSpace(input.MethodVersion),
		Dimensions: normalizedJSON(input.Dimensions), Assumptions: normalizedJSON(input.Assumptions),
		EvidenceReferences: normalizedJSONArray(input.EvidenceReferences), Confidence: normalizedConfidence(input.Confidence),
		AssessedBy: strings.TrimSpace(input.ActorID), AppetiteStatementID: strings.TrimSpace(input.AppetiteStatementID),
		AppetitePosition: input.AppetitePosition, AppetiteRationale: strings.TrimSpace(input.AppetiteRationale),
		AssessedAt: now, CreatedAt: now,
	}
	if assessment.AppetiteStatementID == "" {
		if assessment.AppetitePosition != AppetiteUnknown {
			return Risk{}, Assessment{}, ErrInvalid
		}
	} else {
		currentAppetite, err := s.repository.CurrentAppetite(ctx, scope, current.ID, assessment.AssessedAt)
		if err != nil {
			return Risk{}, Assessment{}, err
		}
		if currentAppetite == nil || currentAppetite.ID != assessment.AppetiteStatementID {
			return Risk{}, Assessment{}, ErrInvalid
		}
	}
	if err := validateAssessment(assessment); err != nil {
		return Risk{}, Assessment{}, err
	}
	assessment.ID, err = newID()
	if err != nil {
		return Risk{}, Assessment{}, err
	}
	event, err := riskEventWithVersion(current, current.Version+1, EventRiskAssessed, input.ActorID, assessment, now)
	if err != nil {
		return Risk{}, Assessment{}, err
	}
	return s.repository.AddAssessment(ctx, scope, current.ID, input.ExpectedRiskVersion, assessment, event)
}

func (s *Service) ActivateAppetite(ctx context.Context, input AppetiteInput) (Risk, AppetiteStatement, error) {
	if s == nil || s.repository == nil {
		return Risk{}, AppetiteStatement{}, ErrInvalid
	}
	scope, err := normalizeScope(Scope{TenantID: input.TenantID, LegalEntityID: input.LegalEntityID})
	if err != nil {
		return Risk{}, AppetiteStatement{}, err
	}
	current, err := s.repository.Get(ctx, scope, strings.TrimSpace(input.RiskID))
	if err != nil {
		return Risk{}, AppetiteStatement{}, err
	}
	if input.ExpectedRiskVersion <= 0 || current.Version != input.ExpectedRiskVersion {
		return Risk{}, AppetiteStatement{}, ErrVersionConflict
	}
	statements, err := s.repository.AppetiteStatements(ctx, scope, current.ID, 100)
	if err != nil {
		return Risk{}, AppetiteStatement{}, err
	}
	version := int64(1)
	for _, value := range statements {
		if value.Version >= version {
			version = value.Version + 1
		}
	}
	now := s.now()
	effectiveFrom := input.EffectiveFrom.UTC()
	if effectiveFrom.IsZero() {
		effectiveFrom = now
	}
	statement := AppetiteStatement{
		RiskID: current.ID, RiskVersion: current.Version + 1, Version: version,
		Statement: strings.TrimSpace(input.Statement), Rule: normalizedJSON(input.Rule),
		Rationale: strings.TrimSpace(input.Rationale), OwnerPrincipalID: strings.TrimSpace(input.OwnerPrincipalID),
		AuthorityPrincipalID: strings.TrimSpace(input.ActorID), Status: AppetiteActive,
		EffectiveFrom: effectiveFrom, EffectiveUntil: normalizedTimePointer(input.EffectiveUntil), CreatedAt: now,
	}
	if statement.OwnerPrincipalID == "" {
		statement.OwnerPrincipalID = current.OwnerPrincipalID
	}
	if err := validateAppetite(statement); err != nil {
		return Risk{}, AppetiteStatement{}, err
	}
	statement.ID, err = newID()
	if err != nil {
		return Risk{}, AppetiteStatement{}, err
	}
	event, err := riskEventWithVersion(current, current.Version+1, EventAppetiteActivated, input.ActorID, statement, now)
	if err != nil {
		return Risk{}, AppetiteStatement{}, err
	}
	return s.repository.AddAppetite(ctx, scope, current.ID, input.ExpectedRiskVersion, statement, event)
}

func (s *Service) Get(ctx context.Context, scope Scope, riskID string) (Aggregate, error) {
	if s == nil || s.repository == nil {
		return Aggregate{}, ErrInvalid
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return Aggregate{}, err
	}
	current, err := s.repository.Get(ctx, scope, strings.TrimSpace(riskID))
	if err != nil {
		return Aggregate{}, err
	}
	assessments, err := s.repository.Assessments(ctx, scope, current.ID, 100)
	if err != nil {
		return Aggregate{}, err
	}
	appetite, err := s.repository.AppetiteStatements(ctx, scope, current.ID, 100)
	if err != nil {
		return Aggregate{}, err
	}
	return Aggregate{Risk: current, Assessments: assessments, Appetite: appetite}, nil
}

func (s *Service) List(ctx context.Context, scope Scope, filter ListFilter) (Page, error) {
	if s == nil || s.repository == nil {
		return Page{}, ErrInvalid
	}
	scope, err := normalizeScope(scope)
	if err != nil {
		return Page{}, err
	}
	filter.Status = Status(strings.ToUpper(strings.TrimSpace(string(filter.Status))))
	filter.Category = strings.TrimSpace(filter.Category)
	filter.OwnerPrincipalID = strings.TrimSpace(filter.OwnerPrincipalID)
	filter.Search = strings.TrimSpace(filter.Search)
	filter.AppetitePosition = AppetitePosition(strings.ToUpper(strings.TrimSpace(string(filter.AppetitePosition))))
	filter.Cursor = strings.TrimSpace(filter.Cursor)
	if filter.Limit <= 0 {
		filter.Limit = 50
	} else if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Status != "" && !validStatus(filter.Status) {
		return Page{}, ErrInvalid
	}
	if filter.AppetitePosition != "" && !validAppetitePosition(filter.AppetitePosition) {
		return Page{}, ErrInvalid
	}
	filter.AsOf = s.now()
	return s.repository.List(ctx, scope, filter)
}

func validateRisk(value Risk) error {
	if strings.TrimSpace(value.Code) == "" || strings.TrimSpace(value.Name) == "" ||
		strings.TrimSpace(value.Statement) == "" || strings.TrimSpace(value.Impact) == "" ||
		!validStatus(value.Status) || value.Version <= 0 || !validJSONObject(value.Scope) {
		return ErrInvalid
	}
	return nil
}

func validateAssessment(value Assessment) error {
	if value.RiskID == "" || value.RiskVersion <= 1 || !validAssessmentKind(value.Kind) ||
		value.MethodCode == "" || value.MethodVersion == "" || !validJSONObject(value.Dimensions) ||
		!validJSONObject(value.Assumptions) || !validJSONArray(value.EvidenceReferences) ||
		!validAppetitePosition(value.AppetitePosition) || value.AssessedAt.IsZero() {
		return ErrInvalid
	}
	if value.Confidence != nil && (*value.Confidence < 0 || *value.Confidence > 1) {
		return ErrInvalid
	}
	return nil
}

func validateAppetite(value AppetiteStatement) error {
	if value.RiskID == "" || value.RiskVersion <= 1 || value.Version <= 0 ||
		strings.TrimSpace(value.Statement) == "" || !validJSONObject(value.Rule) ||
		value.Status != AppetiteActive || value.EffectiveFrom.IsZero() ||
		(value.EffectiveUntil != nil && !value.EffectiveUntil.After(value.EffectiveFrom)) {
		return ErrInvalid
	}
	return nil
}

func validStatus(value Status) bool {
	return value == StatusDraft || value == StatusActive || value == StatusRetired
}

func validAssessmentKind(value AssessmentKind) bool {
	switch value {
	case AssessmentInherent, AssessmentCurrent, AssessmentResidual, AssessmentTarget, AssessmentStressed, AssessmentAccepted:
		return true
	default:
		return false
	}
}

func validAppetitePosition(value AppetitePosition) bool {
	switch value {
	case AppetiteWithin, AppetiteApproaching, AppetiteBreached, AppetiteUnknown:
		return true
	default:
		return false
	}
}

func statementAppliesAt(value AppetiteStatement, at time.Time) bool {
	return !at.Before(value.EffectiveFrom) && (value.EffectiveUntil == nil || at.Before(*value.EffectiveUntil))
}

func normalizedJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`{}`)
	}
	return append(json.RawMessage(nil), value...)
}

func normalizedJSONArray(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`[]`)
	}
	return append(json.RawMessage(nil), value...)
}

func validJSONObject(value json.RawMessage) bool {
	var decoded map[string]any
	return json.Unmarshal(value, &decoded) == nil
}

func validJSONArray(value json.RawMessage) bool {
	var decoded []any
	return json.Unmarshal(value, &decoded) == nil
}

func normalizedConfidence(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func normalizedTimePointer(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func riskEvent(risk Risk, eventType, actorID string, payload any, occurredAt time.Time) (Event, error) {
	return riskEventWithVersion(risk, risk.Version, eventType, actorID, payload, occurredAt)
}

func riskEventWithVersion(risk Risk, version int64, eventType, actorID string, payload any, occurredAt time.Time) (Event, error) {
	eventID, err := newID()
	if err != nil {
		return Event{}, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Event{}, fmt.Errorf("marshal risk event: %w", err)
	}
	return Event{
		ID: eventID, TenantID: risk.TenantID, LegalEntityID: risk.LegalEntityID,
		RiskID: risk.ID, RiskVersion: version, Type: eventType,
		Payload: body, ActorID: strings.TrimSpace(actorID), OccurredAt: occurredAt.UTC(),
	}, nil
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}


func latestApplicableAppetite(values []AppetiteStatement, at time.Time) *AppetiteStatement {
	var selected *AppetiteStatement
	for i := range values {
		value := values[i]
		if at.Before(value.EffectiveFrom) {
			continue
		}
		if selected == nil || value.Version > selected.Version {
			copy := value
			selected = &copy
		}
	}
	if selected == nil || selected.Status != AppetiteActive ||
		(selected.EffectiveUntil != nil && !at.Before(*selected.EffectiveUntil)) {
		return nil
	}
	return selected
}
