package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formpolicy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type resultPolicyRepo struct {
	*formpolicy.MemoryRepository
	policy formpolicy.Policy
	result formpolicy.ExecutionResultSource
	calls  int
}

func (s *resultPolicyRepo) GetPolicy(_ context.Context, tenant, entity, id string) (formpolicy.Policy, error) {
	if tenant != s.policy.TenantID || entity != s.policy.LegalEntityID || id != s.policy.ID {
		return formpolicy.Policy{}, formpolicy.ErrNotFound
	}
	return s.policy, nil
}
func (s *resultPolicyRepo) GetExecutionResult(_ context.Context, tenant, entity, policy, id string) (formpolicy.ExecutionResultSource, error) {
	s.calls++
	if tenant != s.policy.TenantID || entity != s.policy.LegalEntityID || policy != s.policy.ID || id != s.result.ID {
		return formpolicy.ExecutionResultSource{}, formpolicy.ErrNotFound
	}
	return s.result, nil
}

type resultMatterRepo struct {
	*continuity.MemoryRepository
	matter continuity.MatterAggregate
	calls  int
	err    error
}

func (s *resultMatterRepo) GetMatter(_ context.Context, tenant, id string) (continuity.MatterAggregate, error) {
	s.calls++
	if s.err != nil {
		return continuity.MatterAggregate{}, s.err
	}
	if tenant != s.matter.Matter.TenantID || id != s.matter.Matter.ID {
		return continuity.MatterAggregate{}, continuity.ErrNotFound
	}
	return s.matter, nil
}

type resultResponseStore struct {
	*evidence.MemoryDistributionStore
	allow bool
	calls int
}

func (s *resultResponseStore) GetCompletedResponse(_ context.Context, tenant, entity, principal, id string) (evidence.CompletedResponseSummary, evidence.ResponseRevision, error) {
	s.calls++
	if !s.allow || tenant != "bank" || entity != "entity" || principal != "reader" || id != "private-response" {
		return evidence.CompletedResponseSummary{}, evidence.ResponseRevision{}, evidence.ErrNotFound
	}
	return evidence.CompletedResponseSummary{ID: id, TenantID: tenant, LegalEntityID: entity, Title: "Permitted response"}, evidence.ResponseRevision{}, nil
}

func TestPolicyResultChecksIssueAndResponsePermissionsIndependently(t *testing.T) {
	for _, tc := range []struct {
		name, owner, actionOwner, scope string
		response, issue, readFailure    bool
	}{
		{name: "denied", scope: `{"access":"RESTRICTED","allowed_principal_ids":["other"]}`},
		{name: "response only", scope: `{"access":"RESTRICTED","allowed_principal_ids":["other"]}`, response: true},
		{name: "allowlist", scope: `{"access":"RESTRICTED","allowed_principal_ids":["reader"]}`, issue: true},
		{name: "issue owner", owner: "reader", scope: `{"access":"RESTRICTED","allowed_principal_ids":["other"]}`, issue: true},
		{name: "action owner", actionOwner: "reader", scope: `{"access":"RESTRICTED","allowed_principal_ids":["other"]}`, response: true, issue: true},
		{name: "revoked action owner", actionOwner: "replacement", scope: `{"access":"RESTRICTED","allowed_principal_ids":["other"]}`},
		{name: "matter unavailable", scope: `{}`, response: true, readFailure: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			policy := &resultPolicyRepo{MemoryRepository: formpolicy.NewMemoryRepository(), policy: formpolicy.Policy{ID: "policy", TenantID: "bank", LegalEntityID: "entity"}, result: formpolicy.ExecutionResultSource{ExecutionHistoryItem: formpolicy.ExecutionHistoryItem{ID: "execution", State: formpolicy.ExecutionApplied}, MatterID: "private-issue", ResponseRevisionID: "private-response"}}
			matters := &resultMatterRepo{MemoryRepository: continuity.NewMemoryRepository(), matter: continuity.MatterAggregate{Matter: continuity.Matter{ID: "private-issue", TenantID: "bank", LegalEntityID: "entity", Title: "Permitted issue", OwnerPrincipalID: tc.owner, Scope: json.RawMessage(tc.scope)}, Actions: []continuity.Action{{OwnerPrincipalID: tc.actionOwner}}}}
			if tc.readFailure {
				matters.err = errors.New("unavailable")
			}
			responses := &resultResponseStore{MemoryDistributionStore: evidence.NewMemoryDistributionStore(nil, nil, nil), allow: tc.response}
			api := &API{deps: Dependencies{FormPolicies: formpolicy.NewService(policy, nil, nil), Continuity: continuity.NewService(matters), FormDistributions: evidence.NewDistributionService(responses)}}
			actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "reader", ExpiresAt: time.Now().Add(time.Hour)}
			r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(identity.WithActor(context.Background(), actor))
			r.SetPathValue("id", "policy")
			r.SetPathValue("execution_id", "execution")
			w := httptest.NewRecorder()
			api.getFormPolicyExecutionResult(w, r)
			if w.Code != 200 || strings.Contains(w.Body.String(), "private-issue") != tc.issue || strings.Contains(w.Body.String(), "private-response") != tc.response {
				t.Fatalf("result=%d %s", w.Code, w.Body.String())
			}
			if policy.calls != 1 || matters.calls != 1 || responses.calls != 1 {
				t.Fatalf("unbounded reads %d/%d/%d", policy.calls, matters.calls, responses.calls)
			}
			r.SetPathValue("execution_id", "missing")
			w = httptest.NewRecorder()
			api.getFormPolicyExecutionResult(w, r)
			if w.Code != 404 || matters.calls != 1 || responses.calls != 1 {
				t.Fatalf("missing execution reached targets: %d", w.Code)
			}
		})
	}
}
