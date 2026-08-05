package autonomy

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

func (s *Service) Ingest(ctx context.Context, value Signal) (Drift, bool, error) {
	if strings.TrimSpace(value.TenantID) == "" || value.Type == "" || strings.TrimSpace(value.SubjectType) == "" || strings.TrimSpace(value.SubjectID) == "" || strings.TrimSpace(value.Source) == "" || strings.TrimSpace(value.DedupeKey) == "" {
		return Drift{}, false, fmt.Errorf("tenant, type, subject, source and dedupe_key are required")
	}
	if value.ID == "" {
		generated, err := id.NewUUIDv7()
		if err != nil {
			return Drift{}, false, err
		}
		value.ID = generated
	}
	if value.ObservedAt.IsZero() {
		value.ObservedAt = s.now().UTC()
	}
	if value.EffectiveAt.IsZero() {
		value.EffectiveAt = value.ObservedAt
	}
	drift, err := assess(value, s.now().UTC())
	if err != nil {
		return Drift{}, false, err
	}
	inserted, err := s.repo.Ingest(ctx, value, drift)
	if err != nil || !inserted {
		return Drift{}, inserted, err
	}
	return drift, true, nil
}

func (s *Service) Readiness(ctx context.Context, tenant string) (Readiness, error) {
	if strings.TrimSpace(tenant) == "" {
		return Readiness{}, fmt.Errorf("tenant_id is required")
	}
	drifts, err := s.repo.ListDrifts(ctx, tenant)
	if err != nil {
		return Readiness{}, err
	}
	dimensions := ReadinessDimensions{}
	actions := []string{}
	maxSeverity := 0
	for _, drift := range drifts {
		if drift.Severity > maxSeverity {
			maxSeverity = drift.Severity
		}
		switch drift.Dimension {
		case "evidence_freshness":
			if drift.Severity <= 2 {
				dimensions.Aging++
			} else {
				dimensions.AtRisk++
			}
		case "source_quality":
			dimensions.Unknown++
		case "routing_integrity":
			dimensions.BlockedRouting++
		case "applicability", "control_effectiveness", "verification":
			dimensions.PendingHuman++
		default:
			dimensions.AtRisk++
		}
		actions = append(actions, drift.RequiredAction)
	}
	status := "UNKNOWN"
	if maxSeverity >= 5 {
		status = "CRITICAL"
	} else if maxSeverity >= 3 {
		status = "AT_RISK"
	} else if maxSeverity > 0 {
		status = "WATCH"
	}
	actions = dedupe(actions)
	if len(actions) == 0 {
		actions = []string{"Connect or approve a governed compliance baseline before representing the institution as current."}
	}
	if len(actions) > 5 {
		actions = actions[:5]
	}
	return Readiness{TenantID: tenant, Status: status, BaselineKnown: false, GeneratedAt: s.now().UTC(), Dimensions: dimensions, ActiveDrifts: drifts, RecommendedActions: actions}, nil
}

func assess(signal Signal, detected time.Time) (Drift, error) {
	dimension, severity, summary, action := "context", 1, "Institutional context changed.", "Review the affected scope."
	switch signal.Type {
	case SignalEvidenceAging:
		dimension, severity, summary, action = "evidence_freshness", 2, "Evidence is approaching its freshness limit.", "Refresh or confirm the evidence before it becomes stale."
	case SignalEvidenceExpired:
		dimension, severity, summary, action = "evidence_freshness", 4, "Required evidence has expired.", "Collect current proof and reassess dependent conclusions."
	case SignalSourceDegraded:
		dimension, severity, summary, action = "source_quality", 3, "An authoritative source is stale or unavailable.", "Use the approved fallback and restore source health."
	case SignalRequirementChanged:
		dimension, severity, summary, action = "applicability", 3, "A governing requirement changed.", "Review applicability and affected controls."
	case SignalRoutingGap, SignalOwnerRemoved:
		dimension, severity, summary, action = "routing_integrity", 4, "Responsible or authorized coverage is incomplete.", "Repair the assignment or escalation path before the next material action."
	case SignalControlFailed:
		dimension, severity, summary, action = "control_effectiveness", 5, "A control failure was observed.", "Create or update the Matter and obtain accountable treatment."
	case SignalVerificationFailed:
		dimension, severity, summary, action = "verification", 5, "Implemented remediation failed verification.", "Reopen treatment and update the current risk conclusion."
	}
	driftID, err := id.NewUUIDv7()
	if err != nil {
		return Drift{}, err
	}
	return Drift{ID: driftID, TenantID: signal.TenantID, SubjectType: signal.SubjectType, SubjectID: signal.SubjectID, Dimension: dimension, Severity: severity, State: "ACTIVE", Summary: summary, RequiredAction: action, SignalID: signal.ID, DetectedAt: detected}, nil
}

func dedupe(values []string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func SeedDemo(ctx context.Context, service *Service) {
	signals := []Signal{
		{TenantID: "bank-demo", Type: SignalEvidenceAging, SubjectType: "VENDOR", SubjectID: "payment-processor", Source: "evidence-scheduler", DedupeKey: "vendor-cert-aging"},
		{TenantID: "bank-demo", Type: SignalRequirementChanged, SubjectType: "PROGRAM", SubjectID: "cbn-digital", Source: "regulatory-feed", DedupeKey: "cbn-circular-2026-08"},
		{TenantID: "bank-demo", Type: SignalRoutingGap, SubjectType: "CONTROL", SubjectID: "branch-resilience", Source: "routing-integrity", DedupeKey: "branch-resilience-owner-gap"},
	}
	for _, signal := range signals {
		_, _, _ = service.Ingest(ctx, signal)
	}
}
