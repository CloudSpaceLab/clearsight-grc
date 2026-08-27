//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *PostgresDistributionStore) GetDistribution(ctx context.Context, tenantID, legalEntityID, distributionID string) (DistributionBundle, error) {
	if s.repo == nil || s.repo.pool == nil {
		return DistributionBundle{}, fmt.Errorf("postgres distribution repository is required")
	}
	row := s.repo.pool.QueryRow(ctx, `
		SELECT d.id::text,d.tenant_id::text,d.legal_entity_id::text,d.form_template_id::text,d.form_template_version,
		       d.subject_type,d.subject_id::text,d.title,d.purpose,d.access_policy,d.status,d.deadline,d.route_expires_at,
		       d.reminder_policy,d.created_by::text,d.version,d.created_at,d.updated_at
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
