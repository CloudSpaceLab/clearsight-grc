package formpolicy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

type ScoredResponseHandler interface {
	Handle(context.Context, ScoredResponseEvent) ([]ExecutionReceipt, error)
}

type ScoredResponsePublisher struct {
	Handler ScoredResponseHandler
}

func (publisher ScoredResponsePublisher) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if event.EventType != "FORM_RESPONSE_SCORED" && event.EventType != "FORM_RESPONSE_ASSESSED" {
		return nil
	}
	if publisher.Handler == nil || strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.TenantID) == "" || event.OccurredAt.IsZero() || len(event.Payload) == 0 || len(event.Payload) > 16*1024 {
		return fmt.Errorf("scored response event is invalid")
	}
	var payload struct {
		ResultBasis         ResultBasis                 `json:"result_basis,omitempty"`
		AssessmentVersion   int64                       `json:"assessment_version,omitempty"`
		Version             int64                       `json:"version"`
		RequestID           *string                     `json:"request_id,omitempty"`
		RequestVersion      *int64                      `json:"request_version,omitempty"`
		ResponseRevisionID  string                      `json:"response_revision_id"`
		FormTemplateID      string                      `json:"form_template_id"`
		FormTemplateVersion int64                       `json:"form_template_version"`
		ScoreState          evidence.ResponseScoreState `json:"score_state"`
	}
	decoder := json.NewDecoder(bytes.NewReader(event.Payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || payload.Version < 1 || payload.FormTemplateVersion < 1 || strings.TrimSpace(payload.ResponseRevisionID) == "" || len(payload.ResponseRevisionID) > 512 || strings.TrimSpace(payload.FormTemplateID) == "" || len(payload.FormTemplateID) > 512 || payload.ScoreState != evidence.ResponseScoreFinal && payload.ScoreState != evidence.ResponseScoreProvisional && payload.ScoreState != evidence.ResponseScoreFailed && payload.ScoreState != evidence.ResponseScoreNotConfigured {
		return fmt.Errorf("scored response event payload is invalid")
	}
	// Older durable events predate request provenance. New events must provide
	// the complete pair; this is the consumed request revision, not its live version.
	requestID, requestVersion := "", int64(0)
	if payload.RequestID != nil || payload.RequestVersion != nil {
		if payload.RequestID == nil || payload.RequestVersion == nil || strings.TrimSpace(*payload.RequestID) == "" || len(*payload.RequestID) > 512 || *payload.RequestVersion < 1 {
			return fmt.Errorf("scored response request provenance is invalid")
		}
		requestID, requestVersion = strings.TrimSpace(*payload.RequestID), *payload.RequestVersion
	}
	basis := ResultAutomatic
	if event.EventType == "FORM_RESPONSE_ASSESSED" {
		if payload.ResultBasis != ResultBankAssessed || payload.AssessmentVersion < 1 {
			return fmt.Errorf("bank assessment event is invalid")
		}
		if payload.ScoreState != evidence.ResponseScoreFinal {
			return nil
		}
		basis = ResultBankAssessed
	} else if payload.ResultBasis != "" && payload.ResultBasis != ResultAutomatic || payload.AssessmentVersion != 0 {
		return fmt.Errorf("automatic score event is invalid")
	}
	_, err := publisher.Handler.Handle(ctx, ScoredResponseEvent{ResultBasis: basis, AssessmentVersion: payload.AssessmentVersion, ID: event.ID, TenantID: event.TenantID, ResponseRevisionID: strings.TrimSpace(payload.ResponseRevisionID), RequestID: requestID, RequestVersion: requestVersion, OccurredAt: event.OccurredAt.UTC()})
	return err
}

var _ workflowruntime.Publisher = ScoredResponsePublisher{}
