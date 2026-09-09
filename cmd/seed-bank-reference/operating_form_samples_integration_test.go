//go:build postgres && postgresintegration

package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func TestCloudspaceRiskRegisterSampleIsSubmittedAndRepeatSafe(t *testing.T) {
	pool, cfg, seed := sampleTestSetup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	seed.Now = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	seed.SignatoryPrincipalID = "00000000-0000-4000-8000-000000000102"

	continuityRepo := continuity.NewPostgresRepository(pool)
	continuityService := continuity.NewServiceWithClock(continuityRepo, func() time.Time { return seed.Now })
	evidenceRepo := evidence.NewPostgresRepository(pool)
	evidenceService := evidence.NewService(evidenceRepo, evidence.NewMemoryObjectStore())
	monitoringRepo := monitoring.NewPostgresRepository(pool)
	installer := bankverticals.NewService(continuityService, evidenceService)
	installer.ConfigureMonitoring(monitoring.NewService(monitoringRepo, evidenceService))
	thirdParties := thirdparty.NewService(thirdparty.NewPostgresRepository(pool))
	actor := thirdparty.Actor{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, PrincipalID: seed.OwnerPrincipalID}
	existing, err := thirdParties.CreateRelationship(ctx, actor, thirdparty.CreateRelationshipInput{
		LegalName: "Cloudspace Technologies Ltd", ServiceName: "OEM", Criticality: thirdparty.CriticalityStandard, PrivacyRole: thirdparty.PrivacyProcessor,
	})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := installer.EnsureOperatingDemo(ctx, seed)
	if err != nil {
		t.Fatal(err)
	}
	vendors, err := installer.EnsureOperatingVendors(ctx, seed, thirdParties)
	if err != nil {
		t.Fatal(err)
	}
	for _, vendor := range vendors {
		if vendor.Vendor.LegalName == "Cloudspace Technologies Ltd" && vendor.Relationship.ServiceName == "OEM" && vendor.Relationship.ID != existing.Relationship.ID {
			t.Fatalf("Cloudspace relationship=%q, want existing %q", vendor.Relationship.ID, existing.Relationship.ID)
		}
	}
	legacyForm, err := monitoringRepo.ReusableFormRevision(ctx, seed.TenantID, seed.LegalEntityID, catalog.Forms["vendor_due_diligence"].ID, catalog.Forms["vendor_due_diligence"].Version)
	if err != nil {
		t.Fatal(err)
	}
	keyring, err := evidence.NewRecipientKeyring(cfg.RecipientSecurity.ActiveKeyID, cfg.RecipientSecurity.Keyring)
	if err != nil {
		t.Fatal(err)
	}
	legacyStore := evidence.NewPostgresDistributionStore(evidenceRepo, keyring)
	legacyDistributions := evidence.NewDistributionService(legacyStore)
	legacyAccess, err := evidence.NewDistributionAccessService(legacyStore, keyring, nil, cfg.RecipientSecurity.AccessHMACKey, cfg.CaptureSessionTTL)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := legacyDistributions.Create(ctx, evidence.CreateDistributionInput{
		TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, FormTemplateID: legacyForm.ID, FormTemplateVersion: legacyForm.Version,
		SubjectType: "VENDOR_RELATIONSHIP", SubjectID: existing.Relationship.ID, Title: "Vendor security and privacy review", Purpose: "Collect vendor security information.",
		AccessPolicy: evidence.AccessDirectMagicLink, EstimatedMinutes: 8, Deadline: seed.Now.Add(21 * 24 * time.Hour), RouteExpiresAt: seed.Now.Add(7 * 24 * time.Hour), CreatedBy: seed.ActorID,
		Recipients: []evidence.DistributionRecipientInput{{Role: evidence.RecipientTo, Type: evidence.RecipientExternalAudience, Address: "legacy-cloudspace@vendor.demo.invalid", AudienceHint: "Vendor assurance contact", ContactLabel: "Vendor assurance contact"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = legacyAccess.IssueDistributionAccessRoutes(ctx, seed.TenantID, seed.LegalEntityID, legacy.Distribution.ID, seed.ActorID); err != nil {
		t.Fatal(err)
	}
	draft := legacyForm
	draft.Version++
	draft.Status = monitoring.LifecycleDraft
	draft.IsCurrent = false
	draft.CreatedBy = seed.OwnerPrincipalID
	draft.SubmittedBy, draft.ApprovedBy = "", ""
	draft.CreatedAt, draft.UpdatedAt = seed.Now, seed.Now
	draft, err = monitoringRepo.CreateFormRevision(ctx, draft)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := monitoringRepo.TransitionForm(ctx, monitoring.LifecycleTransition{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, ProgramID: draft.ProgramID, ID: draft.ID, ExpectedVersion: draft.Version, To: monitoring.LifecyclePendingApproval, ActorID: seed.OwnerPrincipalID, At: seed.Now})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = monitoringRepo.TransitionForm(ctx, monitoring.LifecycleTransition{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, ProgramID: pending.ProgramID, ID: pending.ID, ExpectedVersion: pending.Version, To: monitoring.LifecycleActive, ActorID: seed.ReviewerPrincipalID, At: seed.Now}); err != nil {
		t.Fatal(err)
	}
	// The confirmed production request is an older revision 3. The active form above
	// advances to revision 4; retain the historical source version on the open request.
	if _, err = pool.Exec(ctx, `UPDATE capture_form_distributions SET form_template_version=$2 WHERE id=$1::uuid`, legacy.Distribution.ID, cloudspaceLegacyFormVersion); err != nil {
		t.Fatal(err)
	}

	first, err := seedOperatingFormSamples(ctx, cfg, pool, seed, catalog, vendors, monitoringRepo, evidenceRepo)
	if err != nil {
		t.Fatal(err)
	}
	if first.States["cloudspace-oem-risk-register"] != string(evidence.RequestSubmitted) {
		t.Fatalf("Cloudspace state=%q", first.States["cloudspace-oem-risk-register"])
	}

	var distributionID string
	if err = pool.QueryRow(ctx, `SELECT d.id::text FROM capture_form_distributions d JOIN capture_requests r ON r.tenant_id=d.tenant_id AND r.distribution_id=d.id JOIN capture_submissions s ON s.tenant_id=r.tenant_id AND s.request_id=r.id WHERE d.tenant_id=$1::uuid AND d.legal_entity_id=$2::uuid AND d.subject_id=$3::uuid AND r.status='SUBMITTED' AND s.answers->'assurance_gap'->>'text' LIKE 'Sample register findings:%'`, seed.TenantID, seed.LegalEntityID, existing.Relationship.ID).Scan(&distributionID); err != nil {
		t.Fatal(err)
	}
	if distributionID == legacy.Distribution.ID {
		t.Fatal("Cloudspace legacy request was not superseded")
	}
	var legacyStatus string
	if err = pool.QueryRow(ctx, `SELECT status FROM capture_form_distributions WHERE id=$1::uuid`, legacy.Distribution.ID).Scan(&legacyStatus); err != nil {
		t.Fatal(err)
	}
	if legacyStatus != string(evidence.DistributionSuperseded) {
		t.Fatalf("legacy Cloudspace distribution status=%q", legacyStatus)
	}
	var fixtureReceipts int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM capture_distribution_creation_receipts WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND idempotency_key=$3 AND distribution_id=$4::uuid`, seed.TenantID, seed.LegalEntityID, operatingFormSamplePackage+":cloudspace-oem-risk-register", distributionID).Scan(&fixtureReceipts); err != nil {
		t.Fatal(err)
	}
	if fixtureReceipts != 1 {
		t.Fatalf("Cloudspace fixture receipts=%d, want 1", fixtureReceipts)
	}
	var answersRaw []byte
	var revisions, currentRevisions int
	var scoreState string
	if err = pool.QueryRow(ctx, `SELECT s.answers, (SELECT count(*) FROM capture_response_revisions WHERE distribution_id=$1::uuid), (SELECT count(*) FROM capture_response_revisions WHERE distribution_id=$1::uuid AND is_current), (SELECT score_state FROM capture_response_revisions WHERE distribution_id=$1::uuid AND is_current) FROM capture_submissions s WHERE s.distribution_id=$1::uuid`, distributionID).Scan(&answersRaw, &revisions, &currentRevisions, &scoreState); err != nil {
		t.Fatal(err)
	}
	if revisions != 1 || currentRevisions != 1 || scoreState != string(evidence.ResponseScoreNotConfigured) {
		t.Fatalf("Cloudspace revisions=%d current=%d score=%q", revisions, currentRevisions, scoreState)
	}
	var answers map[string]formcontract.AnswerValue
	if err = json.Unmarshal(answersRaw, &answers); err != nil {
		t.Fatal(err)
	}
	if values := answers["data_classes"].Values; len(values) != 1 || values[0] != "Payment data" {
		t.Fatalf("payment data answer=%+v", answers["data_classes"])
	}
	gap, ok := answers["assurance_gap"].ScalarText()
	if !ok {
		t.Fatal("missing assurance gap")
	}
	for _, want := range []string{"ISO 27001", "ISO 22301", "VAPT", "right-to-audit", "PCI-DSS certificate was expired", "31 March 2026"} {
		if !strings.Contains(gap, want) {
			t.Fatalf("assurance gap missing %q: %s", want, gap)
		}
	}

	seed.Now = seed.Now.Add(24 * time.Hour)
	second, err := seedOperatingFormSamples(ctx, cfg, pool, seed, catalog, vendors, monitoringRepo, evidenceRepo)
	if err != nil {
		t.Fatal(err)
	}
	if second.States["cloudspace-oem-risk-register"] != string(evidence.RequestSubmitted) {
		t.Fatalf("Cloudspace rerun state=%q", second.States["cloudspace-oem-risk-register"])
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM capture_response_revisions WHERE distribution_id=$1::uuid`, distributionID).Scan(&revisions); err != nil {
		t.Fatal(err)
	}
	if revisions != 1 {
		t.Fatalf("Cloudspace response revisions after rerun=%d, want 1", revisions)
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_submissions SET answers=jsonb_set(answers,'{assurance_gap}',jsonb_build_object('text','Altered sample response')) WHERE distribution_id=$1::uuid`, distributionID); err != nil {
		t.Fatal(err)
	}
	if _, err = seedOperatingFormSamples(ctx, cfg, pool, seed, catalog, vendors, monitoringRepo, evidenceRepo); err == nil {
		t.Fatal("expected altered Cloudspace sample response to be rejected")
	}
}
