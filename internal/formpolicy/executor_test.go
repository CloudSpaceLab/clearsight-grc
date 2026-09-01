package formpolicy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type executionResponseReaderStub struct {
	responses map[string]evidence.CompletedResponseSummary
	err       error
}

func (stub executionResponseReaderStub) GetCompletedResponseForExecution(_ context.Context, tenantID, revisionID string) (evidence.CompletedResponseSummary, error) {
	if stub.err != nil {
		return evidence.CompletedResponseSummary{}, stub.err
	}
	value, ok := stub.responses[revisionID]
	if !ok || value.TenantID != tenantID {
		return evidence.CompletedResponseSummary{}, evidence.ErrNotFound
	}
	return value, nil
}

type executionAuthorityStub struct {
	route              ExecutionRoute
	err                error
	exceptionPrincipal string
}

func (stub executionAuthorityStub) ResolvePolicyExecution(_ context.Context, _ Policy, response evidence.CompletedResponseSummary) (ExecutionRoute, error) {
	value := stub.route
	value.TenantID, value.LegalEntityID = response.TenantID, response.LegalEntityID
	value.CanonicalSubjectType, value.CanonicalSubjectID = response.SubjectType, response.SubjectID
	return value, stub.err
}

func (stub executionAuthorityStub) ResolvePolicyExecutionException(_ context.Context, _ Policy, response evidence.CompletedResponseSummary) (ExecutionExceptionRoute, error) {
	if stub.exceptionPrincipal == "" {
		return ExecutionExceptionRoute{}, ErrActivationAuthority
	}
	return ExecutionExceptionRoute{TenantID: response.TenantID, LegalEntityID: response.LegalEntityID, PrincipalID: stub.exceptionPrincipal}, nil
}

func TestExecutorReusesOpenAdverseEpisodeMatter(t *testing.T) {
	executor := newExecutorFixture(t, RolloutEnforce)
	repo := executor.store.(*MemoryRepository)
	policy, _ := repo.GetPolicy(t.Context(), "bank", "entity", "policy-a")
	policy.BlastRadius.PerRun = 1
	policy.RecordVersion++
	if _, err := repo.UpdatePolicy(t.Context(), policy, policy.RecordVersion-1); err != nil {
		t.Fatal(err)
	}
	first, err := executor.Handle(t.Context(), scoredEvent("event-1", "response-1"))
	if err != nil || len(first) != 1 || first[0].MatterID == "" || first[0].State != ExecutionApplied || !first[0].CreatedMatter {
		t.Fatalf("first = %#v, err = %v", first, err)
	}
	if len(repo.contracts) != 1 {
		t.Fatalf("created Matter outcome contracts = %#v", repo.contracts)
	}
	for _, contract := range repo.contracts {
		if contract.MatterID != first[0].MatterID || contract.ExpectedOutcome == "" || contract.AuthorityPrincipalID != "reviewer-current" || contract.Status != "ACTIVE" {
			t.Fatalf("outcome contract = %#v", contract)
		}
	}
	second, err := executor.Handle(t.Context(), scoredEvent("event-2", "response-2"))
	if err != nil || len(second) != 1 || second[0].MatterID != first[0].MatterID || second[0].State != ExecutionReused || second[0].CreatedMatter {
		t.Fatalf("second = %#v, err = %v", second, err)
	}
	replayed, err := executor.Handle(t.Context(), scoredEvent("event-replayed", "response-1"))
	if err != nil || len(replayed) != 1 || replayed[0].ID != first[0].ID || replayed[0].MatterID != first[0].MatterID {
		t.Fatalf("replay = %#v, err = %v", replayed, err)
	}
}

func TestMemoryBlastRadiusIsScopedToTenantAndLegalEntity(t *testing.T) {
	executor := newExecutorFixture(t, RolloutEnforce)
	repo := executor.store.(*MemoryRepository)
	policy, _ := repo.GetPolicy(t.Context(), "bank", "entity", "policy-a")
	policy.BlastRadius.PerDay = 1
	policy.RecordVersion++
	if _, err := repo.UpdatePolicy(t.Context(), policy, policy.RecordVersion-1); err != nil {
		t.Fatal(err)
	}
	repo.executions["other"] = ExecutionReceipt{ID: "other", TenantID: "other-bank", LegalEntityID: "other-entity", PolicyID: policy.ID, PolicyVersion: policy.Version, CreatedMatter: true, CreatedAt: executor.currentTime()}
	receipts, err := executor.Handle(t.Context(), scoredEvent("event-1", "response-1"))
	if err != nil || len(receipts) != 1 || receipts[0].State != ExecutionApplied {
		t.Fatalf("receipts=%#v err=%v", receipts, err)
	}
}

func TestMemoryExecutionClosesPriorEpisodeAtMatterClosureTime(t *testing.T) {
	executor := newExecutorFixture(t, RolloutEnforce)
	repo := executor.store.(*MemoryRepository)
	first, err := executor.Handle(t.Context(), scoredEvent("event-1", "response-1"))
	if err != nil || len(first) != 1 {
		t.Fatal(err)
	}
	closedAt := executor.currentTime().Add(time.Hour)
	repo.mu.Lock()
	matter := repo.matters[first[0].MatterID]
	matter.Status, matter.ClosedAt = "CLOSED", &closedAt
	repo.matters[first[0].MatterID] = matter
	repo.mu.Unlock()
	executor.now = func() time.Time { return closedAt.Add(time.Hour) }
	second, err := executor.Handle(t.Context(), scoredEvent("event-2", "response-2"))
	if err != nil || len(second) != 1 || second[0].MatterID == first[0].MatterID {
		t.Fatalf("second=%#v err=%v", second, err)
	}
	for _, episode := range repo.episodeHistory {
		if episode.MatterID == first[0].MatterID && (episode.ClosedAt == nil || !episode.ClosedAt.Equal(closedAt)) {
			t.Fatalf("prior episode=%#v", episode)
		}
	}
}

func TestExecutorStartsNewEpisodeAfterVerifiedMatterClosure(t *testing.T) {
	executor := newExecutorFixture(t, RolloutEnforce)
	repo := executor.store.(*MemoryRepository)
	first, err := executor.Handle(t.Context(), scoredEvent("event-1", "response-1"))
	if err != nil || len(first) != 1 || !first[0].CreatedMatter {
		t.Fatalf("first = %#v, err = %v", first, err)
	}
	repo.mu.Lock()
	matter := repo.matters[first[0].MatterID]
	matter.Status = "CLOSED"
	closedAt := executor.currentTime().Add(time.Hour)
	matter.ClosedAt = &closedAt
	repo.matters[first[0].MatterID] = matter
	repo.mu.Unlock()
	if processed, err := repo.MaintainOutcomeChecks(t.Context(), "worker-a", closedAt.Add(30*time.Minute), time.Minute, 10); err != nil || processed != 1 {
		t.Fatalf("outcome checks processed=%d err=%v", processed, err)
	}
	for _, episode := range repo.episodes {
		if episode.State == EpisodeClosed && (episode.ClosedAt == nil || !episode.ClosedAt.Equal(closedAt)) {
			t.Fatalf("episode closure timestamp=%v want Matter closed_at %v", episode.ClosedAt, closedAt)
		}
	}
	second, err := executor.Handle(t.Context(), scoredEvent("event-2", "response-2"))
	if err != nil || len(second) != 1 || !second[0].CreatedMatter || second[0].MatterID == first[0].MatterID {
		t.Fatalf("second episode = %#v, err = %v", second, err)
	}
}

func TestExecutorRoutesRollbackAppliedMatterForReviewWithoutClosingIt(t *testing.T) {
	executor := newExecutorFixture(t, RolloutEnforce)
	repo := executor.store.(*MemoryRepository)
	applied, err := executor.Handle(t.Context(), scoredEvent("event-1", "response-1"))
	if err != nil || len(applied) != 1 || !applied[0].CreatedMatter {
		t.Fatalf("applied=%#v err=%v", applied, err)
	}
	statusBefore := repo.matters[applied[0].MatterID].Status
	rollback := repo.policies[policyKey("bank", "entity", "policy-a")]
	rollback.ID, rollback.Version, rollback.Code = "rollback-a", 2, "poor-vendor-response"
	rollback.RollbackOfPolicyID, rollback.SupersedesPolicyID = "policy-a", "policy-a"
	rollback.ActivatedAt = ptrTime(executor.currentTime())
	rollback.CreatedAt, rollback.UpdatedAt = executor.currentTime(), executor.currentTime()
	if _, err = repo.CreatePolicy(t.Context(), rollback); err != nil {
		t.Fatal(err)
	}
	candidates, err := repo.ListPendingCompensations(t.Context(), executor.currentTime(), 10)
	if err != nil || len(candidates) != 1 || candidates[0].OriginalExecution.ID != applied[0].ID {
		t.Fatalf("candidates=%#v err=%v", candidates, err)
	}
	receipt, err := executor.HandleCompensation(t.Context(), candidates[0])
	if err != nil || receipt.State != CompensationReviewRequired || receipt.MatterID != applied[0].MatterID || receipt.ReviewMatterID == "" || receipt.ReviewerPrincipalID != "reviewer-current" {
		t.Fatalf("receipt=%#v err=%v", receipt, err)
	}
	matter := repo.matters[applied[0].MatterID]
	if matter.Status != statusBefore || matter.ClosedAt != nil || matter.Version != 3 {
		t.Fatalf("compensated matter=%#v", matter)
	}
	if review := repo.matters[receipt.ReviewMatterID]; review.SourceID != applied[0].ID || review.Status != "TRIAGE" {
		t.Fatalf("review Matter=%#v", review)
	}
	if len(repo.operationalActions) != 1 {
		t.Fatalf("review actions=%#v", repo.operationalActions)
	}
	for _, action := range repo.operationalActions {
		if action.MatterID != receipt.ReviewMatterID || action.OwnerPrincipalID != "reviewer-current" || action.RequiredResponsibility != "REVIEWER" {
			t.Fatalf("review action=%#v", action)
		}
	}
	replayed, err := executor.HandleCompensation(t.Context(), candidates[0])
	if err != nil || replayed.ID != receipt.ID || len(repo.compensations) != 1 {
		t.Fatalf("replayed=%#v compensations=%#v err=%v", replayed, repo.compensations, err)
	}
}

func TestExecutorCompensationKeepsClosedMatterClosedAndCreatesReviewerWork(t *testing.T) {
	executor := newExecutorFixture(t, RolloutEnforce)
	repo := executor.store.(*MemoryRepository)
	applied, err := executor.Handle(t.Context(), scoredEvent("event-1", "response-1"))
	if err != nil || len(applied) != 1 {
		t.Fatal(err)
	}
	closedAt := executor.currentTime()
	matter := repo.matters[applied[0].MatterID]
	matter.Status, matter.ClosedAt = "CLOSED", &closedAt
	repo.matters[matter.ID] = matter
	rollback := repo.policies[policyKey("bank", "entity", "policy-a")]
	rollback.ID, rollback.Version, rollback.RollbackOfPolicyID = "rollback-a", 2, "policy-a"
	rollback.ActivatedAt = ptrTime(executor.currentTime())
	rollback.CreatedAt, rollback.UpdatedAt = executor.currentTime(), executor.currentTime()
	if _, err = repo.CreatePolicy(t.Context(), rollback); err != nil {
		t.Fatal(err)
	}
	candidates, _ := repo.ListPendingCompensations(t.Context(), executor.currentTime(), 10)
	receipt, err := executor.HandleCompensation(t.Context(), candidates[0])
	if err != nil || repo.matters[applied[0].MatterID].Status != "CLOSED" || repo.matters[applied[0].MatterID].ClosedAt == nil || repo.matters[receipt.ReviewMatterID].Status != "TRIAGE" || len(repo.operationalActions) != 1 {
		t.Fatalf("receipt=%#v original=%#v review=%#v actions=%#v err=%v", receipt, repo.matters[applied[0].MatterID], repo.matters[receipt.ReviewMatterID], repo.operationalActions, err)
	}
}

func TestExecutorCompensationFailsClosedWhenCurrentAuthorityIsUnavailable(t *testing.T) {
	executor := newExecutorFixture(t, RolloutEnforce)
	repo := executor.store.(*MemoryRepository)
	applied, err := executor.Handle(t.Context(), scoredEvent("event-1", "response-1"))
	if err != nil || len(applied) != 1 {
		t.Fatal(err)
	}
	rollback := repo.policies[policyKey("bank", "entity", "policy-a")]
	rollback.ID, rollback.Version, rollback.RollbackOfPolicyID = "rollback-a", 2, "policy-a"
	rollback.ActivatedAt = ptrTime(executor.currentTime())
	rollback.CreatedAt, rollback.UpdatedAt = executor.currentTime(), executor.currentTime()
	if _, err = repo.CreatePolicy(t.Context(), rollback); err != nil {
		t.Fatal(err)
	}
	candidates, _ := repo.ListPendingCompensations(t.Context(), executor.currentTime(), 10)
	executor.authority = executionAuthorityStub{err: ErrAuthorityUnavailable}
	if receipt, err := executor.HandleCompensation(t.Context(), candidates[0]); !errors.Is(err, ErrAuthorityUnavailable) || receipt.ID != "" || len(repo.compensations) != 0 {
		t.Fatalf("receipt=%#v compensations=%#v err=%v", receipt, repo.compensations, err)
	}
}

func TestExecutorLinksProgramSubjectAndUsesItsCurrentOwner(t *testing.T) {
	executor := newExecutorFixture(t, RolloutEnforce)
	repo := executor.store.(*MemoryRepository)
	policy, _ := repo.GetPolicy(t.Context(), "bank", "entity", "policy-a")
	policy.Eligibility.SubjectTypes = []string{"PROGRAM"}
	policy.RecordVersion++
	_, _ = repo.UpdatePolicy(t.Context(), policy, policy.RecordVersion-1)
	response := completedResponse("response-program", "program-a", 82)
	response.SubjectType = "PROGRAM"
	executor.responses = executionResponseReaderStub{responses: map[string]evidence.CompletedResponseSummary{response.ID: response}}
	executor.authority = executionAuthorityStub{route: ExecutionRoute{ServicePrincipalID: "automation-service", OwnerPrincipalID: "program-owner", ReviewerPrincipalID: "reviewer-current", ProgramID: "program-a"}}
	receipts, err := executor.Handle(t.Context(), scoredEvent("event-program", response.ID))
	if err != nil || len(receipts) != 1 || len(repo.links) != 1 {
		t.Fatalf("receipts=%#v links=%#v err=%v", receipts, repo.links, err)
	}
	for _, link := range repo.links {
		if link.ProgramID != "program-a" || link.MatterID != receipts[0].MatterID || repo.matters[receipts[0].MatterID].OwnerPrincipalID != "program-owner" {
			t.Fatalf("link=%#v matter=%#v", link, repo.matters[receipts[0].MatterID])
		}
	}
}

func TestExecutorRecordsNonMatchShadowAndRunBlastSuppression(t *testing.T) {
	t.Run("not matched", func(t *testing.T) {
		executor := newExecutorFixture(t, RolloutEnforce)
		executor.responses = executionResponseReaderStub{responses: map[string]evidence.CompletedResponseSummary{"response-low": completedResponse("response-low", "subject-b", 20)}}
		receipts, err := executor.Handle(t.Context(), scoredEvent("event-low", "response-low"))
		if err != nil || len(receipts) != 1 || receipts[0].State != ExecutionNotMatched || receipts[0].MatterID != "" {
			t.Fatalf("not matched = %#v, err = %v", receipts, err)
		}
	})
	t.Run("shadow", func(t *testing.T) {
		executor := newExecutorFixture(t, RolloutShadow)
		receipts, err := executor.Handle(t.Context(), scoredEvent("event-shadow", "response-1"))
		if err != nil || len(receipts) != 1 || receipts[0].State != ExecutionShadow || receipts[0].MatterID != "" {
			t.Fatalf("shadow = %#v, err = %v", receipts, err)
		}
	})
	t.Run("per run", func(t *testing.T) {
		executor := newExecutorFixture(t, RolloutEnforce)
		repo := executor.store.(*MemoryRepository)
		policy, _ := repo.GetPolicy(t.Context(), "bank", "entity", "policy-a")
		policy.BlastRadius.PerRun = 1
		policy.RecordVersion++
		_, _ = repo.UpdatePolicy(t.Context(), policy, policy.RecordVersion-1)
		receipts, err := executor.HandleBatch(t.Context(), []ScoredResponseEvent{scoredEvent("event-1", "response-1"), scoredEvent("event-3", "response-3")})
		if err != nil || len(receipts) != 2 || receipts[0].State != ExecutionApplied || receipts[1].State != ExecutionBlastSuppressed {
			t.Fatalf("run blast radius = %#v, err = %v", receipts, err)
		}
	})
	t.Run("per run across event deliveries", func(t *testing.T) {
		executor := newExecutorFixture(t, RolloutEnforce)
		repo := executor.store.(*MemoryRepository)
		policy, _ := repo.GetPolicy(t.Context(), "bank", "entity", "policy-a")
		policy.BlastRadius.PerRun = 1
		policy.RecordVersion++
		_, _ = repo.UpdatePolicy(t.Context(), policy, policy.RecordVersion-1)
		first, firstErr := executor.Handle(t.Context(), scoredEvent("event-1", "response-1"))
		second, secondErr := executor.Handle(t.Context(), scoredEvent("event-3", "response-3"))
		if firstErr != nil || secondErr != nil || first[0].State != ExecutionApplied || second[0].State != ExecutionBlastSuppressed || second[0].ReasonCode != "PER_RUN_LIMIT" {
			t.Fatalf("first=%#v second=%#v errors=%v/%v", first, second, firstErr, secondErr)
		}
	})
}

func TestExecutorSkipsSuspendedExpiredAndNeverActivatedPolicies(t *testing.T) {
	for name, mutate := range map[string]func(*Policy, time.Time){
		"suspended":       func(policy *Policy, _ time.Time) { policy.Status = PolicySuspended },
		"expired":         func(policy *Policy, now time.Time) { until := now.Add(-2 * time.Hour); policy.EffectiveUntil = &until },
		"never activated": func(policy *Policy, _ time.Time) { policy.ActivatedAt = nil },
		"activated after completion": func(policy *Policy, now time.Time) {
			activated := now.Add(-30 * time.Minute)
			policy.ActivatedAt = &activated
		},
	} {
		t.Run(name, func(t *testing.T) {
			executor := newExecutorFixture(t, RolloutEnforce)
			repo := executor.store.(*MemoryRepository)
			policy, _ := repo.GetPolicy(t.Context(), "bank", "entity", "policy-a")
			mutate(&policy, executor.currentTime())
			policy.RecordVersion++
			if _, err := repo.UpdatePolicy(t.Context(), policy, policy.RecordVersion-1); err != nil {
				t.Fatal(err)
			}
			receipts, err := executor.Handle(t.Context(), scoredEvent("event-1", "response-1"))
			if err != nil || len(receipts) != 0 {
				t.Fatalf("receipts=%#v err=%v", receipts, err)
			}
		})
	}
}

func TestExecutorFailsClosedBeforeWritingWhenCurrentRouteCannotResolve(t *testing.T) {
	executor := newExecutorFixture(t, RolloutEnforce)
	executor.authority = executionAuthorityStub{err: ErrAuthorityUnavailable}
	if receipts, err := executor.Handle(t.Context(), scoredEvent("event-1", "response-1")); !errors.Is(err, ErrAuthorityUnavailable) || len(receipts) != 1 || receipts[0].State != ExecutionFailed {
		t.Fatalf("route failure receipts=%#v err=%v", receipts, err)
	}
	repo := executor.store.(*MemoryRepository)
	if len(repo.executions) != 0 || len(repo.executionFailures) != 1 || len(repo.episodes) != 0 || len(repo.matters) != 0 {
		t.Fatalf("route failure state: executions=%#v failures=%#v episodes=%#v matters=%#v", repo.executions, repo.executionFailures, repo.episodes, repo.matters)
	}
}

func TestExecutorRecordsDefinitiveAuthorityFailureWithoutCreatingMatter(t *testing.T) {
	executor := newExecutorFixture(t, RolloutEnforce)
	executor.authority = executionAuthorityStub{err: ErrActivationAuthority, exceptionPrincipal: "escalation-current"}
	receipts, err := executor.Handle(t.Context(), scoredEvent("event-1", "response-1"))
	if !errors.Is(err, ErrActivationAuthority) || len(receipts) != 1 || receipts[0].State != ExecutionFailed || receipts[0].ReasonCode != "AUTHORITY_ROUTE_INVALID" || receipts[0].MatterID != "" {
		t.Fatalf("receipts=%#v err=%v", receipts, err)
	}
	repo := executor.store.(*MemoryRepository)
	if len(repo.executions) != 0 || len(repo.executionFailures) != 1 || len(repo.episodes) != 0 || len(repo.operationalActions) != 1 {
		t.Fatalf("authority failure state executions=%#v failures=%#v episodes=%#v actions=%#v", repo.executions, repo.executionFailures, repo.episodes, repo.operationalActions)
	}
}

func TestExecutorRunsAfterAnAuthorityFailureIsRepaired(t *testing.T) {
	executor := newExecutorFixture(t, RolloutEnforce)
	executor.authority = executionAuthorityStub{err: ErrActivationAuthority, exceptionPrincipal: "escalation-current"}
	failed, err := executor.Handle(t.Context(), scoredEvent("event-1", "response-1"))
	if !errors.Is(err, ErrActivationAuthority) || len(failed) != 1 || failed[0].State != ExecutionFailed {
		t.Fatalf("failed=%#v err=%v", failed, err)
	}
	executor.authority = executionAuthorityStub{route: ExecutionRoute{ServicePrincipalID: "automation-service", OwnerPrincipalID: "owner-current", ReviewerPrincipalID: "reviewer-current"}, exceptionPrincipal: "escalation-current"}
	applied, err := executor.Handle(t.Context(), scoredEvent("event-1", "response-1"))
	if err != nil || len(applied) != 1 || applied[0].State != ExecutionApplied || !applied[0].CreatedMatter {
		t.Fatalf("applied=%#v err=%v", applied, err)
	}
	repo := executor.store.(*MemoryRepository)
	if len(repo.executionFailures) != 1 || len(repo.executions) != 1 {
		t.Fatalf("failures=%#v executions=%#v", repo.executionFailures, repo.executions)
	}
	if len(repo.operationalActions) != 1 {
		t.Fatalf("operational actions=%#v", repo.operationalActions)
	}
	for _, action := range repo.operationalActions {
		if action.Status != continuity.ActionImplemented || action.ImplementedAt == nil || !strings.Contains(action.Description, applied[0].ID) {
			t.Fatalf("recovery action=%#v execution=%#v", action, applied[0])
		}
	}
	for _, matter := range repo.matters {
		if matter.SourceType != "FORM_RESPONSE_POLICY_EXECUTION" {
			continue
		}
		var facts map[string]any
		if json.Unmarshal(matter.KnownFacts, &facts) != nil || facts["route_recovery_execution_id"] != applied[0].ID || matter.Status != continuity.MatterInitialReview {
			t.Fatalf("recovery matter=%#v facts=%#v", matter, facts)
		}
	}
}

func TestExecutorSelectsEffectivePoliciesByExactFormRevision(t *testing.T) {
	executor := newExecutorFixture(t, RolloutEnforce)
	repo := executor.store.(*MemoryRepository)
	for index := 0; index < 205; index++ {
		policy := Policy{
			ID: fmt.Sprintf("inactive-%03d", index), TenantID: "bank", LegalEntityID: "entity", Code: fmt.Sprintf("aaa-%03d", index),
			Name: "Unrelated policy", Purpose: "Prove execution does not page through unrelated definitions.", ActionClass: ActionClassCreateMatter,
			AutomationPolicyID: "automation-a", AutomationPolicyVersion: 2,
			Eligibility: Eligibility{FormTemplateID: "other-form", FormTemplateVersion: 1, SubjectTypes: []string{"VENDOR_RELATIONSHIP"}, CurrentOnly: true, MinimumCoverage: 0.8},
			Action:      MatterAction{Type: "VENDOR_DEFICIENCY", Priority: 4, TitleTemplate: "Review {{form_title}}", SummaryTemplate: "Review the response.", RequestedHandling: "Review the response."},
			BlastRadius: BlastRadius{PerRun: 10, PerDay: 25}, Outcome: OutcomeContract{ExpectedOutcome: "Resolved", CheckAfterMinutes: 60, FailureResponse: "REVIEW"},
			Rollout: RolloutShadow, Status: PolicyRetired, MakerID: "maker", CheckerID: "checker", Checksum: fmt.Sprintf("checksum-%03d", index), Version: 1, RecordVersion: 1,
			CreatedAt: executor.currentTime().Add(-time.Hour), UpdatedAt: executor.currentTime().Add(-time.Minute),
		}
		if _, err := repo.CreatePolicy(t.Context(), policy); err != nil {
			t.Fatal(err)
		}
	}
	receipts, err := executor.Handle(t.Context(), scoredEvent("event-1", "response-1"))
	if err != nil || len(receipts) != 1 || receipts[0].PolicyID != "policy-a" || receipts[0].State != ExecutionApplied {
		t.Fatalf("exact active policies = %#v, err = %v", receipts, err)
	}
}

func TestExecutorFailsClosedWhenEffectivePolicyPopulationExceedsBound(t *testing.T) {
	executor := newExecutorFixture(t, RolloutEnforce)
	repo := executor.store.(*MemoryRepository)
	base, _ := repo.GetPolicy(t.Context(), "bank", "entity", "policy-a")
	for index := 0; index < executionPolicyLimit; index++ {
		policy := base
		policy.ID = fmt.Sprintf("active-%03d", index)
		policy.Code = fmt.Sprintf("active-%03d", index)
		if _, err := repo.CreatePolicy(t.Context(), policy); err != nil {
			t.Fatal(err)
		}
	}
	if receipts, err := executor.Handle(t.Context(), scoredEvent("event-1", "response-1")); !errors.Is(err, ErrExecutionPolicyLimit) || len(receipts) != 0 {
		t.Fatalf("receipts=%#v err=%v", receipts, err)
	}
}

func newExecutorFixture(t *testing.T, rollout RolloutMode) *Executor {
	t.Helper()
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	threshold := 70.0
	policy := Policy{
		ID: "policy-a", TenantID: "bank", LegalEntityID: "entity", Code: "poor-vendor-response", Name: "Poor vendor response", Purpose: "Open one issue for the current adverse episode.",
		ActionClass: ActionClassCreateMatter, AutomationPolicyID: "automation-a", AutomationPolicyVersion: 2,
		Eligibility: Eligibility{FormTemplateID: "form-a", FormTemplateVersion: 3, SubjectTypes: []string{"VENDOR_RELATIONSHIP"}, CurrentOnly: true, MinimumCoverage: 0.8, AdverseAtLeast: &threshold},
		Action:      MatterAction{Type: "VENDOR_DEFICIENCY", Priority: 4, TitleTemplate: "Review {{form_title}}", SummaryTemplate: "The latest {{subject_type}} response needs review.", RequestedHandling: "Review the response and record corrective evidence."},
		BlastRadius: BlastRadius{PerRun: 10, PerDay: 25}, Outcome: OutcomeContract{ExpectedOutcome: "The adverse response is resolved with current evidence.", CheckAfterMinutes: 60, FailureResponse: "REVIEW"},
		Rollout: rollout, Status: PolicyActive, MakerID: "maker", CheckerID: "checker", Checksum: "policy-checksum", ActivatedAt: ptrTime(now.Add(-2 * time.Hour)), Version: 1, RecordVersion: 4, CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-time.Minute),
	}
	if _, err := repo.CreatePolicy(t.Context(), policy); err != nil {
		t.Fatal(err)
	}
	reader := executionResponseReaderStub{responses: map[string]evidence.CompletedResponseSummary{
		"response-1": completedResponse("response-1", "subject-a", 82),
		"response-2": completedResponse("response-2", "subject-a", 88),
		"response-3": completedResponse("response-3", "subject-c", 91),
	}}
	executor := NewExecutor(repo, reader, executionAuthorityStub{route: ExecutionRoute{ServicePrincipalID: "automation-service", OwnerPrincipalID: "owner-current", ReviewerPrincipalID: "reviewer-current"}, exceptionPrincipal: "escalation-current"})
	executor.now = func() time.Time { return now }
	next := 0
	executor.newID = func() (string, error) { next++; return fmt.Sprintf("generated-%d", next), nil }
	return executor
}

func ptrTime(value time.Time) *time.Time { return &value }

func completedResponse(id, subjectID string, adverse float64) evidence.CompletedResponseSummary {
	raw := 100 - adverse
	return evidence.CompletedResponseSummary{
		ID: id, TenantID: "bank", LegalEntityID: "entity", DistributionID: "distribution-a", FormTemplateID: "form-a", FormTemplateVersion: 3,
		Title: "Vendor assurance review", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: subjectID, Revision: 1, Current: true, State: evidence.ResponseRevisionFinal,
		Score: &evidence.ResponseScoreResult{Mode: formcontract.ScoringRisk, RawScore: &raw, AdverseScore: &adverse, Band: formcontract.ConcernHigh, Coverage: 1, Final: true, State: evidence.ResponseScoreFinal}, CompletedAt: time.Date(2026, 9, 1, 11, 0, 0, 0, time.UTC),
	}
}

func scoredEvent(eventID, responseID string) ScoredResponseEvent {
	return ScoredResponseEvent{ID: eventID, TenantID: "bank", ResponseRevisionID: responseID, OccurredAt: time.Date(2026, 9, 1, 11, 0, 0, 0, time.UTC)}
}
