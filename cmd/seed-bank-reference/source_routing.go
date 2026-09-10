//go:build postgres

package main

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

// repairSourceMatterOwner separates the initial bank review from the source
// action performer. It does not change the action, source facts, or authority.
// Run before other repairs: version 3 is the original create/link/action shape.
func repairSourceMatterOwner(ctx context.Context, cs *continuity.Service, seed bankverticals.SeedConfig, matter continuity.MatterAggregate, sourceOwner string) (continuity.MatterAggregate, error) {
	if !sourceMatterOwnerRepairEligible(seed, matter, sourceOwner) {
		return matter, nil
	}
	return cs.AssignMatter(ctx, continuity.AssignMatterInput{
		TenantID: seed.TenantID, MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		OwnerPrincipalID: seed.OwnerPrincipalID, ActorID: seed.ActorID,
		Rationale:         "Assign bank accountability for the initial review; the source action performer and recorded source owner remain unchanged.",
		ReassignmentBasis: sourceRecordPackage + ":accountability-repair-v1",
	})
}

func sourceMatterOwnerRepairEligible(seed bankverticals.SeedConfig, aggregate continuity.MatterAggregate, sourceOwner string) bool {
	matter := aggregate.Matter
	if seed.TenantID != "00000000-0000-4000-8000-000000000001" || seed.LegalEntityID != "00000000-0000-4000-8000-000000000002" || seed.OwnerPrincipalID != "00000000-0000-4000-8000-000000000107" {
		return false
	}
	if matter.TenantID != seed.TenantID || matter.LegalEntityID != seed.LegalEntityID || matter.Status != continuity.MatterInitialReview || matter.Version != 3 || sourceOwner == "" || sourceOwner == seed.OwnerPrincipalID || matter.OwnerPrincipalID != sourceOwner || matter.TriggerType != "SOURCE_REGISTER_IMPORT" || !strings.HasPrefix(matter.TriggerKey, sourceRecordPackage+":") {
		return false
	}
	var scope struct {
		Sample      bool   `json:"sample"`
		SeedPackage string `json:"seed_package"`
	}
	if json.Unmarshal(matter.Scope, &scope) != nil || !scope.Sample || scope.SeedPackage != sourceRecordPackage || len(aggregate.Actions) != 1 {
		return false
	}
	action := aggregate.Actions[0]
	return action.OriginKey == matter.TriggerKey && action.OwnerPrincipalID == sourceOwner && action.Status == continuity.ActionPlanned && action.Version == 1
}
