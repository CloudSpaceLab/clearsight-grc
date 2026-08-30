//go:build postgres && postgresintegration

package integration_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestGovernedFormsStayBoundedAtBankScale(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `TRUNCATE tenants CASCADE`); err != nil {
		t.Fatal(err)
	}

	const tenant = "91919191-9191-7191-8191-919191919191"
	const entity = "91919191-9191-7191-8191-919191919192"
	const otherEntity = "91919191-9191-7191-8191-919191919193"
	const actor = "91919191-9191-7191-8191-919191919194"
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'forms-scale-bank','Forms Scale Bank')`, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES
		($2::uuid,$1::uuid,'BANK-NG','Forms Scale Bank Nigeria','Nigeria'),
		($3::uuid,$1::uuid,'BANK-GH','Forms Scale Bank Ghana','Ghana')`, tenant, entity, otherEntity); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($2::uuid,$1::uuid,'PERSON','Forms Operations Owner')`, tenant, actor); err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO monitoring_form_templates(
			id,tenant_id,legal_entity_id,code,name,purpose,fields,status,is_current,effective_from,version,
			created_by,approved_by,created_at,updated_at,responsible_team,approved_uses,tags,jurisdiction,industry,sensitivity)
		SELECT md5('scale-form-'||i)::uuid,$1::uuid,$2::uuid,format('SCALE-%s',i),format('Vendor review %s',i),
			'Collect current vendor evidence','[{"id":"confirmation","type":"TEXT","label":"Confirmation","required":true}]'::jsonb,
			'ACTIVE',true,clock_timestamp()-interval '1 day',1,$3::uuid,$3::uuid,
			clock_timestamp()-(i||' milliseconds')::interval,clock_timestamp()-(i||' milliseconds')::interval,
			'Third-party risk',ARRAY['VENDOR_DUE_DILIGENCE'],ARRAY['scale'],'Nigeria','Financial services','CONFIDENTIAL'
		FROM generate_series(1,1000) i`, tenant, entity, actor); err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO capture_form_distributions(
			id,tenant_id,legal_entity_id,form_template_id,form_template_version,subject_type,subject_id,title,purpose,
			access_policy,status,deadline,route_expires_at,created_by,version,created_at,updated_at)
		SELECT md5('scale-distribution-'||i)::uuid,$1::uuid,$2::uuid,md5('scale-form-'||i)::uuid,1,'VENDOR_RELATIONSHIP',
			md5('scale-vendor-'||i)::uuid,format('Vendor review %s',i),'Confirm current vendor evidence','DIRECT_LINK_EMAIL_OTP','OPEN',
			clock_timestamp()+interval '90 days',clock_timestamp()+interval '60 days',$3::uuid,1,
			clock_timestamp()-(i||' milliseconds')::interval,clock_timestamp()-(i||' milliseconds')::interval
		FROM generate_series(1,400) i`, tenant, entity, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO capture_requests(
			id,tenant_id,legal_entity_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,
			estimated_minutes,deadline,known_facts,fields,status,created_by,version,created_at,updated_at,
			recipient_type,recipient_principal_id,recipient_hint,recipient_state,recipient_revision,distribution_id,
			form_template_id,form_template_version)
		SELECT md5(format('scale-request-%s-%s',i,recipient_number))::uuid,$1::uuid,$2::uuid,
			'VENDOR_RELATIONSHIP',md5('scale-vendor-'||i)::uuid::text,format('Vendor review %s',i),
			'Confirm current vendor evidence','You are responsible for the selected vendor evidence.','CONFIDENTIAL','INTERNAL',
			3,clock_timestamp()+interval '90 days','{}'::jsonb,
			'[{"id":"confirmation","type":"TEXT","label":"Confirmation","required":true}]'::jsonb,
			'SUBMITTED',$3::uuid,2,clock_timestamp()-interval '2 days',clock_timestamp()-interval '1 day',
			'INTERNAL_PRINCIPAL',$3::uuid,'','ASSIGNED',1,
			md5('scale-distribution-'||i)::uuid,md5('scale-form-'||i)::uuid,1
		FROM generate_series(1,400) i CROSS JOIN generate_series(1,2) recipient_number`, tenant, entity, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO capture_distribution_recipients(
			id,distribution_id,tenant_id,legal_entity_id,role,recipient_type,principal_id,request_id,
			audience_hint,contact_label,state,version,created_at,updated_at)
		SELECT md5(format('scale-recipient-%s-%s',i,recipient_number))::uuid,
			md5('scale-distribution-'||i)::uuid,$1::uuid,$2::uuid,
			CASE WHEN recipient_number<=2 THEN 'TO' ELSE 'CC' END,
			'INTERNAL_PRINCIPAL',$3::uuid,
			CASE WHEN recipient_number<=2 THEN md5(format('scale-request-%s-%s',i,recipient_number))::uuid ELSE NULL END,
			'Forms Operations Owner','Forms Operations Owner',
			CASE WHEN recipient_number<=2 THEN 'COMPLETED' ELSE 'DELIVERED' END,
			1,clock_timestamp()-interval '2 days',clock_timestamp()-interval '1 day'
		FROM generate_series(1,400) i CROSS JOIN generate_series(1,3) recipient_number`, tenant, entity, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO capture_response_workspaces(
			id,tenant_id,legal_entity_id,distribution_id,status,version,created_at,updated_at)
		SELECT md5('scale-workspace-'||i)::uuid,$1::uuid,$2::uuid,md5('scale-distribution-'||i)::uuid,
			'COMPLETED',3,clock_timestamp()-interval '2 days',clock_timestamp()-interval '1 day'
		FROM generate_series(1,400) i`, tenant, entity); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO capture_submissions(
			id,tenant_id,request_id,submitted_by,channel,answers,submitted_at,created_at,distribution_id,answer_provenance)
		SELECT md5(format('scale-submission-%s-%s',i,revision))::uuid,$1::uuid,
			md5(format('scale-request-%s-1',i))::uuid,$2::uuid,'INTERNAL',
			jsonb_build_object('confirmation',CASE WHEN revision=1 THEN 'Initial' ELSE 'Amended' END),
			clock_timestamp()-(3-revision)*interval '1 day',clock_timestamp()-(3-revision)*interval '1 day',
			md5('scale-distribution-'||i)::uuid,'{}'::jsonb
		FROM generate_series(1,400) i CROSS JOIN generate_series(1,2) revision`, tenant, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO capture_response_revisions(
			id,tenant_id,legal_entity_id,distribution_id,workspace_id,submission_id,revision,supersedes_revision_id,
			achieved_assurance,signoff_summary,compliance_score,scored_weight_coverage,state,critical_field_results,
			scoring_policy_version,is_current,created_at)
		SELECT md5(format('scale-response-%s-%s',i,revision))::uuid,$1::uuid,$2::uuid,
			md5('scale-distribution-'||i)::uuid,md5('scale-workspace-'||i)::uuid,
			md5(format('scale-submission-%s-%s',i,revision))::uuid,revision,
			CASE WHEN revision=1 THEN NULL ELSE md5(format('scale-response-%s-1',i))::uuid END,
			'EMAIL_VERIFIED',jsonb_build_object('attested',true,'revision',revision),
			CASE WHEN revision=1 THEN 80 ELSE 90 END,100,'FINAL','[]'::jsonb,
			'scale-policy-v1',revision=2,clock_timestamp()-(3-revision)*interval '1 day'
		FROM generate_series(1,400) i CROSS JOIN generate_series(1,2) revision`, tenant, entity); err != nil {
		t.Fatal(err)
	}

	t.Run("representative recipient and revision population", func(t *testing.T) {
		for table, want := range map[string]int{
			"monitoring_form_templates":       1000,
			"capture_form_distributions":      400,
			"capture_distribution_recipients": 1200,
			"capture_response_revisions":      800,
		} {
			var got int
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE tenant_id=$1::uuid", tenant).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("%s=%d, want %d", table, got, want)
			}
		}
	})

	t.Run("bounded reminder maintenance advances past completed batches", func(t *testing.T) {
		now := time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)
		if _, err := pool.Exec(ctx, `
			UPDATE capture_form_distributions
			SET deadline=$1::timestamptz+interval '48 hours',route_expires_at=$1::timestamptz+interval '24 hours',
				reminder_policy='{"reminder_hours_before":[72]}'::jsonb
			WHERE tenant_id=$2::uuid AND id IN (
				SELECT id FROM capture_form_distributions
				WHERE tenant_id=$2::uuid ORDER BY id LIMIT 20
			)`, now, tenant); err != nil {
			t.Fatal(err)
		}

		scheduler := evidence.NewCommunicationReminderScheduler(evidence.NewPostgresCommunicationReminderRepository(pool))
		for run, want := range []int{7, 7, 6, 0} {
			got, err := scheduler.Maintain(ctx, now, 7)
			if err != nil {
				t.Fatalf("maintenance run %d: %v", run+1, err)
			}
			if got != want {
				t.Fatalf("maintenance run %d created %d reminders, want %d; a completed batch must not starve later distributions", run+1, got, want)
			}
		}
		var reminders int
		if err := pool.QueryRow(ctx, `
			SELECT count(*) FROM outbox_events
			WHERE tenant_id=$1::uuid AND event_type='FORM_COMMUNICATION_REMINDER_DUE'`, tenant).Scan(&reminders); err != nil {
			t.Fatal(err)
		}
		if reminders != 20 {
			t.Fatalf("reminder outbox events=%d, want 20", reminders)
		}
	})

	t.Run("bounded vendor refresh maintenance advances past covered batches", func(t *testing.T) {
		now := time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)
		if _, err := pool.Exec(ctx, `
			INSERT INTO third_parties(id,tenant_id,legal_name,status,created_at,updated_at,version)
			SELECT md5('scale-refresh-vendor-'||i)::uuid,$1::uuid,format('Refresh vendor %s',i),'ACTIVE',
				$2::timestamptz-interval '400 days',$2::timestamptz-interval '400 days',1
			FROM generate_series(1,12) i`, tenant, now); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO third_party_relationships(
				id,tenant_id,legal_entity_id,vendor_id,service_name,business_owner_principal_id,
				criticality,privacy_role,status,created_at,updated_at,version)
			SELECT md5('scale-refresh-relationship-'||i)::uuid,$1::uuid,$2::uuid,
				md5('scale-refresh-vendor-'||i)::uuid,format('Refresh service %s',i),$3::uuid,
				'STANDARD','NONE','ACTIVE',$4::timestamptz-interval '400 days',$4::timestamptz-interval '400 days',1
			FROM generate_series(1,12) i`, tenant, entity, actor, now); err != nil {
			t.Fatal(err)
		}

		repository := thirdparty.NewPostgresRepository(pool)
		policy := thirdparty.RefreshMaintenancePolicy{
			BatchSize:                5,
			Lease:                    time.Minute,
			DocumentLead:             30 * 24 * time.Hour,
			FactConfirmationInterval: 365 * 24 * time.Hour,
		}
		for run, want := range []int{5, 5, 2, 0} {
			receipt, err := repository.MaintainVendorRefresh(ctx, now, policy)
			if err != nil {
				t.Fatalf("refresh maintenance run %d: %v", run+1, err)
			}
			if receipt.RelationshipsExamined != want || receipt.AttentionsCreated != want {
				t.Fatalf("refresh maintenance run %d examined/created %d/%d, want %d/%d; covered work must not starve later relationships", run+1, receipt.RelationshipsExamined, receipt.AttentionsCreated, want, want)
			}
		}
		var attentions int
		if err := pool.QueryRow(ctx, `
			SELECT count(*) FROM third_party_refresh_attentions
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND state='OPEN'`, tenant, entity).Scan(&attentions); err != nil {
			t.Fatal(err)
		}
		if attentions != 12 {
			t.Fatalf("open vendor refresh attentions=%d, want 12", attentions)
		}
	})

	forms := monitoring.NewPostgresRepository(pool)
	formIDs := map[string]struct{}{}
	formCursor := ""
	for pageNumber := 1; pageNumber <= 10; pageNumber++ {
		page, err := forms.ListFormLibrary(ctx, monitoring.FormLibraryFilter{TenantID: "forms-scale-bank", LegalEntityID: entity, Cursor: formCursor, Limit: 100})
		if err != nil {
			t.Fatalf("form page %d: %v", pageNumber, err)
		}
		if len(page.Items) != 100 {
			t.Fatalf("form page %d items=%d, want 100", pageNumber, len(page.Items))
		}
		for _, item := range page.Items {
			if _, duplicate := formIDs[item.Template.ID]; duplicate {
				t.Fatalf("form page %d repeated template %s", pageNumber, item.Template.ID)
			}
			formIDs[item.Template.ID] = struct{}{}
		}
		if pageNumber < 10 && page.NextCursor == "" {
			t.Fatalf("form page %d omitted the next cursor", pageNumber)
		}
		if pageNumber == 10 && page.NextCursor != "" {
			t.Fatalf("final form page exposed unexpected cursor %q", page.NextCursor)
		}
		formCursor = page.NextCursor
	}
	if len(formIDs) != 1000 {
		t.Fatalf("stable form pagination returned %d unique templates, want 1000", len(formIDs))
	}
	otherForms, err := forms.ListFormLibrary(ctx, monitoring.FormLibraryFilter{TenantID: "forms-scale-bank", LegalEntityID: otherEntity, Limit: 100})
	if err != nil || len(otherForms.Items) != 0 {
		t.Fatalf("form library crossed the legal-entity boundary: %#v, %v", otherForms, err)
	}

	distributions := evidence.NewDistributionService(evidence.NewPostgresDistributionStore(evidence.NewPostgresRepository(pool), nil))
	query := evidence.DistributionListQuery{TenantID: "forms-scale-bank", LegalEntityID: entity, Now: time.Now().UTC(), Limit: 100}
	distributionIDs := map[string]struct{}{}
	for pageNumber := 1; pageNumber <= 4; pageNumber++ {
		page, err := distributions.List(ctx, query)
		if err != nil {
			t.Fatalf("distribution page %d: %v", pageNumber, err)
		}
		if len(page.Items) != 100 {
			t.Fatalf("distribution page %d items=%d, want 100", pageNumber, len(page.Items))
		}
		for _, item := range page.Items {
			if _, duplicate := distributionIDs[item.ID]; duplicate {
				t.Fatalf("distribution page %d repeated distribution %s", pageNumber, item.ID)
			}
			distributionIDs[item.ID] = struct{}{}
		}
		if pageNumber < 4 && page.NextCursor == "" {
			t.Fatalf("distribution page %d omitted the next cursor", pageNumber)
		}
		if pageNumber == 4 && page.NextCursor != "" {
			t.Fatalf("final distribution page exposed unexpected cursor %q", page.NextCursor)
		}
		query.Cursor = page.NextCursor
	}
	if len(distributionIDs) != 400 {
		t.Fatalf("stable distribution pagination returned %d unique records, want 400", len(distributionIDs))
	}
	otherDistributions, err := distributions.List(ctx, evidence.DistributionListQuery{TenantID: "forms-scale-bank", LegalEntityID: otherEntity, Now: time.Now().UTC(), Limit: 100})
	if err != nil || len(otherDistributions.Items) != 0 {
		t.Fatalf("sent forms crossed the legal-entity boundary: %#v, %v", otherDistributions, err)
	}

	t.Run("exact response lookup stays scoped and indexed", func(t *testing.T) {
		distributionID := ""
		responseID := ""
		if err := pool.QueryRow(ctx, `SELECT md5('scale-distribution-1')::uuid::text,md5('scale-response-1-2')::uuid::text`).Scan(&distributionID, &responseID); err != nil {
			t.Fatal(err)
		}
		var revision int64
		if err := pool.QueryRow(ctx, `
			SELECT revision FROM capture_response_revisions
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND distribution_id=$3::uuid AND id=$4::uuid`,
			tenant, entity, distributionID, responseID).Scan(&revision); err != nil {
			t.Fatal(err)
		}
		if revision != 2 {
			t.Fatalf("exact response revision=%d, want 2", revision)
		}
		err := pool.QueryRow(ctx, `
			SELECT revision FROM capture_response_revisions
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND distribution_id=$3::uuid AND id=$4::uuid`,
			tenant, otherEntity, distributionID, responseID).Scan(&revision)
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("cross-entity exact response lookup error=%v, want no rows", err)
		}

		assertScaleIndex(t, pool, `SELECT revision FROM capture_response_revisions WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND distribution_id=$3::uuid AND id=$4::uuid`, "capture_response_revisions_pkey", tenant, entity, distributionID, responseID)
		assertScaleIndex(t, pool, `SELECT id FROM capture_response_revisions WHERE tenant_id=$1::uuid AND workspace_id=md5('scale-workspace-1')::uuid ORDER BY revision DESC,id DESC LIMIT 3`, "capture_response_revisions_history_idx", tenant)
	})

	t.Run("material Forms records reconstruct before and after change", func(t *testing.T) {
		before := time.Date(2026, time.August, 30, 9, 0, 0, 0, time.UTC)
		changed := before.Add(time.Hour)
		asOf := before.Add(30 * time.Minute)

		if _, err := pool.Exec(ctx, `
			UPDATE monitoring_form_templates
			SET status='RETIRED',is_current=false,effective_until=$1,updated_at=$1
			WHERE tenant_id=$2::uuid AND id=md5('scale-form-1')::uuid AND version=1`, changed, tenant); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO monitoring_form_templates(
				id,tenant_id,legal_entity_id,code,name,purpose,fields,status,is_current,effective_from,version,
				created_by,approved_by,created_at,updated_at,responsible_team,approved_uses,tags,jurisdiction,industry,sensitivity)
			VALUES(md5('scale-form-1')::uuid,$1::uuid,$2::uuid,'SCALE-1','Vendor review 1 amended',
				'Collect current vendor evidence','[{"id":"confirmation","type":"TEXT","label":"Confirmation","required":true}]'::jsonb,
				'ACTIVE',true,$3,2,$4::uuid,$4::uuid,$3,$3,'Third-party risk',ARRAY['VENDOR_DUE_DILIGENCE'],
				ARRAY['scale'],'Nigeria','Financial services','CONFIDENTIAL')`, tenant, entity, changed, actor); err != nil {
			t.Fatal(err)
		}
		var historicalForm, currentForm string
		if err := pool.QueryRow(ctx, `
			SELECT name FROM monitoring_form_templates
			WHERE tenant_id=$1::uuid AND id=md5('scale-form-1')::uuid
			  AND effective_from<=$2 AND (effective_until IS NULL OR $2<effective_until)
			ORDER BY version DESC LIMIT 1`, tenant, asOf).Scan(&historicalForm); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `
			SELECT name FROM monitoring_form_templates
			WHERE tenant_id=$1::uuid AND id=md5('scale-form-1')::uuid AND is_current`, tenant).Scan(&currentForm); err != nil {
			t.Fatal(err)
		}
		if historicalForm != "Vendor review 1" || currentForm != "Vendor review 1 amended" {
			t.Fatalf("form reconstruction historical/current=%q/%q", historicalForm, currentForm)
		}

		if _, err := pool.Exec(ctx, `
			INSERT INTO form_communication_templates(
				id,tenant_id,legal_entity_id,action,locale,version,subject_template,document,status,
				effective_from,effective_until,maker_id,created_at,updated_at)
			VALUES
				(md5('scale-communication-1')::uuid,$1::uuid,$2::uuid,'REMINDER','en',1,'Evidence reminder',
				 '[{"type":"PARAGRAPH","text":"Please confirm the requested evidence."}]'::jsonb,'RETIRED',$3,$4,$5::uuid,$3,$4),
				(md5('scale-communication-2')::uuid,$1::uuid,$2::uuid,'REMINDER','en',2,'Evidence reminder amended',
				 '[{"type":"PARAGRAPH","text":"Please confirm the amended evidence request."}]'::jsonb,'ACTIVE',$4,NULL,$5::uuid,$4,$4)`,
			tenant, entity, before, changed, actor); err != nil {
			t.Fatal(err)
		}
		var historicalSubject, currentSubject string
		if err := pool.QueryRow(ctx, `
			SELECT subject_template FROM form_communication_templates
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND action='REMINDER' AND locale='en'
			  AND effective_from<=$3 AND (effective_until IS NULL OR $3<effective_until)
			ORDER BY version DESC LIMIT 1`, tenant, entity, asOf).Scan(&historicalSubject); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `
			SELECT subject_template FROM form_communication_templates
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND action='REMINDER' AND locale='en' AND status='ACTIVE'
			ORDER BY version DESC LIMIT 1`, tenant, entity).Scan(&currentSubject); err != nil {
			t.Fatal(err)
		}
		if historicalSubject != "Evidence reminder" || currentSubject != "Evidence reminder amended" {
			t.Fatalf("communication reconstruction historical/current=%q/%q", historicalSubject, currentSubject)
		}

		if _, err := pool.Exec(ctx, `
			INSERT INTO capture_distribution_events(
				tenant_id,legal_entity_id,distribution_id,distribution_version,event_type,payload,actor_id,occurred_at)
			VALUES
				($1::uuid,$2::uuid,md5('scale-distribution-1')::uuid,1,'FORM_DISTRIBUTION_CREATED',
				 '{"title":"Vendor review 1"}'::jsonb,$3::uuid,$4),
				($1::uuid,$2::uuid,md5('scale-distribution-1')::uuid,2,'FORM_DISTRIBUTION_AMENDED',
				 '{"title":"Vendor review 1 amended"}'::jsonb,$3::uuid,$5)`, tenant, entity, actor, before, changed); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			UPDATE capture_form_distributions
			SET title='Vendor review 1 amended',version=2,updated_at=$1
			WHERE tenant_id=$2::uuid AND id=md5('scale-distribution-1')::uuid`, changed, tenant); err != nil {
			t.Fatal(err)
		}
		var historicalDistribution, currentDistribution string
		if err := pool.QueryRow(ctx, `
			SELECT payload->>'title' FROM capture_distribution_events
			WHERE tenant_id=$1::uuid AND distribution_id=md5('scale-distribution-1')::uuid AND occurred_at<=$2
			ORDER BY occurred_at DESC,id DESC LIMIT 1`, tenant, asOf).Scan(&historicalDistribution); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT title FROM capture_form_distributions WHERE tenant_id=$1::uuid AND id=md5('scale-distribution-1')::uuid`, tenant).Scan(&currentDistribution); err != nil {
			t.Fatal(err)
		}
		if historicalDistribution != "Vendor review 1" || currentDistribution != "Vendor review 1 amended" {
			t.Fatalf("distribution reconstruction historical/current=%q/%q", historicalDistribution, currentDistribution)
		}

		if _, err := pool.Exec(ctx, `
			UPDATE capture_response_revisions
			SET created_at=CASE revision WHEN 1 THEN $1::timestamptz ELSE $2::timestamptz END
			WHERE tenant_id=$3::uuid AND workspace_id=md5('scale-workspace-1')::uuid`, before, changed, tenant); err != nil {
			t.Fatal(err)
		}
		var historicalResponse, currentResponse int64
		if err := pool.QueryRow(ctx, `
			SELECT revision FROM capture_response_revisions
			WHERE tenant_id=$1::uuid AND workspace_id=md5('scale-workspace-1')::uuid AND created_at<=$2
			ORDER BY revision DESC,id DESC LIMIT 1`, tenant, asOf).Scan(&historicalResponse); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `
			SELECT revision FROM capture_response_revisions
			WHERE tenant_id=$1::uuid AND workspace_id=md5('scale-workspace-1')::uuid AND is_current`, tenant).Scan(&currentResponse); err != nil {
			t.Fatal(err)
		}
		if historicalResponse != 1 || currentResponse != 2 {
			t.Fatalf("response reconstruction historical/current=%d/%d", historicalResponse, currentResponse)
		}

		if _, err := pool.Exec(ctx, `
			INSERT INTO third_party_assessments(
				id,tenant_id,legal_entity_id,relationship_id,review_kind,stable_episode_key,status,
				form_template_id,form_template_version,review_due_at,started_by_principal_id,started_at,version,created_at,updated_at)
			VALUES(md5('scale-assessment-1')::uuid,$1::uuid,$2::uuid,md5('scale-refresh-relationship-1')::uuid,
				'ONBOARDING',repeat('a',64),'SETUP_PENDING',md5('scale-form-1')::uuid,1,$3::timestamptz+interval '30 days',
				$4::uuid,$3,1,$3,$3)`, tenant, entity, before, actor); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			UPDATE third_parties SET legal_name='Refresh vendor 1 corrected',version=2,updated_at=$1
			WHERE tenant_id=$2::uuid AND id=md5('scale-refresh-vendor-1')::uuid`, changed, tenant); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			UPDATE third_party_assessments SET version=2,updated_at=$1
			WHERE tenant_id=$2::uuid AND id=md5('scale-assessment-1')::uuid`, changed, tenant); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO third_party_response_application_receipts(
				id,tenant_id,legal_entity_id,assessment_id,distribution_id,response_revision_id,vendor_id,actor_principal_id,
				accepted_field_ids,rejected_field_ids,decisions,prior_vendor_version,result_vendor_version,result_assessment_version,applied_at)
			VALUES
				(md5('scale-application-1')::uuid,$1::uuid,$2::uuid,md5('scale-assessment-1')::uuid,
				 md5('scale-distribution-1')::uuid,md5('scale-response-1-1')::uuid,md5('scale-refresh-vendor-1')::uuid,$3::uuid,
				 '["legal_name"]'::jsonb,'[]'::jsonb,'[{"field_id":"legal_name","decision":"CONFIRM"}]'::jsonb,1,1,1,$4),
				(md5('scale-application-2')::uuid,$1::uuid,$2::uuid,md5('scale-assessment-1')::uuid,
				 md5('scale-distribution-1')::uuid,md5('scale-response-1-2')::uuid,md5('scale-refresh-vendor-1')::uuid,$3::uuid,
				 '["legal_name"]'::jsonb,'[]'::jsonb,'[{"field_id":"legal_name","decision":"CORRECT"}]'::jsonb,1,2,2,$5)`,
			tenant, entity, actor, before, changed); err != nil {
			t.Fatal(err)
		}
		var historicalApplication, currentApplication string
		if err := pool.QueryRow(ctx, `
			SELECT response_revision_id::text FROM third_party_response_application_receipts
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND assessment_id=md5('scale-assessment-1')::uuid AND applied_at<=$3
			ORDER BY applied_at DESC,id DESC LIMIT 1`, tenant, entity, asOf).Scan(&historicalApplication); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `
			SELECT response_revision_id::text FROM third_party_response_application_receipts
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND assessment_id=md5('scale-assessment-1')::uuid
			ORDER BY applied_at DESC,id DESC LIMIT 1`, tenant, entity).Scan(&currentApplication); err != nil {
			t.Fatal(err)
		}
		if historicalApplication == currentApplication {
			t.Fatalf("application receipt history was overwritten: %s", currentApplication)
		}

		if _, err := pool.Exec(ctx, `
			INSERT INTO capture_artifacts(
				id,tenant_id,request_id,file_name,media_type,size_bytes,sha256,storage_key,status,created_by,created_at)
			VALUES
				(md5('scale-document-artifact-1')::uuid,$1::uuid,md5('scale-request-1-1')::uuid,'certificate-2025.pdf','application/pdf',100,repeat('b',64),'scale/certificate-2025.pdf','AVAILABLE',$2::uuid,$3),
				(md5('scale-document-artifact-2')::uuid,$1::uuid,md5('scale-request-1-1')::uuid,'certificate-2026.pdf','application/pdf',120,repeat('c',64),'scale/certificate-2026.pdf','AVAILABLE',$2::uuid,$4)`,
			tenant, actor, before, changed); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO third_party_documents(
				id,tenant_id,legal_entity_id,relationship_id,assessment_id,request_id,artifact_id,document_type,
				evidence_class,status,validated_by_principal_id,validated_at,created_at,updated_at,version,supersedes_document_id)
			VALUES
				(md5('scale-document-1')::uuid,$1::uuid,$2::uuid,md5('scale-refresh-relationship-1')::uuid,
				 md5('scale-assessment-1')::uuid,md5('scale-request-1-1')::uuid,md5('scale-document-artifact-1')::uuid,
				 'INSURANCE_CERTIFICATE','BANK_VALIDATED','SUPERSEDED',NULL,NULL,$4,$5,2,NULL),
				(md5('scale-document-2')::uuid,$1::uuid,$2::uuid,md5('scale-refresh-relationship-1')::uuid,
				 md5('scale-assessment-1')::uuid,md5('scale-request-1-1')::uuid,md5('scale-document-artifact-2')::uuid,
				 'INSURANCE_CERTIFICATE','BANK_VALIDATED','VALIDATED',$3::uuid,$5,$5,$5,1,md5('scale-document-1')::uuid)`,
			tenant, entity, actor, before, changed); err != nil {
			t.Fatal(err)
		}
		var priorDocument, replacementDocument, priorStatus, replacementStatus string
		if err := pool.QueryRow(ctx, `
			SELECT prior.id::text,replacement.id::text,prior.status,replacement.status
			FROM third_party_documents replacement
			JOIN third_party_documents prior ON prior.tenant_id=replacement.tenant_id AND prior.id=replacement.supersedes_document_id
			WHERE replacement.tenant_id=$1::uuid AND replacement.id=md5('scale-document-2')::uuid`, tenant).
			Scan(&priorDocument, &replacementDocument, &priorStatus, &replacementStatus); err != nil {
			t.Fatal(err)
		}
		if priorDocument == replacementDocument || priorStatus != "SUPERSEDED" || replacementStatus != "VALIDATED" {
			t.Fatalf("document lineage prior/current=%s %s / %s %s", priorDocument, priorStatus, replacementDocument, replacementStatus)
		}
	})

	assertScaleIndex(t, pool, `SELECT id FROM monitoring_form_templates WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid ORDER BY updated_at DESC,id DESC,version DESC LIMIT 101`, "monitoring_form_templates_library_idx", tenant, entity)
	assertScaleIndex(t, pool, `SELECT id FROM capture_form_distributions WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid ORDER BY updated_at DESC,id DESC LIMIT 101`, "capture_form_distributions_updated_idx", tenant, entity)
	assertScaleIndex(t, pool, `SELECT id FROM capture_distribution_recipients WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND distribution_id=md5('scale-distribution-1')::uuid ORDER BY role,state,id LIMIT 4`, "capture_distribution_recipients_distribution_idx", tenant, entity)
}

func assertScaleIndex(t *testing.T, pool *pgxpool.Pool, query string, expected string, args ...any) {
	t.Helper()
	ctx := context.Background()
	connection, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Release()
	if _, err := connection.Exec(ctx, `SET enable_seqscan=off`); err != nil {
		t.Fatal(err)
	}
	rows, err := connection.Query(ctx, "EXPLAIN (COSTS OFF) "+query, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		plan.WriteString(line)
		plan.WriteByte('\n')
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.String(), expected) {
		t.Fatalf("expected %s in bounded query plan:\n%s", expected, plan.String())
	}
	if strings.Contains(plan.String(), "Seq Scan") {
		t.Fatalf("bounded query used a sequential scan:\n%s", plan.String())
	}
}
