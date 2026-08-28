package evidence

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type memoryWorkspaceState struct {
	mu         sync.Mutex
	workspace  ResponseWorkspace
	answers    map[string]formcontract.AnswerValue
	mode       formcontract.PresentationMode
	sequences  map[string]int64
	provenance map[string]WorkspaceFieldProvenance
	edits      []workspaceEditRecord
	revisions  []ResponseRevision
}

type memoryWorkspaceRegistry struct {
	mu         sync.Mutex
	workspaces map[string]*memoryWorkspaceState
}

var memoryWorkspaceRegistries sync.Map

func memoryWorkspaceRegistryFor(store *MemoryDistributionAccessStore) *memoryWorkspaceRegistry {
	value, _ := memoryWorkspaceRegistries.LoadOrStore(store, &memoryWorkspaceRegistry{workspaces: map[string]*memoryWorkspaceState{}})
	return value.(*memoryWorkspaceRegistry)
}

func (store *MemoryDistributionAccessStore) GetResponseWorkspace(_ context.Context, session DistributionAccessSession, request Request, now time.Time) (ResponseWorkspaceView, error) {
	workspace, err := store.validateMemoryWorkspaceAccess(session, request, now)
	if err != nil {
		return ResponseWorkspaceView{}, err
	}
	state := store.memoryWorkspaceState(workspace, request)
	state.mu.Lock()
	defer state.mu.Unlock()
	return memoryWorkspaceView(state), nil
}

func (store *MemoryDistributionAccessStore) SaveResponseWorkspace(_ context.Context, command workspaceSaveCommand) (ResponseWorkspaceView, error) {
	workspace, err := store.validateMemoryWorkspaceAccess(command.Session, command.Request, command.Now)
	if err != nil {
		return ResponseWorkspaceView{}, err
	}
	state := store.memoryWorkspaceState(workspace, command.Request)
	state.mu.Lock()
	defer state.mu.Unlock()
	return store.saveMemoryWorkspaceLocked(state, command)
}

func (store *MemoryDistributionAccessStore) ClearResponseWorkspace(ctx context.Context, command workspaceSaveCommand) (ResponseWorkspaceView, error) {
	view, err := store.GetResponseWorkspace(ctx, command.Session, command.Request, command.Now)
	if err != nil {
		return ResponseWorkspaceView{}, err
	}
	if command.Input.ExpectedVersion != view.Workspace.Version {
		return ResponseWorkspaceView{}, WorkspaceConflict{CurrentVersion: view.Workspace.Version}
	}
	if len(view.Answers) == 0 {
		return view, nil
	}
	fieldIDs := make([]string, 0, len(view.Answers))
	for fieldID := range view.Answers {
		fieldIDs = append(fieldIDs, fieldID)
	}
	sort.Strings(fieldIDs)
	command.Input.Edits = make([]FieldEdit, 0, len(fieldIDs))
	for _, fieldID := range fieldIDs {
		command.Input.Edits = append(command.Input.Edits, FieldEdit{FieldID: fieldID, BaseSequence: view.FieldSequences[fieldID]})
	}
	return store.SaveResponseWorkspace(ctx, command)
}

func (store *MemoryDistributionAccessStore) SubmitResponseWorkspace(_ context.Context, command workspaceSubmitCommand) (WorkspaceSubmissionResult, error) {
	workspace, err := store.validateMemoryWorkspaceAccess(command.Session, command.Request, command.Now)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	state := store.memoryWorkspaceState(workspace, command.Request)
	state.mu.Lock()
	defer state.mu.Unlock()
	if command.Input.ExpectedVersion != state.workspace.Version || state.workspace.Status != ResponseWorkspaceOpen {
		return WorkspaceSubmissionResult{}, WorkspaceConflict{CurrentVersion: state.workspace.Version}
	}
	answers := cloneAnswerValues(state.answers)
	if command.Validate == nil || command.BuildRevision == nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	if err := command.Validate(answers); err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	metadata, err := command.BuildRevision(answers)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	submissionID, err := id.NewUUIDv7()
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	revisionID, err := id.NewUUIDv7()
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	artifactIDs, err := submissionArtifactIDs(command.Request, answers)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}

	distributions := store.distributions
	if distributions == nil || distributions.repo == nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	distributions.mu.Lock()
	defer distributions.mu.Unlock()
	if err := validateMemoryDistributionWorkspaceLocked(distributions, command.Session, state.workspace, command.Now); err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	eligibleRequests := memoryDistributionTORequestIDsLocked(distributions, command.Session.DistributionID)
	repo := distributions.repo
	repo.mu.Lock()
	defer repo.mu.Unlock()
	request, ok := repo.requests[command.Request.ID]
	if !ok || request.TenantID != command.Session.TenantID || !requestOpenAt(request, command.Now) {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	for _, artifactID := range artifactIDs {
		artifact, exists := repo.artifacts[artifactID]
		if !exists || artifact.TenantID != command.Session.TenantID || !eligibleRequests[artifact.RequestID] {
			return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
		}
	}
	if _, exists := repo.submissions[submissionID]; exists {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}

	submission := Submission{
		ID: submissionID, TenantID: command.Session.TenantID, LegalEntityID: command.Session.LegalEntityID,
		RequestID: command.Request.ID, Channel: "MAGIC_LINK",
		Answers: cloneAnswerValues(answers), AnswerProvenance: respondentWorkspaceProvenance(answers),
		ExpectedVersion: request.Version, SubmittedAt: command.Now.UTC(),
	}
	repo.submissions[submissionID] = submission
	for _, artifactID := range artifactIDs {
		artifact := repo.artifacts[artifactID]
		if artifact.SubmissionID == "" {
			artifact.SubmissionID = submissionID
			repo.artifacts[artifactID] = artifact
		}
	}

	revisionNumber := int64(len(state.revisions) + 1)
	metadata.ID = revisionID
	metadata.TenantID = command.Session.TenantID
	metadata.LegalEntityID = command.Session.LegalEntityID
	metadata.DistributionID = command.Session.DistributionID
	metadata.WorkspaceID = state.workspace.ID
	metadata.SubmissionID = submissionID
	metadata.Revision = revisionNumber
	metadata.CreatedAt = command.Now.UTC()
	if len(state.revisions) > 0 {
		previous := &state.revisions[len(state.revisions)-1]
		previous.Current = false
		metadata.SupersedesRevisionID = previous.ID
	}
	metadata.Current = true
	state.revisions = append(state.revisions, cloneResponseRevision(metadata))
	state.workspace.Version++
	state.workspace.UpdatedAt = command.Now.UTC()
	distributions.workspaces[command.Session.DistributionID] = state.workspace
	event := distributionEvent{
		DistributionID: command.Session.DistributionID,
		Version:        state.workspace.Version,
		EventType:      fmt.Sprintf("FORM_RESPONSE_REVISION_SUBMITTED_%d", revisionNumber),
		OccurredAt:     command.Now.UTC(),
	}
	distributions.events = append(distributions.events, event)
	distributions.outbox = append(distributions.outbox, event)
	return WorkspaceSubmissionResult{
		Workspace: state.workspace,
		Revision:  cloneResponseRevision(metadata),
		Submission: SubmissionReceipt{
			SubmissionID: submissionID, RequestID: request.ID, Status: request.Status,
			SubmittedAt: command.Now.UTC(), Version: request.Version,
		},
	}, nil
}

func (store *MemoryDistributionAccessStore) ValidateWorkspaceAnswers(ctx context.Context, session DistributionAccessSession, request Request, answers map[string]formcontract.AnswerValue, requireComplete bool) error {
	if store == nil || store.distributions == nil || store.distributions.repo == nil {
		return ErrWorkspaceUnavailable
	}
	store.distributions.mu.RLock()
	eligibleRequests := memoryDistributionTORequestIDsLocked(store.distributions, session.DistributionID)
	store.distributions.mu.RUnlock()
	repo := store.distributions.repo
	return validateWorkspaceAnswerSet(ctx, repo, request, answers, requireComplete, func(_ context.Context, tenantID, _ string, artifactID string) (Artifact, error) {
		repo.mu.RLock()
		defer repo.mu.RUnlock()
		artifact, ok := repo.artifacts[artifactID]
		if !ok || artifact.TenantID != tenantID || !eligibleRequests[artifact.RequestID] {
			return Artifact{}, ErrNotFound
		}
		return artifact, nil
	})
}

func (store *MemoryDistributionAccessStore) saveMemoryWorkspaceLocked(state *memoryWorkspaceState, command workspaceSaveCommand) (ResponseWorkspaceView, error) {
	if state.workspace.Status != ResponseWorkspaceOpen || command.Input.ExpectedVersion > state.workspace.Version {
		return ResponseWorkspaceView{}, WorkspaceConflict{CurrentVersion: state.workspace.Version}
	}
	conflict := WorkspaceConflict{CurrentVersion: state.workspace.Version}
	for _, edit := range command.Input.Edits {
		if state.sequences[edit.FieldID] == edit.BaseSequence {
			continue
		}
		conflict.Changed = append(conflict.Changed, FieldChange{
			FieldID: edit.FieldID, ServerValue: state.answers[edit.FieldID], Sequence: state.sequences[edit.FieldID],
		})
	}
	if len(conflict.Changed) > 0 {
		sort.Slice(conflict.Changed, func(i, j int) bool { return conflict.Changed[i].FieldID < conflict.Changed[j].FieldID })
		return ResponseWorkspaceView{}, conflict
	}
	merged := cloneAnswerValues(state.answers)
	changes := make([]FieldEdit, 0, len(command.Input.Edits))
	for _, edit := range command.Input.Edits {
		current := merged[edit.FieldID]
		if answerValueEqual(current, edit.Value) {
			continue
		}
		applyWorkspaceEdit(merged, edit)
		changes = append(changes, edit)
	}
	if command.Validate == nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	if err := command.Validate(merged); err != nil {
		return ResponseWorkspaceView{}, err
	}
	if len(changes) == 0 {
		state.mode = command.Input.PresentationMode
		return memoryWorkspaceView(state), nil
	}
	ids := make([]string, len(changes))
	for index := range changes {
		value, err := id.NewUUIDv7()
		if err != nil {
			return ResponseWorkspaceView{}, err
		}
		ids[index] = value
	}

	distributions := store.distributions
	distributions.mu.Lock()
	defer distributions.mu.Unlock()
	if err := validateMemoryDistributionWorkspaceLocked(distributions, command.Session, state.workspace, command.Now); err != nil {
		return ResponseWorkspaceView{}, err
	}
	state.answers = merged
	state.mode = command.Input.PresentationMode
	for index, edit := range changes {
		baseVersion := state.workspace.Version
		state.workspace.Version++
		sequence := state.workspace.Version
		state.sequences[edit.FieldID] = sequence
		state.provenance[edit.FieldID] = WorkspaceFieldProvenance{
			RecipientID: command.Session.RecipientID, RequestID: command.Session.RequestID,
			Assurance: command.Session.Assurance, Sequence: sequence, UpdatedAt: command.Now.UTC(),
		}
		state.edits = append(state.edits, workspaceEditRecord{
			ID: ids[index], TenantID: command.Session.TenantID, LegalEntityID: command.Session.LegalEntityID,
			DistributionID: command.Session.DistributionID, WorkspaceID: state.workspace.ID,
			RecipientID: command.Session.RecipientID, RequestID: command.Session.RequestID,
			BaseVersion: baseVersion, ResultVersion: sequence, FieldID: edit.FieldID, Value: edit.Value,
			PresentationMode: command.Input.PresentationMode, SessionID: command.Session.ID, RouteID: command.Session.RouteID,
			Assurance: command.Session.Assurance, CreatedAt: command.Now.UTC(),
		})
	}
	state.workspace.UpdatedAt = command.Now.UTC()
	distributions.workspaces[command.Session.DistributionID] = state.workspace
	event := distributionEvent{
		DistributionID: command.Session.DistributionID,
		Version:        state.workspace.Version,
		EventType:      fmt.Sprintf("FORM_RESPONSE_WORKSPACE_SAVED_%d", state.workspace.Version),
		OccurredAt:     command.Now.UTC(),
	}
	distributions.events = append(distributions.events, event)
	distributions.outbox = append(distributions.outbox, event)
	return memoryWorkspaceView(state), nil
}

func (store *MemoryDistributionAccessStore) memoryWorkspaceState(workspace ResponseWorkspace, request Request) *memoryWorkspaceState {
	registry := memoryWorkspaceRegistryFor(store)
	registry.mu.Lock()
	defer registry.mu.Unlock()
	state := registry.workspaces[workspace.DistributionID]
	if state != nil {
		return state
	}
	state = &memoryWorkspaceState{
		workspace:  workspace,
		answers:    map[string]formcontract.AnswerValue{},
		mode:       defaultDraftPresentation(request),
		sequences:  map[string]int64{},
		provenance: map[string]WorkspaceFieldProvenance{},
	}
	registry.workspaces[workspace.DistributionID] = state
	return state
}

func (store *MemoryDistributionAccessStore) validateMemoryWorkspaceAccess(session DistributionAccessSession, request Request, now time.Time) (ResponseWorkspace, error) {
	if store == nil || store.distributions == nil || request.ID != session.RequestID || request.TenantID != session.TenantID || request.LegalEntityID != session.LegalEntityID {
		return ResponseWorkspace{}, ErrWorkspaceUnavailable
	}
	store.mu.RLock()
	activeSession := false
	for _, persisted := range store.sessions {
		if persisted.ID == session.ID && persisted.TenantID == session.TenantID && persisted.LegalEntityID == session.LegalEntityID &&
			persisted.DistributionID == session.DistributionID && persisted.RecipientID == session.RecipientID && persisted.RequestID == session.RequestID &&
			persisted.RouteID == session.RouteID && persisted.Assurance == session.Assurance && persisted.RevokedAt == nil && persisted.ExpiresAt.After(now) {
			activeSession = true
			break
		}
	}
	route := store.routes[session.RouteID]
	store.mu.RUnlock()
	if !activeSession || route.ID == "" || route.RevokedAt != nil || !route.ExpiresAt.After(now) || !accessGrantAssuranceMatches(route.Policy, session.Assurance) {
		return ResponseWorkspace{}, ErrWorkspaceUnavailable
	}
	store.distributions.mu.RLock()
	defer store.distributions.mu.RUnlock()
	distribution := store.distributions.distributions[session.DistributionID]
	workspace := store.distributions.workspaces[session.DistributionID]
	if !distributionOpenForAccess(distribution, now) || workspace.ID == "" || workspace.Status != ResponseWorkspaceOpen ||
		workspace.TenantID != session.TenantID || workspace.LegalEntityID != session.LegalEntityID {
		return ResponseWorkspace{}, ErrWorkspaceUnavailable
	}
	return workspace, nil
}

func validateMemoryDistributionWorkspaceLocked(distributions *MemoryDistributionStore, session DistributionAccessSession, workspace ResponseWorkspace, now time.Time) error {
	distribution := distributions.distributions[session.DistributionID]
	persisted := distributions.workspaces[session.DistributionID]
	if !distributionOpenForAccess(distribution, now) || persisted.ID != workspace.ID || persisted.Status != ResponseWorkspaceOpen ||
		persisted.Version != workspace.Version || persisted.TenantID != session.TenantID || persisted.LegalEntityID != session.LegalEntityID {
		return ErrWorkspaceUnavailable
	}
	return nil
}

func memoryDistributionTORequestIDsLocked(distributions *MemoryDistributionStore, distributionID string) map[string]bool {
	result := map[string]bool{}
	for _, recipient := range distributions.recipients[distributionID] {
		if recipient.safe.Role == RecipientTo && recipient.safe.RequestID != "" && recipient.safe.State != DistributionRecipientRevoked {
			result[recipient.safe.RequestID] = true
		}
	}
	return result
}

func memoryWorkspaceView(state *memoryWorkspaceState) ResponseWorkspaceView {
	view := ResponseWorkspaceView{
		Workspace:        state.workspace,
		Answers:          cloneAnswerValues(state.answers),
		PresentationMode: state.mode,
		FieldSequences:   cloneInt64Map(state.sequences),
		FieldProvenance:  make(map[string]WorkspaceFieldProvenance, len(state.provenance)),
	}
	for key, value := range state.provenance {
		view.FieldProvenance[key] = value
	}
	if len(state.revisions) > 0 {
		revision := cloneResponseRevision(state.revisions[len(state.revisions)-1])
		view.CurrentRevision = &revision
	}
	return view
}

func respondentWorkspaceProvenance(answers map[string]formcontract.AnswerValue) map[string]AnswerProvenance {
	result := make(map[string]AnswerProvenance, len(answers))
	for fieldID := range answers {
		result[fieldID] = AnswerProvenance{Origin: AnswerRespondentEntered}
	}
	return result
}

var (
	_ responseWorkspaceStore            = (*MemoryDistributionAccessStore)(nil)
	_ workspaceAnswerValidationProvider = (*MemoryDistributionAccessStore)(nil)
)
