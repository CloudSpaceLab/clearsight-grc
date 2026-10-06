package rcsa

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

type CycleSummary struct {
	Cycle        Cycle `json:"cycle"`
	RiskCount    int   `json:"risk_count"`
	ControlCount int   `json:"control_count"`
	CursorAfter  string `json:"-"`
}

type CyclePage struct {
	Items      []CycleSummary `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

type CycleFilter struct {
	Status Status
	Cursor string
	Limit  int
}

type CycleListRepository interface {
	ListCycles(context.Context, Scope, CycleFilter) (CyclePage, error)
}

type cycleCursor struct {
	UpdatedAt time.Time `json:"updated_at"`
	ID        string    `json:"id"`
}

func (s *Service) List(ctx context.Context, scope Scope, filter CycleFilter) (CyclePage, error) {
	if s == nil || s.repo == nil {
		return CyclePage{}, ErrInvalid
	}
	repository, ok := s.repo.(CycleListRepository)
	if !ok {
		return CyclePage{}, ErrInvalid
	}
	normalized, err := normalizeScope(scope)
	if err != nil {
		return CyclePage{}, err
	}
	filter.Status = Status(strings.ToUpper(strings.TrimSpace(string(filter.Status))))
	filter.Cursor = strings.TrimSpace(filter.Cursor)
	if filter.Status != "" && !validStatus(filter.Status) {
		return CyclePage{}, ErrInvalid
	}
	if filter.Limit <= 0 {
		filter.Limit = 25
	} else if filter.Limit > 100 {
		filter.Limit = 100
	}
	if _, err := decodeCycleCursor(filter.Cursor); err != nil {
		return CyclePage{}, err
	}
	page, err := repository.ListCycles(ctx, normalized, filter)
	if err != nil {
		return CyclePage{}, err
	}
	for index := range page.Items {
		page.Items[index].CursorAfter, err = encodeCycleCursor(page.Items[index])
		if err != nil {
			return CyclePage{}, err
		}
	}
	return page, nil
}

func validStatus(status Status) bool {
	switch status {
	case StatusDraft, StatusAssessmentOpen, StatusAwaitingChallenge, StatusCompleted, StatusCancelled:
		return true
	default:
		return false
	}
}

func encodeCycleCursor(value CycleSummary) (string, error) {
	payload, err := json.Marshal(cycleCursor{UpdatedAt: value.Cycle.UpdatedAt.UTC(), ID: value.Cycle.ID})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeCycleCursor(value string) (cycleCursor, error) {
	if value == "" {
		return cycleCursor{}, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return cycleCursor{}, ErrInvalid
	}
	var cursor cycleCursor
	if err := json.Unmarshal(payload, &cursor); err != nil || cursor.UpdatedAt.IsZero() || strings.TrimSpace(cursor.ID) == "" {
		return cycleCursor{}, ErrInvalid
	}
	cursor.UpdatedAt = cursor.UpdatedAt.UTC()
	cursor.ID = strings.TrimSpace(cursor.ID)
	return cursor, nil
}

func cycleAfterCursor(value CycleSummary, cursor cycleCursor) bool {
	if cursor.UpdatedAt.IsZero() {
		return true
	}
	if !value.Cycle.UpdatedAt.Equal(cursor.UpdatedAt) {
		return value.Cycle.UpdatedAt.Before(cursor.UpdatedAt)
	}
	return value.Cycle.ID < cursor.ID
}
