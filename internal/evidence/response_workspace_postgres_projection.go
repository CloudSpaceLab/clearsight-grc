//go:build postgres

package evidence

import (
	"context"
	"database/sql"
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
		  AND d.status='OPEN' AND d.deadline>$9 AND d.route_expires_at>$9
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
	var value ResponseRevision
	var signoffJSON, criticalJSON []byte
	var score sql.NullFloat64
	err := tx.QueryRow(ctx, `
		SELECT id::text,tenant_id::text,legal_entity_id::text,distribution_id::text,workspace_id::text,submission_id::text,
		       revision,COALESCE(supersedes_revision_id::text,''),achieved_assurance,signoff_summary,compliance_score,
		       scored_weight_coverage,state,critical_field_results,scoring_policy_version,is_current,created_at
		FROM capture_response_revisions
		WHERE tenant_id=$1::uuid AND workspace_id=$2::uuid AND is_current`, tenantID, workspaceID).Scan(
		&value.ID, &value.TenantID, &value.LegalEntityID, &value.DistributionID, &value.WorkspaceID, &value.SubmissionID,
		&value.Revision, &value.SupersedesRevisionID, &value.AchievedAssurance, &signoffJSON, &score,
		&value.ScoredWeightCoverage, &value.State, &criticalJSON, &value.ScoringPolicyVersion, &value.Current, &value.CreatedAt,
	)
	if err != nil {
		return ResponseRevision{}, err
	}
	if err := json.Unmarshal(signoffJSON, &value.SignoffSummary); err != nil {
		return ResponseRevision{}, err
	}
	if err := json.Unmarshal(criticalJSON, &value.CriticalFieldResults); err != nil {
		return ResponseRevision{}, err
	}
	if score.Valid {
		value.ComplianceScore = &score.Float64
	}
	return value, nil
}
