//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// lockInternalWorkspaceTenant must be called before any workspace or
// eligibility row locks. It pins the verified actor's tenant to its UUID for
// the rest of the transaction. This is only one part of the authority fence
// rollout; private workspace commands remain unavailable to HTTP callers.
func lockInternalWorkspaceTenant(ctx context.Context, tx pgx.Tx, now time.Time) (identity.Actor, int64, error) {
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(now) != nil || !strings.EqualFold(strings.TrimSpace(actor.Kind), "PERSON") || actor.LegalEntityID == "*" || tx == nil {
		return identity.Actor{}, 0, ErrWorkspaceUnavailable
	}
	var candidate pgtype.UUID
	if err := candidate.Scan(actor.TenantID); err != nil {
		candidate = pgtype.UUID{}
	}
	var tenant pgtype.UUID
	if err := tx.QueryRow(ctx, `SELECT CASE WHEN count(*)=1 THEN min(id::text)::uuid END FROM tenants WHERE id=$2 OR slug=$1`, actor.TenantID, candidate).Scan(&tenant); err != nil {
		return identity.Actor{}, 0, ErrWorkspaceUnavailable
	}
	epoch, err := authority.LockTenantAuthorityForCommand(ctx, tx, tenant)
	if err != nil {
		return identity.Actor{}, 0, ErrWorkspaceUnavailable
	}
	value, err := tenant.Value()
	if err != nil {
		return identity.Actor{}, 0, ErrWorkspaceUnavailable
	}
	canonical, ok := value.(string)
	if !ok || canonical == "" {
		return identity.Actor{}, 0, ErrWorkspaceUnavailable
	}
	actor.TenantID = canonical
	return actor, epoch, nil
}

func (store *PostgresDistributionStore) getInternalWorkspace(ctx context.Context, requestID string, now time.Time) (Request, ResponseWorkspaceView, error) {
	return store.getInternalPostgresWorkspace(ctx, requestID, now)
}
func (store *PostgresDistributionStore) saveInternalWorkspace(ctx context.Context, requestID string, input SaveWorkspaceInput, now time.Time) (ResponseWorkspaceView, error) {
	return store.saveInternalPostgresWorkspace(ctx, requestID, input, now)
}
func (store *PostgresDistributionStore) submitInternalWorkspace(ctx context.Context, requestID string, input SubmitWorkspaceInput, now time.Time) (WorkspaceSubmissionResult, error) {
	return store.submitInternalPostgresWorkspace(ctx, requestID, input, now)
}

// getInternalPostgresWorkspace is deliberately private. It must not be
// exposed until save and submit have matching transaction-time checks.
func (store *PostgresDistributionStore) getInternalPostgresWorkspace(ctx context.Context, requestID string, now time.Time) (Request, ResponseWorkspaceView, error) {
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(now) != nil || !strings.EqualFold(strings.TrimSpace(actor.Kind), "PERSON") || actor.LegalEntityID == "*" || strings.TrimSpace(requestID) == "" || store == nil || store.repo == nil || store.repo.pool == nil {
		return Request{}, ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return Request{}, ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	defer tx.Rollback(ctx)
	request, state, err := loadInternalPostgresWorkspaceState(ctx, tx, actor, requestID, now, false)
	if err != nil {
		return Request{}, ResponseWorkspaceView{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Request{}, ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	return RespondentRequest(request), cloneResponseWorkspaceView(state.View), nil
}

// loadInternalPostgresWorkspaceState does not accept a route, token or
// external session. Eligibility is checked within the same transaction.
func loadInternalPostgresWorkspaceState(ctx context.Context, tx pgx.Tx, actor identity.Actor, requestID string, now time.Time, lock bool) (Request, postgresWorkspaceState, error) {
	if tx == nil || actor.Valid(now) != nil || !strings.EqualFold(strings.TrimSpace(actor.Kind), "PERSON") || actor.LegalEntityID == "*" || strings.TrimSpace(requestID) == "" {
		return Request{}, postgresWorkspaceState{}, ErrWorkspaceUnavailable
	}
	lockSQL := ""
	if lock {
		lockSQL = " FOR UPDATE OF w,d,r,er,p"
	}
	var workspace ResponseWorkspace
	var distributionVersion int64
	var recipientID string
	err := tx.QueryRow(ctx, `
		SELECT w.id::text,w.tenant_id::text,w.legal_entity_id::text,w.distribution_id::text,
		       w.status,w.version,w.created_at,w.updated_at,d.version,r.id::text
		FROM capture_requests er
		JOIN tenants t ON t.id=er.tenant_id
		JOIN capture_form_distributions d ON d.id=er.distribution_id AND d.tenant_id=er.tenant_id AND d.legal_entity_id=er.legal_entity_id
		JOIN capture_response_workspaces w ON w.distribution_id=d.id AND w.tenant_id=d.tenant_id AND w.legal_entity_id=d.legal_entity_id
		JOIN capture_distribution_recipients r ON r.request_id=er.id AND r.distribution_id=d.id AND r.tenant_id=d.tenant_id AND r.legal_entity_id=d.legal_entity_id
		JOIN principals p ON p.id=r.principal_id AND p.tenant_id=r.tenant_id
		WHERE er.id::text=$1 AND (t.id::text=$2 OR t.slug=$2) AND er.legal_entity_id::text=$3 AND p.id::text=$4
		  AND r.role='TO' AND r.recipient_type='INTERNAL_PRINCIPAL' AND r.state IN ('PENDING','DELIVERED','VERIFIED')
		  AND er.recipient_type='INTERNAL_PRINCIPAL' AND er.recipient_principal_id=p.id AND er.recipient_state='ASSIGNED'
		  AND er.status IN ('READY','IN_PROGRESS') AND er.deadline>$5 AND d.status='OPEN' AND d.deadline>$5 AND w.status='OPEN'
		  AND er.subject_type=d.subject_type AND er.subject_id=d.subject_id::text
		  AND er.form_template_id=d.form_template_id AND er.form_template_version=d.form_template_version
		  AND `+internalRecipientEligibilityPredicate("p", "d.tenant_id", "d.legal_entity_id", "d.subject_type", "d.subject_id::text")+lockSQL,
		requestID, actor.TenantID, actor.LegalEntityID, actor.PrincipalID, now.UTC(),
	).Scan(&workspace.ID, &workspace.TenantID, &workspace.LegalEntityID, &workspace.DistributionID, &workspace.Status, &workspace.Version, &workspace.CreatedAt, &workspace.UpdatedAt, &distributionVersion, &recipientID)
	if err != nil {
		return Request{}, postgresWorkspaceState{}, ErrWorkspaceUnavailable
	}
	if lock {
		// The eligibility predicate reads additional rows which the main FOR UPDATE
		// does not lock. Hold those rows through commit, then re-evaluate against
		// a fresh READ COMMITTED snapshot after any concurrent revocation finishes.
		if err := lockInternalEligibilityRows(ctx, tx, workspace.TenantID, workspace.LegalEntityID, actor.PrincipalID, requestID); err != nil {
			return Request{}, postgresWorkspaceState{}, ErrWorkspaceUnavailable
		}
		return loadInternalPostgresWorkspaceState(ctx, tx, actor, requestID, now, false)
	}
	request, err := scanRequest(tx.QueryRow(ctx, requestSelect+` WHERE er.id::text=$1 AND er.tenant_id::text=$2 AND er.legal_entity_id::text=$3`, requestID, workspace.TenantID, workspace.LegalEntityID))
	if err != nil {
		return Request{}, postgresWorkspaceState{}, ErrWorkspaceUnavailable
	}
	state, err := projectPostgresWorkspace(ctx, tx, workspace, request, workspace.TenantID, workspace.LegalEntityID, workspace.DistributionID, distributionVersion)
	if err != nil {
		return Request{}, postgresWorkspaceState{}, err
	}
	state.RecipientID = recipientID
	return request, state, nil
}

func lockInternalEligibilityRows(ctx context.Context, tx pgx.Tx, tenant, entity, principal, requestID string) error {
	queries := []struct {
		sql  string
		args []any
	}{
		{`SELECT 1 FROM org_positions WHERE tenant_id=$1::uuid AND occupant_principal_id=$3::uuid AND (legal_entity_id IS NULL OR legal_entity_id=$2::uuid) FOR SHARE`, []any{tenant, entity, principal}},
		{`SELECT 1 FROM scim_users u JOIN scim_sources s ON s.tenant_id=u.tenant_id AND s.id=u.source_id JOIN directory_group_members m ON m.tenant_id=u.tenant_id AND m.scim_user_id=u.id JOIN directory_groups g ON g.tenant_id=m.tenant_id AND g.id=m.group_id AND g.source_id=u.source_id JOIN directory_group_role_bindings b ON b.tenant_id=g.tenant_id AND b.group_id=g.id JOIN role_templates rt ON rt.tenant_id=b.tenant_id AND rt.id=b.role_template_id WHERE u.tenant_id=$1::uuid AND u.principal_id=$3::uuid AND b.legal_entity_id=$2::uuid FOR SHARE OF u,s,m,g,b,rt`, []any{tenant, entity, principal}},
		{`SELECT 1 FROM programs subject JOIN capture_requests r ON r.subject_type='PROGRAM' AND r.subject_id=subject.id::text WHERE subject.tenant_id=$1::uuid AND subject.legal_entity_id=$2::uuid AND r.id=$3::uuid FOR SHARE OF subject`, []any{tenant, entity, requestID}},
		{`SELECT 1 FROM matters subject JOIN capture_requests r ON r.subject_type='MATTER' AND r.subject_id=subject.id::text WHERE subject.tenant_id=$1::uuid AND subject.legal_entity_id=$2::uuid AND r.id=$3::uuid FOR SHARE OF subject`, []any{tenant, entity, requestID}},
	}
	for _, query := range queries {
		rows, err := tx.Query(ctx, query.sql, query.args...)
		if err != nil {
			return err
		}
		for rows.Next() {
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// The save remains private until submission and authenticated handlers have
// equivalent transaction-time authorization and PostgreSQL integration tests.
func (store *PostgresDistributionStore) saveInternalPostgresWorkspace(ctx context.Context, requestID string, input SaveWorkspaceInput, now time.Time) (ResponseWorkspaceView, error) {
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(now) != nil || !strings.EqualFold(strings.TrimSpace(actor.Kind), "PERSON") || actor.LegalEntityID == "*" || store == nil || store.repo == nil || store.repo.pool == nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	defer tx.Rollback(ctx)
	actor, epoch, err := lockInternalWorkspaceTenant(ctx, tx, now)
	if err != nil {
		return ResponseWorkspaceView{}, err
	}
	request, state, err := loadInternalPostgresWorkspaceState(ctx, tx, actor, requestID, now, true)
	if err != nil {
		return ResponseWorkspaceView{}, err
	}
	authorityRequest := request
	// The scoped lookup already matched the verified tenant slug or UUID to
	// this request's tenant. Authority resolves using that verified identity.
	authorityRequest.TenantID = actor.TenantID
	authorityRoute, err := authorizeInternalWorkspaceLocked(authority.WithPostgresTransaction(ctx, tx), actor, authorityRequest)
	if err != nil {
		return ResponseWorkspaceView{}, err
	}
	if err := normalizeSaveWorkspaceInput(&input, request); err != nil {
		return ResponseWorkspaceView{}, err
	}
	if input.ExpectedVersion > state.View.Workspace.Version {
		return ResponseWorkspaceView{}, WorkspaceConflict{CurrentVersion: state.View.Workspace.Version}
	}
	if conflict := postgresWorkspaceConflict(state.View, input.Edits); conflict != nil {
		return ResponseWorkspaceView{}, *conflict
	}
	answers := cloneAnswerValues(state.View.Answers)
	changes := make([]FieldEdit, 0, len(input.Edits))
	for _, edit := range input.Edits {
		if !answerValueEqual(answers[edit.FieldID], edit.Value) {
			applyWorkspaceEdit(answers, edit)
			changes = append(changes, edit)
		}
	}
	if err := validateWorkspaceAnswerSet(ctx, store.repo, request, answers, false, func(ctx context.Context, tenantID, _ string, artifactID string) (Artifact, error) {
		artifact, err := loadScopedPostgresWorkspaceArtifact(ctx, tx, state.View.Workspace.DistributionID, state.View.Workspace.LegalEntityID, tenantID, "", artifactID, false)
		return withArtifactUsePolicy(artifact, store.repo.demoUnscannedAllowed), err
	}, now); err != nil {
		return ResponseWorkspaceView{}, err
	}
	for _, edit := range changes {
		editID, err := id.NewUUIDv7()
		if err != nil {
			return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
		}
		base := state.View.Workspace.Version
		next := base + 1
		patch, err := json.Marshal(postgresWorkspaceEditPatch{FieldID: edit.FieldID, Value: edit.Value, PresentationMode: input.PresentationMode, Origin: "INTERNAL", PrincipalID: actor.PrincipalID, Assurance: AssuranceInternalAuthenticated})
		if err != nil {
			return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
		}
		if _, err := tx.Exec(ctx, `INSERT INTO capture_response_workspace_edits(id,tenant_id,legal_entity_id,distribution_id,workspace_id,recipient_id,request_id,base_version,result_version,patch,created_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7::uuid,$8,$9,$10::jsonb,$11)`, editID, state.View.Workspace.TenantID, state.View.Workspace.LegalEntityID, state.View.Workspace.DistributionID, state.View.Workspace.ID, state.RecipientID, request.ID, base, next, string(patch), now.UTC()); err != nil {
			return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
		}
		applyWorkspaceEdit(state.View.Answers, edit)
		state.View.PresentationMode = input.PresentationMode
		state.View.FieldSequences[edit.FieldID] = next
		state.View.FieldProvenance[edit.FieldID] = WorkspaceFieldProvenance{RecipientID: state.RecipientID, RequestID: request.ID, Assurance: AssuranceInternalAuthenticated, Sequence: next, UpdatedAt: now.UTC()}
		state.View.Workspace.Version = next
		state.View.Workspace.UpdatedAt = now.UTC()
	}
	if len(changes) > 0 {
		if err := updatePostgresWorkspaceRow(ctx, tx, state.View.Workspace); err != nil {
			return ResponseWorkspaceView{}, err
		}
		if err := appendInternalPostgresWorkspaceEvent(ctx, tx, actor, state, request.ID, "FORM_RESPONSE_WORKSPACE_SAVED", authorityRoute, epoch, state.View.Workspace.Version, now); err != nil {
			return ResponseWorkspaceView{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	return cloneResponseWorkspaceView(state.View), nil
}

func appendInternalPostgresWorkspaceEvent(ctx context.Context, tx pgx.Tx, actor identity.Actor, state postgresWorkspaceState, requestID, eventType, authorityRoute string, epoch, version int64, now time.Time) error {
	payload, err := json.Marshal(map[string]any{"version": version, "workspace_event": fmt.Sprintf("%s_%d", eventType, version), "recipient_id": state.RecipientID, "request_id": requestID, "assurance": AssuranceInternalAuthenticated, "authority_route": authorityRoute, "authority_epoch": epoch})
	if err != nil {
		return ErrWorkspaceUnavailable
	}
	if _, err := tx.Exec(ctx, `INSERT INTO capture_distribution_events(tenant_id,legal_entity_id,distribution_id,distribution_version,event_type,payload,actor_id,occurred_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6::jsonb,$7::uuid,$8)`, state.View.Workspace.TenantID, state.View.Workspace.LegalEntityID, state.View.Workspace.DistributionID, state.DistributionVersion, fmt.Sprintf("%s_%d", eventType, version), string(payload), actor.PrincipalID, now.UTC()); err != nil {
		return ErrWorkspaceUnavailable
	}
	if _, err := tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,'FORM_DISTRIBUTION',$2::uuid,$3,$4::jsonb,$5,$5,$5)`, state.View.Workspace.TenantID, state.View.Workspace.DistributionID, eventType, string(payload), now.UTC()); err != nil {
		return ErrWorkspaceUnavailable
	}
	return nil
}

// submitInternalPostgresWorkspace is private until both store implementations
// and authenticated handlers pass the full integration and authorization gate.
func (store *PostgresDistributionStore) submitInternalPostgresWorkspace(ctx context.Context, requestID string, input SubmitWorkspaceInput, now time.Time) (WorkspaceSubmissionResult, error) {
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(now) != nil || !strings.EqualFold(strings.TrimSpace(actor.Kind), "PERSON") || actor.LegalEntityID == "*" || store == nil || store.repo == nil || store.repo.pool == nil || input.ExpectedVersion < 1 {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	attestations, err := normalizeAttestationFieldIDs(input.AttestationFieldIDs)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	defer tx.Rollback(ctx)
	actor, epoch, err := lockInternalWorkspaceTenant(ctx, tx, now)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	request, state, err := loadInternalPostgresWorkspaceState(ctx, tx, actor, requestID, now, true)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	authorityRequest := request
	authorityRequest.TenantID = actor.TenantID
	authorityRoute, err := authorizeInternalWorkspaceLocked(authority.WithPostgresTransaction(ctx, tx), actor, authorityRequest)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	if input.ExpectedVersion != state.View.Workspace.Version {
		return WorkspaceSubmissionResult{}, WorkspaceConflict{CurrentVersion: state.View.Workspace.Version}
	}
	answers := cloneAnswerValues(state.View.Answers)
	request = RefreshCollectionResolutions(ctx, request, func(ctx context.Context, tenant, sourceRequest, artifactID string) (Artifact, error) {
		artifact, err := loadScopedPostgresWorkspaceArtifact(ctx, tx, state.View.Workspace.DistributionID, state.View.Workspace.LegalEntityID, tenant, sourceRequest, artifactID, true)
		return withArtifactUsePolicy(artifact, store.repo.demoUnscannedAllowed), err
	}, now)
	request, err = RefreshCollectionReviewsPostgres(ctx, tx, request)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	if err := validateWorkspaceAnswerSet(ctx, store.repo, request, answers, true, func(ctx context.Context, tenant, _ string, artifactID string) (Artifact, error) {
		artifact, err := loadScopedPostgresWorkspaceArtifact(ctx, tx, state.View.Workspace.DistributionID, state.View.Workspace.LegalEntityID, tenant, "", artifactID, false)
		return withArtifactUsePolicy(artifact, store.repo.demoUnscannedAllowed), err
	}, now); err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	revision, err := buildResponseRevision(request, AssuranceInternalAuthenticated, attestations, answers)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	artifactIDs, err := submissionArtifactIDs(request, answers)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	submissionID, err := id.NewUUIDv7()
	if err != nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	revisionID, err := id.NewUUIDv7()
	if err != nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	if len(artifactIDs) > 0 {
		var eligible int
		if err := tx.QueryRow(ctx, `SELECT count(DISTINCT a.id) FROM capture_artifacts a JOIN capture_distribution_recipients r ON r.request_id=a.request_id AND r.tenant_id=a.tenant_id WHERE a.tenant_id=$1::uuid AND a.id::text=ANY($2::text[]) AND r.distribution_id=$3::uuid AND r.legal_entity_id=$4::uuid AND r.role='TO' AND r.state<>'REVOKED'`, state.View.Workspace.TenantID, artifactIDs, state.View.Workspace.DistributionID, state.View.Workspace.LegalEntityID).Scan(&eligible); err != nil || eligible != len(artifactIDs) {
			return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
		}
	}
	answersJSON, err := json.Marshal(answers)
	if err != nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	provenanceJSON, err := json.Marshal(respondentWorkspaceProvenance(answers))
	if err != nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	if _, err := tx.Exec(ctx, `INSERT INTO capture_submissions(id,tenant_id,request_id,session_id,submitted_by,channel,answers,answer_provenance,submitted_at,distribution_id) VALUES($1::uuid,$2::uuid,$3::uuid,NULL,$4::uuid,'INTERNAL',$5::jsonb,$6::jsonb,$7,$8::uuid)`, submissionID, state.View.Workspace.TenantID, request.ID, actor.PrincipalID, string(answersJSON), string(provenanceJSON), now.UTC(), state.View.Workspace.DistributionID); err != nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	if len(artifactIDs) > 0 {
		tag, err := tx.Exec(ctx, `UPDATE capture_artifacts a SET submission_id=COALESCE(a.submission_id,$3::uuid) FROM capture_distribution_recipients r WHERE a.tenant_id=$1::uuid AND a.id::text=ANY($2::text[]) AND r.request_id=a.request_id AND r.tenant_id=a.tenant_id AND r.distribution_id=$4::uuid AND r.legal_entity_id=$5::uuid AND r.role='TO' AND r.state<>'REVOKED'`, state.View.Workspace.TenantID, artifactIDs, submissionID, state.View.Workspace.DistributionID, state.View.Workspace.LegalEntityID)
		if err != nil || tag.RowsAffected() != int64(len(artifactIDs)) {
			return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
		}
	}
	revision.ID, revision.TenantID, revision.LegalEntityID = revisionID, state.View.Workspace.TenantID, state.View.Workspace.LegalEntityID
	revision.DistributionID, revision.WorkspaceID, revision.SubmissionID = state.View.Workspace.DistributionID, state.View.Workspace.ID, submissionID
	revision.SubmittedRecipientID, revision.SubmittedRecipientPrincipalID, revision.SubmittedRecipientType = state.RecipientID, actor.PrincipalID, RecipientInternalPrincipal
	revision.Revision, revision.Current, revision.CreatedAt = 1, true, now.UTC()
	if state.View.CurrentRevision != nil {
		revision.Revision = state.View.CurrentRevision.Revision + 1
		revision.SupersedesRevisionID = state.View.CurrentRevision.ID
	}
	if revision.Score != nil {
		revision.Score.CalculatedAt = revision.CreatedAt
	}
	if err := insertPostgresResponseRevision(ctx, tx, revision, state.View.CurrentRevision); err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	state.View.Workspace.Version++
	state.View.Workspace.UpdatedAt = now.UTC()
	if err := updatePostgresWorkspaceRow(ctx, tx, state.View.Workspace); err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	if err := appendInternalPostgresWorkspaceEvent(ctx, tx, actor, state, request.ID, "FORM_RESPONSE_REVISION_SUBMITTED", authorityRoute, epoch, revision.Revision, now); err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	if err := appendInternalPostgresScoreEvent(ctx, tx, actor, state, request, revision, now); err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	return WorkspaceSubmissionResult{Workspace: state.View.Workspace, Revision: revision, Submission: SubmissionReceipt{SubmissionID: submissionID, RequestID: request.ID, Status: request.Status, SubmittedAt: now.UTC(), Version: request.Version}}, nil
}

func appendInternalPostgresScoreEvent(ctx context.Context, tx pgx.Tx, actor identity.Actor, state postgresWorkspaceState, request Request, revision ResponseRevision, now time.Time) error {
	scoreState := ResponseScoreNotConfigured
	if revision.Score != nil {
		scoreState = revision.Score.State
	}
	payload, err := json.Marshal(map[string]any{"version": revision.Revision, "response_revision_id": revision.ID, "request_id": request.ID, "request_version": request.Version, "form_template_id": request.FormTemplateID, "form_template_version": request.FormTemplateVersion, "score_state": string(scoreState)})
	if err != nil {
		return ErrWorkspaceUnavailable
	}
	if _, err := tx.Exec(ctx, `INSERT INTO capture_distribution_events(tenant_id,legal_entity_id,distribution_id,distribution_version,event_type,payload,actor_id,occurred_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6::jsonb,$7::uuid,$8)`, state.View.Workspace.TenantID, state.View.Workspace.LegalEntityID, state.View.Workspace.DistributionID, state.DistributionVersion, fmt.Sprintf("FORM_RESPONSE_SCORED_%d", revision.Revision), string(payload), actor.PrincipalID, now.UTC()); err != nil {
		return ErrWorkspaceUnavailable
	}
	if _, err := tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,'FORM_DISTRIBUTION',$2::uuid,'FORM_RESPONSE_SCORED',$3::jsonb,$4,$4,$4)`, state.View.Workspace.TenantID, state.View.Workspace.DistributionID, string(payload), now.UTC()); err != nil {
		return ErrWorkspaceUnavailable
	}
	return nil
}
