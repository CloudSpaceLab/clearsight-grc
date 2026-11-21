package apiquestion

// AdoptionManualDraft pins a question selected by a Program owner. It is a
// proposal only and does not alter the adopted form or assessment execution.
type AdoptionManualDraft struct {
	ProgramID       string
	ProgramVersion  int64
	AdoptionID      string
	AdoptionVersion int64
	FormID          string
	FormVersion     int64
	FieldID         string
	ExpectedVersion int64
	Rationale       string
	Key             string
}

type AdoptionQuestionDraft struct {
	Version   int64  `json:"version"`
	Mode      string `json:"mode"`
	Rationale string `json:"rationale,omitempty"`
}

// AdoptionQuestionApproval pins the exact proposal seen by an independent reviewer.
type AdoptionQuestionApproval struct {
	ProgramID       string
	ProgramVersion  int64
	AdoptionID      string
	AdoptionVersion int64
	DraftID         string
	DraftVersion    int64
	Rationale       string
	Key             string
}

type AdoptionQuestionCandidate struct {
	DraftID      string `json:"draft_id"`
	DraftVersion int64  `json:"draft_version"`
	FormID       string `json:"form_id"`
	FormVersion  int64  `json:"form_version"`
	FieldID      string `json:"field_id"`
	Mode         string `json:"mode"`
	Rationale    string `json:"rationale,omitempty"`
}
