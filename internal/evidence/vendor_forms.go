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

type VendorFormsQuery struct {
	TenantID, LegalEntityID, PrincipalID string
	RelationshipIDs                      []string
	FormTemplateID, Filter, Cursor       string
	Limit                                int
}
type VendorMissingField struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}
type VendorFormAttention struct {
	FieldID string `json:"field_id,omitempty"`
	RuleID  string `json:"rule_id,omitempty"`
	Label   string `json:"label"`
	State   string `json:"state"`
	Source  string `json:"source"`
}
type VendorFormRow struct {
	reviewedFields      map[string]bool
	AttentionItems      []VendorFormAttention `json:"attention_items"`
	Outdated            *bool                 `json:"outdated"`
	ResponseCurrency    string                `json:"response_currency"`
	RequestID           string                `json:"request_id"`
	RelationshipID      string                `json:"relationship_id"`
	DistributionID      string                `json:"distribution_id,omitempty"`
	ResponseID          string                `json:"response_id,omitempty"`
	FormTemplateID      string                `json:"form_template_id"`
	FormTemplateVersion int64                 `json:"form_template_version"`
	Title               string                `json:"title"`
	Purpose             string                `json:"purpose,omitempty"`
	OriginType          string                `json:"origin_type,omitempty"`
	OriginID            string                `json:"origin_id,omitempty"`
	ResponseState       string                `json:"response_state"`
	RecipientHint       string                `json:"recipient_hint,omitempty"`
	DeliveryState       string                `json:"delivery_state,omitempty"`
	Deadline            time.Time             `json:"deadline"`
	UpdatedAt           time.Time             `json:"updated_at"`
	SubmittedAt         *time.Time            `json:"submitted_at,omitempty"`
	RequiredCount       *int                  `json:"required_count"`
	AnsweredRequired    *int                  `json:"answered_required"`
	HeldRequired        *int                  `json:"held_required"`
	MissingFields       []VendorMissingField  `json:"missing_fields"`
	Score               *ResponseScoreResult  `json:"score,omitempty"`
	AssessedScore       *ResponseScoreResult  `json:"assessed_score,omitempty"`
	AssessmentState     string                `json:"assessment_state,omitempty"`
	RequiredReviews     int                   `json:"required_reviews"`
	CompletedReviews    int                   `json:"completed_reviews"`
	Current             bool                  `json:"current"`
}
type VendorFormsPage struct {
	Items      []VendorFormRow `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
	ObservedAt time.Time       `json:"observed_at"`
}
type VendorFormSummary struct {
	FreshnessUnknownForms  int                      `json:"freshness_unknown_forms"`
	OutdatedForms          int                      `json:"outdated_forms"`
	PartiallyReplacedForms int                      `json:"partially_replaced_forms"`
	RelationshipID         string                   `json:"relationship_id"`
	OutstandingForms       int                      `json:"outstanding_forms"`
	OverdueForms           int                      `json:"overdue_forms"`
	SubmittedForms         int                      `json:"submitted_forms"`
	AwaitingReview         int                      `json:"awaiting_review"`
	UnassessedForms        int                      `json:"unassessed_forms"`
	AssessedForms          int                      `json:"assessed_forms"`
	HighestConcern         formcontract.ConcernBand `json:"highest_concern,omitempty"`
	ObservedAt             time.Time                `json:"observed_at"`
}
type vendorFormsStore interface {
	ListVendorForms(context.Context, VendorFormsQuery) (VendorFormsPage, error)
	VendorFormSummaries(context.Context, VendorFormsQuery) ([]VendorFormSummary, error)
}

func normalizeVendorFormsQuery(q *VendorFormsQuery) error {
	if q == nil || q.TenantID == "" || q.LegalEntityID == "" || q.PrincipalID == "" || len(q.RelationshipIDs) == 0 || len(q.RelationshipIDs) > 50 || q.Limit < 1 || q.Limit > 100 {
		return ErrDistributionInvalid
	}
	seen := map[string]bool{}
	for _, id := range q.RelationshipIDs {
		if strings.TrimSpace(id) == "" || seen[id] {
			return ErrDistributionInvalid
		}
		seen[id] = true
	}
	switch q.Filter {
	case "", "AWAITING_VENDOR", "AWAITING_REVIEW", "WITH_RISKS", "HIGH_RISK", "OVERDUE", "NOT_ASSESSED":
	default:
		return ErrDistributionInvalid
	}
	if q.Cursor != "" {
		var c vendorFormsCursor
		if b, e := base64.RawURLEncoding.DecodeString(q.Cursor); e != nil {
			return ErrDistributionInvalid
		} else if json.Unmarshal(b, &c) != nil || c.ID == "" || c.UpdatedAt.IsZero() {
			return ErrDistributionInvalid
		}
	}
	return nil
}
func (s *DistributionService) ListVendorForms(ctx context.Context, q VendorFormsQuery) (VendorFormsPage, error) {
	if err := normalizeVendorFormsQuery(&q); err != nil {
		return VendorFormsPage{}, err
	}
	if s == nil {
		return VendorFormsPage{}, ErrDistributionInvalid
	}
	reader, ok := s.store.(vendorFormsStore)
	if !ok {
		return VendorFormsPage{}, ErrDistributionInvalid
	}
	return reader.ListVendorForms(s.withResponseDiscovery(ctx, q.TenantID, q.LegalEntityID, q.PrincipalID), q)
}
func (s *DistributionService) VendorFormSummaries(ctx context.Context, q VendorFormsQuery) ([]VendorFormSummary, error) {
	if err := normalizeVendorFormsQuery(&q); err != nil {
		return nil, err
	}
	if s == nil {
		return nil, ErrDistributionInvalid
	}
	reader, ok := s.store.(vendorFormsStore)
	if !ok {
		return nil, ErrDistributionInvalid
	}
	return reader.VendorFormSummaries(s.withResponseDiscovery(ctx, q.TenantID, q.LegalEntityID, q.PrincipalID), q)
}

type vendorFormsCursor struct {
	UpdatedAt time.Time `json:"at"`
	ID        string    `json:"id"`
}

func vendorFormsPage(rows []VendorFormRow, limit int, now time.Time) VendorFormsPage {
	page := VendorFormsPage{Items: rows, ObservedAt: now}
	if len(rows) > limit {
		page.Items = rows[:limit]
		last := page.Items[limit-1]
		b, _ := json.Marshal(vendorFormsCursor{last.UpdatedAt, last.RequestID})
		page.NextCursor = base64.RawURLEncoding.EncodeToString(b)
	}
	return page
}
func vendorFormRow(req Request, answers map[string]formcontract.AnswerValue, known bool, revision *ResponseRevision, now time.Time) VendorFormRow {
	row := VendorFormRow{RequestID: req.ID, RelationshipID: req.SubjectID, FormTemplateID: req.FormTemplateID, FormTemplateVersion: req.FormTemplateVersion, Title: req.Title, Purpose: req.Purpose, OriginType: req.Origin.Type, OriginID: req.Origin.ID, Deadline: req.Deadline, UpdatedAt: req.UpdatedAt, RecipientHint: req.Recipient.AudienceHint, Current: true, MissingFields: []VendorMissingField{}}
	switch req.Status {
	case RequestSubmitted:
		row.ResponseState = "SUBMITTED"
	case RequestInProgress:
		row.ResponseState = "IN_PROGRESS"
	case RequestCancelled:
		row.ResponseState = "CANCELLED"
	case RequestExpired:
		row.ResponseState = "EXPIRED"
	case RequestDraft:
		row.ResponseState = "REQUEST_READY"
	default:
		row.ResponseState = "AWAITING_RESPONSE"
	}
	if known {
		contract, err := requestAnswerContract(req)
		if err == nil {
			fields, err := formcontract.VisibleFields(contract, answers)
			if err == nil && !collectionApplicabilityUnknown(contract, answers) {
				required, answered, held := 0, 0, 0
				byID := make(map[string]Field, len(req.Fields))
				for _, field := range req.Fields {
					byID[field.ID] = field
				}
				for _, field := range fields {
					if field.Required {
						required++
						if answers[field.ID].Answered() {
							answered++
						} else if CollectionFieldFulfilled(byID[field.ID], now) {
							held++
						} else {
							row.MissingFields = append(row.MissingFields, VendorMissingField{field.ID, field.Label})
						}
					}
				}
				row.RequiredCount = &required
				row.AnsweredRequired = &answered
				row.HeldRequired = &held
				if required > 0 && held == required && (row.ResponseState == "REQUEST_READY" || row.ResponseState == "AWAITING_RESPONSE" || row.ResponseState == "IN_PROGRESS" || row.ResponseState == "EXPIRED") {
					row.ResponseState = "NO_VENDOR_ACTION"
				}
				if row.ResponseState == "IN_PROGRESS" || row.ResponseState == "AWAITING_RESPONSE" {
					if len(answers) > 0 {
						row.ResponseState = "IN_PROGRESS"
					}
					if answered+held == required && answered > 0 {
						row.ResponseState = "READY_TO_SUBMIT"
					}
				}
			}
		}
	}
	if revision != nil {
		row.ResponseID = revision.ID
		row.Score = revision.Score
		row.ResponseState = "SUBMITTED"
		row.SubmittedAt = &revision.CreatedAt
		row.Current = revision.Current
	}
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = now
	}
	populateVendorFormAttention(req, answers, known, &row, nil, now)
	return row
}

// An unanswered applicability question does not prove a conditional
// requirement is absent. Keep the denominator unknown until it is resolved.
func collectionApplicabilityUnknown(contract formcontract.Contract, answers map[string]formcontract.AnswerValue) bool {
	sections := make(map[string]*formcontract.VisibilityCondition, len(contract.Sections))
	for _, section := range contract.Sections {
		sections[section.ID] = section.Condition
	}
	for _, field := range contract.Fields {
		if !field.Required {
			continue
		}
		unknown, omitted := false, false
		for _, condition := range []*formcontract.VisibilityCondition{sections[field.SectionID], field.Condition} {
			if condition == nil {
				continue
			}
			answer := answers[condition.FieldID]
			if !answer.Answered() {
				unknown = true
				continue
			}
			if condition.Operator == formcontract.ConditionAnswered {
				continue
			}
			values := answer.Values
			if text, ok := answer.ScalarText(); ok {
				values = []string{text}
			}
			matches := false
			for _, value := range values {
				if containsOption(condition.Values, strings.TrimSpace(value)) {
					matches = true
					break
				}
			}
			if condition.Operator == formcontract.ConditionNotEquals || condition.Operator == formcontract.ConditionNotIn {
				matches = !matches
			}
			if !matches {
				omitted = true
			}
		}
		if unknown && !omitted {
			return true
		}
	}
	return false
}
func vendorFormOutstanding(row VendorFormRow) bool {
	return row.ResponseState == "AWAITING_RESPONSE" || row.ResponseState == "IN_PROGRESS" || row.ResponseState == "READY_TO_SUBMIT" || row.ResponseState == "REQUEST_READY" || row.ResponseState == "EXPIRED"
}
func vendorEffectiveScore(row VendorFormRow) *ResponseScoreResult {
	if row.AssessmentState != "" && row.AssessmentState != "NOT_REQUIRED" {
		return row.AssessedScore
	}
	return row.Score
}
func vendorFormMatches(row VendorFormRow, q VendorFormsQuery, now time.Time) bool {
	if q.FormTemplateID != "" && q.FormTemplateID != row.FormTemplateID {
		return false
	}
	score := vendorEffectiveScore(row)
	switch q.Filter {
	case "AWAITING_VENDOR":
		return vendorFormOutstanding(row)
	case "AWAITING_REVIEW":
		return row.AssessmentState == "AWAITING_REVIEW" || row.AssessmentState == "IN_REVIEW"
	case "WITH_RISKS":
		return row.Current && row.ResponseState == "SUBMITTED" && score != nil && score.State == ResponseScoreFinal && score.Band != "" && score.Band != formcontract.ConcernLow
	case "HIGH_RISK":
		return row.Current && row.ResponseState == "SUBMITTED" && score != nil && score.State == ResponseScoreFinal && (score.Band == formcontract.ConcernHigh || score.Band == formcontract.ConcernCritical)
	case "OVERDUE":
		return vendorFormOutstanding(row) && row.Deadline.Before(now)
	case "NOT_ASSESSED":
		return score == nil || score.State != ResponseScoreFinal
	}
	return true
}
func summarizeVendorForms(id string, rows []VendorFormRow, now time.Time) VendorFormSummary {
	v := VendorFormSummary{RelationshipID: id, ObservedAt: now}
	rank := map[formcontract.ConcernBand]int{formcontract.ConcernLow: 1, formcontract.ConcernModerate: 2, formcontract.ConcernHigh: 3, formcontract.ConcernCritical: 4}
	for _, r := range rows {
		if r.RelationshipID != id || !r.Current {
			continue
		}
		if r.ResponseCurrency == "PARTIALLY_REPLACED" {
			v.PartiallyReplacedForms++
		}
		if r.ResponseState == "SUBMITTED" && r.Outdated != nil && *r.Outdated {
			v.OutdatedForms++
		}
		if vendorFormOutstanding(r) {
			v.OutstandingForms++
			if r.Deadline.Before(now) {
				v.OverdueForms++
			}
		}
		if r.ResponseState == "SUBMITTED" {
			v.SubmittedForms++
			if r.Outdated == nil {
				v.FreshnessUnknownForms++
			}
			if r.AssessmentState == "AWAITING_REVIEW" || r.AssessmentState == "IN_REVIEW" {
				v.AwaitingReview++
			}
			score := vendorEffectiveScore(r)
			if score == nil || score.State != ResponseScoreFinal {
				v.UnassessedForms++
			} else {
				v.AssessedForms++
				if rank[score.Band] > rank[v.HighestConcern] {
					v.HighestConcern = score.Band
				}
			}
		}
	}
	return v
}
func decodeVendorFormsCursor(raw string) vendorFormsCursor {
	var c vendorFormsCursor
	b, _ := base64.RawURLEncoding.DecodeString(raw)
	_ = json.Unmarshal(b, &c)
	return c
}
func vendorFormsAfter(row VendorFormRow, c vendorFormsCursor) bool {
	return c.ID == "" || row.UpdatedAt.Before(c.UpdatedAt) || row.UpdatedAt.Equal(c.UpdatedAt) && row.RequestID < c.ID
}
func vendorFormsError(err error) error { return fmt.Errorf("vendor form records unavailable: %w", err) }
