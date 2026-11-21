package apiquestion

// BindingProposal is a proposed exact association; it is not active.
type BindingProposal struct {
	FormID          string
	FormVersion     int64
	FieldID         string
	DraftID         string
	DraftVersion    int64
	BindingID       string
	BindingVersion  int64
	ExpectedVersion int64
	Key             string
}

type BindingReviewInput struct {
	FormID          string
	FormVersion     int64
	ProposalID      string
	ProposalVersion int64
	Rationale       string
	Key             string
}

type BindingReviewCandidate struct {
	ProposalID      string `json:"proposal_id"`
	ProposalVersion int64  `json:"proposal_version"`
	FormID          string `json:"form_id"`
	FormVersion     int64  `json:"form_version"`
	FieldID         string `json:"field_id"`
	BindingID       string `json:"binding_id"`
	BindingVersion  int64  `json:"binding_version"`
	ResultType      string `json:"result_type"`
}
