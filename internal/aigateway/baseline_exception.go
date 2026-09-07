package aigateway

import "time"

const (
	GatewayBaselineExceptionCodeRoot    = "ORG_AI_BASELINE_EXCEPTION"
	GatewayBaselineExceptionActionClass = "AI_GATEWAY_BASELINE_EXCEPTION"
	MaxBaselineExceptionRules           = 16
	MaxBaselineExceptionWorkloads       = 16
)

// PolicyRevisionRef is compact reconstructive attribution for a governed policy
// revision that affected a gateway decision.
type PolicyRevisionRef struct {
	ID      string `json:"id"`
	Code    string `json:"code"`
	Version int64  `json:"version"`
}

// BaselineException is the already-approved, runtime-safe projection of an
// exception policy. It can waive only named rules on one exact organization
// baseline revision and one bounded workload/environment scope.
type BaselineException struct {
	PolicyRevisionRef
	TargetBaselineID      string    `json:"target_baseline_id"`
	TargetBaselineVersion int64     `json:"target_baseline_version"`
	WorkloadRecordIDs     []string  `json:"workload_record_ids"`
	Environments          []string  `json:"environments"`
	WaivedRuleIDs         []string  `json:"waived_rule_ids"`
	ExpiresAt             time.Time `json:"expires_at"`
}
