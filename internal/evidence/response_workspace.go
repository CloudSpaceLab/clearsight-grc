package evidence

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

var (
	ErrWorkspaceUnavailable = errors.New("response workspace is unavailable")
	ErrWorkspaceConflict    = errors.New("response workspace has changed")
)

type FieldEdit struct {
	FieldID      string                   `json:"field_id"`
	Value        formcontract.AnswerValue `json:"value"`
	BaseSequence int64                    `json:"base_sequence"`
}

type SaveWorkspaceInput struct {
	ExpectedVersion  int64                         `json:"expected_version"`
	PresentationMode formcontract.PresentationMode `json:"presentation_mode"`
	Edits            []FieldEdit                   `json:"edits"`
}

type FieldChange struct {
	FieldID     string                   `json:"field_id"`
	ServerValue formcontract.AnswerValue `json:"server_value"`
	Sequence    int64                    `json:"sequence"`
}

type WorkspaceConflict struct {
	CurrentVersion int64         `json:"current_version"`
	Changed        []FieldChange `json:"changed_fields"`
}

func (conflict WorkspaceConflict) Error() string { return ErrWorkspaceConflict.Error() }
func (conflict WorkspaceConflict) Unwrap() error { return ErrWorkspaceConflict }

type SubmitWorkspaceInput struct {
	ExpectedVersion     int64    `json:"expected_version"`
	AttestationFieldIDs []string `json:"attestation_field_ids,omitempty"`
}

type WorkspaceFieldProvenance struct {
	RecipientID string          `json:"recipient_id"`
	RequestID   string          `json:"request_id"`
	Assurance   AccessAssurance `json:"assurance"`
	Sequence    int64           `json:"sequence"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type ResponseWorkspaceView struct {
	Workspace        ResponseWorkspace                   `json:"workspace"`
	Answers          map[string]formcontract.AnswerValue `json:"answers"`
	PresentationMode formcontract.PresentationMode       `json:"presentation_mode"`
	FieldSequences   map[string]int64                    `json:"field_sequences"`
	FieldProvenance  map[string]WorkspaceFieldProvenance `json:"field_provenance,omitempty"`
	CurrentRevision  *ResponseRevision                   `json:"current_revision,omitempty"`
}

type WorkspaceSubmissionResult struct {
	Workspace  ResponseWorkspace `json:"workspace"`
	Revision   ResponseRevision  `json:"revision"`
	Submission SubmissionReceipt `json:"submission"`
}

type workspaceEditRecord struct {
	ID               string
	TenantID         string
	LegalEntityID    string
	DistributionID   string
	WorkspaceID      string
	RecipientID      string
	RequestID        string
	BaseVersion      int64
	ResultVersion    int64
	FieldID          string
	Value            formcontract.AnswerValue
	PresentationMode formcontract.PresentationMode
	SessionID        string
	RouteID          string
	Assurance        AccessAssurance
	CreatedAt        time.Time
}

type workspaceSaveCommand struct {
	Session  DistributionAccessSession
	Request  Request
	Input    SaveWorkspaceInput
	Now      time.Time
	Validate func(map[string]formcontract.AnswerValue) error
}

type workspaceSubmitCommand struct {
	Session       DistributionAccessSession
	Request       Request
	Input         SubmitWorkspaceInput
	Now           time.Time
	Validate      func(map[string]formcontract.AnswerValue) error
	BuildRevision func(map[string]formcontract.AnswerValue) (ResponseRevision, error)
}

type responseWorkspaceStore interface {
	GetResponseWorkspace(context.Context, DistributionAccessSession, Request, time.Time) (ResponseWorkspaceView, error)
	SaveResponseWorkspace(context.Context, workspaceSaveCommand) (ResponseWorkspaceView, error)
	SubmitResponseWorkspace(context.Context, workspaceSubmitCommand) (WorkspaceSubmissionResult, error)
	ClearResponseWorkspace(context.Context, workspaceSaveCommand) (ResponseWorkspaceView, error)
}

type workspaceAnswerValidationProvider interface {
	ValidateWorkspaceAnswers(context.Context, DistributionAccessSession, Request, map[string]formcontract.AnswerValue, bool) error
}

func applyWorkspaceEdit(answers map[string]formcontract.AnswerValue, edit FieldEdit) {
	if !edit.Value.Answered() {
		delete(answers, edit.FieldID)
		return
	}
	answers[edit.FieldID] = cloneAnswerValues(map[string]formcontract.AnswerValue{edit.FieldID: edit.Value})[edit.FieldID]
}

func answerValueEqual(left, right formcontract.AnswerValue) bool {
	return reflect.DeepEqual(left, right)
}

func cloneResponseWorkspaceView(value ResponseWorkspaceView) ResponseWorkspaceView {
	value.Answers = cloneAnswerValues(value.Answers)
	value.FieldSequences = cloneInt64Map(value.FieldSequences)
	if value.FieldProvenance == nil {
		value.FieldProvenance = map[string]WorkspaceFieldProvenance{}
	} else {
		provenance := make(map[string]WorkspaceFieldProvenance, len(value.FieldProvenance))
		for key, item := range value.FieldProvenance {
			provenance[key] = item
		}
		value.FieldProvenance = provenance
	}
	if value.CurrentRevision != nil {
		revision := cloneResponseRevision(*value.CurrentRevision)
		value.CurrentRevision = &revision
	}
	return value
}

func cloneInt64Map(value map[string]int64) map[string]int64 {
	result := make(map[string]int64, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func cloneResponseRevision(value ResponseRevision) ResponseRevision {
	value.SignoffSummary = cloneAnyMap(value.SignoffSummary)
	if value.Score != nil {
		score := *value.Score
		if value.Score.RawScore != nil {
			raw := *value.Score.RawScore
			score.RawScore = &raw
		}
		if value.Score.AdverseScore != nil {
			adverse := *value.Score.AdverseScore
			score.AdverseScore = &adverse
		}
		score.ContributionResults = append([]formcontract.ContributionResult(nil), value.Score.ContributionResults...)
		score.RuleResults = append([]formcontract.AdvancedRuleResult(nil), value.Score.RuleResults...)
		value.Score = &score
	}
	if value.ComplianceScore != nil {
		score := *value.ComplianceScore
		value.ComplianceScore = &score
	}
	if value.CriticalFieldResults == nil {
		value.CriticalFieldResults = []map[string]any{}
	} else {
		results := make([]map[string]any, len(value.CriticalFieldResults))
		for index, item := range value.CriticalFieldResults {
			results[index] = cloneAnyMap(item)
		}
		value.CriticalFieldResults = results
	}
	return value
}

func workspaceDefaultView(workspace ResponseWorkspace, request Request) ResponseWorkspaceView {
	return ResponseWorkspaceView{
		Workspace:        workspace,
		Answers:          map[string]formcontract.AnswerValue{},
		PresentationMode: defaultDraftPresentation(request),
		FieldSequences:   map[string]int64{},
		FieldProvenance:  map[string]WorkspaceFieldProvenance{},
	}
}
