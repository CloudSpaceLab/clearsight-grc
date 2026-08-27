//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresLifecycleActorsPersistAndReconstruct(t *testing.T) {
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

	const (
		tenantID    = "99999999-9999-7999-8999-999999999991"
		proposer    = "99999999-9999-7999-8999-999999999992"
		reviewer    = "99999999-9999-7999-8999-999999999993"
		challenger  = "99999999-9999-7999-8999-999999999994"
		authorizer  = "99999999-9999-7999-8999-999999999995"
		preparer    = "99999999-9999-7999-8999-999999999996"
		signatory   = "99999999-9999-7999-8999-999999999997"
		transmitter = "99999999-9999-7999-8999-999999999998"
		ackRecorder = "99999999-9999-7999-8999-999999999999"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'lifecycle-actor-test','Lifecycle Actor Test')`, tenantID); err != nil {
		t.Fatal(err)
	}
	entityID := seedPostgresTestLegalEntity(t, ctx, pool, tenantID, "ENTITY-A")
	ctx = WithTrustedSystemEntityScope(ctx, "lifecycle-actor-test", entityID)
	for _, principal := range []string{proposer, reviewer, challenger, authorizer, preparer, signatory, transmitter, ackRecorder} {
		if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($1::uuid,$2::uuid,'PERSON',$1::text,'Lifecycle actor')`, principal, tenantID); err != nil {
			t.Fatal(err)
		}
	}

	repo := NewPostgresRepository(pool)
	service := NewService(repo)
	now := time.Now().UTC().Truncate(time.Second)
	service.now = func() time.Time { return now }
	matter, err := service.CreateMatter(ctx, CreateMatterInput{TenantID: "lifecycle-actor-test", LegalEntityID: entityID, Type: MatterAuthorityRequest, Priority: 4, Title: "Lifecycle actor persistence", Summary: "Persist every actor in the command lifecycle.", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}

	decisionSteps := []struct {
		status DecisionStatus
		actor  string
	}{
		{DecisionProposed, proposer},
		{DecisionInReview, reviewer},
		{DecisionChallenged, challenger},
		{DecisionApproved, authorizer},
	}
	for _, step := range decisionSteps {
		input := AddDecisionInput{TenantID: "lifecycle-actor-test", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Type: "REGULATORY_POSITION", Status: step.status, Rationale: "Lifecycle progression.", AuthorityPrincipalID: step.actor}
		if step.status == DecisionApproved {
			input.SelectedOption = "APPROVE"
		}
		matter, err = service.RecordDecisionLifecycle(ctx, input)
		if err != nil {
			t.Fatalf("record %s: %v", step.status, err)
		}
	}

	var proposedBy, reviewedBy, challengedBy, approvedBy string
	if err := pool.QueryRow(ctx, `SELECT COALESCE(max(proposed_by::text),''), COALESCE(max(reviewed_by::text),''), COALESCE(max(challenged_by::text),''), COALESCE(max(authority_principal_id::text),'') FROM matter_decisions WHERE matter_id=$1::uuid`, matter.Matter.ID).Scan(&proposedBy, &reviewedBy, &challengedBy, &approvedBy); err != nil {
		t.Fatal(err)
	}
	if proposedBy != proposer || reviewedBy != reviewer || challengedBy != challenger || approvedBy != authorizer {
		t.Fatalf("unexpected persisted decision actors proposer=%s reviewer=%s challenger=%s authorizer=%s", proposedBy, reviewedBy, challengedBy, approvedBy)
	}
	currentDecision := CurrentDecisionForType(matter.Decisions, "REGULATORY_POSITION")
	if currentDecision == nil || currentDecision.AuthorityPrincipalID != authorizer {
		t.Fatalf("current decision did not reconstruct authorizer: %#v", currentDecision)
	}

	matter, err = service.AddResponsePackage(ctx, AddResponsePackageInput{TenantID: "lifecycle-actor-test", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Purpose: "Provide requested records", Audience: "Regulator", Manifest: json.RawMessage(`[]`), ActorID: preparer})
	if err != nil {
		t.Fatal(err)
	}
	responseID := matter.ResponsePackages[0].ID
	for _, step := range []struct {
		status ResponseStatus
		actor  string
	}{
		{ResponseInReview, reviewer},
		{ResponseApproved, signatory},
		{ResponseTransmitted, transmitter},
		{ResponseAcknowledged, ackRecorder},
	} {
		matter, err = service.TransitionResponsePackage(ctx, TransitionResponseInput{TenantID: "lifecycle-actor-test", MatterID: matter.Matter.ID, ResponseID: responseID, ExpectedVersion: matter.Matter.Version, To: step.status, ActorID: step.actor, Rationale: "Lifecycle progression."})
		if err != nil {
			t.Fatalf("response %s: %v", step.status, err)
		}
	}

	var preparedBy, responseReviewedBy, transmittedBy, acknowledgedBy string
	if err := pool.QueryRow(ctx, `SELECT COALESCE(prepared_by::text,''), COALESCE(reviewed_by::text,''), COALESCE(transmitted_by::text,''), COALESCE(acknowledged_by::text,'') FROM response_packages WHERE id=$1::uuid`, responseID).Scan(&preparedBy, &responseReviewedBy, &transmittedBy, &acknowledgedBy); err != nil {
		t.Fatal(err)
	}
	if preparedBy != preparer || responseReviewedBy != reviewer || transmittedBy != transmitter || acknowledgedBy != ackRecorder {
		t.Fatalf("unexpected persisted response actors preparer=%s reviewer=%s transmitter=%s acknowledger=%s", preparedBy, responseReviewedBy, transmittedBy, acknowledgedBy)
	}
	response := matter.ResponsePackages[0]
	if response.PreparedBy != preparer || response.ReviewedBy != reviewer || response.ApprovedBy != signatory || response.TransmittedBy != transmitter || response.AcknowledgedBy != ackRecorder {
		t.Fatalf("response actors did not reconstruct from events: %#v", response)
	}
}
