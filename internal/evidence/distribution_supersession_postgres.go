//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
)

func (store *PostgresDistributionStore) LoadSupersessionTargetForm(ctx context.Context, tenantID, legalEntityID, formID string, version int64) (DistributionFormRevision, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return DistributionFormRevision{}, ErrDistributionInvalid
	}
	var form DistributionFormRevision
	var presentationJSON, sectionsJSON, fieldsJSON []byte
	err := store.repo.pool.QueryRow(ctx, `
		SELECT f.id::text,f.tenant_id::text,f.legal_entity_id::text,f.version,f.sensitivity,f.presentation,f.sections,f.fields
		FROM monitoring_form_templates f
		JOIN tenants t ON t.id=f.tenant_id
		JOIN legal_entities le ON le.id=f.legal_entity_id AND le.tenant_id=f.tenant_id
		WHERE f.id=$1::uuid AND f.version=$2 AND (t.id::text=$3 OR t.slug=$3) AND (le.id::text=$4 OR le.code=$4)
		  AND f.status='ACTIVE' AND f.is_current`, formID, version, tenantID, legalEntityID).Scan(
		&form.ID, &form.TenantID, &form.LegalEntityID, &form.Version, &form.Sensitivity,
		&presentationJSON, &sectionsJSON, &fieldsJSON,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DistributionFormRevision{}, ErrNotFound
	}
	if err != nil {
		return DistributionFormRevision{}, err
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

func (store *PostgresDistributionStore) LoadSupersessionSnapshot(ctx context.Context, tenantID, legalEntityID, distributionID string) (supersessionSnapshot, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return supersessionSnapshot{}, ErrDistributionInvalid
	}
	bundle, err := store.GetDistribution(ctx, tenantID, legalEntityID, distributionID)
	if err != nil {
		return supersessionSnapshot{}, err
	}
	var request Request
	for _, recipient := range bundle.Recipients {
		if recipient.Role != RecipientTo || recipient.RequestID == "" {
			continue
		}
		request, err = store.GetRequest(ctx, bundle.Distribution.TenantID, recipient.RequestID)
		if err == nil {
			break
		}
	}
	if request.ID == "" {
		return supersessionSnapshot{}, ErrDistributionInvalid
	}

	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return supersessionSnapshot{}, err
	}
	defer tx.Rollback(ctx)
	view, err := loadPostgresSupersessionWorkspace(ctx, tx, bundle.Workspace, request)
	if err != nil {
		return supersessionSnapshot{}, err
	}
	protected, err := loadPostgresSupersessionProtectedRecipients(ctx, tx, bundle.Distribution)
	if err != nil {
		return supersessionSnapshot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return supersessionSnapshot{}, err
	}
	return supersessionSnapshot{
		Bundle: bundle, Workspace: view, Request: request, EstimatedMinutes: request.EstimatedMinutes,
		ProtectedAddresses: protected,
	}, nil
}

func loadPostgresSupersessionWorkspace(ctx context.Context, tx pgx.Tx, workspace ResponseWorkspace, request Request) (ResponseWorkspaceView, error) {
	view := workspaceDefaultView(workspace, request)
	rows, err := tx.Query(ctx, `
		SELECT e.recipient_id::text,e.request_id::text,e.base_version,e.result_version,e.patch,e.created_at
		FROM capture_response_workspace_edits e
		WHERE e.tenant_id=$1::uuid AND e.legal_entity_id=$2::uuid AND e.distribution_id=$3::uuid AND e.workspace_id=$4::uuid
		ORDER BY e.result_version,e.id`, workspace.TenantID, workspace.LegalEntityID, workspace.DistributionID, workspace.ID)
	if err != nil {
		return ResponseWorkspaceView{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var recipientID, requestID string
		var baseVersion, resultVersion int64
		var patchJSON []byte
		var createdAt time.Time
		if err := rows.Scan(&recipientID, &requestID, &baseVersion, &resultVersion, &patchJSON, &createdAt); err != nil {
			return ResponseWorkspaceView{}, err
		}
		var patch postgresWorkspaceEditPatch
		if err := json.Unmarshal(patchJSON, &patch); err != nil || patch.FieldID == "" || resultVersion != baseVersion+1 || !validPersistedWorkspaceEditOrigin(patch) || !validAccessAssurance(patch.Assurance) {
			return ResponseWorkspaceView{}, fmt.Errorf("%w: invalid persisted workspace edit", ErrWorkspaceUnavailable)
		}
		applyWorkspaceEdit(view.Answers, FieldEdit{FieldID: patch.FieldID, Value: patch.Value})
		view.PresentationMode = patch.PresentationMode
		view.FieldSequences[patch.FieldID] = resultVersion
		view.FieldProvenance[patch.FieldID] = WorkspaceFieldProvenance{
			RecipientID: recipientID, RequestID: requestID, Assurance: patch.Assurance,
			Sequence: resultVersion, UpdatedAt: createdAt.UTC(),
		}
	}
	if err := rows.Err(); err != nil {
		return ResponseWorkspaceView{}, err
	}
	revision, err := loadCurrentPostgresResponseRevision(ctx, tx, workspace.TenantID, workspace.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return ResponseWorkspaceView{}, err
	}
	if err == nil {
		view.CurrentRevision = &revision
	}
	return view, nil
}

func loadPostgresSupersessionProtectedRecipients(ctx context.Context, tx pgx.Tx, distribution FormDistribution) (map[string]protectedRecipientAddress, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text,address_hash,address_ciphertext,address_key_id
		FROM capture_distribution_recipients
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND distribution_id=$3::uuid
		  AND recipient_type='EXTERNAL_AUDIENCE' AND state<>'REVOKED'`,
		distribution.TenantID, distribution.LegalEntityID, distribution.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]protectedRecipientAddress{}
	for rows.Next() {
		var recipientID string
		var value protectedRecipientAddress
		if err := rows.Scan(&recipientID, &value.Hash, &value.Ciphertext, &value.KeyID); err != nil {
			return nil, err
		}
		if len(value.Hash) != 32 || len(value.Ciphertext) == 0 || value.KeyID == "" {
			return nil, ErrProtectedRecipientInvalid
		}
		result[recipientID] = value
	}
	return result, rows.Err()
}

func (store *PostgresDistributionStore) CommitSupersession(ctx context.Context, command supersessionCommit) (DistributionBundle, DistributionBundle, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return DistributionBundle{}, DistributionBundle{}, ErrDistributionInvalid
	}
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	defer tx.Rollback(ctx)

	previous, err := lockPostgresDistribution(ctx, tx, command.TenantID, command.LegalEntityID, command.PreviousDistributionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return DistributionBundle{}, DistributionBundle{}, ErrNotFound
	}
	if err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	replacement, err := lockPostgresDistribution(ctx, tx, command.TenantID, command.LegalEntityID, command.ReplacementDistributionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return DistributionBundle{}, DistributionBundle{}, ErrNotFound
	}
	if err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	oldWorkspace, err := lockPostgresSupersessionWorkspace(ctx, tx, previous)
	if err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	newWorkspace, err := lockPostgresSupersessionWorkspace(ctx, tx, replacement)
	if err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	if previous.Version != command.ExpectedPreviousVersion || oldWorkspace.Version != command.ExpectedWorkspaceVersion ||
		replacement.Version != command.ExpectedReplacementVersion || replacement.Status != DistributionDraft || previous.Status == DistributionSuperseded || !previous.Deadline.After(command.Now) {
		return DistributionBundle{}, DistributionBundle{}, ErrSupersessionPreviewMismatch
	}
	if err := verifyPostgresReplacementRoutes(ctx, tx, replacement, command.Now); err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	if err := insertPostgresSupersessionCarries(ctx, tx, previous.ID, newWorkspace, command); err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	newWorkspace.Version += int64(len(command.Carries))
	newWorkspace.UpdatedAt = command.Now.UTC()
	if len(command.Carries) > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE capture_response_workspaces SET version=$5,updated_at=$6
			WHERE id=$1::uuid AND tenant_id=$2::uuid AND legal_entity_id=$3::uuid AND distribution_id=$4::uuid`,
			newWorkspace.ID, newWorkspace.TenantID, newWorkspace.LegalEntityID, newWorkspace.DistributionID, newWorkspace.Version, newWorkspace.UpdatedAt); err != nil {
			return DistributionBundle{}, DistributionBundle{}, err
		}
	}

	previous.Status = DistributionSuperseded
	previous.Version++
	previous.UpdatedAt = command.Now.UTC()
	if _, err := tx.Exec(ctx, `
		UPDATE capture_form_distributions SET status='SUPERSEDED',version=$5,updated_at=$6
		WHERE id=$1::uuid AND tenant_id=$2::uuid AND legal_entity_id=$3::uuid AND version=$4`,
		previous.ID, previous.TenantID, previous.LegalEntityID, command.ExpectedPreviousVersion, previous.Version, previous.UpdatedAt); err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	oldWorkspace.Status = ResponseWorkspaceLocked
	oldWorkspace.Version++
	oldWorkspace.UpdatedAt = command.Now.UTC()
	if _, err := tx.Exec(ctx, `
		UPDATE capture_response_workspaces SET status='LOCKED',version=$5,updated_at=$6
		WHERE id=$1::uuid AND tenant_id=$2::uuid AND legal_entity_id=$3::uuid AND distribution_id=$4::uuid`,
		oldWorkspace.ID, oldWorkspace.TenantID, oldWorkspace.LegalEntityID, oldWorkspace.DistributionID, oldWorkspace.Version, oldWorkspace.UpdatedAt); err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE capture_requests SET status='CANCELLED',version=version+1,updated_at=$3
		WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid AND status NOT IN ('CANCELLED','EXPIRED')`, previous.TenantID, previous.ID, command.Now.UTC()); err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE capture_access_routes SET revoked_at=COALESCE(revoked_at,$3) WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid`, previous.TenantID, previous.ID, command.Now.UTC()); err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE capture_distribution_sessions SET revoked_at=COALESCE(revoked_at,$3) WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid`, previous.TenantID, previous.ID, command.Now.UTC()); err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}

	replacement.Status = DistributionOpen
	replacement.Version++
	replacement.UpdatedAt = command.Now.UTC()
	if _, err := tx.Exec(ctx, `
		UPDATE capture_form_distributions SET status='OPEN',version=$5,updated_at=$6
		WHERE id=$1::uuid AND tenant_id=$2::uuid AND legal_entity_id=$3::uuid AND version=$4 AND status='DRAFT'`,
		replacement.ID, replacement.TenantID, replacement.LegalEntityID, command.ExpectedReplacementVersion, replacement.Version, replacement.UpdatedAt); err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	if err := appendPostgresSupersessionEvent(ctx, tx, previous, replacement.ID, command.ActorID, command.Now); err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	if err := appendPostgresSupersessionEvent(ctx, tx, replacement, previous.ID, command.ActorID, command.Now); err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	previousBundle, err := store.GetDistribution(ctx, previous.TenantID, previous.LegalEntityID, previous.ID)
	if err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	replacementBundle, err := store.GetDistribution(ctx, replacement.TenantID, replacement.LegalEntityID, replacement.ID)
	if err != nil {
		return DistributionBundle{}, DistributionBundle{}, err
	}
	return previousBundle, replacementBundle, nil
}

func lockPostgresSupersessionWorkspace(ctx context.Context, tx pgx.Tx, distribution FormDistribution) (ResponseWorkspace, error) {
	return scanWorkspace(tx.QueryRow(ctx, `
		SELECT id::text,tenant_id::text,legal_entity_id::text,distribution_id::text,status,version,created_at,updated_at
		FROM capture_response_workspaces
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND distribution_id=$3::uuid
		FOR UPDATE`, distribution.TenantID, distribution.LegalEntityID, distribution.ID))
}

func verifyPostgresReplacementRoutes(ctx context.Context, tx pgx.Tx, replacement FormDistribution, now time.Time) error {
	var externalTO, activeRoutes int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM capture_distribution_recipients
		WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid AND role='TO' AND recipient_type='EXTERNAL_AUDIENCE' AND state<>'REVOKED'`,
		replacement.TenantID, replacement.ID).Scan(&externalTO); err != nil {
		return err
	}
	if externalTO == 0 {
		return nil
	}
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM capture_access_routes
		WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid AND revoked_at IS NULL AND expires_at>$3`,
		replacement.TenantID, replacement.ID, now.UTC()).Scan(&activeRoutes); err != nil {
		return err
	}
	required := externalTO
	if replacement.AccessPolicy == AccessSharedEmailOTP {
		required = 1
	}
	if activeRoutes < required {
		return ErrDistributionAccessUnavailable
	}
	return nil
}

func insertPostgresSupersessionCarries(ctx context.Context, tx pgx.Tx, previousDistributionID string, workspace ResponseWorkspace, command supersessionCommit) error {
	version := workspace.Version
	for _, carry := range command.Carries {
		editID, err := id.NewUUIDv7()
		if err != nil {
			return err
		}
		baseVersion := version
		version++
		patchJSON, err := json.Marshal(postgresWorkspaceEditPatch{
			FieldID: carry.FieldID, Value: carry.Value, PresentationMode: command.PresentationMode,
			Assurance: carry.Assurance, CarriedFromDistributionID: previousDistributionID,
		})
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO capture_response_workspace_edits(
				id,tenant_id,legal_entity_id,distribution_id,workspace_id,recipient_id,request_id,base_version,result_version,patch,created_at
			) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7::uuid,$8,$9,$10::jsonb,$11)`,
			editID, workspace.TenantID, workspace.LegalEntityID, workspace.DistributionID, workspace.ID,
			carry.RecipientID, carry.RequestID, baseVersion, version, string(patchJSON), command.Now.UTC()); err != nil {
			return err
		}
	}
	return nil
}

func appendPostgresSupersessionEvent(ctx context.Context, tx pgx.Tx, distribution FormDistribution, counterpartID, actorID string, now time.Time) error {
	eventType := "FORM_DISTRIBUTION_OPEN"
	payload := map[string]any{"version": distribution.Version, "status": distribution.Status, "supersedes_distribution_id": counterpartID}
	if distribution.Status == DistributionSuperseded {
		eventType = "FORM_DISTRIBUTION_SUPERSEDED"
		payload = map[string]any{"version": distribution.Version, "status": distribution.Status, "superseded_by_distribution_id": counterpartID}
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO capture_distribution_events(tenant_id,legal_entity_id,distribution_id,distribution_version,event_type,payload,actor_id,occurred_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6::jsonb,NULLIF($7,'')::uuid,$8)`,
		distribution.TenantID, distribution.LegalEntityID, distribution.ID, distribution.Version, eventType, string(payloadJSON), actorID, now.UTC()); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
		VALUES($1::uuid,'FORM_DISTRIBUTION',$2::uuid,$3,$4::jsonb,$5,$5,$5)`,
		distribution.TenantID, distribution.ID, eventType, string(payloadJSON), now.UTC())
	return err
}

func validPersistedWorkspaceEditOrigin(patch postgresWorkspaceEditPatch) bool {
	return (patch.SessionID != "" && patch.RouteID != "") || patch.CarriedFromDistributionID != ""
}

var _ distributionSupersessionStore = (*PostgresDistributionStore)(nil)
