//go:build postgres

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
)

const formProposalAggregateType = "FORM_TEMPLATE_PROPOSAL"

type formProposalGenerationPublisher struct {
	service *monitoring.FormProposalService
}

type formProposalGenerationPayload struct {
	ProposalID     string `json:"proposal_id"`
	LegalEntityID  string `json:"legal_entity_id"`
	SourceDocument string `json:"source_document_id"`
	SourceVersion  int64  `json:"source_document_version"`
}

func buildFormProposalGenerationPublisher(pool *pgxpool.Pool, documents *documentimport.Service) workflowruntime.Publisher {
	store := monitoring.NewPostgresFormProposalStore(pool)
	return &formProposalGenerationPublisher{service: monitoring.NewFormProposalService(store, documents, nil)}
}

func (p *formProposalGenerationPublisher) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if event.EventType != monitoring.EventFormProposalGenerationRequested {
		return nil
	}
	if event.AggregateType != formProposalAggregateType {
		return fmt.Errorf("form proposal generation event has aggregate type %q", event.AggregateType)
	}
	if p == nil || p.service == nil {
		return fmt.Errorf("form proposal generation service is unavailable")
	}
	var payload formProposalGenerationPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("decode form proposal generation event: %w", err)
	}
	payload.ProposalID = strings.TrimSpace(payload.ProposalID)
	payload.LegalEntityID = strings.TrimSpace(payload.LegalEntityID)
	if payload.ProposalID == "" || payload.LegalEntityID == "" || payload.ProposalID != event.AggregateID {
		return fmt.Errorf("form proposal generation event does not identify the queued proposal")
	}
	_, err := p.service.Generate(ctx, event.TenantID, payload.LegalEntityID, payload.ProposalID)
	if err != nil {
		return fmt.Errorf("generate form proposal %s: %w", payload.ProposalID, err)
	}
	return nil
}

var _ workflowruntime.Publisher = (*formProposalGenerationPublisher)(nil)
