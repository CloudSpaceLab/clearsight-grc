package documentimport

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	EventDocumentProposalAccepted       = "DocumentProposalAccepted"
	EventDocumentProposalHandoffChanged = "DocumentProposalHandoffChanged"
)

type HandoffReviewAction string
type HandoffAuthorizationAction string

const (
	HandoffReviewReturn HandoffReviewAction = "RETURN"
	HandoffReviewReject HandoffReviewAction = "REJECT"
	HandoffReviewSubmit HandoffReviewAction = "SUBMIT_FOR_AUTHORIZATION"

	HandoffAuthorizeReturn HandoffAuthorizationAction = "RETURN"
	HandoffAuthorizeReject HandoffAuthorizationAction = "REJECT"
	HandoffAuthorizeApprove HandoffAuthorizationAction = "APPROVE"
)

type HandoffReviewInput struct {
	TenantID             string                     `json:"tenant_id,omitempty"`
	DocumentID           string                     `json:"document_id,omitempty"`
	ProposalID           string                     `json:"proposal_id,omitempty"`
	ActorID              string                     `json:"actor_id,omitempty"`
	Action               HandoffReviewAction        `json:"action"`
	ExpectedDocumentVersion int64                    `json:"expected_document_version"`
	ExpectedHandoffVersion  int64                    `json:"expected_handoff_version"`
	Title                string                     `json:"title,omitempty"`
	Statement            string                     `json:"statement,omitempty"`
	TargetType           ConversionTarget           `json:"target_type,omitempty"`
	TargetProgramID      string                     `json:"target_program_id,omitempty"`
	TargetProgramVersion int64                      `json:"target_program_version,omitempty"`
	Note                 string                     `json:"note,omitempty"`
}

type HandoffAuthorizationInput struct {
	TenantID               string                     `json:"tenant_id,omitempty"`
	DocumentID             string                     `json:"document_id,omitempty"`
	ProposalID             string                     `json:"proposal_id,omitempty"`
	ActorID                string                     `json:"actor_id,omitempty"`
	Action                 HandoffAuthorizationAction `json:"action"`
	ExpectedDocumentVersion int64                     `json:"expected_document_version"`
	ExpectedHandoffVersion  int64                     `json:"expected_handoff_version"`
	Note                   string                     `json:"note,omitempty"`
	ResultObjectType       string                     `json:"result_object_type,omitempty"`
	ResultObjectID         string                     `json:"result_object_id,omitempty"`
}

func proposalHandoffID(documentID, proposalID string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(documentID) + "\x00document-proposal-handoff\x00" + strings.TrimSpace(proposalID)))
	raw := append([]byte(nil), digest[:16]...)
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(raw)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]
}

func handoffForAcceptedProposal(input ReviewInput, title, statement string, nowTime interface{ UTC() }) *ProposalHandoff {
	// This helper intentionally accepts only the UTC capability so callers cannot
	// accidentally derive authority or lifecycle state from a wall-clock wrapper.
	_ = nowTime
	return nil
}
