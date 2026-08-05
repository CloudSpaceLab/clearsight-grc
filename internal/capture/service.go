package capture

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrRequestNotFound = errors.New("capture request not found")
	ErrVersionConflict = errors.New("capture request version conflict")
	ErrRequestClosed   = errors.New("capture request is not open")
)

type ValidationError struct {
	FieldID string
	Message string
}

func (e ValidationError) Error() string { return e.Message }

type Service struct {
	mu       sync.RWMutex
	requests map[string]Request
	now      func() time.Time
}

func NewService(requests []Request) *Service {
	stored := make(map[string]Request, len(requests))
	for _, request := range requests {
		request.KnownFacts = cloneMap(request.KnownFacts)
		request.Answers = cloneMap(request.Answers)
		stored[request.ID] = request
	}
	return &Service{requests: stored, now: time.Now}
}

func (s *Service) Get(id string) (Request, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	request, ok := s.requests[id]
	if !ok {
		return Request{}, ErrRequestNotFound
	}
	request.KnownFacts = cloneMap(request.KnownFacts)
	request.Answers = cloneMap(request.Answers)
	request.Fields = append([]Field(nil), request.Fields...)
	return request, nil
}

func (s *Service) Submit(id string, submission Submission) (Receipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	request, ok := s.requests[id]
	if !ok {
		return Receipt{}, ErrRequestNotFound
	}
	if request.Status == StatusSubmitted || request.Status == StatusCancelled {
		return Receipt{}, ErrRequestClosed
	}
	if request.Version != submission.Version {
		return Receipt{}, ErrVersionConflict
	}
	for _, field := range request.Fields {
		if field.Required && strings.TrimSpace(submission.Answers[field.ID]) == "" {
			return Receipt{}, ValidationError{FieldID: field.ID, Message: fmt.Sprintf("%s is required", field.Label)}
		}
	}
	request.Answers = cloneMap(submission.Answers)
	request.Status = StatusSubmitted
	request.Version++
	s.requests[id] = request
	return Receipt{RequestID: id, Status: request.Status, SubmittedAt: s.now().UTC(), Version: request.Version}, nil
}

func cloneMap(input map[string]string) map[string]string {
	if input == nil {
		return map[string]string{}
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func DemoRequests() []Request {
	return []Request{{
		ID: "req_branch_generator", Title: "Confirm branch backup-power condition", Purpose: "Resolve the only missing fact for the August resilience review.", WhyYou: "You are the current Enugu Main Branch operations manager.", Status: StatusReady, Sensitivity: "Internal", EstimatedMinutes: 2, Deadline: time.Now().UTC().Add(48 * time.Hour),
		KnownFacts: map[string]string{"Branch": "Enugu Main Branch", "Last service": "2026-07-18", "Maintenance firm": "Northstar Engineering"},
		Fields: []Field{{ID: "condition", Label: "Current generator condition", Type: "single_select", Required: true, Options: []string{"Operational", "Operational with concern", "Unavailable"}}, {ID: "concern", Label: "Concern or supporting note", Type: "text", Description: "Add only information relevant to the current condition."}},
		Version: 1,
	}}
}
