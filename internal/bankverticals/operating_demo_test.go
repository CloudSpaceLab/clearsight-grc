package bankverticals

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

func operatingTestService(t *testing.T) (*Service, SeedConfig) {
	t.Helper()
	config := DemoSeedConfig()
	config.Now = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	repo := continuity.NewMemoryRepository()
	capture := newReferenceEvidenceService(config.Now, config.LegalEntityID)
	s := NewService(continuity.NewServiceWithClock(repo, func() time.Time { return config.Now }), capture)
	s.ConfigureMonitoring(monitoring.NewService(monitoring.NewMemoryRepository(), capture))
	return s, config
}

func TestOperatingDemoCreatesConnectedPopulationAndStableTimeline(t *testing.T) {
	s, c := operatingTestService(t)
	got, err := s.EnsureOperatingDemo(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Programs) != 5 || len(got.Forms) != 7 || len(got.Matters) != 7 {
		t.Fatalf("incomplete catalogue %+v", got)
	}
	for key, p := range got.Programs {
		if p.ID == "" || !p.Eligible {
			t.Fatalf("program %s unavailable %+v", key, p)
		}
	}
	for key, f := range got.Forms {
		if !f.Eligible || f.Status != "ACTIVE" || f.ProgramID == "" {
			t.Fatalf("form %s unavailable %+v", key, f)
		}
	}
	ctx := continuity.WithTrustedSystemEntityScope(context.Background(), c.TenantID, c.LegalEntityID)
	access, err := s.continuity.GetMatter(ctx, c.TenantID, got.Matters["access_remediation"].ID)
	if err != nil {
		t.Fatal(err)
	}
	actions := currentActions(access.Actions)
	if len(actions) != 2 || actions[0].Status != continuity.ActionBlocked || actions[1].Status != continuity.ActionImplemented || len(access.VerificationContracts) != 1 || len(access.VerificationResults) != 0 {
		t.Fatalf("access owner/reviewer work lost %+v", access)
	}
	if access.Closure.Ready {
		t.Fatal("open action and missing review allowed closure")
	}
	branch, _ := s.continuity.GetMatter(ctx, c.TenantID, got.Matters["branch_control_followup"].ID)
	if branch.Matter.Status != continuity.MatterVerification || len(branch.VerificationResults) != 0 {
		t.Fatalf("repair must await independent verification %+v", branch)
	}
	c.Now = c.Now.AddDate(0, 1, 0)
	again, err := s.EnsureOperatingDemo(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if !got.StartedAt.Equal(again.StartedAt) || !reflect.DeepEqual(got.Programs, again.Programs) || !reflect.DeepEqual(got.Forms, again.Forms) || !reflect.DeepEqual(got.Matters, again.Matters) {
		t.Fatalf("repeat changed catalogue before=%+v after=%+v", got, again)
	}
}

func TestOperatingDemoPreservesPausedAndCustomForms(t *testing.T) {
	s, c := operatingTestService(t)
	got, err := s.EnsureOperatingDemo(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	form := got.Forms["branch_control_return"]
	actor := monitoring.Actor{TenantID: c.TenantID, LegalEntityID: c.LegalEntityID, PrincipalID: c.ActorID}
	paused, err := s.monitoring.TransitionForm(context.Background(), actor, monitoring.TransitionInput{ID: form.ID, ProgramID: form.ProgramID, LegalEntityID: c.LegalEntityID, ExpectedVersion: form.Version, To: monitoring.LifecyclePaused})
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.EnsureOperatingDemo(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	preserved := again.Forms["branch_control_return"]
	if preserved.Eligible || preserved.ID != paused.ID || preserved.Version != paused.Version || preserved.Status != "PAUSED" {
		t.Fatalf("paused form changed %+v", preserved)
	}
}

func TestOperatingDemoRejectsSameMakerReviewerBeforeWrites(t *testing.T) {
	s, c := operatingTestService(t)
	c.ReviewerPrincipalID = c.ActorID
	if _, err := s.EnsureOperatingDemo(context.Background(), c); err == nil {
		t.Fatal("independent reviewer required")
	}
}
