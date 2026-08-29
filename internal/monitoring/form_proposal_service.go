package monitoring

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type proposalFormAuthor interface {
	GetLibraryForm(context.Context, string, int64) (FormTemplate, error)
	CreateFormRevision(context.Context, string, CreateFormRevisionInput) (FormTemplate, error)
	CreateLibraryForm(context.Context, CreateFormInput) (FormTemplate, error)
}

type FormProposalService struct {
	store FormProposalStore
	docs  formProposalDocumentReader
	forms proposalFormAuthor
	ai    FormAIClient
	now   func() time.Time
	newID func() (string, error)
}

func NewFormProposalService(store FormProposalStore, docs formProposalDocumentReader, forms proposalFormAuthor) *FormProposalService {
	return &FormProposalService{store: store, docs: docs, forms: forms, now: time.Now, newID: id.NewUUIDv7}
}

func (s *FormProposalService) ConfigureAIClient(client FormAIClient) {
	s.ai = client
}

func (s *FormProposalService) RequestFromDocument(ctx context.Context, documentID string, input RequestDocumentFormProposalInput) (FormTemplateProposal, error) {
	actor, err := proposalIdentity(ctx, s.now)
	if err != nil {
		return FormTemplateProposal{}, err
	}
	if s.store == nil || s.docs == nil || s.forms == nil {
		return FormTemplateProposal{}, errors.Join(ErrInvalid, errors.New("form proposal service is not configured"))
	}
	documentID = strings.TrimSpace(documentID)
	if documentID == "" || input.ExpectedDocumentVersion < 1 {
		return FormTemplateProposal{}, errors.Join(ErrInvalid, errors.New("document and expected document version are required"))
	}
	if (strings.TrimSpace(input.BaseTemplateID) == "") != (input.BaseTemplateVersion == 0) || input.BaseTemplateVersion < 0 {
		return FormTemplateProposal{}, errors.Join(ErrInvalid, errors.New("base template id and version must be supplied together"))
	}
	document, err := s.docs.Get(ctx, actor.TenantID, documentID)
	if err != nil {
		return FormTemplateProposal{}, err
	}
	if document.LegalEntityID != actor.LegalEntityID {
		return FormTemplateProposal{}, ErrNotFound
	}
	if document.Version != input.ExpectedDocumentVersion {
		return FormTemplateProposal{}, ErrConflict
	}
	if document.ExtractionStatus != documentimport.ExtractionExtracted && document.ExtractionStatus != documentimport.ExtractionTruncated {
		return FormTemplateProposal{}, errors.Join(ErrInvalid, fmt.Errorf("document extraction status %s cannot generate a form proposal", document.ExtractionStatus))
	}
	if input.BaseTemplateID != "" {
		if _, err := s.forms.GetLibraryForm(ctx, strings.TrimSpace(input.BaseTemplateID), input.BaseTemplateVersion); err != nil {
			return FormTemplateProposal{}, err
		}
	}
	proposalID, err := s.newID()
	if err != nil {
		return FormTemplateProposal{}, err
	}
	now := s.now().UTC()
	created, err := s.store.Create(ctx, FormTemplateProposal{
		ID: proposalID, TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID,
		SourceKind: FormProposalSourceDocument, SourceDocumentID: document.ID, SourceDocumentVersion: document.Version, SourceSHA256: document.SHA256,
		BaseTemplateID: strings.TrimSpace(input.BaseTemplateID), BaseTemplateVersion: input.BaseTemplateVersion,
		Status: FormProposalGenerating, CreatedBy: actor.PrincipalID, CreatedAt: now, UpdatedAt: now, Version: 1,
	})
	if err != nil {
		return FormTemplateProposal{}, err
	}
	if s.store.QueuesGeneration() || created.Status != FormProposalGenerating {
		return created, nil
	}
	return s.Generate(ctx, created.TenantID, created.LegalEntityID, created.ID)
}

func (s *FormProposalService) Generate(ctx context.Context, tenantID, legalEntityID, proposalID string) (FormTemplateProposal, error) {
	if s.store == nil || s.docs == nil {
		return FormTemplateProposal{}, errors.Join(ErrInvalid, errors.New("form proposal generation is not configured"))
	}
	current, err := s.store.Get(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID), strings.TrimSpace(proposalID))
	if err != nil {
		return FormTemplateProposal{}, err
	}
	if current.Status != FormProposalGenerating {
		return current, nil
	}
	if current.SourceKind != FormProposalSourceDocument {
		return s.failGeneration(ctx, current, "UNSUPPORTED_SOURCE_KIND", "Background deterministic generation requires a document source.")
	}
	document, err := s.docs.Get(ctx, current.TenantID, current.SourceDocumentID)
	if err != nil {
		return FormTemplateProposal{}, err
	}
	if !proposalSourceMatchesDocument(current, document) {
		return s.failGeneration(ctx, current, "SOURCE_CHANGED", ErrFormProposalSourceChanged.Error())
	}
	generated, err := documentimport.ProposeFormTemplate(document, documentimport.DefaultProposalPolicy())
	if err != nil {
		return s.failGeneration(ctx, current, "DETERMINISTIC_PROPOSAL_FAILED", err.Error())
	}
	now := s.now().UTC()
	current.Status = FormProposalReviewRequired
	current.ProposedContract = generated.Contract
	current.FieldChanges = generated.FieldChanges
	current.UnresolvedItems = generated.UnresolvedItems
	current.Provenance = FormProposalProvenance{FormProposalProvenance: generated.Provenance}
	current.FailureCode = ""
	current.FailureMessage = ""
	current.UpdatedAt = now
	return s.store.CompleteGeneration(ctx, current, current.Version)
}

func (s *FormProposalService) Get(ctx context.Context, proposalID string) (FormTemplateProposal, error) {
	actor, err := proposalIdentity(ctx, s.now)
	if err != nil {
		return FormTemplateProposal{}, err
	}
	if s.store == nil {
		return FormTemplateProposal{}, errors.Join(ErrInvalid, errors.New("form proposal store is not configured"))
	}
	return s.store.Get(ctx, actor.TenantID, actor.LegalEntityID, strings.TrimSpace(proposalID))
}

func (s *FormProposalService) Reject(ctx context.Context, proposalID string, input RejectFormProposalInput) (FormTemplateProposal, error) {
	actor, err := proposalIdentity(ctx, s.now)
	if err != nil {
		return FormTemplateProposal{}, err
	}
	if input.ExpectedVersion < 1 || strings.TrimSpace(proposalID) == "" {
		return FormTemplateProposal{}, ErrInvalid
	}
	return s.store.Review(ctx, FormProposalReviewMutation{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ProposalID: strings.TrimSpace(proposalID),
		ExpectedVersion: input.ExpectedVersion, Status: FormProposalRejected, ReviewerID: actor.PrincipalID, At: s.now().UTC(),
	})
}

func (s *FormProposalService) Accept(ctx context.Context, proposalID string, input AcceptFormProposalInput) (FormTemplateProposal, error) {
	actor, err := proposalIdentity(ctx, s.now)
	if err != nil {
		return FormTemplateProposal{}, err
	}
	if s.store == nil || s.forms == nil || input.ExpectedVersion < 1 || strings.TrimSpace(proposalID) == "" {
		return FormTemplateProposal{}, ErrInvalid
	}
	changeIDs := normalizeProposalChangeIDs(input.ChangeIDs)
	if len(changeIDs) == 0 || len(changeIDs) > 500 {
		return FormTemplateProposal{}, ErrFormProposalSelection
	}
	proposal, err := s.store.Get(ctx, actor.TenantID, actor.LegalEntityID, strings.TrimSpace(proposalID))
	if err != nil {
		return FormTemplateProposal{}, err
	}
	if proposal.Status == FormProposalAccepted && proposal.ReviewedBy == actor.PrincipalID && slices.Equal(proposal.AcceptedChangeIDs, changeIDs) {
		return proposal, nil
	}
	if proposal.Version != input.ExpectedVersion {
		return FormTemplateProposal{}, ErrConflict
	}
	if proposal.Status != FormProposalReviewRequired {
		return FormTemplateProposal{}, ErrFormProposalState
	}

	var sourceDocument *documentimport.Document
	if proposal.SourceDocumentID != "" {
		if s.docs == nil {
			return FormTemplateProposal{}, ErrInvalid
		}
		document, err := s.docs.Get(ctx, actor.TenantID, proposal.SourceDocumentID)
		if err != nil {
			return FormTemplateProposal{}, err
		}
		switch proposal.SourceKind {
		case FormProposalSourceDocument:
			if !proposalSourceMatchesDocument(proposal, document) {
				return FormTemplateProposal{}, ErrFormProposalSourceChanged
			}
		case FormProposalSourceAI:
			if !proposalAISourceMatchesDocument(proposal, document) {
				return FormTemplateProposal{}, ErrFormProposalSourceChanged
			}
		default:
			return FormTemplateProposal{}, ErrFormProposalSourceChanged
		}
		sourceDocument = &document
	} else if proposal.SourceKind == FormProposalSourceDocument {
		return FormTemplateProposal{}, ErrFormProposalSourceChanged
	}

	var base FormTemplate
	if proposal.BaseTemplateID != "" {
		base, err = s.forms.GetLibraryForm(ctx, proposal.BaseTemplateID, proposal.BaseTemplateVersion)
		if err != nil {
			return FormTemplateProposal{}, err
		}
	}
	contract, err := applySelectedProposalChanges(base, proposal, changeIDs)
	if err != nil {
		return FormTemplateProposal{}, err
	}
	formInput := proposalFormInput(base, sourceDocument, proposal, contract)
	mutation := FormProposalReviewMutation{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ProposalID: proposal.ID,
		ExpectedVersion: proposal.Version, Status: FormProposalAccepted, ReviewerID: actor.PrincipalID,
		ChangeIDs: changeIDs, At: s.now().UTC(),
	}

	if atomic, ok := s.store.(formProposalAtomicAcceptor); ok {
		preparer, ok := s.forms.(proposalFormPreparer)
		if !ok {
			return FormTemplateProposal{}, errors.Join(ErrInvalid, errors.New("atomic proposal acceptance requires a transaction-capable form preparer"))
		}
		var draft FormTemplate
		if base.ID != "" {
			draft, err = preparer.PrepareFormRevision(ctx, base.ID, CreateFormRevisionInput{ExpectedVersion: base.Version, Form: formInput})
		} else {
			draft, err = preparer.PrepareLibraryForm(ctx, formInput)
		}
		if err != nil {
			return FormTemplateProposal{}, err
		}
		mutation.ResultTemplateID = draft.ID
		mutation.ResultTemplateVersion = draft.Version
		return atomic.AcceptWithDraft(ctx, mutation, draft)
	}

	var draft FormTemplate
	if base.ID != "" {
		draft, err = s.forms.CreateFormRevision(ctx, base.ID, CreateFormRevisionInput{ExpectedVersion: base.Version, Form: formInput})
	} else {
		draft, err = s.forms.CreateLibraryForm(ctx, formInput)
	}
	if err != nil {
		return FormTemplateProposal{}, err
	}
	mutation.ResultTemplateID = draft.ID
	mutation.ResultTemplateVersion = draft.Version
	return s.store.Review(ctx, mutation)
}

func (s *FormProposalService) failGeneration(ctx context.Context, current FormTemplateProposal, code, message string) (FormTemplateProposal, error) {
	return s.store.FailGeneration(ctx, current.TenantID, current.LegalEntityID, current.ID, current.Version, code, message, s.now().UTC())
}

func proposalSourceMatchesDocument(proposal FormTemplateProposal, document documentimport.Document) bool {
	return proposal.SourceKind == FormProposalSourceDocument &&
		proposal.TenantID == document.TenantID && proposal.LegalEntityID == document.LegalEntityID &&
		proposal.SourceDocumentID == document.ID && proposal.SourceDocumentVersion == document.Version &&
		proposal.SourceSHA256 == document.SHA256
}

func proposalAISourceMatchesDocument(proposal FormTemplateProposal, document documentimport.Document) bool {
	return proposal.SourceKind == FormProposalSourceAI && proposal.Provenance.AI != nil &&
		proposal.TenantID == document.TenantID && proposal.LegalEntityID == document.LegalEntityID &&
		proposal.SourceDocumentID == document.ID && proposal.SourceDocumentVersion == document.Version &&
		proposal.Provenance.AI.SourceDocumentSHA256 == document.SHA256
}

func proposalFormInput(base FormTemplate, document *documentimport.Document, proposal FormTemplateProposal, contract formcontract.Contract) CreateFormInput {
	if base.ID != "" {
		return CreateFormInput{
			ProgramID: base.ProgramID, LegalEntityID: base.LegalEntityID, Code: base.Code, Name: base.Name, Purpose: base.Purpose,
			OwnerPrincipalID: base.OwnerPrincipalID, ResponsibleTeam: base.ResponsibleTeam,
			ApprovedUses: append([]string(nil), base.ApprovedUses...), Tags: append([]string(nil), base.Tags...),
			Jurisdiction: base.Jurisdiction, Industry: base.Industry, Sensitivity: base.Sensitivity,
			ScoringMode: contract.ScoringMode, NextReviewAt: base.NextReviewAt,
			Presentation: contract.Presentation, Sections: contract.Sections, Fields: contract.Fields,
		}
	}
	if document != nil {
		name := strings.TrimSpace(document.FileName)
		if extension := filepath.Ext(name); extension != "" {
			name = strings.TrimSpace(strings.TrimSuffix(name, extension))
		}
		name = boundedProposalText(name, 512)
		if name == "" {
			name = "Imported form"
		}
		purpose := boundedProposalText(document.Purpose, 2000)
		if purpose == "" {
			purpose = "Form template derived from an imported document."
		}
		return CreateFormInput{
			Code: proposalDraftCode("IMPORT", proposal.SourceSHA256), Name: name, Purpose: purpose,
			Sensitivity: "INTERNAL", ScoringMode: contract.ScoringMode,
			Presentation: contract.Presentation, Sections: contract.Sections, Fields: contract.Fields,
		}
	}
	return CreateFormInput{
		Code: proposalDraftCode("AI", proposal.SourceSHA256), Name: "AI-assisted form", Purpose: "Draft form template created from a governed AI proposal.",
		Sensitivity: "INTERNAL", ScoringMode: contract.ScoringMode,
		Presentation: contract.Presentation, Sections: contract.Sections, Fields: contract.Fields,
	}
}

func proposalDraftCode(prefix, sha string) string {
	suffix := strings.TrimSpace(sha)
	if len(suffix) > 12 {
		suffix = suffix[:12]
	}
	if suffix == "" {
		suffix = "DRAFT"
	}
	return prefix + "-" + strings.ToUpper(suffix)
}
