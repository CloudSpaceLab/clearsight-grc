package bankverticals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

type Service struct {
	continuity *continuity.Service
	evidence   *evidence.Service
}

func NewService(continuityService *continuity.Service, evidenceService *evidence.Service) *Service {
	return &Service{continuity: continuityService, evidence: evidenceService}
}

func (s *Service) List(ctx context.Context, tenant string) ([]Journey, error) {
	if s == nil || s.continuity == nil || s.evidence == nil {
		return nil, fmt.Errorf("bank journeys are unavailable")
	}
	if strings.TrimSpace(tenant) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}

	ndpa := definition(JourneyNDPAContinuous)
	program, err := s.continuity.ProgramByCode(ctx, tenant, programCodeNDPA)
	if err == nil {
		request, requestFound, requestErr := s.latestRequest(ctx, tenant, "PROGRAM", program.Program.ID)
		if requestErr != nil {
			return nil, requestErr
		}
		sources, sourceErr := s.evidence.SourcesByCodes(ctx, tenant, ndpaSourceCodes)
		if sourceErr != nil {
			return nil, sourceErr
		}
		ndpa = buildNDPAJourney(program, request, requestFound, sources)
	} else if !errors.Is(err, continuity.ErrNotFound) {
		return nil, err
	}

	journeys := []Journey{ndpa}
	for _, item := range []struct {
		code       Code
		triggerKey string
	}{
		{JourneyRegulatoryChange, triggerRegulatoryChange},
		{JourneyAuthorityRequest, triggerAuthorityRequest},
		{JourneyFindingRemediation, triggerFindingRemediation},
	} {
		journey := definition(item.code)
		matter, matterErr := s.continuity.MatterByTriggerKey(ctx, tenant, item.triggerKey)
		if matterErr == nil {
			request, requestFound, requestErr := s.latestRequest(ctx, tenant, "MATTER", matter.Matter.ID)
			if requestErr != nil {
				return nil, requestErr
			}
			codes := []string{}
			switch item.code {
			case JourneyRegulatoryChange:
				codes = []string{"NDPA-GAID-2025"}
			case JourneyAuthorityRequest:
				codes = []string{"NDPC-REQUEST-2026"}
			case JourneyFindingRemediation:
				codes = []string{"INTERNAL-AUDIT-2024"}
			}
			sources, sourceErr := s.evidence.SourcesByCodes(ctx, tenant, codes)
			if sourceErr != nil {
				return nil, sourceErr
			}
			journey = buildMatterJourney(item.code, matter, request, requestFound, sources)
		} else if !errors.Is(matterErr, continuity.ErrNotFound) {
			return nil, matterErr
		}
		journeys = append(journeys, journey)
	}
	return journeys, nil
}

func (s *Service) latestRequest(ctx context.Context, tenant, subjectType, subjectID string) (evidence.Request, bool, error) {
	request, err := s.evidence.LatestRequestForSubject(ctx, tenant, subjectType, subjectID)
	if errors.Is(err, evidence.ErrNotFound) {
		return evidence.Request{}, false, nil
	}
	if err != nil {
		return evidence.Request{}, false, err
	}
	return request, true, nil
}

func buildNDPAJourney(program continuity.ProgramAggregate, request evidence.Request, requestFound bool, sources []evidence.Source) Journey {
	journey := definition(JourneyNDPAContinuous)
	journey.Sample = scopeString(program.Program.Scope, "sample") == "true"
	journey.ProgramID = program.Program.ID
	journey.Owner = program.Program.OwningFunction
	journey.Status = string(program.Program.Status)
	journey.StatusLabel = program.StateLabel
	journey.NextAction = "Review the latest evidence and open issues"
	updated := program.Program.UpdatedAt
	journey.UpdatedAt = &updated
	if program.CurrentState == nil {
		journey.Status = "STATUS_PENDING"
		journey.StatusLabel = "Status update pending"
		journey.NextAction = "Wait for the current status check or run a governed rebuild"
	} else if len(program.CurrentState.Reasons) > 0 {
		journey.NextAction = program.CurrentState.Reasons[0].Summary
	}
	if requestFound {
		journey.EvidenceRequestID = request.ID
		journey.DueAt = &request.Deadline
		if request.Status != evidence.RequestSubmitted && request.Status != evidence.RequestCancelled {
			journey.NextAction = request.Title
		}
	}
	journey.SourceNames = sourceNames(sources)
	journey.Steps = []Step{
		{Code: "sources", Label: "Authoritative sources registered", Complete: len(journey.SourceNames) >= 4},
		{Code: "requirements", Label: "Bank obligations recorded", Complete: len(program.Requirements) >= 5},
		{Code: "safeguards", Label: "Safeguards linked to obligations", Complete: len(program.ControlImplementations) >= 5 && len(program.RequirementControlLinks) >= 5},
		{Code: "evidence", Label: "Evidence checks defined", Complete: len(program.EvidenceContracts) >= 5},
		{Code: "review", Label: "Current evidence reviewed", Complete: len(program.EvidenceAssessments) >= 5},
		{Code: "active", Label: "Program approved and active", Complete: program.Program.Status == continuity.ProgramActive},
	}
	finalize(&journey)
	return journey
}

func buildMatterJourney(code Code, matter continuity.MatterAggregate, request evidence.Request, requestFound bool, sources []evidence.Source) Journey {
	journey := definition(code)
	journey.Sample = scopeString(matter.Matter.Scope, "sample") == "true"
	journey.MatterID = matter.Matter.ID
	journey.Owner = ownerForMatter(code)
	journey.Status = string(matter.Matter.Status)
	journey.StatusLabel = matter.StatusLabel
	journey.NextAction = matter.NextAction
	journey.DueAt = matter.Matter.DueAt
	journey.AllowedPrincipalIDs = scopeStrings(matter.Matter.Scope, "allowed_principal_ids")
	updated := matter.Matter.UpdatedAt
	journey.UpdatedAt = &updated
	if requestFound {
		journey.EvidenceRequestID = request.ID
		if journey.DueAt == nil || request.Deadline.Before(*journey.DueAt) {
			journey.DueAt = &request.Deadline
		}
	}
	journey.SourceNames = sourceNames(sources)

	switch code {
	case JourneyRegulatoryChange:
		journey.Steps = regulatorySteps(matter)
	case JourneyAuthorityRequest:
		journey.Sensitive = true
		journey.Steps = authoritySteps(matter)
	case JourneyFindingRemediation:
		journey.Steps = findingSteps(matter)
	}
	finalize(&journey)
	return journey
}

func regulatorySteps(m continuity.MatterAggregate) []Step {
	return []Step{
		{Code: "source", Label: "Official change recorded", Complete: m.Matter.SourceID != ""},
		{Code: "assessment", Label: "Affected obligations assessed", Complete: statusAtLeast(m.Matter.Status, continuity.MatterAssessment)},
		{Code: "decision", Label: "Bank position approved", Complete: hasApprovedDecision(m.Decisions)},
		{Code: "action", Label: "Required change assigned", Complete: len(m.Actions) > 0},
		{Code: "outcome", Label: "Outcome confirmed", Complete: latestVerificationPassed(m)},
	}
}

func authoritySteps(m continuity.MatterAggregate) []Step {
	return []Step{
		{Code: "received", Label: "Request recorded with restricted access", Complete: m.Matter.ID != ""},
		{Code: "prepared", Label: "Response package prepared", Complete: len(m.ResponsePackages) > 0},
		{Code: "approved", Label: "Response approved by the signatory", Complete: responseAtLeast(m, continuity.ResponseApproved)},
		{Code: "sent", Label: "Response transmitted", Complete: responseAtLeast(m, continuity.ResponseTransmitted)},
		{Code: "acknowledged", Label: "Authority acknowledgement recorded", Complete: responseAtLeast(m, continuity.ResponseAcknowledged) && m.Matter.Status == continuity.MatterClosed},
	}
}

func findingSteps(m continuity.MatterAggregate) []Step {
	return []Step{
		{Code: "recorded", Label: "Finding and affected scope recorded", Complete: m.Matter.ID != ""},
		{Code: "assigned", Label: "Remediation assigned", Complete: len(m.Actions) > 0},
		{Code: "implemented", Label: "Remediation implemented", Complete: hasImplementedAction(m.Actions)},
		{Code: "checked", Label: "Outcome checked independently", Complete: latestVerificationPassed(m)},
		{Code: "closed", Label: "Finding closed with evidence", Complete: m.Matter.Status == continuity.MatterClosed && m.Closure.Ready},
	}
}

func definition(code Code) Journey {
	journey := Journey{Code: code, Status: "NOT_SET_UP", StatusLabel: "Not set up", NextAction: "Set up this journey", SourceNames: []string{}, Steps: []Step{}}
	switch code {
	case JourneyNDPAContinuous:
		journey.Title = "Nigeria data protection"
		journey.Summary = "Keep the bank's NDP Act and GAID obligations, safeguards, evidence and open gaps current."
		journey.Owner = "Data Protection Office"
	case JourneyRegulatoryChange:
		journey.Title = "Regulatory change"
		journey.Summary = "Move an official change from source review to an approved bank position, assigned action and confirmed outcome."
		journey.Owner = "Regulatory Compliance"
	case JourneyAuthorityRequest:
		journey.Title = "Regulator request"
		journey.Summary = "Prepare, approve, send and track a protected response without exposing the request beyond its approved team."
		journey.Owner = "Regulatory Affairs"
		journey.Sensitive = true
	case JourneyFindingRemediation:
		journey.Title = "Finding remediation"
		journey.Summary = "Take an audit finding or exception through remediation, independent checking and evidence-based closure."
		journey.Owner = "Control Assurance"
	}
	return journey
}

func finalize(journey *Journey) {
	journey.TotalSteps = len(journey.Steps)
	for _, step := range journey.Steps {
		if step.Complete {
			journey.CompletedSteps++
		}
	}
	if journey.StatusLabel == "" {
		journey.StatusLabel = "Status unavailable"
	}
	if journey.NextAction == "" {
		journey.NextAction = "Review the current record"
	}
}

func sourceNames(sources []evidence.Source) []string {
	values := make([]string, 0, len(sources))
	for _, source := range sources {
		values = append(values, source.Name)
	}
	sort.Strings(values)
	return values
}

func scopeString(raw json.RawMessage, key string) string {
	var values map[string]any
	if json.Unmarshal(raw, &values) != nil {
		return ""
	}
	value, ok := values[key]
	if !ok {
		return ""
	}
	return fmt.Sprint(value)
}

func scopeStrings(raw json.RawMessage, key string) []string {
	var values map[string]any
	if json.Unmarshal(raw, &values) != nil {
		return nil
	}
	rawValues, ok := values[key].([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(rawValues))
	for _, value := range rawValues {
		if text := strings.TrimSpace(fmt.Sprint(value)); text != "" {
			result = append(result, text)
		}
	}
	return result
}

func statusAtLeast(status, expected continuity.MatterStatus) bool {
	rank := map[continuity.MatterStatus]int{
		continuity.MatterDraft: 0, continuity.MatterInitialReview: 1, continuity.MatterAssessment: 2,
		continuity.MatterDecisionRequired: 3, continuity.MatterActionsInProgress: 4,
		continuity.MatterResponsePreparation: 4, continuity.MatterVerification: 5,
		continuity.MatterClosed: 6, continuity.MatterCancelled: -1,
	}
	return rank[status] >= rank[expected]
}

func hasApprovedDecision(values []continuity.Decision) bool {
	for _, value := range values {
		if value.Status == continuity.DecisionApproved || value.Status == continuity.DecisionConditionallyApproved {
			return true
		}
	}
	return false
}

func hasImplementedAction(values []continuity.Action) bool {
	for _, value := range values {
		if value.Status == continuity.ActionImplemented {
			return true
		}
	}
	return false
}

func latestVerificationPassed(m continuity.MatterAggregate) bool {
	if len(m.VerificationContracts) == 0 {
		return false
	}
	latest := map[string]continuity.VerificationResult{}
	for _, result := range m.VerificationResults {
		current, ok := latest[result.ContractID]
		if !ok || result.ObservedAt.After(current.ObservedAt) {
			latest[result.ContractID] = result
		}
	}
	for _, contract := range m.VerificationContracts {
		if contract.Status != continuity.VerificationActive {
			continue
		}
		result, ok := latest[contract.ID]
		if !ok || result.Result != continuity.VerificationPassed {
			return false
		}
	}
	return true
}

func responseAtLeast(m continuity.MatterAggregate, expected continuity.ResponseStatus) bool {
	rank := map[continuity.ResponseStatus]int{
		continuity.ResponseDraft: 0, continuity.ResponseInReview: 1, continuity.ResponseApproved: 2,
		continuity.ResponseTransmitted: 3, continuity.ResponseAcknowledged: 4,
		continuity.ResponseRejected: -1, continuity.ResponseWithdrawn: -1,
	}
	for _, response := range m.ResponsePackages {
		if rank[response.Status] >= rank[expected] {
			return true
		}
	}
	return false
}

func ownerForMatter(code Code) string {
	switch code {
	case JourneyRegulatoryChange:
		return "Regulatory Compliance"
	case JourneyAuthorityRequest:
		return "Regulatory Affairs"
	case JourneyFindingRemediation:
		return "Control Assurance"
	default:
		return ""
	}
}

func timePointer(value time.Time) *time.Time { return &value }
