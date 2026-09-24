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

	if err := validateActivityWhitespace(activity); err != nil {
		return ProcessingActivity{}, err
	}
	activity = normalizeProcessingActivity(activity)
	if activity.Status != StatusNew || activity.Version != 1 {
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
	activityKey := processingActivityKey(activity.TenantID, activity.LegalEntityID, activity.ID)
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

func (r *MemoryRepository) GetActivity(ctx context.Context, scope ActivityScope, activityID string) (ProcessingActivity, error) {
	if err := ropaContextError(ctx); err != nil {
		return ProcessingActivity{}, err
	}
	scope, err := normalizeActivityScope(scope)
	activityID = strings.TrimSpace(activityID)
	if err != nil || activityID == "" {
		return ProcessingActivity{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	activity, ok := r.activities[processingActivityKey(scope.TenantID, scope.LegalEntityID, activityID)]
	if !ok {
		return ProcessingActivity{}, ErrNotFound
	}
	return cloneProcessingActivity(activity), nil
}

func (r *MemoryRepository) ActivityByCode(ctx context.Context, scope ActivityScope, code string) (ProcessingActivity, error) {
	if err := ropaContextError(ctx); err != nil {
		return ProcessingActivity{}, err
	}
	scope, err := normalizeActivityScope(scope)
	code = strings.TrimSpace(code)
	if err != nil || code == "" {
		return ProcessingActivity{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	activityID, ok := r.byCode[activityCodeKey(scope.TenantID, scope.LegalEntityID, code)]
	if !ok {
		return ProcessingActivity{}, ErrNotFound
	}
	activity, ok := r.activities[processingActivityKey(scope.TenantID, scope.LegalEntityID, activityID)]
	if !ok {
		return ProcessingActivity{}, ErrNotFound
	}
	return cloneProcessingActivity(activity), nil
}

func (r *MemoryRepository) ApplyActivityEvent(ctx context.Context, scope ActivityScope, activityID string, expectedVersion int64, event Event) (int64, error) {
	if err := ropaContextError(ctx); err != nil {
		return 0, err
	}
	scope, err := normalizeActivityScope(scope)
	activityID = strings.TrimSpace(activityID)
	if err != nil || activityID == "" || expectedVersion <= 0 {
		return 0, ErrInvalid
	}
	if strings.TrimSpace(event.ID) == "" {
		generated, generationErr := newEventID()
		if generationErr != nil {
			return 0, generationErr
		}
		event.ID = generated
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	key := processingActivityKey(scope.TenantID, scope.LegalEntityID, activityID)
	current, ok := r.activities[key]
	if !ok {
		return 0, ErrNotFound
	}
	next, event, err := prepareActivityEvent(current, expectedVersion, event)
	if err != nil {
		return 0, err
	}

	oldCodeKey := activityCodeKey(current.TenantID, current.LegalEntityID, current.Code)
	newCodeKey := activityCodeKey(next.TenantID, next.LegalEntityID, next.Code)
	if oldCodeKey != newCodeKey {
		if existingID, exists := r.byCode[newCodeKey]; exists && existingID != current.ID {
			return 0, ErrDuplicate
		}
	}

	if oldCodeKey != newCodeKey {
		delete(r.byCode, oldCodeKey)
		r.byCode[newCodeKey] = current.ID
	}
	r.activities[key] = cloneProcessingActivity(next)
	r.events[key] = append(r.events[key], cloneEvent(event))
	r.revisions[key] = append(r.revisions[key], cloneProcessingActivity(next))
	return next.Version, nil
}

func (r *MemoryRepository) ActivityEvents(ctx context.Context, scope ActivityScope, activityID string, afterVersion int64, limit int) ([]Event, bool, error) {
	if err := ropaContextError(ctx); err != nil {
		return nil, false, err
	}
	scope, err := normalizeActivityScope(scope)
	activityID = strings.TrimSpace(activityID)
	if err != nil || activityID == "" || afterVersion < 0 || limit < 0 {
		return nil, false, ErrInvalid
	}
	if limit == 0 {
		limit = DefaultActivityEventPageSize
	}
	if limit > MaxActivityEventPageSize {
		limit = MaxActivityEventPageSize
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := processingActivityKey(scope.TenantID, scope.LegalEntityID, activityID)
	events, ok := r.events[key]
	if !ok {
		return nil, false, ErrNotFound
	}
	filtered := make([]Event, 0, limit+1)
	for _, event := range events {
		if event.AggregateVersion <= afterVersion {
			continue
		}
		filtered = append(filtered, cloneEvent(event))
		if len(filtered) == limit+1 {
			break
		}
	}
	hasMore := len(filtered) > limit
	if hasMore {
		filtered = filtered[:limit]
	}
	return filtered, hasMore, nil
}

func (r *MemoryRepository) ListActivities(ctx context.Context, scope ActivityScope, filter ListActivitiesFilter) (ActivityPage, error) {
	if err := ropaContextError(ctx); err != nil {
		return ActivityPage{}, err
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

	cursor, err := decodeActivityCursor(filter.Cursor)
	if err != nil {
		return ActivityPage{}, ErrInvalid
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	rows := make([]ProcessingActivity, 0, len(r.activities))
	for _, value := range r.activities {
		if value.TenantID != scope.TenantID || value.LegalEntityID != scope.LegalEntityID {
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
	return validateCreatedActivityEvent(event, activity)
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
	position.ID = strings.TrimSpace(position.ID)
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

func processingActivityKey(tenantID, legalEntityID, activityID string) string {
	return tenantID + "\x00" + legalEntityID + "\x00" + activityID
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
	key := summaryKey(summary.TenantID, summary.LegalEntityID)
	if existing, ok := r.summaries[key]; ok && existing.GeneratedAt.After(summary.GeneratedAt) {
		// A run that started earlier can finish later. The in-memory half of
		// the projection guarantee is a monotonic conditional write: a strictly
		// older result is superseded and cannot replace newer state. The
		// production PostgreSQL writer must additionally lease each scope and
		// use a conditional ON CONFLICT ... WHERE
		// excluded.generated_at >= existing.generated_at write.
		return nil
	}
	r.summaries[key] = cloneSummary(summary)
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
