package ropa

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	EventActivityCreated      = "processing_activity.created"
	EventActivityUpdated      = "processing_activity.updated"
	EventActivityTransitioned = "processing_activity.transitioned"

	// Longer aliases make the event names discoverable without changing the
	// persisted vocabulary used by the ROPA ledger.
	EventProcessingActivityCreated      = EventActivityCreated
	EventProcessingActivityUpdated      = EventActivityUpdated
	EventProcessingActivityTransitioned = EventActivityTransitioned
)

// The domain bounds below are deliberately smaller than PostgreSQL's unbounded
// text type so an event payload and a child collection cannot grow without a
// limit. 200 characters is enough for stable codes, names and identifiers used
// by the register; 4,000 characters is enough for operational descriptions,
// purposes, retention notes and other long text. 1,000 rows per child
// collection keeps a single activity's write and reconstruction bounded while
// leaving room for a large bank's detailed inventory.
const (
	MaxActivityCodeLength          = 200
	MaxActivityNameLength          = 200
	MaxActivityIdentifierLength    = 200
	MaxActivityTextLength          = 4000
	MaxActivityChildCollectionSize = 1000
)

type CreateActivityInput struct {
	TenantID                     string         `json:"tenant_id"`
	LegalEntityID                string         `json:"legal_entity_id"`
	Code                         string         `json:"code"`
	Name                         string         `json:"name"`
	Description                  string         `json:"description"`
	Purpose                      string         `json:"purpose"`
	LawfulBasis                  string         `json:"lawful_basis"`
	Controller                   string         `json:"controller"`
	Processor                    string         `json:"processor"`
	AutomatedDecisionMaking      bool           `json:"automated_decision_making"`
	DataSubjectCategories        string         `json:"data_subject_categories"`
	PersonalDataCategories       string         `json:"personal_data_categories"`
	SecurityMeasures             string         `json:"security_measures"`
	RetentionPeriod              string         `json:"retention_period"`
	StartDate                    *time.Time     `json:"start_date,omitempty"`
	EndDate                      *time.Time     `json:"end_date,omitempty"`
	NextReviewDate               *time.Time     `json:"next_review_date,omitempty"`
	OwnerPrincipalID             string         `json:"owner_principal_id,omitempty"`
	RequiredAuthorityPrincipalID string         `json:"required_authority_principal_id,omitempty"`
	ProgramID                    string         `json:"program_id,omitempty"`
	DataCategories               []DataCategory `json:"data_categories,omitempty"`
	Recipients                   []Recipient    `json:"recipients,omitempty"`
	Systems                      []System       `json:"systems,omitempty"`
	Reviews                      []Review       `json:"reviews,omitempty"`
	ActorID                      string         `json:"actor_id,omitempty"`
}

type UpdateActivityInput struct {
	TenantID                     string         `json:"tenant_id"`
	LegalEntityID                string         `json:"legal_entity_id"`
	ActivityID                   string         `json:"activity_id"`
	ID                           string         `json:"id,omitempty"`
	ExpectedVersion              int64          `json:"expected_version"`
	Code                         *string        `json:"code,omitempty"`
	Name                         *string        `json:"name,omitempty"`
	Description                  *string        `json:"description,omitempty"`
	Purpose                      *string        `json:"purpose,omitempty"`
	LawfulBasis                  *string        `json:"lawful_basis,omitempty"`
	Controller                   *string        `json:"controller,omitempty"`
	Processor                    *string        `json:"processor,omitempty"`
	AutomatedDecisionMaking      *bool          `json:"automated_decision_making,omitempty"`
	DataSubjectCategories        *string        `json:"data_subject_categories,omitempty"`
	PersonalDataCategories       *string        `json:"personal_data_categories,omitempty"`
	SecurityMeasures             *string        `json:"security_measures,omitempty"`
	RetentionPeriod              *string        `json:"retention_period,omitempty"`
	StartDate                    *time.Time     `json:"start_date,omitempty"`
	EndDate                      *time.Time     `json:"end_date,omitempty"`
	NextReviewDate               *time.Time     `json:"next_review_date,omitempty"`
	OwnerPrincipalID             *string        `json:"owner_principal_id,omitempty"`
	RequiredAuthorityPrincipalID *string        `json:"required_authority_principal_id,omitempty"`
	ProgramID                    *string        `json:"program_id,omitempty"`
	DataCategories               []DataCategory `json:"data_categories,omitempty"`
	Recipients                   []Recipient    `json:"recipients,omitempty"`
	Systems                      []System       `json:"systems,omitempty"`
	Reviews                      []Review       `json:"reviews,omitempty"`
	ActorID                      string         `json:"actor_id,omitempty"`
}

type TransitionActivityInput struct {
	TenantID        string     `json:"tenant_id"`
	LegalEntityID   string     `json:"legal_entity_id"`
	ActivityID      string     `json:"activity_id"`
	ID              string     `json:"id,omitempty"`
	ExpectedVersion int64      `json:"expected_version"`
	To              Status     `json:"to"`
	EndDate         *time.Time `json:"end_date,omitempty"`
	ActorID         string     `json:"actor_id,omitempty"`
}

type Service struct {
	repository Repository
	lister     ActivityLister
	summaries  SummaryRepository
	Now        func() time.Time
}

func NewService(repository Repository, summaries SummaryRepository) *Service {
	service := &Service{repository: repository, summaries: summaries, Now: time.Now}
	if lister, ok := repository.(ActivityLister); ok {
		service.lister = lister
	}
	return service
}

// SetLister supplies the bounded list reader. Production can use a different
// read repository from the command repository when the two have different
// query and visibility requirements.
func (s *Service) SetLister(lister ActivityLister) {
	if s != nil && lister != nil {
		s.lister = lister
	}
}

func (s *Service) CreateActivity(ctx context.Context, input CreateActivityInput) (ProcessingActivity, error) {
	if s == nil || s.repository == nil {
		return ProcessingActivity{}, ErrInvalid
	}

	now := s.now()
	activity := ProcessingActivity{
		TenantID:                     input.TenantID,
		LegalEntityID:                input.LegalEntityID,
		Code:                         input.Code,
		Name:                         input.Name,
		Description:                  input.Description,
		Status:                       StatusNew,
		Purpose:                      input.Purpose,
		LawfulBasis:                  input.LawfulBasis,
		Controller:                   input.Controller,
		Processor:                    input.Processor,
		AutomatedDecisionMaking:      input.AutomatedDecisionMaking,
		DataSubjectCategories:        input.DataSubjectCategories,
		PersonalDataCategories:       input.PersonalDataCategories,
		SecurityMeasures:             input.SecurityMeasures,
		RetentionPeriod:              input.RetentionPeriod,
		StartDate:                    input.StartDate,
		EndDate:                      input.EndDate,
		NextReviewDate:               input.NextReviewDate,
		OwnerPrincipalID:             input.OwnerPrincipalID,
		RequiredAuthorityPrincipalID: input.RequiredAuthorityPrincipalID,
		ProgramID:                    input.ProgramID,
		DataCategories:               input.DataCategories,
		Recipients:                   input.Recipients,
		Systems:                      input.Systems,
		Reviews:                      input.Reviews,
		Version:                      1,
		CreatedAt:                    now,
		UpdatedAt:                    now,
	}
	if err := validateActivityWhitespace(activity); err != nil {
		return ProcessingActivity{}, err
	}
	activity = normalizeProcessingActivity(activity)
	if err := validateActivity(activity); err != nil {
		return ProcessingActivity{}, err
	}

	scope := ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}
	if existing, err := s.repository.ActivityByCode(ctx, scope, activity.Code); err == nil {
		_ = existing
		return ProcessingActivity{}, ErrDuplicate
	} else if !errors.Is(err, ErrNotFound) {
		return ProcessingActivity{}, err
	}

	activityID, err := newActivityID()
	if err != nil {
		return ProcessingActivity{}, err
	}
	activity.ID = activityID
	event, err := s.newActivityEvent(activity, EventActivityCreated, input.ActorID, now)
	if err != nil {
		return ProcessingActivity{}, err
	}
	return s.repository.CreateActivity(ctx, activity, event)
}

func (s *Service) UpdateActivity(ctx context.Context, input UpdateActivityInput) (ProcessingActivity, error) {
	if s == nil || s.repository == nil {
		return ProcessingActivity{}, ErrInvalid
	}
	activityID := inputActivityID(input.ActivityID, input.ID)
	scope, err := normalizeActivityScope(ActivityScope{TenantID: input.TenantID, LegalEntityID: input.LegalEntityID})
	if err != nil || activityID == "" {
		return ProcessingActivity{}, ErrInvalid
	}

	current, err := s.GetActivity(ctx, scope, activityID)
	if err != nil {
		return ProcessingActivity{}, err
	}
	if current.TenantID != scope.TenantID || current.LegalEntityID != scope.LegalEntityID {
		return ProcessingActivity{}, ErrNotFound
	}
	if input.ExpectedVersion <= 0 || input.ExpectedVersion != current.Version {
		return ProcessingActivity{}, ErrVersionConflict
	}

	updated := current
	applyUpdateActivityInput(&updated, input)
	updated.Version = current.Version + 1
	updated.UpdatedAt = s.now()
	if err := validateActivityWhitespace(updated); err != nil {
		return ProcessingActivity{}, err
	}
	updated = normalizeProcessingActivity(updated)
	if err := validateActivity(updated); err != nil {
		return ProcessingActivity{}, err
	}
	if err := ValidateTransitionForWrite(current, updated, EventActivityUpdated); err != nil {
		return ProcessingActivity{}, err
	}

	if existing, lookupErr := s.repository.ActivityByCode(ctx, scope, updated.Code); lookupErr == nil {
		if existing.ID != current.ID {
			return ProcessingActivity{}, ErrDuplicate
		}
	} else if !errors.Is(lookupErr, ErrNotFound) {
		return ProcessingActivity{}, lookupErr
	}

	event, err := s.newActivityEvent(updated, EventActivityUpdated, input.ActorID, updated.UpdatedAt)
	if err != nil {
		return ProcessingActivity{}, err
	}
	version, err := s.repository.ApplyActivityEvent(ctx, scope, updated.ID, input.ExpectedVersion, event)
	if err != nil {
		return ProcessingActivity{}, err
	}
	updated.Version = version
	updated.UpdatedAt = event.OccurredAt
	return updated, nil
}

func (s *Service) TransitionActivity(ctx context.Context, input TransitionActivityInput) (ProcessingActivity, error) {
	if s == nil || s.repository == nil {
		return ProcessingActivity{}, ErrInvalid
	}
	activityID := inputActivityID(input.ActivityID, input.ID)
	scope, err := normalizeActivityScope(ActivityScope{TenantID: input.TenantID, LegalEntityID: input.LegalEntityID})
	if err != nil || activityID == "" {
		return ProcessingActivity{}, ErrInvalid
	}

	current, err := s.GetActivity(ctx, scope, activityID)
	if err != nil {
		return ProcessingActivity{}, err
	}
	if current.TenantID != scope.TenantID || current.LegalEntityID != scope.LegalEntityID {
		return ProcessingActivity{}, ErrNotFound
	}
	if input.ExpectedVersion <= 0 || input.ExpectedVersion != current.Version {
		return ProcessingActivity{}, ErrVersionConflict
	}
	if input.To == current.Status {
		return ProcessingActivity{}, ErrInvalid
	}

	updated := current
	updated.Status = input.To
	if input.To == StatusClosed && input.EndDate != nil {
		updated.EndDate = input.EndDate
	}
	updated.Version = current.Version + 1
	updated.UpdatedAt = s.now()
	if err := validateActivityWhitespace(updated); err != nil {
		return ProcessingActivity{}, err
	}
	updated = normalizeProcessingActivity(updated)
	if err := validateActivity(updated); err != nil {
		return ProcessingActivity{}, err
	}
	if err := ValidateTransitionForWrite(current, updated, EventActivityTransitioned); err != nil {
		return ProcessingActivity{}, err
	}

	event, err := s.newActivityEvent(updated, EventActivityTransitioned, input.ActorID, updated.UpdatedAt)
	if err != nil {
		return ProcessingActivity{}, err
	}
	version, err := s.repository.ApplyActivityEvent(ctx, scope, updated.ID, input.ExpectedVersion, event)
	if err != nil {
		return ProcessingActivity{}, err
	}
	updated.Version = version
	updated.UpdatedAt = event.OccurredAt
	return updated, nil
}

func (s *Service) GetActivity(ctx context.Context, scope ActivityScope, activityID string) (ProcessingActivity, error) {
	if s == nil || s.repository == nil {
		return ProcessingActivity{}, ErrNotFound
	}
	scope, err := normalizeActivityScope(scope)
	if err != nil {
		return ProcessingActivity{}, err
	}
	activity, err := s.repository.GetActivity(ctx, scope, strings.TrimSpace(activityID))
	if err != nil {
		return ProcessingActivity{}, err
	}
	if activity.TenantID != scope.TenantID || activity.LegalEntityID != scope.LegalEntityID {
		return ProcessingActivity{}, ErrNotFound
	}
	return activity, nil
}

// ActivityEvents reads one bounded, exact-scope history page. The caller can
// continue from the last returned aggregate version until hasMore is false.
func (s *Service) ActivityEvents(ctx context.Context, scope ActivityScope, activityID string, afterVersion int64, limit int) ([]Event, bool, error) {
	if s == nil || s.repository == nil {
		return nil, false, ErrNotFound
	}
	scope, err := normalizeActivityScope(scope)
	activityID = strings.TrimSpace(activityID)
	if err != nil || activityID == "" || afterVersion < 0 || limit < 0 {
		return nil, false, ErrInvalid
	}
	events, hasMore, err := s.repository.ActivityEvents(ctx, scope, activityID, afterVersion, limit)
	if err != nil {
		return nil, false, err
	}
	for _, event := range events {
		if event.AggregateType != "PROCESSING_ACTIVITY" || event.TenantID != scope.TenantID || event.LegalEntityID != scope.LegalEntityID || event.AggregateID != activityID {
			return nil, false, ErrScopeMismatch
		}
	}
	return events, hasMore, nil
}

func (s *Service) ListActivities(ctx context.Context, scope ActivityScope, filter ListActivitiesFilter) (ActivityPage, error) {
	if s == nil || s.lister == nil {
		return ActivityPage{}, ErrNotFound
	}
	scope, err := normalizeActivityScope(scope)
	if err != nil {
		return ActivityPage{}, err
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	} else if filter.Limit > 200 {
		filter.Limit = 200
	}
	filter.Status = Status(strings.ToUpper(strings.TrimSpace(string(filter.Status))))
	if filter.Status != "" && !validStatus(filter.Status) {
		return ActivityPage{}, ErrInvalid
	}
	filter.LawfulBasis = strings.TrimSpace(filter.LawfulBasis)
	filter.OwnerPrincipalID = strings.TrimSpace(filter.OwnerPrincipalID)
	filter.Search = strings.TrimSpace(filter.Search)
	filter.Cursor = strings.TrimSpace(filter.Cursor)
	page, err := s.lister.ListActivities(ctx, scope, filter)
	if err != nil {
		return ActivityPage{}, err
	}
	for _, activity := range page.Rows {
		if activity.TenantID != scope.TenantID || activity.LegalEntityID != scope.LegalEntityID {
			return ActivityPage{}, ErrScopeMismatch
		}
	}
	return page, nil
}

func (s *Service) RegisterSummary(ctx context.Context, tenantID, legalEntityID string) (RegisterSummary, error) {
	if s == nil || s.summaries == nil {
		return RegisterSummary{}, ErrNotFound
	}
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	if tenantID == "" || legalEntityID == "" {
		return RegisterSummary{}, ErrInvalid
	}
	summary, err := s.summaries.LatestSummary(ctx, tenantID, legalEntityID)
	if err != nil {
		return RegisterSummary{}, err
	}
	summary.Freshness = FreshnessCurrent
	now := s.now()
	age := now.Sub(summary.GeneratedAt)
	if summary.GeneratedAt.IsZero() || age < 0 || age > 15*time.Minute || summary.ProjectionVersion != ProjectionVersion {
		summary.Freshness = FreshnessStale
	}
	return summary, nil
}

func (s *Service) ClosureBlockers(ctx context.Context, scope ActivityScope, activityID string) ([]string, error) {
	if s == nil || s.repository == nil {
		return nil, ErrInvalid
	}
	scope, err := normalizeActivityScope(scope)
	activityID = strings.TrimSpace(activityID)
	if err != nil || activityID == "" {
		return nil, ErrInvalid
	}
	// Use the command repository's exact tenant-and-entity-scoped read. An
	// unscoped fallback could select another entity's same-ID activity.
	activity, err := s.repository.GetActivity(ctx, scope, activityID)
	if err != nil {
		return nil, err
	}
	if activity.TenantID != scope.TenantID || activity.LegalEntityID != scope.LegalEntityID {
		return nil, ErrNotFound
	}
	return closureBlockers(activity), nil
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) newActivityEvent(activity ProcessingActivity, eventType, actorID string, occurredAt time.Time) (Event, error) {
	payload, err := json.Marshal(activity)
	if err != nil {
		return Event{}, err
	}
	eventID, err := newEventID()
	if err != nil {
		return Event{}, err
	}
	actorID = strings.TrimSpace(actorID)
	actorType := "SERVICE"
	if actorID != "" {
		actorType = "USER"
	}
	return Event{
		ID:               eventID,
		TenantID:         activity.TenantID,
		LegalEntityID:    activity.LegalEntityID,
		AggregateType:    "PROCESSING_ACTIVITY",
		AggregateID:      activity.ID,
		AggregateVersion: activity.Version,
		Type:             eventType,
		Payload:          payload,
		ActorType:        actorType,
		ActorID:          actorID,
		OccurredAt:       occurredAt.UTC(),
	}, nil
}

func inputActivityID(activityID, id string) string {
	if strings.TrimSpace(activityID) != "" {
		return strings.TrimSpace(activityID)
	}
	return strings.TrimSpace(id)
}

func applyUpdateActivityInput(activity *ProcessingActivity, input UpdateActivityInput) {
	if input.Code != nil {
		activity.Code = *input.Code
	}
	if input.Name != nil {
		activity.Name = *input.Name
	}
	if input.Description != nil {
		activity.Description = *input.Description
	}
	if input.Purpose != nil {
		activity.Purpose = *input.Purpose
	}
	if input.LawfulBasis != nil {
		activity.LawfulBasis = *input.LawfulBasis
	}
	if input.Controller != nil {
		activity.Controller = *input.Controller
	}
	if input.Processor != nil {
		activity.Processor = *input.Processor
	}
	if input.AutomatedDecisionMaking != nil {
		activity.AutomatedDecisionMaking = *input.AutomatedDecisionMaking
	}
	if input.DataSubjectCategories != nil {
		activity.DataSubjectCategories = *input.DataSubjectCategories
	}
	if input.PersonalDataCategories != nil {
		activity.PersonalDataCategories = *input.PersonalDataCategories
	}
	if input.SecurityMeasures != nil {
		activity.SecurityMeasures = *input.SecurityMeasures
	}
	if input.RetentionPeriod != nil {
		activity.RetentionPeriod = *input.RetentionPeriod
	}
	if input.StartDate != nil {
		activity.StartDate = input.StartDate
	}
	if input.EndDate != nil {
		activity.EndDate = input.EndDate
	}
	if input.NextReviewDate != nil {
		activity.NextReviewDate = input.NextReviewDate
	}
	if input.OwnerPrincipalID != nil {
		activity.OwnerPrincipalID = *input.OwnerPrincipalID
	}
	if input.RequiredAuthorityPrincipalID != nil {
		activity.RequiredAuthorityPrincipalID = *input.RequiredAuthorityPrincipalID
	}
	if input.ProgramID != nil {
		activity.ProgramID = *input.ProgramID
	}
	if input.DataCategories != nil {
		activity.DataCategories = input.DataCategories
	}
	if input.Recipients != nil {
		activity.Recipients = input.Recipients
	}
	if input.Systems != nil {
		activity.Systems = input.Systems
	}
	if input.Reviews != nil {
		activity.Reviews = input.Reviews
	}
}

func validateActivityWhitespace(activity ProcessingActivity) error {
	values := []string{
		activity.ID,
		activity.TenantID,
		activity.LegalEntityID,
		activity.Code,
		activity.Name,
		activity.Description,
		activity.Purpose,
		activity.LawfulBasis,
		activity.Controller,
		activity.Processor,
		activity.DataSubjectCategories,
		activity.PersonalDataCategories,
		activity.SecurityMeasures,
		activity.RetentionPeriod,
		activity.OwnerPrincipalID,
		activity.RequiredAuthorityPrincipalID,
		activity.ProgramID,
	}
	for _, value := range values {
		if value != "" && strings.TrimSpace(value) == "" {
			return ErrInvalid
		}
	}
	for _, value := range activity.DataCategories {
		if whitespaceOnly(value.Category) || whitespaceOnly(value.Sensitivity) {
			return ErrInvalid
		}
	}
	for _, value := range activity.Recipients {
		if whitespaceOnly(value.Recipient) || whitespaceOnly(value.RecipientKind) || whitespaceOnly(value.CountryCode) || whitespaceOnly(string(value.TransferBasis)) {
			return ErrInvalid
		}
	}
	for _, value := range activity.Systems {
		if whitespaceOnly(value.SystemName) || whitespaceOnly(value.SystemKind) {
			return ErrInvalid
		}
	}
	for _, value := range activity.Reviews {
		if whitespaceOnly(value.ID) || whitespaceOnly(value.Outcome) || whitespaceOnly(value.ReviewerPrincipalID) {
			return ErrInvalid
		}
	}
	return nil
}

func whitespaceOnly(value string) bool {
	return value != "" && strings.TrimSpace(value) == ""
}

func validateActivity(activity ProcessingActivity) error {
	if activity.Version <= 0 || !validStatus(activity.Status) {
		return ErrInvalid
	}
	if activity.UpdatedAt.Before(activity.CreatedAt) {
		return ErrInvalid
	}
	if strings.TrimSpace(activity.TenantID) == "" ||
		strings.TrimSpace(activity.LegalEntityID) == "" ||
		strings.TrimSpace(activity.Code) == "" ||
		strings.TrimSpace(activity.Name) == "" ||
		strings.TrimSpace(activity.Controller) == "" {
		return ErrInvalid
	}
	if !boundedText(activity.ID, MaxActivityIdentifierLength, false) ||
		!boundedText(activity.Code, MaxActivityCodeLength, true) ||
		!boundedText(activity.Name, MaxActivityNameLength, true) ||
		!boundedText(activity.TenantID, MaxActivityIdentifierLength, true) ||
		!boundedText(activity.LegalEntityID, MaxActivityIdentifierLength, true) ||
		!boundedText(activity.Controller, MaxActivityNameLength, true) ||
		!boundedText(activity.Description, MaxActivityTextLength, false) ||
		!boundedText(activity.Purpose, MaxActivityTextLength, false) ||
		!boundedText(activity.LawfulBasis, MaxActivityTextLength, false) ||
		!boundedText(activity.Processor, MaxActivityNameLength, false) ||
		!boundedText(activity.DataSubjectCategories, MaxActivityTextLength, false) ||
		!boundedText(activity.PersonalDataCategories, MaxActivityTextLength, false) ||
		!boundedText(activity.SecurityMeasures, MaxActivityTextLength, false) ||
		!boundedText(activity.RetentionPeriod, MaxActivityTextLength, false) ||
		!boundedText(activity.OwnerPrincipalID, MaxActivityIdentifierLength, false) ||
		!boundedText(activity.RequiredAuthorityPrincipalID, MaxActivityIdentifierLength, false) ||
		!boundedText(activity.ProgramID, MaxActivityIdentifierLength, false) {
		return ErrInvalid
	}
	if activity.StartDate != nil && activity.EndDate != nil && activity.EndDate.Before(*activity.StartDate) {
		return ErrInvalid
	}
	if len(activity.DataCategories) > MaxActivityChildCollectionSize ||
		len(activity.Recipients) > MaxActivityChildCollectionSize ||
		len(activity.Systems) > MaxActivityChildCollectionSize ||
		len(activity.Reviews) > MaxActivityChildCollectionSize {
		return ErrInvalid
	}

	seenCategories := make(map[string]struct{}, len(activity.DataCategories))
	for _, value := range activity.DataCategories {
		if !boundedText(value.Category, MaxActivityIdentifierLength, true) ||
			!validSensitivity(value.Sensitivity) {
			return ErrInvalid
		}
		if _, exists := seenCategories[value.Category]; exists {
			return ErrInvalid
		}
		seenCategories[value.Category] = struct{}{}
	}

	seenRecipients := make(map[string]struct{}, len(activity.Recipients))
	for _, value := range activity.Recipients {
		if !boundedText(value.Recipient, MaxActivityNameLength, true) ||
			!validRecipientKind(value.RecipientKind) ||
			!value.TransferBasis.Valid() ||
			!validCountryCode(value.CountryCode) ||
			(value.IsCrossBorder && (value.CountryCode == "" || !value.TransferBasis.Valid() || value.TransferBasis == TransferBasisNotApplicable)) ||
			(!value.IsCrossBorder && (value.CountryCode != "" || value.TransferBasis != TransferBasisNotApplicable)) {
			return ErrInvalid
		}
		if _, exists := seenRecipients[value.Recipient]; exists {
			return ErrInvalid
		}
		seenRecipients[value.Recipient] = struct{}{}
	}

	seenSystems := make(map[string]struct{}, len(activity.Systems))
	for _, value := range activity.Systems {
		if !boundedText(value.SystemName, MaxActivityNameLength, true) || !validSystemKind(value.SystemKind) {
			return ErrInvalid
		}
		if _, exists := seenSystems[value.SystemName]; exists {
			return ErrInvalid
		}
		seenSystems[value.SystemName] = struct{}{}
	}

	seenReviews := make(map[string]struct{}, len(activity.Reviews))
	for _, value := range activity.Reviews {
		if !boundedText(value.ID, MaxActivityIdentifierLength, true) ||
			!validReviewOutcome(value.Outcome) ||
			!boundedText(value.ReviewerPrincipalID, MaxActivityIdentifierLength, false) {
			return ErrInvalid
		}
		if _, exists := seenReviews[value.ID]; exists {
			return ErrInvalid
		}
		seenReviews[value.ID] = struct{}{}
	}
	return nil
}

// ValidateTransitionForWrite is the single lifecycle and closure gate used by
// both the service and repository command paths. An unchanged status is valid
// for an update event, while a transitioned event must change status and a
// changed status must follow validTransition.
func ValidateTransitionForWrite(current, next ProcessingActivity, eventType string) error {
	if !validStatus(current.Status) || !validStatus(next.Status) {
		return ErrInvalid
	}
	// Creation is a separate command and can never be replayed through the
	// update/event path. An update preserves status; a transition changes it.
	if eventType == EventActivityCreated {
		return ErrInvalid
	}
	if eventType == EventActivityUpdated {
		if current.Status != next.Status {
			return ErrInvalid
		}
	} else if eventType == EventActivityTransitioned {
		if current.Status == next.Status || !validTransition(current.Status, next.Status) {
			return ErrInvalid
		}
	} else {
		return ErrInvalid
	}
	if next.Status == StatusClosed && len(closureBlockers(next)) > 0 {
		return ErrClosureBlocked
	}
	return nil
}

func boundedText(value string, maximum int, required bool) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return !required
	}
	return utf8.RuneCountInString(trimmed) <= maximum
}

func validRecipientKind(value string) bool {
	switch value {
	case "INTERNAL", "EXTERNAL", "AUTHORITY":
		return true
	default:
		return false
	}
}

func validSystemKind(value string) bool {
	switch value {
	case "APPLICATION", "DATABASE", "FILE", "MANUAL", "THIRD_PARTY":
		return true
	default:
		return false
	}
}

func validSensitivity(value string) bool {
	switch value {
	case "UNCLASSIFIED", "DIRECT_PERSONAL", "INDIRECT_PERSONAL", "SENSITIVE_BY_NATURE", "SENSITIVE_BY_LAW":
		return true
	default:
		return false
	}
}

func validReviewOutcome(value string) bool {
	return value == "" || value == "CONFIRMED" || value == "REVISED" || value == "WITHDRAWN"
}

func validCountryCode(value string) bool {
	return value == "" || (len(value) == 2 && value[0] >= 'A' && value[0] <= 'Z' && value[1] >= 'A' && value[1] <= 'Z')
}

func validActivityEventType(value string) bool {
	return value == EventActivityCreated || value == EventActivityUpdated || value == EventActivityTransitioned
}

func validTransition(from, to Status) bool {
	switch from {
	case StatusNew:
		return to == StatusOpen || to == StatusClosed
	case StatusOpen:
		return to == StatusNew || to == StatusClosed
	default:
		return false
	}
}

// ClosureBlockersForTest exposes the register's closure decision to other
// packages so the report's exception predicate can be proven equal to it. A
// second implementation of this rule is how a register and its report would
// start disagreeing.
func ClosureBlockersForTest(activity ProcessingActivity) []string { return closureBlockers(activity) }

// closureBlockers names each fact an operator must supply before closure. A
// completed review means at least one review has a non-nil CompletedAt and an
// Outcome of CONFIRMED or REVISED; a nil completion, empty outcome, or
// WITHDRAWN outcome does not satisfy the completed-review rule.
func closureBlockers(activity ProcessingActivity) []string {
	blockers := make([]string, 0, 4)
	if strings.TrimSpace(activity.LawfulBasis) == "" {
		blockers = append(blockers, "lawful basis")
	}
	if strings.TrimSpace(activity.OwnerPrincipalID) == "" {
		blockers = append(blockers, "named owner")
	}
	if strings.TrimSpace(activity.DataSubjectCategories) == "" {
		blockers = append(blockers, "data subject category")
	}
	if !hasCompletedReview(activity.Reviews) {
		blockers = append(blockers, "completed review")
	}
	return blockers
}

func hasCompletedReview(reviews []Review) bool {
	for _, review := range reviews {
		if review.CompletedAt != nil && (review.Outcome == "CONFIRMED" || review.Outcome == "REVISED") {
			return true
		}
	}
	return false
}

func validStatus(status Status) bool {
	return status == StatusNew || status == StatusOpen || status == StatusClosed
}

func normalizedTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := value.UTC()
	return &result
}

func normalizeProcessingActivity(activity ProcessingActivity) ProcessingActivity {
	activity.TenantID = strings.TrimSpace(activity.TenantID)
	activity.LegalEntityID = strings.TrimSpace(activity.LegalEntityID)
	activity.Code = strings.TrimSpace(activity.Code)
	activity.Name = strings.TrimSpace(activity.Name)
	activity.Description = strings.TrimSpace(activity.Description)
	activity.Status = Status(strings.ToUpper(strings.TrimSpace(string(activity.Status))))
	activity.Purpose = strings.TrimSpace(activity.Purpose)
	activity.LawfulBasis = strings.TrimSpace(activity.LawfulBasis)
	activity.Controller = strings.TrimSpace(activity.Controller)
	activity.Processor = strings.TrimSpace(activity.Processor)
	activity.DataSubjectCategories = strings.TrimSpace(activity.DataSubjectCategories)
	activity.PersonalDataCategories = strings.TrimSpace(activity.PersonalDataCategories)
	activity.SecurityMeasures = strings.TrimSpace(activity.SecurityMeasures)
	activity.RetentionPeriod = strings.TrimSpace(activity.RetentionPeriod)
	activity.OwnerPrincipalID = strings.TrimSpace(activity.OwnerPrincipalID)
	activity.RequiredAuthorityPrincipalID = strings.TrimSpace(activity.RequiredAuthorityPrincipalID)
	activity.ProgramID = strings.TrimSpace(activity.ProgramID)
	activity.StartDate = normalizedTime(activity.StartDate)
	activity.EndDate = normalizedTime(activity.EndDate)
	activity.NextReviewDate = normalizedTime(activity.NextReviewDate)
	if !activity.CreatedAt.IsZero() {
		activity.CreatedAt = activity.CreatedAt.UTC()
	}
	if !activity.UpdatedAt.IsZero() {
		activity.UpdatedAt = activity.UpdatedAt.UTC()
	}
	activity.DataCategories = cloneDataCategories(activity.DataCategories)
	activity.Recipients = cloneRecipients(activity.Recipients)
	activity.Systems = cloneSystems(activity.Systems)
	activity.Reviews = cloneReviews(activity.Reviews)
	return activity
}

func cloneProcessingActivity(activity ProcessingActivity) ProcessingActivity {
	return normalizeProcessingActivity(activity)
}

func cloneDataCategories(values []DataCategory) []DataCategory {
	if values == nil {
		return nil
	}
	result := make([]DataCategory, len(values))
	for index, value := range values {
		result[index] = DataCategory{
			Category:    strings.TrimSpace(value.Category),
			Sensitivity: strings.TrimSpace(value.Sensitivity),
		}
	}
	return result
}

func cloneRecipients(values []Recipient) []Recipient {
	if values == nil {
		return nil
	}
	result := make([]Recipient, len(values))
	for index, value := range values {
		result[index] = value
		result[index].Recipient = strings.TrimSpace(value.Recipient)
		result[index].RecipientKind = strings.TrimSpace(value.RecipientKind)
		result[index].CountryCode = strings.TrimSpace(value.CountryCode)
		result[index].TransferBasis = TransferBasis(strings.TrimSpace(string(value.TransferBasis)))
	}
	return result
}

func cloneSystems(values []System) []System {
	if values == nil {
		return nil
	}
	result := make([]System, len(values))
	for index, value := range values {
		result[index] = System{
			SystemName: strings.TrimSpace(value.SystemName),
			SystemKind: strings.TrimSpace(value.SystemKind),
		}
	}
	return result
}

func cloneReviews(values []Review) []Review {
	if values == nil {
		return nil
	}
	result := make([]Review, len(values))
	for index, value := range values {
		value.ID = strings.TrimSpace(value.ID)
		value.Outcome = strings.TrimSpace(value.Outcome)
		value.ReviewerPrincipalID = strings.TrimSpace(value.ReviewerPrincipalID)
		value.CompletedAt = normalizedTime(value.CompletedAt)
		if !value.CreatedAt.IsZero() {
			value.CreatedAt = value.CreatedAt.UTC()
		}
		if !value.DueDate.IsZero() {
			value.DueDate = value.DueDate.UTC()
		}
		result[index] = value
	}
	return result
}
