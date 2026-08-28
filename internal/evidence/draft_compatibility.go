package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

// DraftCompatibilityService preserves the legacy session-owned draft contract
// while translating distribution sessions onto the shared field-edit workspace.
// HTTP composition can use this facade during the migration window without
// teaching the legacy invitation Service about distribution persistence.
type DraftCompatibilityService struct {
	legacy       *Service
	distribution *DistributionAccessService
}

func NewDraftCompatibilityService(legacy *Service, distribution *DistributionAccessService) *DraftCompatibilityService {
	return &DraftCompatibilityService{legacy: legacy, distribution: distribution}
}

func (service *DraftCompatibilityService) GetDraft(ctx context.Context, sessionToken string) (ResponseDraft, error) {
	if session, _, ok := service.distributionSession(ctx, sessionToken); ok {
		view, err := service.distribution.GetResponseWorkspace(ctx, sessionToken)
		if err != nil {
			return ResponseDraft{}, err
		}
		return responseDraftFromWorkspace(session, view), nil
	}
	if service == nil || service.legacy == nil {
		return ResponseDraft{}, ErrSessionInvalid
	}
	return service.legacy.GetDraft(ctx, sessionToken)
}

func (service *DraftCompatibilityService) SaveDraft(ctx context.Context, sessionToken string, input SaveDraftInput) (ResponseDraft, error) {
	session, request, ok := service.distributionSession(ctx, sessionToken)
	if !ok {
		if service == nil || service.legacy == nil {
			return ResponseDraft{}, ErrSessionInvalid
		}
		return service.legacy.SaveDraft(ctx, sessionToken, input)
	}
	if input.ExpectedVersion < 0 {
		return ResponseDraft{}, fmt.Errorf("%w: expected_version must not be negative", ErrDraftInvalid)
	}
	mode, err := normalizeCompatibilityDraftMode(input.PresentationMode, request)
	if err != nil {
		return ResponseDraft{}, err
	}
	if input.Answers == nil {
		input.Answers = map[string]formcontract.AnswerValue{}
	}
	encoded, err := json.Marshal(input.Answers)
	if err != nil || len(encoded) > maxDraftAnswersBytes {
		return ResponseDraft{}, fmt.Errorf("%w: draft answers exceed the permitted size", ErrDraftInvalid)
	}
	view, err := service.distribution.GetResponseWorkspace(ctx, sessionToken)
	if err != nil {
		return ResponseDraft{}, err
	}
	if input.ExpectedVersion != view.Workspace.Version {
		return ResponseDraft{}, ErrVersionConflict
	}
	_, validator, err := service.distribution.workspaceDependencies()
	if err != nil {
		return ResponseDraft{}, err
	}
	if err := validator.ValidateWorkspaceAnswers(ctx, session, request, input.Answers, false); err != nil {
		return ResponseDraft{}, fmt.Errorf("%w: %v", ErrDraftInvalid, err)
	}

	edits := compatibilityDraftEdits(view, input.Answers)
	if len(edits) == 0 {
		return responseDraftFromWorkspace(session, view), nil
	}
	updated, err := service.distribution.SaveResponseWorkspace(ctx, sessionToken, SaveWorkspaceInput{
		ExpectedVersion:  view.Workspace.Version,
		PresentationMode: mode,
		Edits:            edits,
	})
	if err != nil {
		if errors.Is(err, ErrWorkspaceConflict) {
			return ResponseDraft{}, ErrVersionConflict
		}
		return ResponseDraft{}, err
	}
	return responseDraftFromWorkspace(session, updated), nil
}

func (service *DraftCompatibilityService) DeleteDraft(ctx context.Context, sessionToken string) error {
	session, _, ok := service.distributionSession(ctx, sessionToken)
	if !ok {
		if service == nil || service.legacy == nil {
			return ErrSessionInvalid
		}
		return service.legacy.DeleteDraft(ctx, sessionToken)
	}
	view, err := service.distribution.GetResponseWorkspace(ctx, sessionToken)
	if err != nil {
		return err
	}
	fieldIDs := make([]string, 0)
	for fieldID, provenance := range view.FieldProvenance {
		if provenance.RecipientID == session.RecipientID && provenance.RequestID == session.RequestID {
			fieldIDs = append(fieldIDs, fieldID)
		}
	}
	if len(fieldIDs) == 0 {
		return nil
	}
	sort.Strings(fieldIDs)
	edits := make([]FieldEdit, 0, len(fieldIDs))
	for _, fieldID := range fieldIDs {
		edits = append(edits, FieldEdit{FieldID: fieldID, BaseSequence: view.FieldSequences[fieldID]})
	}
	_, err = service.distribution.SaveResponseWorkspace(ctx, sessionToken, SaveWorkspaceInput{
		ExpectedVersion:  view.Workspace.Version,
		PresentationMode: view.PresentationMode,
		Edits:            edits,
	})
	if errors.Is(err, ErrWorkspaceConflict) {
		return ErrVersionConflict
	}
	return err
}

func (service *DraftCompatibilityService) distributionSession(ctx context.Context, sessionToken string) (DistributionAccessSession, Request, bool) {
	if service == nil || service.distribution == nil || strings.TrimSpace(sessionToken) == "" {
		return DistributionAccessSession{}, Request{}, false
	}
	session, request, err := service.distribution.SessionRequest(ctx, sessionToken)
	if err != nil {
		return DistributionAccessSession{}, Request{}, false
	}
	return session, request, true
}

func compatibilityDraftEdits(view ResponseWorkspaceView, replacement map[string]formcontract.AnswerValue) []FieldEdit {
	fieldIDs := make(map[string]struct{}, len(view.Answers)+len(replacement))
	for fieldID := range view.Answers {
		fieldIDs[fieldID] = struct{}{}
	}
	for fieldID := range replacement {
		fieldIDs[fieldID] = struct{}{}
	}
	ordered := make([]string, 0, len(fieldIDs))
	for fieldID := range fieldIDs {
		ordered = append(ordered, fieldID)
	}
	sort.Strings(ordered)
	edits := make([]FieldEdit, 0, len(ordered))
	for _, fieldID := range ordered {
		current := view.Answers[fieldID]
		next := replacement[fieldID]
		if answerValueEqual(current, next) {
			continue
		}
		edits = append(edits, FieldEdit{FieldID: fieldID, Value: next, BaseSequence: view.FieldSequences[fieldID]})
	}
	return edits
}

func normalizeCompatibilityDraftMode(mode formcontract.PresentationMode, request Request) (formcontract.PresentationMode, error) {
	mode = formcontract.PresentationMode(strings.ToUpper(strings.TrimSpace(string(mode))))
	if mode == "" {
		mode = defaultDraftPresentation(request)
	}
	switch mode {
	case formcontract.PresentationClassic, formcontract.PresentationWizard, formcontract.PresentationAutomatic:
		return mode, nil
	default:
		return "", fmt.Errorf("%w: presentation_mode is invalid", ErrDraftInvalid)
	}
}

func responseDraftFromWorkspace(session DistributionAccessSession, view ResponseWorkspaceView) ResponseDraft {
	return ResponseDraft{
		ID:               view.Workspace.ID,
		TenantID:         view.Workspace.TenantID,
		RequestID:        session.RequestID,
		SessionID:        session.ID,
		Answers:          cloneAnswerValues(view.Answers),
		PresentationMode: view.PresentationMode,
		Version:          view.Workspace.Version,
		CreatedAt:        view.Workspace.CreatedAt,
		UpdatedAt:        view.Workspace.UpdatedAt,
	}
}
