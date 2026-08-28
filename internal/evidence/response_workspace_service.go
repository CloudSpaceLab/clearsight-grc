package evidence

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func (service *DistributionAccessService) GetResponseWorkspace(ctx context.Context, sessionToken string) (ResponseWorkspaceView, error) {
	session, request, err := service.SessionRequest(ctx, sessionToken)
	if err != nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	store, ok := service.store.(responseWorkspaceStore)
	if !ok {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	view, err := store.GetResponseWorkspace(ctx, session, request, service.currentTime())
	if err != nil {
		return ResponseWorkspaceView{}, normalizeWorkspaceError(err)
	}
	return cloneResponseWorkspaceView(view), nil
}

func (service *DistributionAccessService) SaveResponseWorkspace(ctx context.Context, sessionToken string, input SaveWorkspaceInput) (ResponseWorkspaceView, error) {
	session, request, err := service.SessionRequest(ctx, sessionToken)
	if err != nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	if err := normalizeSaveWorkspaceInput(&input, request); err != nil {
		return ResponseWorkspaceView{}, err
	}
	store, validator, err := service.workspaceDependencies()
	if err != nil {
		return ResponseWorkspaceView{}, err
	}
	view, err := store.SaveResponseWorkspace(ctx, workspaceSaveCommand{
		Session: session,
		Request: request,
		Input:   input,
		Now:     service.currentTime(),
		Validate: func(answers map[string]formcontract.AnswerValue, provenance map[string]WorkspaceFieldProvenance) error {
			return validator.ValidateWorkspaceAnswers(ctx, request, answers, provenance, false)
		},
	})
	if err != nil {
		return ResponseWorkspaceView{}, normalizeWorkspaceError(err)
	}
	return cloneResponseWorkspaceView(view), nil
}

func (service *DistributionAccessService) SubmitResponseWorkspace(ctx context.Context, sessionToken string, input SubmitWorkspaceInput) (WorkspaceSubmissionResult, error) {
	session, request, err := service.SessionRequest(ctx, sessionToken)
	if err != nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	if input.ExpectedVersion < 1 {
		return WorkspaceSubmissionResult{}, fmt.Errorf("%w: expected_version must be positive", ErrWorkspaceUnavailable)
	}
	input.AttestationFieldIDs, err = normalizeAttestationFieldIDs(input.AttestationFieldIDs)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	store, validator, err := service.workspaceDependencies()
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	result, err := store.SubmitResponseWorkspace(ctx, workspaceSubmitCommand{
		Session: session,
		Request: request,
		Input:   input,
		Now:     service.currentTime(),
		Validate: func(answers map[string]formcontract.AnswerValue, provenance map[string]WorkspaceFieldProvenance) error {
			return validator.ValidateWorkspaceAnswers(ctx, request, answers, provenance, true)
		},
		BuildRevision: func(answers map[string]formcontract.AnswerValue) (ResponseRevision, error) {
			return buildResponseRevision(request, session.Assurance, input.AttestationFieldIDs, answers)
		},
	})
	if err != nil {
		return WorkspaceSubmissionResult{}, normalizeWorkspaceError(err)
	}
	return result, nil
}

func (service *DistributionAccessService) ClearResponseWorkspace(ctx context.Context, sessionToken string, expectedVersion int64) (ResponseWorkspaceView, error) {
	session, request, err := service.SessionRequest(ctx, sessionToken)
	if err != nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	if expectedVersion < 1 {
		return ResponseWorkspaceView{}, fmt.Errorf("%w: expected_version must be positive", ErrWorkspaceUnavailable)
	}
	store, validator, err := service.workspaceDependencies()
	if err != nil {
		return ResponseWorkspaceView{}, err
	}
	view, err := store.ClearResponseWorkspace(ctx, workspaceSaveCommand{
		Session: session,
		Request: request,
		Input: SaveWorkspaceInput{
			ExpectedVersion:  expectedVersion,
			PresentationMode: defaultDraftPresentation(request),
		},
		Now: service.currentTime(),
		Validate: func(answers map[string]formcontract.AnswerValue, provenance map[string]WorkspaceFieldProvenance) error {
			return validator.ValidateWorkspaceAnswers(ctx, request, answers, provenance, false)
		},
	})
	if err != nil {
		return ResponseWorkspaceView{}, normalizeWorkspaceError(err)
	}
	return cloneResponseWorkspaceView(view), nil
}

func (service *DistributionAccessService) workspaceDependencies() (responseWorkspaceStore, workspaceAnswerValidationProvider, error) {
	if service == nil || service.store == nil {
		return nil, nil, ErrWorkspaceUnavailable
	}
	store, ok := service.store.(responseWorkspaceStore)
	if !ok {
		return nil, nil, ErrWorkspaceUnavailable
	}
	validator, ok := service.store.(workspaceAnswerValidationProvider)
	if !ok {
		return nil, nil, ErrWorkspaceUnavailable
	}
	return store, validator, nil
}

func normalizeSaveWorkspaceInput(input *SaveWorkspaceInput, request Request) error {
	if input == nil || input.ExpectedVersion < 1 {
		return fmt.Errorf("%w: expected_version must be positive", ErrWorkspaceUnavailable)
	}
	input.PresentationMode = formcontract.PresentationMode(strings.ToUpper(strings.TrimSpace(string(input.PresentationMode))))
	if input.PresentationMode == "" {
		input.PresentationMode = defaultDraftPresentation(request)
	}
	switch input.PresentationMode {
	case formcontract.PresentationClassic, formcontract.PresentationWizard, formcontract.PresentationAutomatic:
	default:
		return fmt.Errorf("%w: presentation_mode is invalid", ErrDraftInvalid)
	}
	if len(input.Edits) == 0 || len(input.Edits) > formcontract.MaxFields {
		return fmt.Errorf("%w: between 1 and %d field edits are required", ErrWorkspaceUnavailable, formcontract.MaxFields)
	}
	seen := make(map[string]struct{}, len(input.Edits))
	for index := range input.Edits {
		fieldID := strings.TrimSpace(input.Edits[index].FieldID)
		if fieldID == "" || input.Edits[index].BaseSequence < 0 {
			return fmt.Errorf("%w: field_id and non-negative base_sequence are required", ErrWorkspaceUnavailable)
		}
		if _, duplicate := seen[fieldID]; duplicate {
			return fmt.Errorf("%w: duplicate field edit %q", ErrWorkspaceUnavailable, fieldID)
		}
		seen[fieldID] = struct{}{}
		input.Edits[index].FieldID = fieldID
	}
	return nil
}

func normalizeAttestationFieldIDs(values []string) ([]string, error) {
	if len(values) > formcontract.MaxFields {
		return nil, fmt.Errorf("%w: too many attestation fields", ErrWorkspaceUnavailable)
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("%w: attestation field id is required", ErrWorkspaceUnavailable)
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, fmt.Errorf("%w: duplicate attestation field %q", ErrWorkspaceUnavailable, value)
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func normalizeWorkspaceError(err error) error {
	if err == nil {
		return nil
	}
	var conflict WorkspaceConflict
	if errors.As(err, &conflict) {
		return conflict
	}
	if errors.Is(err, ErrWorkspaceConflict) {
		return err
	}
	return ErrWorkspaceUnavailable
}
