package evidence

import (
	"context"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func (store *MemoryDistributionAccessStore) CreateDistribution(ctx context.Context, input CreateDistributionInput) (DistributionBundle, error) {
	if store == nil || store.distributions == nil {
		return DistributionBundle{}, ErrDistributionInvalid
	}
	return store.distributions.CreateDistribution(ctx, input)
}

func (store *MemoryDistributionAccessStore) LoadSupersessionTargetForm(ctx context.Context, tenantID, legalEntityID, formID string, version int64) (DistributionFormRevision, error) {
	if store == nil || store.distributions == nil || store.distributions.forms == nil {
		return DistributionFormRevision{}, ErrDistributionInvalid
	}
	form, err := store.distributions.forms.GetDistributionFormRevision(ctx, tenantID, legalEntityID, formID, version)
	if err != nil {
		return DistributionFormRevision{}, err
	}
	if !form.Active || form.TenantID != tenantID || form.LegalEntityID != legalEntityID || form.ID != formID || form.Version != version {
		return DistributionFormRevision{}, ErrDistributionInvalid
	}
	return form, nil
}

func (store *MemoryDistributionAccessStore) LoadSupersessionSnapshot(_ context.Context, tenantID, legalEntityID, distributionID string) (supersessionSnapshot, error) {
	if store == nil || store.distributions == nil || store.distributions.repo == nil {
		return supersessionSnapshot{}, ErrDistributionInvalid
	}
	distributions := store.distributions
	distributions.mu.RLock()
	distribution, ok := distributions.distributions[distributionID]
	if !ok || distribution.TenantID != tenantID || distribution.LegalEntityID != legalEntityID {
		distributions.mu.RUnlock()
		return supersessionSnapshot{}, ErrNotFound
	}
	storedRecipients := cloneMemoryDistributionRecipients(distributions.recipients[distributionID])
	workspace := distributions.workspaces[distributionID]
	distributions.mu.RUnlock()

	var request Request
	distributions.repo.mu.RLock()
	for _, recipient := range storedRecipients {
		if recipient.safe.Role != RecipientTo || recipient.safe.RequestID == "" {
			continue
		}
		candidate, exists := distributions.repo.requests[recipient.safe.RequestID]
		if exists {
			request = cloneRequest(candidate)
			break
		}
	}
	distributions.repo.mu.RUnlock()
	if request.ID == "" || workspace.ID == "" {
		return supersessionSnapshot{}, ErrDistributionInvalid
	}

	state := store.memoryWorkspaceState(workspace, request)
	state.mu.Lock()
	view := memoryWorkspaceView(state)
	state.mu.Unlock()
	protected := make(map[string]protectedRecipientAddress)
	safeRecipients := make([]DistributionRecipient, len(storedRecipients))
	for index, recipient := range storedRecipients {
		safeRecipients[index] = recipient.safe
		if recipient.safe.Type == RecipientExternalAudience && recipient.safe.State != DistributionRecipientRevoked {
			protected[recipient.safe.ID] = cloneProtectedRecipientAddress(recipient.protected)
		}
	}
	return supersessionSnapshot{
		Bundle:    DistributionBundle{Distribution: cloneDistribution(distribution), Recipients: safeRecipients, Workspace: workspace},
		Workspace: view, Request: request, EstimatedMinutes: request.EstimatedMinutes, ProtectedAddresses: protected,
	}, nil
}

func (store *MemoryDistributionAccessStore) CommitSupersession(_ context.Context, command supersessionCommit) (DistributionBundle, DistributionBundle, error) {
	if store == nil || store.distributions == nil || store.distributions.repo == nil || command.ExpectedPreviousVersion < 1 || command.ExpectedWorkspaceVersion < 1 || command.ExpectedReplacementVersion < 1 {
		return DistributionBundle{}, DistributionBundle{}, ErrDistributionInvalid
	}
	distributions := store.distributions

	distributions.mu.RLock()
	oldWorkspace := distributions.workspaces[command.PreviousDistributionID]
	newWorkspace := distributions.workspaces[command.ReplacementDistributionID]
	oldRecipients := cloneMemoryDistributionRecipients(distributions.recipients[command.PreviousDistributionID])
	newRecipients := cloneMemoryDistributionRecipients(distributions.recipients[command.ReplacementDistributionID])
	distributions.mu.RUnlock()
	if oldWorkspace.ID == "" || newWorkspace.ID == "" {
		return DistributionBundle{}, DistributionBundle{}, ErrNotFound
	}

	var oldRequest, replacementRequest Request
	distributions.repo.mu.RLock()
	for _, recipient := range oldRecipients {
		if recipient.safe.Role == RecipientTo && recipient.safe.RequestID != "" {
			oldRequest = cloneRequest(distributions.repo.requests[recipient.safe.RequestID])
			if oldRequest.ID != "" {
				break
			}
		}
	}
	for _, recipient := range newRecipients {
		if recipient.safe.Role == RecipientTo && recipient.safe.RequestID != "" {
			replacementRequest = cloneRequest(distributions.repo.requests[recipient.safe.RequestID])
			if replacementRequest.ID != "" {
				break
			}
		}
	}
	distributions.repo.mu.RUnlock()
	if oldRequest.ID == "" || replacementRequest.ID == "" {
		return DistributionBundle{}, DistributionBundle{}, ErrDistributionInvalid
	}
	oldState := store.memoryWorkspaceState(oldWorkspace, oldRequest)
	newState := store.memoryWorkspaceState(newWorkspace, replacementRequest)

	editIDs := make([]string, len(command.Carries))
	for index := range editIDs {
		value, err := id.NewUUIDv7()
		if err != nil {
			return DistributionBundle{}, DistributionBundle{}, err
		}
		editIDs[index] = value
	}

	oldState.mu.Lock()
	defer oldState.mu.Unlock()
	newState.mu.Lock()
	defer newState.mu.Unlock()
	store.mu.Lock()
	defer store.mu.Unlock()
	distributions.mu.Lock()
	defer distributions.mu.Unlock()
	distributions.repo.mu.Lock()
	defer distributions.repo.mu.Unlock()

	previous, oldOK := distributions.distributions[command.PreviousDistributionID]
	replacement, newOK := distributions.distributions[command.ReplacementDistributionID]
	persistedOldWorkspace := distributions.workspaces[command.PreviousDistributionID]
	persistedNewWorkspace := distributions.workspaces[command.ReplacementDistributionID]
	if !oldOK || !newOK || previous.TenantID != command.TenantID || previous.LegalEntityID != command.LegalEntityID || replacement.TenantID != command.TenantID || replacement.LegalEntityID != command.LegalEntityID {
		return DistributionBundle{}, DistributionBundle{}, ErrNotFound
	}
	if previous.Version != command.ExpectedPreviousVersion || oldState.workspace.Version != command.ExpectedWorkspaceVersion || persistedOldWorkspace.Version != command.ExpectedWorkspaceVersion ||
		replacement.Version != command.ExpectedReplacementVersion || replacement.Status != DistributionDraft || previous.Status == DistributionSuperseded || !previous.Deadline.After(command.Now) {
		return DistributionBundle{}, DistributionBundle{}, ErrSupersessionPreviewMismatch
	}

	newState.workspace = persistedNewWorkspace
	newState.mode = command.PresentationMode
	if newState.answers == nil {
		newState.answers = map[string]formcontract.AnswerValue{}
	}
	if newState.sequences == nil {
		newState.sequences = map[string]int64{}
	}
	if newState.provenance == nil {
		newState.provenance = map[string]WorkspaceFieldProvenance{}
	}
	for index, carry := range command.Carries {
		baseVersion := newState.workspace.Version
		newState.workspace.Version++
		sequence := newState.workspace.Version
		newState.answers[carry.FieldID] = cloneAnswerValues(map[string]formcontract.AnswerValue{carry.FieldID: carry.Value})[carry.FieldID]
		newState.sequences[carry.FieldID] = sequence
		newState.provenance[carry.FieldID] = WorkspaceFieldProvenance{
			RecipientID: carry.RecipientID, RequestID: carry.RequestID, Assurance: carry.Assurance,
			Sequence: sequence, UpdatedAt: command.Now.UTC(),
		}
		newState.edits = append(newState.edits, workspaceEditRecord{
			ID: editIDs[index], TenantID: command.TenantID, LegalEntityID: command.LegalEntityID,
			DistributionID: replacement.ID, WorkspaceID: newState.workspace.ID,
			RecipientID: carry.RecipientID, RequestID: carry.RequestID,
			BaseVersion: baseVersion, ResultVersion: sequence, FieldID: carry.FieldID, Value: carry.Value,
			PresentationMode: command.PresentationMode, Assurance: carry.Assurance, CreatedAt: command.Now.UTC(),
		})
	}
	newState.workspace.UpdatedAt = command.Now.UTC()
	persistedNewWorkspace = newState.workspace

	previous.Status = DistributionSuperseded
	previous.Version++
	previous.UpdatedAt = command.Now.UTC()
	oldState.workspace.Status = ResponseWorkspaceLocked
	oldState.workspace.Version++
	oldState.workspace.UpdatedAt = command.Now.UTC()
	persistedOldWorkspace = oldState.workspace

	replacement.Status = DistributionOpen
	replacement.Version++
	replacement.UpdatedAt = command.Now.UTC()
	distributions.distributions[previous.ID] = previous
	distributions.distributions[replacement.ID] = replacement
	distributions.workspaces[previous.ID] = persistedOldWorkspace
	distributions.workspaces[replacement.ID] = persistedNewWorkspace

	for requestID, linkedDistributionID := range distributions.requestDistribution {
		if linkedDistributionID != previous.ID {
			continue
		}
		request := distributions.repo.requests[requestID]
		if request.Status != RequestCancelled && request.Status != RequestExpired {
			request.Status = RequestCancelled
			request.Version++
			request.UpdatedAt = command.Now.UTC()
			distributions.repo.requests[requestID] = request
		}
	}
	for routeID, route := range store.routes {
		if route.DistributionID == previous.ID && route.RevokedAt == nil {
			revoked := command.Now.UTC()
			route.RevokedAt = &revoked
			store.routes[routeID] = route
		}
	}
	for key, session := range store.sessions {
		if session.DistributionID == previous.ID && session.RevokedAt == nil {
			revoked := command.Now.UTC()
			session.RevokedAt = &revoked
			store.sessions[key] = session
		}
	}

	oldEvent := distributionEvent{DistributionID: previous.ID, Version: previous.Version, EventType: "FORM_DISTRIBUTION_SUPERSEDED", ActorID: command.ActorID, OccurredAt: command.Now.UTC()}
	newEvent := distributionEvent{DistributionID: replacement.ID, Version: replacement.Version, EventType: "FORM_DISTRIBUTION_OPEN", ActorID: command.ActorID, OccurredAt: command.Now.UTC()}
	distributions.events = append(distributions.events, oldEvent, newEvent)
	distributions.outbox = append(distributions.outbox, oldEvent, newEvent)

	previousBundle := bundleFromMemory(previous, distributions.recipients[previous.ID], persistedOldWorkspace)
	replacementBundle := bundleFromMemory(replacement, distributions.recipients[replacement.ID], persistedNewWorkspace)
	return previousBundle, replacementBundle, nil
}

var _ distributionSupersessionStore = (*MemoryDistributionAccessStore)(nil)
