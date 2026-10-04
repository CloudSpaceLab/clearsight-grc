//go:build postgres && postgresintegration

package metricview

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDomainMetricProjectionRetainsExactCrossDomainTruth(t *testing.T) {
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

	tenantID := mustDomainID(t)
	entityID := mustDomainID(t)
	principalID := mustDomainID(t)
	riskID := mustDomainID(t)
	appetiteID := mustDomainID(t)
	programID := mustDomainID(t)
	formID := mustDomainID(t)
	checkID := mustDomainID(t)
	resultID := mustDomainID(t)
	indicatorLinkID := mustDomainID(t)
	objectiveID := mustDomainID(t)
	implementationID := mustDomainID(t)
	definitionID := mustDomainID(t)
	catalogLinkID := mustDomainID(t)
	riskControlLinkID := mustDomainID(t)
	contractID := mustDomainID(t)
	lossID := mustDomainID(t)

	now := time.Now().UTC().Truncate(time.Hour).Add(30 * time.Minute)
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, execErr := pool.Exec(ctx, query, args...); execErr != nil {
			t.Fatal(execErr)
		}
	}

	mustExec(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Domain Metrics')`, tenantID, "domain-metrics-"+tenantID[len(tenantID)-8:])
	mustExec(`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,$3,'Domain Metrics Nigeria','NG',$4)`,
		entityID, tenantID, "DM-"+entityID[len(entityID)-8:], now.Add(-365*24*time.Hour))
	mustExec(`
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from)
		VALUES($1::uuid,$2::uuid,'PERSON','Domain risk owner','ACTIVE',$3)`,
		principalID, tenantID, now.Add(-365*24*time.Hour))
	mustExec(`
		INSERT INTO risks(
			id,tenant_id,legal_entity_id,code,name,category,statement,impact,status,version,created_at,updated_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,'DM-RISK','Domain metric risk','Operational',
		         'Operational exposure','Material impact','ACTIVE',2,$4,$4)`,
		riskID, tenantID, entityID, now.Add(-time.Hour))
	mustExec(`
		INSERT INTO risk_appetite_statements(
			id,tenant_id,legal_entity_id,risk_id,risk_version,version,statement,rule,
			owner_principal_id,authority_principal_id,status,effective_from,created_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,2,1,'Domain appetite','{}'::jsonb,
		         $5::uuid,$5::uuid,'ACTIVE',$6,$6)`,
		appetiteID, tenantID, entityID, riskID, principalID, now.Add(-24*time.Hour))
	mustExec(`
		INSERT INTO risk_assessments(
			tenant_id,legal_entity_id,risk_id,risk_version,assessment_kind,method_code,method_version,
			dimensions,assumptions,evidence_references,assessed_by,appetite_statement_id,appetite_position,assessed_at,created_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,2,'CURRENT','DOMAIN','1',
		         '{}'::jsonb,'{}'::jsonb,'[]'::jsonb,$4::uuid,$5::uuid,'BREACHED',$6,$6)`,
		tenantID, entityID, riskID, principalID, appetiteID, now.Add(-30*time.Minute))

	mustExec(`
		INSERT INTO programs(
			id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,owner_principal_id,effective_from
		) VALUES($1::uuid,$2::uuid,$3::uuid,'DM-PROGRAM','Domain monitoring','COMPLIANCE','ACTIVE','Risk',$4::uuid,$5)`,
		programID, tenantID, entityID, principalID, now.Add(-30*24*time.Hour))
	mustExec(`
		INSERT INTO monitoring_form_templates(
			id,tenant_id,legal_entity_id,program_id,code,name,purpose,fields,status,is_current,
			effective_from,version,created_by,approved_by,created_at,updated_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'DM-FORM','Domain metric form','Collect indicator input.',
		         '[{"id":"score","label":"Score","type":"number","required":true}]'::jsonb,
		         'ACTIVE',true,$5,1,$6::uuid,$6::uuid,$5,$5)`,
		formID, tenantID, entityID, programID, now.Add(-24*time.Hour), principalID)
	mustExec(`
		INSERT INTO monitoring_checks(
			id,tenant_id,program_id,code,name,claim,input_kind,form_template_id,form_template_version,
			source_rules,thresholds,freshness_minutes,minimum_coverage,owner_principal_id,reviewer_principal_id,
			failure_action,status,is_current,effective_from,version,created_by,submitted_by,approved_by,created_at,updated_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,'DM-KRI','Domain KRI','Risk score remains acceptable','FORM',$4::uuid,1,
		         '[]'::jsonb,'{}'::jsonb,1440,0.8,$5::uuid,$5::uuid,'MATTER','ACTIVE',true,$6,1,
		         $5::uuid,$5::uuid,$5::uuid,$6,$6)`,
		checkID, tenantID, programID, formID, principalID, now.Add(-24*time.Hour))
	mustExec(`
		INSERT INTO monitoring_results(
			id,tenant_id,program_id,monitoring_check_id,monitoring_check_version,input_kind,
			input_reference_id,input_reference_version,evaluation,evaluated_at,evaluator_version,created_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,1,'FORM',$5,1,
		         '{"band":"HIGH","coverage":1}'::jsonb,$6,'domain-test-v1',$6)`,
		resultID, tenantID, programID, checkID, "domain-input-"+resultID, now.Add(-10*time.Minute))
	mustExec(`
		INSERT INTO risk_indicator_links(
			id,tenant_id,legal_entity_id,risk_id,risk_version,program_id,monitoring_check_id,
			monitoring_check_version,kind,measurement,linked_by,created_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,2,$5::uuid,$6::uuid,1,'KRI',
		         '{"kind":"RISK_SCORE_0_100"}'::jsonb,$7::uuid,$8)`,
		indicatorLinkID, tenantID, entityID, riskID, programID, checkID, principalID, now.Add(-time.Hour))

	mustExec(`
		INSERT INTO control_objectives(id,tenant_id,program_id,code,name,outcome,status)
		VALUES($1::uuid,$2::uuid,$3::uuid,'DM-OBJ','Domain objective','Control remains effective','ACTIVE')`,
		objectiveID, tenantID, programID)
	mustExec(`
		INSERT INTO control_implementations(
			id,tenant_id,program_id,objective_id,name,description,implementation_type,status,effective_from
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'Domain control','Domain assurance control','CHECKLIST','IMPLEMENTED',$5)`,
		implementationID, tenantID, programID, objectiveID, now.Add(-30*24*time.Hour))
	mustExec(`
		INSERT INTO control_definitions(id,tenant_id,code,name,objective,status,version,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,'DM-CONTROL','Domain control','Control objective','ACTIVE',1,$3,$3)`,
		definitionID, tenantID, now.Add(-30*24*time.Hour))
	mustExec(`
		INSERT INTO control_catalog_implementation_links(
			id,tenant_id,legal_entity_id,definition_id,program_id,implementation_id,created_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7)`,
		catalogLinkID, tenantID, entityID, definitionID, programID, implementationID, now.Add(-20*24*time.Hour))
	mustExec(`
		INSERT INTO risk_control_links(
			id,tenant_id,legal_entity_id,risk_id,risk_version,catalog_link_id,linked_by,created_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,2,$5::uuid,$6::uuid,$7)`,
		riskControlLinkID, tenantID, entityID, riskID, catalogLinkID, principalID, now.Add(-10*24*time.Hour))
	mustExec(`
		INSERT INTO evidence_contracts(
			id,tenant_id,program_id,control_implementation_id,code,name,claim,
			freshness_minutes,minimum_coverage,independence_required,contradiction_policy,failure_action,status
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'DM-EVIDENCE','Domain evidence','Control operates',
		         1440,0.8,false,'REVIEW','FLAG','ACTIVE')`,
		contractID, tenantID, programID, implementationID)
	mustExec(`
		INSERT INTO evidence_assessments(
			tenant_id,program_id,contract_id,conclusion,coverage,assessed_by,assessed_at,valid_until
		) VALUES($1::uuid,$2::uuid,$3::uuid,'UNSUPPORTED',1,$4::uuid,$5,$6)`,
		tenantID, programID, contractID, principalID, now.Add(-15*time.Minute), now.Add(24*time.Hour))

	mustExec(`
		INSERT INTO operational_losses(
			id,tenant_id,legal_entity_id,code,title,category,occurred_at,discovered_at,owner_principal_id,status,version,created_at,updated_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,'DM-LOSS','Domain metric loss','PROCESS',$4,$4,$5::uuid,'ACTIVE',1,$4,$4)`,
		lossID, tenantID, entityID, now.Add(-2*time.Hour), principalID)

	observationRepository := NewObservationRepository(pool)
	inserted, err := maintainDomainScope(ctx, pool, domainScope{TenantID: tenantID, LegalEntityID: entityID}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !inserted {
		t.Fatal("expected first domain metric snapshot to be inserted")
	}
	inserted, err = maintainDomainScope(ctx, pool, domainScope{TenantID: tenantID, LegalEntityID: entityID}, now.Add(10*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if inserted {
		t.Fatal("same UTC hour must reuse the retained domain snapshot")
	}

	bundle, err := NewDomainRepository(pool).LatestDomainMetrics(ctx, tenantID, entityID)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.SourceID == "" || bundle.DefinitionRevision != DomainDefinitionRevision || len(bundle.Items) != len(domainDefinitions) {
		t.Fatalf("bundle=%#v", bundle)
	}
	for _, metricID := range []string{"risks_outside_appetite", "indicator_breaches", "assurance_failures", "losses_without_intervention"} {
		item := domainMetricByID(t, bundle, metricID)
		if item.Value != 1 || item.Population != 1 || item.Unknown == nil || *item.Unknown != 0 ||
			item.Excluded == nil || *item.Excluded != 0 || item.Completeness != CompletenessComplete {
			t.Fatalf("%s=%#v", metricID, item)
		}
	}

	members := NewMembershipRepository(pool)
	for _, tc := range []struct {
		metricID   string
		targetType string
		targetID   string
	}{
		{"risks_outside_appetite", "RISK", riskID},
		{"indicator_breaches", "RISK", riskID},
		{"assurance_failures", "RISK", riskID},
		{"losses_without_intervention", "LOSS", lossID},
	} {
		page, err := members.ListSnapshotMembers(
			ctx, tenantID, entityID, "", bundle.SourceID, tc.metricID, DomainDefinitionRevision, principalID, "", 10,
		)
		if err != nil {
			t.Fatalf("%s members: %v", tc.metricID, err)
		}
		if page.Total != 1 || len(page.Items) != 1 || page.Items[0].TargetType != tc.targetType || page.Items[0].TargetID != tc.targetID {
			t.Fatalf("%s page=%#v", tc.metricID, page)
		}
	}

	series, err := observationRepository.Trend(ctx, tenantID, entityID, "risks_outside_appetite", now.Add(-time.Hour), now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if series.DefinitionRevision != DomainDefinitionRevision || series.Current == nil || series.Current.Value != 1 {
		t.Fatalf("series=%#v", series)
	}
}

func domainMetricByID(t *testing.T, bundle DomainBundle, id string) Metric {
	t.Helper()
	for _, item := range bundle.Items {
		if item.ID == id {
			return item
		}
	}
	t.Fatalf("domain metric %q missing: %#v", id, bundle.Items)
	return Metric{}
}

func mustDomainID(t *testing.T) string {
	t.Helper()
	value, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	return value
}
