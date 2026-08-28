package evidence

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type memoryDistributionRecipient struct {
	safe      DistributionRecipient
	protected protectedRecipientAddress
}

type MemoryDistributionStore struct {
	mu                  sync.RWMutex
	repo                *MemoryRepository
	forms               DistributionFormReader
	protector           recipientAddressProtector
	now                 func() time.Time
	distributions       map[string]FormDistribution
	recipients          map[string][]memoryDistributionRecipient
	workspaces          map[string]ResponseWorkspace
	requestDistribution map[string]string
	events              []distributionEvent
	outbox              []distributionEvent
}

func NewMemoryDistributionStore(repo *MemoryRepository, forms DistributionFormReader, protector recipientAddressProtector) *MemoryDistributionStore {
	return &MemoryDistributionStore{
		repo:                repo,
		forms:               forms,
		protector:           protector,
		now:                 time.Now,
		distributions:       map[string]FormDistribution{},
		recipients:          map[string][]memoryDistributionRecipient{},
		workspaces:          map[string]ResponseWorkspace{},
		requestDistribution: map[string]string{},
	}
}

func (s *MemoryDistributionStore) CreateDistribution(ctx context.Context, input CreateDistributionInput) (DistributionBundle, error) {
	if s.repo == nil || s.forms == nil {
		return DistributionBundle{}, fmt.Errorf("distribution persistence dependencies are required")
	}
	if err := validateCreateDistributionInput(input); err != nil {
		return DistributionBundle{}, err
	}
	form, err := s.forms.GetDistributionFormRevision(ctx, input.TenantID, input.LegalEntityID, input.FormTemplateID, input.FormTemplateVersion)
	if err != nil {
		return DistributionBundle{}, err
	}
	if !form.Active || form.ID != input.FormTemplateID || form.Version != input.FormTemplateVersion || form.TenantID != input.TenantID || form.LegalEntityID != input.LegalEntityID {
		return DistributionBundle{}, fmt.Errorf("form revision must be the exact active revision in the requested legal entity")
	}

	distributionID, err := id.NewUUIDv7()
	if err != nil {
		return DistributionBundle{}, err
	}
	workspaceID, err := id.NewUUIDv7()
	if err != nil {
		return DistributionBundle{}, err
	}
	now := s.now().UTC()
	distribution := FormDistribution{
		ID: distributionID, TenantID: input.TenantID, LegalEntityID: input.LegalEntityID,
		FormTemplateID: input.FormTemplateID, FormTemplateVersion: input.FormTemplateVersion,
		SubjectType: strings.TrimSpace(input.SubjectType), SubjectID: input.SubjectID,
		Title: strings.TrimSpace(input.Title), Purpose: strings.TrimSpace(input.Purpose),
		AccessPolicy: input.AccessPolicy, Status: DistributionDraft, Deadline: input.Deadline.UTC(),
		RouteExpiresAt: input.RouteExpiresAt.UTC(), ReminderPolicy: cloneAnyMap(input.ReminderPolicy),
		CreatedBy: input.CreatedBy, Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	workspace := ResponseWorkspace{
		ID: workspaceID, TenantID: input.TenantID, LegalEntityID: input.LegalEntityID,
		DistributionID: distributionID, Status: ResponseWorkspaceOpen, Version: 1,
		CreatedAt: now, UpdatedAt: now,
	}

	storedRecipients := make([]memoryDistributionRecipient, 0, len(input.Recipients))
	requests := make([]Request, 0, len(input.Recipients))
	for _, recipientInput := range input.Recipients {
		recipientID, idErr := id.NewUUIDv7()
		if idErr != nil {
			return DistributionBundle{}, idErr
		}
		safe := DistributionRecipient{
			ID: recipientID, DistributionID: distributionID, TenantID: input.TenantID,
			LegalEntityID: input.LegalEntityID, Role: recipientInput.Role, Type: recipientInput.Type,
			PrincipalID: strings.TrimSpace(recipientInput.PrincipalID), AudienceHint: strings.TrimSpace(recipientInput.AudienceHint),
			ContactLabel: strings.TrimSpace(recipientInput.ContactLabel), State: DistributionRecipientPending,
			Version: 1, CreatedAt: now, UpdatedAt: now,
		}
		stored := memoryDistributionRecipient{safe: safe}
		if recipientInput.Type == RecipientExternalAudience {
			if s.protector == nil {
				return DistributionBundle{}, fmt.Errorf("external recipient protection is unavailable")
			}
			protected, protectErr := s.protector.ProtectRecipientAddress(ctx, input.TenantID, distributionID, recipientID, strings.TrimSpace(recipientInput.Address))
			if protectErr != nil {
				return DistributionBundle{}, protectErr
			}
			if len(protected.Hash) == 0 || len(protected.Ciphertext) == 0 || strings.TrimSpace(protected.KeyID) == "" {
				return DistributionBundle{}, fmt.Errorf("recipient protector returned incomplete protected material")
			}
			stored.protected = protected
		}
		if recipientInput.Role == RecipientTo {
			requestID, requestErr := id.NewUUIDv7()
			if requestErr != nil {
				return DistributionBundle{}, requestErr
			}
			stored.safe.RequestID = requestID
			requests = append(requests, materializeDistributionRequest(requestID, distribution, stored.safe, form, input.EstimatedMinutes, now))
		}
		storedRecipients = append(storedRecipients, stored)
	}

	// No mutation occurs before every form/recipient/request has been validated
	// and every external address has been protected. The two locks make the
	// remaining in-memory commit indivisible to both legacy request readers and
	// distribution readers.
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repo.mu.Lock()
	defer s.repo.mu.Unlock()
	if _, exists := s.distributions[distributionID]; exists {
		return DistributionBundle{}, fmt.Errorf("distribution id collision")
	}
	for _, request := range requests {
		if _, exists := s.repo.requests[request.ID]; exists {
			return DistributionBundle{}, fmt.Errorf("request id collision")
		}
	}
	for _, request := range requests {
		s.repo.requests[request.ID] = cloneRequest(request)
		s.requestDistribution[request.ID] = distributionID
	}
	s.distributions[distributionID] = cloneDistribution(distribution)
	s.recipients[distributionID] = cloneMemoryDistributionRecipients(storedRecipients)
	s.workspaces[distributionID] = workspace
	event := distributionEvent{DistributionID: distributionID, Version: 1, EventType: "FORM_DISTRIBUTION_CREATED", ActorID: input.CreatedBy, OccurredAt: now}
	s.events = append(s.events, event)
	s.outbox = append(s.outbox, event)
	return bundleFromMemory(distribution, storedRecipients, workspace), nil
}

func (s *MemoryDistributionStore) GetDistribution(_ context.Context, tenantID, legalEntityID, distributionID string) (DistributionBundle, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	distribution, ok := s.distributions[distributionID]
	if !ok || distribution.TenantID != tenantID || distribution.LegalEntityID != legalEntityID {
		return DistributionBundle{}, ErrNotFound
	}
	return bundleFromMemory(distribution, s.recipients[distributionID], s.workspaces[distributionID]), nil
}

func (s *MemoryDistributionStore) ListDistributions(_ context.Context, query DistributionListQuery) ([]FormDistribution, error) {
	if !normalizeDistributionListQuery(&query, s.now()) {
		return nil, fmt.Errorf("distribution list filters are invalid")
	}
	cursor, err := decodeDistributionCursor(query.Cursor)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	values := make([]FormDistribution, 0)
	for _, value := range s.distributions {
		if !distributionMatchesListQuery(value, query) || !distributionAfterCursor(value, cursor) {
			continue
		}
		values = append(values, cloneDistribution(value))
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].UpdatedAt.Equal(values[j].UpdatedAt) {
			return values[i].ID > values[j].ID
		}
		return values[i].UpdatedAt.After(values[j].UpdatedAt)
	})
	if len(values) > query.Limit {
		values = values[:query.Limit]
	}
	return values, nil
}

func validateCreateDistributionInput(input CreateDistributionInput) error {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.LegalEntityID) == "" || strings.TrimSpace(input.FormTemplateID) == "" || input.FormTemplateVersion < 1 || strings.TrimSpace(input.SubjectType) == "" || strings.TrimSpace(input.SubjectID) == "" || strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Purpose) == "" || strings.TrimSpace(input.CreatedBy) == "" {
		return fmt.Errorf("distribution scope, exact form revision, subject, title, purpose and creator are required")
	}
	if input.AccessPolicy != AccessDirectMagicLink && input.AccessPolicy != AccessSharedEmailOTP && input.AccessPolicy != AccessDirectEmailOTP {
		return fmt.Errorf("unsupported distribution access policy")
	}
	if input.EstimatedMinutes < 1 || input.EstimatedMinutes > 60 {
		return fmt.Errorf("estimated_minutes must be between 1 and 60")
	}
	if input.Deadline.IsZero() || input.RouteExpiresAt.IsZero() || input.RouteExpiresAt.After(input.Deadline) {
		return fmt.Errorf("route expiry must be set and no later than the distribution deadline")
	}
	if len(input.Recipients) == 0 || len(input.Recipients) > 500 {
		return fmt.Errorf("distribution recipients must contain between 1 and 500 entries")
	}
	toCount := 0
	for _, recipient := range input.Recipients {
		if recipient.Role != RecipientTo && recipient.Role != RecipientCC {
			return fmt.Errorf("recipient role must be TO or CC")
		}
		if recipient.Role == RecipientTo {
			toCount++
		}
		switch recipient.Type {
		case RecipientInternalPrincipal:
			if strings.TrimSpace(recipient.PrincipalID) == "" || strings.TrimSpace(recipient.Address) != "" {
				return fmt.Errorf("internal recipients require principal_id and cannot carry an external address")
			}
		case RecipientExternalAudience:
			if strings.TrimSpace(recipient.PrincipalID) != "" || strings.TrimSpace(recipient.Address) == "" {
				return fmt.Errorf("external recipients require an address and cannot carry principal_id")
			}
		default:
			return fmt.Errorf("unsupported recipient type")
		}
	}
	if toCount == 0 {
		return fmt.Errorf("at least one TO recipient is required")
	}
	return nil
}

func materializeDistributionRequest(requestID string, distribution FormDistribution, recipient DistributionRecipient, form DistributionFormRevision, estimatedMinutes int, now time.Time) Request {
	audienceType := "EXTERNAL"
	if recipient.Type == RecipientInternalPrincipal {
		audienceType = "INTERNAL"
	}
	return Request{
		ID: requestID, TenantID: distribution.TenantID, LegalEntityID: distribution.LegalEntityID,
		SubjectType: distribution.SubjectType, SubjectID: distribution.SubjectID, Title: distribution.Title,
		Purpose: distribution.Purpose, WhyYou: distribution.Purpose, Sensitivity: form.Sensitivity,
		AudienceType: audienceType,
		Recipient: Recipient{Type: recipient.Type, PrincipalID: recipient.PrincipalID, AudienceHint: recipient.AudienceHint, State: RecipientStateAssigned, Revision: 1},
		EstimatedMinutes: estimatedMinutes, Deadline: distribution.Deadline, KnownFacts: map[string]string{},
		Presentation: form.Presentation, Sections: cloneSections(form.Sections), Fields: requestFieldsFromContract(form.Fields),
		FormTemplateID: form.ID, FormTemplateVersion: form.Version, Status: RequestReady,
		CreatedBy: distribution.CreatedBy, Version: 1, CreatedAt: now, UpdatedAt: now,
	}
}

func requestFieldsFromContract(fields []formcontract.Field) []Field {
	result := make([]Field, len(fields))
	for index, field := range fields {
		result[index] = Field{
			ID: field.ID, SectionID: field.SectionID, Label: field.Label, Type: string(field.Type), Required: field.Required,
			Description: field.Description, Options: append([]string(nil), field.Options...), AcceptedFormats: append([]string(nil), field.AcceptedFormats...),
			Attestation: field.Attestation, Constraints: field.Constraints, Condition: field.Condition, Scoring: field.Scoring,
			CollectionIntent: field.CollectionIntent, RecordTarget: field.RecordTarget, BrowserCachePolicy: field.BrowserCachePolicy,
		}
	}
	return result
}

func bundleFromMemory(distribution FormDistribution, recipients []memoryDistributionRecipient, workspace ResponseWorkspace) DistributionBundle {
	safe := make([]DistributionRecipient, len(recipients))
	for index, recipient := range recipients {
		safe[index] = recipient.safe
	}
	return DistributionBundle{Distribution: cloneDistribution(distribution), Recipients: safe, Workspace: workspace}
}

func cloneDistribution(value FormDistribution) FormDistribution {
	value.ReminderPolicy = cloneAnyMap(value.ReminderPolicy)
	return value
}

func cloneAnyMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func cloneMemoryDistributionRecipients(values []memoryDistributionRecipient) []memoryDistributionRecipient {
	result := make([]memoryDistributionRecipient, len(values))
	for index, value := range values {
		result[index] = value
		result[index].protected.Hash = append([]byte(nil), value.protected.Hash...)
		result[index].protected.Ciphertext = append([]byte(nil), value.protected.Ciphertext...)
	}
	return result
}
