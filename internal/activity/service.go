package activity

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
)

var (
	ErrInvalid  = errors.New("activity query is invalid")
	ErrNotFound = errors.New("activity event was not found")
)

const (
	defaultLimit = 50
	maxLimit     = 100
)

type Repository interface {
	List(context.Context, Query) (Page, error)
	Get(context.Context, string, string) (Event, error)
}

type Service struct {
	repository Repository
	now        func() time.Time
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, now: time.Now}
}

func (s *Service) List(ctx context.Context, query Query) (Page, error) {
	if s == nil || s.repository == nil {
		return Page{}, ErrInvalid
	}
	query = normalizeQuery(query)
	if query.TenantID == "" || (query.From != nil && query.To != nil && query.From.After(*query.To)) || !validActorKind(query.ActorKind) {
		return Page{}, ErrInvalid
	}
	page, err := s.repository.List(ctx, query)
	if err != nil {
		return Page{}, err
	}
	if page.Items == nil {
		page.Items = []Event{}
	}
	if page.AsOf.IsZero() {
		page.AsOf = s.now().UTC()
	}
	for index := range page.Items {
		decorate(&page.Items[index])
	}
	return page, nil
}

func (s *Service) Get(ctx context.Context, tenantID, eventID string) (Event, error) {
	if s == nil || s.repository == nil || strings.TrimSpace(tenantID) == "" || strings.TrimSpace(eventID) == "" {
		return Event{}, ErrInvalid
	}
	value, err := s.repository.Get(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(eventID))
	if err != nil {
		return Event{}, err
	}
	decorate(&value)
	return value, nil
}

func normalizeQuery(query Query) Query {
	query.TenantID = strings.TrimSpace(query.TenantID)
	query.Cursor = strings.TrimSpace(query.Cursor)
	query.Category = strings.ToUpper(strings.TrimSpace(query.Category))
	query.EventType = strings.TrimSpace(query.EventType)
	query.ObjectType = strings.ToUpper(strings.TrimSpace(query.ObjectType))
	query.ObjectID = strings.TrimSpace(query.ObjectID)
	query.ActorID = strings.TrimSpace(query.ActorID)
	query.ActorQuery = strings.TrimSpace(query.ActorQuery)
	query.ActorKind = strings.ToUpper(strings.TrimSpace(query.ActorKind))
	query.LegalEntityID = strings.TrimSpace(query.LegalEntityID)
	if query.Limit <= 0 {
		query.Limit = defaultLimit
	} else if query.Limit > maxLimit {
		query.Limit = maxLimit
	}
	if query.From != nil {
		value := query.From.UTC()
		query.From = &value
	}
	if query.To != nil {
		value := query.To.UTC()
		query.To = &value
	}
	return query
}

func validActorKind(value string) bool {
	switch value {
	case "", ActorInternalUser, ActorExternalParticipant, ActorService, ActorSystem, ActorUnknown:
		return true
	default:
		return false
	}
}

func decorate(value *Event) {
	if value == nil {
		return
	}
	value.ObjectType = strings.ToUpper(strings.TrimSpace(value.ObjectType))
	value.EventType = strings.TrimSpace(value.EventType)
	value.Category = categoryFor(value.ObjectType)
	value.Action = humanize(value.EventType)
	if value.Outcome == "" {
		value.Outcome = OutcomeSucceeded
	}
	if value.Source == "" {
		value.Source = "OUTBOX_EVENT"
	}
	value.ActorKind = actorKind(value.ActorKind, value.ObjectType)
}

func categoryFor(objectType string) string {
	objectType = strings.ToUpper(strings.TrimSpace(objectType))
	switch {
	case strings.HasPrefix(objectType, "THIRD_PARTY"), strings.HasPrefix(objectType, "VENDOR"):
		return CategoryVendor
	case strings.HasPrefix(objectType, "FORM"), strings.HasPrefix(objectType, "CAPTURE"), strings.HasPrefix(objectType, "EVIDENCE"):
		return CategoryFormsEvidence
	case strings.HasPrefix(objectType, "AI"):
		return CategoryAI
	case strings.Contains(objectType, "POLICY"), objectType == "DELEGATION", strings.HasPrefix(objectType, "SOURCE_"):
		return CategoryConfiguration
	case objectType == "PROGRAM", objectType == "MATTER", strings.HasPrefix(objectType, "WORKFLOW"), strings.HasPrefix(objectType, "DECISION"), strings.HasPrefix(objectType, "ACTION"):
		return CategoryGRCWork
	case strings.HasPrefix(objectType, "PROJECTION"), strings.HasPrefix(objectType, "BACKGROUND_JOB"), strings.HasPrefix(objectType, "RUNTIME"):
		return CategorySystem
	default:
		return CategoryOther
	}
}

func actorKind(principalKind, objectType string) string {
	switch strings.ToUpper(strings.TrimSpace(principalKind)) {
	case "PERSON", "TEAM", "QUEUE", "COMMITTEE":
		return ActorInternalUser
	case "EXTERNAL_PARTY":
		return ActorExternalParticipant
	case "SERVICE":
		return ActorService
	}
	if categoryFor(objectType) == CategorySystem {
		return ActorSystem
	}
	return ActorUnknown
}

func humanize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Activity recorded"
	}
	var out []rune
	var previous rune
	for index, current := range []rune(value) {
		if current == '_' || current == '-' || current == '.' {
			if len(out) > 0 && out[len(out)-1] != ' ' {
				out = append(out, ' ')
			}
			previous = current
			continue
		}
		if index > 0 && unicode.IsUpper(current) && unicode.IsLower(previous) {
			out = append(out, ' ')
		}
		out = append(out, unicode.ToLower(current))
		previous = current
	}
	text := strings.Join(strings.Fields(string(out)), " ")
	if text == "" {
		return "Activity recorded"
	}
	runes := []rune(text)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
