package formpolicy

import (
	"encoding/json"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"testing"
)

func TestContextualGuardrailCreatesGovernedPolicyWithoutCopiedID(t *testing.T) {
	service, _, _ := newPolicyTestService(t)
	input := validPolicyInput("context-guardrail", RolloutShadow)
	input.AutomationPolicyID = ""
	input.AutomationPolicyVersion = 0
	payload, _ := json.Marshal(input)
	var raw map[string]any
	_ = json.Unmarshal(payload, &raw)
	raw["create_automation_policy"] = true
	payload, _ = json.Marshal(raw)
	_ = json.Unmarshal(payload, &input)
	draft, err := service.Create(t.Context(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker"}, input)
	if err != nil {
		t.Fatalf("contextual guardrail setup must not need a copied ID: %v", err)
	}
	if draft.AutomationPolicyID == "" || draft.AutomationPolicyVersion != 1 || draft.Status != PolicyDraft {
		t.Fatalf("draft = %#v", draft)
	}
}

func TestContextualGuardrailSharesApprovalSuspensionAndConflicts(t *testing.T) {
	service, _, _ := newPolicyTestService(t)
	repo := service.repo.(*MemoryRepository)
	canonical := autonomy.NewMemoryRepository()
	repo.automation = canonical
	maker := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker"}
	checker := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "checker"}
	input := validPolicyInput("governed-guardrail", RolloutShadow)
	input.CreateAutomationPolicy = true
	input.AutomationPolicyID = ""
	input.AutomationPolicyVersion = 0
	draft, err := service.Create(t.Context(), maker, input)
	if err != nil {
		t.Fatal(err)
	}
	check := func(policy Policy) {
		t.Helper()
		guard, err := canonical.GetAutomationPolicy(t.Context(), "bank", policy.AutomationPolicyID, policy.AutomationPolicyVersion)
		if err != nil || string(guard.Status) != string(policy.Status) || guard.RecordVersion != policy.RecordVersion || guard.Checksum != policy.Checksum {
			t.Fatalf("guardrail=%#v policy=%#v err=%v", guard, policy, err)
		}
	}
	check(draft)
	simulation, err := service.Simulate(t.Context(), maker, draft.ID, draft.RecordVersion)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := service.Submit(t.Context(), maker, draft.ID, draft.RecordVersion, simulation.ID)
	if err != nil {
		t.Fatal(err)
	}
	check(pending)
	if _, err := service.Approve(t.Context(), maker, pending.ID, pending.RecordVersion, simulation.ID); !errors.Is(err, ErrMakerChecker) {
		t.Fatal(err)
	}
	check(pending)
	approved, err := service.Approve(t.Context(), checker, pending.ID, pending.RecordVersion, simulation.ID)
	if err != nil {
		t.Fatal(err)
	}
	check(approved)
	if _, err := service.Activate(t.Context(), checker, approved.ID, approved.RecordVersion+1); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	check(approved)
	active, err := service.Activate(t.Context(), checker, approved.ID, approved.RecordVersion)
	if err != nil {
		t.Fatal(err)
	}
	check(active)
	suspended, err := service.Suspend(t.Context(), checker, active.ID, active.RecordVersion)
	if err != nil {
		t.Fatal(err)
	}
	check(suspended)
	choices, err := service.AutomationChoices(t.Context(), maker, draft.Eligibility.FormTemplateID, draft.Eligibility.FormTemplateVersion)
	if err != nil || len(choices) != 1 || choices[0].Status != "SUSPENDED" {
		t.Fatalf("choices=%#v err=%v", choices, err)
	}
	other := maker
	other.LegalEntityID = "other"
	if _, err := service.AutomationChoices(t.Context(), other, draft.Eligibility.FormTemplateID, draft.Eligibility.FormTemplateVersion); !errors.Is(err, ErrFormInactive) {
		t.Fatal("cross entity choices must fail closed", err)
	}
}
