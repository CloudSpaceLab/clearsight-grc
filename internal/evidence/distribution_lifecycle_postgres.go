//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *PostgresDistributionStore) AmendDistribution(ctx context.Context, tenantID, legalEntityID, distributionID string, input AmendDistributionInput, now time.Time) (AmendDistributionResult, error) {
	if s == nil || s.repo == nil || s.repo.pool == nil {
		return AmendDistributionResult{}, ErrDistributionInvalid
	}
	tx, err := s.repo.pool.Begin(ctx)
	if err != nil {
		return AmendDistributionResult{}, err
	}
	defer tx.Rollback(ctx)
	current, err := lockPostgresDistribution(ctx, tx, tenantID, legalEntityID, distributionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AmendDistributionResult{}, ErrNotFound
	}
	if err != nil {
		return AmendDistributionResult{}, err
	}
	if current.Version != input.ExpectedVersion || current.Status == DistributionRevoked || current.Status == DistributionCompleted || current.Status == DistributionSuperseded {
		return AmendDistributionResult{}, ErrDistributionConflict
	}

	next := current
	impact := DistributionImpact{CurrentVersion: current.Version, NextVersion: current.Version + 1, EffectiveDeadline: current.Deadline, EffectiveRouteExpiry: current.RouteExpiresAt}
	if input.Deadline != nil {
		deadline := input.Deadline.UTC()
		if !deadline.After(now) {
			return AmendDistributionResult{}, fmt.Errorf("%w: deadline must be in the future", ErrDistributionInvalid)
		}
		next.Deadline = deadline
		impact.DeadlineChanged = !deadline.Equal(current.Deadline)
	}
	if input.RouteExpiresAt != nil {
		expiresAt := input.RouteExpiresAt.UTC()
		if !expiresAt.After(now) {
			return AmendDistributionResult{}, fmt.Errorf("%w: route expiry must be in the future", ErrDistributionInvalid)
		}
		next.RouteExpiresAt = expiresAt
		impact.RouteExpiryChanged = !expiresAt.Equal(current.RouteExpiresAt)
	}
	if next.RouteExpiresAt.After(next.Deadline) {
		next.RouteExpiresAt = next.Deadline
		impact.RouteExpiryChanged = true
	}
	if input.ReminderPolicy != nil {
		next.ReminderPolicy = cloneAnyMap(*input.ReminderPolicy)
		impact.ReminderPolicyChanged = !reflect.DeepEqual(next.ReminderPolicy, current.ReminderPolicy)
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM capture_distribution_recipients WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid`, current.TenantID, current.ID).Scan(&impact.AffectedRecipients); err != nil {
		return AmendDistributionResult{}, err
	}
	if !impact.DeadlineChanged && !impact.RouteExpiryChanged && !impact.ReminderPolicyChanged {
		bundle, err := postgresBundleInTx(ctx, tx, current)
		if err != nil { return AmendDistributionResult{}, err }
		return AmendDistributionResult{Bundle: bundle, Impact: impact}, nil
	}
	policyJSON, err := json.Marshal(next.ReminderPolicy)
	if err != nil {
		return AmendDistributionResult{}, ErrDistributionInvalid
	}
	next.Version++
	next.UpdatedAt = now.UTC()
	tag, err := tx.Exec(ctx, `
		UPDATE capture_form_distributions
		SET deadline=$5,route_expires_at=$6,reminder_policy=$7::jsonb,version=$8,updated_at=$9
		WHERE id=$1::uuid AND tenant_id=$2::uuid AND legal_entity_id=$3::uuid AND version=$4`,
		current.ID, current.TenantID, current.LegalEntityID, input.ExpectedVersion, next.Deadline, next.RouteExpiresAt, string(policyJSON), next.Version, next.UpdatedAt)
	if err != nil || tag.RowsAffected() != 1 {
		return AmendDistributionResult{}, ErrDistributionConflict
	}
	if _, err := tx.Exec(ctx, `UPDATE capture_requests SET deadline=$3,version=version+1,updated_at=$4 WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid AND status NOT IN ('CANCELLED','EXPIRED')`, current.TenantID, current.ID, next.Deadline, next.UpdatedAt); err != nil {
		return AmendDistributionResult{}, err
	}
	if err := appendPostgresDistributionLifecycleEvent(ctx, tx, next, "FORM_DISTRIBUTION_AMENDED", next.CreatedBy, now); err != nil {
		return AmendDistributionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AmendDistributionResult{}, err
	}
	impact.EffectiveDeadline = next.Deadline
	impact.EffectiveRouteExpiry = next.RouteExpiresAt
	bundle, err := s.GetDistribution(ctx, next.TenantID, next.LegalEntityID, next.ID)
	if err != nil { return AmendDistributionResult{}, err }
	return AmendDistributionResult{Bundle: bundle, Impact: impact}, nil
}

func (s *PostgresDistributionStore) TransitionDistribution(ctx context.Context, tenantID, legalEntityID, distributionID string, input TransitionDistributionInput, now time.Time) (DistributionBundle, error) {
	if s == nil || s.repo == nil || s.repo.pool == nil {
		return DistributionBundle{}, ErrDistributionInvalid
	}
	tx, err := s.repo.pool.Begin(ctx)
	if err != nil { return DistributionBundle{}, err }
	defer tx.Rollback(ctx)
	current, err := lockPostgresDistribution(ctx, tx, tenantID, legalEntityID, distributionID)
	if errors.Is(err, pgx.ErrNoRows) { return DistributionBundle{}, ErrNotFound }
	if err != nil { return DistributionBundle{}, err }
	if current.Version != input.ExpectedVersion || !validDistributionTransition(current.Status, input.To, now.UTC(), current.Deadline) {
		return DistributionBundle{}, ErrDistributionConflict
	}
	current.Status = input.To
	current.Version++
	current.UpdatedAt = now.UTC()
	tag, err := tx.Exec(ctx, `UPDATE capture_form_distributions SET status=$5,version=$6,updated_at=$7 WHERE id=$1::uuid AND tenant_id=$2::uuid AND legal_entity_id=$3::uuid AND version=$4`, current.ID, current.TenantID, current.LegalEntityID, input.ExpectedVersion, current.Status, current.Version, current.UpdatedAt)
	if err != nil || tag.RowsAffected() != 1 { return DistributionBundle{}, ErrDistributionConflict }
	workspaceStatus := ResponseWorkspaceOpen
	switch input.To {
	case DistributionLocked: workspaceStatus = ResponseWorkspaceLocked
	case DistributionRevoked: workspaceStatus = ResponseWorkspaceRevoked
	}
	if _, err := tx.Exec(ctx, `UPDATE capture_response_workspaces SET status=$4,version=version+1,updated_at=$5 WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND distribution_id=$3::uuid`, current.TenantID, current.LegalEntityID, current.ID, workspaceStatus, now.UTC()); err != nil { return DistributionBundle{}, err }
	if input.To == DistributionRevoked {
		if _, err := tx.Exec(ctx, `UPDATE capture_distribution_recipients SET state='REVOKED',version=version+1,updated_at=$3 WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid AND state<>'REVOKED'`, current.TenantID, current.ID, now.UTC()); err != nil { return DistributionBundle{}, err }
		if _, err := tx.Exec(ctx, `UPDATE capture_requests SET status='CANCELLED',version=version+1,updated_at=$3 WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid AND status NOT IN ('CANCELLED','EXPIRED')`, current.TenantID, current.ID, now.UTC()); err != nil { return DistributionBundle{}, err }
		if _, err := tx.Exec(ctx, `UPDATE capture_access_routes SET revoked_at=COALESCE(revoked_at,$3) WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid`, current.TenantID, current.ID, now.UTC()); err != nil { return DistributionBundle{}, err }
		if _, err := tx.Exec(ctx, `UPDATE capture_distribution_sessions SET revoked_at=COALESCE(revoked_at,$3) WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid`, current.TenantID, current.ID, now.UTC()); err != nil { return DistributionBundle{}, err }
	}
	if err := appendPostgresDistributionLifecycleEvent(ctx, tx, current, "FORM_DISTRIBUTION_"+string(input.To), input.ActorID, now); err != nil { return DistributionBundle{}, err }
	if err := tx.Commit(ctx); err != nil { return DistributionBundle{}, err }
	return s.GetDistribution(ctx, current.TenantID, current.LegalEntityID, current.ID)
}

func lockPostgresDistribution(ctx context.Context, tx pgx.Tx, tenantID, legalEntityID, distributionID string) (FormDistribution, error) {
	return scanDistribution(tx.QueryRow(ctx, `
		SELECT d.id::text,d.tenant_id::text,d.legal_entity_id::text,d.form_template_id::text,d.form_template_version,
		       d.subject_type,d.subject_id::text,d.title,d.purpose,d.access_policy,d.status,d.deadline,d.route_expires_at,
		       d.reminder_policy,d.created_by::text,d.version,d.created_at,d.updated_at
		FROM capture_form_distributions d
		JOIN tenants t ON t.id=d.tenant_id JOIN legal_entities le ON le.id=d.legal_entity_id AND le.tenant_id=d.tenant_id
		WHERE d.id=$1::uuid AND (t.id::text=$2 OR t.slug=$2) AND (le.id::text=$3 OR le.code=$3) FOR UPDATE`, distributionID, tenantID, legalEntityID))
}

func appendPostgresDistributionLifecycleEvent(ctx context.Context, tx pgx.Tx, distribution FormDistribution, eventType, actorID string, now time.Time) error {
	payloadJSON, err := json.Marshal(map[string]any{"version": distribution.Version, "status": distribution.Status})
	if err != nil { return err }
	if _, err := tx.Exec(ctx, `INSERT INTO capture_distribution_events(tenant_id,legal_entity_id,distribution_id,distribution_version,event_type,payload,actor_id,occurred_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6::jsonb,NULLIF($7,'')::uuid,$8)`, distribution.TenantID, distribution.LegalEntityID, distribution.ID, distribution.Version, eventType, string(payloadJSON), actorID, now.UTC()); err != nil { return err }
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,'FORM_DISTRIBUTION',$2::uuid,$3,$4::jsonb,$5,$5,$5)`, distribution.TenantID, distribution.ID, eventType, string(payloadJSON), now.UTC())
	return err
}

func postgresBundleInTx(ctx context.Context, tx pgx.Tx, distribution FormDistribution) (DistributionBundle, error) {
	rows, err := tx.Query(ctx, `SELECT id::text,distribution_id::text,tenant_id::text,legal_entity_id::text,role,recipient_type,COALESCE(principal_id::text,''),COALESCE(request_id::text,''),audience_hint,contact_label,state,version,created_at,updated_at FROM capture_distribution_recipients WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND distribution_id=$3::uuid ORDER BY created_at,id`, distribution.TenantID, distribution.LegalEntityID, distribution.ID)
	if err != nil { return DistributionBundle{}, err }
	defer rows.Close()
	recipients := []DistributionRecipient{}
	for rows.Next() {
		var value DistributionRecipient
		if err := rows.Scan(&value.ID,&value.DistributionID,&value.TenantID,&value.LegalEntityID,&value.Role,&value.Type,&value.PrincipalID,&value.RequestID,&value.AudienceHint,&value.ContactLabel,&value.State,&value.Version,&value.CreatedAt,&value.UpdatedAt); err != nil { return DistributionBundle{}, err }
		recipients = append(recipients,value)
	}
	if err := rows.Err(); err != nil { return DistributionBundle{}, err }
	workspace, err := scanWorkspace(tx.QueryRow(ctx, `SELECT id::text,tenant_id::text,legal_entity_id::text,distribution_id::text,status,version,created_at,updated_at FROM capture_response_workspaces WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND distribution_id=$3::uuid`, distribution.TenantID, distribution.LegalEntityID, distribution.ID))
	if err != nil { return DistributionBundle{}, err }
	return DistributionBundle{Distribution: distribution, Recipients: recipients, Workspace: workspace}, nil
}
