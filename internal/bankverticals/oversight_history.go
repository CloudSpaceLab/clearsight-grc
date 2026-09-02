package bankverticals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

const referenceOversightJourneyCode = "OVERSIGHT_HISTORY"

type referenceMatterHistory struct {
	Key, Title, Summary string
	Type                continuity.MatterType
	Priority            int
	OwnerID             string
	ActionOwnerID       string
	OpenedBefore        time.Duration
	WorkStartedAfter    time.Duration
	ImplementedAfter    time.Duration
	VerifiedAfter       time.Duration
	ClosedAfter         time.Duration
	DueAfter            time.Duration
	ReassignTo          string
	ReturnDecision      bool
	Reopen              bool
}

func referenceOversightHistories(config SeedConfig) []referenceMatterHistory {
	owners := []string{config.OwnerPrincipalID, config.ActorID, config.ReviewerPrincipalID}
	return []referenceMatterHistory{
		{Key: "vendor-access", Title: "Review vendor privileged-access evidence", Summary: "Confirm the vendor's current privileged-access controls and independent review evidence.", Type: continuity.MatterVendorReview, Priority: 3, OwnerID: owners[0], ActionOwnerID: config.ContributorPrincipalID, OpenedBefore: 82 * 24 * time.Hour, WorkStartedAfter: 4 * time.Hour, ImplementedAfter: 22 * time.Hour, VerifiedAfter: 28 * time.Hour, ClosedAfter: 30 * time.Hour, DueAfter: 40 * time.Hour},
		{Key: "vendor-resilience", Title: "Review vendor resilience test evidence", Summary: "Confirm the latest service resilience exercise and recorded recovery outcome.", Type: continuity.MatterVendorReview, Priority: 4, OwnerID: owners[1], ActionOwnerID: config.ContributorPrincipalID, OpenedBefore: 70 * 24 * time.Hour, WorkStartedAfter: 5 * time.Hour, ImplementedAfter: 32 * time.Hour, VerifiedAfter: 40 * time.Hour, ClosedAfter: 42 * time.Hour, DueAfter: 36 * time.Hour, ReassignTo: owners[0]},
		{Key: "vendor-certification", Title: "Review vendor certification renewal", Summary: "Confirm that the vendor's renewed assurance certificate covers the supplied service.", Type: continuity.MatterVendorReview, Priority: 3, OwnerID: owners[0], ActionOwnerID: config.ContributorPrincipalID, OpenedBefore: 58 * 24 * time.Hour, WorkStartedAfter: 6 * time.Hour, ImplementedAfter: 42 * time.Hour, VerifiedAfter: 52 * time.Hour, ClosedAfter: 55 * time.Hour, DueAfter: 72 * time.Hour, ReturnDecision: true},
		{Key: "vendor-address", Title: "Verify vendor registered office address", Summary: "Confirm the registered office address against current independent evidence.", Type: continuity.MatterVendorReview, Priority: 2, OwnerID: owners[1], ActionOwnerID: config.ContributorPrincipalID, OpenedBefore: 43 * 24 * time.Hour, WorkStartedAfter: 8 * time.Hour, ImplementedAfter: 58 * time.Hour, VerifiedAfter: 70 * time.Hour, ClosedAfter: 74 * time.Hour, DueAfter: 96 * time.Hour},
		{Key: "vendor-payment-controls", Title: "Review vendor payment-control evidence", Summary: "Confirm the current operating evidence for payment processing safeguards.", Type: continuity.MatterVendorReview, Priority: 5, OwnerID: owners[2], ActionOwnerID: config.ContributorPrincipalID, OpenedBefore: 28 * 24 * time.Hour, WorkStartedAfter: 10 * time.Hour, ImplementedAfter: 74 * time.Hour, VerifiedAfter: 90 * time.Hour, ClosedAfter: 96 * time.Hour, DueAfter: 80 * time.Hour, Reopen: true},
		{Key: "control-gap-logging", Title: "Close privileged logging control gap", Summary: "Implement the missing privileged-session logging and confirm retained event coverage.", Type: continuity.MatterControlGap, Priority: 4, OwnerID: owners[0], ActionOwnerID: config.ContributorPrincipalID, OpenedBefore: 65 * 24 * time.Hour, WorkStartedAfter: 12 * time.Hour, ImplementedAfter: 60 * time.Hour, VerifiedAfter: 78 * time.Hour, ClosedAfter: 84 * time.Hour, DueAfter: 120 * time.Hour},
		{Key: "audit-access-review", Title: "Complete quarterly access-review finding", Summary: "Complete the overdue account review and independently confirm the resolved population.", Type: continuity.MatterAuditFinding, Priority: 4, OwnerID: owners[1], ActionOwnerID: config.ContributorPrincipalID, OpenedBefore: 36 * 24 * time.Hour, WorkStartedAfter: 10 * time.Hour, ImplementedAfter: 46 * time.Hour, VerifiedAfter: 66 * time.Hour, ClosedAfter: 72 * time.Hour, DueAfter: 60 * time.Hour},
	}
}

func (s *Service) ensureOversightHistory(ctx context.Context, config SeedConfig) error {
	if s.referenceTimeline == nil {
		return fmt.Errorf("reference oversight history requires a configured timeline factory")
	}
	for _, spec := range referenceOversightHistories(config) {
		if err := s.ensureReferenceMatterHistory(ctx, config, spec); err != nil {
			return fmt.Errorf("install reference oversight history %s: %w", spec.Key, err)
		}
	}
	return nil
}

func (s *Service) ensureReferenceMatterHistory(ctx context.Context, config SeedConfig, spec referenceMatterHistory) error {
	triggerKey := "reference:oversight:" + spec.Key
	existing, err := s.continuity.MatterByTriggerKey(ctx, config.TenantID, triggerKey)
	if err == nil {
		if !isExpectedReferenceHistory(existing, spec) {
			return fmt.Errorf("trigger key %s belongs to a non-reference or incompatible issue", triggerKey)
		}
		if reason := incompleteReferenceHistoryReason(existing, spec); reason == "" {
			return nil
		} else {
			return fmt.Errorf("reference issue %s has incomplete history and cannot yet be resumed safely: %s", spec.Key, reason)
		}
	}
	if !errors.Is(err, continuity.ErrNotFound) {
		return err
	}

	openedAt := config.Now.Add(-spec.OpenedBefore).UTC()
	at := func(offset time.Duration) (*continuity.Service, error) {
		service := s.referenceTimeline(openedAt.Add(offset).UTC())
		if service == nil {
			return nil, fmt.Errorf("timeline factory returned no continuity service")
		}
		return service, nil
	}
	call := func(offset time.Duration, operation func(*continuity.Service) (continuity.MatterAggregate, error)) (continuity.MatterAggregate, error) {
		service, serviceErr := at(offset)
		if serviceErr != nil {
			return continuity.MatterAggregate{}, serviceErr
		}
		return operation(service)
	}

	created, err := call(0, func(service *continuity.Service) (continuity.MatterAggregate, error) {
		due := openedAt.Add(spec.DueAfter)
		return service.CreateMatter(ctx, continuity.CreateMatterInput{
			TenantID: config.TenantID, LegalEntityID: config.LegalEntityID, Type: spec.Type, Priority: spec.Priority,
			Title: spec.Title, Summary: spec.Summary,
			Scope:       mustJSON(map[string]any{"sample": true, "reference_data": true, "journey_code": referenceOversightJourneyCode, "history_key": spec.Key}),
			TriggerType: "REFERENCE_HISTORY", TriggerKey: triggerKey,
			KnownFacts:   mustJSON(map[string]any{"source": "ClearSight reference history", "as_of": config.Now.Format("2006-01-02")}),
			MissingFacts: mustJSON([]string{}), Contradictions: mustJSON([]string{}),
			OwnerPrincipalID: spec.OwnerID, RequiredAuthority: "REVIEWER", DueAt: &due, ActorID: config.ActorID,
		})
	})
	if err != nil {
		return err
	}
	matter := created
	transition := func(offset time.Duration, target continuity.MatterStatus, rationale string, actor string) error {
		updated, updateErr := call(offset, func(service *continuity.Service) (continuity.MatterAggregate, error) {
			return service.TransitionMatter(ctx, continuity.TransitionInput{TenantID: config.TenantID, ID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, To: target, ActorID: actor, Rationale: rationale})
		})
		if updateErr == nil {
			matter = updated
		}
		return updateErr
	}
	if err = transition(time.Hour, continuity.MatterAssessment, "The reference issue has sufficient scope for assessment.", config.ActorID); err != nil {
		return err
	}
	if spec.ReassignTo != "" {
		matter, err = call(2*time.Hour, func(service *continuity.Service) (continuity.MatterAggregate, error) {
			return service.AssignMatter(ctx, continuity.AssignMatterInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, OwnerPrincipalID: spec.ReassignTo, ActorID: config.ActorID, Rationale: "Transfer this review to the current accountable owner.", ReassignmentBasis: "Planned reference-data absence handoff"})
		})
		if err != nil {
			return err
		}
	}

	decisionAt := 90 * time.Minute
	decisionSteps := []struct {
		status continuity.DecisionStatus
		actor  string
	}{
		{continuity.DecisionProposed, spec.ActionOwnerID},
		{continuity.DecisionInReview, config.ReviewerPrincipalID},
	}
	if spec.ReturnDecision {
		decisionSteps = append(decisionSteps,
			struct {
				status continuity.DecisionStatus
				actor  string
			}{continuity.DecisionReturned, config.ReviewerPrincipalID},
			struct {
				status continuity.DecisionStatus
				actor  string
			}{continuity.DecisionProposed, spec.ActionOwnerID},
			struct {
				status continuity.DecisionStatus
				actor  string
			}{continuity.DecisionInReview, config.ReviewerPrincipalID},
		)
	}
	decisionSteps = append(decisionSteps, struct {
		status continuity.DecisionStatus
		actor  string
	}{continuity.DecisionApproved, config.SignatoryPrincipalID})
	for index, step := range decisionSteps {
		offset := decisionAt + time.Duration(index)*10*time.Minute
		matter, err = call(offset, func(service *continuity.Service) (continuity.MatterAggregate, error) {
			selected := ""
			if step.status == continuity.DecisionApproved {
				selected = "PROCEED_WITH_RECORDED_ACTION"
			}
			return service.RecordDecisionLifecycle(ctx, continuity.AddDecisionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Type: "REFERENCE_HANDLING", Status: step.status, Options: json.RawMessage(`["PROCEED_WITH_RECORDED_ACTION"]`), SelectedOption: selected, Rationale: "Record the governed handling decision for this reference issue.", Conditions: json.RawMessage(`[]`), AuthorityPrincipalID: step.actor})
		})
		if err != nil {
			return err
		}
	}

	if err = transition(spec.WorkStartedAfter, continuity.MatterActionsInProgress, "The approved work can now be implemented.", config.ActorID); err != nil {
		return err
	}
	matter, err = call(spec.WorkStartedAfter+10*time.Minute, func(service *continuity.Service) (continuity.MatterAggregate, error) {
		due := openedAt.Add(spec.DueAfter)
		return service.AddAction(ctx, continuity.AddActionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Title: "Complete " + strings.ToLower(spec.Title), Description: "Complete the scoped work and retain the evidence required for independent confirmation.", OwnerPrincipalID: spec.ActionOwnerID, DueAt: &due, ActorID: config.ActorID, OriginKey: "reference:oversight:action:" + spec.Key})
	})
	if err != nil {
		return err
	}
	action := matter.Actions[len(matter.Actions)-1]
	matter, err = call(spec.WorkStartedAfter+20*time.Minute, func(service *continuity.Service) (continuity.MatterAggregate, error) {
		return service.TransitionAction(ctx, continuity.TransitionActionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version, To: continuity.ActionInProgress, ActorID: spec.ActionOwnerID})
	})
	if err != nil {
		return err
	}
	matter, err = call(spec.WorkStartedAfter+30*time.Minute, func(service *continuity.Service) (continuity.MatterAggregate, error) {
		return service.AddVerificationContract(ctx, continuity.AddVerificationContractInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ActionID: action.ID, ExpectedOutcome: "The recorded work achieved the scoped control outcome.", Baseline: json.RawMessage(`{"state":"open"}`), Scope: json.RawMessage(`{"reference_data":true}`), Threshold: json.RawMessage(`{"result":"passed"}`), AuthorityPrincipalID: config.ReviewerPrincipalID, FailureResponse: "BLOCK_CLOSE", ActorID: config.ActorID})
	})
	if err != nil {
		return err
	}
	contract := matter.VerificationContracts[len(matter.VerificationContracts)-1]
	matter, err = call(spec.ImplementedAfter, func(service *continuity.Service) (continuity.MatterAggregate, error) {
		return service.TransitionAction(ctx, continuity.TransitionActionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version, To: continuity.ActionImplemented, ActorID: spec.ActionOwnerID})
	})
	if err != nil {
		return err
	}
	if err = transition(spec.ImplementedAfter+time.Minute, continuity.MatterVerification, "The implemented work is ready for independent outcome confirmation.", config.ActorID); err != nil {
		return err
	}
	recordPass := func(offset time.Duration) error {
		observedAt := openedAt.Add(offset)
		updated, resultErr := call(offset, func(service *continuity.Service) (continuity.MatterAggregate, error) {
			return service.RecordVerificationResult(ctx, continuity.RecordVerificationResultInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, ContractID: contract.ID, Result: continuity.VerificationPassed, Observations: json.RawMessage(`{"outcome":"confirmed"}`), EvidenceReferences: json.RawMessage(`[]`), ReviewerPrincipalID: config.ReviewerPrincipalID, ReviewerAuthorityPrincipalID: config.ReviewerPrincipalID, Rationale: "Independent review confirmed the recorded outcome.", ObservedAt: observedAt})
		})
		if resultErr == nil {
			matter = updated
		}
		return resultErr
	}
	if err = recordPass(spec.VerifiedAfter); err != nil {
		return err
	}
	if !spec.Reopen {
		return transition(spec.ClosedAfter, continuity.MatterClosed, "The action is implemented and the independent outcome check passed.", config.SignatoryPrincipalID)
	}
	firstClose := spec.ClosedAfter - 6*time.Hour
	if err = transition(firstClose, continuity.MatterClosed, "The first recorded outcome check passed.", config.SignatoryPrincipalID); err != nil {
		return err
	}
	if err = transition(spec.ClosedAfter-5*time.Hour, continuity.MatterAssessment, "New information requires the closed review to be reassessed.", config.ReviewerPrincipalID); err != nil {
		return err
	}
	if err = transition(spec.ClosedAfter-4*time.Hour, continuity.MatterVerification, "The reassessed scope requires a current outcome check.", config.ReviewerPrincipalID); err != nil {
		return err
	}
	if err = recordPass(spec.ClosedAfter - time.Hour); err != nil {
		return err
	}
	return transition(spec.ClosedAfter, continuity.MatterClosed, "The reopened review has a new independent passing outcome check.", config.SignatoryPrincipalID)
}

func isExpectedReferenceHistory(aggregate continuity.MatterAggregate, spec referenceMatterHistory) bool {
	if aggregate.Matter.Type != spec.Type || aggregate.Matter.Title != spec.Title {
		return false
	}
	var scope map[string]any
	if json.Unmarshal(aggregate.Matter.Scope, &scope) != nil {
		return false
	}
	return scope["sample"] == true && scope["reference_data"] == true && scope["journey_code"] == referenceOversightJourneyCode && scope["history_key"] == spec.Key
}

func incompleteReferenceHistoryReason(aggregate continuity.MatterAggregate, spec referenceMatterHistory) string {
	if aggregate.Matter.Status != continuity.MatterClosed || aggregate.Matter.ClosedAt == nil {
		return "issue is not closed"
	}
	wantReopenCount := 0
	if spec.Reopen {
		wantReopenCount = 1
	}
	if aggregate.Matter.ReopenCount != wantReopenCount {
		return fmt.Sprintf("reopen count is %d; expected %d", aggregate.Matter.ReopenCount, wantReopenCount)
	}
	wantOwner := spec.OwnerID
	if spec.ReassignTo != "" {
		wantOwner = spec.ReassignTo
	}
	if aggregate.Matter.OwnerPrincipalID != wantOwner {
		return "current accountable owner does not match the reference handoff"
	}
	approvedDecision := false
	for _, decision := range aggregate.Decisions {
		if decision.Type == "REFERENCE_HANDLING" && decision.Status == continuity.DecisionApproved && decision.SelectedOption == "PROCEED_WITH_RECORDED_ACTION" {
			approvedDecision = true
			break
		}
	}
	if !approvedDecision {
		return "approved handling decision is missing"
	}
	actionOrigin := "reference:oversight:action:" + spec.Key
	for _, action := range aggregate.Actions {
		if action.OriginKey != actionOrigin {
			continue
		}
		if action.Status != continuity.ActionImplemented || action.ImplementedAt == nil {
			return "reference action is not implemented"
		}
		for _, contract := range aggregate.VerificationContracts {
			if contract.ActionID != action.ID || contract.Status != continuity.VerificationActive {
				continue
			}
			for _, result := range aggregate.VerificationResults {
				if result.ContractID == contract.ID && result.Result == continuity.VerificationPassed && result.ReviewerPrincipalID != action.OwnerPrincipalID {
					return ""
				}
			}
			return "independent passing outcome check is missing"
		}
		return "active outcome-check contract is missing"
	}
	return "reference action is missing"
}
