package evidence

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

var ErrSupersessionPreviewMismatch = errors.New("form distribution supersession preview is stale")

type SupersessionPreviewInput struct {
	ExpectedVersion   int64 `json:"expected_version"`
	TargetFormVersion int64 `json:"target_form_version"`
}

type SupersessionFieldDecision struct {
	FieldID string `json:"field_id"`
	Reason  string `json:"reason,omitempty"`
}

type DistributionSupersessionPreview struct {
	DistributionID           string                      `json:"distribution_id"`
	ExpectedVersion          int64                       `json:"expected_version"`
	ExpectedWorkspaceVersion int64                       `json:"expected_workspace_version"`
	TargetFormTemplateID     string                      `json:"target_form_template_id"`
	TargetFormVersion        int64                       `json:"target_form_version"`
	CompatibleFields         []SupersessionFieldDecision `json:"compatible_fields"`
	ExcludedFields           []SupersessionFieldDecision `json:"excluded_fields"`
}

type SupersedeDistributionInput struct {
	ExpectedVersion          int64
	ExpectedWorkspaceVersion int64
	TargetFormVersion        int64
	CarryForward             bool
	ConfirmedFieldIDs        []string
	ActorID                  string
}

type DistributionSupersessionResult struct {
	Previous        DistributionBundle  `json:"previous"`
	Replacement     DistributionBundle  `json:"replacement"`
	CarriedFieldIDs []string            `json:"carried_field_ids"`
	IssuedRoutes    []IssuedAccessRoute `json:"issued_access_routes,omitempty"`
}

type supersessionSnapshot struct {
	Bundle             DistributionBundle
	Workspace          ResponseWorkspaceView
	Request            Request
	EstimatedMinutes   int
	ProtectedAddresses map[string]protectedRecipientAddress
}

type supersessionCarry struct {
	FieldID     string
	Value       formcontract.AnswerValue
	RecipientID string
	RequestID   string
	Assurance   AccessAssurance
}

type supersessionCommit struct {
	TenantID                   string
	LegalEntityID              string
	PreviousDistributionID     string
	ReplacementDistributionID  string
	ExpectedPreviousVersion    int64
	ExpectedWorkspaceVersion   int64
	ExpectedReplacementVersion int64
	ActorID                    string
	PresentationMode           formcontract.PresentationMode
	Carries                    []supersessionCarry
	Now                        time.Time
}

type distributionSupersessionStore interface {
	DistributionAccessStore
	CreateDistribution(context.Context, CreateDistributionInput) (DistributionBundle, error)
	LoadSupersessionSnapshot(context.Context, string, string, string) (supersessionSnapshot, error)
	LoadSupersessionTargetForm(context.Context, string, string, string, int64) (DistributionFormRevision, error)
	CommitSupersession(context.Context, supersessionCommit) (DistributionBundle, DistributionBundle, error)
}

func (service *DistributionAccessService) PreviewDistributionSupersession(ctx context.Context, tenantID, legalEntityID, distributionID string, input SupersessionPreviewInput) (DistributionSupersessionPreview, error) {
	store, err := service.supersessionStore()
	if err != nil {
		return DistributionSupersessionPreview{}, err
	}
	if input.ExpectedVersion < 1 || input.TargetFormVersion < 1 {
		return DistributionSupersessionPreview{}, fmt.Errorf("%w: expected and target versions are required", ErrDistributionInvalid)
	}
	snapshot, err := store.LoadSupersessionSnapshot(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID), strings.TrimSpace(distributionID))
	if err != nil {
		return DistributionSupersessionPreview{}, normalizeSupersessionError(err)
	}
	current := snapshot.Bundle.Distribution
	if current.Version != input.ExpectedVersion || !supersessionSourceOpen(current, service.currentTime()) {
		return DistributionSupersessionPreview{}, ErrDistributionConflict
	}
	if input.TargetFormVersion == current.FormTemplateVersion {
		return DistributionSupersessionPreview{}, fmt.Errorf("%w: replacement revision must differ from the current revision", ErrDistributionInvalid)
	}
	target, err := store.LoadSupersessionTargetForm(ctx, current.TenantID, current.LegalEntityID, current.FormTemplateID, input.TargetFormVersion)
	if err != nil || !target.Active || target.ID != current.FormTemplateID || target.Version != input.TargetFormVersion {
		return DistributionSupersessionPreview{}, fmt.Errorf("%w: replacement form revision must be the exact active revision", ErrDistributionInvalid)
	}
	return buildSupersessionPreview(snapshot, target), nil
}

func (service *DistributionAccessService) SupersedeDistribution(ctx context.Context, tenantID, legalEntityID, distributionID string, input SupersedeDistributionInput) (DistributionSupersessionResult, error) {
	store, err := service.supersessionStore()
	if err != nil {
		return DistributionSupersessionResult{}, err
	}
	input.ActorID = strings.TrimSpace(input.ActorID)
	if input.ActorID == "" || input.ExpectedWorkspaceVersion < 1 {
		return DistributionSupersessionResult{}, fmt.Errorf("%w: actor and workspace version are required", ErrDistributionInvalid)
	}
	preview, err := service.PreviewDistributionSupersession(ctx, tenantID, legalEntityID, distributionID, SupersessionPreviewInput{
		ExpectedVersion: input.ExpectedVersion, TargetFormVersion: input.TargetFormVersion,
	})
	if err != nil {
		return DistributionSupersessionResult{}, err
	}
	if preview.ExpectedWorkspaceVersion != input.ExpectedWorkspaceVersion {
		return DistributionSupersessionResult{}, ErrSupersessionPreviewMismatch
	}
	compatible := decisionFieldIDs(preview.CompatibleFields)
	confirmed, err := normalizeConfirmedSupersessionFields(input.ConfirmedFieldIDs)
	if err != nil {
		return DistributionSupersessionResult{}, err
	}
	if input.CarryForward {
		if !sameStrings(compatible, confirmed) {
			return DistributionSupersessionResult{}, ErrSupersessionPreviewMismatch
		}
	} else if len(confirmed) != 0 {
		return DistributionSupersessionResult{}, fmt.Errorf("%w: confirmed fields require carry_forward", ErrDistributionInvalid)
	}

	snapshot, err := store.LoadSupersessionSnapshot(ctx, tenantID, legalEntityID, distributionID)
	if err != nil {
		return DistributionSupersessionResult{}, normalizeSupersessionError(err)
	}
	if snapshot.Bundle.Distribution.Version != input.ExpectedVersion || snapshot.Workspace.Workspace.Version != input.ExpectedWorkspaceVersion {
		return DistributionSupersessionResult{}, ErrSupersessionPreviewMismatch
	}
	recipientInputs, sourceRecipientIDs, err := service.supersessionRecipients(ctx, snapshot)
	if err != nil {
		return DistributionSupersessionResult{}, err
	}
	current := snapshot.Bundle.Distribution
	replacement, err := store.CreateDistribution(ctx, CreateDistributionInput{
		TenantID: current.TenantID, LegalEntityID: current.LegalEntityID,
		FormTemplateID: current.FormTemplateID, FormTemplateVersion: input.TargetFormVersion,
		SubjectType: current.SubjectType, SubjectID: current.SubjectID, Title: current.Title, Purpose: current.Purpose,
		AccessPolicy: current.AccessPolicy, EstimatedMinutes: snapshot.EstimatedMinutes,
		Deadline: current.Deadline, RouteExpiresAt: current.RouteExpiresAt, ReminderPolicy: cloneAnyMap(current.ReminderPolicy),
		CreatedBy: input.ActorID, Recipients: recipientInputs,
	})
	if err != nil {
		return DistributionSupersessionResult{}, normalizeSupersessionError(err)
	}
	if len(replacement.Recipients) != len(sourceRecipientIDs) {
		return DistributionSupersessionResult{}, ErrDistributionInvalid
	}

	mapping := make(map[string]DistributionRecipient, len(sourceRecipientIDs))
	for index, oldID := range sourceRecipientIDs {
		mapping[oldID] = replacement.Recipients[index]
	}
	carries := []supersessionCarry{}
	if input.CarryForward {
		for _, fieldID := range compatible {
			provenance := snapshot.Workspace.FieldProvenance[fieldID]
			recipient, ok := mapping[provenance.RecipientID]
			if !ok || recipient.Role != RecipientTo || recipient.RequestID == "" {
				return DistributionSupersessionResult{}, ErrSupersessionPreviewMismatch
			}
			carries = append(carries, supersessionCarry{
				FieldID: fieldID, Value: snapshot.Workspace.Answers[fieldID],
				RecipientID: recipient.ID, RequestID: recipient.RequestID, Assurance: provenance.Assurance,
			})
		}
	}

	issued := []IssuedAccessRoute{}
	if hasExternalTORecipients(replacement.Recipients) {
		issued, err = service.EnsureDistributionAccessRoutes(ctx, replacement.Distribution.TenantID, replacement.Distribution.LegalEntityID, replacement.Distribution.ID, input.ActorID)
		if err != nil {
			return DistributionSupersessionResult{}, err
		}
	}
	previous, opened, err := store.CommitSupersession(ctx, supersessionCommit{
		TenantID: current.TenantID, LegalEntityID: current.LegalEntityID,
		PreviousDistributionID: current.ID, ReplacementDistributionID: replacement.Distribution.ID,
		ExpectedPreviousVersion: input.ExpectedVersion, ExpectedWorkspaceVersion: input.ExpectedWorkspaceVersion,
		ExpectedReplacementVersion: replacement.Distribution.Version, ActorID: input.ActorID,
		PresentationMode: snapshot.Workspace.PresentationMode, Carries: carries, Now: service.currentTime(),
	})
	if err != nil {
		return DistributionSupersessionResult{}, normalizeSupersessionError(err)
	}
	return DistributionSupersessionResult{Previous: previous, Replacement: opened, CarriedFieldIDs: compatibleIf(input.CarryForward, compatible), IssuedRoutes: issued}, nil
}

func (service *DistributionAccessService) supersessionStore() (distributionSupersessionStore, error) {
	if service == nil || service.store == nil || service.revealer == nil {
		return nil, ErrDistributionAccessUnavailable
	}
	store, ok := service.store.(distributionSupersessionStore)
	if !ok {
		return nil, ErrDistributionAccessUnavailable
	}
	return store, nil
}

func (service *DistributionAccessService) supersessionRecipients(ctx context.Context, snapshot supersessionSnapshot) ([]DistributionRecipientInput, []string, error) {
	inputs := make([]DistributionRecipientInput, 0, len(snapshot.Bundle.Recipients))
	sourceIDs := make([]string, 0, len(snapshot.Bundle.Recipients))
	for _, recipient := range snapshot.Bundle.Recipients {
		if recipient.State == DistributionRecipientRevoked {
			continue
		}
		input := DistributionRecipientInput{
			Role: recipient.Role, Type: recipient.Type, PrincipalID: recipient.PrincipalID,
			AudienceHint: recipient.AudienceHint, ContactLabel: recipient.ContactLabel,
		}
		if recipient.Type == RecipientExternalAudience {
			protected, ok := snapshot.ProtectedAddresses[recipient.ID]
			if !ok {
				return nil, nil, ErrProtectedRecipientInvalid
			}
			address, err := service.revealer.RevealRecipientAddress(ctx, recipient.TenantID, recipient.DistributionID, recipient.ID, protected)
			if err != nil {
				return nil, nil, ErrProtectedRecipientInvalid
			}
			input.Address = address
		}
		inputs = append(inputs, input)
		sourceIDs = append(sourceIDs, recipient.ID)
	}
	return inputs, sourceIDs, nil
}

func buildSupersessionPreview(snapshot supersessionSnapshot, target DistributionFormRevision) DistributionSupersessionPreview {
	preview := DistributionSupersessionPreview{
		DistributionID:           snapshot.Bundle.Distribution.ID,
		ExpectedVersion:          snapshot.Bundle.Distribution.Version,
		ExpectedWorkspaceVersion: snapshot.Workspace.Workspace.Version,
		TargetFormTemplateID:     target.ID, TargetFormVersion: target.Version,
		CompatibleFields: []SupersessionFieldDecision{}, ExcludedFields: []SupersessionFieldDecision{},
	}
	targetByID := make(map[string]formcontract.Field, len(target.Fields))
	for _, field := range target.Fields {
		targetByID[field.ID] = field
	}
	sourceByID := make(map[string]Field, len(snapshot.Request.Fields))
	for _, field := range snapshot.Request.Fields {
		sourceByID[field.ID] = field
	}
	activeTO := map[string]bool{}
	for _, recipient := range snapshot.Bundle.Recipients {
		if recipient.Role == RecipientTo && recipient.State != DistributionRecipientRevoked {
			activeTO[recipient.ID] = true
		}
	}
	for fieldID, answer := range snapshot.Workspace.Answers {
		source, sourceOK := sourceByID[fieldID]
		targetField, targetOK := targetByID[fieldID]
		decision := SupersessionFieldDecision{FieldID: fieldID}
		switch {
		case !sourceOK || !targetOK:
			decision.Reason = "field_removed"
		case formcontract.Type(source.Type) != targetField.Type:
			decision.Reason = "field_type_changed"
		case unsafeSupersessionField(targetField.Type):
			decision.Reason = "provenance_sensitive_field"
		case !activeTO[snapshot.Workspace.FieldProvenance[fieldID].RecipientID]:
			decision.Reason = "contributor_not_carried"
		case !validAccessAssurance(snapshot.Workspace.FieldProvenance[fieldID].Assurance):
			decision.Reason = "assurance_unavailable"
		default:
			requestField := source
			requestField.Type = string(targetField.Type)
			requestField.Options = append([]string(nil), targetField.Options...)
			requestField.Constraints = targetField.Constraints
			requestField.Attestation = targetField.Attestation
			if err := (&Service{}).validateTypedAnswer(context.Background(), snapshot.Request, requestField, targetField, answer); err != nil {
				decision.Reason = "answer_invalid_for_replacement"
			}
		}
		if decision.Reason == "" {
			preview.CompatibleFields = append(preview.CompatibleFields, decision)
		} else {
			preview.ExcludedFields = append(preview.ExcludedFields, decision)
		}
	}
	sort.Slice(preview.CompatibleFields, func(i, j int) bool { return preview.CompatibleFields[i].FieldID < preview.CompatibleFields[j].FieldID })
	sort.Slice(preview.ExcludedFields, func(i, j int) bool { return preview.ExcludedFields[i].FieldID < preview.ExcludedFields[j].FieldID })
	return preview
}

func unsafeSupersessionField(fieldType formcontract.Type) bool {
	switch fieldType {
	case formcontract.TypeFile, formcontract.TypePhoto, formcontract.TypeSignature, formcontract.TypeVendorDocument, formcontract.TypeAttestation:
		return true
	default:
		return false
	}
}

func supersessionSourceOpen(value FormDistribution, now time.Time) bool {
	return (value.Status == DistributionOpen || value.Status == DistributionLocked) && value.Deadline.After(now)
}

func decisionFieldIDs(values []SupersessionFieldDecision) []string {
	result := make([]string, len(values))
	for index := range values {
		result[index] = values[index].FieldID
	}
	sort.Strings(result)
	return result
}

func normalizeConfirmedSupersessionFields(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("%w: confirmed field id is required", ErrDistributionInvalid)
		}
		if _, ok := seen[value]; ok {
			return nil, fmt.Errorf("%w: duplicate confirmed field id", ErrDistributionInvalid)
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func hasExternalTORecipients(values []DistributionRecipient) bool {
	for _, recipient := range values {
		if recipient.Role == RecipientTo && recipient.Type == RecipientExternalAudience && recipient.State != DistributionRecipientRevoked {
			return true
		}
	}
	return false
}

func compatibleIf(enabled bool, values []string) []string {
	if !enabled {
		return []string{}
	}
	return append([]string(nil), values...)
}

func normalizeSupersessionError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrDistributionInvalid) || errors.Is(err, ErrDistributionConflict) || errors.Is(err, ErrSupersessionPreviewMismatch) || errors.Is(err, ErrDistributionAccessUnavailable) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrDistributionInvalid, err)
}
