package formpolicy

import (
	"encoding/json"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
)

// Contextual automation shares the typed policy revision's generated identity.
// Callers cannot supply the typed policy ID, so existing selected guardrails
// cannot be converted into contextually managed records.
func (policy Policy) ManagesAutomation() bool {
	return policy.ID != "" && policy.ID == policy.AutomationPolicyID && policy.Version == policy.AutomationPolicyVersion
}

func managedAutomation(policy Policy) autonomy.AutomationPolicy {
	eligibility, _ := json.Marshal(policy.Eligibility)
	blast, _ := json.Marshal(policy.BlastRadius)
	outcome, _ := json.Marshal(policy.Outcome)
	return autonomy.AutomationPolicy{
		ID: policy.AutomationPolicyID, TenantID: policy.TenantID, Code: "forms:" + policy.LegalEntityID + ":" + policy.Code, Name: policy.Name,
		ActionClass: ActionClassCreateMatter, Eligibility: eligibility, BlastRadiusLimit: blast, VerificationContract: outcome,
		Status: autonomy.AutomationPolicyState(policy.Status), RolloutMode: string(policy.Rollout), Checksum: policy.Checksum,
		MakerID: policy.MakerID, CheckerID: policy.CheckerID, EffectiveFrom: policy.EffectiveFrom, EffectiveUntil: policy.EffectiveUntil,
		SubmittedAt: policy.SubmittedAt, ApprovedAt: policy.ApprovedAt, ActivatedAt: policy.ActivatedAt, SuspendedAt: policy.SuspendedAt, RetiredAt: policy.RetiredAt,
		CreatedAt: policy.CreatedAt, UpdatedAt: policy.UpdatedAt, Version: policy.AutomationPolicyVersion, RecordVersion: policy.RecordVersion,
	}
}
