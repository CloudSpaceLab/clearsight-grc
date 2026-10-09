package main

import "testing"

func TestSeedModeAllowsOnlyOneScopedOperation(t *testing.T) {
	if err := validateSeedMode(seedMode{SourceRisksOnly: true}); err != nil {
		t.Fatalf("Risk-only mode rejected: %v", err)
	}
	for name, mode := range map[string]seedMode{
		"source records": {SourceRisksOnly: true, SourceRecordsOnly: true},
		"source losses":  {SourceRisksOnly: true, SourceLossesOnly: true},
		"employees":      {SourceRisksOnly: true, SourceEmployeesOnly: true},
		"documents":      {SourceRisksOnly: true, DocumentSamplesOnly: true},
		"cloudspace":     {SourceRisksOnly: true, CloudspaceRelationshipID: "relationship-1"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateSeedMode(mode); err == nil {
				t.Fatal("expected mutually exclusive scoped modes to fail")
			}
		})
	}
}

func TestSourceRiskSeedScopeRequiresClearBankDemo(t *testing.T) {
	if err := validateSourceRiskSeedScope("00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-000000000002"); err != nil {
		t.Fatalf("canonical demo scope rejected: %v", err)
	}
	if err := validateSourceRiskSeedScope("another-tenant", "00000000-0000-4000-8000-000000000002"); err == nil {
		t.Fatal("expected another tenant to be rejected")
	}
}
