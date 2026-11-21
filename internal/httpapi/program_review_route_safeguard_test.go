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
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type reviewSafeguardRepository struct {
	*continuity.MemoryRepository
	readCalls  int
	writeCalls int
}

func (r *reviewSafeguardRepository) LatestProgramReview(ctx context.Context, tenant, programID, principalID string) (*continuity.ProgramReviewCheckpoint, error) {
	r.readCalls++
	return r.MemoryRepository.LatestProgramReview(ctx, tenant, programID, principalID)
}

func (r *reviewSafeguardRepository) RecordProgramReview(ctx context.Context, checkpoint continuity.ProgramReviewCheckpoint, event continuity.Event) (continuity.ProgramReviewCheckpoint, error) {
	r.writeCalls++
	return r.MemoryRepository.RecordProgramReview(ctx, checkpoint, event)
}

type expiredReviewAuthenticator struct{}

func (expiredReviewAuthenticator) Authenticate(*http.Request) (identity.Actor, bool, error) {
	return identity.Actor{}, false, identity.ErrExpiredIdentity
}

func TestRoutedProgramReviewAcceptanceNeverWritesAndPreservesHistory(t *testing.T) {
	ctx := continuity.WithTrustedSystemScope(context.Background())
	repo := &reviewSafeguardRepository{MemoryRepository: continuity.NewMemoryRepository()}
	service := continuity.NewService(repo)
	program, err := service.CreateProgram(ctx, continuity.CreateProgramInput{TenantID: "bank", LegalEntityID: "entity", Code: "REVIEW", Name: "Review", Type: "REGULATORY", OwningFunction: "Risk", Scope: json.RawMessage(`{}`), EffectiveFrom: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	digest, err := service.ProgramReviewDigest(ctx, "bank", program.Program.ID, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := service.AcceptProgramReview(ctx, continuity.AcceptProgramReviewInput{TenantID: "bank", ProgramID: program.Program.ID, PrincipalID: "reviewer", ExpectedProgramVersion: program.Program.Version, ExpectedProjectionVersion: digest.CurrentProjectionVersion})
	if err != nil || baseline.Checkpoint == nil {
		t.Fatalf("seed checkpoint: %+v %v", baseline, err)
	}
	guard, err := commandauth.New(nil, commandauth.ModeOff, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, tc := range []struct {
		name   string
		auth   identity.Authenticator
		status int
	}{
		{"missing identity", nil, http.StatusUnauthorized},
		{"expired identity", expiredReviewAuthenticator{}, http.StatusUnauthorized},
		{"verified identity", identity.NewDevelopmentAuthenticator("bank", "reviewer", "entity"), http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo.readCalls, repo.writeCalls = 0, 0
			handler := New(Dependencies{Logger: logger, Mode: "test-memory", Identity: tc.auth, Continuity: service, CommandGuard: guard})
			request := httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID+"/reviews", strings.NewReader(`{"expected_program_version":1,"expected_projection_version":1}`))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if repo.readCalls != 0 || repo.writeCalls != 0 {
				t.Fatalf("review service invoked: reads=%d writes=%d", repo.readCalls, repo.writeCalls)
			}
			stored, err := repo.MemoryRepository.LatestProgramReview(ctx, "bank", program.Program.ID, "reviewer")
			if err != nil || stored == nil || stored.ID != baseline.Checkpoint.ID {
				t.Fatalf("checkpoint changed: %+v %v", stored, err)
			}
		})
	}
}
