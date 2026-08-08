//go:build postgres && postgresintegration

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	evidenceTenantID      = "33333333-3333-7333-8333-333333333331"
	evidenceOtherTenantID = "33333333-3333-7333-8333-333333333332"
	evidenceRequesterID   = "33333333-3333-7333-8333-333333333333"
	evidenceRecipientID   = "33333333-3333-7333-8333-333333333334"
)

func TestEvidenceCapturePostgresContracts(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `TRUNCATE tenants CASCADE`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'evidence-bank','Evidence Bank'),($2::uuid,'other-bank','Other Bank')`, evidenceTenantID, evidenceOtherTenantID); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$3::uuid,'PERSON','Evidence requester','ACTIVE',$4),
		($2::uuid,$3::uuid,'PERSON','Evidence recipient','ACTIVE',$4)`, evidenceRequesterID, evidenceRecipientID, evidenceTenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	store, err := evidence.NewLocalObjectStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := evidence.NewService(evidence.NewPostgresRepository(pool), store)

	t.Run("source observations and freshness changes emit durable events", func(t *testing.T) {
		source, err := service.CreateSource(ctx, evidence.CreateSourceInput{TenantID: "evidence-bank", Code: "IAM", Name: "Identity directory", Type: evidence.SourceSystem, AuthorityClass: "SYSTEM_OF_RECORD", ExpectedFreshnessMinutes: 30})
		if err != nil {
			t.Fatal(err)
		}
		observed := now.Add(-2 * time.Hour)
		updated, err := service.RecordSourceObservation(ctx, evidence.SourceObservation{TenantID: "evidence-bank", SourceID: source.ID, ObservedAt: observed, Success: true, LatencyMS: 75})
		if err != nil {
			t.Fatal(err)
		}
		if updated.Health != evidence.HealthCurrent {
			t.Fatalf("expected current source, got %#v", updated)
		}
		changed, err := service.Maintain(ctx, now, 10)
		if err != nil || changed != 1 {
			t.Fatalf("maintain count=%d err=%v", changed, err)
		}
		sources, err := service.ListSources(ctx, "evidence-bank", 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(sources) != 1 || sources[0].Health != evidence.HealthStale {
			t.Fatalf("expected stale source, got %#v", sources)
		}
		var events int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_type='EVIDENCE_SOURCE'`, evidenceTenantID).Scan(&events); err != nil {
			t.Fatal(err)
		}
		if events != 2 {
			t.Fatalf("expected current and stale events, got %d", events)
		}
	})

	t.Run("request submission is tenant scoped recipient bound and versioned", func(t *testing.T) {
		request, err := createInternalEvidenceRequest(ctx, service, now, "CONTROL", "control-1")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.GetRequest(ctx, "other-bank", request.ID); !errors.Is(err, evidence.ErrNotFound) {
			t.Fatalf("expected tenant-scoped not found, got %v", err)
		}
		receipt, err := service.Submit(ctx, evidence.Submission{TenantID: "evidence-bank", RequestID: request.ID, SubmittedBy: evidenceRecipientID, Channel: "INTERNAL", ExpectedVersion: request.Version, Answers: map[string]string{"state": "Operating"}})
		if err != nil {
			t.Fatal(err)
		}
		if receipt.Version != 2 || receipt.Status != evidence.RequestSubmitted {
			t.Fatalf("unexpected receipt %#v", receipt)
		}
		var submissions, events int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_submissions WHERE request_id=$1::uuid`, request.ID).Scan(&submissions); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_id=$1::uuid AND event_type='EvidenceResponseSubmitted'`, request.ID).Scan(&events); err != nil {
			t.Fatal(err)
		}
		if submissions != 1 || events != 1 {
			t.Fatalf("expected submission/event, got %d/%d", submissions, events)
		}
	})

	t.Run("magic links are canonical-audience-bound hash-only one-time bounded and revocable", func(t *testing.T) {
		const audience = "security@example.com"
		request, err := createExternalEvidenceRequest(ctx, service, now, "VENDOR", "vendor-1", audience)
		if err != nil {
			t.Fatal(err)
		}
		first, err := service.IssueInvitation(ctx, evidence.IssueInvitationInput{TenantID: "evidence-bank", RequestID: request.ID, Audience: audience, Purpose: "Vendor assurance response", TTLMinutes: 60, CreatedBy: evidenceRequesterID})
		if err != nil {
			t.Fatal(err)
		}
		var tokenBytes, audienceBytes int
		if err := pool.QueryRow(ctx, `SELECT octet_length(token_hash),octet_length(audience_hash) FROM capture_invitations WHERE id=$1::uuid`, first.InvitationID).Scan(&tokenBytes, &audienceBytes); err != nil {
			t.Fatal(err)
		}
		if tokenBytes != 32 || audienceBytes != 32 {
			t.Fatalf("expected hash-only storage, got %d/%d", tokenBytes, audienceBytes)
		}
		if _, err := service.RedeemInvitation(ctx, first.Token, "other@example.com"); !errors.Is(err, evidence.ErrInvitationInvalid) {
			t.Fatalf("expected audience mismatch rejection, got %v", err)
		}
		session, err := service.RedeemInvitation(ctx, first.Token, audience)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.RedeemInvitation(ctx, first.Token, audience); !errors.Is(err, evidence.ErrInvitationInvalid) {
			t.Fatalf("expected replay rejection, got %v", err)
		}
		if session.ExpiresAt.After(first.ExpiresAt) {
			t.Fatalf("session exceeded invitation expiry: %#v %#v", session, first)
		}
		if err := service.RevokeSession(ctx, "evidence-bank", session.SessionID); err != nil {
			t.Fatal(err)
		}
		if _, _, err := service.SessionRequest(ctx, session.SessionToken); !errors.Is(err, evidence.ErrSessionInvalid) {
			t.Fatalf("expected revoked session rejection, got %v", err)
		}

		const cancelledAudience = "cancel@example.com"
		cancelled, err := createExternalEvidenceRequest(ctx, service, now, "VENDOR", "vendor-cancelled", cancelledAudience)
		if err != nil {
			t.Fatal(err)
		}
		issued, err := service.IssueInvitation(ctx, evidence.IssueInvitationInput{TenantID: "evidence-bank", RequestID: cancelled.ID, Audience: cancelledAudience, Purpose: "Cancelled response", TTLMinutes: 60, CreatedBy: evidenceRequesterID})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE capture_requests SET status='CANCELLED',version=version+1 WHERE id=$1::uuid`, cancelled.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := service.RedeemInvitation(ctx, issued.Token, cancelledAudience); !errors.Is(err, evidence.ErrInvitationInvalid) {
			t.Fatalf("expected cancelled-request rejection, got %v", err)
		}
	})

	t.Run("capture relationships reject cross-tenant rows", func(t *testing.T) {
		request, err := createExternalEvidenceRequest(ctx, service, now, "VENDOR", "vendor-tenant-integrity", "tenant@example.com")
		if err != nil {
			t.Fatal(err)
		}
		_, err = pool.Exec(ctx, `INSERT INTO capture_invitations(id,tenant_id,request_id,token_hash,audience_hash,audience_hint,purpose,expires_at,created_at) VALUES('33333333-3333-7333-8333-333333333339'::uuid,$1::uuid,$2::uuid,decode(repeat('aa',32),'hex'),decode(repeat('bb',32),'hex'),'x***@example.com','Invalid tenant link',$3,$4)`, evidenceOtherTenantID, request.ID, now.Add(time.Hour), now)
		if err == nil {
			t.Fatal("expected cross-tenant invitation to be rejected")
		}
	})

	t.Run("artifact manifests preserve integrity and require assigned internal recipient", func(t *testing.T) {
		request, err := createInternalEvidenceRequest(ctx, service, now, "CONTROL", "control-artifact")
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := service.StoreArtifact(ctx, evidence.ArtifactInput{TenantID: "evidence-bank", RequestID: request.ID, FileName: "control.txt", MediaType: "text/plain", CreatedBy: evidenceRecipientID}, bytes.NewBufferString("current control evidence"))
		if err != nil {
			t.Fatal(err)
		}
		if artifact.Status != evidence.ArtifactStoredUnscanned || artifact.SHA256 == "" || artifact.SizeBytes == 0 {
			t.Fatalf("unexpected artifact %#v", artifact)
		}
		var storedStatus, storedHash string
		if err := pool.QueryRow(ctx, `SELECT status,sha256 FROM capture_artifacts WHERE id=$1::uuid`, artifact.ID).Scan(&storedStatus, &storedHash); err != nil {
			t.Fatal(err)
		}
		if storedStatus != "STORED_UNSCANNED" || storedHash != artifact.SHA256 {
			t.Fatalf("manifest mismatch %s %s", storedStatus, storedHash)
		}
	})
}

func createInternalEvidenceRequest(ctx context.Context, service *evidence.Service, now time.Time, subjectType, subjectID string) (evidence.Request, error) {
	return service.CreateRequest(ctx, evidence.CreateRequestInput{
		TenantID:         "evidence-bank",
		SubjectType:      subjectType,
		SubjectID:        subjectID,
		Title:            "Confirm current evidence",
		Purpose:          "Complete the current assurance review.",
		WhyYou:           "You are the assigned evidence owner.",
		Sensitivity:      "INTERNAL",
		AudienceType:     "INTERNAL",
		Recipient:        evidence.RecipientInput{Type: evidence.RecipientInternalPrincipal, PrincipalID: evidenceRecipientID},
		EstimatedMinutes: 2,
		Deadline:         now.Add(24 * time.Hour),
		KnownFacts:       map[string]string{"Scope": "Bank NG"},
		Fields:           []evidence.Field{{ID: "state", Label: "Current state", Type: "single_select", Required: true, Options: []string{"Operating", "Unavailable"}}},
		CreatedBy:        evidenceRequesterID,
	})
}

func createExternalEvidenceRequest(ctx context.Context, service *evidence.Service, now time.Time, subjectType, subjectID, audience string) (evidence.Request, error) {
	return service.CreateRequest(ctx, evidence.CreateRequestInput{
		TenantID:         "evidence-bank",
		SubjectType:      subjectType,
		SubjectID:        subjectID,
		Title:            "Confirm external evidence",
		Purpose:          "Complete the current assurance review.",
		WhyYou:           "You are the intended external respondent.",
		Sensitivity:      "CONFIDENTIAL",
		AudienceType:     "VENDOR",
		Recipient:        evidence.RecipientInput{Type: evidence.RecipientExternalAudience, Audience: audience},
		EstimatedMinutes: 2,
		Deadline:         now.Add(24 * time.Hour),
		KnownFacts:       map[string]string{"Scope": "Bank NG"},
		Fields:           []evidence.Field{{ID: "state", Label: "Current state", Type: "single_select", Required: true, Options: []string{"Operating", "Unavailable"}}},
		CreatedBy:        evidenceRequesterID,
	})
}
