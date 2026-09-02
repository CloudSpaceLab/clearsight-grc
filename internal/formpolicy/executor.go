package formpolicy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

const (
	executionPolicyLimit = 100
	executionRunWindow   = 30 * time.Second
)

type ScoredResponseEvent struct {
	ID                 string
	TenantID           string
	ResponseRevisionID string
	OccurredAt         time.Time
}

type ExecutionRoute struct {
	TenantID             string
	LegalEntityID        string
	CanonicalSubjectType string
	CanonicalSubjectID   string
	ServicePrincipalID   string
	OwnerPrincipalID     string
	ReviewerPrincipalID  string
	ProgramID            string
}

type ExecutionResponseReader interface {
	GetCompletedResponseForExecution(context.Context, string, string) (evidence.CompletedResponseSummary, error)
}

type ExecutionAuthority interface {
	ResolvePolicyExecution(context.Context, Policy, evidence.CompletedResponseSummary) (ExecutionRoute, error)
	ResolvePolicyExecutionException(context.Context, Policy, evidence.CompletedResponseSummary) (ExecutionExceptionRoute, error)
}

type ExecutionExceptionRoute struct {
	TenantID      string
	LegalEntityID string
	PrincipalID   string
}

type ExecutionStore interface {
	ListEffectivePolicies(context.Context, string, string, string, int64, time.Time, int) ([]Policy, error)
	ApplyExecution(context.Context, ExecutionCommand) (ExecutionReceipt, error)
	ApplyCompensation(context.Context, CompensationCommand) (CompensationReceipt, error)
}

type ExecutionCommand struct {
	EventID       string
	Policy        Policy
	Response      evidence.CompletedResponseSummary
	Route         ExecutionRoute
	Receipt       ExecutionReceipt
	Episode       AdverseEpisode
	Matter        continuity.Matter
	Outcome       continuity.VerificationContract
	Link          *continuity.MatterLink
	FailureMatter *continuity.Matter
	FailureAction *continuity.Action
}

type CompensationCommand struct {
	Candidate    CompensationCandidate
	Response     evidence.CompletedResponseSummary
	Route        ExecutionRoute
	Receipt      CompensationReceipt
	ReviewMatter continuity.Matter
	ReviewAction continuity.Action
}

type Executor struct {
	store     ExecutionStore
	responses ExecutionResponseReader
	authority ExecutionAuthority
	now       func() time.Time
	newID     func() (string, error)
}

func NewExecutor(store ExecutionStore, responses ExecutionResponseReader, authority ExecutionAuthority) *Executor {
	return &Executor{store: store, responses: responses, authority: authority, now: time.Now, newID: id.NewUUIDv7}
}

func (executor *Executor) HandleCompensation(ctx context.Context, candidate CompensationCandidate) (CompensationReceipt, error) {
	policy, execution := candidate.RollbackPolicy, candidate.OriginalExecution
	if executor == nil || executor.store == nil || executor.responses == nil || executor.authority == nil ||
		policy.Status != PolicyActive || policy.RollbackOfPolicyID == "" || policy.RollbackOfPolicyID != execution.PolicyID ||
		policy.ActivatedAt == nil || execution.CreatedAt.After(*policy.ActivatedAt) || !execution.CreatedMatter || execution.MatterID == "" ||
		policy.TenantID != execution.TenantID || policy.LegalEntityID != execution.LegalEntityID {
		return CompensationReceipt{}, ErrInvalid
	}
	response, err := executor.responses.GetCompletedResponseForExecution(ctx, execution.TenantID, execution.ResponseRevisionID)
	if err != nil {
		return CompensationReceipt{}, err
	}
	if response.ID != execution.ResponseRevisionID || response.TenantID != execution.TenantID || response.LegalEntityID != execution.LegalEntityID {
		return CompensationReceipt{}, ErrInvalid
	}
	route, err := executor.authority.ResolvePolicyExecution(ctx, policy, response)
	if err != nil {
		return CompensationReceipt{}, err
	}
	if !validExecutionRoute(route, policy, response) {
		return CompensationReceipt{}, ErrActivationAuthority
	}
	receiptID, err := executor.newID()
	if err != nil {
		return CompensationReceipt{}, err
	}
	receipt := CompensationReceipt{
		ID: receiptID, TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID,
		RollbackPolicyID: policy.ID, RollbackPolicyVersion: policy.Version,
		OriginalExecutionID: execution.ID, MatterID: execution.MatterID,
		ActorID: route.ServicePrincipalID, ReviewerPrincipalID: route.ReviewerPrincipalID, State: CompensationReviewRequired, ReasonCode: "ROLLBACK_POLICY_ACTIVE", CreatedAt: executor.currentTime(),
	}
	reviewMatterID, err := executor.newID()
	if err != nil {
		return CompensationReceipt{}, err
	}
	reviewActionID, err := executor.newID()
	if err != nil {
		return CompensationReceipt{}, err
	}
	receipt.ReviewMatterID = reviewMatterID
	reviewMatter := continuity.Matter{ID: reviewMatterID, TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, Reference: executorMatterReference(reviewMatterID), Type: continuity.MatterException, Status: continuity.MatterInitialReview, Priority: maxExecutionPriority(policy.Action.Priority), Title: "Review form policy rollback impact", Summary: "Confirm whether the original policy-created issue remains valid after the active rollback revision.", Scope: mustExecutionJSON(map[string]any{"original_matter_id": execution.MatterID, "original_execution_id": execution.ID, "rollback_policy_id": policy.ID, "rollback_policy_version": policy.Version}), SourceType: "FORM_RESPONSE_POLICY_COMPENSATION", SourceID: execution.ID, TriggerType: "FORM_RESPONSE_POLICY_COMPENSATION_REVIEW_REQUIRED", TriggerID: receipt.ID, TriggerKey: "form-response-policy-compensation:" + policy.ID + ":" + execution.ID, KnownFacts: mustExecutionJSON(map[string]any{"original_matter_id": execution.MatterID, "original_execution_id": execution.ID, "rollback_policy_id": policy.ID, "rollback_policy_version": policy.Version}), MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`), OwnerPrincipalID: route.OwnerPrincipalID, RequiredAuthority: "ACCOUNTABLE_OWNER", CreatedAt: receipt.CreatedAt, UpdatedAt: receipt.CreatedAt, Version: 1}
	reviewAction := continuity.Action{ID: reviewActionID, TenantID: policy.TenantID, MatterID: reviewMatterID, OriginKey: "form-response-policy-compensation-review", Title: "Review rollback impact", Description: "Review the original execution receipt and Matter, then record whether the issue remains valid, needs correction or can proceed unchanged.", OwnerPrincipalID: route.ReviewerPrincipalID, RequiredResponsibility: "REVIEWER", Status: continuity.ActionPlanned, CreatedAt: receipt.CreatedAt, UpdatedAt: receipt.CreatedAt, Version: 1}
	return executor.store.ApplyCompensation(ctx, CompensationCommand{Candidate: candidate, Response: response, Route: route, Receipt: receipt, ReviewMatter: reviewMatter, ReviewAction: reviewAction})
}

func (executor *Executor) Handle(ctx context.Context, event ScoredResponseEvent) ([]ExecutionReceipt, error) {
	return executor.HandleBatch(ctx, []ScoredResponseEvent{event})
}

func (executor *Executor) HandleBatch(ctx context.Context, events []ScoredResponseEvent) ([]ExecutionReceipt, error) {
	if executor == nil || executor.store == nil || executor.responses == nil || executor.authority == nil || len(events) == 0 || len(events) > 100 {
		return nil, ErrInvalid
	}
	runApplied := map[string]int{}
	receipts := make([]ExecutionReceipt, 0)
	for _, event := range events {
		if strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.TenantID) == "" || strings.TrimSpace(event.ResponseRevisionID) == "" || event.OccurredAt.IsZero() {
			return nil, ErrInvalid
		}
		response, err := executor.responses.GetCompletedResponseForExecution(ctx, strings.TrimSpace(event.TenantID), strings.TrimSpace(event.ResponseRevisionID))
		if err != nil {
			return nil, err
		}
		if response.TenantID != event.TenantID || strings.TrimSpace(response.LegalEntityID) == "" || response.ID != event.ResponseRevisionID {
			return nil, ErrInvalid
		}
		policies, err := executor.store.ListEffectivePolicies(ctx, response.TenantID, response.LegalEntityID, response.FormTemplateID, response.FormTemplateVersion, response.CompletedAt, executionPolicyLimit)
		if err != nil {
			return nil, err
		}
		for _, policy := range policies {
			if !effectiveForResponse(policy, response, executor.currentTime()) {
				continue
			}
			route, err := executor.authority.ResolvePolicyExecution(ctx, policy, response)
			if err != nil {
				receipt, failureErr := executor.recordAuthorityFailure(ctx, event, policy, response, err)
				return append(receipts, receipt), failureErr
			}
			if !validExecutionRoute(route, policy, response) {
				receipt, failureErr := executor.recordAuthorityFailure(ctx, event, policy, response, ErrActivationAuthority)
				return append(receipts, receipt), failureErr
			}
			state, reason := ExecutionNotMatched, "SCORE_NOT_MATCHED"
			matched := policyMatches(policy, response)
			if matched && policy.Rollout == RolloutShadow {
				state, reason = ExecutionShadow, "POLICY_MATCHED_SHADOW"
			} else if matched {
				key := policy.TenantID + "|" + policy.LegalEntityID + "|" + policy.ID + "|" + fmt.Sprint(policy.Version)
				if runApplied[key] >= policy.BlastRadius.PerRun {
					state, reason = ExecutionBlastSuppressed, "PER_RUN_LIMIT"
				} else {
					state, reason = ExecutionApplied, "POLICY_MATCHED"
					runApplied[key]++
				}
			}
			command, err := executor.executionCommand(event, policy, response, route, state, reason)
			if err != nil {
				return nil, err
			}
			receipt, err := executor.store.ApplyExecution(ctx, command)
			if err != nil {
				return nil, err
			}
			receipts = append(receipts, receipt)
		}
	}
	return receipts, nil
}

func (executor *Executor) recordAuthorityFailure(ctx context.Context, event ScoredResponseEvent, policy Policy, response evidence.CompletedResponseSummary, cause error) (ExecutionReceipt, error) {
	reason := "AUTHORITY_ROUTE_INVALID"
	if errors.Is(cause, ErrAuthorityUnavailable) {
		reason = "AUTHORITY_SERVICE_UNAVAILABLE"
	}
	exceptionRoute, exceptionErr := executor.authority.ResolvePolicyExecutionException(ctx, policy, response)
	principalID := ""
	if exceptionErr == nil && exceptionRoute.TenantID == policy.TenantID && exceptionRoute.LegalEntityID == policy.LegalEntityID {
		principalID = strings.TrimSpace(exceptionRoute.PrincipalID)
	}
	command, err := executor.failedExecutionCommand(event, policy, response, reason, principalID)
	if err != nil {
		return ExecutionReceipt{}, err
	}
	receipt, err := executor.store.ApplyExecution(ctx, command)
	if err != nil {
		return ExecutionReceipt{}, err
	}
	return receipt, errors.Join(cause, exceptionErr)
}

func (executor *Executor) failedExecutionCommand(event ScoredResponseEvent, policy Policy, response evidence.CompletedResponseSummary, reason, exceptionPrincipalID string) (ExecutionCommand, error) {
	receiptID, err := executor.newID()
	if err != nil {
		return ExecutionCommand{}, err
	}
	command := ExecutionCommand{
		EventID: event.ID, Policy: policy, Response: response,
		Receipt: ExecutionReceipt{
			ID: receiptID, TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, PolicyID: policy.ID, PolicyVersion: policy.Version,
			AutomationPolicyID: policy.AutomationPolicyID, AutomationPolicyVersion: policy.AutomationPolicyVersion,
			ResponseRevisionID: response.ID, State: ExecutionFailed, ReasonCode: reason, CreatedAt: executor.currentTime(),
		},
	}
	if strings.TrimSpace(exceptionPrincipalID) == "" {
		return command, nil
	}
	matterID, err := executor.newID()
	if err != nil {
		return ExecutionCommand{}, err
	}
	actionID, err := executor.newID()
	if err != nil {
		return ExecutionCommand{}, err
	}
	now := command.Receipt.CreatedAt
	command.FailureMatter = &continuity.Matter{
		ID: matterID, TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, Reference: executorMatterReference(matterID),
		Type: continuity.MatterException, Status: continuity.MatterInitialReview, Priority: maxExecutionPriority(policy.Action.Priority),
		Title: "Repair form response policy routing", Summary: "A scored response could not be processed because its current authority route is unavailable or invalid.",
		Scope:      mustExecutionJSON(map[string]any{"form_response_policy_id": policy.ID, "policy_version": policy.Version, "response_revision_id": response.ID}),
		SourceType: "FORM_RESPONSE_POLICY_EXECUTION", SourceID: command.Receipt.ID, TriggerType: "FORM_RESPONSE_POLICY_EXECUTION_FAILED", TriggerID: command.Receipt.ID,
		TriggerKey:   "form-response-policy-execution-failure:" + policy.ID + ":" + response.ID,
		KnownFacts:   mustExecutionJSON(map[string]any{"reason_code": reason, "policy_id": policy.ID, "policy_version": policy.Version, "response_revision_id": response.ID}),
		MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`), OwnerPrincipalID: exceptionPrincipalID, RequiredAuthority: "ESCALATION_OWNER", CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	command.FailureAction = &continuity.Action{ID: actionID, TenantID: policy.TenantID, MatterID: matterID, OriginKey: "form-response-policy-execution-recovery", Title: "Restore the authority route", Description: "Review the current subject, Matter and automation routes, then retry the scored response.", OwnerPrincipalID: exceptionPrincipalID, RequiredResponsibility: "ESCALATION_OWNER", Status: continuity.ActionPlanned, CreatedAt: now, UpdatedAt: now, Version: 1}
	return command, nil
}

func maxExecutionPriority(value int) int {
	if value < 3 {
		return 3
	}
	return value
}

func (executor *Executor) executionCommand(event ScoredResponseEvent, policy Policy, response evidence.CompletedResponseSummary, route ExecutionRoute, state ExecutionState, reason string) (ExecutionCommand, error) {
	now := executor.currentTime()
	receiptID, err := executor.newID()
	if err != nil {
		return ExecutionCommand{}, err
	}
	receipt := ExecutionReceipt{ID: receiptID, TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, PolicyID: policy.ID, PolicyVersion: policy.Version, AutomationPolicyID: policy.AutomationPolicyID, AutomationPolicyVersion: policy.AutomationPolicyVersion, ResponseRevisionID: response.ID, State: state, ReasonCode: reason, CreatedAt: now}
	command := ExecutionCommand{EventID: event.ID, Policy: policy, Response: response, Route: route, Receipt: receipt}
	if state != ExecutionApplied {
		return command, nil
	}
	episodeID, err := executor.newID()
	if err != nil {
		return ExecutionCommand{}, err
	}
	matterID, err := executor.newID()
	if err != nil {
		return ExecutionCommand{}, err
	}
	outcomeID, err := executor.newID()
	if err != nil {
		return ExecutionCommand{}, err
	}
	command.Episode = AdverseEpisode{ID: episodeID, TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, PolicyCode: policy.Code, PolicyID: policy.ID, PolicyVersion: policy.Version, SubjectType: route.CanonicalSubjectType, SubjectID: route.CanonicalSubjectID, State: EpisodeOpen, MatterID: matterID, LastResponseRevisionID: response.ID, OpenedAt: now, UpdatedAt: now, RecordVersion: 1}
	command.Matter = continuity.Matter{
		ID: matterID, TenantID: policy.TenantID, LegalEntityID: policy.LegalEntityID, Reference: executorMatterReference(matterID), Type: continuity.MatterType(policy.Action.Type), Status: continuity.MatterInitialReview, Priority: policy.Action.Priority,
		Title: renderMatterTemplate(policy.Action.TitleTemplate, response), Summary: renderMatterTemplate(policy.Action.SummaryTemplate, response),
		Scope:      mustExecutionJSON(map[string]any{"form_response_policy_id": policy.ID, "policy_version": policy.Version, "response_revision_id": response.ID, "subject_type": route.CanonicalSubjectType, "subject_id": route.CanonicalSubjectID}),
		SourceType: "FORM_RESPONSE", SourceID: response.ID, TriggerType: "FORM_RESPONSE_POLICY_MATCHED", TriggerID: receipt.ID,
		TriggerKey:   fmt.Sprintf("form-response-policy:%s:%s:%s:%s", policy.Code, strings.ToLower(route.CanonicalSubjectType), route.CanonicalSubjectID, episodeID),
		KnownFacts:   mustExecutionJSON(map[string]any{"form_title": response.Title, "score_state": response.Score.State, "raw_score": response.Score.RawScore, "adverse_score": response.Score.AdverseScore, "concern": response.Score.Band, "coverage": response.Score.Coverage}),
		MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`), OwnerPrincipalID: route.OwnerPrincipalID, RequiredAuthority: "ACCOUNTABLE_OWNER", CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	command.Outcome = continuity.VerificationContract{
		ID: outcomeID, TenantID: policy.TenantID, MatterID: matterID, ExpectedOutcome: policy.Outcome.ExpectedOutcome,
		Baseline:  mustExecutionJSON(map[string]any{"response_revision_id": response.ID, "score_state": response.Score.State, "adverse_score": response.Score.AdverseScore}),
		Scope:     mustExecutionJSON(map[string]any{"policy_id": policy.ID, "policy_version": policy.Version, "subject_type": route.CanonicalSubjectType, "subject_id": route.CanonicalSubjectID}),
		Threshold: json.RawMessage(`{"result":"PASS"}`), ObservationPeriodMinutes: policy.Outcome.CheckAfterMinutes,
		AuthorityPrincipalID: route.ReviewerPrincipalID, FailureResponse: policy.Outcome.FailureResponse,
		Status: continuity.VerificationActive, CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	if strings.EqualFold(route.CanonicalSubjectType, "PROGRAM") {
		linkID, linkErr := executor.newID()
		if linkErr != nil {
			return ExecutionCommand{}, linkErr
		}
		command.Link = &continuity.MatterLink{ID: linkID, TenantID: policy.TenantID, MatterID: matterID, ProgramID: route.ProgramID, Relationship: "AFFECTS", CreatedAt: now}
	}
	return command, nil
}

func effectiveForResponse(policy Policy, response evidence.CompletedResponseSummary, now time.Time) bool {
	return policy.Status == PolicyActive && policy.ActivatedAt != nil && !policy.ActivatedAt.After(response.CompletedAt) && !policy.ActivatedAt.After(now) && policy.TenantID == response.TenantID && policy.LegalEntityID == response.LegalEntityID && policy.Eligibility.FormTemplateID == response.FormTemplateID && policy.Eligibility.FormTemplateVersion == response.FormTemplateVersion && (policy.EffectiveFrom == nil || !policy.EffectiveFrom.After(response.CompletedAt)) && (policy.EffectiveUntil == nil || policy.EffectiveUntil.After(response.CompletedAt))
}

func validExecutionRoute(route ExecutionRoute, policy Policy, response evidence.CompletedResponseSummary) bool {
	programValid := !strings.EqualFold(response.SubjectType, "PROGRAM") || route.ProgramID == response.SubjectID
	return strings.TrimSpace(route.ServicePrincipalID) != "" && strings.TrimSpace(route.OwnerPrincipalID) != "" && strings.TrimSpace(route.ReviewerPrincipalID) != "" && route.TenantID == policy.TenantID && route.LegalEntityID == policy.LegalEntityID && strings.EqualFold(route.CanonicalSubjectType, response.SubjectType) && route.CanonicalSubjectID == response.SubjectID && programValid
}

func validCompensationCommand(command CompensationCommand) bool {
	policy, execution, receipt := command.Candidate.RollbackPolicy, command.Candidate.OriginalExecution, command.Receipt
	return strings.TrimSpace(receipt.ID) != "" && receipt.State == CompensationReviewRequired && !receipt.CreatedAt.IsZero() &&
		receipt.TenantID == policy.TenantID && receipt.LegalEntityID == policy.LegalEntityID &&
		receipt.RollbackPolicyID == policy.ID && receipt.RollbackPolicyVersion == policy.Version &&
		receipt.OriginalExecutionID == execution.ID && receipt.MatterID == execution.MatterID &&
		receipt.ActorID == command.Route.ServicePrincipalID && strings.TrimSpace(receipt.ActorID) != "" &&
		receipt.ReviewerPrincipalID == command.Route.ReviewerPrincipalID && strings.TrimSpace(receipt.ReviewerPrincipalID) != "" && receipt.ReviewMatterID == command.ReviewMatter.ID &&
		command.ReviewMatter.TenantID == receipt.TenantID && command.ReviewMatter.LegalEntityID == receipt.LegalEntityID && command.ReviewAction.TenantID == receipt.TenantID && command.ReviewAction.MatterID == command.ReviewMatter.ID && command.ReviewAction.OwnerPrincipalID == receipt.ReviewerPrincipalID && command.ReviewAction.RequiredResponsibility == "REVIEWER" &&
		policy.RollbackOfPolicyID == execution.PolicyID && execution.CreatedMatter && execution.MatterID != "" &&
		command.Response.ID == execution.ResponseRevisionID && validExecutionRoute(command.Route, policy, command.Response)
}

func renderMatterTemplate(template string, response evidence.CompletedResponseSummary) string {
	values := map[string]string{"form_title": response.Title, "subject_type": response.SubjectType, "subject_id": response.SubjectID}
	if response.Score != nil {
		values["concern"] = string(response.Score.Band)
		if response.Score.RawScore != nil {
			values["score"] = fmt.Sprintf("%.2f", *response.Score.RawScore)
		}
	}
	return templateVariablePattern.ReplaceAllStringFunc(template, func(match string) string {
		parts := templateVariablePattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return ""
		}
		return values[parts[1]]
	})
}

func executorMatterReference(value string) string {
	clean := strings.ToUpper(strings.ReplaceAll(value, "-", ""))
	if len(clean) > 16 {
		clean = clean[len(clean)-16:]
	}
	return "MAT-" + clean
}

func mustExecutionJSON(value any) json.RawMessage {
	raw, _ := json.Marshal(value)
	return raw
}

func (executor *Executor) currentTime() time.Time {
	if executor == nil || executor.now == nil {
		return time.Now().UTC()
	}
	return executor.now().UTC()
}
