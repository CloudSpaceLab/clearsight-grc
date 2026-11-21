package evidence

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

// getInternalMemoryWorkspace is not an HTTP entry point. The authenticated
// mutation and Postgres variants must exist before this path can be exposed.
func (store *MemoryDistributionAccessStore) getInternalMemoryWorkspace(ctx context.Context, requestID string, now time.Time) (Request, ResponseWorkspaceView, error) {
	if store == nil || store.distributions == nil || store.distributions.repo == nil || strings.TrimSpace(requestID) == "" {
		return Request{}, ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	request, workspace, _, ok := store.internalMemoryAccessSnapshot(ctx, requestID, now)
	if !ok {
		return Request{}, ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	state := store.memoryWorkspaceState(workspace, request)
	state.mu.Lock()
	defer state.mu.Unlock()
	request, workspace, _, ok = store.internalMemoryAccessSnapshot(ctx, requestID, now)
	if !ok || state.workspace.ID != workspace.ID || state.workspace.Version != workspace.Version || state.workspace.Status != workspace.Status {
		return Request{}, ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	return RespondentRequest(request), cloneResponseWorkspaceView(memoryWorkspaceView(state)), nil
}

// This snapshot obtains the directory, request, recipient and distribution
// under their existing locks. It re-evaluates current eligibility, not a
// caller-provided principal or an earlier successful access decision.
func (store *MemoryDistributionAccessStore) internalMemoryAccessSnapshot(ctx context.Context, requestID string, now time.Time) (Request, ResponseWorkspace, DistributionRecipient, bool) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return Request{}, ResponseWorkspace{}, DistributionRecipient{}, false
	}
	distributions := store.distributions
	distributions.mu.RLock()
	defer distributions.mu.RUnlock()
	repo := distributions.repo
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	return internalMemoryAccessLocked(ctx, distributions, actor, requestID, now)
}

// Caller holds the distribution and repository locks in that order.
func internalMemoryAccessLocked(ctx context.Context, distributions *MemoryDistributionStore, actor identity.Actor, requestID string, now time.Time) (Request, ResponseWorkspace, DistributionRecipient, bool) {
	repo := distributions.repo
	distributionID := distributions.requestDistribution[requestID]
	distribution := distributions.distributions[distributionID]
	workspace := distributions.workspaces[distributionID]
	request, exists := repo.requests[requestID]
	if !exists {
		return Request{}, ResponseWorkspace{}, DistributionRecipient{}, false
	}
	candidate, exists := repo.candidates[actor.PrincipalID]
	eligible := exists && candidate.PrincipalID == actor.PrincipalID && candidate.TenantID == actor.TenantID && candidate.Kind == "PERSON" && candidate.Active && candidate.ReadableSubjects[strings.ToUpper(strings.TrimSpace(request.SubjectType))+":"+strings.TrimSpace(request.SubjectID)]
	withinEntity := false
	for _, entityID := range candidate.LegalEntityIDs {
		if entityID == actor.LegalEntityID {
			withinEntity = true
			break
		}
	}
	eligible = eligible && withinEntity
	for _, entry := range distributions.recipients[distributionID] {
		if internalWorkspaceAccessAllowed(ctx, distribution, workspace, entry.safe, request, eligible, now) {
			return cloneRequest(request), workspace, entry.safe, true
		}
	}
	return Request{}, ResponseWorkspace{}, DistributionRecipient{}, false
}

// saveInternalMemoryWorkspace is private until the matching Postgres and HTTP
// implementations have equivalent transactional authorization and provenance.
func (store *MemoryDistributionAccessStore) saveInternalMemoryWorkspace(ctx context.Context, requestID string, input SaveWorkspaceInput, now time.Time) (ResponseWorkspaceView, error) {
	if store == nil || store.distributions == nil || store.distributions.repo == nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	request, workspace, _, ok := store.internalMemoryAccessSnapshot(ctx, requestID, now)
	if !ok {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	if err := normalizeSaveWorkspaceInput(&input, request); err != nil {
		return ResponseWorkspaceView{}, err
	}
	state := store.memoryWorkspaceState(workspace, request)
	state.mu.Lock()
	defer state.mu.Unlock()
	distributions := store.distributions
	distributions.mu.Lock()
	defer distributions.mu.Unlock()
	repo := distributions.repo
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	actor, err := identity.Require(ctx)
	if err != nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	currentRequest, currentWorkspace, recipient, ok := internalMemoryAccessLocked(ctx, distributions, actor, requestID, now)
	if !ok || currentRequest.Version != request.Version || currentWorkspace.ID != state.workspace.ID || currentWorkspace.Version != state.workspace.Version || currentWorkspace.Status != state.workspace.Status {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	authorityRoute, err := authorizeInternalWorkspaceLocked(ctx, actor, currentRequest)
	if err != nil {
		return ResponseWorkspaceView{}, err
	}
	if input.ExpectedVersion > state.workspace.Version {
		return ResponseWorkspaceView{}, WorkspaceConflict{CurrentVersion: state.workspace.Version}
	}
	conflict := WorkspaceConflict{CurrentVersion: state.workspace.Version}
	for _, edit := range input.Edits {
		if state.sequences[edit.FieldID] != edit.BaseSequence {
			conflict.Changed = append(conflict.Changed, FieldChange{FieldID: edit.FieldID, ServerValue: state.answers[edit.FieldID], Sequence: state.sequences[edit.FieldID]})
		}
	}
	if len(conflict.Changed) > 0 {
		sort.Slice(conflict.Changed, func(i, j int) bool { return conflict.Changed[i].FieldID < conflict.Changed[j].FieldID })
		return ResponseWorkspaceView{}, conflict
	}
	merged := cloneAnswerValues(state.answers)
	changes := make([]FieldEdit, 0, len(input.Edits))
	for _, edit := range input.Edits {
		if !answerValueEqual(merged[edit.FieldID], edit.Value) {
			applyWorkspaceEdit(merged, edit)
			changes = append(changes, edit)
		}
	}
	eligibleRequests := memoryDistributionTORequestIDsLocked(distributions, currentWorkspace.DistributionID)
	if err := validateWorkspaceAnswerSet(ctx, repo, currentRequest, merged, false, func(_ context.Context, tenantID, _ string, artifactID string) (Artifact, error) {
		artifact, exists := repo.artifacts[artifactID]
		if !exists || artifact.TenantID != tenantID || !eligibleRequests[artifact.RequestID] {
			return Artifact{}, ErrNotFound
		}
		return artifact, nil
	}, now); err != nil {
		return ResponseWorkspaceView{}, err
	}
	if len(changes) == 0 {
		return cloneResponseWorkspaceView(memoryWorkspaceView(state)), nil
	}
	ids := make([]string, len(changes))
	for index := range changes {
		ids[index], err = id.NewUUIDv7()
		if err != nil {
			return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
		}
	}
	state.answers = merged
	state.mode = input.PresentationMode
	for index, edit := range changes {
		base := state.workspace.Version
		state.workspace.Version++
		sequence := state.workspace.Version
		state.sequences[edit.FieldID] = sequence
		state.provenance[edit.FieldID] = WorkspaceFieldProvenance{RecipientID: recipient.ID, RequestID: requestID, Assurance: AssuranceInternalAuthenticated, Sequence: sequence, UpdatedAt: now.UTC()}
		state.edits = append(state.edits, workspaceEditRecord{ID: ids[index], TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, DistributionID: currentWorkspace.DistributionID, WorkspaceID: state.workspace.ID, RecipientID: recipient.ID, RequestID: requestID, BaseVersion: base, ResultVersion: sequence, FieldID: edit.FieldID, Value: edit.Value, PresentationMode: input.PresentationMode, Origin: "INTERNAL", PrincipalID: actor.PrincipalID, Assurance: AssuranceInternalAuthenticated, CreatedAt: now.UTC()})
	}
	state.workspace.UpdatedAt = now.UTC()
	distributions.workspaces[currentWorkspace.DistributionID] = state.workspace
	event := distributionEvent{DistributionID: currentWorkspace.DistributionID, Version: state.workspace.Version, EventType: fmt.Sprintf("FORM_RESPONSE_WORKSPACE_SAVED_%d", state.workspace.Version), ActorID: actor.PrincipalID, OccurredAt: now.UTC(), Payload: map[string]any{"authority_route": authorityRoute}}
	distributions.events = append(distributions.events, event)
	distributions.outbox = append(distributions.outbox, event)
	return cloneResponseWorkspaceView(memoryWorkspaceView(state)), nil
}

// submitInternalMemoryWorkspace remains private until authenticated handlers
// and the PostgreSQL transaction provide equivalent authorization and audit.
func (store *MemoryDistributionAccessStore) submitInternalMemoryWorkspace(ctx context.Context, requestID string, input SubmitWorkspaceInput, now time.Time) (WorkspaceSubmissionResult, error) {
	if store == nil || store.distributions == nil || store.distributions.repo == nil || input.ExpectedVersion < 1 {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	attestations, err := normalizeAttestationFieldIDs(input.AttestationFieldIDs)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	request, workspace, _, ok := store.internalMemoryAccessSnapshot(ctx, requestID, now)
	if !ok {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	state := store.memoryWorkspaceState(workspace, request)
	state.mu.Lock()
	defer state.mu.Unlock()
	if input.ExpectedVersion != state.workspace.Version {
		return WorkspaceSubmissionResult{}, WorkspaceConflict{CurrentVersion: state.workspace.Version}
	}
	distributions := store.distributions
	distributions.mu.Lock()
	defer distributions.mu.Unlock()
	repo := distributions.repo
	repo.mu.Lock()
	defer repo.mu.Unlock()
	actor, err := identity.Require(ctx)
	if err != nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	current, currentWorkspace, recipient, ok := internalMemoryAccessLocked(ctx, distributions, actor, requestID, now)
	if !ok || current.Version != request.Version || currentWorkspace.ID != state.workspace.ID || currentWorkspace.Version != state.workspace.Version || currentWorkspace.Status != state.workspace.Status {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	authorityRoute, err := authorizeInternalWorkspaceLocked(ctx, actor, current)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	answers := cloneAnswerValues(state.answers)
	eligible := memoryDistributionTORequestIDsLocked(distributions, currentWorkspace.DistributionID)
	current = RefreshCollectionResolutions(ctx, current, func(_ context.Context, tenantID, sourceRequestID, artifactID string) (Artifact, error) {
		artifact, exists := repo.artifacts[artifactID]
		if !exists || artifact.TenantID != tenantID || artifact.RequestID != sourceRequestID {
			return Artifact{}, ErrNotFound
		}
		return withArtifactUsePolicy(artifact, repo.demoUnscannedAllowed), nil
	}, now)
	current = refreshCollectionSourceCurrencyMemoryLocked(distributions, current)
	current, err = refreshCollectionSourceReviews(ctx, current, repo.collectionReviews)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	if err := validateWorkspaceAnswerSet(ctx, repo, current, answers, true, func(_ context.Context, tenantID, _ string, artifactID string) (Artifact, error) {
		artifact, exists := repo.artifacts[artifactID]
		if !exists || artifact.TenantID != tenantID || !eligible[artifact.RequestID] {
			return Artifact{}, ErrNotFound
		}
		return withArtifactUsePolicy(artifact, repo.demoUnscannedAllowed), nil
	}, now); err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	metadata, err := buildResponseRevision(current, AssuranceInternalAuthenticated, attestations, answers)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	artifactIDs, err := submissionArtifactIDs(current, answers)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	for _, artifactID := range artifactIDs {
		artifact, exists := repo.artifacts[artifactID]
		if !exists || artifact.TenantID != actor.TenantID || !eligible[artifact.RequestID] {
			return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
		}
	}
	submissionID, err := id.NewUUIDv7()
	if err != nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	revisionID, err := id.NewUUIDv7()
	if err != nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	if _, exists := repo.submissions[submissionID]; exists {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	revisionNumber := int64(len(state.revisions) + 1)
	metadata.ID, metadata.TenantID, metadata.LegalEntityID = revisionID, actor.TenantID, actor.LegalEntityID
	metadata.DistributionID, metadata.WorkspaceID, metadata.SubmissionID = currentWorkspace.DistributionID, currentWorkspace.ID, submissionID
	metadata.SubmittedRecipientID, metadata.SubmittedRecipientPrincipalID, metadata.SubmittedRecipientType, metadata.SubmittedRecipientLabel = recipient.ID, recipient.PrincipalID, recipient.Type, recipient.ContactLabel
	metadata.Revision, metadata.CreatedAt, metadata.Current = revisionNumber, now.UTC(), true
	if metadata.Score != nil {
		metadata.Score.CalculatedAt = metadata.CreatedAt
	}
	if len(state.revisions) > 0 {
		metadata.SupersedesRevisionID = state.revisions[len(state.revisions)-1].ID
	}
	// All validation and ID generation precede the locked state changes.
	repo.submissions[submissionID] = Submission{ID: submissionID, TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, RequestID: requestID, SubmittedBy: actor.PrincipalID, Channel: "INTERNAL", Answers: cloneAnswerValues(answers), AnswerProvenance: respondentWorkspaceProvenance(answers), ExpectedVersion: current.Version, SubmittedAt: now.UTC()}
	for _, artifactID := range artifactIDs {
		artifact := repo.artifacts[artifactID]
		if artifact.SubmissionID == "" {
			artifact.SubmissionID = submissionID
			repo.artifacts[artifactID] = artifact
		}
	}
	if len(state.revisions) > 0 {
		state.revisions[len(state.revisions)-1].Current = false
	}
	state.revisions = append(state.revisions, cloneResponseRevision(metadata))
	prior := distributions.responseRevisions[currentWorkspace.DistributionID]
	if len(prior) > 0 {
		prior[len(prior)-1].Current = false
	}
	distributions.responseRevisions[currentWorkspace.DistributionID] = append(prior, cloneResponseRevision(metadata))
	state.workspace.Version++
	state.workspace.UpdatedAt = now.UTC()
	distributions.workspaces[currentWorkspace.DistributionID] = state.workspace
	scoreState := ResponseScoreNotConfigured
	if metadata.Score != nil {
		scoreState = metadata.Score.State
	}
	event := distributionEvent{DistributionID: currentWorkspace.DistributionID, Version: state.workspace.Version, EventType: fmt.Sprintf("FORM_RESPONSE_REVISION_SUBMITTED_%d", revisionNumber), ActorID: actor.PrincipalID, OccurredAt: now.UTC(), Payload: map[string]any{"authority_route": authorityRoute}}
	scored := distributionEvent{DistributionID: currentWorkspace.DistributionID, Version: state.workspace.Version, EventType: fmt.Sprintf("FORM_RESPONSE_SCORED_%d", revisionNumber), ActorID: actor.PrincipalID, OccurredAt: now.UTC(), Payload: map[string]any{"version": revisionNumber, "response_revision_id": metadata.ID, "request_id": current.ID, "request_version": current.Version, "form_template_id": current.FormTemplateID, "form_template_version": current.FormTemplateVersion, "score_state": string(scoreState)}}
	distributions.events = append(distributions.events, event, scored)
	distributions.outbox = append(distributions.outbox, event, scored)
	return WorkspaceSubmissionResult{Workspace: state.workspace, Revision: cloneResponseRevision(metadata), Submission: SubmissionReceipt{SubmissionID: submissionID, RequestID: requestID, Status: current.Status, SubmittedAt: now.UTC(), Version: current.Version}}, nil
}
