//go:build postgres

package evidence

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
	now := s.now().UTC()
	prepared, err := s.prepareRecipients(ctx, input, distributionID, now)
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
	reminderPolicy, err := json.Marshal(cloneAnyMap(input.ReminderPolicy))
	if err != nil {
		return DistributionBundle{}, fmt.Errorf("encode reminder policy: %w", err)
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
	if _, err := tx.Exec(ctx, `
		INSERT INTO capture_form_distributions(
			id,tenant_id,legal_entity_id,form_template_id,form_template_version,subject_type,subject_id,
			title,purpose,access_policy,status,deadline,route_expires_at,reminder_policy,created_by,version,created_at,updated_at
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7::uuid,$8,$9,$10,$11,$12,$13,$14::jsonb,$15::uuid,1,$16,$16
		)`, distribution.ID, distribution.TenantID, distribution.LegalEntityID, distribution.FormTemplateID,
		distribution.FormTemplateVersion, distribution.SubjectType, distribution.SubjectID, distribution.Title,
		distribution.Purpose, distribution.AccessPolicy, distribution.Status, distribution.Deadline,
		distribution.RouteExpiresAt, string(reminderPolicy), distribution.CreatedBy, now); err != nil {
		return DistributionBundle{}, fmt.Errorf("insert form distribution: %w", err)
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
	if _, err := tx.Exec(ctx, `
		INSERT INTO capture_response_workspaces(id,tenant_id,legal_entity_id,distribution_id,status,version,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,1,$6,$6)`, workspace.ID, workspace.TenantID,
		workspace.LegalEntityID, workspace.DistributionID, workspace.Status, now); err != nil {
		return DistributionBundle{}, fmt.Errorf("insert response workspace: %w", err)
	}

	payload := map[string]any{
		"version":               1,
		"form_template_id":      distribution.FormTemplateID,
		"form_template_version": distribution.FormTemplateVersion,
		"recipient_count":       len(prepared),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return DistributionBundle{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO capture_distribution_events(
			tenant_id,legal_entity_id,distribution_id,distribution_version,event_type,payload,actor_id,occurred_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,1,'FORM_DISTRIBUTION_CREATED',$4::jsonb,$5::uuid,$6)`,
		distribution.TenantID, distribution.LegalEntityID, distribution.ID, string(payloadJSON), distribution.CreatedBy, now); err != nil {
		return DistributionBundle{}, fmt.Errorf("insert distribution event: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO outbox_events(
			tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at
		) VALUES($1::uuid,'FORM_DISTRIBUTION',$2::uuid,'FORM_DISTRIBUTION_CREATED',$3::jsonb,$4,$4,$4)`,
		distribution.TenantID, distribution.ID, string(payloadJSON), now); err != nil {
		return DistributionBundle{}, fmt.Errorf("insert distribution outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return DistributionBundle{}, err
	}
	return bundleFromPrepared(distribution, prepared, workspace), nil
}

func (s *PostgresDistributionStore) GetDistribution(ctx context.Context, tenantID, legalEntityID, distributionID string) (DistributionBundle, error) {
	if s.repo == nil || s.repo.pool == nil {
		return DistributionBundle{}, fmt.Errorf("postgres distribution repository is required")
	}
	row := s.repo.pool.QueryRow(ctx, `
		SELECT d.id::text,d.tenant_id::text,d.legal_entity_id::text,d.form_template_id::text,d.form_template_version,
		       d.subject_type,d.subject_id::text,d.title,d.purpose,d.access_policy,d.status,d.deadline,d.route_expires_at,
		       d.reminder_policy, d.created_by::text,d.version,d.created_at,d.updated_at
		FROM capture_form_distributions d
		JOIN tenants t ON t.id=d.tenant_id
		JOIN legal_entities le ON le.id=d.legal_entity_id AND le.tenant_id=d.tenant_id
		WHERE d.id=$1::uuid AND (t.id::text=$2 OR t.slug=$2) AND (le.id::text=$3 OR le.code=$3)`,
		distributionID, tenantID, legalEntityID)
	distribution, err := scanDistribution(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return DistributionBundle{}, ErrNotFound
	}
	if err != nil {
		return DistributionBundle{}, err
	}

	recipients, err := s.safeRecipients(ctx, distribution)
	if err != nil {
		return DistributionBundle{}, err
	}
	workspace, err := scanWorkspace(s.repo.pool.QueryRow(ctx, `
		SELECT id::text,tenant_id::text,legal_entity_id::text,distribution_id::text,status,version,created_at,updated_at
		FROM capture_response_workspaces
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND distribution_id=$3::uuid`,
		distribution.TenantID, distribution.LegalEntityID, distribution.ID))
	if errors.Is(err, pgx.ErrNoRows) {
		return DistributionBundle{}, ErrNotFound
	}
	if err != nil {
		return DistributionBundle{}, err
	}
	return DistributionBundle{Distribution: distribution, Recipients: recipients, Workspace: workspace}, nil
}

func (s *PostgresDistributionStore) ListDistributions(ctx context.Context, query DistributionListQuery) ([]FormDistribution, error) {
	if s.repo == nil || s.repo.pool == nil {
		return nil, fmt.Errorf("postgres distribution repository is required")
	}
	if strings.TrimSpace(query.TenantID) == "" || strings.TrimSpace(query.LegalEntityID) == "" || query.Limit < 1 || query.Limit > 100 {
		return nil, fmt.Errorf("tenant_id, legal_entity_id and limit between 1 and 100 are required")
	}
	cursor, err := decodeDistributionCursor(query.Cursor)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.pool.Query(ctx, `
		SELECT d.id::text,d.tenant_id::text,d.legal_entity_id::text,d.form_template_id::text,d.form_template_version,
		       d.subject_type,d.subject_id::text,d.title,d.purpose,d.access_policy,d.status,d.deadline,d.route_expires_at,
		       d.reminder_policy,d.created_by::text,d.version,d.created_at,d.updated_at
		FROM capture_form_distributions d
		JOIN tenants t ON t.id=d.tenant_id
		JOIN legal_entities le ON le.id=d.legal_entity_id AND le.tenant_id=d.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2)
		  AND ($3='' OR d.status=$3)
		  AND ($4::timestamptz IS NULL OR (d.updated_at,d.id) < ($4::timestamptz,$5::uuid))
		ORDER BY d.updated_at DESC,d.id DESC
		LIMIT $6`, query.TenantID, query.LegalEntityID, string(query.Status), cursor.UpdatedAt, nullableUUID(cursor.ID), query.Limit)
	if err != nil {
		return nil, fmt.Errorf("list form distributions: %w", err)
	}
	defer rows.Close()
	values := make([]FormDistribution, 0, query.Limit)
	for rows.Next() {
		value, scanErr := scanDistribution(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *PostgresDistributionStore) prepareRecipients(ctx context.Context, input CreateDistributionInput, distributionID string, now time.Time) ([]postgresPreparedRecipient, error) {
	prepared := make([]postgresPreparedRecipient, 0, len(input.Recipients))
	for _, recipientInput := range input.Recipients {
		recipientID, err := id.NewUUIDv7()
		if err != nil {
			return nil, err
		}
		value := postgresPreparedRecipient{safe: DistributionRecipient{
			ID: recipientID, DistributionID: distributionID, TenantID: input.TenantID, LegalEntityID: input.LegalEntityID,
			Role: recipientInput.Role, Type: recipientInput.Type, PrincipalID: strings.TrimSpace(recipientInput.PrincipalID),
			AudienceHint: strings.TrimSpace(recipientInput.AudienceHint), ContactLabel: strings.TrimSpace(recipientInput.ContactLabel),
			State: DistributionRecipientPending, Version: 1, CreatedAt: now, UpdatedAt: now,
		}}
		if recipientInput.Type == RecipientExternalAudience {
			if s.protector == nil {
				return nil, fmt.Errorf("external recipient protection is unavailable")
			}
			protected, err := s.protector.ProtectRecipientAddress(ctx, input.TenantID, distributionID, recipientID, strings.TrimSpace(recipientInput.Address))
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

func loadExactActiveDistributionForm(ctx context.Context, tx pgx.Tx, input CreateDistributionInput) (DistributionFormRevision, error) {
	var form DistributionFormRevision
	var presentationJSON, sectionsJSON, fieldsJSON []byte
	err := tx.QueryRow(ctx, `
		SELECT f.id::text,f.tenant_id::text,f.legal_entity_id::text,f.version,f.sensitivity,f.presentation,f.sections,f.fields
		FROM monitoring_form_templates f
		JOIN tenants t ON t.id=f.tenant_id
		JOIN legal_entities le ON le.id=f.legal_entity_id AND le.tenant_id=f.tenant_id
		WHERE f.id=$1::uuid AND f.version=$2
		  AND (t.id::text=$3 OR t.slug=$3)
		  AND (le.id::text=$4 OR le.code=$4)
		  AND f.status='ACTIVE' AND f.is_current
		FOR KEY SHARE`, input.FormTemplateID, input.FormTemplateVersion, input.TenantID, input.LegalEntityID).Scan(
		&form.ID, &form.TenantID, &form.LegalEntityID, &form.Version, &form.Sensitivity,
		&presentationJSON, &sectionsJSON, &fieldsJSON,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DistributionFormRevision{}, fmt.Errorf("form revision must be the exact active revision in the requested legal entity")
	}
	if err != nil {
		return DistributionFormRevision{}, fmt.Errorf("load active form revision: %w", err)
	}
	if err := json.Unmarshal(presentationJSON, &form.Presentation); err != nil {
		return DistributionFormRevision{}, err
	}
	if err := json.Unmarshal(sectionsJSON, &form.Sections); err != nil {
		return DistributionFormRevision{}, err
	}
	if err := json.Unmarshal(fieldsJSON, &form.Fields); err != nil {
		return DistributionFormRevision{}, err
	}
	form.Active = true
	return form, nil
}

func insertDistributionRequest(ctx context.Context, tx pgx.Tx, distribution FormDistribution, recipient *postgresPreparedRecipient, form DistributionFormRevision, estimatedMinutes int, now time.Time) error {
	presentationJSON, err := json.Marshal(form.Presentation)
	if err != nil {
		return err
	}
	sectionsJSON, err := json.Marshal(form.Sections)
	if err != nil {
		return err
	}
	fieldsJSON, err := json.Marshal(form.Fields)
	if err != nil {
		return err
	}
	audienceType := "EXTERNAL"
	principalID := ""
	var audienceHash any
	hint := recipient.safe.AudienceHint
	if recipient.safe.Type == RecipientInternalPrincipal {
		audienceType = "INTERNAL"
		principalID = recipient.safe.PrincipalID
		hint = ""
	} else {
		audienceHash = recipient.protected.Hash
		if hint == "" {
			return fmt.Errorf("external TO recipient requires an audience hint")
		}
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO capture_requests(
			id,tenant_id,legal_entity_id,distribution_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,
			recipient_type,recipient_principal_id,recipient_audience_hash,recipient_hint,recipient_state,recipient_revision,recipient_issue_reason,
			estimated_minutes,deadline,known_facts,presentation,sections,fields,source_bindings,form_template_id,form_template_version,
			status,created_by,version,created_at,updated_at
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,$8,$9,$10,
			$11,NULLIF($12,'')::uuid,$13,$14,'ASSIGNED',1,'',$15,$16,'{}'::jsonb,$17::jsonb,$18::jsonb,$19::jsonb,'[]'::jsonb,$20::uuid,$21,
			'READY',$22::uuid,1,$23,$23
		)`, recipient.safe.RequestID, distribution.TenantID, distribution.LegalEntityID, distribution.ID,
		distribution.SubjectType, distribution.SubjectID, distribution.Title, distribution.Purpose, form.Sensitivity,
		audienceType, recipient.safe.Type, principalID, audienceHash, hint, estimatedMinutes, distribution.Deadline,
		string(presentationJSON), string(sectionsJSON), string(fieldsJSON), form.ID, form.Version, distribution.CreatedBy, now)
	if err != nil {
		return fmt.Errorf("insert TO capture request: %w", err)
	}
	return nil
}

func insertDistributionRecipient(ctx context.Context, tx pgx.Tx, distribution FormDistribution, recipient *postgresPreparedRecipient) error {
	var addressHash, ciphertext any
	keyID := ""
	if recipient.safe.Type == RecipientExternalAudience {
		addressHash = recipient.protected.Hash
		ciphertext = recipient.protected.Ciphertext
		keyID = recipient.protected.KeyID
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO capture_distribution_recipients(
			id,distribution_id,tenant_id,legal_entity_id,role,recipient_type,principal_id,request_id,
			address_hash,address_ciphertext,address_key_id,audience_hint,contact_label,state,version,created_at,updated_at
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,NULLIF($7,'')::uuid,NULLIF($8,'')::uuid,
			$9,$10,NULLIF($11,''),$12,$13,$14,1,$15,$15
		)`, recipient.safe.ID, distribution.ID, distribution.TenantID, distribution.LegalEntityID,
		recipient.safe.Role, recipient.safe.Type, recipient.safe.PrincipalID, recipient.safe.RequestID,
		addressHash, ciphertext, keyID, recipient.safe.AudienceHint, recipient.safe.ContactLabel,
		recipient.safe.State, recipient.safe.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert distribution recipient: %w", err)
	}
	return nil
}

func (s *PostgresDistributionStore) safeRecipients(ctx context.Context, distribution FormDistribution) ([]DistributionRecipient, error) {
	rows, err := s.repo.pool.Query(ctx, `
		SELECT id::text,distribution_id::text,tenant_id::text,legal_entity_id::text,role,recipient_type,
		       COALESCE(principal_id::text,''),COALESCE(request_id::text,''),audience_hint,contact_label,state,version,created_at,updated_at
		FROM capture_distribution_recipients
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND distribution_id=$3::uuid
		ORDER BY created_at,id`, distribution.TenantID, distribution.LegalEntityID, distribution.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]DistributionRecipient, 0)
	for rows.Next() {
		var value DistributionRecipient
		if err := rows.Scan(&value.ID, &value.DistributionID, &value.TenantID, &value.LegalEntityID,
			&value.Role, &value.Type, &value.PrincipalID, &value.RequestID, &value.AudienceHint,
			&value.ContactLabel, &value.State, &value.Version, &value.CreatedAt, &value.UpdatedAt); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func scanDistribution(row scanner) (FormDistribution, error) {
	var value FormDistribution
	var reminderPolicy []byte
	if err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.FormTemplateID,
		&value.FormTemplateVersion, &value.SubjectType, &value.SubjectID, &value.Title, &value.Purpose,
		&value.AccessPolicy, &value.Status, &value.Deadline, &value.RouteExpiresAt, &reminderPolicy,
		&value.CreatedBy, &value.Version, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return FormDistribution{}, err
	}
	if err := json.Unmarshal(reminderPolicy, &value.ReminderPolicy); err != nil {
		return FormDistribution{}, err
	}
	return value, nil
}

func scanWorkspace(row scanner) (ResponseWorkspace, error) {
	var value ResponseWorkspace
	if err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.DistributionID,
		&value.Status, &value.Version, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return ResponseWorkspace{}, err
	}
	return value, nil
}

func bundleFromPrepared(distribution FormDistribution, recipients []postgresPreparedRecipient, workspace ResponseWorkspace) DistributionBundle {
	safe := make([]DistributionRecipient, len(recipients))
	for index := range recipients {
		safe[index] = recipients[index].safe
	}
	return DistributionBundle{Distribution: distribution, Recipients: safe, Workspace: workspace}
}

type distributionCursor struct {
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	ID        string     `json:"id,omitempty"`
}

func encodeDistributionCursor(value FormDistribution) string {
	payload, _ := json.Marshal(distributionCursor{UpdatedAt: &value.UpdatedAt, ID: value.ID})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeDistributionCursor(value string) (distributionCursor, error) {
	if strings.TrimSpace(value) == "" {
		return distributionCursor{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return distributionCursor{}, fmt.Errorf("invalid distribution cursor")
	}
	var cursor distributionCursor
	if err := json.Unmarshal(raw, &cursor); err != nil || cursor.UpdatedAt == nil || strings.TrimSpace(cursor.ID) == "" {
		return distributionCursor{}, fmt.Errorf("invalid distribution cursor")
	}
	return cursor, nil
}

var _ DistributionStore = (*PostgresDistributionStore)(nil)
