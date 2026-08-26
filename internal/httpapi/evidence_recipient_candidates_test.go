package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestEvidenceRecipientCandidatesUseVerifiedRequesterScopeAndSafeLabels(t *testing.T) {
	now := time.Date(2026, 8, 26, 14, 0, 0, 0, time.UTC)
	requestValue := evidence.Request{
		ID: "request-1", TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "PROGRAM", SubjectID: "program-1",
		AudienceType: "INTERNAL", Status: evidence.RequestReady, CreatedBy: "requester", Deadline: now.Add(time.Hour),
		Recipient: evidence.Recipient{Type: evidence.RecipientInternalPrincipal, PrincipalID: "candidate", State: evidence.RecipientStateAssigned},
	}
	repo := evidence.NewMemoryRepositoryWithRecipientCandidates(nil, []evidence.Request{requestValue}, []evidence.RecipientCandidate{{
		PrincipalID: "candidate", DisplayName: "Ada Candidate", ContextLabel: "Operations manager", TenantID: "bank", LegalEntityIDs: []string{"entity-1"},
		Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program-1": true},
	}, {
		PrincipalID: "requester", DisplayName: "Reni Requester", TenantID: "bank", LegalEntityIDs: []string{"entity-1"},
		Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program-1": true},
	}})
	service := evidence.NewServiceWithClock(repo, nil, func() time.Time { return now })
	api := &API{deps: Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Evidence: service}}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/request-1/recipient-candidates?limit=50&q=operations", nil)
	request.SetPathValue("id", requestValue.ID)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-1", PrincipalID: "requester", ExpiresAt: now.Add(time.Hour)}))
	response := httptest.NewRecorder()

	api.listEvidenceRecipientCandidates(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("candidate read status = %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Items   []evidence.RecipientCandidate `json:"items"`
		HasMore bool                          `json:"has_more"`
	}
	raw := response.Body.String()
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].PrincipalID != "candidate" || body.Items[0].DisplayName != "Ada Candidate" || body.Items[0].ContextLabel != "Operations manager" || body.HasMore {
		t.Fatalf("unexpected candidate response: %#v", body.Items)
	}
	if containsAny(raw, "tenant_id", "legal_entity_ids", "readable_subjects", "active", "kind") {
		t.Fatalf("candidate response exposed directory metadata: %s", raw)
	}

	detail := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/request-1", nil)
	detail.SetPathValue("id", requestValue.ID)
	detail = detail.WithContext(request.Context())
	detailResponse := httptest.NewRecorder()
	api.getEvidenceRequest(detailResponse, detail)
	if detailResponse.Code != http.StatusOK || !containsAny(detailResponse.Body.String(), `"display_name":"Ada Candidate"`) {
		t.Fatalf("request detail omitted safe recipient label: %d %s", detailResponse.Code, detailResponse.Body.String())
	}
}

func TestEvidenceRecipientCandidatesHideRequestFromRequesterWithoutCurrentEntityMembership(t *testing.T) {
	now := time.Now().UTC()
	requestValue := evidence.Request{
		ID: "request-1", TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "PROGRAM", SubjectID: "program-1",
		AudienceType: "INTERNAL", Status: evidence.RequestReady, CreatedBy: "requester", Deadline: now.Add(time.Hour),
	}
	repo := evidence.NewMemoryRepositoryWithRecipientCandidates(nil, []evidence.Request{requestValue}, []evidence.RecipientCandidate{
		{PrincipalID: "requester", TenantID: "bank", LegalEntityIDs: []string{"entity-2"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program-1": true}},
		{PrincipalID: "candidate", TenantID: "bank", LegalEntityIDs: []string{"entity-1"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program-1": true}},
	})
	api := &API{deps: Dependencies{Evidence: evidence.NewServiceWithClock(repo, nil, func() time.Time { return now })}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/request-1/recipient-candidates", nil)
	request.SetPathValue("id", requestValue.ID)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "entity-1", PrincipalID: "requester", ExpiresAt: now.Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.listEvidenceRecipientCandidates(response, request)

	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), `"error":"not_found"`) {
		t.Fatalf("wrong-entity requester response = %d %s", response.Code, response.Body.String())
	}
}

func TestEvidenceRecipientCandidatesReturnSameNotFoundOutsideRequesterScope(t *testing.T) {
	now := time.Date(2026, 8, 26, 14, 0, 0, 0, time.UTC)
	repo := evidence.NewMemoryRepositoryWithRecipientCandidates(nil, []evidence.Request{{
		ID: "request-1", TenantID: "bank", LegalEntityID: "entity-1", SubjectType: "PROGRAM", SubjectID: "program-1",
		Status: evidence.RequestReady, CreatedBy: "requester", Deadline: now.Add(time.Hour),
	}}, nil)
	api := &API{deps: Dependencies{Evidence: evidence.NewServiceWithClock(repo, nil, func() time.Time { return now })}}

	responses := make([]string, 0, 3)
	for _, test := range []struct {
		name, requestID, entityID, actorID string
	}{
		{name: "unrelated request", requestID: "missing", entityID: "entity-1", actorID: "requester"},
		{name: "cross entity", requestID: "request-1", entityID: "entity-2", actorID: "requester"},
		{name: "non requester", requestID: "request-1", entityID: "entity-1", actorID: "other"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/"+test.requestID+"/recipient-candidates", nil)
			request.SetPathValue("id", test.requestID)
			request = request.WithContext(identity.WithActor(context.Background(), identity.Actor{TenantID: "bank", LegalEntityID: test.entityID, PrincipalID: test.actorID, ExpiresAt: now.Add(time.Hour)}))
			response := httptest.NewRecorder()
			api.listEvidenceRecipientCandidates(response, request)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d: %s", response.Code, response.Body.String())
			}
			responses = append(responses, response.Body.String())
		})
	}
	for index := 1; index < len(responses); index++ {
		if responses[index] != responses[0] {
			t.Fatalf("not-found responses enumerate scope: %#v", responses)
		}
	}
}

func TestEvidenceRecipientCandidatesRejectInvalidLimit(t *testing.T) {
	api := &API{deps: Dependencies{Evidence: evidence.NewService(evidence.NewMemoryRepository(nil, nil), nil)}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/request-1/recipient-candidates?limit=51", nil)
	request.SetPathValue("id", "request-1")
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "entity-1", PrincipalID: "requester", ExpiresAt: time.Now().Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.listEvidenceRecipientCandidates(response, request)

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"error":"invalid_limit"`) {
		t.Fatalf("invalid limit response = %d %s", response.Code, response.Body.String())
	}
}

func TestEvidenceRecipientCandidatesRejectOversizedSearch(t *testing.T) {
	api := &API{deps: Dependencies{Evidence: evidence.NewService(evidence.NewMemoryRepository(nil, nil), nil)}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/request-1/recipient-candidates?q="+strings.Repeat("a", 101), nil)
	request.SetPathValue("id", "request-1")
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "entity-1", PrincipalID: "requester", ExpiresAt: time.Now().Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.listEvidenceRecipientCandidates(response, request)

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"error":"invalid_search"`) {
		t.Fatalf("oversized search response = %d %s", response.Code, response.Body.String())
	}
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}
