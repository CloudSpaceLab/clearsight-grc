package ropa

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
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
	LegalEntityID                string         `json:"legal_entity_id,omitempty"`
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
	LegalEntityID   string     `json:"legal_entity_id,omitempty"`
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
	activity := normalizeProcessingActivity(ProcessingActivity{
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
	})
	if err := validateActivity(activity); err != nil {
		return ProcessingActivity{}, err
	}

	if existing, err := s.repository.ActivityByCode(ctx, activity.TenantID, activity.LegalEntityID, activity.Code); err == nil {
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
	tenantID := strings.TrimSpace(input.TenantID)
	if tenantID == "" || activityID == "" {
		return ProcessingActivity{}, ErrInvalid
	}

	current, err := s.GetActivity(ctx, tenantID, activityID)
	if err != nil {
		return ProcessingActivity{}, err
	}
	if entityID := strings.TrimSpace(input.LegalEntityID); entityID != "" && entityID != current.LegalEntityID {
		return ProcessingActivity{}, ErrNotFound
	}
	if input.ExpectedVersion <= 0 || input.ExpectedVersion != current.Version {
		return ProcessingActivity{}, ErrVersionConflict
	}

	updated := current
	applyUpdateActivityInput(&updated, input)
	updated.Version = current.Version + 1
	updated.UpdatedAt = s.now()
	updated = normalizeProcessingActivity(updated)
	if err := validateActivity(updated); err != nil {
		return ProcessingActivity{}, err
	}

	if existing, lookupErr := s.repository.ActivityByCode(ctx, updated.TenantID, updated.LegalEntityID, updated.Code); lookupErr == nil {
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
	version, err := s.repository.ApplyActivityEvent(ctx, updated.TenantID, updated.ID, input.ExpectedVersion, event)
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
	tenantID := strings.TrimSpace(input.TenantID)
	if tenantID == "" || activityID == "" {
		return ProcessingActivity{}, ErrInvalid
	}

	current, err := s.GetActivity(ctx, tenantID, activityID)
	if err != nil {
		return ProcessingActivity{}, err
	}
	if entityID := strings.TrimSpace(input.LegalEntityID); entityID != "" && entityID != current.LegalEntityID {
		return ProcessingActivity{}, ErrNotFound
	}
	if input.ExpectedVersion <= 0 || input.ExpectedVersion != current.Version {
		return ProcessingActivity{}, ErrVersionConflict
	}
	if !validTransition(current.Status, input.To) {
		return ProcessingActivity{}, ErrInvalid
	}

	updated := current
	updated.Status = input.To
	if input.To == StatusClosed && input.EndDate != nil {
		updated.EndDate = input.EndDate
	}
	updated.Version = current.Version + 1
	updated.UpdatedAt = s.now()
	updated = normalizeProcessingActivity(updated)
	if err := validateActivity(updated); err != nil {
		return ProcessingActivity{}, err
	}
	if updated.Status == StatusClosed {
		if blockers := closureBlockers(updated); len(blockers) > 0 {
			return ProcessingActivity{}, ErrClosureBlocked
		}
	}

	event, err := s.newActivityEvent(updated, EventActivityTransitioned, input.ActorID, updated.UpdatedAt)
	if err != nil {
		return ProcessingActivity{}, err
	}
	version, err := s.repository.ApplyActivityEvent(ctx, updated.TenantID, updated.ID, input.ExpectedVersion, event)
	if err != nil {
		return ProcessingActivity{}, err
	}
	updated.Version = version
	updated.UpdatedAt = event.OccurredAt
	return updated, nil
}

func (s *Service) GetActivity(ctx context.Context, tenantID, activityID string) (ProcessingActivity, error) {
	if s == nil || s.repository == nil {
		return ProcessingActivity{}, ErrNotFound
	}
	return s.repository.GetActivity(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(activityID))
}

func (s *Service) ListActivities(ctx context.Context, filter ListActivitiesFilter) (ActivityPage, error) {
	if s == nil || s.lister == nil {
		return ActivityPage{}, ErrNotFound
	}
	filter.TenantID = strings.TrimSpace(filter.TenantID)
	filter.LegalEntityID = strings.TrimSpace(filter.LegalEntityID)
	if filter.TenantID == "" || filter.LegalEntityID == "" {
		return ActivityPage{}, ErrInvalid
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	} else if filter.Limit > 200 {
		filter.Limit = 200
	}
	filter.Status = Status(strings.ToUpper(strings.TrimSpace(string(filter.Status))))
	filter.LawfulBasis = strings.TrimSpace(filter.LawfulBasis)
	filter.OwnerPrincipalID = strings.TrimSpace(filter.OwnerPrincipalID)
	filter.Search = strings.TrimSpace(filter.Search)
	filter.Cursor = strings.TrimSpace(filter.Cursor)
	return s.lister.ListActivities(ctx, filter)
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
	if summary.GeneratedAt.IsZero() || now.Sub(summary.GeneratedAt) > 15*time.Minute || summary.ProjectionVersion != ProjectionVersion {
		summary.Freshness = FreshnessStale
	}
	return summary, nil
}

func (s *Service) ClosureBlockers(ctx context.Context, activityID string) ([]string, error) {
	if s == nil || s.repository == nil || strings.TrimSpace(activityID) == "" {
		return nil, ErrInvalid
	}
	activityID = strings.TrimSpace(activityID)
	if reader, ok := s.repository.(interface {
		ActivityByID(context.Context, string) (ProcessingActivity, error)
	}); ok {
		activity, err := reader.ActivityByID(ctx, activityID)
		if err != nil {
			return nil, err
		}
		return closureBlockers(activity), nil
	}

	// Some read repositories support an unscoped exact-id lookup. It is a
	// fallback only; the normal command paths always use a verified tenant.
	activity, err := s.repository.GetActivity(ctx, "", activityID)
	if err != nil {
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
		activity.DataCategories = cloneDataCategories(input.DataCategories)
	}
	if input.Recipients != nil {
		activity.Recipients = cloneRecipients(input.Recipients)
	}
	if input.Systems != nil {
		activity.Systems = cloneSystems(input.Systems)
	}
	if input.Reviews != nil {
		activity.Reviews = cloneReviews(input.Reviews)
	}
}

func validateActivity(activity ProcessingActivity) error {
	if strings.TrimSpace(activity.TenantID) == "" ||
		strings.TrimSpace(activity.LegalEntityID) == "" ||
		strings.TrimSpace(activity.Code) == "" ||
		strings.TrimSpace(activity.Name) == "" ||
		strings.TrimSpace(activity.Controller) == "" {
		return ErrInvalid
	}
	if activity.StartDate != nil && activity.EndDate != nil && activity.EndDate.Before(*activity.StartDate) {
		return ErrInvalid
	}
	return nil
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

func closureBlockers(activity ProcessingActivity) []string {
	blockers := make([]string, 0, 3)
	if strings.TrimSpace(activity.LawfulBasis) == "" {
		blockers = append(blockers, "lawful basis")
	}
	if strings.TrimSpace(activity.OwnerPrincipalID) == "" {
		blockers = append(blockers, "named owner")
	}
	if strings.TrimSpace(activity.DataSubjectCategories) == "" {
		blockers = append(blockers, "data subject category")
	}
	return blockers
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
