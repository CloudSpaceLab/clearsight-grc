package apiquestion

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

// bindingManifest is an internal preview contract, not an activation result.
type bindingManifest struct {
	TenantID          string                `json:"tenant_id"`
	LegalEntityID     string                `json:"legal_entity_id"`
	FormID            string                `json:"form_id"`
	DraftRevisionID   string                `json:"draft_revision_id"`
	PendingRevisionID string                `json:"pending_revision_id"`
	DraftVersion      int64                 `json:"draft_version"`
	PendingVersion    int64                 `json:"pending_version"`
	Items             []bindingManifestItem `json:"items"`
}
type bindingManifestItem struct {
	FieldID               string `json:"field_id"`
	QuestionHash          string `json:"question_hash"`
	DraftID               string `json:"draft_id"`
	DraftVersion          int64  `json:"draft_version"`
	ProposalID            string `json:"proposal_id"`
	ProposalVersion       int64  `json:"proposal_version"`
	ApprovalID            string `json:"approval_id"`
	QuestionMakerID       string `json:"question_maker_id"`
	ProposalMakerID       string `json:"proposal_maker_id"`
	ReviewerID            string `json:"reviewer_id"`
	ApprovalRuleID        string `json:"approval_rule_id"`
	ApprovalPolicyVersion string `json:"approval_policy_version"`
	BindingID             string `json:"binding_id"`
	BindingVersion        int64  `json:"binding_version"`
	SourceID              string `json:"source_id"`
	ViewID                string `json:"view_id"`
	ViewVersion           int64  `json:"view_version"`
	ConnectionID          string `json:"connection_id"`
	ConnectionVersion     int64  `json:"connection_version"`
	EndpointHash          string `json:"endpoint_hash"`
	ResultType            string `json:"result_type"`
}

func bindingManifestDigest(input bindingManifest) (string, error) {
	if !manifestID(input.TenantID) || !manifestID(input.LegalEntityID) || !manifestID(input.FormID) || !manifestID(input.DraftRevisionID) || !manifestID(input.PendingRevisionID) || input.DraftRevisionID == input.PendingRevisionID || input.DraftVersion < 1 || input.PendingVersion != input.DraftVersion+1 || len(input.Items) < 1 || len(input.Items) > 200 {
		return "", ErrInvalid
	}
	items := append([]bindingManifestItem(nil), input.Items...)
	sort.Slice(items, func(i, j int) bool { return items[i].FieldID < items[j].FieldID })
	fields, proposals, approvals := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, item := range items {
		if !manifestID(item.FieldID) || len(item.FieldID) > 80 || !manifestID(item.DraftID) || !manifestID(item.ProposalID) || !manifestID(item.ApprovalID) || !manifestID(item.QuestionMakerID) || !manifestID(item.ProposalMakerID) || !manifestID(item.ReviewerID) || item.ReviewerID == item.QuestionMakerID || item.ReviewerID == item.ProposalMakerID || !manifestID(item.ApprovalRuleID) || !manifestID(item.ApprovalPolicyVersion) || !manifestID(item.BindingID) || !manifestID(item.SourceID) || !manifestID(item.ViewID) || !manifestID(item.ConnectionID) || item.DraftVersion < 1 || item.ProposalVersion < 1 || item.BindingVersion < 1 || item.ViewVersion < 1 || item.ConnectionVersion < 1 || !manifestHash(item.QuestionHash) || !manifestHash(item.EndpointHash) || (item.ResultType != "BOOLEAN" && item.ResultType != "NUMBER") || fields[item.FieldID] || proposals[item.ProposalID] || approvals[item.ApprovalID] {
			return "", ErrInvalid
		}
		fields[item.FieldID] = true
		proposals[item.ProposalID] = true
		approvals[item.ApprovalID] = true
	}
	input.Items = items
	encoded, err := json.Marshal(struct {
		Schema   string          `json:"schema"`
		Manifest bindingManifest `json:"manifest"`
	}{"form-binding-impact-v2", input})
	if err != nil {
		return "", ErrInvalid
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}
func manifestID(value string) bool {
	return value != "" && len(value) <= 128 && strings.TrimSpace(value) == value
}
func manifestHash(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
