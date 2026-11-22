package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

func TestRiskIndicatorReadsEnforceInterventionVisibility(t *testing.T) {
	fixture := newLinkedIssueFixture(t)
	ctx := t.Context()
	matter, err := fixture.continuity.CreateMatter(continuity.WithTrustedSystemScope(ctx), continuity.CreateMatterInput{TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterControlGap, Priority: 3, Title: "Restricted issue", Summary: "Restricted issue", Scope: json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["program-owner"]}`), TriggerKey: "monitoring-check-adverse:" + fixture.check.ID, ProgramID: fixture.program.Program.ID})
	if err != nil {
		t.Fatal(err)
	}
	svc := risk.NewService(risk.NewMemoryRepository())
	svc.ConfigureIndicatorLinkValidator(func(_ context.Context, _ risk.Scope, _ string, _ int64) (string, error) {
		return fixture.program.Program.ID, nil
	})
	created, err := svc.Create(ctx, risk.CreateInput{TenantID: "bank", LegalEntityID: "entity-a", Code: "R-TEST", Name: "Indicator risk", Statement: "A service may fail.", Impact: "Service unavailable.", Scope: json.RawMessage(`{}`), ActorID: "program-owner"})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.LinkIndicator(ctx, risk.LinkIndicatorInput{TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedRiskVersion: created.Version, MonitoringCheckID: fixture.check.ID, MonitoringCheckVersion: fixture.check.Version, Kind: risk.IndicatorKRI, ActorID: "program-owner"})
	if err != nil {
		t.Fatal(err)
	}
	for _, principal := range []string{"program-owner", "other-reader"} {
		t.Run(principal, func(t *testing.T) {
			handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", principal, "entity-a", "CRO"), Risk: svc, Monitoring: fixture.monitoring, Continuity: fixture.continuity})
			for _, path := range []string{"/api/v1/risks/" + created.ID, "/api/v1/risk-indicators"} {
				res := httptest.NewRecorder()
				handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
				if res.Code != http.StatusOK {
					t.Fatalf("%s status %d: %s", path, res.Code, res.Body.String())
				}
				body := res.Body.String()
				if !strings.Contains(body, fixture.check.ID) {
					t.Fatalf("indicator absent: %s", body)
				}
				if principal == "program-owner" {
					if !strings.Contains(body, `"intervention"`) || !strings.Contains(body, matter.Matter.ID) || !strings.Contains(body, matter.Matter.Reference) {
						t.Fatalf("visible intervention missing: %s", body)
					}
				} else if strings.Contains(body, `"intervention"`) || strings.Contains(body, matter.Matter.ID) || strings.Contains(body, matter.Matter.Reference) {
					t.Fatalf("intervention metadata exposed: %s", body)
				}
			}
		})
	}
}
