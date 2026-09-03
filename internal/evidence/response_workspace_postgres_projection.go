//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/jackc/pgx/v5"
)

type postgresWorkspaceState struct {
	View                ResponseWorkspaceView
	DistributionVersion int64
}

type postgresWorkspaceEditPatch struct {
	FieldID                   string                        `json:"field_id"`
	Value                     formcontract.AnswerValue      `json:"value"`
	PresentationMode          formcontract.PresentationMode `json:"presentation_mode"`
	SessionID                 string                        `json:"session_id,omitempty"`
	RouteID                   string                        `json:"route_id,omitempty"`
	Assurance                 AccessAssurance               `json:"assurance"`
	CarriedFromDistributionID string                        `json:"carried_from_distribution_id,omitempty"`
}

func loadPostgresWorkspaceState(ctx context.Context, tx pgx.Tx, session DistributionAccessSession, request Request, now time.Time, lock bool) (postgresWorkspaceState, error) {
	if request.ID != session.RequestID || request.TenantID != session.TenantID || request.LegalEntityID != session.LegalEntityID {
		return postgresWorkspaceState{}, ErrWorkspaceUnavailable
	}
	lockSQL := ""
	if lock {
		lockSQL = " FOR UPDATE OF w,d,s,ar,er,r"
	}
	var workspace ResponseWorkspace
	var distributionVersion int64
	err := tx.QueryRow(ctx, `
		SELECT w.id::text,w.tenant_id::text,w.legal_entity_id::text,w.distribution_id::text,
		       w.status,w.version,w.created_at,w.updated_at,d.version
		FROM capture_response_workspaces w
		JOIN capture_form_distributions d
		  ON d.id=w.distribution_id AND d.tenant_id=w.tenant_id AND d.legal_entity_id=w.legal_entity_id
		JOIN capture_distribution_sessions s
		  ON s.distribution_id=d.id AND s.tenant_id=d.tenant_id AND s.legal_entity_id=d.legal_entity_id
		JOIN capture_access_routes ar
		  ON ar.id=s.route_id AND ar.tenant_id=s.tenant_id AND ar.legal_entity_id=s.legal_entity_id AND ar.distribution_id=s.distribution_id
		JOIN capture_distribution_recipients r
		  ON r.id=s.recipient_id AND r.tenant_id=s.tenant_id AND r.legal_entity_id=s.legal_entity_id AND r.distribution_id=s.distribution_id
		JOIN capture_requests er
		  ON er.id=s.request_id AND er.tenant_id=s.tenant_id AND er.legal_entity_id=s.legal_entity_id AND er.distribution_id=s.distribution_id
		WHERE d.id=$1::uuid AND s.id=$2::uuid AND s.tenant_id=$3::uuid AND s.legal_entity_id=$4::uuid
		  AND s.recipient_id=$5::uuid AND s.request_id=$6::uuid AND s.route_id=$7::uuid AND s.assurance=$8
		  AND s.revoked_at IS NULL AND s.expires_at>$9
		  AND ar.revoked_at IS NULL AND ar.expires_at>$9 AND ar.access_policy=d.access_policy
		  AND d.status='OPEN'
		  AND w.status='OPEN'
		  AND r.role='TO' AND r.state NOT IN ('REVOKED','COMPLETED') AND r.request_id=s.request_id
		  AND er.status IN ('READY','IN_PROGRESS') AND er.deadline>$9`+lockSQL,
		session.DistributionID, session.ID, session.TenantID, session.LegalEntityID,
		session.RecipientID, session.RequestID, session.RouteID, session.Assurance, now,
	).Scan(&workspace.ID, &workspace.TenantID, &workspace.LegalEntityID, &workspace.DistributionID,
		&workspace.Status, &workspace.Version, &workspace.CreatedAt, &workspace.UpdatedAt, &distributionVersion)
	if errors.Is(err, pgx.ErrNoRows) || err != nil {
		return postgresWorkspaceState{}, ErrWorkspaceUnavailable
	}

	view := workspaceDefaultView(workspace, request)
	rows, err := tx.Query(ctx, `
		SELECT e.recipient_id::text,e.request_id::text,e.base_version,e.result_version,e.patch,e.created_at
		FROM capture_response_workspace_edits e
		WHERE e.tenant_id=$1::uuid AND e.legal_entity_id=$2::uuid AND e.distribution_id=$3::uuid AND e.workspace_id=$4::uuid
		ORDER BY e.result_version,e.id`, session.TenantID, session.LegalEntityID, session.DistributionID, workspace.ID)
	if err != nil {
		return postgresWorkspaceState{}, ErrWorkspaceUnavailable
	}
	defer rows.Close()
	for rows.Next() {
		var recipientID, requestID string
		var baseVersion, resultVersion int64
		var patchJSON []byte
		var createdAt time.Time
		if err := rows.Scan(&recipientID, &requestID, &baseVersion, &resultVersion, &patchJSON, &createdAt); err != nil {
			return postgresWorkspaceState{}, ErrWorkspaceUnavailable
		}
		var patch postgresWorkspaceEditPatch
		if err := json.Unmarshal(patchJSON, &patch); err != nil || patch.FieldID == "" || resultVersion != baseVersion+1 || !validPersistedWorkspaceEditOrigin(patch) || !validAccessAssurance(patch.Assurance) {
			return postgresWorkspaceState{}, fmt.Errorf("%w: invalid persisted workspace edit", ErrWorkspaceUnavailable)
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
		return postgresWorkspaceState{}, ErrWorkspaceUnavailable
	}

	revision, err := loadCurrentPostgresResponseRevision(ctx, tx, session.TenantID, workspace.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return postgresWorkspaceState{}, ErrWorkspaceUnavailable
	}
	if err == nil {
		view.CurrentRevision = &revision
	}
	return postgresWorkspaceState{View: view, DistributionVersion: distributionVersion}, nil
}

func loadCurrentPostgresResponseRevision(ctx context.Context, tx pgx.Tx, tenantID, workspaceID string) (ResponseRevision, error) {
	value, err := scanPostgresResponseRevision(tx.QueryRow(ctx, `
		SELECT `+responseRevisionProjection+`
		FROM capture_response_revisions r
		WHERE r.tenant_id=$1::uuid AND r.workspace_id=$2::uuid AND r.is_current`, tenantID, workspaceID))
	return value, err
}
