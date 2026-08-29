package monitoring

import (
	"context"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

const formAIPromptVersionDefault = "FORM_AUTHORING_V1"

type RequestAIFormProposalInput struct {
	Objective                     string   `json:"objective"`
	SourceDocumentID              string   `json:"source_document_id,omitempty"`
	ExpectedSourceDocumentVersion int64    `json:"expected_source_document_version,omitempty"`
	SourceElementRefs             []string `json:"source_element_refs,omitempty"`
}

type FormAISourceSnapshot struct {
	DocumentID string
	Version    int64
	SHA256     string
	Elements   []documentimport.ExtractedElement
}

type FormAIClientRequest struct {
	TenantID            string
	LegalEntityID       string
	PrincipalID         string
	Objective           string
	SnapshotSHA256      string
	BaseTemplateID      string
	BaseTemplateVersion int64
	BaseContract        formcontract.Contract
	Source              *FormAISourceSnapshot
}

type FormAIProvenance struct {
	WorkloadID           string   `json:"workload_id"`
	PolicyRef            string   `json:"policy_ref,omitempty"`
	GatewayRequestID     string   `json:"gateway_request_id,omitempty"`
	GatewayResponseID    string   `json:"gateway_response_id,omitempty"`
	RouteID              string   `json:"route_id,omitempty"`
	ModelAlias           string   `json:"model_alias"`
	PromptVersion        string   `json:"prompt_version"`
	SnapshotSHA256       string   `json:"snapshot_sha256"`
	SourceDocumentSHA256 string   `json:"source_document_sha256,omitempty"`
	SourceElementRefs    []string `json:"source_element_refs,omitempty"`
	ValidationResults    []string `json:"validation_results"`
}

type FormProposalProvenance struct {
	documentimport.FormProposalProvenance
	AI *FormAIProvenance `json:"ai,omitempty"`
}

type FormAIClientResult struct {
	Contract        formcontract.Contract
	FieldChanges    []documentimport.FormFieldChange
	UnresolvedItems []documentimport.ProposalUnresolvedItem
	Provenance      FormAIProvenance
}

type FormAIClient interface {
	Propose(context.Context, FormAIClientRequest) (FormAIClientResult, error)
}
