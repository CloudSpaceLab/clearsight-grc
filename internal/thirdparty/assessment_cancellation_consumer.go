package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

type AssessmentCancellationConsumer struct {
	revoker AssessmentCancellationRevoker
}

func NewAssessmentCancellationConsumer(revoker AssessmentCancellationRevoker) *AssessmentCancellationConsumer {
	return &AssessmentCancellationConsumer{revoker: revoker}
}

func (c *AssessmentCancellationConsumer) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if event.AggregateType != assessmentObjectType || event.EventType != "AssessmentCancelled" {
		return nil
	}
	if c == nil || c.revoker == nil {
		return errors.New("assessment cancellation revoker is unavailable")
	}
	if !validAssessmentIdentifiers(event.TenantID, event.AggregateID) {
		return ErrInvalid
	}
	var payload struct {
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("decode assessment cancellation event: %w", err)
	}
	payload.RequestID = strings.TrimSpace(payload.RequestID)
	if payload.RequestID == "" {
		return nil
	}
	if !validAssessmentIdentifier(payload.RequestID) {
		return ErrInvalid
	}
	return c.revoker.RevokeRequestCapabilities(ctx, event.TenantID, payload.RequestID)
}
