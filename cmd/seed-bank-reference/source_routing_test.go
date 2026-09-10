//go:build postgres

package main

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestSourceMatterOwnerRepairPreservesUserChanges(t *testing.T) {
	seed := bankverticals.SeedConfig{TenantID: "00000000-0000-4000-8000-000000000001", LegalEntityID: "00000000-0000-4000-8000-000000000002", OwnerPrincipalID: "00000000-0000-4000-8000-000000000107"}
	fixture := func() continuity.MatterAggregate {
		return continuity.MatterAggregate{
			Matter:  continuity.Matter{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, Status: continuity.MatterInitialReview, Version: 3, OwnerPrincipalID: "sample-performer", TriggerType: "SOURCE_REGISTER_IMPORT", TriggerKey: sourceRecordPackage + ":synthetic-record", Scope: sourceJSON(map[string]any{"sample": true, "seed_package": sourceRecordPackage})},
			Actions: []continuity.Action{{OwnerPrincipalID: "sample-performer", OriginKey: sourceRecordPackage + ":synthetic-record", Status: continuity.ActionPlanned, Version: 1}},
		}
	}
	if !sourceMatterOwnerRepairEligible(seed, fixture(), "sample-performer") {
		t.Fatal("untouched source owner must be eligible for accountability repair")
	}
	for name, change := range map[string]func(*continuity.MatterAggregate){
		"later edit":         func(m *continuity.MatterAggregate) { m.Matter.Version++ },
		"different owner":    func(m *continuity.MatterAggregate) { m.Matter.OwnerPrincipalID = "another-owner" },
		"advanced lifecycle": func(m *continuity.MatterAggregate) { m.Matter.Status = continuity.MatterAssessment },
		"other tenant":       func(m *continuity.MatterAggregate) { m.Matter.TenantID = "other" },
		"other entity":       func(m *continuity.MatterAggregate) { m.Matter.LegalEntityID = "other" },
		"other import": func(m *continuity.MatterAggregate) {
			m.Matter.Scope = sourceJSON(map[string]any{"sample": true, "seed_package": "other"})
		},
		"other trigger":     func(m *continuity.MatterAggregate) { m.Matter.TriggerKey = "other:record" },
		"action changed":    func(m *continuity.MatterAggregate) { m.Actions[0].Version++ },
		"action reassigned": func(m *continuity.MatterAggregate) { m.Actions[0].OwnerPrincipalID = "another-performer" },
		"action started":    func(m *continuity.MatterAggregate) { m.Actions[0].Status = continuity.ActionInProgress },
	} {
		t.Run(name, func(t *testing.T) {
			matter := fixture()
			change(&matter)
			if sourceMatterOwnerRepairEligible(seed, matter, "sample-performer") {
				t.Fatal("repair must preserve changed or out-of-scope records")
			}
		})
	}
}
