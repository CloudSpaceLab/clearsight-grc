package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

const maxDraftAnswersBytes = 1 << 20

type RequestOrigin struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Version int64  `json:"version"`
}

func (o RequestOrigin) normalized() RequestOrigin {
	o.Type = strings.TrimSpace(o.Type)
	o.ID = strings.TrimSpace(o.ID)
	return o
}

func (o RequestOrigin) empty() bool {
	return o.Type == "" && o.ID == "" && o.Version == 0
}

func (o RequestOrigin) validate() error {
	o = o.normalized()
	if o.empty() {
		return nil
	}
	if o.Type == "" || o.ID == "" || o.Version < 1 || len(o.Type) > 128 || len(o.ID) > 512 {
		return fmt.Errorf("request origin type, id and positive version must be provided together")
	}
	return nil
}

type ResponseDraft struct {
	ID               string                              `json:"id"`
	TenantID         string                              `json:"tenant_id"`
	RequestID        string                              `json:"request_id"`
	SessionID        string                              `json:"session_id"`
	Answers          map[string]formcontract.AnswerValue `json:"answers"`
	PresentationMode formcontract.PresentationMode       `json:"presentation_mode"`
	Version          int64                               `json:"version"`
	CreatedAt        time.Time                           `json:"created_at"`
	UpdatedAt        time.Time                           `json:"updated_at"`
}

type SaveDraftInput struct {
	Answers          map[string]formcontract.AnswerValue `json:"answers"`
	PresentationMode formcontract.PresentationMode       `json:"presentation_mode"`
	ExpectedVersion  int64                               `json:"expected_version"`
}

type SaveDraftRecord struct {
	ID               string
	TenantID         string
	RequestID        string
	SessionID        string
	Answers          map[string]formcontract.AnswerValue
	PresentationMode formcontract.PresentationMode
	ExpectedVersion  int64
	UpdatedAt        time.Time
}

func (s *Service) GetRequestByOrigin(ctx context.Context, tenant string, origin RequestOrigin) (Request, error) {
	tenant = strings.TrimSpace(tenant)
	origin = origin.normalized()
	if tenant == "" || origin.empty() || origin.validate() != nil {
		return Request{}, fmt.Errorf("tenant and complete request origin are required")
	}
	store, ok := s.repo.(OriginRequestStore)
	if !ok {
		return Request{}, fmt.Errorf("request origin lookup is unavailable")
	}
	value, err := store.GetRequestByOrigin(ctx, tenant, origin)
	if err != nil {
		return Request{}, err
	}
	value, err = hydrateRequestRecipient(ctx, s.repo, value)
	if err != nil {
		return Request{}, err
	}
	return effectiveRequest(value, s.now().UTC()), nil
}

func (s *Service) GetDraft(ctx context.Context, sessionToken string) (ResponseDraft, error) {
	session, _, err := s.SessionRequest(ctx, sessionToken)
	if err != nil {
		return ResponseDraft{}, err
	}
	store, ok := s.repo.(DraftStore)
	if !ok {
		return ResponseDraft{}, fmt.Errorf("response drafts are unavailable")
	}
	return store.GetDraft(ctx, session.TenantID, session.RequestID, session.ID)
}

func (s *Service) SaveDraft(ctx context.Context, sessionToken string, input SaveDraftInput) (ResponseDraft, error) {
	if input.ExpectedVersion < 0 {
		return ResponseDraft{}, fmt.Errorf("expected_version must not be negative")
	}
	if input.PresentationMode != formcontract.PresentationClassic && input.PresentationMode != formcontract.PresentationWizard && input.PresentationMode != formcontract.PresentationAutomatic {
		return ResponseDraft{}, fmt.Errorf("presentation_mode is invalid")
	}
	if input.Answers == nil {
		input.Answers = map[string]formcontract.AnswerValue{}
	}
	encoded, err := json.Marshal(input.Answers)
	if err != nil || len(encoded) > maxDraftAnswersBytes {
		return ResponseDraft{}, fmt.Errorf("draft answers exceed the permitted size")
	}
	session, _, err := s.SessionRequest(ctx, sessionToken)
	if err != nil {
		return ResponseDraft{}, err
	}
	store, ok := s.repo.(DraftStore)
	if !ok {
		return ResponseDraft{}, fmt.Errorf("response drafts are unavailable")
	}
	draftID, err := id.NewUUIDv7()
	if err != nil {
		return ResponseDraft{}, err
	}
	return store.SaveDraft(ctx, SaveDraftRecord{
		ID: draftID, TenantID: session.TenantID, RequestID: session.RequestID, SessionID: session.ID,
		Answers: cloneAnswerValues(input.Answers), PresentationMode: input.PresentationMode,
		ExpectedVersion: input.ExpectedVersion, UpdatedAt: s.now().UTC(),
	})
}

func (s *Service) DeleteDraft(ctx context.Context, sessionToken string) error {
	session, _, err := s.SessionRequest(ctx, sessionToken)
	if err != nil {
		return err
	}
	store, ok := s.repo.(DraftStore)
	if !ok {
		return fmt.Errorf("response drafts are unavailable")
	}
	return store.DeleteDraft(ctx, session.TenantID, session.RequestID, session.ID)
}
