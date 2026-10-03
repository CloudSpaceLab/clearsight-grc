package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/controlcatalog"
	"github.com/CloudSpaceLab/clearsight-grc/internal/rcsa"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

type rcsaPopulationResolver struct {
	Risks      *risk.Service
	Catalog    *controlcatalog.Service
	Continuity *continuity.Service
}

func (r rcsaPopulationResolver) ResolvePopulation(ctx context.Context, scope rcsa.Scope, riskIDs []string, at time.Time) (rcsa.Population, error) {
	if r.Risks == nil || r.Catalog == nil || r.Continuity == nil {
		return rcsa.Population{}, rcsa.ErrInvalid
	}
	population := rcsa.Population{
		Risks:    make([]rcsa.RiskSnapshot, 0, len(riskIDs)),
		Controls: make([]rcsa.ControlSnapshot, 0),
	}
	programs := map[string]continuity.ProgramAggregate{}
	definitions := map[string]controlcatalog.Definition{}
	catalogLinks := map[string]controlcatalog.ImplementationLink{}

	for _, riskID := range riskIDs {
		aggregate, err := r.Risks.Get(ctx, risk.Scope{TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID}, strings.TrimSpace(riskID))
		if err != nil {
			if errors.Is(err, risk.ErrNotFound) || errors.Is(err, risk.ErrInvalid) {
				return rcsa.Population{}, rcsa.ErrInvalid
			}
			return rcsa.Population{}, err
		}
		current := aggregate.Risk
		if current.Status != risk.StatusActive {
			return rcsa.Population{}, rcsa.ErrInvalid
		}
		population.Risks = append(population.Risks, rcsa.RiskSnapshot{
			RiskID: current.ID, RiskVersion: current.Version, Code: current.Code, Name: current.Name, Category: current.Category,
		})
		for _, control := range aggregate.Controls {
			catalogLink, ok := catalogLinks[control.CatalogLinkID]
			if !ok {
				catalogLink, err = r.Catalog.GetImplementationLink(ctx, scope.TenantID, scope.LegalEntityID, control.CatalogLinkID)
				if err != nil {
					return rcsa.Population{}, rcsa.ErrInvalid
				}
				catalogLinks[control.CatalogLinkID] = catalogLink
			}
			definition, ok := definitions[catalogLink.DefinitionID]
			if !ok {
				definition, err = r.Catalog.GetDefinition(ctx, scope.TenantID, catalogLink.DefinitionID)
				if err != nil || definition.Status != controlcatalog.DefinitionActive {
					return rcsa.Population{}, rcsa.ErrInvalid
				}
				definitions[catalogLink.DefinitionID] = definition
			}
			program, ok := programs[catalogLink.ProgramID]
			if !ok {
				program, err = r.Continuity.GetProgram(continuity.WithTrustedSystemScope(ctx), scope.TenantID, catalogLink.ProgramID)
				if err != nil || program.Program.LegalEntityID != scope.LegalEntityID {
					return rcsa.Population{}, rcsa.ErrInvalid
				}
				programs[catalogLink.ProgramID] = program
			}
			implementation, found := rcsaControlImplementation(program, catalogLink.ImplementationID)
			if !found || !rcsaImplementationAssessable(implementation, at) {
				return rcsa.Population{}, rcsa.ErrInvalid
			}
			population.Controls = append(population.Controls, rcsa.ControlSnapshot{
				RiskID: current.ID, RiskVersion: current.Version,
				RiskControlLinkID: control.ID, CatalogLinkID: catalogLink.ID,
				DefinitionID: definition.ID, DefinitionCode: definition.Code, DefinitionName: definition.Name,
				ProgramID: program.Program.ID, ImplementationID: implementation.ID,
				ImplementationVersion: implementation.Version, ImplementationName: implementation.Name,
				ImplementationStatus: string(implementation.Status), ImplementationEffectiveFrom: implementation.EffectiveFrom,
				ImplementationEffectiveUntil: implementation.EffectiveUntil,
			})
		}
	}
	return population, nil
}

func rcsaControlImplementation(program continuity.ProgramAggregate, id string) (continuity.ControlImplementation, bool) {
	for _, value := range program.ControlImplementations {
		if value.ID == id {
			return value, true
		}
	}
	return continuity.ControlImplementation{}, false
}


func rcsaImplementationAssessable(value continuity.ControlImplementation, at time.Time) bool {
	if value.Status == continuity.ImplementationInactive || value.Status == continuity.ImplementationRetired {
		return false
	}
	if value.EffectiveFrom.IsZero() || at.Before(value.EffectiveFrom) {
		return false
	}
	return value.EffectiveUntil == nil || at.Before(*value.EffectiveUntil)
}
