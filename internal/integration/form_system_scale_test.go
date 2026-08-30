//go:build postgres && postgresintegration

package integration_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
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
				$4::timestamptz-interval '400 days',$4::timestamptz-interval '400 days',1
			FROM generate_series(1,12) i;
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
	firstForms, err := forms.ListFormLibrary(ctx, monitoring.FormLibraryFilter{TenantID: "forms-scale-bank", LegalEntityID: entity, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(firstForms.Items) != 100 || firstForms.NextCursor == "" {
		t.Fatalf("expected a bounded first form page, got %d items and cursor %q", len(firstForms.Items), firstForms.NextCursor)
	}
	secondForms, err := forms.ListFormLibrary(ctx, monitoring.FormLibraryFilter{TenantID: "forms-scale-bank", LegalEntityID: entity, Cursor: firstForms.NextCursor, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(secondForms.Items) != 100 || firstForms.Items[99].Template.ID == secondForms.Items[0].Template.ID {
		t.Fatalf("form keyset pagination repeated or omitted its boundary: %#v", secondForms)
	}
	otherForms, err := forms.ListFormLibrary(ctx, monitoring.FormLibraryFilter{TenantID: "forms-scale-bank", LegalEntityID: otherEntity, Limit: 100})
	if err != nil || len(otherForms.Items) != 0 {
		t.Fatalf("form library crossed the legal-entity boundary: %#v, %v", otherForms, err)
	}

	distributions := evidence.NewDistributionService(evidence.NewPostgresDistributionStore(evidence.NewPostgresRepository(pool), nil))
	query := evidence.DistributionListQuery{TenantID: "forms-scale-bank", LegalEntityID: entity, Now: time.Now().UTC(), Limit: 100}
	firstDistributions, err := distributions.List(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstDistributions.Items) != 100 || firstDistributions.NextCursor == "" {
		t.Fatalf("expected a bounded first distribution page, got %d items and cursor %q", len(firstDistributions.Items), firstDistributions.NextCursor)
	}
	query.Cursor = firstDistributions.NextCursor
	secondDistributions, err := distributions.List(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondDistributions.Items) != 100 || firstDistributions.Items[99].ID == secondDistributions.Items[0].ID {
		t.Fatalf("distribution keyset pagination repeated or omitted its boundary: %#v", secondDistributions)
	}
	otherDistributions, err := distributions.List(ctx, evidence.DistributionListQuery{TenantID: "forms-scale-bank", LegalEntityID: otherEntity, Now: time.Now().UTC(), Limit: 100})
	if err != nil || len(otherDistributions.Items) != 0 {
		t.Fatalf("sent forms crossed the legal-entity boundary: %#v, %v", otherDistributions, err)
	}

	assertScaleIndex(t, pool, `SELECT id FROM monitoring_form_templates WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid ORDER BY updated_at DESC,id DESC,version DESC LIMIT 101`, tenant, entity, "monitoring_form_templates_library_idx")
	assertScaleIndex(t, pool, `SELECT id FROM capture_form_distributions WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid ORDER BY updated_at DESC,id DESC LIMIT 101`, tenant, entity, "capture_form_distributions_updated_idx")
}

func assertScaleIndex(t *testing.T, pool *pgxpool.Pool, query string, tenantID string, entityID string, expected string) {
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
	rows, err := connection.Query(ctx, "EXPLAIN "+query, tenantID, entityID)
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
}
