package evidence

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type completedResponseStore interface {
	ListCompletedResponses(context.Context, CompletedResponseQuery) (CompletedResponsePage, error)
	GetCompletedResponse(context.Context, string, string, string, string) (CompletedResponseSummary, ResponseRevision, error)
}

type ResponseSort string

const (
	ResponseSortConcern ResponseSort = "CONCERN_DESC"
	ResponseSortNewest  ResponseSort = "COMPLETED_DESC"
	ResponseSortRawAsc  ResponseSort = "RAW_ASC"
	ResponseSortRawDesc ResponseSort = "RAW_DESC"
)

type CompletedResponseQuery struct {
	TenantID            string
	LegalEntityID       string
	PrincipalID         string
	FormTemplateID      string
	FormTemplateVersion int64
	SubjectType         string
	SubjectID           string
	Modes               []formcontract.ScoringMode
	Bands               []formcontract.ConcernBand
	States              []ResponseScoreState
	RawMinimum          *float64
	RawMaximum          *float64
	AdverseMinimum      *float64
	AdverseMaximum      *float64
	CompletedFrom       *time.Time
	CompletedUntil      *time.Time
	CurrentOnly         bool
	Sort                ResponseSort
	Cursor              string
	Limit               int
}

type CompletedResponseSummary struct {
	ID                  string                `json:"id"`
	TenantID            string                `json:"tenant_id"`
	LegalEntityID       string                `json:"legal_entity_id"`
	DistributionID      string                `json:"distribution_id"`
	FormTemplateID      string                `json:"form_template_id"`
	FormTemplateVersion int64                 `json:"form_template_version"`
	Title               string                `json:"title"`
	SubjectType         string                `json:"subject_type"`
	SubjectID           string                `json:"subject_id"`
	Revision            int64                 `json:"revision"`
	Current             bool                  `json:"current"`
	State               ResponseRevisionState `json:"state"`
	Score               *ResponseScoreResult  `json:"score"`
	CompletedAt         time.Time             `json:"completed_at"`
}

type CompletedResponsePage struct {
	Items      []CompletedResponseSummary `json:"items"`
	NextCursor string                     `json:"next_cursor,omitempty"`
}

type completedResponseCursor struct {
	Sort        ResponseSort `json:"sort"`
	Score       *float64     `json:"score,omitempty"`
	CompletedAt time.Time    `json:"completed_at"`
	ID          string       `json:"id"`
}

func (service *DistributionService) ListCompletedResponses(ctx context.Context, query CompletedResponseQuery) (CompletedResponsePage, error) {
	if service == nil || service.store == nil {
		return CompletedResponsePage{}, ErrDistributionInvalid
	}
	probe := query
	if _, err := normalizeCompletedResponseQuery(&probe); err != nil {
		return CompletedResponsePage{}, fmt.Errorf("%w: %v", ErrDistributionInvalid, err)
	}
	reader, ok := service.store.(completedResponseStore)
	if !ok {
		return CompletedResponsePage{}, ErrDistributionInvalid
	}
	page, err := reader.ListCompletedResponses(ctx, query)
	if err != nil {
		return CompletedResponsePage{}, err
	}
	return page, nil
}

func (service *DistributionService) GetCompletedResponse(ctx context.Context, tenantID, legalEntityID, principalID, revisionID string) (CompletedResponseSummary, ResponseRevision, error) {
	if service == nil || service.store == nil || strings.TrimSpace(tenantID) == "" || strings.TrimSpace(legalEntityID) == "" || strings.TrimSpace(principalID) == "" || strings.TrimSpace(revisionID) == "" {
		return CompletedResponseSummary{}, ResponseRevision{}, ErrDistributionInvalid
	}
	reader, ok := service.store.(completedResponseStore)
	if !ok {
		return CompletedResponseSummary{}, ResponseRevision{}, ErrDistributionInvalid
	}
	return reader.GetCompletedResponse(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID), strings.TrimSpace(principalID), strings.TrimSpace(revisionID))
}

func normalizeCompletedResponseQuery(query *CompletedResponseQuery) (completedResponseCursor, error) {
	if query == nil || strings.TrimSpace(query.TenantID) == "" || strings.TrimSpace(query.LegalEntityID) == "" || strings.TrimSpace(query.PrincipalID) == "" || query.Limit < 1 || query.Limit > 100 || query.FormTemplateVersion < 0 {
		return completedResponseCursor{}, fmt.Errorf("completed response query is invalid")
	}
	if query.FormTemplateVersion > 0 && strings.TrimSpace(query.FormTemplateID) == "" {
		return completedResponseCursor{}, fmt.Errorf("form template version requires a form template")
	}
	if query.Sort == "" {
		query.Sort = ResponseSortConcern
	}
	switch query.Sort {
	case ResponseSortConcern, ResponseSortNewest, ResponseSortRawAsc, ResponseSortRawDesc:
	default:
		return completedResponseCursor{}, fmt.Errorf("completed response sort is invalid")
	}
	if !validScoreRange(query.RawMinimum, query.RawMaximum) || !validScoreRange(query.AdverseMinimum, query.AdverseMaximum) {
		return completedResponseCursor{}, fmt.Errorf("completed response score range is invalid")
	}
	if query.CompletedFrom != nil && query.CompletedUntil != nil && query.CompletedFrom.After(*query.CompletedUntil) {
		return completedResponseCursor{}, fmt.Errorf("completed response date range is invalid")
	}
	for _, mode := range query.Modes {
		if mode != formcontract.ScoringRisk && mode != formcontract.ScoringCompliance && mode != formcontract.ScoringNone {
			return completedResponseCursor{}, fmt.Errorf("completed response score mode is invalid")
		}
	}
	for _, band := range query.Bands {
		if band != formcontract.ConcernLow && band != formcontract.ConcernModerate && band != formcontract.ConcernHigh && band != formcontract.ConcernCritical {
			return completedResponseCursor{}, fmt.Errorf("completed response concern band is invalid")
		}
	}
	for _, state := range query.States {
		if state != ResponseScoreNotConfigured && state != ResponseScoreFinal && state != ResponseScoreProvisional && state != ResponseScoreFailed {
			return completedResponseCursor{}, fmt.Errorf("completed response score state is invalid")
		}
	}
	if strings.TrimSpace(query.Cursor) == "" {
		return completedResponseCursor{}, nil
	}
	cursor, err := decodeCompletedResponseCursor(query.Cursor)
	if err != nil || cursor.Sort != query.Sort || cursor.ID == "" || cursor.CompletedAt.IsZero() {
		return completedResponseCursor{}, fmt.Errorf("completed response cursor is invalid")
	}
	return cursor, nil
}

func validScoreRange(minimum, maximum *float64) bool {
	if minimum != nil && (*minimum < 0 || *minimum > 100) || maximum != nil && (*maximum < 0 || *maximum > 100) {
		return false
	}
	return minimum == nil || maximum == nil || *minimum <= *maximum
}

func encodeCompletedResponseCursor(value CompletedResponseSummary, sort ResponseSort) string {
	cursor := completedResponseCursor{Sort: sort, CompletedAt: value.CompletedAt.UTC(), ID: value.ID}
	if value.Score != nil {
		switch sort {
		case ResponseSortConcern:
			cursor.Score = value.Score.AdverseScore
		case ResponseSortRawAsc, ResponseSortRawDesc:
			cursor.Score = value.Score.RawScore
		}
	}
	encoded, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func decodeCompletedResponseCursor(value string) (completedResponseCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return completedResponseCursor{}, err
	}
	var cursor completedResponseCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return completedResponseCursor{}, err
	}
	return cursor, nil
}

func completedResponseSummary(distribution FormDistribution, revision ResponseRevision) CompletedResponseSummary {
	return CompletedResponseSummary{
		ID: revision.ID, TenantID: revision.TenantID, LegalEntityID: revision.LegalEntityID,
		DistributionID: revision.DistributionID, FormTemplateID: distribution.FormTemplateID,
		FormTemplateVersion: distribution.FormTemplateVersion, Title: distribution.Title,
		SubjectType: distribution.SubjectType, SubjectID: distribution.SubjectID, Revision: revision.Revision,
		Current: revision.Current, State: revision.State, Score: cloneResponseRevision(revision).Score, CompletedAt: revision.CreatedAt.UTC(),
	}
}
