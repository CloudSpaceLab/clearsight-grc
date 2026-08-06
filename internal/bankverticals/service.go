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

var (
	ndpaRequirementCodes = []string{
		"PROCESSING-ACCOUNTABILITY",
		"DATA-SUBJECT-RIGHTS",
		"PRIVACY-INCIDENTS",
		"DPIA-HIGH-RISK",
		"CAR-ANNUAL",
	}
	ndpaEvidenceCodes = []string{
		"PROCESSING-COVERAGE",
		"RIGHTS-REGISTER",
		"INCIDENT-COVERAGE",
		"DPIA-COVERAGE",
		"CAR-EVIDENCE",
	}
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
		request, requestFound, requestErr := s.currentRequest(ctx, tenant, "PROGRAM", program.Program.ID)
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
			request, requestFound, requestErr := s.currentRequest(ctx, tenant, "MATTER", matter.Matter.ID)
			if requestErr != nil {
				return nil, requestErr
			}
			codes := sourceCodesForJourney(item.code)
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

func sourceCodesForJourney(code Code) []string {
	switch code {
	case JourneyRegulatoryChange:
		return []string{"NDPA-GAID-2025"}
	case JourneyAuthorityRequest:
		return []string{"NDPC-REQUEST-2026"}
	case JourneyFindingRemediation:
		return []string{"INTERNAL-AUDIT-2024"}
	default:
		return nil
	}
}

func (s *Service) currentRequest(ctx context.Context, tenant, subjectType, subjectID string) (evidence.Request, bool, error) {
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
	journey.OwnerPrincipalID = program.Program.OwnerPrincipalID
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
		if requestIsActionable(request) {
			journey.DueAt = &request.Deadline
			journey.NextAction = request.Title
		}
	}
	journey.SourceNames = sourceNames(sources)
	journey.Steps = []Step{
		{Code: "sources", Label: "Official legal and regulatory sources registered", Complete: hasActiveSourceCodes(sources, "NDPA-ACT-2023", "NDPA-GAID-2025")},
		{Code: "requirements", Label: "Required bank obligations approved", Complete: hasApprovedRequirements(program, ndpaRequirementCodes)},
		{Code: "safeguards", Label: "Implemented safeguards linked to each obligation", Complete: hasRequiredSafeguards(program, ndpaRequirementCodes)},
		{Code: "evidence", Label: "Active evidence checks defined", Complete: hasActiveEvidenceContracts(program, ndpaEvidenceCodes)},
		{Code: "review", Label: "Current evidence reviewed", Complete: hasCurrentEvidenceAssessments(program, ndpaEvidenceCodes, time.Now().UTC())},
		{Code: "active", Label: "Program approved and active", Complete: program.Program.Status == continuity.ProgramActive},
	}
	setJourneyAction(&journey, request, requestFound)
	finalize(&journey)
	return journey
}

func buildMatterJourney(code Code, matter continuity.MatterAggregate, request evidence.Request, requestFound bool, sources []evidence.Source) Journey {
	journey := definition(code)
	journey.Sample = scopeString(matter.Matter.Scope, "sample") == "true"
	journey.MatterID = matter.Matter.ID
	journey.Owner = ownerForMatter(code)
	journey.OwnerPrincipalID = matter.Matter.OwnerPrincipalID
	journey.Status = string(matter.Matter.Status)
	journey.StatusLabel = matter.StatusLabel
	journey.NextAction = matter.NextAction
	journey.DueAt = matter.Matter.DueAt
	journey.AllowedPrincipalIDs = scopeStrings(matter.Matter.Scope, "allowed_principal_ids")
	updated := matter.Matter.UpdatedAt
	journey.UpdatedAt = &updated
	if requestFound {
		journey.EvidenceRequestID = request.ID
		if requestIsActionable(request) && (journey.DueAt == nil || request.Deadline.Before(*journey.DueAt)) {
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
	setJourneyAction(&journey, request, requestFound)
	finalize(&journey)
	return journey
}

func setJourneyAction(journey *Journey, request evidence.Request, requestFound bool) {
	if requestFound && requestIsActionable(request) {
		journey.ActionTargetType = ActionTargetEvidenceRequest
		journey.ActionTargetID = request.ID
		journey.ActionLabel = "Open evidence request"
		journey.ActionAvailable = true
		return
	}
	if journey.MatterID != "" {
		journey.ActionTargetType = ActionTargetMatter
		journey.ActionTargetID = journey.MatterID
		journey.ActionAvailable = true
		if journey.Status == string(continuity.MatterClosed) {
			journey.ActionLabel = "Review completed issue"
		} else {
			journey.ActionLabel = "Open issue and continue"
		}
		return
	}
	if journey.ProgramID != "" {
		journey.ActionTargetType = ActionTargetProgram
		journey.ActionTargetID = journey.ProgramID
		journey.ActionLabel = "Open program"
		journey.ActionAvailable = true
		return
	}
	journey.ActionUnavailableReason = "This journey has not been configured in the current scope."
}

func regulatorySteps(m continuity.MatterAggregate) []Step {
	return []Step{
		{Code: "source", Label: "Official change recorded", Complete: m.Matter.SourceID != ""},
		{Code: "assessment", Label: "Affected obligations assessed", Complete: statusAtLeast(m.Matter.Status, continuity.MatterAssessment)},
		{Code: "decision", Label: "Current bank position approved", Complete: currentDecisionApproved(m.Decisions)},
		{Code: "action", Label: "Required change assigned", Complete: len(currentActions(m.Actions)) > 0},
		{Code: "outcome", Label: "Outcome confirmed", Complete: latestVerificationPassed(m)},
	}
}

func authoritySteps(m continuity.MatterAggregate) []Step {
	return []Step{
		{Code: "received", Label: "Request recorded with valid restricted access", Complete: restrictedPolicyComplete(m.Matter)},
		{Code: "prepared", Label: "Current response package prepared", Complete: currentResponse(m.ResponsePackages) != nil},
		{Code: "approved", Label: "Current response approved by the signatory", Complete: responseAtLeast(m, continuity.ResponseApproved)},
		{Code: "sent", Label: "Current approved response transmitted", Complete: responseAtLeast(m, continuity.ResponseTransmitted)},
		{Code: "acknowledged", Label: "Authority acknowledgement recorded", Complete: responseAtLeast(m, continuity.ResponseAcknowledged) && m.Matter.Status == continuity.MatterClosed},
	}
}

func findingSteps(m continuity.MatterAggregate) []Step {
	actions := currentActions(m.Actions)
	return []Step{
		{Code: "recorded", Label: "Finding and affected scope recorded", Complete: m.Matter.ID != ""},
		{Code: "assigned", Label: "Current remediation assigned", Complete: len(actions) > 0},
		{Code: "implemented", Label: "Current remediation implemented", Complete: allActionsImplemented(actions)},
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
	journey.CompletedSteps = 0
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

func hasActiveSourceCodes(sources []evidence.Source, codes ...string) bool {
	found := make(map[string]bool, len(sources))
	for _, source := range sources {
		if source.Status == evidence.SourceActive && source.Type == evidence.SourceRegulatory {
			found[strings.ToUpper(source.Code)] = true
		}
	}
	for _, code := range codes {
		if !found[strings.ToUpper(code)] {
			return false
		}
	}
	return len(codes) > 0
}

func hasApprovedRequirements(program continuity.ProgramAggregate, codes []string) bool {
	found := map[string]bool{}
	for _, requirement := range program.Requirements {
		if requirement.Status == continuity.RequirementApproved {
			found[strings.ToUpper(requirement.Code)] = true
		}
	}
	for _, code := range codes {
		if !found[strings.ToUpper(code)] {
			return false
		}
	}
	return len(codes) > 0
}

func hasRequiredSafeguards(program continuity.ProgramAggregate, codes []string) bool {
	requirements := map[string]string{}
	for _, requirement := range program.Requirements {
		if requirement.Status == continuity.RequirementApproved {
			requirements[strings.ToUpper(requirement.Code)] = requirement.ID
		}
	}
	implemented := map[string]bool{}
	for _, implementation := range program.ControlImplementations {
		if implementation.Status == continuity.ImplementationImplemented {
			implemented[implementation.ID] = true
		}
	}
	linked := map[string]bool{}
	for _, link := range program.RequirementControlLinks {
		if implemented[link.ImplementationID] {
			linked[link.RequirementID] = true
		}
	}
	for _, code := range codes {
		id := requirements[strings.ToUpper(code)]
		if id == "" || !linked[id] {
			return false
		}
	}
	return len(codes) > 0
}

func activeEvidenceContracts(program continuity.ProgramAggregate, codes []string) map[string]continuity.EvidenceContract {
	wanted := map[string]bool{}
	for _, code := range codes {
		wanted[strings.ToUpper(code)] = true
	}
	contracts := map[string]continuity.EvidenceContract{}
	for _, contract := range program.EvidenceContracts {
		code := strings.ToUpper(contract.Code)
		if wanted[code] && contract.Status == continuity.EvidenceContractActive && contract.ControlImplementationID != "" {
			contracts[code] = contract
		}
	}
	return contracts
}

func hasActiveEvidenceContracts(program continuity.ProgramAggregate, codes []string) bool {
	return len(activeEvidenceContracts(program, codes)) == len(codes) && len(codes) > 0
}

func hasCurrentEvidenceAssessments(program continuity.ProgramAggregate, codes []string, now time.Time) bool {
	contracts := activeEvidenceContracts(program, codes)
	if len(contracts) != len(codes) || len(codes) == 0 {
		return false
	}
	latest := map[string]continuity.EvidenceAssessment{}
	for _, assessment := range program.EvidenceAssessments {
		current, exists := latest[assessment.ContractID]
		if !exists || assessment.AssessedAt.After(current.AssessedAt) {
			latest[assessment.ContractID] = assessment
		}
	}
	for _, contract := range contracts {
		assessment, exists := latest[contract.ID]
		if !exists || (assessment.ValidUntil != nil && !assessment.ValidUntil.After(now)) {
			return false
		}
	}
	return true
}

func requestIsActionable(request evidence.Request) bool {
	return request.Status == evidence.RequestDraft || request.Status == evidence.RequestReady || request.Status == evidence.RequestInProgress
}

func restrictedPolicyComplete(matter continuity.Matter) bool {
	policy, valid := continuity.ParseMatterAccessPolicy(matter.Scope)
	return valid && policy.Access == continuity.MatterAccessRestricted && len(policy.AllowedPrincipalIDs) > 0
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

func currentDecisionApproved(values []continuity.Decision) bool {
	var selected *continuity.Decision
	for index := range values {
		value := values[index]
		if selected == nil || value.UpdatedAt.After(selected.UpdatedAt) || (value.UpdatedAt.Equal(selected.UpdatedAt) && value.ID > selected.ID) {
			copy := value
			selected = &copy
		}
	}
	return selected != nil && (selected.Status == continuity.DecisionApproved || selected.Status == continuity.DecisionConditionallyApproved)
}

func currentActions(values []continuity.Action) []continuity.Action {
	result := make([]continuity.Action, 0, len(values))
	for _, value := range values {
		if value.Status != continuity.ActionCancelled {
			result = append(result, value)
		}
	}
	return result
}

func allActionsImplemented(values []continuity.Action) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if value.Status != continuity.ActionImplemented {
			return false
		}
	}
	return true
}

func latestVerificationPassed(m continuity.MatterAggregate) bool {
	active := make([]continuity.VerificationContract, 0, len(m.VerificationContracts))
	for _, contract := range m.VerificationContracts {
		if contract.Status == continuity.VerificationActive {
			active = append(active, contract)
		}
	}
	if len(active) == 0 {
		return false
	}
	latest := map[string]continuity.VerificationResult{}
	for _, result := range m.VerificationResults {
		current, ok := latest[result.ContractID]
		if !ok || result.ObservedAt.After(current.ObservedAt) || (result.ObservedAt.Equal(current.ObservedAt) && result.ID > current.ID) {
			latest[result.ContractID] = result
		}
	}
	actionOwners := map[string]string{}
	for _, action := range m.Actions {
		actionOwners[action.ID] = action.OwnerPrincipalID
	}
	for _, contract := range active {
		result, ok := latest[contract.ID]
		if !ok || result.Result != continuity.VerificationPassed || strings.TrimSpace(result.ReviewerPrincipalID) == "" {
			return false
		}
		if contract.AuthorityPrincipalID != "" && result.ReviewerPrincipalID != contract.AuthorityPrincipalID {
			return false
		}
		if owner := actionOwners[contract.ActionID]; owner != "" && owner == result.ReviewerPrincipalID {
			return false
		}
	}
	return true
}

func currentResponse(values []continuity.ResponsePackage) *continuity.ResponsePackage {
	var selected *continuity.ResponsePackage
	for index := range values {
		value := values[index]
		if selected == nil || value.UpdatedAt.After(selected.UpdatedAt) || (value.UpdatedAt.Equal(selected.UpdatedAt) && value.ID > selected.ID) {
			copy := value
			selected = &copy
		}
	}
	return selected
}

func responseAtLeast(m continuity.MatterAggregate, expected continuity.ResponseStatus) bool {
	response := currentResponse(m.ResponsePackages)
	if response == nil {
		return false
	}
	rank := map[continuity.ResponseStatus]int{
		continuity.ResponseDraft: 0, continuity.ResponseInReview: 1, continuity.ResponseApproved: 2,
		continuity.ResponseTransmitted: 3, continuity.ResponseAcknowledged: 4,
		continuity.ResponseRejected: -1, continuity.ResponseWithdrawn: -1,
	}
	return rank[response.Status] >= rank[expected]
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
