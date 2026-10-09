package main

import (
	"fmt"
	"strings"
)

const (
	clearBankDemoTenantID      = "00000000-0000-4000-8000-000000000001"
	clearBankDemoLegalEntityID = "00000000-0000-4000-8000-000000000002"
)

type seedMode struct {
	SourceRecordsOnly        bool
	SourceLossesOnly         bool
	SourceRisksOnly          bool
	SourceEmployeesOnly      bool
	DocumentSamplesOnly      bool
	CloudspaceRelationshipID string
}

func validateSeedMode(mode seedMode) error {
	selected := 0
	for _, enabled := range []bool{
		mode.SourceRecordsOnly,
		mode.SourceLossesOnly,
		mode.SourceRisksOnly,
		mode.SourceEmployeesOnly,
		mode.DocumentSamplesOnly,
		strings.TrimSpace(mode.CloudspaceRelationshipID) != "",
	} {
		if enabled {
			selected++
		}
	}
	if selected > 1 {
		return fmt.Errorf("choose one scoped sample operation")
	}
	return nil
}

func validateSourceRiskSeedScope(tenantID, legalEntityID string) error {
	if strings.TrimSpace(tenantID) != clearBankDemoTenantID || strings.TrimSpace(legalEntityID) != clearBankDemoLegalEntityID {
		return fmt.Errorf("source Risk reconciliation requires the non-production Clear Bank demo scope")
	}
	return nil
}
