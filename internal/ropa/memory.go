package ropa

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu         sync.Mutex
	activities map[string]ProcessingActivity
	byCode     map[string]string
	events     map[string][]Event
	revisions  map[string][]ProcessingActivity
}

var _ Repository = (*MemoryRepository)(nil)
var _ ActivityLister = (*MemoryRepository)(nil)

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		activities: make(map[string]ProcessingActivity),
		byCode:     make(map[string]string),
		events:     make(map[string][]Event),
		revisions:  make(map[string][]ProcessingActivity),
	}
}

func (r *MemoryRepository) CreateActivity(ctx context.Context, activity ProcessingActivity, event Event) (ProcessingActivity, error) {
	if err := ropaContextError(ctx); err != nil {
		return ProcessingActivity{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	activity = normalizeProcessingActivity(activity)
	if activity.Status == "" {
		activity.Status = StatusNew
	}
	if activity.Version == 0 {
		activity.Version = 1
	}
	if !validStatus(activity.Status) {
		return ProcessingActivity{}, ErrInvalid
	}
	if err := validateActivity(activity); err != nil {
		return ProcessingActivity{}, err
	}

	codeKey := activityCodeKey(activity.TenantID, activity.LegalEntityID, activity.Code)
	if _, exists := r.byCode[codeKey]; exists {
		return ProcessingActivity{}, ErrDuplicate
	}
	if activity.ID == "" {
		generated, err := newActivityID()
		if err != nil {
			return ProcessingActivity{}, err
		}
		activity.ID = generated
	}
	activityKey := processingActivityKey(activity.TenantID, activity.ID)
	if _, exists := r.activities[activityKey]; exists {
		return ProcessingActivity{}, ErrDuplicate
	}

	event, err := normalizeCreatedEvent(event, activity)
	if err != nil {
		return ProcessingActivity{}, err
	}
	r.activities[activityKey] = cloneProcessingActivity(activity)
	r.byCode[codeKey] = activity.ID
	r.events[activityKey] = []Event{cloneEvent(event)}
	r.revisions[activityKey] = []ProcessingActivity{cloneProcessingActivity(activity)}
	return cloneProcessingActivity(activity), nil
}

func (r *MemoryRepository) GetActivity(ctx context.Context, tenantID, activityID string) (ProcessingActivity, error) {
	if err := ropaContextError(ctx); err != nil {
		return ProcessingActivity{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	activity, ok := r.activities[processingActivityKey(strings.TrimSpace(tenantID), strings.TrimSpace(activityID))]
	if !ok {
		return ProcessingActivity{}, ErrNotFound
	}
	return cloneProcessingActivity(activity), nil
}

func (r *MemoryRepository) ActivityByID(ctx context.Context, activityID string) (ProcessingActivity, error) {
	if err := ropaContextError(ctx); err != nil {
		return ProcessingActivity{}, err
	}
	activityID = strings.TrimSpace(activityID)
	if activityID == "" {
		return ProcessingActivity{}, ErrNotFound
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, activity := range r.activities {
		if activity.ID == activityID {
			return cloneProcessingActivity(activity), nil
		}
	}
	return ProcessingActivity{}, ErrNotFound
}

func (r *MemoryRepository) ActivityByCode(ctx context.Context, tenantID, legalEntityID, code string) (ProcessingActivity, error) {
	if err := ropaContextError(ctx); err != nil {
		return ProcessingActivity{}, err
	}
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	code = strings.TrimSpace(code)
	r.mu.Lock()
	defer r.mu.Unlock()
	activityID, ok := r.byCode[activityCodeKey(tenantID, legalEntityID, code)]
	if !ok {
		return ProcessingActivity{}, ErrNotFound
	}
	activity, ok := r.activities[processingActivityKey(tenantID, activityID)]
	if !ok {
		return ProcessingActivity{}, ErrNotFound
	}
	return cloneProcessingActivity(activity), nil
}

func (r *MemoryRepository) ApplyActivityEvent(ctx context.Context, tenantID, activityID string, expectedVersion int64, event Event) (int64, error) {
	if err := ropaContextError(ctx); err != nil {
		return 0, err
	}
	tenantID = strings.TrimSpace(tenantID)
	activityID = strings.TrimSpace(activityID)
	r.mu.Lock()
	defer r.mu.Unlock()

	key := processingActivityKey(tenantID, activityID)
	current, ok := r.activities[key]
	if !ok {
		return 0, ErrNotFound
	}
	if current.Version != expectedVersion {
		return 0, ErrVersionConflict
	}
	if event.AggregateVersion != 0 && event.AggregateVersion != expectedVersion+1 {
		return 0, ErrVersionConflict
	}
	if event.AggregateVersion == 0 {
		event.AggregateVersion = expectedVersion + 1
	}
	if event.TenantID != "" && event.TenantID != current.TenantID {
		return 0, ErrInvalid
	}
	if event.LegalEntityID != "" && event.LegalEntityID != current.LegalEntityID {
		return 0, ErrInvalid
	}
	if event.AggregateID != "" && event.AggregateID != current.ID {
		return 0, ErrInvalid
	}
	if event.AggregateType != "" && event.AggregateType != "PROCESSING_ACTIVITY" {
		return 0, ErrInvalid
	}
	if strings.TrimSpace(event.ID) == "" {
		generated, err := newEventID()
		if err != nil {
			return 0, err
		}
		event.ID = generated
	}

	decoded, err := decodeActivity(event.Payload)
	if err != nil {
		return 0, err
	}
	decoded = normalizeProcessingActivity(decoded)
	if !validStatus(decoded.Status) {
		return 0, ErrInvalid
	}
	if err := validateActivity(decoded); err != nil {
		return 0, err
	}
	if decoded.TenantID != "" && decoded.TenantID != current.TenantID {
		return 0, ErrInvalid
	}
	if decoded.LegalEntityID != "" && decoded.LegalEntityID != current.LegalEntityID {
		return 0, ErrInvalid
	}
	decoded.ID = current.ID
	decoded.TenantID = current.TenantID
	decoded.LegalEntityID = current.LegalEntityID
	decoded.Version = expectedVersion + 1
	decoded.CreatedAt = current.CreatedAt
	if !event.OccurredAt.IsZero() {
		decoded.UpdatedAt = event.OccurredAt.UTC()
	} else if decoded.UpdatedAt.IsZero() {
		decoded.UpdatedAt = current.UpdatedAt
	}

	oldCodeKey := activityCodeKey(current.TenantID, current.LegalEntityID, current.Code)
	newCodeKey := activityCodeKey(decoded.TenantID, decoded.LegalEntityID, decoded.Code)
	if oldCodeKey != newCodeKey {
		if existingID, exists := r.byCode[newCodeKey]; exists && existingID != current.ID {
			return 0, ErrDuplicate
		}
	}

	event.TenantID = current.TenantID
	event.LegalEntityID = current.LegalEntityID
	event.AggregateType = "PROCESSING_ACTIVITY"
	event.AggregateID = current.ID
	event.AggregateVersion = expectedVersion + 1
	event.ActorID = strings.TrimSpace(event.ActorID)
	if event.ActorID == "" {
		event.ActorType = "SERVICE"
	} else {
		event.ActorType = "USER"
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = decoded.UpdatedAt
	}
	event.Payload = mustMarshalActivity(decoded)

	if oldCodeKey != newCodeKey {
		delete(r.byCode, oldCodeKey)
		r.byCode[newCodeKey] = current.ID
	}
	r.activities[key] = cloneProcessingActivity(decoded)
	r.events[key] = append(r.events[key], cloneEvent(event))
	r.revisions[key] = append(r.revisions[key], cloneProcessingActivity(decoded))
	return expectedVersion + 1, nil
}

func (r *MemoryRepository) ActivityEvents(ctx context.Context, tenantID, activityID string) ([]Event, error) {
	if err := ropaContextError(ctx); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := processingActivityKey(strings.TrimSpace(tenantID), strings.TrimSpace(activityID))
	events, ok := r.events[key]
	if !ok {
		return nil, ErrNotFound
	}
	result := make([]Event, len(events))
	for index, event := range events {
		result[index] = cloneEvent(event)
	}
	return result, nil
}

func (r *MemoryRepository) ListActivities(ctx context.Context, filter ListActivitiesFilter) (ActivityPage, error) {
	if err := ropaContextError(ctx); err != nil {
		return ActivityPage{}, err
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

	cursor, err := decodeActivityCursor(filter.Cursor)
	if err != nil {
		return ActivityPage{}, ErrInvalid
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	rows := make([]ProcessingActivity, 0, len(r.activities))
	for _, value := range r.activities {
		if value.TenantID != filter.TenantID || value.LegalEntityID != filter.LegalEntityID {
			continue
		}
		if !filter.IncludeRetired && (Aggregate{ProcessingActivity: value}).IsRetired() {
			continue
		}
		if filter.Status != "" && value.Status != filter.Status {
			continue
		}
		if filter.LawfulBasis != "" && value.LawfulBasis != filter.LawfulBasis {
			continue
		}
		if filter.OwnerPrincipalID != "" && value.OwnerPrincipalID != filter.OwnerPrincipalID {
			continue
		}
		if filter.Search != "" && !activityMatchesSearch(value, filter.Search) {
			continue
		}
		rows = append(rows, cloneProcessingActivity(value))
	}
	sort.Slice(rows, func(leftIndex, rightIndex int) bool {
		return activitySortBefore(rows[leftIndex], rows[rightIndex])
	})
	if filter.Cursor != "" {
		filtered := rows[:0]
		for _, value := range rows {
			if activityAfterCursor(value, cursor) {
				filtered = append(filtered, value)
			}
		}
		rows = filtered
	}

	page := ActivityPage{}
	if len(rows) > filter.Limit {
		page.Rows = cloneProcessingActivities(rows[:filter.Limit])
		page.NextCursor, err = encodeActivityCursor(rows[filter.Limit-1])
		if err != nil {
			return ActivityPage{}, err
		}
		page.HasMore = true
		return page, nil
	}
	page.Rows = cloneProcessingActivities(rows)
	return page, nil
}

func normalizeCreatedEvent(event Event, activity ProcessingActivity) (Event, error) {
	if strings.TrimSpace(event.ID) == "" {
		generated, err := newEventID()
		if err != nil {
			return Event{}, err
		}
		event.ID = generated
	}
	if event.TenantID != "" && event.TenantID != activity.TenantID {
		return Event{}, ErrInvalid
	}
	if event.LegalEntityID != "" && event.LegalEntityID != activity.LegalEntityID {
		return Event{}, ErrInvalid
	}
	if event.AggregateID != "" && event.AggregateID != activity.ID {
		return Event{}, ErrInvalid
	}
	if event.AggregateType != "" && event.AggregateType != "PROCESSING_ACTIVITY" {
		return Event{}, ErrInvalid
	}
	if event.AggregateVersion != 0 && event.AggregateVersion != activity.Version {
		return Event{}, ErrVersionConflict
	}
	event.TenantID = activity.TenantID
	event.LegalEntityID = activity.LegalEntityID
	event.AggregateType = "PROCESSING_ACTIVITY"
	event.AggregateID = activity.ID
	event.AggregateVersion = activity.Version
	event.ActorID = strings.TrimSpace(event.ActorID)
	if event.ActorID == "" {
		event.ActorType = "SERVICE"
	} else {
		event.ActorType = "USER"
	}
	// The event payload is the complete aggregate snapshot, even when a
	// repository caller supplied a placeholder payload.
	event.Payload = mustMarshalActivity(activity)
	if event.OccurredAt.IsZero() {
		event.OccurredAt = activity.UpdatedAt
	}
	return event, nil
}

func decodeActivity(payload []byte) (ProcessingActivity, error) {
	if len(strings.TrimSpace(string(payload))) == 0 {
		return ProcessingActivity{}, ErrInvalid
	}
	var activity ProcessingActivity
	if err := json.Unmarshal(payload, &activity); err != nil {
		return ProcessingActivity{}, err
	}
	return activity, nil
}

func mustMarshalActivity(activity ProcessingActivity) json.RawMessage {
	payload, _ := json.Marshal(activity)
	return payload
}

func cloneEvent(event Event) Event {
	event.Payload = append(json.RawMessage(nil), event.Payload...)
	return event
}

func cloneProcessingActivities(values []ProcessingActivity) []ProcessingActivity {
	if values == nil {
		return nil
	}
	result := make([]ProcessingActivity, len(values))
	for index, value := range values {
		result[index] = cloneProcessingActivity(value)
	}
	return result
}

func activityMatchesSearch(activity ProcessingActivity, search string) bool {
	needle := strings.ToLower(strings.TrimSpace(search))
	if needle == "" {
		return true
	}
	values := []string{
		activity.ID,
		activity.Code,
		activity.Name,
		activity.Description,
		activity.Purpose,
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
		if strings.Contains(strings.ToLower(value), needle) {
			return true
		}
	}
	for _, value := range activity.DataCategories {
		if strings.Contains(strings.ToLower(value.Category+" "+value.Sensitivity), needle) {
			return true
		}
	}
	for _, value := range activity.Recipients {
		if strings.Contains(strings.ToLower(value.Recipient+" "+value.RecipientKind+" "+value.CountryCode+" "+string(value.TransferBasis)), needle) {
			return true
		}
	}
	for _, value := range activity.Systems {
		if strings.Contains(strings.ToLower(value.SystemName+" "+value.SystemKind), needle) {
			return true
		}
	}
	return false
}

func activityStatusRank(status Status) int {
	switch status {
	case StatusNew:
		return 1
	case StatusOpen:
		return 2
	case StatusClosed:
		return 3
	default:
		return 4
	}
}

func activityReviewDate(activity ProcessingActivity) time.Time {
	if activity.NextReviewDate == nil {
		return time.Time{}
	}
	return activity.NextReviewDate.UTC()
}

func activitySortBefore(left, right ProcessingActivity) bool {
	leftRank, rightRank := activityStatusRank(left.Status), activityStatusRank(right.Status)
	if leftRank != rightRank {
		return leftRank < rightRank
	}
	leftDate, rightDate := activityReviewDate(left), activityReviewDate(right)
	if !leftDate.Equal(rightDate) {
		return leftDate.Before(rightDate)
	}
	return left.ID < right.ID
}

type activityCursor struct {
	Status         string    `json:"s"`
	NextReviewDate time.Time `json:"r"`
	ID             string    `json:"i"`
}

func encodeActivityCursor(activity ProcessingActivity) (string, error) {
	position := activityCursor{Status: string(activity.Status), NextReviewDate: activityReviewDate(activity), ID: activity.ID}
	payload, err := json.Marshal(position)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeActivityCursor(value string) (activityCursor, error) {
	if strings.TrimSpace(value) == "" {
		return activityCursor{}, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return activityCursor{}, fmt.Errorf("invalid cursor")
	}
	var position activityCursor
	if err := json.Unmarshal(payload, &position); err != nil {
		return activityCursor{}, fmt.Errorf("invalid cursor")
	}
	if position.ID == "" || !validStatus(Status(strings.ToUpper(strings.TrimSpace(position.Status)))) {
		return activityCursor{}, fmt.Errorf("invalid cursor")
	}
	position.Status = strings.ToUpper(strings.TrimSpace(position.Status))
	position.NextReviewDate = position.NextReviewDate.UTC()
	return position, nil
}

func activityAfterCursor(activity ProcessingActivity, cursor activityCursor) bool {
	leftRank := activityStatusRank(activity.Status)
	rightRank := activityStatusRank(Status(cursor.Status))
	if leftRank != rightRank {
		return leftRank > rightRank
	}
	leftDate, rightDate := activityReviewDate(activity), cursor.NextReviewDate.UTC()
	if !leftDate.Equal(rightDate) {
		return leftDate.After(rightDate)
	}
	return activity.ID > cursor.ID
}

func processingActivityKey(tenantID, activityID string) string {
	return tenantID + "\x00" + activityID
}

func activityCodeKey(tenantID, legalEntityID, code string) string {
	return tenantID + "\x00" + legalEntityID + "\x00" + code
}

func ropaContextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

type MemorySummaryRepository struct {
	mu        sync.Mutex
	summaries map[string]RegisterSummary
}

var _ SummaryRepository = (*MemorySummaryRepository)(nil)

func NewMemorySummaryRepository() *MemorySummaryRepository {
	return &MemorySummaryRepository{summaries: make(map[string]RegisterSummary)}
}

func (r *MemorySummaryRepository) LatestSummary(ctx context.Context, tenantID, legalEntityID string) (RegisterSummary, error) {
	if err := ropaContextError(ctx); err != nil {
		return RegisterSummary{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	summary, ok := r.summaries[summaryKey(tenantID, legalEntityID)]
	if !ok {
		return RegisterSummary{}, ErrNotFound
	}
	return cloneSummary(summary), nil
}

func (r *MemorySummaryRepository) ReplaceSummary(ctx context.Context, summary RegisterSummary) error {
	if err := ropaContextError(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(summary.TenantID) == "" || strings.TrimSpace(summary.LegalEntityID) == "" {
		return ErrInvalid
	}
	summary.TenantID = strings.TrimSpace(summary.TenantID)
	summary.LegalEntityID = strings.TrimSpace(summary.LegalEntityID)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.summaries[summaryKey(summary.TenantID, summary.LegalEntityID)] = cloneSummary(summary)
	return nil
}

func summaryKey(tenantID, legalEntityID string) string {
	return strings.TrimSpace(tenantID) + "\x00" + strings.TrimSpace(legalEntityID)
}

func cloneSummary(summary RegisterSummary) RegisterSummary {
	result := summary
	if summary.Coverage.Excluded != nil {
		value := *summary.Coverage.Excluded
		result.Coverage.Excluded = &value
	}
	if summary.Coverage.Unknown != nil {
		value := *summary.Coverage.Unknown
		result.Coverage.Unknown = &value
	}
	return result
}
