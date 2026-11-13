//go:build postgres

package monitoring

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

const questionnaireProjection = `q.id::text,q.program_id::text,q.adoption_id::text,q.edition_id::text,q.form_template_id::text,q.form_template_version,f.name,q.linked_by::text,q.linked_at,q.version`

func scanFrameworkQuestionnaire(row scanner) (FrameworkQuestionnaire, error) {
	var value FrameworkQuestionnaire
	err := row.Scan(&value.ID, &value.ProgramID, &value.AdoptionID, &value.EditionID, &value.FormID, &value.FormVersion, &value.FormName, &value.LinkedBy, &value.LinkedAt, &value.Version)
	return value, mapPostgresError(err)
}

func (r *PostgresRepository) GetFrameworkQuestionnaire(ctx context.Context, tenant, entity, program, adoption string) (FrameworkQuestionnaire, error) {
	return scanFrameworkQuestionnaire(r.pool.QueryRow(ctx, `SELECT `+questionnaireProjection+` FROM framework_adoption_questionnaires q JOIN monitoring_form_templates f ON f.tenant_id=q.tenant_id AND f.legal_entity_id=q.legal_entity_id AND f.id=q.form_template_id AND f.version=q.form_template_version WHERE q.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND q.legal_entity_id=$2::uuid AND q.program_id=$3::uuid AND q.adoption_id=$4::uuid AND q.ended_at IS NULL`, tenant, entity, program, adoption))
}

// A Program and adoption row lock serializes changes. The old link, new link
// and their audit events commit atomically; the form revision is rechecked.
func (r *PostgresRepository) SetFrameworkQuestionnaire(ctx context.Context, input FrameworkQuestionnaire, expected int64) (FrameworkQuestionnaire, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return FrameworkQuestionnaire{}, mapPostgresError(err)
	}
	defer tx.Rollback(ctx)
	var edition string
	err = tx.QueryRow(ctx, `SELECT a.edition_id::text FROM program_framework_adoptions a JOIN tenants t ON t.id=a.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND a.legal_entity_id=$2::uuid AND a.program_id=$3::uuid AND a.id=$4::uuid AND a.status IN ('DRAFT','ACTIVE') FOR UPDATE OF a`, input.TenantID, input.LegalEntityID, input.ProgramID, input.AdoptionID).Scan(&edition)
	if err != nil {
		return FrameworkQuestionnaire{}, mapPostgresError(err)
	}
	var lockedForm string
	err = tx.QueryRow(ctx, `SELECT f.id::text FROM monitoring_form_templates f WHERE f.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND f.legal_entity_id=$2::uuid AND f.id=$3::uuid AND f.version=$4 AND f.status='ACTIVE' AND f.is_current FOR SHARE`, input.TenantID, input.LegalEntityID, input.FormID, input.FormVersion).Scan(&lockedForm)
	if err != nil {
		return FrameworkQuestionnaire{}, mapPostgresError(err)
	}
	var currentID, currentForm string
	var currentVersion, currentFormVersion int64
	err = tx.QueryRow(ctx, `SELECT id::text,version,form_template_id::text,form_template_version FROM framework_adoption_questionnaires WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND legal_entity_id=$2::uuid AND program_id=$3::uuid AND adoption_id=$4::uuid AND ended_at IS NULL FOR UPDATE`, input.TenantID, input.LegalEntityID, input.ProgramID, input.AdoptionID).Scan(&currentID, &currentVersion, &currentForm, &currentFormVersion)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return FrameworkQuestionnaire{}, mapPostgresError(err)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		if expected != 0 {
			return FrameworkQuestionnaire{}, ErrConflict
		}
	} else {
		if currentVersion != expected {
			return FrameworkQuestionnaire{}, ErrConflict
		}
		if currentForm == input.FormID && currentFormVersion == input.FormVersion {
			return scanFrameworkQuestionnaire(tx.QueryRow(ctx, `SELECT `+questionnaireProjection+` FROM framework_adoption_questionnaires q JOIN monitoring_form_templates f ON f.tenant_id=q.tenant_id AND f.legal_entity_id=q.legal_entity_id AND f.id=q.form_template_id AND f.version=q.form_template_version WHERE q.id=$1::uuid`, currentID))
		}
		_, err = tx.Exec(ctx, `UPDATE framework_adoption_questionnaires SET ended_at=$2,ended_by=$3::uuid,end_reason='Replaced by Program owner',version=version+1 WHERE id=$1::uuid`, currentID, input.LinkedAt, input.LinkedBy)
		if err != nil {
			return FrameworkQuestionnaire{}, mapPostgresError(err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO framework_adoption_questionnaire_events(link_id,tenant_id,legal_entity_id,event_type,actor_id,payload) SELECT id,tenant_id,legal_entity_id,'UNLINKED',$2::uuid,jsonb_build_object('reason','Replaced by Program owner') FROM framework_adoption_questionnaires WHERE id=$1::uuid`, currentID, input.LinkedBy)
		if err != nil {
			return FrameworkQuestionnaire{}, mapPostgresError(err)
		}
	}
	var inserted string
	err = tx.QueryRow(ctx, `INSERT INTO framework_adoption_questionnaires(id,tenant_id,legal_entity_id,program_id,adoption_id,edition_id,form_template_id,form_template_version,linked_by,linked_at,version) SELECT $1::uuid,f.tenant_id,f.legal_entity_id,$4::uuid,$5::uuid,$6::uuid,f.id,f.version,$7::uuid,$8,$11 FROM monitoring_form_templates f WHERE f.tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND f.legal_entity_id=$3::uuid AND f.id=$9::uuid AND f.version=$10 AND f.status='ACTIVE' AND f.is_current RETURNING id::text`, input.ID, input.TenantID, input.LegalEntityID, input.ProgramID, input.AdoptionID, edition, input.LinkedBy, input.LinkedAt, input.FormID, input.FormVersion, expected+1).Scan(&inserted)
	if err != nil {
		return FrameworkQuestionnaire{}, mapPostgresError(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO framework_adoption_questionnaire_events(link_id,tenant_id,legal_entity_id,event_type,actor_id) SELECT id,tenant_id,legal_entity_id,'LINKED',$2::uuid FROM framework_adoption_questionnaires WHERE id=$1::uuid`, inserted, input.LinkedBy)
	if err != nil {
		return FrameworkQuestionnaire{}, mapPostgresError(err)
	}
	value, err := scanFrameworkQuestionnaire(tx.QueryRow(ctx, `SELECT `+questionnaireProjection+` FROM framework_adoption_questionnaires q JOIN monitoring_form_templates f ON f.tenant_id=q.tenant_id AND f.legal_entity_id=q.legal_entity_id AND f.id=q.form_template_id AND f.version=q.form_template_version WHERE q.id=$1::uuid`, inserted))
	if err != nil {
		return FrameworkQuestionnaire{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return FrameworkQuestionnaire{}, mapPostgresError(err)
	}
	return value, nil
}
