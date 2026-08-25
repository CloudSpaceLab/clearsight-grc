package bankverticals

import (
	"context"
	"errors"
	"fmt"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func (s *Service) ensureAuthorityRequest(ctx context.Context, config SeedConfig, program continuity.ProgramAggregate, sourceID string) error {
	matter, err := s.continuity.MatterByTriggerKey(ctx, config.TenantID, triggerAuthorityRequest)
	if errors.Is(err, continuity.ErrNotFound) {
		_, err = s.seedAuthorityRequest(ctx, config, program, sourceID)
		return err
	}
	if err != nil {
		return err
	}
	if !referenceMatter(matter.Matter, JourneyAuthorityRequest) || !restrictedPolicyComplete(matter.Matter) {
		return fmt.Errorf("authority-request reference issue has invalid provenance or restricted-access metadata")
	}
	if matter.Matter.Status == continuity.MatterCancelled {
		return fmt.Errorf("authority-request reference issue is cancelled and cannot be repaired")
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterAssessment, "The authority request and restricted handling group were reconciled.")
	if err != nil {
		return err
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterResponsePreparation, "The response package and signatory approval are being prepared.")
	if err != nil {
		return err
	}
	request, requestErr := s.evidence.LatestRequestForSubject(ctx, config.TenantID, "MATTER", matter.Matter.ID)
	if errors.Is(requestErr, evidence.ErrNotFound) {
		request, requestErr = s.evidence.CreateRequest(ctx, authorityEvidenceRequest(config, matter.Matter.ID))
	}
	if requestErr != nil {
		return requestErr
	}
	if requestIsActionable(request) {
		_, err = s.evidence.Submit(ctx, evidence.Submission{
			TenantID:        config.TenantID,
			RequestID:       request.ID,
			SubmittedBy:     config.OwnerPrincipalID,
			Channel:         "INTERNAL",
			Answers:         formcontract.TextAnswers(map[string]string{"containment_record": "Incident containment record PRI-2026-008", "communication_decision": "No direct customer notice was approved after the documented impact assessment."}),
			ExpectedVersion: request.Version,
		})
		if err != nil {
			return err
		}
	}
	response := currentResponse(matter.ResponsePackages)
	if response == nil || response.Status == continuity.ResponseRejected || response.Status == continuity.ResponseWithdrawn {
		matter, err = s.continuity.AddResponsePackage(ctx, continuity.AddResponsePackageInput{
			TenantID:        config.TenantID,
			MatterID:        matter.Matter.ID,
			ExpectedVersion: matter.Matter.Version,
			Purpose:         "Respond to NDPC request NDPC/ENF/2026/0142",
			Audience:        "Nigeria Data Protection Commission",
			Manifest:        mustJSON([]map[string]any{{"classification": "RESTRICTED", "evidence_request_id": request.ID}, {"document": "incident assessment"}, {"document": "containment record"}, {"document": "notification decision"}, {"document": "customer communication decision"}}),
			ActorID:         config.ActorID,
		})
		if err != nil {
			return err
		}
		response = currentResponse(matter.ResponsePackages)
	}
	for _, target := range []continuity.ResponseStatus{continuity.ResponseInReview, continuity.ResponseApproved, continuity.ResponseTransmitted, continuity.ResponseAcknowledged} {
		if responseRank(response.Status) >= responseRank(target) {
			continue
		}
		actorID := config.ActorID
		if target == continuity.ResponseApproved {
			actorID = config.SignatoryPrincipalID
		}
		matter, err = s.continuity.TransitionResponsePackage(ctx, continuity.TransitionResponseInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ResponseID: response.ID, ExpectedVersion: matter.Matter.Version, To: target, ActorID: actorID, Rationale: responseRationale(target)})
		if err != nil {
			return err
		}
		response = currentResponse(matter.ResponsePackages)
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterVerification, "The response was transmitted and acknowledgement was recorded.")
	if err != nil {
		return err
	}
	_, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterClosed, "The authority acknowledged receipt of the approved response package.")
	return err
}
