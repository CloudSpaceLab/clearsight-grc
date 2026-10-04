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

func TestMetricMatricesUseCurrentRiskAndProgramEvidenceTruth(t *testing.T) {
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

	tenantID := mustMatrixID(t)
	entityID := mustMatrixID(t)
	appetiteID := mustMatrixID(t)
	riskBreached := mustMatrixID(t)
	riskUnknown := mustMatrixID(t)
	riskSupported := mustMatrixID(t)
	riskFailed := mustMatrixID(t)
	programID := mustMatrixID(t)
	objectiveID := mustMatrixID(t)
	implementationID := mustMatrixID(t)
	definitionID := mustMatrixID(t)
	catalogLinkID := mustMatrixID(t)
	contractID := mustMatrixID(t)
	failedImplementationID := mustMatrixID(t)
	failedCatalogLinkID := mustMatrixID(t)
	failedContractID := mustMatrixID(t)

	now := time.Now().UTC().Truncate(time.Second)
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, execErr := pool.Exec(ctx, query, args...); execErr != nil {
			t.Fatal(execErr)
		}
	}
	mustExec(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Metric Matrix')`, tenantID, "metric-matrix-"+tenantID[len(tenantID)-8:])
	mustExec(`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($1::uuid,$2::uuid,'MATRIX-NG','Matrix Nigeria','NG',$3)`, entityID, tenantID, now.Add(-365*24*time.Hour))
	mustExec(`
		INSERT INTO risks(id,tenant_id,legal_entity_id,code,name,category,statement,impact,status,version,created_at,updated_at) VALUES
		($1::uuid,$5::uuid,$6::uuid,'R-BREACH','Breached risk','Operational','Breached statement','Material impact','ACTIVE',3,$7,$7),
		($2::uuid,$5::uuid,$6::uuid,'R-UNKNOWN','Unknown risk','Operational','Unknown statement','Material impact','ACTIVE',1,$7,$7),
		($3::uuid,$5::uuid,$6::uuid,'R-SUPPORTED','Supported risk','Cyber','Supported statement','Material impact','ACTIVE',2,$7,$7),
		($4::uuid,$5::uuid,$6::uuid,'R-FAILED','Failed risk','Cyber','Failed statement','Material impact','ACTIVE',2,$7,$7)`,
		riskBreached, riskUnknown, riskSupported, riskFailed, tenantID, entityID, now.Add(-time.Hour))
	mustExec(`
		INSERT INTO risk_appetite_statements(
			id,tenant_id,legal_entity_id,risk_id,risk_version,version,statement,rule,status,effective_from,created_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,2,1,'Operational appetite','{}'::jsonb,'ACTIVE',$5,$5)`,
		appetiteID, tenantID, entityID, riskBreached, now.Add(-24*time.Hour))
	mustExec(`
		INSERT INTO risk_assessments(
			tenant_id,legal_entity_id,risk_id,risk_version,assessment_kind,method_code,method_version,
			dimensions,assumptions,evidence_references,appetite_statement_id,appetite_position,assessed_at,created_at
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,3,'CURRENT','MATRIX','1','{}'::jsonb,'{}'::jsonb,'[]'::jsonb,$4::uuid,'BREACHED',$5,$5
		)`, tenantID, entityID, riskBreached, appetiteID, now.Add(-time.Hour))

	mustExec(`
		INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,effective_from)
		VALUES($1::uuid,$2::uuid,$3::uuid,'MATRIX-PROGRAM','Matrix Program','COMPLIANCE','ACTIVE','Risk',$4)`,
		programID, tenantID, entityID, now.Add(-30*24*time.Hour))
	mustExec(`
		INSERT INTO control_objectives(id,tenant_id,program_id,code,name,outcome,status)
		VALUES($1::uuid,$2::uuid,$3::uuid,'MATRIX-OBJ','Matrix objective','Evidence supports operation','ACTIVE')`,
		objectiveID, tenantID, programID)
	mustExec(`
		INSERT INTO control_implementations(
			id,tenant_id,program_id,objective_id,name,description,implementation_type,status,effective_from
		) VALUES
		($1::uuid,$3::uuid,$4::uuid,$5::uuid,'Supported control','Supported implementation','CHECKLIST','IMPLEMENTED',$6),
		($2::uuid,$3::uuid,$4::uuid,$5::uuid,'Failed control','Failed implementation','CHECKLIST','IMPLEMENTED',$6)`,
		implementationID, failedImplementationID, tenantID, programID, objectiveID, now.Add(-30*24*time.Hour))
	mustExec(`
		INSERT INTO control_definitions(id,tenant_id,code,name,objective,status,version,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,'MATRIX-CONTROL','Matrix control','Control objective','ACTIVE',1,$3,$3)`,
		definitionID, tenantID, now.Add(-30*24*time.Hour))
	mustExec(`
		INSERT INTO control_catalog_implementation_links(
			id,tenant_id,legal_entity_id,definition_id,program_id,implementation_id,created_at
		) VALUES
		($1::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7::uuid,$8),
		($2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$9::uuid,$8)`,
		catalogLinkID, failedCatalogLinkID, tenantID, entityID, definitionID, programID, implementationID, now.Add(-20*24*time.Hour), failedImplementationID)
	mustExec(`
		INSERT INTO risk_control_links(tenant_id,legal_entity_id,risk_id,risk_version,catalog_link_id,created_at)
		VALUES
		($1::uuid,$2::uuid,$3::uuid,2,$4::uuid,$7),
		($1::uuid,$2::uuid,$5::uuid,2,$6::uuid,$7)`,
		tenantID, entityID, riskSupported, catalogLinkID, riskFailed, failedCatalogLinkID, now.Add(-10*24*time.Hour))
	mustExec(`
		INSERT INTO evidence_contracts(
			id,tenant_id,program_id,control_implementation_id,code,name,claim,
			freshness_minutes,minimum_coverage,independence_required,contradiction_policy,failure_action,status
		) VALUES
		($1::uuid,$3::uuid,$4::uuid,$5::uuid,'SUPPORTED-EVIDENCE','Supported evidence','Supported claim',1440,0.8,false,'REVIEW','FLAG','ACTIVE'),
		($2::uuid,$3::uuid,$4::uuid,$6::uuid,'FAILED-EVIDENCE','Failed evidence','Failed claim',1440,0.8,false,'REVIEW','FLAG','ACTIVE')`,
		contractID, failedContractID, tenantID, programID, implementationID, failedImplementationID)
	mustExec(`
		INSERT INTO evidence_assessments(tenant_id,program_id,contract_id,conclusion,coverage,assessed_at,valid_until)
		VALUES
		($1::uuid,$2::uuid,$3::uuid,'SUPPORTED',1,$5,$6),
		($1::uuid,$2::uuid,$4::uuid,'UNSUPPORTED',1,$5,$6)`,
		tenantID, programID, contractID, failedContractID, now.Add(-time.Hour), now.Add(24*time.Hour))

	repository := NewMatrixRepository(pool)
	appetite, err := repository.RiskAppetiteMatrix(ctx, tenantID, entityID, now)
	if err != nil {
		t.Fatal(err)
	}
	if appetite.Population != 4 {
		t.Fatalf("appetite population=%d rows=%#v", appetite.Population, appetite.Rows)
	}
	operational := matrixRowByLabel(t, appetite, "Operational")
	if operational.Cells[2].Count != 1 || operational.Cells[3].Count != 1 {
		t.Fatalf("operational appetite=%#v", operational)
	}

	assurance, err := repository.AssuranceCoverageMatrix(ctx, tenantID, entityID, now)
	if err != nil {
		t.Fatal(err)
	}
	if assurance.Population != 4 {
		t.Fatalf("assurance population=%d rows=%#v", assurance.Population, assurance.Rows)
	}
	cyber := matrixRowByLabel(t, assurance, "Cyber")
	if cyber.Cells[0].Count != 1 || cyber.Cells[2].Count != 1 {
		t.Fatalf("cyber assurance=%#v", cyber)
	}
	operationalAssurance := matrixRowByLabel(t, assurance, "Operational")
	if operationalAssurance.Cells[3].Count != 2 {
		t.Fatalf("operational assurance=%#v", operationalAssurance)
	}
}

func matrixRowByLabel(t *testing.T, matrix Matrix, label string) MatrixRow {
	t.Helper()
	for _, row := range matrix.Rows {
		if row.Label == label {
			return row
		}
	}
	t.Fatalf("matrix row %q missing: %#v", label, matrix.Rows)
	return MatrixRow{}
}

func mustMatrixID(t *testing.T) string {
	t.Helper()
	value, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	return value
}
