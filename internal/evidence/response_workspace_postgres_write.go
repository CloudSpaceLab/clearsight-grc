//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
)

func insertPostgresWorkspaceEdits(ctx context.Context, tx pgx.Tx, view *ResponseWorkspaceView, command workspaceSaveCommand, changes []FieldEdit) error {
	for _, edit := range changes {
		editID, err := id.NewUUIDv7()
		if err != nil {
			return ErrWorkspaceUnavailable
		}
		baseVersion := view.Workspace.Version
		resultVersion := baseVersion + 1
		patch := postgresWorkspaceEditPatch{
			FieldID: edit.FieldID, Value: edit.Value, PresentationMode: command.Input.PresentationMode,
			SessionID: command.Session.ID, RouteID: command.Session.RouteID, Assurance: command.Session.Assurance,
		}
		patchJSON, err := json.Marshal(patch)
		if err != nil {
			return ErrWorkspaceUnavailable
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO capture_response_workspace_edits(
				id,tenant_id,legal_entity_id,distribution_id,workspace_id,recipient_id,request_id,
				base_version,result_version,patch,created_at
			) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7::uuid,$8,$9,$10::jsonb,$11)`,
			editID, command.Session.TenantID, command.Session.LegalEntityID, command.Session.DistributionID,
			view.Workspace.ID, command.Session.RecipientID, command.Session.RequestID,
			baseVersion, resultVersion, string(patchJSON), command.Now.UTC()); err != nil {
			return ErrWorkspaceUnavailable
		}
		applyWorkspaceEdit(view.Answers, edit)
		view.PresentationMode = command.Input.PresentationMode
		view.FieldSequences[edit.FieldID] = resultVersion
		view.FieldProvenance[edit.FieldID] = WorkspaceFieldProvenance{
			RecipientID: command.Session.RecipientID, RequestID: command.Session.RequestID,
			Assurance: command.Session.Assurance, Sequence: resultVersion, UpdatedAt: command.Now.UTC(),
		}
		view.Workspace.Version = resultVersion
		view.Workspace.UpdatedAt = command.Now.UTC()
	}
	return nil
}

func updatePostgresWorkspaceRow(ctx context.Context, tx pgx.Tx, workspace ResponseWorkspace) error {
	tag, err := tx.Exec(ctx, `
		UPDATE capture_response_workspaces
		SET version=$5,updated_at=$6
		WHERE id=$1::uuid AND tenant_id=$2::uuid AND legal_entity_id=$3::uuid AND distribution_id=$4::uuid
		  AND status='OPEN' AND version<=$5`,
		workspace.ID, workspace.TenantID, workspace.LegalEntityID, workspace.DistributionID,
		workspace.Version, workspace.UpdatedAt.UTC())
	if err != nil || tag.RowsAffected() != 1 {
		return ErrWorkspaceUnavailable
	}
	return nil
}

func insertPostgresWorkspaceSubmission(ctx context.Context, tx pgx.Tx, command workspaceSubmitCommand, view ResponseWorkspaceView, submissionID string, answers map[string]formcontract.AnswerValue) (SubmissionReceipt, error) {
	artifactIDs, err := submissionArtifactIDs(command.Request, answers)
	if err != nil {
		return SubmissionReceipt{}, err
	}
	if len(artifactIDs) > 0 {
		var eligible int
		if err := tx.QueryRow(ctx, `
			SELECT count(DISTINCT a.id)
			FROM capture_artifacts a
			JOIN capture_distribution_recipients r
			  ON r.request_id=a.request_id AND r.tenant_id=a.tenant_id
			WHERE a.tenant_id=$1::uuid AND a.id::text=ANY($2::text[])
			  AND r.distribution_id=$3::uuid AND r.legal_entity_id=$4::uuid AND r.role='TO'
			  AND r.state NOT IN ('REVOKED','COMPLETED')`,
			command.Session.TenantID, artifactIDs, command.Session.DistributionID, command.Session.LegalEntityID).Scan(&eligible); err != nil || eligible != len(artifactIDs) {
			return SubmissionReceipt{}, ErrWorkspaceUnavailable
		}
	}
	answersJSON, err := json.Marshal(answers)
	if err != nil {
		return SubmissionReceipt{}, ErrWorkspaceUnavailable
	}
	provenanceJSON, err := json.Marshal(respondentWorkspaceProvenance(answers))
	if err != nil {
		return SubmissionReceipt{}, ErrWorkspaceUnavailable
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO capture_submissions(
			id,tenant_id,request_id,session_id,submitted_by,channel,answers,answer_provenance,submitted_at,distribution_id
		) VALUES($1::uuid,$2::uuid,$3::uuid,NULL,NULL,'MAGIC_LINK',$4::jsonb,$5::jsonb,$6,$7::uuid)`,
		submissionID, command.Session.TenantID, command.Session.RequestID,
		string(answersJSON), string(provenanceJSON), command.Now.UTC(), command.Session.DistributionID); err != nil {
		return SubmissionReceipt{}, ErrWorkspaceUnavailable
	}
	if len(artifactIDs) > 0 {
		tag, err := tx.Exec(ctx, `
			UPDATE capture_artifacts a
			SET submission_id=COALESCE(a.submission_id,$3::uuid)
			FROM capture_distribution_recipients r
			WHERE a.tenant_id=$1::uuid AND a.id::text=ANY($2::text[])
			  AND r.request_id=a.request_id AND r.tenant_id=a.tenant_id
			  AND r.distribution_id=$4::uuid AND r.legal_entity_id=$5::uuid AND r.role='TO'`,
			command.Session.TenantID, artifactIDs, submissionID, command.Session.DistributionID, command.Session.LegalEntityID)
		if err != nil || tag.RowsAffected() != int64(len(artifactIDs)) {
			return SubmissionReceipt{}, ErrWorkspaceUnavailable
		}
	}
	return SubmissionReceipt{
		SubmissionID: submissionID, RequestID: command.Session.RequestID, Status: command.Request.Status,
		SubmittedAt: command.Now.UTC(), Version: command.Request.Version,
	}, nil
}

func insertPostgresResponseRevision(ctx context.Context, tx pgx.Tx, revision ResponseRevision, previous *ResponseRevision) error {
	if previous != nil {
		tag, err := tx.Exec(ctx, `
			UPDATE capture_response_revisions SET is_current=false
			WHERE id=$1::uuid AND tenant_id=$2::uuid AND workspace_id=$3::uuid AND is_current`,
			previous.ID, previous.TenantID, previous.WorkspaceID)
		if err != nil || tag.RowsAffected() != 1 {
			return ErrWorkspaceUnavailable
		}
	}
	signoffJSON, err := json.Marshal(revision.SignoffSummary)
	if err != nil {
		return ErrWorkspaceUnavailable
	}
	criticalJSON, err := json.Marshal(revision.CriticalFieldResults)
	if err != nil {
		return ErrWorkspaceUnavailable
	}
	var score any
	if revision.ComplianceScore != nil {
		score = *revision.ComplianceScore
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO capture_response_revisions(
			id,tenant_id,legal_entity_id,distribution_id,workspace_id,submission_id,revision,supersedes_revision_id,
			achieved_assurance,signoff_summary,compliance_score,scored_weight_coverage,state,critical_field_results,
			scoring_policy_version,is_current,created_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7,NULLIF($8,'')::uuid,
			$9,$10::jsonb,$11,$12,$13,$14::jsonb,$15,true,$16)`,
		revision.ID, revision.TenantID, revision.LegalEntityID, revision.DistributionID, revision.WorkspaceID,
		revision.SubmissionID, revision.Revision, revision.SupersedesRevisionID, revision.AchievedAssurance,
		string(signoffJSON), score, revision.ScoredWeightCoverage, revision.State, string(criticalJSON),
		revision.ScoringPolicyVersion, revision.CreatedAt.UTC()); err != nil {
		return ErrWorkspaceUnavailable
	}
	return nil
}

func appendPostgresWorkspaceEvent(ctx context.Context, tx pgx.Tx, session DistributionAccessSession, distributionVersion int64, eventType string, version int64, now time.Time) error {
	eventType = fmt.Sprintf("%s_%d", eventType, version)
	payload := map[string]any{
		"version": version, "workspace_event": eventType,
		"recipient_id": session.RecipientID, "request_id": session.RequestID,
		"assurance": session.Assurance,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return ErrWorkspaceUnavailable
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO capture_distribution_events(
			tenant_id,legal_entity_id,distribution_id,distribution_version,event_type,payload,actor_id,occurred_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6::jsonb,NULL,$7)`,
		session.TenantID, session.LegalEntityID, session.DistributionID, distributionVersion, eventType, string(payloadJSON), now.UTC()); err != nil {
		return ErrWorkspaceUnavailable
	}
	baseEventType := eventType
	for index := len(eventType) - 1; index >= 0; index-- {
		if eventType[index] == '_' {
			baseEventType = eventType[:index]
			break
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO outbox_events(
			tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at
		) VALUES($1::uuid,'FORM_DISTRIBUTION',$2::uuid,$3,$4::jsonb,$5,$5,$5)`,
		session.TenantID, session.DistributionID, baseEventType, string(payloadJSON), now.UTC()); err != nil {
		return ErrWorkspaceUnavailable
	}
	return nil
}
