//go:build postgres

package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed source_employee_manifest.json
var sourceEmployeeManifest []byte

const sourceEmployeeProfile = "fidelity-source-samples-v1"

type sourceEmployeeSource struct {
	File          string   `json:"file"`
	SHA256        string   `json:"sha256"`
	Sheet         string   `json:"sheet"`
	Cells         []string `json:"cells"`
	Archive       string   `json:"archive,omitempty"`
	ArchiveSHA256 string   `json:"archive_sha256,omitempty"`
}

type sourceEmployee struct {
	DisplayName   string                 `json:"display_name"`
	Function      string                 `json:"function"`
	PositionTitle string                 `json:"position_title"`
	Sources       []sourceEmployeeSource `json:"sources"`
	PrincipalID   string                 `json:"principal_id"`
	PositionID    string                 `json:"position_id"`
	ExternalRef   string                 `json:"external_ref"`
	PositionCode  string                 `json:"position_code"`
	Username      string                 `json:"username"`
	RoleCode      string                 `json:"role_code"`
}

type sourceEmployeeReceipt struct {
	Profile       string           `json:"profile"`
	TenantID      string           `json:"tenant_id"`
	LegalEntityID string           `json:"legal_entity_id"`
	Count         int              `json:"count"`
	Created       int              `json:"created"`
	Employees     []sourceEmployee `json:"employees"`
}

func sourceEmployeeID(key string) string {
	h := sha256.Sum256([]byte(key))
	h[6] = (h[6] & 0x0f) | 0x50
	h[8] = (h[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}

func sourceEmployees() ([]sourceEmployee, error) {
	var manifest struct {
		Profile   string           `json:"profile"`
		Employees []sourceEmployee `json:"employees"`
	}
	if err := json.Unmarshal(sourceEmployeeManifest, &manifest); err != nil {
		return nil, err
	}
	if manifest.Profile != sourceEmployeeProfile || len(manifest.Employees) != 17 {
		return nil, fmt.Errorf("invalid source employee manifest")
	}
	seen := map[string]bool{}
	for i := range manifest.Employees {
		person := &manifest.Employees[i]
		key := strings.ToLower(strings.ReplaceAll(person.DisplayName, " ", "-"))
		person.ExternalRef = sourceEmployeeProfile + ":person:" + key
		person.PrincipalID = identity.DemoSourceEmployeePrincipalID(person.DisplayName)
		person.Username = strings.ToLower(strings.Fields(person.DisplayName)[0]) + "@demo.com"
		person.RoleCode = "EVIDENCE_RESPONDENT"
		person.PositionID = sourceEmployeeID(sourceEmployeeProfile + ":position:" + key)
		person.PositionCode = "FIDELITY-SAMPLE-" + strings.ToUpper(key)
		if key == "" || seen[key] || !strings.HasPrefix(person.PositionTitle, "Sample ") || person.Function == "" || len(person.Sources) == 0 {
			return nil, fmt.Errorf("invalid source employee entry")
		}
		seen[key] = true
		for _, source := range person.Sources {
			if source.File == "" || len(source.SHA256) != 64 || source.Sheet == "" || len(source.Cells) == 0 {
				return nil, fmt.Errorf("missing employee source coordinates")
			}
		}
	}
	return manifest.Employees, nil
}

// seedSourceEmployees provisions sample directory people and the existing
// evidence-performer role. Demo login credentials live only in DemoAuthenticator.
// No SCIM identity, mail delivery address, admin or signatory grant is created.
func seedSourceEmployees(ctx context.Context, pool *pgxpool.Pool, seed bankverticals.SeedConfig) (sourceEmployeeReceipt, error) {
	people, err := sourceEmployees()
	if err != nil {
		return sourceEmployeeReceipt{}, err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return sourceEmployeeReceipt{}, err
	}
	defer tx.Rollback(ctx)
	var tenant, entity string
	err = tx.QueryRow(ctx, `SELECT t.id::text,le.id::text FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id
	 WHERE t.id='00000000-0000-4000-8000-000000000001' AND t.slug='clearsight-demo'
	 AND (t.id::text=$1 OR t.slug=$1) AND le.id='00000000-0000-4000-8000-000000000002'
	 AND (le.id::text=$2 OR le.code=$2) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR le.valid_until>clock_timestamp())
	 FOR UPDATE OF t,le`, seed.TenantID, seed.LegalEntityID).Scan(&tenant, &entity)
	if err != nil {
		return sourceEmployeeReceipt{}, fmt.Errorf("source employees require the existing clearsight-demo tenant and demo legal entity: %w", err)
	}
	var roleID string
	err = tx.QueryRow(ctx, `SELECT id::text FROM role_templates WHERE tenant_id=$1::uuid AND id='00000000-0000-4000-8000-000000000408' AND code='EVIDENCE_RESPONDENT' AND responsibilities=ARRAY['PERFORMER']::text[] AND capabilities=ARRAY['respond:evidence']::text[] AND valid_until IS NULL`, tenant).Scan(&roleID)
	if err != nil {
		return sourceEmployeeReceipt{}, fmt.Errorf("source employees require the existing evidence-performer role: %w", err)
	}
	receipt := sourceEmployeeReceipt{Profile: sourceEmployeeProfile, TenantID: tenant, LegalEntityID: entity, Count: len(people), Employees: people}
	at := seed.Now.UTC()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	for _, person := range people {
		var matches int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM principals WHERE (id=$1::uuid OR (tenant_id=$2::uuid AND (external_ref=$3 OR lower(btrim(display_name))=lower($4))))`, person.PrincipalID, tenant, person.ExternalRef, person.DisplayName).Scan(&matches); err != nil {
			return sourceEmployeeReceipt{}, err
		}
		if matches == 0 {
			if _, err = tx.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON',$3,$4,'ACTIVE',$5)`, person.PrincipalID, tenant, person.ExternalRef, person.DisplayName, at); err != nil {
				return sourceEmployeeReceipt{}, err
			}
			receipt.Created++
		} else {
			var exact bool
			err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM principals WHERE id=$1::uuid AND tenant_id=$2::uuid AND kind='PERSON' AND external_ref=$3 AND display_name=$4 AND status='ACTIVE' AND valid_until IS NULL)`, person.PrincipalID, tenant, person.ExternalRef, person.DisplayName).Scan(&exact)
			if err != nil {
				return sourceEmployeeReceipt{}, err
			}
			if matches != 1 || !exact {
				return sourceEmployeeReceipt{}, fmt.Errorf("source employee identity conflict for %s; existing record preserved", person.DisplayName)
			}
		}
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM org_positions WHERE id=$1::uuid OR (tenant_id=$2::uuid AND (code=$3 OR occupant_principal_id=$4::uuid))`, person.PositionID, tenant, person.PositionCode, person.PrincipalID).Scan(&matches); err != nil {
			return sourceEmployeeReceipt{}, err
		}
		if matches == 0 {
			if _, err = tx.Exec(ctx, `INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,function_name,occupant_principal_id,department_path,valid_from,version) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7::uuid,$8,$9,1)`, person.PositionID, tenant, entity, person.PositionCode, person.PositionTitle, person.Function, person.PrincipalID, []string{person.Function}, at); err != nil {
				return sourceEmployeeReceipt{}, err
			}
		} else {
			var exact bool
			err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM org_positions WHERE id=$1::uuid AND tenant_id=$2::uuid AND legal_entity_id=$3::uuid AND code=$4 AND title=$5 AND function_name=$6 AND occupant_principal_id=$7::uuid AND department_path=$8 AND parent_position_id IS NULL AND valid_until IS NULL AND version=1)`, person.PositionID, tenant, entity, person.PositionCode, person.PositionTitle, person.Function, person.PrincipalID, []string{person.Function}).Scan(&exact)
			if err != nil {
				return sourceEmployeeReceipt{}, err
			}
			if matches != 1 || !exact {
				return sourceEmployeeReceipt{}, fmt.Errorf("source employee position conflict for %s; existing record preserved", person.DisplayName)
			}
		}
		bindingID := sourceEmployeeID(person.ExternalRef + ":performer-binding")
		if _, err = tx.Exec(ctx, `INSERT INTO position_role_bindings(id,tenant_id,position_id,role_template_id,scope,priority,valid_from) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,jsonb_build_object('legal_entity_id',$5::text),100,$6) ON CONFLICT(id) DO NOTHING`, bindingID, tenant, person.PositionID, roleID, entity, at); err != nil {
			return sourceEmployeeReceipt{}, err
		}
		var bindingExact bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM position_role_bindings WHERE id=$1::uuid AND tenant_id=$2::uuid AND position_id=$3::uuid AND role_template_id=$4::uuid AND scope=jsonb_build_object('legal_entity_id',$5::text) AND priority=100 AND valid_until IS NULL)`, bindingID, tenant, person.PositionID, roleID, entity).Scan(&bindingExact)
		if err != nil {
			return sourceEmployeeReceipt{}, err
		}
		if !bindingExact {
			return sourceEmployeeReceipt{}, fmt.Errorf("source employee performer binding conflict for %s; existing record preserved", person.DisplayName)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return sourceEmployeeReceipt{}, err
	}
	return receipt, nil
}
