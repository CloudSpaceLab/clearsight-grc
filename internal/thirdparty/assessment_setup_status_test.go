package thirdparty

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestAssessmentSetupStatusDistinguishesRetryAndTerminalFailure(t *testing.T) {
	now := time.Date(2026, 8, 26, 16, 0, 0, 0, time.UTC)
	repository, assessment := assessmentProvisionerFixture(t, now)
	service := NewAssessmentService(repository, nil)
	matterService := &recordingAssessmentMatterService{matters: map[string]continuity.MatterAggregate{}, alwaysFail: true}
	provisioner := NewAssessmentProvisioner(repository, matterService, "worker-a")
	provisioner.Configure(time.Minute, 2, time.Second)
	actor := Actor{TenantID: assessment.TenantID, LegalEntityID: assessment.LegalEntityID, PrincipalID: "owner-1"}

	if _, err := provisioner.Maintain(context.Background(), now, 1); err == nil {
		t.Fatal("first Matter failure was not reported")
	}
	retrying, err := service.GetAssessmentSetupStatus(context.Background(), actor, assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retrying.State != AssessmentJobReady || retrying.Attempts != 1 || retrying.NextAttemptAt == nil || retrying.TerminalAt != nil || retrying.FailureCode != AssessmentSetupFailureMatter {
		t.Fatalf("retrying status = %#v", retrying)
	}

	if _, err := provisioner.Maintain(context.Background(), *retrying.NextAttemptAt, 1); err == nil {
		t.Fatal("second Matter failure was not reported")
	}
	terminal, err := service.GetAssessmentSetupStatus(context.Background(), actor, assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if terminal.State != AssessmentJobFailed || terminal.Attempts != 2 || terminal.NextAttemptAt != nil || terminal.TerminalAt == nil || terminal.FailureCode != AssessmentSetupFailureMatter {
		t.Fatalf("terminal status = %#v", terminal)
	}
}

func TestAssessmentSetupStatusIsLegalEntityScoped(t *testing.T) {
	now := time.Date(2026, 8, 26, 17, 0, 0, 0, time.UTC)
	repository, assessment := assessmentProvisionerFixture(t, now)
	service := NewAssessmentService(repository, nil)
	_, err := service.GetAssessmentSetupStatus(context.Background(), Actor{TenantID: assessment.TenantID, LegalEntityID: "other-entity", PrincipalID: "owner-1"}, assessment.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity setup status error = %v", err)
	}
}
