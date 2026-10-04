package metricview

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
)

var (
	ErrDrillInvalid   = errors.New("metric drill query is invalid")
	ErrDrillNotFound  = errors.New("metric drill population is unavailable")
	ErrDrillMismatch  = errors.New("metric drill population does not match its observation")
)

type DrillQuery struct {
	TenantID      string
	LegalEntityID string
	SourceID      string
	MetricID      string
	Cursor        string
	Limit         int
}

type DrillPage struct {
	SourceID           string                       `json:"source_id"`
	MetricID           string                       `json:"metric_id"`
	DefinitionRevision string                       `json:"definition_revision"`
	GeneratedAt        time.Time                    `json:"generated_at"`
	Total              int                          `json:"total"`
	Items              []oversight.MetricMemberItem `json:"items"`
	NextCursor         string                       `json:"next_cursor,omitempty"`
}

type DrillReader interface {
	ListExactDrill(context.Context, DrillQuery) (DrillPage, error)
}

type drillCursor struct {
	MemberType string `json:"member_type"`
	MemberID   string `json:"member_id"`
}

func normalizeDrillQuery(query DrillQuery) (DrillQuery, drillCursor, error) {
	query.TenantID = strings.TrimSpace(query.TenantID)
	query.LegalEntityID = strings.TrimSpace(query.LegalEntityID)
	query.SourceID = strings.TrimSpace(query.SourceID)
	query.MetricID = strings.TrimSpace(query.MetricID)
	if query.TenantID == "" || query.LegalEntityID == "" || query.SourceID == "" || query.MetricID == "" {
		return DrillQuery{}, drillCursor{}, ErrDrillInvalid
	}
	definition, ok := HomeDefinition(query.MetricID)
	if !ok || definition.Drill.Consistency != DrillSnapshotExact {
		return DrillQuery{}, drillCursor{}, ErrDrillInvalid
	}
	if query.Limit <= 0 {
		query.Limit = 50
	}
	if query.Limit > 100 {
		return DrillQuery{}, drillCursor{}, ErrDrillInvalid
	}
	cursor, err := decodeDrillCursor(query.Cursor)
	if err != nil {
		return DrillQuery{}, drillCursor{}, err
	}
	return query, cursor, nil
}

func encodeDrillCursor(value drillCursor) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeDrillCursor(value string) (drillCursor, error) {
	if strings.TrimSpace(value) == "" {
		return drillCursor{}, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return drillCursor{}, ErrDrillInvalid
	}
	var cursor drillCursor
	if err := json.Unmarshal(payload, &cursor); err != nil ||
		(cursor.MemberType != "MATTER" && cursor.MemberType != "WORKFLOW_TASK") ||
		strings.TrimSpace(cursor.MemberID) == "" {
		return drillCursor{}, ErrDrillInvalid
	}
	return cursor, nil
}
