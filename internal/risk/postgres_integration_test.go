//go:build postgres && postgresintegration

package risk

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresRiskLifecycleIsScopedVersionedAndAtomic(t *testing.T) {
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

	tenantID := mustRiskID(t)
	entityA := mustRiskID(t)
	entityB := mustRiskID(t)
	ownerID := mustRiskID(t)
	reviewerID := mustRiskID(t)
	authorizerID := mustRiskID(t)
	missingPrincipal := mustRiskID(t)
	parentScopeID := mustRiskID(t)
	childScopeID := mustRiskID(t)
	siblingScopeID := mustRiskID(t)
	otherEntityScopeID := mustRiskID(t)
	suffix := tenantID[len(tenantID)-8:]
	entityACode := "RISK-A-" + suffix
	entityBCode := "RISK-B-" + suffix
	now := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,$3)
	`, tenantID, "risk-"+suffix, "Risk Test "+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
			($1::uuid,$2::uuid,$3,'Risk Entity A','NG',$6),
			($4::uuid,$2::uuid,$5,'Risk Entity B','GH',$6)
	`, entityA, tenantID, entityACode, entityB, entityBCode, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
			($1::uuid,$2::uuid,'PERSON','Risk owner','ACTIVE',$5),
			($3::uuid,$2::uuid,'PERSON','Risk reviewer','ACTIVE',$5),
			($4::uuid,$2::uuid,'PERSON','Risk authorizer','ACTIVE',$5)
	`, ownerID, tenantID, reviewerID, authorizerID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO organization_scopes(
			id,tenant_id,legal_entity_id,parent_scope_id,code,name,kind,department_path,origin,status,valid_from
		) VALUES
			($1::uuid,$5::uuid,$6::uuid,NULL,$8,'Risk','DEPARTMENT',ARRAY['BANK','RISK'],'MANAGED','ACTIVE',$12),
			($2::uuid,$5::uuid,$6::uuid,$1::uuid,$9,'Risk Operations','DEPARTMENT',ARRAY['BANK','RISK','OPERATIONS'],'MANAGED','ACTIVE',$12),
			($3::uuid,$5::uuid,$6::uuid,NULL,$10,'Finance','DEPARTMENT',ARRAY['BANK','FINANCE'],'MANAGED','ACTIVE',$12),
			($4::uuid,$5::uuid,$7::uuid,NULL,$11,'Other entity risk','DEPARTMENT',ARRAY['BANK','RISK'],'MANAGED','ACTIVE',$12)
	`, parentScopeID, childScopeID, siblingScopeID, otherEntityScopeID, tenantID, entityA, entityB,
		"RISK-"+suffix, "RISK-OPS-"+suffix, "FIN-"+suffix, "OTHER-RISK-"+suffix, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	repository := NewPostgresRepository(pool)
	service := NewService(repository)
	service.Now = func() time.Time { return now }

	created, err := service.Create(ctx, CreateInput{
		TenantID: "risk-" + suffix, LegalEntityID: entityACode, OrganizationScopeID: parentScopeID, Code: "NET-" + suffix,
		Name: "Network resilience", Category: "Operational resilience",
		Statement: "Critical network service may exceed approved recovery tolerance.",
		Cause:     "Primary and recovery paths can become unavailable.", Event: "Network service interruption",
		Impact: "Customers cannot access critical services within the approved tolerance.",
		Scope:  json.RawMessage(`{"service":"critical-network"}`), OwnerPrincipalID: ownerID, ActorID: ownerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.TenantID != tenantID || created.LegalEntityID != entityA || created.OrganizationScopeID != parentScopeID || created.Version != 1 {
		t.Fatalf("created risk = %#v", created)
	}
	if _, err := service.Get(ctx, Scope{TenantID: "risk-" + suffix, LegalEntityID: entityBCode}, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity risk read error = %v", err)
	}

	if _, err := service.Create(ctx, CreateInput{
		TenantID: "risk-" + suffix, LegalEntityID: entityACode, OrganizationScopeID: otherEntityScopeID, Code: "CROSS-" + suffix,
		Name: "Cross-entity scope", Statement: "Cross-entity scope must be rejected.", Impact: "Invalid attribution.",
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-entity organization scope error = %v", err)
	}

	now = now.Add(time.Minute)
	riskWithAppetite, appetite, err := service.ActivateAppetite(ctx, AppetiteInput{
		TenantID: "risk-" + suffix, LegalEntityID: entityACode, RiskID: created.ID, ExpectedRiskVersion: 1,
		Statement: "Keep disruption below 30 minutes.", Rule: json.RawMessage(`{"max_minutes":30}`),
		Rationale: "Protect critical customer service.", OwnerPrincipalID: ownerID, ActorID: authorizerID,
		EffectiveFrom: now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if riskWithAppetite.Version != 2 || appetite.Version != 1 || appetite.RiskVersion != 2 {
		t.Fatalf("appetite result risk=%#v appetite=%#v", riskWithAppetite, appetite)
	}
	currentAppetite, err := repository.CurrentAppetite(ctx, Scope{TenantID: "risk-" + suffix, LegalEntityID: entityACode}, created.ID, now)
	if err != nil || currentAppetite == nil || currentAppetite.ID != appetite.ID {
		t.Fatalf("current appetite=%#v err=%v", currentAppetite, err)
	}

	now = now.Add(time.Minute)
	_, _, err = service.AddAssessment(ctx, AssessmentInput{
		TenantID: "risk-" + suffix, LegalEntityID: entityACode, RiskID: created.ID, ExpectedRiskVersion: 2,
		Kind: AssessmentResidual, MethodCode: "QUAL-5X5", MethodVersion: "v1",
		Dimensions: json.RawMessage(`{"likelihood":4,"impact":5}`), AppetiteStatementID: appetite.ID,
		AppetitePosition: AppetiteBreached, AppetiteRationale: "Recovery exceeds tolerance.", ActorID: missingPrincipal,
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid assessor error = %v", err)
	}
	afterFailure, err := service.Get(ctx, Scope{TenantID: "risk-" + suffix, LegalEntityID: entityACode}, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterFailure.Risk.Version != 2 || len(afterFailure.Assessments) != 0 {
		t.Fatalf("failed assessment changed risk: %#v", afterFailure)
	}

	now = now.Add(time.Minute)
	updated, assessment, err := service.AddAssessment(ctx, AssessmentInput{
		TenantID: "risk-" + suffix, LegalEntityID: entityACode, RiskID: created.ID, ExpectedRiskVersion: 2,
		Kind: AssessmentResidual, MethodCode: "QUAL-5X5", MethodVersion: "v1",
		Dimensions: json.RawMessage(`{"likelihood":4,"impact":5}`), AppetiteStatementID: appetite.ID,
		AppetitePosition: AppetiteBreached, AppetiteRationale: "Recovery exceeds tolerance.", ActorID: reviewerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 3 || assessment.RiskVersion != 3 {
		t.Fatalf("assessment version risk=%d assessment=%d", updated.Version, assessment.RiskVersion)
	}

	page, err := service.List(ctx, Scope{TenantID: "risk-" + suffix, LegalEntityID: entityACode}, ListFilter{
		AppetitePosition: AppetiteBreached, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Risk.ID != created.ID ||
		page.Items[0].LatestAssessment == nil || page.Items[0].LatestAssessment.ID != assessment.ID ||
		page.Items[0].ActiveAppetite == nil || page.Items[0].ActiveAppetite.ID != appetite.ID {
		t.Fatalf("risk page = %#v", page)
	}

	for table, want := range map[string]int{"risk_revisions": 3, "risk_events": 3} {
		var count int
		query := "SELECT count(*) FROM " + table + " WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND risk_id=$3::uuid"
		if err := pool.QueryRow(ctx, query, tenantID, entityA, created.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("%s count=%d want=%d", table, count, want)
		}
	}
	var outboxCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM outbox_events
		WHERE tenant_id=$1::uuid AND aggregate_type='RISK' AND aggregate_id=$2::uuid
	`, tenantID, created.ID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if outboxCount != 3 {
		t.Fatalf("risk outbox count=%d want=3", outboxCount)
	}

	now = now.Add(time.Minute)
	updatedAfterAssessment, err := service.Update(ctx, UpdateInput{
		TenantID: "risk-" + suffix, LegalEntityID: entityACode, RiskID: created.ID, ExpectedVersion: updated.Version,
		Name: updated.Name, Category: updated.Category, Statement: updated.Statement,
		Cause: updated.Cause, Event: updated.Event, Impact: updated.Impact, Scope: updated.Scope,
		OwnerPrincipalID: updated.OwnerPrincipalID, Status: StatusActive, ActorID: ownerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	staleFiltered, err := service.List(ctx, Scope{TenantID: "risk-" + suffix, LegalEntityID: entityACode}, ListFilter{
		AppetitePosition: AppetiteBreached, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updatedAfterAssessment.Version != assessment.RiskVersion+1 || len(staleFiltered.Items) != 0 {
		t.Fatalf("stale assessment drove BREACHED appetite filter: risk=%#v page=%#v", updatedAfterAssessment, staleFiltered)
	}
	unknownFiltered, err := service.List(ctx, Scope{TenantID: "risk-" + suffix, LegalEntityID: entityACode}, ListFilter{
		AppetitePosition: AppetiteUnknown, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(unknownFiltered.Items) != 1 || unknownFiltered.Items[0].Risk.ID != created.ID {
		t.Fatalf("stale assessment was not exposed as current UNKNOWN: %#v", unknownFiltered)
	}

	childRiskID := mustRiskID(t)
	siblingRiskID := mustRiskID(t)
	unattributedRiskID := mustRiskID(t)
	if _, err := pool.Exec(ctx, `
		INSERT INTO risks(id,tenant_id,legal_entity_id,organization_scope_id,code,name,statement,impact,scope,status,version,created_at,updated_at) VALUES
			($1::uuid,$4::uuid,$5::uuid,$7::uuid,$10,'Child scope risk','Child scope risk.','Material impact.','{}'::jsonb,'ACTIVE',1,$9,$9),
			($2::uuid,$4::uuid,$5::uuid,$8::uuid,$11,'Sibling scope risk','Sibling scope risk.','Material impact.','{}'::jsonb,'ACTIVE',1,$9,$9),
			($3::uuid,$4::uuid,$5::uuid,NULL,$12,'Unattributed risk','Unattributed risk.','Material impact.','{}'::jsonb,'ACTIVE',1,$9,$9)
	`, childRiskID, siblingRiskID, unattributedRiskID, tenantID, entityA, created.ID, childScopeID, siblingScopeID, now,
		"CHILD-"+suffix, "SIBLING-"+suffix, "UNATTRIBUTED-"+suffix); err != nil {
		t.Fatal(err)
	}
	scoped, err := service.List(ctx, Scope{TenantID: "risk-" + suffix, LegalEntityID: entityACode}, ListFilter{
		OrganizationScopeID: parentScopeID, OrganizationScopeIDs: []string{parentScopeID, childScopeID}, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if scoped.OrganizationScopeID != parentScopeID || len(scoped.Items) != 2 {
		t.Fatalf("organization scoped page = %#v", scoped)
	}
	seen := map[string]bool{}
	for _, item := range scoped.Items {
		seen[item.Risk.ID] = true
	}
	if !seen[created.ID] || !seen[childRiskID] || seen[siblingRiskID] || seen[unattributedRiskID] {
		t.Fatalf("organization scope membership = %#v", seen)
	}
}

func TestPostgresRiskIndicatorRejectsCrossEntityMonitoringCheck(t *testing.T) {
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

	tenantID := mustRiskID(t)
	entityA := mustRiskID(t)
	entityB := mustRiskID(t)
	programA := mustRiskID(t)
	programB := mustRiskID(t)
	checkA := mustRiskID(t)
	checkB := mustRiskID(t)
	riskID := mustRiskID(t)
	linkA := mustRiskID(t)
	linkB := mustRiskID(t)
	bindingA := mustRiskID(t)
	bindingB := mustRiskID(t)
	suffix := tenantID[len(tenantID)-8:]
	now := time.Date(2026, 10, 2, 16, 0, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,$3)
	`, tenantID, "risk-ind-"+suffix, "Risk Indicator "+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
			($1::uuid,$2::uuid,$3,'Indicator Entity A','NG',$6::timestamptz),
			($4::uuid,$2::uuid,$5,'Indicator Entity B','GH',$6::timestamptz)
	`, entityA, tenantID, "RIA-"+suffix, entityB, "RIB-"+suffix, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,scope,effective_from,created_at,updated_at,version) VALUES
			($1::uuid,$2::uuid,$3::uuid,$4,'Program A','ASSURANCE','ACTIVE','Risk','{}'::jsonb,$8::timestamptz,$8::timestamptz,$8::timestamptz,1),
			($5::uuid,$2::uuid,$6::uuid,$7,'Program B','ASSURANCE','ACTIVE','Risk','{}'::jsonb,$8::timestamptz,$8::timestamptz,$8::timestamptz,1)
	`, programA, tenantID, entityA, "PIA-"+suffix, programB, entityB, "PIB-"+suffix, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO monitoring_checks(
			id,tenant_id,program_id,code,name,claim,input_kind,binding_id,binding_version,source_rules,thresholds,
			freshness_minutes,minimum_coverage,failure_action,status,is_current,effective_from,version,created_at,updated_at) VALUES
			($1::uuid,$2::uuid,$3::uuid,'CHECK-A','Entity A check','A remains within bounds.','SOURCE',$4::uuid,1,
			 '[{"id":"state","field":"state","operator":"EQUALS","expected":"ok","risk_points":100}]'::jsonb,
			 '{"moderate_from":25,"high_from":50,"critical_from":75}'::jsonb,60,1,'REVIEW','ACTIVE',true,$8::timestamptz,1,$8::timestamptz,$8::timestamptz),
			($5::uuid,$2::uuid,$6::uuid,'CHECK-B','Entity B check','B remains within bounds.','SOURCE',$7::uuid,1,
			 '[{"id":"state","field":"state","operator":"EQUALS","expected":"ok","risk_points":100}]'::jsonb,
			 '{"moderate_from":25,"high_from":50,"critical_from":75}'::jsonb,60,1,'REVIEW','ACTIVE',true,$8::timestamptz,1,$8::timestamptz,$8::timestamptz)
	`, checkA, tenantID, programA, bindingA, checkB, programB, bindingB, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO risks(id,tenant_id,legal_entity_id,code,name,statement,impact,scope,status,version,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4,'Indicator risk','Indicator source may breach tolerance.','Material service impact.','{}'::jsonb,'ACTIVE',1,$5::timestamptz,$5::timestamptz)
	`, riskID, tenantID, entityA, "IND-"+suffix, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO risk_indicator_links(
			id,tenant_id,legal_entity_id,risk_id,risk_version,program_id,monitoring_check_id,monitoring_check_version,kind,measurement,created_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,2,$5::uuid,$6::uuid,1,'KRI','MONITORING_RISK_SCORE',$7::timestamptz)
	`, linkA, tenantID, entityA, riskID, programA, checkA, now); err != nil {
		t.Fatalf("same-entity indicator link rejected: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO risk_indicator_links(
			id,tenant_id,legal_entity_id,risk_id,risk_version,program_id,monitoring_check_id,monitoring_check_version,kind,measurement,created_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,3,$5::uuid,$6::uuid,1,'KCI','MONITORING_RISK_SCORE',$7::timestamptz)
	`, linkB, tenantID, entityA, riskID, programB, checkB, now.Add(time.Minute))
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("cross-entity indicator link error=%v, want foreign-key rejection", err)
	}
}

func mustRiskID(t *testing.T) string {
	t.Helper()
	value, err := newID()
	if err != nil {
		t.Fatal(err)
	}
	return value
}
