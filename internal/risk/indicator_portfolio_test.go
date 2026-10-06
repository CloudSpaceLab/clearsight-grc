package risk

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestMemoryIndicatorPortfolioDeduplicatesRiskLinksAndPaginates(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 10, 6, 7, 0, 0, 0, time.UTC)
	scope := Scope{TenantID: "bank-a", LegalEntityID: "entity-a"}

	create := func(id, code string, status Status) Risk {
		value := Risk{
			ID: id, TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
			Code: code, Name: code + " risk", Statement: "Risk statement", Impact: "Material impact",
			Scope: json.RawMessage(`{}`), Status: status, Version: 1, CreatedAt: now, UpdatedAt: now,
		}
		created, err := repo.Create(context.Background(), value, Event{RiskID: id, RiskVersion: 1, Type: EventRiskCreated, OccurredAt: now})
		if err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
		return created
	}
	link := func(current Risk, id, programID, checkID string, version int64, kind IndicatorKind) Risk {
		nextVersion := current.Version + 1
		updated, _, err := repo.AddIndicator(context.Background(), scope, current.ID, current.Version, IndicatorLink{
			ID: id, RiskID: current.ID, RiskVersion: nextVersion, ProgramID: programID,
			MonitoringCheckID: checkID, MonitoringCheckVersion: version, Kind: kind,
			Measurement: IndicatorMonitoringRiskScore, CreatedAt: now,
		}, Event{RiskID: current.ID, RiskVersion: nextVersion, Type: EventIndicatorLinked, OccurredAt: now})
		if err != nil {
			t.Fatalf("link %s: %v", id, err)
		}
		return updated
	}

	riskA := create("risk-a", "RISK-A", StatusActive)
	riskB := create("risk-b", "RISK-B", StatusActive)
	riskRetired := create("risk-c", "RISK-C", StatusRetired)

	riskA = link(riskA, "link-a-v1", "program-1", "check-1", 1, IndicatorKRI)
	riskA = link(riskA, "link-a-v2", "program-1", "check-1", 2, IndicatorKRI)
	riskB = link(riskB, "link-b-v2", "program-1", "check-1", 2, IndicatorKRI)
	riskB = link(riskB, "link-b-kci", "program-2", "check-2", 1, IndicatorKCI)
	_ = link(riskRetired, "link-retired", "program-1", "check-1", 2, IndicatorKRI)

	kriPage, err := repo.IndicatorPortfolio(context.Background(), scope, IndicatorPortfolioFilter{Kind: IndicatorKRI, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(kriPage.Items) != 1 {
		t.Fatalf("KRI items = %#v", kriPage.Items)
	}
	item := kriPage.Items[0]
	if item.MonitoringCheckID != "check-1" || item.MonitoringCheckVersion != 2 || item.RiskCount != 2 {
		t.Fatalf("deduplicated KRI item = %#v", item)
	}

	first, err := repo.IndicatorPortfolio(context.Background(), scope, IndicatorPortfolioFilter{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || first.NextCursor == "" {
		t.Fatalf("first page = %#v", first)
	}
	second, err := repo.IndicatorPortfolio(context.Background(), scope, IndicatorPortfolioFilter{Limit: 1, Cursor: first.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].MonitoringCheckID == first.Items[0].MonitoringCheckID {
		t.Fatalf("second page = %#v", second)
	}
}

func TestIndicatorPortfolioRejectsInvalidCursorAndKind(t *testing.T) {
	repo := NewMemoryRepository()
	scope := Scope{TenantID: "bank-a", LegalEntityID: "entity-a"}

	if _, err := repo.IndicatorPortfolio(context.Background(), scope, IndicatorPortfolioFilter{Kind: "KPI"}); err != ErrInvalid {
		t.Fatalf("invalid kind error = %v", err)
	}
	if _, err := repo.IndicatorPortfolio(context.Background(), scope, IndicatorPortfolioFilter{Cursor: "not-a-cursor"}); err != ErrInvalid {
		t.Fatalf("invalid cursor error = %v", err)
	}
}
