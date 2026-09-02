package formpolicy

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
)

// AutomationPolicyAllows confirms that the exact approved generic policy
// revision grants precisely the rollout and guardrails carried by the typed
// form policy. Both activation and execution use this same fail-closed check.
func AutomationPolicyAllows(approved autonomy.AutomationPolicy, policy Policy, now time.Time) bool {
	if approved.Status != autonomy.AutomationPolicyActive || approved.ActionClass != ActionClassCreateMatter || strings.TrimSpace(approved.Checksum) == "" || approved.EffectiveFrom != nil && approved.EffectiveFrom.After(now) || approved.EffectiveUntil != nil && !approved.EffectiveUntil.After(now) || !strings.EqualFold(strings.TrimSpace(approved.RolloutMode), string(policy.Rollout)) {
		return false
	}
	eligibility, err := json.Marshal(policy.Eligibility)
	if err != nil || !sameJSON(approved.Eligibility, eligibility) {
		return false
	}
	blastRadius, err := json.Marshal(policy.BlastRadius)
	if err != nil || !sameJSON(approved.BlastRadiusLimit, blastRadius) {
		return false
	}
	outcome, err := json.Marshal(policy.Outcome)
	return err == nil && sameJSON(approved.VerificationContract, outcome)
}

func sameJSON(left, right []byte) bool {
	var leftValue, rightValue any
	if json.Unmarshal(left, &leftValue) != nil || json.Unmarshal(right, &rightValue) != nil {
		return false
	}
	leftCanonical, _ := json.Marshal(leftValue)
	rightCanonical, _ := json.Marshal(rightValue)
	return bytes.Equal(leftCanonical, rightCanonical)
}
