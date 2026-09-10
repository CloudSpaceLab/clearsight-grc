//go:build postgres

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
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

const (
	cloudspaceLegacyFormVersion = 3
	cloudspaceLegacyFormTitle   = "Vendor security and privacy review"
)

type operatingFormSampleSpec struct {
	key, vendorRef, formKey, formCode, title, purpose, state string
	answers                                                  map[string]formcontract.AnswerValue
	totalFields                                              int
}

type operatingFormSampleReceipt struct {
	States map[string]string `json:"states"`
	Scores map[string]any    `json:"scores"`
}

func operatingFormSampleSpecs() []operatingFormSampleSpec {
	return []operatingFormSampleSpec{
		{key: "paywave-controls", vendorRef: "vendor:payment-switching", formKey: "vendor_control_attestation", formCode: "VENDOR-CONTROL-ATTESTATION", title: "Payment service control confirmation", purpose: "Confirm the controls used for the payment switching service.", state: "COMPLETED_HIGH", totalFields: 4, answers: formcontract.TextAnswers(map[string]string{"encryption_enabled": "Yes", "admin_mfa": "Yes", "annual_security_test": "Yes", "critical_weakness": "No"})},
		{key: "archiveguard-controls", vendorRef: "vendor:records-custody", formKey: "vendor_control_attestation", formCode: "VENDOR-CONTROL-ATTESTATION", title: "Records custody control confirmation", purpose: "Confirm current controls and identify weaknesses requiring review.", state: "COMPLETED_GAP", totalFields: 4, answers: formcontract.TextAnswers(map[string]string{"encryption_enabled": "Yes", "admin_mfa": "No", "annual_security_test": "No", "critical_weakness": "Yes"})},
		{key: "peoplelink-due-diligence", vendorRef: "vendor:payroll-processing", formKey: "vendor_due_diligence", formCode: "VENDOR-DUE-DILIGENCE", title: "Payroll service due diligence", purpose: "Complete the remaining security and privacy information for the payroll service.", state: "IN_PROGRESS", totalFields: 10, answers: formcontract.TextAnswers(map[string]string{"contact_email": "security@peoplelink.example.invalid", "service_description": "Payroll calculation and secure employee payslip delivery."})},
		{key: "sentinel-risk-review", vendorRef: "vendor:collections-platform", formKey: "vendor_compliance", formCode: "THIRD-PARTY-RISK-COMPLIANCE", title: "Collections platform risk review", purpose: "Complete the applicable third-party risk requirements for the collections platform.", state: "OUTSTANDING", totalFields: 26},
		{key: "cloudspace-oem-risk-register", vendorRef: "vendor:cloudspace-oem", formKey: "vendor_due_diligence", formCode: "VENDOR-DUE-DILIGENCE", title: "Cloudspace OEM sample security and privacy review", purpose: "Record declared assurance gaps from the supplied sample third-party risk register.", state: "COMPLETED_UNREVIEWED", totalFields: 10, answers: map[string]formcontract.AnswerValue{
			"contact_email":          formcontract.TextAnswer("assurance@cloudspace.sample.invalid"),
			"service_description":    formcontract.TextAnswer("Moneytor GetPaid application and payment terminal service provider work for POS Business."),
			"data_classes":           {Values: []string{"Payment data"}},
			"subprocessors":          formcontract.TextAnswer("No"),
			"security_framework":     formcontract.TextAnswer("ISO 27001"),
			"assurance_available":    formcontract.TextAnswer("No"),
			"assurance_gap":          formcontract.TextAnswer("Sample register findings: ISO 27001 and ISO 22301 assurance was not provided. Cloudspace stated it was undergoing a surveillance audit and committed to close the certifications in Q1 2026. VAPT was not provided; Cloudspace stated it was in progress and would be shared later. The SLA did not include a right-to-audit clause; the business team prepared an addendum. The PCI-DSS certificate was expired; PCI DSS v4.0.1 was recommended. Target date: 31 March 2026. The register does not provide contact or subprocessor information; those fields are sample assumptions."),
			"authorized_attestation": formcontract.TextAnswer("Yes"),
		}},
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
		if strings.EqualFold(strings.TrimSpace(vendor.Vendor.LegalName), "Cloudspace Technologies Ltd") && strings.EqualFold(strings.TrimSpace(vendor.Relationship.ServiceName), "OEM") {
			byVendorRef["vendor:cloudspace-oem"] = vendor
		}
	}
	for index, spec := range operatingFormSampleSpecs() {
		form, formOK := forms[spec.formKey]
		vendor, vendorOK := byVendorRef[spec.vendorRef]
		if !formOK || !vendorOK || form.Code != spec.formCode {
			return result, fmt.Errorf("operating form sample %s has unresolved form or vendor scope", spec.key)
		}
		state, score, seedErr := seedOperatingFormSample(ctx, pool, distributions, access, evidenceRepo, monitoringRepo, seed, form, vendor.Relationship.ID, spec, seed.Now.Add(time.Duration(index)*time.Minute))
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

func seedOperatingFormSample(ctx context.Context, pool *pgxpool.Pool, distributions *evidence.DistributionService, access *evidence.DistributionAccessService, evidenceRepo *evidence.PostgresRepository, monitoringRepo *monitoring.PostgresRepository, seed bankverticals.SeedConfig, form monitoring.FormTemplate, relationshipID string, spec operatingFormSampleSpec, now time.Time) (string, *evidence.ResponseScoreResult, error) {
	idempotencyKey := operatingFormSamplePackage + ":" + spec.key
	var existingID string
	err := pool.QueryRow(ctx, `SELECT r.distribution_id::text FROM capture_distribution_creation_receipts r JOIN tenants t ON t.id=r.tenant_id JOIN legal_entities le ON le.id=r.legal_entity_id WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND r.idempotency_key=$3`, seed.TenantID, seed.LegalEntityID, idempotencyKey).Scan(&existingID)
	hasFixtureReceipt := err == nil
	if hasFixtureReceipt && spec.key != "cloudspace-oem-risk-register" {
		return existingOperatingFormSample(ctx, pool, distributions, access, evidenceRepo, seed, form, relationshipID, existingID, spec)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", nil, err
	}
	if spec.key == "cloudspace-oem-risk-register" {
		replacement, replacementForm, found, findErr := findCloudspaceSupersessionReplacement(ctx, pool, distributions, monitoringRepo, seed, relationshipID, idempotencyKey)
		if findErr != nil {
			return "", nil, findErr
		}
		if found {
			// Workspace submissions retain an open request so respondents can
			// submit later revisions. Existing revisions prove completion.
			revisions, revisionErr := distributions.ListResponseRevisions(ctx, seed.TenantID, seed.LegalEntityID, replacement.Distribution.ID, 2)
			if revisionErr != nil {
				return "", nil, revisionErr
			}
			if len(revisions) > 0 {
				return existingOperatingFormSample(ctx, pool, distributions, access, evidenceRepo, seed, replacementForm, relationshipID, replacement.Distribution.ID, spec)
			}
			request, requestErr := evidenceRepo.GetRequest(ctx, seed.TenantID, replacement.Recipients[0].RequestID)
			if requestErr != nil {
				return "", nil, requestErr
			}
			switch request.Status {
			case evidence.RequestSubmitted:
				return existingOperatingFormSample(ctx, pool, distributions, access, evidenceRepo, seed, replacementForm, relationshipID, replacement.Distribution.ID, spec)
			case evidence.RequestReady, evidence.RequestInProgress:
				return submitOperatingFormSample(ctx, pool, access, seed, replacement, spec, now, nil)
			default:
				return "", nil, fmt.Errorf("Cloudspace supersession replacement request is %s", request.Status)
			}
		}
		migrated, migratedForm, found, findErr := findMigratedCloudspaceSample(ctx, pool, monitoringRepo, seed, relationshipID)
		if findErr != nil {
			return "", nil, findErr
		}
		if found {
			return existingOperatingFormSample(ctx, pool, distributions, access, evidenceRepo, seed, migratedForm, relationshipID, migrated, spec)
		}
		legacy, replacementForm, found, findErr := findCloudspaceLegacyRequest(ctx, pool, distributions, monitoringRepo, seed, relationshipID)
		if findErr != nil {
			return "", nil, findErr
		}
		if found {
			replaced, replaceErr := access.SupersedeDistribution(ctx, seed.TenantID, seed.LegalEntityID, legacy.Distribution.ID, evidence.SupersedeDistributionInput{
				ExpectedVersion: legacy.Distribution.Version, ExpectedWorkspaceVersion: legacy.Workspace.Version, TargetFormVersion: replacementForm.Version,
				IdempotencyKey: idempotencyKey, CarryForward: false, ActorID: seed.ActorID,
			})
			if replaceErr != nil {
				return "", nil, fmt.Errorf("supersede Cloudspace legacy request: %w", replaceErr)
			}
			if replaced.Replacement.Distribution.FormTemplateID != replacementForm.ID || replaced.Replacement.Distribution.FormTemplateVersion != replacementForm.Version {
				return "", nil, fmt.Errorf("Cloudspace replacement did not use the current vendor form")
			}
			return submitOperatingFormSample(ctx, pool, access, seed, replaced.Replacement, spec, now, replaced.IssuedRoutes)
		}
		foreign, foreignErr := hasCloudspaceSupersession(ctx, pool, seed, relationshipID)
		if foreignErr != nil {
			return "", nil, foreignErr
		}
		if foreign {
			return "", nil, fmt.Errorf("Cloudspace legacy request has a replacement outside the sample fixture")
		}
		if hasFixtureReceipt {
			return existingOperatingFormSample(ctx, pool, distributions, access, evidenceRepo, seed, form, relationshipID, existingID, spec)
		}
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
	return submitOperatingFormSample(ctx, pool, access, seed, bundle, spec, now, nil)
}

func submitOperatingFormSample(ctx context.Context, pool *pgxpool.Pool, access *evidence.DistributionAccessService, seed bankverticals.SeedConfig, bundle evidence.DistributionBundle, spec operatingFormSampleSpec, now time.Time, routes []evidence.IssuedAccessRoute) (string, *evidence.ResponseScoreResult, error) {
	if spec.state == "OUTSTANDING" {
		return string(evidence.RequestReady), nil, nil
	}
	var err error
	if len(routes) == 0 {
		routes, err = access.IssueDistributionAccessRoutes(ctx, seed.TenantID, seed.LegalEntityID, bundle.Distribution.ID, seed.ActorID)
	}
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
	if spec.state == "COMPLETED_UNREVIEWED" {
		if submitted.Revision.Score == nil || submitted.Revision.Score.Final || submitted.Revision.Score.State != evidence.ResponseScoreNotConfigured {
			return "", nil, fmt.Errorf("unreviewed sample did not preserve an unscored response")
		}
		return string(evidence.RequestSubmitted), submitted.Revision.Score, nil
	}
	if submitted.Revision.Score == nil || !submitted.Revision.Score.Final || submitted.Revision.Score.State != evidence.ResponseScoreFinal {
		return "", nil, fmt.Errorf("completed sample did not produce a final compliance score")
	}
	return string(evidence.RequestSubmitted), submitted.Revision.Score, nil
}

func findMigratedCloudspaceSample(ctx context.Context, pool *pgxpool.Pool, monitoringRepo *monitoring.PostgresRepository, seed bankverticals.SeedConfig, relationshipID string) (string, monitoring.FormTemplate, bool, error) {
	rows, err := pool.Query(ctx, `
		SELECT d.id::text,d.form_template_id::text,d.form_template_version
		FROM capture_form_distributions d
		JOIN tenants t ON t.id=d.tenant_id
		JOIN capture_requests r ON r.tenant_id=d.tenant_id AND r.distribution_id=d.id
		JOIN monitoring_form_templates f ON f.tenant_id=d.tenant_id AND f.id=d.form_template_id AND f.version=d.form_template_version
		JOIN capture_submissions s ON s.tenant_id=r.tenant_id AND s.request_id=r.id
		JOIN capture_response_revisions revision ON revision.tenant_id=s.tenant_id AND revision.distribution_id=d.id AND revision.submission_id=s.id AND revision.is_current
		WHERE (t.id::text=$1 OR t.slug=$1) AND d.legal_entity_id=$2::uuid AND d.subject_type='VENDOR_RELATIONSHIP' AND d.subject_id=$3::uuid
		  AND f.code='VENDOR-DUE-DILIGENCE' AND r.status IN ('READY','IN_PROGRESS','SUBMITTED') AND d.status NOT IN ('REVOKED','SUPERSEDED')
		  AND COALESCE(s.answers->'assurance_gap'->>'text','') LIKE 'Sample register findings:%'
		ORDER BY d.updated_at DESC,d.id DESC LIMIT 2`, seed.TenantID, seed.LegalEntityID, relationshipID)
	if err != nil {
		return "", monitoring.FormTemplate{}, false, err
	}
	defer rows.Close()
	type candidate struct {
		distributionID, formID string
		version                int64
	}
	values := []candidate{}
	for rows.Next() {
		var value candidate
		if err := rows.Scan(&value.distributionID, &value.formID, &value.version); err != nil {
			return "", monitoring.FormTemplate{}, false, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return "", monitoring.FormTemplate{}, false, err
	}
	if len(values) > 1 {
		return "", monitoring.FormTemplate{}, false, fmt.Errorf("Cloudspace sample response is ambiguous")
	}
	if len(values) == 0 {
		return "", monitoring.FormTemplate{}, false, nil
	}
	form, err := monitoringRepo.ReusableFormRevision(ctx, seed.TenantID, seed.LegalEntityID, values[0].formID, values[0].version)
	if err != nil {
		return "", monitoring.FormTemplate{}, false, err
	}
	return values[0].distributionID, form, true, nil
}

func findCloudspaceLegacyRequest(ctx context.Context, pool *pgxpool.Pool, distributions *evidence.DistributionService, monitoringRepo *monitoring.PostgresRepository, seed bankverticals.SeedConfig, relationshipID string) (evidence.DistributionBundle, monitoring.FormTemplate, bool, error) {
	rows, err := pool.Query(ctx, `
		SELECT d.id::text,d.form_template_id::text,d.form_template_version
		FROM capture_form_distributions d
		JOIN tenants t ON t.id=d.tenant_id
		JOIN capture_requests r ON r.tenant_id=d.tenant_id AND r.distribution_id=d.id
		JOIN monitoring_form_templates f ON f.tenant_id=d.tenant_id AND f.id=d.form_template_id AND f.version=d.form_template_version
		WHERE (t.id::text=$1 OR t.slug=$1) AND d.legal_entity_id=$2::uuid AND d.subject_type='VENDOR_RELATIONSHIP' AND d.subject_id=$3::uuid
		  AND f.code='VENDOR-DUE-DILIGENCE' AND d.form_template_version=$4 AND d.title=$5
		  AND d.status IN ('OPEN','LOCKED') AND r.status IN ('READY','IN_PROGRESS')
		ORDER BY d.updated_at DESC,d.id DESC LIMIT 2`, seed.TenantID, seed.LegalEntityID, relationshipID, cloudspaceLegacyFormVersion, cloudspaceLegacyFormTitle)
	if err != nil {
		return evidence.DistributionBundle{}, monitoring.FormTemplate{}, false, err
	}
	defer rows.Close()
	type candidate struct {
		distributionID, formID string
		version                int64
	}
	values := []candidate{}
	for rows.Next() {
		var value candidate
		if err := rows.Scan(&value.distributionID, &value.formID, &value.version); err != nil {
			return evidence.DistributionBundle{}, monitoring.FormTemplate{}, false, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return evidence.DistributionBundle{}, monitoring.FormTemplate{}, false, err
	}
	if len(values) > 1 {
		return evidence.DistributionBundle{}, monitoring.FormTemplate{}, false, fmt.Errorf("Cloudspace legacy request is ambiguous")
	}
	if len(values) == 0 {
		return evidence.DistributionBundle{}, monitoring.FormTemplate{}, false, nil
	}
	current, err := monitoringRepo.ListReusableFormRevisions(ctx, seed.TenantID, seed.LegalEntityID, 100)
	if err != nil {
		return evidence.DistributionBundle{}, monitoring.FormTemplate{}, false, err
	}
	var replacement monitoring.FormTemplate
	for _, form := range current {
		if form.ID == values[0].formID && form.Code == "VENDOR-DUE-DILIGENCE" {
			replacement = form
			break
		}
	}
	if replacement.ID == "" || replacement.Version <= values[0].version {
		return evidence.DistributionBundle{}, monitoring.FormTemplate{}, false, fmt.Errorf("Cloudspace legacy request has no current governed form revision")
	}
	bundle, err := distributions.Get(ctx, seed.TenantID, seed.LegalEntityID, values[0].distributionID)
	if err != nil {
		return evidence.DistributionBundle{}, monitoring.FormTemplate{}, false, err
	}
	return bundle, replacement, true, nil
}

func findCloudspaceSupersessionReplacement(ctx context.Context, pool *pgxpool.Pool, distributions *evidence.DistributionService, monitoringRepo *monitoring.PostgresRepository, seed bankverticals.SeedConfig, relationshipID, idempotencyKey string) (evidence.DistributionBundle, monitoring.FormTemplate, bool, error) {
	rows, err := pool.Query(ctx, `
		SELECT replacement.id::text,replacement.form_template_id::text,replacement.form_template_version
		FROM capture_form_distributions legacy
		JOIN tenants t ON t.id=legacy.tenant_id
		JOIN monitoring_form_templates legacy_form ON legacy_form.tenant_id=legacy.tenant_id AND legacy_form.id=legacy.form_template_id AND legacy_form.version=legacy.form_template_version
		JOIN capture_distribution_events supersession ON supersession.tenant_id=legacy.tenant_id AND supersession.distribution_id=legacy.id AND supersession.event_type='FORM_DISTRIBUTION_SUPERSEDED'
		JOIN capture_form_distributions replacement ON replacement.tenant_id=legacy.tenant_id AND replacement.legal_entity_id=legacy.legal_entity_id AND replacement.id=(supersession.payload->>'superseded_by_distribution_id')::uuid
		JOIN capture_distribution_creation_receipts fixture ON fixture.tenant_id=replacement.tenant_id AND fixture.legal_entity_id=replacement.legal_entity_id AND fixture.distribution_id=replacement.id AND fixture.idempotency_key=$6
		JOIN capture_requests replacement_request ON replacement_request.tenant_id=replacement.tenant_id AND replacement_request.distribution_id=replacement.id
		WHERE (t.id::text=$1 OR t.slug=$1) AND legacy.legal_entity_id=$2::uuid AND legacy.subject_type='VENDOR_RELATIONSHIP' AND legacy.subject_id=$3::uuid
		  AND legacy_form.code='VENDOR-DUE-DILIGENCE' AND legacy.form_template_version=$4 AND legacy.title=$5
		  AND legacy.status='SUPERSEDED' AND replacement.subject_type='VENDOR_RELATIONSHIP' AND replacement.subject_id=legacy.subject_id
		  AND replacement.status IN ('OPEN','LOCKED') AND replacement_request.status IN ('READY','IN_PROGRESS','SUBMITTED')
		ORDER BY replacement.updated_at DESC,replacement.id DESC LIMIT 2`,
		seed.TenantID, seed.LegalEntityID, relationshipID, cloudspaceLegacyFormVersion, cloudspaceLegacyFormTitle, idempotencyKey)
	if err != nil {
		return evidence.DistributionBundle{}, monitoring.FormTemplate{}, false, err
	}
	defer rows.Close()
	type candidate struct {
		distributionID, formID string
		version                int64
	}
	values := []candidate{}
	for rows.Next() {
		var value candidate
		if err := rows.Scan(&value.distributionID, &value.formID, &value.version); err != nil {
			return evidence.DistributionBundle{}, monitoring.FormTemplate{}, false, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return evidence.DistributionBundle{}, monitoring.FormTemplate{}, false, err
	}
	if len(values) > 1 {
		return evidence.DistributionBundle{}, monitoring.FormTemplate{}, false, fmt.Errorf("Cloudspace supersession replacement is ambiguous")
	}
	if len(values) == 0 {
		return evidence.DistributionBundle{}, monitoring.FormTemplate{}, false, nil
	}
	form, err := monitoringRepo.ReusableFormRevision(ctx, seed.TenantID, seed.LegalEntityID, values[0].formID, values[0].version)
	if err != nil {
		return evidence.DistributionBundle{}, monitoring.FormTemplate{}, false, err
	}
	bundle, err := distributions.Get(ctx, seed.TenantID, seed.LegalEntityID, values[0].distributionID)
	if err != nil {
		return evidence.DistributionBundle{}, monitoring.FormTemplate{}, false, err
	}
	return bundle, form, true, nil
}

func hasCloudspaceSupersession(ctx context.Context, pool *pgxpool.Pool, seed bankverticals.SeedConfig, relationshipID string) (bool, error) {
	var found bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM capture_form_distributions legacy
			JOIN tenants t ON t.id=legacy.tenant_id
			JOIN monitoring_form_templates legacy_form ON legacy_form.tenant_id=legacy.tenant_id AND legacy_form.id=legacy.form_template_id AND legacy_form.version=legacy.form_template_version
			JOIN capture_distribution_events supersession ON supersession.tenant_id=legacy.tenant_id AND supersession.distribution_id=legacy.id AND supersession.event_type='FORM_DISTRIBUTION_SUPERSEDED'
			WHERE (t.id::text=$1 OR t.slug=$1) AND legacy.legal_entity_id=$2::uuid AND legacy.subject_type='VENDOR_RELATIONSHIP' AND legacy.subject_id=$3::uuid
			  AND legacy_form.code='VENDOR-DUE-DILIGENCE' AND legacy.form_template_version=$4 AND legacy.title=$5 AND legacy.status='SUPERSEDED'
		)`, seed.TenantID, seed.LegalEntityID, relationshipID, cloudspaceLegacyFormVersion, cloudspaceLegacyFormTitle).Scan(&found)
	return found, err
}

func existingOperatingFormSample(ctx context.Context, pool *pgxpool.Pool, distributions *evidence.DistributionService, access *evidence.DistributionAccessService, evidenceRepo *evidence.PostgresRepository, seed bankverticals.SeedConfig, form monitoring.FormTemplate, relationshipID, distributionID string, spec operatingFormSampleSpec) (string, *evidence.ResponseScoreResult, error) {
	bundle, err := distributions.Get(ctx, seed.TenantID, seed.LegalEntityID, distributionID)
	if err != nil {
		return "", nil, err
	}
	if bundle.Distribution.SubjectType != "VENDOR_RELATIONSHIP" || bundle.Distribution.SubjectID != relationshipID || bundle.Distribution.FormTemplateID != form.ID || bundle.Distribution.FormTemplateVersion != form.Version {
		return "", nil, fmt.Errorf("existing sample distribution scope does not match %s", spec.key)
	}
	if len(bundle.Recipients) != 1 {
		return "", nil, fmt.Errorf("existing sample has %d recipients", len(bundle.Recipients))
	}
	request, err := evidenceRepo.GetRequest(ctx, seed.TenantID, bundle.Recipients[0].RequestID)
	if err != nil {
		return "", nil, err
	}
	if request.Status != evidence.RequestReady && request.Status != evidence.RequestInProgress && request.Status != evidence.RequestSubmitted {
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
	case "COMPLETED_UNREVIEWED":
		if len(revisions) != 1 || !revisions[0].Current || revisions[0].Score == nil || revisions[0].Score.Final || revisions[0].Score.State != evidence.ResponseScoreNotConfigured {
			return "", nil, fmt.Errorf("existing unreviewed sample does not have one current unscored response")
		}
		if err := validateOperatingFormSampleAnswers(ctx, pool, distributionID, spec.answers); err != nil {
			return "", nil, err
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

func validateOperatingFormSampleAnswers(ctx context.Context, pool *pgxpool.Pool, distributionID string, expected map[string]formcontract.AnswerValue) error {
	var raw []byte
	err := pool.QueryRow(ctx, `SELECT s.answers FROM capture_submissions s JOIN capture_response_revisions r ON r.tenant_id=s.tenant_id AND r.submission_id=s.id WHERE s.distribution_id=$1::uuid AND r.distribution_id=$1::uuid AND r.is_current`, distributionID).Scan(&raw)
	if err != nil {
		return fmt.Errorf("load existing sample answers: %w", err)
	}
	actual := map[string]formcontract.AnswerValue{}
	if err := json.Unmarshal(raw, &actual); err != nil {
		return fmt.Errorf("decode existing sample answers: %w", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		return fmt.Errorf("existing sample answers do not match the immutable fixture")
	}
	return nil
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

func operatingFormSampleEdits(answers map[string]formcontract.AnswerValue, sequences map[string]int64) []evidence.FieldEdit {
	fieldIDs := make([]string, 0, len(answers))
	for fieldID := range answers {
		fieldIDs = append(fieldIDs, fieldID)
	}
	sort.Strings(fieldIDs)
	edits := make([]evidence.FieldEdit, 0, len(fieldIDs))
	for _, fieldID := range fieldIDs {
		edits = append(edits, evidence.FieldEdit{FieldID: fieldID, Value: answers[fieldID], BaseSequence: sequences[fieldID]})
	}
	return edits
}
