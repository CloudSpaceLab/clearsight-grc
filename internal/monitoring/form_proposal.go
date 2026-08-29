package monitoring

import (
	"context"
	"errors"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

var (
	ErrFormProposalSourceChanged = errors.New("form proposal source changed")
	ErrFormProposalState         = errors.New("form proposal is not in the required state")
	ErrFormProposalSelection     = errors.New("form proposal selection is invalid")
)

type FormProposalSourceKind string

const (
	FormProposalSourceDocument FormProposalSourceKind = "DOCUMENT"
	FormProposalSourceAI       FormProposalSourceKind = "AI"
)

type FormProposalStatus string

const (
	FormProposalGenerating     FormProposalStatus = "GENERATING"
	FormProposalReviewRequired FormProposalStatus = "REVIEW_REQUIRED"
	FormProposalAccepted       FormProposalStatus = "ACCEPTED"
	FormProposalRejected       FormProposalStatus = "REJECTED"
	FormProposalFailed         FormProposalStatus = "FAILED"
)

type FormTemplateProposal struct {
	ID                    string                                  `json:"id"`
	TenantID              string                                  `json:"-"`
	LegalEntityID         string                                  `json:"-"`
	SourceKind            FormProposalSourceKind                  `json:"source_kind"`
	SourceDocumentID      string                                  `json:"source_document_id,omitempty"`
	SourceDocumentVersion int64                                   `json:"source_document_version,omitempty"`
	SourceSHA256          string                                  `json:"source_sha256,omitempty"`
	BaseTemplateID        string                                  `json:"base_template_id,omitempty"`
	BaseTemplateVersion   int64                                   `json:"base_template_version,omitempty"`
	Status                FormProposalStatus                      `json:"status"`
	ProposedContract      formcontract.Contract                   `json:"proposed_contract"`
	FieldChanges          []documentimport.FormFieldChange        `json:"field_changes"`
	UnresolvedItems       []documentimport.ProposalUnresolvedItem `json:"unresolved_items"`
	Provenance            documentimport.FormProposalProvenance   `json:"provenance"`
	FailureCode           string                                  `json:"failure_code,omitempty"`
	FailureMessage        string                                  `json:"failure_message,omitempty"`
	CreatedBy             string                                  `json:"created_by"`
	ReviewedBy            string                                  `json:"reviewed_by,omitempty"`
	AcceptedChangeIDs     []string                                `json:"accepted_change_ids,omitempty"`
	ResultTemplateID      string                                  `json:"result_template_id,omitempty"`
	ResultTemplateVersion int64                                   `json:"result_template_version,omitempty"`
	CreatedAt             time.Time                               `json:"created_at"`
	UpdatedAt             time.Time                               `json:"updated_at"`
	ReviewedAt            *time.Time                              `json:"reviewed_at,omitempty"`
	Version               int64                                   `json:"version"`
}

type RequestDocumentFormProposalInput struct {
	ExpectedDocumentVersion int64  `json:"expected_document_version"`
	BaseTemplateID          string `json:"base_template_id,omitempty"`
	BaseTemplateVersion     int64  `json:"base_template_version,omitempty"`
}

type AcceptFormProposalInput struct {
	ExpectedVersion int64    `json:"expected_version"`
	ChangeIDs       []string `json:"change_ids"`
}

type RejectFormProposalInput struct {
	ExpectedVersion int64 `json:"expected_version"`
}

type FormProposalReviewMutation struct {
	TenantID              string
	LegalEntityID         string
	ProposalID            string
	ExpectedVersion       int64
	Status                FormProposalStatus
	ReviewerID            string
	ChangeIDs             []string
	ResultTemplateID      string
	ResultTemplateVersion int64
	At                    time.Time
}

type FormProposalStore interface {
	Create(context.Context, FormTemplateProposal) (FormTemplateProposal, error)
	Get(context.Context, string, string, string) (FormTemplateProposal, error)
	CompleteGeneration(context.Context, FormTemplateProposal, int64) (FormTemplateProposal, error)
	FailGeneration(context.Context, string, string, string, int64, string, string, time.Time) (FormTemplateProposal, error)
	Review(context.Context, FormProposalReviewMutation) (FormTemplateProposal, error)
	QueuesGeneration() bool
}

type formProposalDocumentReader interface {
	Get(context.Context, string, string) (documentimport.Document, error)
}

func cloneFormTemplateProposal(value FormTemplateProposal) FormTemplateProposal {
	cloned := value
	cloned.ProposedContract = cloneProposalContract(value.ProposedContract)
	cloned.FieldChanges = append([]documentimport.FormFieldChange(nil), value.FieldChanges...)
	for index := range cloned.FieldChanges {
		cloned.FieldChanges[index].Field = cloneTemplateField(value.FieldChanges[index].Field)
		cloned.FieldChanges[index].Anchor = cloneProposalAnchor(value.FieldChanges[index].Anchor)
		cloned.FieldChanges[index].Unresolved = append([]string(nil), value.FieldChanges[index].Unresolved...)
	}
	cloned.UnresolvedItems = append([]documentimport.ProposalUnresolvedItem(nil), value.UnresolvedItems...)
	for index := range cloned.UnresolvedItems {
		if value.UnresolvedItems[index].Anchor != nil {
			anchor := cloneProposalAnchor(*value.UnresolvedItems[index].Anchor)
			cloned.UnresolvedItems[index].Anchor = &anchor
		}
	}
	cloned.AcceptedChangeIDs = append([]string(nil), value.AcceptedChangeIDs...)
	return cloned
}

func cloneProposalContract(value formcontract.Contract) formcontract.Contract {
	cloned := value
	cloned.Sections = make([]formcontract.Section, len(value.Sections))
	for index := range value.Sections {
		cloned.Sections[index] = cloneProposalSection(value.Sections[index])
	}
	cloned.Fields = make([]formcontract.Field, len(value.Fields))
	for index := range value.Fields {
		cloned.Fields[index] = cloneTemplateField(value.Fields[index])
	}
	return cloned
}

func cloneTemplateField(value formcontract.Field) formcontract.Field {
	cloned := value
	cloned.Options = append([]string(nil), value.Options...)
	cloned.AcceptedFormats = append([]string(nil), value.AcceptedFormats...)
	cloned.Constraints.MinLength = cloneProposalPointer(value.Constraints.MinLength)
	cloned.Constraints.MaxLength = cloneProposalPointer(value.Constraints.MaxLength)
	cloned.Constraints.Minimum = cloneProposalPointer(value.Constraints.Minimum)
	cloned.Constraints.Maximum = cloneProposalPointer(value.Constraints.Maximum)
	cloned.Constraints.Step = cloneProposalPointer(value.Constraints.Step)
	cloned.Constraints.DecimalPrecision = cloneProposalPointer(value.Constraints.DecimalPrecision)
	cloned.Constraints.MinSelections = cloneProposalPointer(value.Constraints.MinSelections)
	cloned.Constraints.MaxSelections = cloneProposalPointer(value.Constraints.MaxSelections)
	cloned.Constraints.MinFiles = cloneProposalPointer(value.Constraints.MinFiles)
	cloned.Constraints.MaxFiles = cloneProposalPointer(value.Constraints.MaxFiles)
	cloned.Constraints.MaxFileBytes = cloneProposalPointer(value.Constraints.MaxFileBytes)
	cloned.Constraints.MaxTotalFileBytes = cloneProposalPointer(value.Constraints.MaxTotalFileBytes)
	if value.Condition != nil {
		condition := *value.Condition
		condition.Values = append([]string(nil), value.Condition.Values...)
		cloned.Condition = &condition
	}
	if value.Scoring != nil {
		scoring := *value.Scoring
		scoring.AnswerScores = make(map[string]int, len(value.Scoring.AnswerScores))
		for key, score := range value.Scoring.AnswerScores {
			scoring.AnswerScores[key] = score
		}
		scoring.CriticalAnswers = append([]string(nil), value.Scoring.CriticalAnswers...)
		cloned.Scoring = &scoring
	}
	if value.RecordTarget != nil {
		target := *value.RecordTarget
		cloned.RecordTarget = &target
	}
	return cloned
}

func cloneProposalAnchor(value documentimport.SourceAnchor) documentimport.SourceAnchor {
	cloned := value
	if value.BoundingBox != nil {
		box := *value.BoundingBox
		cloned.BoundingBox = &box
	}
	return cloned
}

func cloneProposalPointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
