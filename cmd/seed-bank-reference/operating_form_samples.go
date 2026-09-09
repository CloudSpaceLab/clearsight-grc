//go:build postgres

package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const operatingFormSamplePackage = "bank-operating-form-samples-v1"

type operatingFormSampleSpec struct {
	key, vendorRef, formKey, formCode, title, purpose, state string
	answers                                                  map[string]string
	totalFields                                              int
}

type operatingFormSampleReceipt struct {
	States map[string]string `json:"states"`
	Scores map[string]any    `json:"scores"`
}

func operatingFormSampleSpecs() []operatingFormSampleSpec {
	return []operatingFormSampleSpec{
		{key: "paywave-controls", vendorRef: "vendor:payment-switching", formKey: "vendor_control_attestation", formCode: "VENDOR-CONTROL-ATTESTATION", title: "Payment service control confirmation", purpose: "Confirm the controls used for the payment switching service.", state: "COMPLETED_HIGH", totalFields: 4, answers: map[string]string{"encryption_enabled": "Yes", "admin_mfa": "Yes", "annual_security_test": "Yes", "critical_weakness": "No"}},
		{key: "archiveguard-controls", vendorRef: "vendor:records-custody", formKey: "vendor_control_attestation", formCode: "VENDOR-CONTROL-ATTESTATION", title: "Records custody control confirmation", purpose: "Confirm current controls and identify weaknesses requiring review.", state: "COMPLETED_GAP", totalFields: 4, answers: map[string]string{"encryption_enabled": "Yes", "admin_mfa": "No", "annual_security_test": "No", "critical_weakness": "Yes"}},
		{key: "peoplelink-due-diligence", vendorRef: "vendor:payroll-processing", formKey: "vendor_due_diligence", formCode: "VENDOR-DUE-DILIGENCE", title: "Payroll service due diligence", purpose: "Complete the remaining security and privacy information for the payroll service.", state: "IN_PROGRESS", totalFields: 10, answers: map[string]string{"contact_email": "security@peoplelink.example.invalid", "service_description": "Payroll calculation and secure employee payslip delivery."}},
		{key: "sentinel-risk-review", vendorRef: "vendor:collections-platform", formKey: "vendor_compliance", formCode: "THIRD-PARTY-RISK-COMPLIANCE", title: "Collections platform risk review", purpose: "Complete the applicable third-party risk requirements for the collections platform.", state: "OUTSTANDING", totalFields: 26},
	}
}

func seedOperatingFormSamples(ctx context.Context, cfg config.Config, pool *pgxpool.Pool, seed bankverticals.SeedConfig, catalog bankverticals.OperatingDemoCatalog, vendors []thirdparty.Aggregate, monitoringRepo *monitoring.PostgresRepository, evidenceRepo *evidence.PostgresRepository) (operatingFormSampleReceipt, error) {
	result := operatingFormSampleReceipt{States: map[string]string{}, Scores: map[string]any{}}
	if len(cfg.RecipientSecurity.Keyring) == 0 || strings.TrimSpace(cfg.RecipientSecurity.ActiveKeyID) == "" || cfg.RecipientSecurity.AccessHMACKey == ([32]byte{}) {
		return result, fmt.Errorf("operating form samples require configured recipient security")
	}
	keyring, err := evidence.NewRecipientKeyring(cfg.RecipientSecurity.ActiveKeyID, cfg.RecipientSecurity.Keyring)
	if err != nil {
		return result, err
	}
	store := evidence.NewPostgresDistributionStore(evidenceRepo, keyring)
	distributions := evidence.NewDistributionService(store)
	access, err := evidence.NewDistributionAccessService(store, keyring, nil, cfg.RecipientSecurity.AccessHMACKey, cfg.CaptureSessionTTL)
	if err != nil {
		return result, err
	}
	forms := map[string]monitoring.FormTemplate{}
	for key, record := range catalog.Forms {
		form, formErr := monitoringRepo.ReusableFormRevision(ctx, seed.TenantID, seed.LegalEntityID, record.ID, record.Version)
		if formErr != nil {
			return result, formErr
		}
		forms[key] = form
	}
	byVendorRef := map[string]thirdparty.Aggregate{}
	for _, vendor := range vendors {
		byVendorRef[vendor.Relationship.ExternalRef] = vendor
	}
	for index, spec := range operatingFormSampleSpecs() {
		form, formOK := forms[spec.formKey]
		vendor, vendorOK := byVendorRef[spec.vendorRef]
		if !formOK || !vendorOK || form.Code != spec.formCode {
			return result, fmt.Errorf("operating form sample %s has unresolved form or vendor scope", spec.key)
		}
		state, score, seedErr := seedOperatingFormSample(ctx, pool, distributions, access, evidenceRepo, seed, form, vendor.Relationship.ID, spec, seed.Now.Add(time.Duration(index)*time.Minute))
		if seedErr != nil {
			return result, fmt.Errorf("seed %s: %w", spec.key, seedErr)
		}
		result.States[spec.key] = state
		if score != nil {
			result.Scores[spec.key] = score
		}
	}
	return result, nil
}

func seedOperatingFormSample(ctx context.Context, pool *pgxpool.Pool, distributions *evidence.DistributionService, access *evidence.DistributionAccessService, evidenceRepo *evidence.PostgresRepository, seed bankverticals.SeedConfig, form monitoring.FormTemplate, relationshipID string, spec operatingFormSampleSpec, now time.Time) (string, *evidence.ResponseScoreResult, error) {
	idempotencyKey := operatingFormSamplePackage + ":" + spec.key
	var existingID string
	err := pool.QueryRow(ctx, `SELECT r.distribution_id::text FROM capture_distribution_creation_receipts r JOIN tenants t ON t.id=r.tenant_id JOIN legal_entities le ON le.id=r.legal_entity_id WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND r.idempotency_key=$3`, seed.TenantID, seed.LegalEntityID, idempotencyKey).Scan(&existingID)
	if err == nil {
		return existingOperatingFormSample(ctx, pool, distributions, access, evidenceRepo, seed, existingID, spec)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", nil, err
	}
	bundle, err := distributions.Create(ctx, evidence.CreateDistributionInput{
		IdempotencyKey: idempotencyKey, TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID,
		FormTemplateID: form.ID, FormTemplateVersion: form.Version, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: relationshipID,
		Title: spec.title, Purpose: spec.purpose, AccessPolicy: evidence.AccessDirectMagicLink, EstimatedMinutes: 8,
		Deadline: now.Add(21 * 24 * time.Hour), RouteExpiresAt: now.Add(7 * 24 * time.Hour), CreatedBy: seed.ActorID,
		Recipients: []evidence.DistributionRecipientInput{{Role: evidence.RecipientTo, Type: evidence.RecipientExternalAudience, Address: spec.key + "@vendor.demo.invalid", AudienceHint: "Sample vendor contact", ContactLabel: "Vendor assurance contact"}},
	})
	if err != nil {
		return "", nil, err
	}
	if spec.state == "OUTSTANDING" {
		return string(evidence.RequestReady), nil, nil
	}
	routes, err := access.IssueDistributionAccessRoutes(ctx, seed.TenantID, seed.LegalEntityID, bundle.Distribution.ID, seed.ActorID)
	if err != nil || len(routes) != 1 {
		return "", nil, fmt.Errorf("issue sample response route: routes=%d err=%v", len(routes), err)
	}
	session, err := access.RedeemDirectRoute(ctx, routes[0].Selector)
	if err != nil {
		return "", nil, err
	}
	workspace, err := access.GetResponseWorkspace(ctx, session.SessionToken)
	if err != nil {
		return "", nil, err
	}
	edits := operatingFormSampleEdits(spec.answers, workspace.FieldSequences)
	workspace, err = access.SaveResponseWorkspace(ctx, session.SessionToken, evidence.SaveWorkspaceInput{ExpectedVersion: workspace.Workspace.Version, Edits: edits})
	if err != nil {
		return "", nil, err
	}
	if spec.state == "IN_PROGRESS" {
		if err := markOperatingFormSampleInProgress(ctx, pool, seed.TenantID, bundle.Recipients[0].RequestID, now); err != nil {
			return "", nil, err
		}
		return string(evidence.RequestInProgress), nil, nil
	}
	submitted, err := access.SubmitResponseWorkspace(ctx, session.SessionToken, evidence.SubmitWorkspaceInput{ExpectedVersion: workspace.Workspace.Version})
	if err != nil {
		return "", nil, err
	}
	if submitted.Revision.Score == nil || !submitted.Revision.Score.Final || submitted.Revision.Score.State != evidence.ResponseScoreFinal {
		return "", nil, fmt.Errorf("completed sample did not produce a final compliance score")
	}
	return string(evidence.RequestSubmitted), submitted.Revision.Score, nil
}

func existingOperatingFormSample(ctx context.Context, pool *pgxpool.Pool, distributions *evidence.DistributionService, access *evidence.DistributionAccessService, evidenceRepo *evidence.PostgresRepository, seed bankverticals.SeedConfig, distributionID string, spec operatingFormSampleSpec) (string, *evidence.ResponseScoreResult, error) {
	bundle, err := distributions.Get(ctx, seed.TenantID, seed.LegalEntityID, distributionID)
	if err != nil {
		return "", nil, err
	}
	if len(bundle.Recipients) != 1 {
		return "", nil, fmt.Errorf("existing sample has %d recipients", len(bundle.Recipients))
	}
	request, err := evidenceRepo.GetRequest(ctx, seed.TenantID, bundle.Recipients[0].RequestID)
	if err != nil {
		return "", nil, err
	}
	if request.Status != evidence.RequestReady && request.Status != evidence.RequestInProgress {
		return "", nil, fmt.Errorf("existing sample request is %s", request.Status)
	}
	revisions, err := distributions.ListResponseRevisions(ctx, seed.TenantID, seed.LegalEntityID, distributionID, 2)
	if err != nil {
		return "", nil, err
	}
	switch spec.state {
	case "COMPLETED_HIGH", "COMPLETED_GAP":
		if len(revisions) != 1 || !revisions[0].Current || revisions[0].Score == nil || !revisions[0].Score.Final {
			return "", nil, fmt.Errorf("existing completed sample has no current final scored response")
		}
		return string(evidence.RequestSubmitted), revisions[0].Score, nil
	case "IN_PROGRESS":
		var editCount int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_response_workspace_edits WHERE distribution_id=$1::uuid`, distributionID).Scan(&editCount); err != nil {
			return "", nil, err
		}
		if editCount == 0 {
			routes, routeErr := access.IssueDistributionAccessRoutes(ctx, seed.TenantID, seed.LegalEntityID, distributionID, seed.ActorID)
			if routeErr != nil || len(routes) != 1 {
				return "", nil, fmt.Errorf("repair sample response route: routes=%d err=%v", len(routes), routeErr)
			}
			session, routeErr := access.RedeemDirectRoute(ctx, routes[0].Selector)
			if routeErr != nil {
				return "", nil, routeErr
			}
			workspace, routeErr := access.GetResponseWorkspace(ctx, session.SessionToken)
			if routeErr != nil {
				return "", nil, routeErr
			}
			workspace, routeErr = access.SaveResponseWorkspace(ctx, session.SessionToken, evidence.SaveWorkspaceInput{ExpectedVersion: workspace.Workspace.Version, Edits: operatingFormSampleEdits(spec.answers, workspace.FieldSequences)})
			if routeErr != nil || len(workspace.Answers) == 0 {
				return "", nil, fmt.Errorf("repair partial sample response: answers=%d err=%v", len(workspace.Answers), routeErr)
			}
		}
		if err := markOperatingFormSampleInProgress(ctx, pool, seed.TenantID, request.ID, time.Now().UTC()); err != nil {
			return "", nil, err
		}
		return string(evidence.RequestInProgress), nil, nil
	default:
		return string(evidence.RequestReady), nil, nil
	}
}

func markOperatingFormSampleInProgress(ctx context.Context, pool *pgxpool.Pool, tenantID, requestID string, now time.Time) error {
	if _, err := pool.Exec(ctx, `UPDATE capture_requests r SET status='IN_PROGRESS',version=version+1,updated_at=$3 FROM tenants t WHERE r.tenant_id=t.id AND (t.id::text=$1 OR t.slug=$1) AND r.id=$2::uuid AND r.status='READY'`, tenantID, requestID, now.UTC()); err != nil {
		return err
	}
	var status evidence.RequestStatus
	err := pool.QueryRow(ctx, `SELECT r.status FROM capture_requests r JOIN tenants t ON t.id=r.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND r.id=$2::uuid`, tenantID, requestID).Scan(&status)
	if err != nil {
		return err
	}
	if status != evidence.RequestInProgress {
		return fmt.Errorf("partial sample request is %s", status)
	}
	return nil
}

func operatingFormSampleEdits(answers map[string]string, sequences map[string]int64) []evidence.FieldEdit {
	fieldIDs := make([]string, 0, len(answers))
	for fieldID := range answers {
		fieldIDs = append(fieldIDs, fieldID)
	}
	sort.Strings(fieldIDs)
	edits := make([]evidence.FieldEdit, 0, len(fieldIDs))
	for _, fieldID := range fieldIDs {
		edits = append(edits, evidence.FieldEdit{FieldID: fieldID, Value: formcontract.TextAnswer(answers[fieldID]), BaseSequence: sequences[fieldID]})
	}
	return edits
}
