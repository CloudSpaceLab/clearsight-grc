package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
)

type backgroundRecoveryStub struct{ input operations.RetryInput }

func (s *backgroundRecoveryStub) BackgroundJobs(context.Context, string, int) (operations.Snapshot, error) {
	return operations.Snapshot{}, nil
}

func (s *backgroundRecoveryStub) RetryTerminalJob(_ context.Context, input operations.RetryInput) (operations.RecoveryReceipt, error) {
	s.input = input
	return operations.RecoveryReceipt{JobID: input.JobID, Queue: input.Queue, PreviousAttempts: input.ExpectedAttempts, State: "READY", RetriedAt: time.Now().UTC()}, nil
}

func TestRetryBackgroundJobUsesVerifiedActorAndExactTerminalAttempt(t *testing.T) {
	source := &backgroundRecoveryStub{}
	api := &API{deps: Dependencies{BackgroundJobs: operations.NewService(source)}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/operations/background-jobs/job-1/retry", strings.NewReader(`{"tenant_id":"bank","queue":"outbox-delivery","expected_attempts":5,"rationale":"The tenant lookup defect is fixed and this event can be delivered safely."}`))
	request.SetPathValue("job_id", "job-1")
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "admin-1", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}))
	response := httptest.NewRecorder()

	api.retryBackgroundJob(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("retry returned %d: %s", response.Code, response.Body.String())
	}
	if source.input.ActorPrincipalID != "admin-1" || source.input.JobID != "job-1" || source.input.ExpectedAttempts != 5 {
		t.Fatalf("recovery input = %#v", source.input)
	}
}
