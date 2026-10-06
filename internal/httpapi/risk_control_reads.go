package httpapi

import (
	"context"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/controlcatalog"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

type riskControlEvidenceRead struct {
	ContractID string                        `json:"contract_id"`
	Name       string                        `json:"name"`
	Conclusion continuity.EvidenceConclusion `json:"conclusion,omitempty"`
	AssessedAt *time.Time                    `json:"assessed_at,omitempty"`
	ValidUntil *time.Time                    `json:"valid_until,omitempty"`
}

type riskControlRead struct {
	Link                  risk.ControlLink                       `json:"link"`
	Definition            controlcatalog.Definition              `json:"definition"`
	ProgramID             string                                 `json:"program_id"`
	ProgramName           string                                 `json:"program_name"`
	ImplementationID      string                                 `json:"implementation_id"`
	ObjectiveID           string                                 `json:"objective_id"`
	ImplementationName    string                                 `json:"implementation_name"`
	ImplementationType    string                                 `json:"implementation_type"`
	ImplementationStatus  continuity.ControlImplementationStatus `json:"implementation_status"`
	ImplementationVersion int64                                  `json:"implementation_version"`
	OwnerDisplayName      string                                 `json:"owner_display_name,omitempty"`
	OwnerAssigned         bool                                   `json:"owner_assigned"`
	Evidence              []riskControlEvidenceRead              `json:"evidence"`
}

type riskAggregateRead struct {
	risk.Aggregate
	ControlDetails              []riskControlRead                 `json:"control_details"`
	ControlDetailsComplete      bool                              `json:"control_details_complete"`
	IndicatorDetails            []riskIndicatorRead               `json:"indicator_details"`
	IndicatorDetailsComplete    bool                              `json:"indicator_details_complete"`
	NotificationHistory         []notificationDeliveryHistoryItem `json:"notification_history"`
	NotificationHistoryComplete bool                              `json:"notification_history_complete"`
}

func (a *API) riskAggregateWithControls(ctx context.Context, actor identity.Actor, value risk.Aggregate) riskAggregateRead {
	notificationHistory, notificationHistoryComplete := a.notificationDeliveryHistory(ctx, actor, "RISK", value.Risk.ID)
	result := riskAggregateRead{
		Aggregate:                   value,
		ControlDetails:              []riskControlRead{},
		ControlDetailsComplete:      true,
		NotificationHistory:         notificationHistory,
		NotificationHistoryComplete: notificationHistoryComplete,
	}
	if len(value.Controls) == 0 {
		return result
	}
	if a == nil || a.deps.ControlCatalog == nil || a.deps.Continuity == nil {
		result.ControlDetailsComplete = false
		return result
	}

	type pendingOwner struct {
		index int
		id    string
	}
	owners := make([]pendingOwner, 0, len(value.Controls))
	definitions := map[string]controlcatalog.Definition{}
	programs := map[string]continuity.ProgramAggregate{}
	now := time.Now().UTC()

	for _, link := range value.Controls {
		catalogLink, err := a.deps.ControlCatalog.GetImplementationLink(ctx, actor.TenantID, actor.LegalEntityID, link.CatalogLinkID)
		if err != nil {
			result.ControlDetailsComplete = false
			continue
		}
		definition, ok := definitions[catalogLink.DefinitionID]
		if !ok {
			definition, err = a.deps.ControlCatalog.GetDefinition(ctx, actor.TenantID, catalogLink.DefinitionID)
			if err != nil {
				result.ControlDetailsComplete = false
				continue
			}
			definitions[catalogLink.DefinitionID] = definition
		}
		program, ok := programs[catalogLink.ProgramID]
		if !ok {
			program, err = a.deps.Continuity.GetProgram(ctx, actor.TenantID, catalogLink.ProgramID)
			if err == nil {
				program, err = a.programForActor(ctx, program, nil)
			}
			if err != nil || program.Program.LegalEntityID != actor.LegalEntityID {
				result.ControlDetailsComplete = false
				continue
			}
			programs[catalogLink.ProgramID] = program
		}
		implementation, found := programControlImplementation(program, catalogLink.ImplementationID)
		if !found {
			result.ControlDetailsComplete = false
			continue
		}
		detail := riskControlRead{
			Link:                  link,
			Definition:            definition,
			ProgramID:             program.Program.ID,
			ProgramName:           program.Program.Name,
			ImplementationID:      implementation.ID,
			ObjectiveID:           implementation.ObjectiveID,
			ImplementationName:    implementation.Name,
			ImplementationType:    implementation.ImplementationType,
			ImplementationStatus:  implementation.Status,
			ImplementationVersion: implementation.Version,
			OwnerAssigned:         implementation.OwnerPrincipalID != "",
			Evidence:              controlEvidenceReads(program, implementation.ID, now),
		}
		result.ControlDetails = append(result.ControlDetails, detail)
		if implementation.OwnerPrincipalID != "" {
			owners = append(owners, pendingOwner{index: len(result.ControlDetails) - 1, id: implementation.OwnerPrincipalID})
		}
	}

	ownerIDs := make([]string, 0, len(owners))
	for _, owner := range owners {
		ownerIDs = append(ownerIDs, owner.id)
	}
	labels := a.exactAssessmentLabels(ctx, actor, actor.LegalEntityID, ownerIDs)
	for _, owner := range owners {
		result.ControlDetails[owner.index].OwnerDisplayName = labels[owner.id]
	}
	return result
}

func programControlImplementation(program continuity.ProgramAggregate, id string) (continuity.ControlImplementation, bool) {
	for _, implementation := range program.ControlImplementations {
		if implementation.ID == id {
			return implementation, true
		}
	}
	return continuity.ControlImplementation{}, false
}

func controlEvidenceReads(program continuity.ProgramAggregate, implementationID string, now time.Time) []riskControlEvidenceRead {
	latest := map[string]continuity.EvidenceAssessment{}
	for _, assessment := range program.EvidenceAssessments {
		current, ok := latest[assessment.ContractID]
		if !ok || assessment.AssessedAt.After(current.AssessedAt) {
			latest[assessment.ContractID] = assessment
		}
	}
	values := make([]riskControlEvidenceRead, 0)
	for _, contract := range program.EvidenceContracts {
		if contract.Status != continuity.EvidenceContractActive || contract.ControlImplementationID != implementationID {
			continue
		}
		read := riskControlEvidenceRead{ContractID: contract.ID, Name: contract.Name}
		if assessment, ok := latest[contract.ID]; ok {
			conclusion := assessment.Conclusion
			if assessment.ValidUntil != nil && !now.Before(*assessment.ValidUntil) {
				conclusion = continuity.EvidenceExpired
			}
			assessedAt := assessment.AssessedAt
			read.Conclusion = conclusion
			read.AssessedAt = &assessedAt
			read.ValidUntil = assessment.ValidUntil
		}
		values = append(values, read)
	}
	return values
}
