package monitoring

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

const (
	formAIProposalVersion = "FORM_AI_PROPOSAL_V1"
	maxAIObjectiveRunes   = 4000
	maxAISourceElements   = 100
)

func (s *FormProposalService) RequestAI(ctx context.Context, input RequestAIFormProposalInput) (FormTemplateProposal, error) {
	return s.requestAI(ctx, FormTemplate{}, input)
}

func (s *FormProposalService) RequestAIRevision(ctx context.Context, templateID string, version int64, input RequestAIFormProposalInput) (FormTemplateProposal, error) {
	if s == nil || s.forms == nil || strings.TrimSpace(templateID) == "" || version < 1 {
		return FormTemplateProposal{}, ErrInvalid
	}
	base, err := s.forms.GetLibraryForm(ctx, strings.TrimSpace(templateID), version)
	if err != nil {
		return FormTemplateProposal{}, err
	}
	if base.Version != version {
		return FormTemplateProposal{}, ErrConflict
	}
	return s.requestAI(ctx, base, input)
}

func (s *FormProposalService) requestAI(ctx context.Context, base FormTemplate, input RequestAIFormProposalInput) (FormTemplateProposal, error) {
	actor, err := proposalIdentity(ctx, s.now)
	if err != nil {
		return FormTemplateProposal{}, err
	}
	if s == nil || s.store == nil || s.forms == nil || s.ai == nil {
		return FormTemplateProposal{}, ErrFormAIUnavailable
	}
	objective := strings.TrimSpace(input.Objective)
	if objective == "" || len([]rune(objective)) > maxAIObjectiveRunes {
		return FormTemplateProposal{}, errors.Join(ErrInvalid, errors.New("AI proposal objective must contain 1-4000 characters"))
	}
	if base.ID != "" && (base.TenantID != actor.TenantID || base.LegalEntityID != actor.LegalEntityID) {
		return FormTemplateProposal{}, ErrNotFound
	}

	source, err := s.aiSourceSnapshot(ctx, actor.TenantID, actor.LegalEntityID, input)
	if err != nil {
		return FormTemplateProposal{}, err
	}
	baseContract := contractFromFormTemplate(base)
	snapshotSHA, err := formAISnapshotSHA256(objective, base, baseContract, source)
	if err != nil {
		return FormTemplateProposal{}, err
	}
	proposalID, err := s.newID()
	if err != nil {
		return FormTemplateProposal{}, err
	}
	now := s.now().UTC()
	value := FormTemplateProposal{
		ID: proposalID, TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID,
		SourceKind: FormProposalSourceAI, SourceSHA256: snapshotSHA,
		BaseTemplateID: base.ID, BaseTemplateVersion: base.Version,
		Status: FormProposalGenerating, CreatedBy: actor.PrincipalID, CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	if source != nil {
		value.SourceDocumentID = source.DocumentID
		value.SourceDocumentVersion = source.Version
	}
	created, err := createAIProposal(ctx, s.store, value)
	if err != nil {
		return FormTemplateProposal{}, err
	}
	if created.Status != FormProposalGenerating {
		return created, nil
	}

	result, err := s.ai.Propose(ctx, FormAIClientRequest{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID,
		Objective: objective, SnapshotSHA256: snapshotSHA,
		BaseTemplateID: base.ID, BaseTemplateVersion: base.Version, BaseContract: baseContract,
		Source: source,
	})
	if err != nil {
		failed, failErr := s.failGeneration(ctx, created, "AI_GATEWAY_UNAVAILABLE", "Governed AI authoring is temporarily unavailable; manual and deterministic authoring remain available.")
		if failErr != nil {
			return FormTemplateProposal{}, failErr
		}
		return failed, nil
	}

	created.Status = FormProposalReviewRequired
	created.ProposedContract = result.Contract
	created.FieldChanges = result.FieldChanges
	created.UnresolvedItems = result.UnresolvedItems
	created.Provenance = FormProposalProvenance{
		FormProposalProvenance: documentimport.FormProposalProvenance{
			ProposalVersion: formAIProposalVersion,
			SourceDocumentID: created.SourceDocumentID,
			SourceSHA256: sourceDocumentSHA(source),
			SourceVersion: created.SourceDocumentVersion,
			ExtractionStatus: sourceExtractionStatus(source),
		},
		AI: &result.Provenance,
	}
	created.FailureCode = ""
	created.FailureMessage = ""
	created.UpdatedAt = s.now().UTC()
	return s.store.CompleteGeneration(ctx, created, created.Version)
}

func (s *FormProposalService) aiSourceSnapshot(ctx context.Context, tenantID, legalEntityID string, input RequestAIFormProposalInput) (*FormAISourceSnapshot, error) {
	documentID := strings.TrimSpace(input.SourceDocumentID)
	if documentID == "" {
		if input.ExpectedSourceDocumentVersion != 0 || len(input.SourceElementRefs) != 0 {
			return nil, errors.Join(ErrInvalid, errors.New("source document version and element refs require a source document"))
		}
		return nil, nil
	}
	if s.docs == nil || input.ExpectedSourceDocumentVersion < 1 || len(input.SourceElementRefs) == 0 || len(input.SourceElementRefs) > maxAISourceElements {
		return nil, errors.Join(ErrInvalid, errors.New("AI source requires an exact document version and 1-100 selected element refs"))
	}
	document, err := s.docs.Get(ctx, tenantID, documentID)
	if err != nil {
		return nil, err
	}
	if document.LegalEntityID != legalEntityID {
		return nil, ErrNotFound
	}
	if document.Version != input.ExpectedSourceDocumentVersion {
		return nil, ErrConflict
	}
	if document.ExtractionStatus != documentimport.ExtractionExtracted && document.ExtractionStatus != documentimport.ExtractionTruncated {
		return nil, errors.Join(ErrInvalid, fmt.Errorf("document extraction status %s cannot be used as AI authoring source", document.ExtractionStatus))
	}

	wanted := make(map[string]struct{}, len(input.SourceElementRefs))
	for _, ref := range input.SourceElementRefs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return nil, errors.Join(ErrInvalid, errors.New("source element refs cannot be empty"))
		}
		if _, duplicate := wanted[ref]; duplicate {
			return nil, errors.Join(ErrInvalid, fmt.Errorf("duplicate source element ref %q", ref))
		}
		wanted[ref] = struct{}{}
	}
	selected := make([]documentimport.ExtractedElement, 0, len(wanted))
	for _, element := range document.Elements {
		if _, ok := wanted[element.Ref]; !ok {
			continue
		}
		selected = append(selected, element)
		delete(wanted, element.Ref)
	}
	if len(wanted) != 0 {
		missing := make([]string, 0, len(wanted))
		for ref := range wanted {
			missing = append(missing, ref)
		}
		sort.Strings(missing)
		return nil, errors.Join(ErrInvalid, fmt.Errorf("unknown source element refs: %s", strings.Join(missing, ", ")))
	}
	return &FormAISourceSnapshot{DocumentID: document.ID, Version: document.Version, SHA256: document.SHA256, Elements: selected}, nil
}

func contractFromFormTemplate(base FormTemplate) formcontract.Contract {
	if base.ID == "" {
		return formcontract.Contract{}
	}
	contract := formcontract.Contract{Presentation: base.Presentation, ScoringMode: base.ScoringMode}
	contract.Sections = make([]formcontract.Section, len(base.Sections))
	for index := range base.Sections {
		contract.Sections[index] = cloneProposalSection(base.Sections[index])
	}
	contract.Fields = make([]formcontract.Field, len(base.Fields))
	for index := range base.Fields {
		contract.Fields[index] = cloneTemplateField(base.Fields[index])
	}
	return contract
}

func formAISnapshotSHA256(objective string, base FormTemplate, contract formcontract.Contract, source *FormAISourceSnapshot) (string, error) {
	type sourceDescriptor struct {
		DocumentID string   `json:"document_id"`
		Version    int64    `json:"version"`
		SHA256     string   `json:"sha256"`
		Refs       []string `json:"refs"`
	}
	type snapshotDescriptor struct {
		Objective           string                 `json:"objective"`
		BaseTemplateID      string                 `json:"base_template_id,omitempty"`
		BaseTemplateVersion int64                  `json:"base_template_version,omitempty"`
		BaseContract        formcontract.Contract  `json:"base_contract"`
		Source              *sourceDescriptor      `json:"source,omitempty"`
	}
	descriptor := snapshotDescriptor{Objective: objective, BaseTemplateID: base.ID, BaseTemplateVersion: base.Version, BaseContract: contract}
	if source != nil {
		refs := make([]string, 0, len(source.Elements))
		for _, element := range source.Elements {
			refs = append(refs, element.Ref)
		}
		descriptor.Source = &sourceDescriptor{DocumentID: source.DocumentID, Version: source.Version, SHA256: source.SHA256, Refs: refs}
	}
	encoded, err := json.Marshal(descriptor)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func sourceDocumentSHA(source *FormAISourceSnapshot) string {
	if source == nil {
		return ""
	}
	return source.SHA256
}

func sourceExtractionStatus(source *FormAISourceSnapshot) string {
	if source == nil {
		return "AI_ONLY"
	}
	return "SELECTED_SOURCE"
}
