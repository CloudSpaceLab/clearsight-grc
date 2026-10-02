package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

type linkedIssueFixture struct {
	continuity *continuity.Service
	monitoring *monitoring.Service
	repo       *monitoring.MemoryRepository
	program    continuity.ProgramAggregate
	check      monitoring.MonitoringCheck
	result     monitoring.MonitoringResult
}

func newLinkedIssueFixture(t *testing.T) linkedIssueFixture {
	t.Helper()
	now := time.Date(2026, 8, 26, 16, 0, 0, 0, time.UTC)
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	program, err := continuityService.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "ACCESS", Name: "Access monitoring", Type: "COMPLIANCE",
		OwningFunction: "Information Security", OwnerPrincipalID: "program-owner", AuthorityPrincipalID: "authorizer", Scope: json.RawMessage(`{}`), EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	repo := monitoring.NewMemoryRepository()
	monitoringService := monitoring.NewService(repo, nil)
	check, err := repo.CreateCheckRevision(t.Context(), monitoring.MonitoringCheck{
		ID: "check-1", TenantID: "bank", ProgramID: program.Program.ID, Code: "ACCESS-STATUS", Name: "Access status",
		Claim: "The access control status is current.", InputKind: monitoring.InputSource, BindingID: "binding-1", BindingVersion: 1,
		Thresholds: monitoring.DefaultThresholds(), FreshnessMinutes: 60, MinimumCoverage: 1,
		OwnerPrincipalID: "program-owner", ReviewerPrincipalID: "reviewer-1", FailureAction: monitoring.FailureRecommendMatter,
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 1, CreatedAt: now, UpdatedAt: now},
	})
	if err != nil {
		t.Fatal(err)
	}
	score := 80.0
	result, err := repo.AppendResult(t.Context(), monitoring.MonitoringResult{
		ID: "result-1", TenantID: "bank", ProgramID: program.Program.ID, MonitoringCheckID: check.ID, MonitoringCheckVersion: check.Version,
		InputKind: monitoring.InputSource, InputReferenceID: "receipt-1", InputReferenceVersion: 1,
		Evaluation:  monitoring.Evaluation{Score: &score, Band: monitoring.RiskCritical, Coverage: 1, RuleResults: []monitoring.RuleResult{{FieldID: "status", Outcome: monitoring.RuleFailed, Points: 80, Critical: true, Reason: "The expected status was not returned."}}},
		EvaluatedAt: now.Add(time.Minute), EvaluatorVersion: "risk-v1", CreatedAt: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	return linkedIssueFixture{continuity: continuityService, monitoring: monitoringService, repo: repo, program: program, check: check, result: result}
}

func (f linkedIssueFixture) handler(principal string, resolution authority.Resolution) http.Handler {
	return New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", principal, "entity-a"),
		Monitoring: f.monitoring, MonitoringEpisodes: &monitoring.AdverseEpisodeCoordinator{Repository: f.repo, Continuity: f.continuity},
		Continuity: f.continuity,
		Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{authority.ResponsibilityReviewer: resolution}},
	})
}

func TestReviewerCreatesAndReusesOneIssueForPersistentAdverseMonitoringEpisode(t *testing.T) {
	fixture := newLinkedIssueFixture(t)
	handler := fixture.handler("reviewer-1", authority.Resolution{Principal: authority.Principal{ID: "reviewer-1", DisplayName: "Control assurance reviewer"}})
	path := "/api/v1/monitoring-results/" + fixture.result.ID + "/linked-issue"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{}`)))
	if response.Code != http.StatusCreated {
		t.Fatalf("create linked issue returned %d: %s", response.Code, response.Body.String())
	}
	var created struct {
		Matter  continuity.Matter         `json:"matter"`
		Created bool                      `json:"created"`
		Episode monitoring.AdverseEpisode `json:"episode"`
	}
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if !created.Created || created.Matter.Type != continuity.MatterControlGap || created.Matter.LegalEntityID != fixture.program.Program.LegalEntityID || created.Matter.OwnerPrincipalID != fixture.program.Program.OwnerPrincipalID || created.Matter.RequiredAuthority != "CONTROL_ASSURANCE" {
		t.Fatalf("linked issue is not governed by the Program: %#v", created)
	}
	if created.Episode.ID == "" || created.Episode.LastResultID != fixture.result.ID || created.Episode.MatterID != created.Matter.ID {
		t.Fatalf("linked issue episode = %#v", created.Episode)
	}
	if created.Matter.SourceType != "MONITORING_ADVERSE_EPISODE" || created.Matter.SourceID != created.Episode.ID ||
		created.Matter.TriggerKey != "monitoring-adverse-episode:"+created.Episode.ID {
		t.Fatalf("linked issue lineage = %#v", created.Matter)
	}
	var provenance struct {
		EpisodeID     string   `json:"monitoring_episode_id"`
		ResultID      string   `json:"monitoring_result_id"`
		FailedRuleIDs []string `json:"failed_rule_ids"`
	}
	if err := json.Unmarshal(created.Matter.Scope, &provenance); err != nil ||
		provenance.EpisodeID != created.Episode.ID || provenance.ResultID != fixture.result.ID ||
		len(provenance.FailedRuleIDs) != 1 || provenance.FailedRuleIDs[0] != "status" {
		t.Fatalf("linked issue failure provenance = %#v err=%v", provenance, err)
	}

	replay := httptest.NewRecorder()
	handler.ServeHTTP(replay, httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{}`)))
	if replay.Code != http.StatusOK {
		t.Fatalf("replay returned %d: %s", replay.Code, replay.Body.String())
	}
	var existing struct {
		Matter  continuity.Matter         `json:"matter"`
		Created bool                      `json:"created"`
		Episode monitoring.AdverseEpisode `json:"episode"`
	}
	if err := json.NewDecoder(replay.Body).Decode(&existing); err != nil {
		t.Fatal(err)
	}
	if existing.Created || existing.Matter.ID != created.Matter.ID || existing.Episode.ID != created.Episode.ID {
		t.Fatalf("replay did not return the existing linked issue: %#v", existing)
	}

	newerScore := 95.0
	newerAt := fixture.result.EvaluatedAt.Add(time.Minute)
	newer, err := fixture.repo.AppendResult(t.Context(), monitoring.MonitoringResult{
		ID: "result-2", TenantID: "bank", ProgramID: fixture.program.Program.ID,
		MonitoringCheckID: fixture.check.ID, MonitoringCheckVersion: fixture.check.Version,
		InputKind: monitoring.InputSource, InputReferenceID: "receipt-2", InputReferenceVersion: 2,
		Evaluation: monitoring.Evaluation{
			Score: &newerScore, Band: monitoring.RiskCritical, Coverage: 1,
			RuleResults: []monitoring.RuleResult{{FieldID: "status", Outcome: monitoring.RuleFailed, Points: 95, Critical: true, Reason: "The status is still adverse."}},
		},
		EvaluatedAt: newerAt, EvaluatorVersion: "risk-v1", CreatedAt: newerAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	persistent := httptest.NewRecorder()
	handler.ServeHTTP(persistent, httptest.NewRequest(http.MethodPost, "/api/v1/monitoring-results/"+newer.ID+"/linked-issue", bytes.NewBufferString(`{}`)))
	if persistent.Code != http.StatusOK {
		t.Fatalf("persistent adverse result returned %d: %s", persistent.Code, persistent.Body.String())
	}
	var persistentPayload struct {
		Matter  continuity.Matter         `json:"matter"`
		Created bool                      `json:"created"`
		Episode monitoring.AdverseEpisode `json:"episode"`
	}
	if err := json.NewDecoder(persistent.Body).Decode(&persistentPayload); err != nil {
		t.Fatal(err)
	}
	if persistentPayload.Created || persistentPayload.Matter.ID != created.Matter.ID ||
		persistentPayload.Episode.ID != created.Episode.ID || persistentPayload.Episode.LastResultID != newer.ID {
		t.Fatalf("persistent breach created duplicate intervention: %#v", persistentPayload)
	}
}

func TestMonitoringLinkedIssueRequiresLatestEligibleResultAndStoredReviewerLineage(t *testing.T) {
	fixture := newLinkedIssueFixture(t)
	path := "/api/v1/monitoring-results/" + fixture.result.ID + "/linked-issue"

	unrelated := fixture.handler("unrelated", authority.Resolution{
		Principal: authority.Principal{ID: "reviewer-1"}, CandidatePrincipals: []authority.Principal{{ID: "unrelated"}},
	})
	response := httptest.NewRecorder()
	unrelated.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{}`)))
	if response.Code != http.StatusForbidden {
		t.Fatalf("unrelated reviewer candidate returned %d: %s", response.Code, response.Body.String())
	}

	delegate := fixture.handler("reviewer-delegate", authority.Resolution{
		Principal: authority.Principal{ID: "reviewer-1"}, CandidatePrincipals: []authority.Principal{{ID: "reviewer-delegate"}},
		EffectiveOrigins: []authority.EffectiveOrigin{{PrincipalID: "reviewer-delegate", OriginPrincipalID: "reviewer-1"}},
	})
	delegated := httptest.NewRecorder()
	delegate.ServeHTTP(delegated, httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{}`)))
	if delegated.Code != http.StatusCreated {
		t.Fatalf("valid reviewer delegate returned %d: %s", delegated.Code, delegated.Body.String())
	}

	newerScore := 0.0
	_, err := fixture.repo.AppendResult(t.Context(), monitoring.MonitoringResult{
		ID: "result-2", TenantID: "bank", ProgramID: fixture.program.Program.ID, MonitoringCheckID: fixture.check.ID, MonitoringCheckVersion: fixture.check.Version,
		InputKind: monitoring.InputSource, InputReferenceID: "receipt-2", InputReferenceVersion: 1,
		Evaluation: monitoring.Evaluation{Score: &newerScore, Band: monitoring.RiskLow, Coverage: 1}, EvaluatedAt: fixture.result.EvaluatedAt.Add(time.Minute), EvaluatorVersion: "risk-v1", CreatedAt: fixture.result.CreatedAt.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	stale := httptest.NewRecorder()
	delegate.ServeHTTP(stale, httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{}`)))
	if stale.Code != http.StatusUnprocessableEntity {
		t.Fatalf("superseded adverse result returned %d: %s", stale.Code, stale.Body.String())
	}
}

func TestGenericProgramTriggerRouteCannotForgeAdverseMonitoringIssues(t *testing.T) {
	fixture := newLinkedIssueFixture(t)
	handler := fixture.handler("reviewer-1", authority.Resolution{Principal: authority.Principal{ID: "reviewer-1"}})
	for _, dedupeKey := range []string{"forged-adverse-1", "forged-adverse-2"} {
		body, err := json.Marshal(map[string]any{
			"type": "MONITORING_RESULT_ADVERSE", "subject_type": "MONITORING_CHECK", "subject_id": fixture.check.ID,
			"dedupe_key": dedupeKey, "source": "browser", "observed_at": fixture.result.EvaluatedAt, "payload": map[string]any{"monitoring_result_id": fixture.result.ID},
		})
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+fixture.program.Program.ID+"/triggers", bytes.NewReader(body)))
		if response.Code != http.StatusUnprocessableEntity {
			t.Fatalf("generic adverse trigger %q returned %d: %s", dedupeKey, response.Code, response.Body.String())
		}
	}
	program, err := fixture.continuity.GetProgram(continuity.WithTrustedSystemScope(t.Context()), "bank", fixture.program.Program.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Triggers) != 0 {
		t.Fatalf("generic trigger route retained forged adverse triggers: %#v", program.Triggers)
	}
	matters, err := fixture.continuity.ListMatters(continuity.WithTrustedSystemScope(t.Context()), "bank", "OPEN", 20)
	if err != nil || len(matters) != 0 {
		t.Fatalf("generic trigger route created adverse issues: matters=%#v err=%v", matters, err)
	}
}

func TestProgramOperationsExposeConfiguredIssueActionToStoredReviewerAndDelegate(t *testing.T) {
	fixture := newLinkedIssueFixture(t)
	assertOperation := func(t *testing.T, principal string, resolution authority.Resolution, wantCanAct bool) {
		t.Helper()
		handler := fixture.handler(principal, resolution)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/programs/"+fixture.program.Program.ID+"/operations?tenant_id=bank", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("operations returned %d: %s", response.Code, response.Body.String())
		}
		var payload programOperationsResponse
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		for _, operation := range payload.Operations {
			if operation.Command != "program.monitoring.issue.create" || operation.SubresourceID != fixture.check.ID {
				continue
			}
			if operation.CanAct != wantCanAct || operation.AssignedTo == nil || operation.AssignedTo.ID != "reviewer-1" || operation.Responsibility != string(authority.ResponsibilityReviewer) {
				t.Fatalf("linked issue operation = %#v, want can_act=%v assigned reviewer", operation, wantCanAct)
			}
			return
		}
		t.Fatal("active monitoring check did not expose a per-check linked issue operation")
	}
	assertOperation(t, "reviewer-1", authority.Resolution{Principal: authority.Principal{ID: "reviewer-1", DisplayName: "Control assurance reviewer"}}, true)
	assertOperation(t, "reviewer-delegate", authority.Resolution{
		Principal: authority.Principal{ID: "reviewer-1", DisplayName: "Control assurance reviewer"}, CandidatePrincipals: []authority.Principal{{ID: "reviewer-delegate"}},
		EffectiveOrigins: []authority.EffectiveOrigin{{PrincipalID: "reviewer-delegate", OriginPrincipalID: "reviewer-1"}},
	}, true)
	assertOperation(t, "unrelated", authority.Resolution{
		Principal: authority.Principal{ID: "reviewer-1", DisplayName: "Control assurance reviewer"}, CandidatePrincipals: []authority.Principal{{ID: "unrelated"}},
	}, false)
}
