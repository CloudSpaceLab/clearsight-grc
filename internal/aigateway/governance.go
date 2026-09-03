package aigateway

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	GatewayBaselinePolicyCode          = "ORG_AI_BASELINE"
	GatewayBaselineActionClass         = "AI_GATEWAY_BASELINE"
	ObligationOrganizationInstruction  = "ORG_INSTRUCTION"
	FactPromptInjectionRisk            = "gateway.prompt_injection_risk"
	FactInstructionExfiltration        = "gateway.instruction_exfiltration_attempt"
	FactUntrustedContent               = "gateway.untrusted_content_present"
	maxOrganizationInstructions        = 8
	maxOrganizationInstructionBytes    = 4096
	maxOrganizationInstructionTotal    = 16384
)

type DecisionAction string

const (
	DecisionAllow           DecisionAction = "ALLOW"
	DecisionDeny            DecisionAction = "DENY"
	DecisionModify          DecisionAction = "MODIFY"
	DecisionRoute           DecisionAction = "ROUTE"
	DecisionRequireApproval DecisionAction = "REQUIRE_APPROVAL"
	DecisionShadow          DecisionAction = "SHADOW"
)

type RolloutMode string

const (
	RolloutShadow  RolloutMode = "SHADOW"
	RolloutEnforce RolloutMode = "ENFORCE"
)

type BindingResolutionMode string

const (
	ResolutionMetadata        BindingResolutionMode = "METADATA"
	ResolutionLiveLookup      BindingResolutionMode = "LIVE_LOOKUP"
	ResolutionAdapterCache    BindingResolutionMode = "ADAPTER_CACHE"
	ResolutionAsync           BindingResolutionMode = "ASYNC"
	ResolutionExternalControl BindingResolutionMode = "EXTERNAL_CONTROL"
)

type FactState string

const (
	FactKnown       FactState = "KNOWN"
	FactUnknown     FactState = "UNKNOWN"
	FactStale       FactState = "STALE"
	FactUnavailable FactState = "UNAVAILABLE"
)

type Fact struct {
	Key        string    `json:"key"`
	Value      string    `json:"value,omitempty"`
	State      FactState `json:"state"`
	Source     string    `json:"source,omitempty"`
	ObservedAt time.Time `json:"observed_at,omitempty"`
}

type BindingRequirement struct {
	BindingID      string                `json:"binding_id"`
	BindingVersion int64                 `json:"binding_version"`
	Mode           BindingResolutionMode `json:"mode"`
	FactKey        string                `json:"fact_key"`
	MetadataKey    string                `json:"metadata_key,omitempty"`
	LookupField    string                `json:"lookup_field,omitempty"`
	Required       bool                  `json:"required"`
	MaxAgeSeconds  int64                 `json:"max_age_seconds,omitempty"`
}

type PolicyRule struct {
	ID          string         `json:"id"`
	Priority    int            `json:"priority"`
	FactKey     string         `json:"fact_key,omitempty"`
	Operator    string         `json:"operator,omitempty"`
	Value       string         `json:"value,omitempty"`
	Action      DecisionAction `json:"action"`
	RouteID     string         `json:"route_id,omitempty"`
	ReasonCode  string         `json:"reason_code"`
	Obligations []Obligation   `json:"obligations,omitempty"`
	Redactions  []Redaction    `json:"redactions,omitempty"`
}

type ResponseControl struct {
	MaxBytes       int      `json:"max_bytes,omitempty"`
	DenyPatterns   []string `json:"deny_patterns,omitempty"`
	RedactPatterns []string `json:"redact_patterns,omitempty"`
	AllowStreaming bool     `json:"allow_streaming"`
}

type PolicyDefinition struct {
	Bindings        []BindingRequirement `json:"bindings,omitempty"`
	Rules           []PolicyRule         `json:"rules,omitempty"`
	DefaultAction   DecisionAction       `json:"default_action,omitempty"`
	ResponseControl ResponseControl      `json:"response_control,omitempty"`
}

type PolicySnapshot struct {
	ID          string           `json:"id"`
	Code        string           `json:"code"`
	Version     int64            `json:"version"`
	RolloutMode RolloutMode      `json:"rollout_mode"`
	Definition  PolicyDefinition `json:"definition"`
	Baseline    *PolicySnapshot  `json:"baseline,omitempty"`
}

type Obligation struct {
	Code   string `json:"code"`
	Detail string `json:"detail,omitempty"`
}

type Redaction struct {
	Target      string `json:"target"`
	Pattern     string `json:"pattern"`
	Replacement string `json:"replacement,omitempty"`
}

type Decision struct {
	PolicyID        string          `json:"policy_id"`
	PolicyCode      string          `json:"policy_code"`
	PolicyVersion   int64           `json:"policy_version"`
	RolloutMode     RolloutMode     `json:"rollout_mode"`
	Action          DecisionAction  `json:"action"`
	ProposedAction  DecisionAction  `json:"proposed_action,omitempty"`
	RouteID         string          `json:"route_id,omitempty"`
	ReasonCodes     []string        `json:"reason_codes,omitempty"`
	Obligations     []Obligation    `json:"obligations,omitempty"`
	Redactions      []Redaction     `json:"redactions,omitempty"`
	ResponseControl ResponseControl `json:"-"`

	BaselinePolicyID       string         `json:"baseline_policy_id,omitempty"`
	BaselinePolicyCode     string         `json:"baseline_policy_code,omitempty"`
	BaselinePolicyVersion  int64          `json:"baseline_policy_version,omitempty"`
	BaselineRolloutMode    RolloutMode    `json:"baseline_rollout_mode,omitempty"`
	BaselineAction         DecisionAction `json:"baseline_action,omitempty"`
	BaselineProposedAction DecisionAction `json:"baseline_proposed_action,omitempty"`
	BaselineReasonCodes    []string       `json:"baseline_reason_codes,omitempty"`
}

type GovernedRequest struct {
	Request  Request  `json:"request"`
	Decision Decision `json:"decision"`
}

type WorkloadProvider interface {
	Authenticate(context.Context, string) (*Workload, error)
	Ready() bool
}

type FactResolver interface {
	ResolveFacts(context.Context, Workload, Request, []BindingRequirement) ([]Fact, error)
}

type ReceiptRecord struct {
	RequestID  string
	WorkloadID string
	TenantID   string
	ModelAlias string
	RouteID    string
	Decision   Decision
	Outcome    string
	ErrorCode  string
	ObservedAt time.Time
}

type ReceiptRecorder interface {
	RecordReceipt(context.Context, ReceiptRecord) error
}

type defaultFactResolver struct{}

func (defaultFactResolver) ResolveFacts(_ context.Context, workload Workload, request Request, requirements []BindingRequirement) ([]Fact, error) {
	facts := make([]Fact, 0, len(requirements))
	now := time.Now().UTC()
	for _, requirement := range requirements {
		if requirement.Mode != ResolutionMetadata {
			state := FactUnknown
			if requirement.Required {
				state = FactUnavailable
			}
			facts = append(facts, Fact{Key: requirement.FactKey, State: state})
			continue
		}
		key := strings.TrimSpace(requirement.MetadataKey)
		if key == "" {
			key = strings.TrimSpace(requirement.FactKey)
		}
		value, ok := workload.VerifiedMetadata[key]
		if !ok {
			value, ok = request.Metadata[key]
		}
		if !ok {
			facts = append(facts, Fact{Key: requirement.FactKey, State: FactUnknown})
			continue
		}
		facts = append(facts, Fact{Key: requirement.FactKey, Value: value, State: FactKnown, Source: "METADATA", ObservedAt: now})
	}
	return facts, nil
}

func staticPolicy(workload Workload) PolicySnapshot {
	return PolicySnapshot{
		ID: "static:" + workload.ID, Code: "STATIC_ALLOWLIST", Version: 1, RolloutMode: RolloutEnforce,
		Definition: PolicyDefinition{DefaultAction: DecisionAllow},
	}
}

// EvaluatePolicy evaluates the exact workload policy and, when present, the
// tenant-wide organization baseline. The baseline is a separate governed policy
// revision; it can strengthen or mutate an authorized request but cannot be
// weakened by the workload policy. Both revisions remain explicitly attributed
// on the returned decision.
func EvaluatePolicy(policy PolicySnapshot, workload Workload, request Request, facts []Fact) (Decision, error) {
	factMap := makeFactMap(request, facts)
	workloadPolicy := policy
	workloadPolicy.Baseline = nil
	workloadDecision, err := evaluateSinglePolicy(workloadPolicy, workload, request, factMap)
	if err != nil {
		return workloadDecision, err
	}
	if policy.Baseline == nil || strings.TrimSpace(policy.Baseline.ID) == "" || policy.Baseline.ID == policy.ID {
		return combinePolicyDecisions(workloadDecision, nil), nil
	}
	baselinePolicy := *policy.Baseline
	baselinePolicy.Baseline = nil
	baselineDecision, err := evaluateSinglePolicy(baselinePolicy, workload, request, factMap)
	if err != nil {
		return workloadDecision, err
	}
	return combinePolicyDecisions(workloadDecision, &baselineDecision), nil
}

func makeFactMap(request Request, facts []Fact) map[string]Fact {
	factMap := make(map[string]Fact, len(facts)+3)
	for _, fact := range facts {
		if strings.TrimSpace(fact.Key) == "" || strings.HasPrefix(fact.Key, "gateway.") {
			continue
		}
		if previous, ok := factMap[fact.Key]; ok && factPrecedence(previous.State) >= factPrecedence(fact.State) {
			continue
		}
		factMap[fact.Key] = fact
	}
	for _, fact := range gatewaySecurityFacts(request) {
		factMap[fact.Key] = fact
	}
	return factMap
}

func evaluateSinglePolicy(policy PolicySnapshot, workload Workload, request Request, factMap map[string]Fact) (Decision, error) {
	if strings.TrimSpace(policy.ID) == "" || strings.TrimSpace(policy.Code) == "" || policy.Version < 1 {
		return Decision{}, fmt.Errorf("invalid policy snapshot")
	}
	mode := policy.RolloutMode
	if mode == "" {
		mode = RolloutShadow
	}
	if mode != RolloutShadow && mode != RolloutEnforce {
		return Decision{}, fmt.Errorf("invalid rollout mode")
	}
	decision := Decision{PolicyID: policy.ID, PolicyCode: policy.Code, PolicyVersion: policy.Version, RolloutMode: mode, ResponseControl: cloneResponseControl(policy.Definition.ResponseControl)}
	for _, requirement := range policy.Definition.Bindings {
		if !requirement.Required {
			continue
		}
		fact, ok := factMap[requirement.FactKey]
		if !ok || fact.State == FactUnknown || fact.State == FactStale || fact.State == FactUnavailable {
			proposed := DecisionDeny
			reason := "SOURCE_FACT_UNKNOWN"
			if ok {
				switch fact.State {
				case FactStale:
					reason = "SOURCE_FACT_STALE"
				case FactUnavailable:
					reason = "SOURCE_FACT_UNAVAILABLE"
				}
			}
			return finalizeDecision(decision, proposed, reason, nil, nil, ""), nil
		}
	}

	rules := append([]PolicyRule(nil), policy.Definition.Rules...)
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].Priority > rules[j].Priority })
	for _, rule := range rules {
		matches, err := ruleMatches(rule, workload, request, factMap)
		if err != nil {
			return decision, err
		}
		if !matches {
			continue
		}
		action := rule.Action
		if action == "" {
			action = DecisionDeny
		}
		return finalizeDecision(decision, action, rule.ReasonCode, rule.Obligations, rule.Redactions, rule.RouteID), nil
	}
	action := policy.Definition.DefaultAction
	if action == "" {
		action = DecisionAllow
	}
	return finalizeDecision(decision, action, "POLICY_DEFAULT", nil, nil, ""), nil
}

func combinePolicyDecisions(workloadDecision Decision, baselineDecision *Decision) Decision {
	combined := workloadDecision
	combined.Obligations = nil
	combined.Redactions = nil
	combined.RouteID = ""
	combined.ResponseControl = ResponseControl{}
	combined.ReasonCodes = append([]string(nil), workloadDecision.ReasonCodes...)

	workloadEnforcing := workloadDecision.RolloutMode == RolloutEnforce
	if workloadEnforcing {
		combined.Obligations = append(combined.Obligations, workloadDecision.Obligations...)
		combined.Redactions = append(combined.Redactions, workloadDecision.Redactions...)
		combined.RouteID = strings.TrimSpace(workloadDecision.RouteID)
		combined.ResponseControl = cloneResponseControl(workloadDecision.ResponseControl)
	}

	var baseline Decision
	baselineEnforcing := false
	if baselineDecision != nil {
		baseline = *baselineDecision
		combined.BaselinePolicyID = baseline.PolicyID
		combined.BaselinePolicyCode = baseline.PolicyCode
		combined.BaselinePolicyVersion = baseline.PolicyVersion
		combined.BaselineRolloutMode = baseline.RolloutMode
		combined.BaselineAction = baseline.Action
		combined.BaselineProposedAction = baseline.ProposedAction
		combined.BaselineReasonCodes = append([]string(nil), baseline.ReasonCodes...)
		baselineEnforcing = baseline.RolloutMode == RolloutEnforce
		if baselineEnforcing {
			// Workload mutations are applied first. Baseline redactions/routes are
			// appended/selected last so a lower layer cannot undo them.
			combined.Redactions = append(combined.Redactions, baseline.Redactions...)
			combined.Obligations = append(append([]Obligation(nil), baseline.Obligations...), combined.Obligations...)
			if route := strings.TrimSpace(baseline.RouteID); route != "" {
				combined.RouteID = route
			}
			combined.ResponseControl = mergeResponseControls(combined.ResponseControl, baseline.ResponseControl)
		}
	}

	workloadAction := enforcedAction(workloadDecision)
	baselineAction := DecisionAllow
	if baselineDecision != nil {
		baselineAction = enforcedAction(baseline)
	}
	combined.Action = strongestEffectiveAction(workloadAction, baselineAction, len(combined.Redactions) > 0, combined.RouteID != "")
	combined.ProposedAction = ""

	if combined.Action == DecisionAllow || combined.Action == DecisionShadow {
		if proposed := strongestProposedAction(workloadDecision, baselineDecision); proposed != DecisionAllow {
			combined.Action = DecisionShadow
			combined.ProposedAction = proposed
		}
	}

	if baselineDecision != nil && (baselineEnforcing || baseline.Action == DecisionShadow || baseline.ProposedAction != "") {
		combined.ReasonCodes = uniqueSortedStrings(append(combined.ReasonCodes, baseline.ReasonCodes...))
	}
	return combined
}

func enforcedAction(decision Decision) DecisionAction {
	if decision.RolloutMode != RolloutEnforce {
		return DecisionAllow
	}
	if decision.Action == "" || decision.Action == DecisionShadow {
		return DecisionAllow
	}
	return decision.Action
}

func strongestEffectiveAction(workload, baseline DecisionAction, hasRedactions, hasRoute bool) DecisionAction {
	for _, action := range []DecisionAction{DecisionDeny, DecisionRequireApproval} {
		if baseline == action || workload == action {
			return action
		}
	}
	if baseline == DecisionModify || workload == DecisionModify || hasRedactions {
		return DecisionModify
	}
	if baseline == DecisionRoute || workload == DecisionRoute || hasRoute {
		return DecisionRoute
	}
	return DecisionAllow
}

func strongestProposedAction(workload Decision, baseline *Decision) DecisionAction {
	candidates := []DecisionAction{workload.ProposedAction}
	if baseline != nil {
		candidates = append(candidates, baseline.ProposedAction)
	}
	for _, priority := range []DecisionAction{DecisionDeny, DecisionRequireApproval, DecisionModify, DecisionRoute} {
		for _, candidate := range candidates {
			if candidate == priority {
				return priority
			}
		}
	}
	return DecisionAllow
}

func cloneResponseControl(control ResponseControl) ResponseControl {
	control.DenyPatterns = append([]string(nil), control.DenyPatterns...)
	control.RedactPatterns = append([]string(nil), control.RedactPatterns...)
	return control
}

func mergeResponseControls(a, b ResponseControl) ResponseControl {
	out := cloneResponseControl(a)
	if out.MaxBytes == 0 || (b.MaxBytes > 0 && b.MaxBytes < out.MaxBytes) {
		out.MaxBytes = b.MaxBytes
	}
	out.DenyPatterns = uniqueSortedStrings(append(out.DenyPatterns, b.DenyPatterns...))
	out.RedactPatterns = uniqueSortedStrings(append(out.RedactPatterns, b.RedactPatterns...))
	if responseControlConfigured(a) && responseControlConfigured(b) {
		out.AllowStreaming = a.AllowStreaming && b.AllowStreaming
	} else if responseControlConfigured(b) {
		out.AllowStreaming = b.AllowStreaming
	}
	return out
}

func responseControlConfigured(control ResponseControl) bool {
	return control.MaxBytes > 0 || len(control.DenyPatterns) > 0 || len(control.RedactPatterns) > 0 || control.AllowStreaming
}

func gatewaySecurityFacts(request Request) []Fact {
	now := time.Now().UTC()
	var text strings.Builder
	for _, message := range request.Messages {
		if message.Role != RoleUser && message.Role != RoleTool {
			continue
		}
		if text.Len() > 0 {
			text.WriteByte('\n')
		}
		text.WriteString(strings.ToLower(message.Text))
	}
	value := text.String()
	score := 0
	patterns := []string{
		"ignore previous instructions",
		"ignore all previous instructions",
		"disregard previous instructions",
		"override system instructions",
		"override developer instructions",
		"bypass safety",
		"jailbreak",
		"do anything now",
	}
	for _, pattern := range patterns {
		if strings.Contains(value, pattern) {
			score++
		}
	}
	exfiltration := containsAny(value, "reveal system prompt", "show system prompt", "print system prompt", "expose system prompt", "reveal developer message", "show developer message", "hidden instructions")
	if exfiltration {
		score += 2
	}
	risk := "LOW"
	if score == 1 {
		risk = "MEDIUM"
	} else if score >= 2 {
		risk = "HIGH"
	}
	untrusted := strings.EqualFold(strings.TrimSpace(request.Metadata["untrusted_content"]), "true") || strings.EqualFold(strings.TrimSpace(request.Metadata["content_trust"]), "untrusted")
	return []Fact{
		{Key: FactPromptInjectionRisk, Value: risk, State: FactKnown, Source: "GATEWAY_DETECTOR", ObservedAt: now},
		{Key: FactInstructionExfiltration, Value: fmt.Sprint(exfiltration), State: FactKnown, Source: "GATEWAY_DETECTOR", ObservedAt: now},
		{Key: FactUntrustedContent, Value: fmt.Sprint(untrusted), State: FactKnown, Source: "GATEWAY_DETECTOR", ObservedAt: now},
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

func factPrecedence(state FactState) int {
	switch state {
	case FactKnown:
		return 4
	case FactUnavailable:
		return 3
	case FactStale:
		return 2
	default:
		return 1
	}
}

func ruleMatches(rule PolicyRule, workload Workload, request Request, facts map[string]Fact) (bool, error) {
	key := strings.TrimSpace(rule.FactKey)
	var value string
	var present bool
	switch {
	case key == "model":
		value, present = request.ModelAlias, true
	case key == "purpose":
		value, present = workload.Purpose, strings.TrimSpace(workload.Purpose) != ""
	case strings.HasPrefix(key, "metadata."):
		value, present = request.Metadata[strings.TrimPrefix(key, "metadata.")]
	default:
		fact, ok := facts[key]
		if ok && fact.State == FactKnown {
			value, present = fact.Value, true
		}
	}
	op := strings.ToUpper(strings.TrimSpace(rule.Operator))
	if op == "" {
		op = "EQ"
	}
	switch op {
	case "EXISTS":
		return present, nil
	case "MISSING":
		return !present, nil
	case "EQ":
		return present && value == rule.Value, nil
	case "NEQ":
		return present && value != rule.Value, nil
	case "PREFIX":
		return present && strings.HasPrefix(value, rule.Value), nil
	case "CONTAINS":
		return present && strings.Contains(value, rule.Value), nil
	default:
		return false, fmt.Errorf("unsupported policy operator %q", rule.Operator)
	}
}

func finalizeDecision(base Decision, proposed DecisionAction, reason string, obligations []Obligation, redactions []Redaction, routeID string) Decision {
	if proposed == "" {
		proposed = DecisionAllow
	}
	base.Action = proposed
	base.RouteID = strings.TrimSpace(routeID)
	if strings.TrimSpace(reason) != "" {
		base.ReasonCodes = uniqueSortedStrings([]string{strings.TrimSpace(reason)})
	}
	base.Obligations = append([]Obligation(nil), obligations...)
	base.Redactions = append([]Redaction(nil), redactions...)
	if base.RolloutMode == RolloutShadow && proposed != DecisionAllow {
		base.ProposedAction = proposed
		base.Action = DecisionShadow
	}
	return base
}

func ApplyDecision(request Request, decision Decision) (Request, error) {
	switch decision.Action {
	case DecisionDeny:
		return request, ErrPolicyDenied
	case DecisionRequireApproval:
		return request, ErrApprovalRequired
	case DecisionAllow, DecisionShadow, DecisionModify, DecisionRoute:
	default:
		return request, ErrPolicyUnavailable
	}

	mutated := request
	if decision.RolloutMode == RolloutShadow && decision.BaselinePolicyID == "" {
		return mutated, nil
	}
	if route := strings.TrimSpace(decision.RouteID); route != "" {
		if !validIdentifier(route) {
			return request, invalid("route", "The governed route is invalid.")
		}
		mutated.RouteID = route
	}
	if len(decision.Redactions) > 0 {
		messages := append([]Message(nil), mutated.Messages...)
		for _, redaction := range decision.Redactions {
			if strings.ToLower(strings.TrimSpace(redaction.Target)) != "prompt" {
				continue
			}
			re, err := regexp.Compile(redaction.Pattern)
			if err != nil {
				return request, invalid("policy", "The governed redaction pattern is invalid.")
			}
			replacement := redaction.Replacement
			if replacement == "" {
				replacement = "[REDACTED]"
			}
			for i := range messages {
				messages[i].Text = re.ReplaceAllString(messages[i].Text, replacement)
			}
		}
		mutated.Messages = messages
	}
	return applyOrganizationInstructions(mutated, decision.Obligations)
}

func applyOrganizationInstructions(request Request, obligations []Obligation) (Request, error) {
	instructions := make([]Message, 0, maxOrganizationInstructions)
	total := 0
	for _, obligation := range obligations {
		if !strings.EqualFold(strings.TrimSpace(obligation.Code), ObligationOrganizationInstruction) {
			continue
		}
		instruction := strings.TrimSpace(obligation.Detail)
		if instruction == "" || !utf8.ValidString(instruction) || len(instruction) > maxOrganizationInstructionBytes || len(instructions) >= maxOrganizationInstructions {
			return request, invalid("policy", "The organization instruction obligation is invalid or outside the supported limits.")
		}
		total += len(instruction)
		if total > maxOrganizationInstructionTotal {
			return request, invalid("policy", "The organization instruction obligations exceed the supported limit.")
		}
		instructions = append(instructions, Message{Role: RoleSystem, Text: instruction})
	}
	if len(instructions) == 0 {
		return request, nil
	}
	messages := make([]Message, 0, len(instructions)+len(request.Messages))
	messages = append(messages, instructions...)
	messages = append(messages, request.Messages...)
	request.Messages = messages
	return request, nil
}

func InspectResponse(control ResponseControl, response Response, streaming bool) (Response, error) {
	configured := control.MaxBytes > 0 || len(control.RedactPatterns) > 0 || len(control.DenyPatterns) > 0
	// Pattern inspection/redaction and the current bounded response-size control are
	// whole-response operations. Do not pretend they can be enforced truthfully
	// after SSE bytes have already been committed downstream. A policy can permit
	// streaming only when it configures no whole-response control.
	if streaming && configured {
		return Response{}, invalid("stream", "Streaming is unavailable when governed response inspection requires whole-response control.")
	}
	if control.MaxBytes <= 0 {
		control.MaxBytes = MaxTextBytes
	}
	if control.MaxBytes > MaxTextBytes || len(response.Text) > control.MaxBytes || len(response.Refusal) > control.MaxBytes {
		return Response{}, ErrPolicyDenied
	}
	for _, pattern := range control.DenyPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return Response{}, ErrPolicyUnavailable
		}
		if re.MatchString(response.Text) || re.MatchString(response.Refusal) {
			return Response{}, ErrPolicyDenied
		}
	}
	for _, pattern := range control.RedactPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return Response{}, ErrPolicyUnavailable
		}
		response.Text = re.ReplaceAllString(response.Text, "[REDACTED]")
		response.Refusal = re.ReplaceAllString(response.Refusal, "[REDACTED]")
	}
	return response, nil
}

func uniqueSortedStrings(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func formatPolicyVersion(policy PolicySnapshot) string {
	if strings.TrimSpace(policy.Code) == "" || policy.Version < 1 {
		return ""
	}
	return fmt.Sprintf("%s:%d", policy.Code, policy.Version)
}

func validateRequestMetadata(metadata map[string]string) error {
	if len(metadata) > 64 {
		return invalid("metadata", "The request metadata contains too many entries.")
	}
	for key, value := range metadata {
		if !validIdentifier(key) || len(value) > 4096 || !utf8.ValidString(value) {
			return invalid("metadata", "The request metadata is invalid or outside the supported limits.")
		}
	}
	return nil
}
