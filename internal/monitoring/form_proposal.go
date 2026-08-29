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
	TenantID             string
	LegalEntityID        string
	ProposalID           string
	ExpectedVersion      int64
	Status               FormProposalStatus
	ReviewerID           string
	ResultTemplateID     string
	ResultTemplateVersion int64
	At                   time.Time
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
		cloned.FieldChanges[index].Unresolved = append([]string(nil), value.FieldChanges[index].Unresolved...)
	}
	cloned.UnresolvedItems = append([]documentimport.ProposalUnresolvedItem(nil), value.UnresolvedItems...)
	return cloned
}

func cloneProposalContract(value formcontract.Contract) formcontract.Contract {
	cloned := value
	cloned.Sections = append([]formcontract.Section(nil), value.Sections...)
	cloned.Fields = make([]formcontract.Field, len(value.Fields))
	for index := range value.Fields {
		cloned.Fields[index] = cloneTemplateField(value.Fields[index])
	}
	return cloned
}

func cloneTemplateField(value formcontract.Field) formcontract.Field {
	cloned := value
	cloned.Options = append([]string(nil), value.Options...)
	return cloned
}
