package continuity

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

// VerificationResultBundle is the narrow atomic contract for a failed outcome
// check whose governed failure response changes authoritative state. It avoids
// a generic transaction coordinator while ensuring the result and its required
// consequence commit together.
type VerificationResultBundle struct {
	TenantID          string
	MatterID          string
	ExpectedVersion   int64
	ResultEvent       Event
	TransitionEvent   *Event
	FollowUpMatter    *Matter
	FollowUpEvent     *Event
	FollowUpLink      *MatterLink
	FollowUpLinkEvent *Event
}

type VerificationResultBundleRepository interface {
	ApplyVerificationResultBundle(context.Context, VerificationResultBundle) error
}

func (s *Service) recordVerificationResult(ctx context.Context, input RecordVerificationResultInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	if strings.TrimSpace(input.ContractID) == "" || !validVerificationResult(input.Result) || strings.TrimSpace(input.Rationale) == "" {
		return MatterAggregate{}, fmt.Errorf("contract_id, supported result and rationale are required")
	}
	contract, ok := verificationContractByID(aggregate.VerificationContracts, input.ContractID)
	if !ok {
		return MatterAggregate{}, fmt.Errorf("contract_id does not belong to this matter")
	}
	observations, err := normalizedJSON(input.Observations, `{}`)
	if err != nil {
		return MatterAggregate{}, err
	}
	evidenceReferences, err := normalizedJSON(input.EvidenceReferences, `[]`)
	if err != nil {
		return MatterAggregate{}, err
	}
	if input.ObservedAt.IsZero() {
		input.ObservedAt = s.now().UTC()
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return MatterAggregate{}, err
	}
	now := s.now().UTC()
	value := VerificationResult{
		ID: valueID, TenantID: input.TenantID, MatterID: input.MatterID,
		ContractID: input.ContractID, Result: input.Result, Observations: observations,
		EvidenceReferences: evidenceReferences, ReviewerPrincipalID: input.ReviewerPrincipalID,
		Rationale: strings.TrimSpace(input.Rationale), ObservedAt: input.ObservedAt.UTC(), CreatedAt: now,
	}
	resultEvent, err := newEvent(input.TenantID, "MATTER", input.MatterID, input.ExpectedVersion+1, EventVerificationResultRecorded, value, actorFor(input.ReviewerPrincipalID), input.ReviewerPrincipalID, now)
	if err != nil {
		return MatterAggregate{}, err
	}

	// Passing, inconclusive, or BLOCK_CLOSE results are one-event commands and
	// already use the repository's atomic event/outbox boundary.
	if input.Result != VerificationFailed || aggregate.Matter.Status != MatterVerification || contract.FailureResponse == "BLOCK_CLOSE" {
		if _, err = s.repo.ApplyMatterEvent(ctx, input.TenantID, input.MatterID, input.ExpectedVersion, resultEvent); err != nil {
			return MatterAggregate{}, err
		}
		return s.GetMatter(ctx, input.TenantID, input.MatterID)
	}

	bundleRepo, ok := s.repo.(VerificationResultBundleRepository)
	if !ok {
		return MatterAggregate{}, fmt.Errorf("atomic verification failure handling is not supported by this repository")
	}
	bundle := VerificationResultBundle{TenantID: input.TenantID, MatterID: input.MatterID, ExpectedVersion: input.ExpectedVersion, ResultEvent: resultEvent}

	switch contract.FailureResponse {
	case "REOPEN", "ESCALATE":
		matter := aggregate.Matter
		if contract.FailureResponse == "REOPEN" {
			matter.Status = MatterActionsInProgress
		} else {
			matter.Status = MatterDecisionRequired
		}
		matter.UpdatedAt = now
		transitionEvent, eventErr := newEvent(input.TenantID, "MATTER", input.MatterID, input.ExpectedVersion+2, EventMatterStateChanged, matter, actorFor(input.ReviewerPrincipalID), input.ReviewerPrincipalID, now)
		if eventErr != nil {
			return MatterAggregate{}, eventErr
		}
		bundle.TransitionEvent = &transitionEvent
	case "CREATE_MATTER":
		programIDs, lookupErr := s.repo.LinkedProgramIDs(ctx, input.TenantID, input.MatterID)
		if lookupErr != nil {
			return MatterAggregate{}, lookupErr
		}
		sort.Strings(programIDs)
		programID := ""
		if len(programIDs) > 0 {
			programID = programIDs[0]
		}
		followID, idErr := id.NewUUIDv7()
		if idErr != nil {
			return MatterAggregate{}, idErr
		}
		follow := Matter{
			ID: followID, TenantID: input.TenantID, Reference: matterReference(followID),
			Type: MatterFailedVerification, Status: MatterInitialReview, Priority: maxInt(aggregate.Matter.Priority, 3),
			Title: "Resolve a failed outcome check", Summary: "The expected outcome was not observed and needs separate follow-up.",
			Scope: aggregate.Matter.Scope, SourceType: "MATTER", SourceID: aggregate.Matter.ID,
			TriggerType: "VERIFICATION_FAILED", TriggerID: value.ID, TriggerKey: "verification:" + input.ContractID + ":" + value.ID,
			KnownFacts: observations, MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`),
			CreatedAt: now, UpdatedAt: now, Version: 1,
		}
		followEvent, eventErr := newEvent(input.TenantID, "MATTER", follow.ID, 1, EventMatterCreated, follow, actorFor(input.ReviewerPrincipalID), input.ReviewerPrincipalID, now)
		if eventErr != nil {
			return MatterAggregate{}, eventErr
		}
		bundle.FollowUpMatter = &follow
		bundle.FollowUpEvent = &followEvent
		if programID != "" {
			linkID, linkErr := id.NewUUIDv7()
			if linkErr != nil {
				return MatterAggregate{}, linkErr
			}
			link := MatterLink{ID: linkID, TenantID: input.TenantID, MatterID: follow.ID, ProgramID: programID, Relationship: "AFFECTS", CreatedAt: now}
			linkEvent, eventErr := newEvent(input.TenantID, "MATTER", follow.ID, 2, EventMatterLinked, link, actorFor(input.ReviewerPrincipalID), input.ReviewerPrincipalID, now)
			if eventErr != nil {
				return MatterAggregate{}, eventErr
			}
			follow.Version = 2
			bundle.FollowUpMatter = &follow
			bundle.FollowUpLink = &link
			bundle.FollowUpLinkEvent = &linkEvent
		}
	default:
		return MatterAggregate{}, fmt.Errorf("unsupported verification failure response %q", contract.FailureResponse)
	}

	if err := bundleRepo.ApplyVerificationResultBundle(ctx, bundle); err != nil {
		return MatterAggregate{}, err
	}
	return s.GetMatter(ctx, input.TenantID, input.MatterID)
}
