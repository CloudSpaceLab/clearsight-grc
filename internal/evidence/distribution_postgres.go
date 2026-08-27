//go:build postgres

package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
)

type PostgresDistributionStore struct {
	repo      *PostgresRepository
	protector recipientAddressProtector
	now       func() time.Time
}

type postgresPreparedRecipient struct {
	safe      DistributionRecipient
	protected protectedRecipientAddress
}

func NewPostgresDistributionStore(repo *PostgresRepository, protector recipientAddressProtector) *PostgresDistributionStore {
	return &PostgresDistributionStore{repo: repo, protector: protector, now: time.Now}
}

func (s *PostgresDistributionStore) CreateDistribution(ctx context.Context, input CreateDistributionInput) (DistributionBundle, error) {
	if s.repo == nil || s.repo.pool == nil {
		return DistributionBundle{}, fmt.Errorf("postgres distribution repository is required")
	}
	if err := validateCreateDistributionInput(input); err != nil {
		return DistributionBundle{}, err
	}

	distributionID, err := id.NewUUIDv7()
	if err != nil {
		return DistributionBundle{}, err
	}
	workspaceID, err := id.NewUUIDv7()
	if err != nil {
		return DistributionBundle{}, err
	}
	tenantID, legalEntityID, err := s.resolveRecipientProtectionScope(ctx, input)
	if err != nil {
		return DistributionBundle{}, err
	}
	now := s.now().UTC()
	prepared, err := s.prepareRecipients(ctx, input, distributionID, tenantID, legalEntityID, now)
	if err != nil {
		return DistributionBundle{}, err
	}

	tx, err := s.repo.pool.Begin(ctx)
	if err != nil {
		return DistributionBundle{}, err
	}
	defer tx.Rollback(ctx)

	form, err := loadExactActiveDistributionForm(ctx, tx, input)
	if err != nil {
		return DistributionBundle{}, err
	}
	if form.TenantID != tenantID || form.LegalEntityID != legalEntityID {
		return DistributionBundle{}, fmt.Errorf("form revision scope changed during distribution creation")
	}
	distribution := FormDistribution{
		ID: distributionID, TenantID: form.TenantID, LegalEntityID: form.LegalEntityID,
		FormTemplateID: form.ID, FormTemplateVersion: form.Version,
		SubjectType: strings.TrimSpace(input.SubjectType), SubjectID: strings.TrimSpace(input.SubjectID),
		Title: strings.TrimSpace(input.Title), Purpose: strings.TrimSpace(input.Purpose),
		AccessPolicy: input.AccessPolicy, Status: DistributionDraft,
		Deadline: input.Deadline.UTC(), RouteExpiresAt: input.RouteExpiresAt.UTC(),
		ReminderPolicy: cloneAnyMap(input.ReminderPolicy), CreatedBy: strings.TrimSpace(input.CreatedBy),
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := insertFormDistribution(ctx, tx, distribution); err != nil {
		return DistributionBundle{}, err
	}

	for index := range prepared {
		recipient := &prepared[index]
		if recipient.safe.Role == RecipientTo {
			if err := insertDistributionRequest(ctx, tx, distribution, recipient, form, input.EstimatedMinutes, now); err != nil {
				return DistributionBundle{}, err
			}
		}
		if err := insertDistributionRecipient(ctx, tx, distribution, recipient); err != nil {
			return DistributionBundle{}, err
		}
	}

	workspace := ResponseWorkspace{
		ID: workspaceID, TenantID: distribution.TenantID, LegalEntityID: distribution.LegalEntityID,
		DistributionID: distribution.ID, Status: ResponseWorkspaceOpen, Version: 1,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := insertDistributionWorkspace(ctx, tx, workspace); err != nil {
		return DistributionBundle{}, err
	}
	if err := insertDistributionCreatedEvents(ctx, tx, distribution, len(prepared), now); err != nil {
		return DistributionBundle{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return DistributionBundle{}, err
	}
	return bundleFromPrepared(distribution, prepared, workspace), nil
}

func (s *PostgresDistributionStore) resolveRecipientProtectionScope(ctx context.Context, input CreateDistributionInput) (string, string, error) {
	var tenantID, legalEntityID string
	err := s.repo.pool.QueryRow(ctx, `
		SELECT f.tenant_id::text,f.legal_entity_id::text
		FROM monitoring_form_templates f
		JOIN tenants t ON t.id=f.tenant_id
		JOIN legal_entities le ON le.id=f.legal_entity_id AND le.tenant_id=f.tenant_id
		WHERE f.id=$1::uuid AND f.version=$2
		  AND (t.id::text=$3 OR t.slug=$3)
		  AND (le.id::text=$4 OR le.code=$4)
		  AND f.status='ACTIVE' AND f.is_current`,
		input.FormTemplateID, input.FormTemplateVersion, input.TenantID, input.LegalEntityID).Scan(&tenantID, &legalEntityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", fmt.Errorf("form revision must be the exact active revision in the requested legal entity")
	}
	if err != nil {
		return "", "", fmt.Errorf("resolve recipient protection scope: %w", err)
	}
	return tenantID, legalEntityID, nil
}

func (s *PostgresDistributionStore) prepareRecipients(ctx context.Context, input CreateDistributionInput, distributionID, tenantID, legalEntityID string, now time.Time) ([]postgresPreparedRecipient, error) {
	prepared := make([]postgresPreparedRecipient, 0, len(input.Recipients))
	for _, recipientInput := range input.Recipients {
		recipientID, err := id.NewUUIDv7()
		if err != nil {
			return nil, err
		}
		value := postgresPreparedRecipient{safe: DistributionRecipient{
			ID: recipientID, DistributionID: distributionID, TenantID: tenantID, LegalEntityID: legalEntityID,
			Role: recipientInput.Role, Type: recipientInput.Type, PrincipalID: strings.TrimSpace(recipientInput.PrincipalID),
			AudienceHint: strings.TrimSpace(recipientInput.AudienceHint), ContactLabel: strings.TrimSpace(recipientInput.ContactLabel),
			State: DistributionRecipientPending, Version: 1, CreatedAt: now, UpdatedAt: now,
		}}
		if recipientInput.Type == RecipientExternalAudience {
			if s.protector == nil {
				return nil, fmt.Errorf("external recipient protection is unavailable")
			}
			protected, err := s.protector.ProtectRecipientAddress(ctx, tenantID, distributionID, recipientID, strings.TrimSpace(recipientInput.Address))
			if err != nil {
				return nil, err
			}
			if len(protected.Hash) != 32 || len(protected.Ciphertext) == 0 || strings.TrimSpace(protected.KeyID) == "" {
				return nil, fmt.Errorf("recipient protector returned incomplete protected material")
			}
			value.protected = protected
		}
		if recipientInput.Role == RecipientTo {
			requestID, err := id.NewUUIDv7()
			if err != nil {
				return nil, err
			}
			value.safe.RequestID = requestID
		}
		prepared = append(prepared, value)
	}
	return prepared, nil
}

var _ DistributionStore = (*PostgresDistributionStore)(nil)
