package evidence

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func prepareMemoryRecipientAdditions(ctx context.Context, store *MemoryDistributionStore, distribution FormDistribution, inputs []DistributionRecipientInput, now time.Time) ([]memoryDistributionRecipient, []Request, error) {
	if len(inputs) == 0 {
		return nil, nil, nil
	}
	if len(store.recipients[distribution.ID])+len(inputs) > 500 {
		return nil, nil, fmt.Errorf("%w: distribution may contain at most 500 recipients", ErrDistributionInvalid)
	}
	var pinned Request
	for _, recipient := range store.recipients[distribution.ID] {
		if recipient.safe.Role != RecipientTo || recipient.safe.RequestID == "" {
			continue
		}
		if request, ok := store.repo.requests[recipient.safe.RequestID]; ok {
			pinned = cloneRequest(request)
			break
		}
	}
	if pinned.ID == "" {
		return nil, nil, fmt.Errorf("%w: pinned distribution request is unavailable", ErrDistributionInvalid)
	}

	prepared := make([]memoryDistributionRecipient, 0, len(inputs))
	requests := make([]Request, 0, len(inputs))
	for _, input := range inputs {
		if err := validateDistributionRecipientInput(input); err != nil {
			return nil, nil, err
		}
		recipientID, err := id.NewUUIDv7()
		if err != nil {
			return nil, nil, err
		}
		safe := DistributionRecipient{
			ID: recipientID, DistributionID: distribution.ID, TenantID: distribution.TenantID,
			LegalEntityID: distribution.LegalEntityID, Role: input.Role, Type: input.Type,
			PrincipalID: strings.TrimSpace(input.PrincipalID), AudienceHint: strings.TrimSpace(input.AudienceHint),
			ContactLabel: strings.TrimSpace(input.ContactLabel), State: DistributionRecipientPending,
			Version: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
		}
		stored := memoryDistributionRecipient{safe: safe}
		if input.Type == RecipientExternalAudience {
			if store.protector == nil {
				return nil, nil, fmt.Errorf("%w: external recipient protection is unavailable", ErrDistributionInvalid)
			}
			protected, err := store.protector.ProtectRecipientAddress(ctx, distribution.TenantID, distribution.ID, recipientID, strings.TrimSpace(input.Address))
			if err != nil || len(protected.Hash) == 0 || len(protected.Ciphertext) == 0 || strings.TrimSpace(protected.KeyID) == "" {
				return nil, nil, fmt.Errorf("%w: external recipient protection failed", ErrDistributionInvalid)
			}
			stored.protected = protected
		}
		if input.Role == RecipientTo {
			requestID, err := id.NewUUIDv7()
			if err != nil {
				return nil, nil, err
			}
			stored.safe.RequestID = requestID
			request := cloneRequest(pinned)
			request.ID = requestID
			request.Recipient = Recipient{
				Type: input.Type, PrincipalID: strings.TrimSpace(input.PrincipalID), AudienceHint: strings.TrimSpace(input.AudienceHint),
				State: RecipientStateAssigned, Revision: 1,
			}
			if input.Type == RecipientInternalPrincipal {
				request.AudienceType = "INTERNAL"
			} else {
				request.AudienceType = "EXTERNAL"
			}
			request.Deadline = distribution.Deadline
			request.Status = RequestReady
			request.Version = 1
			request.CreatedBy = distribution.CreatedBy
			request.CreatedAt = now.UTC()
			request.UpdatedAt = now.UTC()
			requests = append(requests, request)
		}
		prepared = append(prepared, stored)
	}
	return prepared, requests, nil
}

func validateDistributionRecipientInput(input DistributionRecipientInput) error {
	if input.Role != RecipientTo && input.Role != RecipientCC {
		return fmt.Errorf("%w: recipient role must be TO or CC", ErrDistributionInvalid)
	}
	switch input.Type {
	case RecipientInternalPrincipal:
		if strings.TrimSpace(input.PrincipalID) == "" || strings.TrimSpace(input.Address) != "" {
			return fmt.Errorf("%w: internal recipients require principal_id and no external address", ErrDistributionInvalid)
		}
	case RecipientExternalAudience:
		if strings.TrimSpace(input.PrincipalID) != "" || strings.TrimSpace(input.Address) == "" || strings.TrimSpace(input.AudienceHint) == "" {
			return fmt.Errorf("%w: external recipients require an address and masked audience hint", ErrDistributionInvalid)
		}
	default:
		return fmt.Errorf("%w: unsupported recipient type", ErrDistributionInvalid)
	}
	return nil
}
