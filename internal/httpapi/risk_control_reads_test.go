package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

func TestRiskControlReadFailsClosedWhenSourceDetailCannotBeResolved(t *testing.T) {
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "risk-owner"}
	value := risk.Aggregate{
		Risk: risk.Risk{
			ID: "risk-1", TenantID: "bank", LegalEntityID: "entity-a",
			Code: "RISK-1", Name: "Scoped risk", Statement: "A material risk exists.",
			Impact: "Critical service impact.", Status: risk.StatusActive, Version: 2,
			CreatedAt: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 10, 2, 12, 5, 0, 0, time.UTC),
		},
		Controls: []risk.ControlLink{{
			ID: "risk-control-1", RiskID: "risk-1", RiskVersion: 2,
			CatalogLinkID: "catalog-link-1", CreatedAt: time.Date(2026, 10, 2, 12, 5, 0, 0, time.UTC),
		}},
	}

	read := (&API{}).riskAggregateWithControls(context.Background(), actor, value)
	if read.ControlDetailsComplete {
		t.Fatal("unresolved control source was reported complete")
	}
	if len(read.ControlDetails) != 0 {
		t.Fatalf("unresolved control source leaked detail: %#v", read.ControlDetails)
	}
	if len(read.Controls) != 1 || read.Controls[0].CatalogLinkID != "catalog-link-1" {
		t.Fatalf("Risk control history was not preserved: %#v", read.Controls)
	}
}
