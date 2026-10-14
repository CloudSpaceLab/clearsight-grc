//go:build postgres

package monitoring

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

const formSubjectLinkProjection = `l.id::text,l.tenant_id::text,l.legal_entity_id::text,
	l.subject_type,l.subject_id::text,l.form_template_id::text,l.form_template_version,
	f.name,f.status,l.linked_by::text,l.linked_at,l.ended_at,l.version`

func scanFormSubjectLink(row scanner) (FormSubjectLink, error) {
	var link FormSubjectLink
	err := row.Scan(&link.ID, &link.TenantID, &link.LegalEntityID, &link.SubjectType, &link.SubjectID,
		&link.FormTemplateID, &link.FormTemplateVersion, &link.FormName, &link.FormStatus,
		&link.LinkedBy, &link.LinkedAt, &link.EndedAt, &link.Version)
	return link, mapPostgresError(err)
}

// Create is idempotent for one active subject/form pair. Only a genuinely new
// association appends a LINKED event. No form revision or distribution changes.
func (r *PostgresRepository) CreateFormSubjectLink(ctx context.Context, input FormSubjectLink) (FormSubjectLink, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return FormSubjectLink{}, mapPostgresError(err)
	}
	defer tx.Rollback(ctx)
	var createdID string
	err = tx.QueryRow(ctx, `
		INSERT INTO form_subject_links (
			id,tenant_id,legal_entity_id,subject_type,subject_id,form_template_id,
			form_template_version,linked_by,linked_at)
		SELECT $1::uuid,f.tenant_id,f.legal_entity_id,$2,$3::uuid,f.id,f.version,$4::uuid,$5
		FROM monitoring_form_templates f
		WHERE f.tenant_id=(SELECT id FROM tenants WHERE id::text=$6 OR slug=$6)
		  AND f.legal_entity_id=$7::uuid
		  AND f.id=$8::uuid AND f.version=$9
		  AND f.status='ACTIVE' AND f.is_current
		ON CONFLICT DO NOTHING
		RETURNING id::text`,
		input.ID, input.SubjectType, input.SubjectID, input.LinkedBy, input.LinkedAt,
		input.TenantID, input.LegalEntityID, input.FormTemplateID, input.FormTemplateVersion,
	).Scan(&createdID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return FormSubjectLink{}, mapPostgresError(err)
	}
	if createdID != "" {
		if _, err = tx.Exec(ctx, `
			INSERT INTO form_subject_link_events(link_id,tenant_id,legal_entity_id,event_type,actor_id)
			SELECT l.id,l.tenant_id,l.legal_entity_id,'LINKED',$2::uuid
			FROM form_subject_links l WHERE l.id=$1::uuid`, createdID, input.LinkedBy); err != nil {
			return FormSubjectLink{}, mapPostgresError(err)
		}
	}
	// An empty INSERT can mean the form is no longer active or an existing
	// association won the race. Return the exact existing link only.
	link, err := scanFormSubjectLink(tx.QueryRow(ctx, `
		SELECT `+formSubjectLinkProjection+`
		FROM form_subject_links l
		JOIN monitoring_form_templates f ON f.tenant_id=l.tenant_id
			AND f.legal_entity_id=l.legal_entity_id
			AND f.id=l.form_template_id AND f.version=l.form_template_version
		WHERE l.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND l.legal_entity_id=$2::uuid AND l.subject_type=$3
		  AND l.subject_id=$4::uuid AND l.form_template_id=$5::uuid
		  AND l.ended_at IS NULL`,
		input.TenantID, input.LegalEntityID, input.SubjectType, input.SubjectID, input.FormTemplateID))
	if err != nil {
		return FormSubjectLink{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return FormSubjectLink{}, mapPostgresError(err)
	}
	return link, nil
}

func (r *PostgresRepository) ListFormSubjectLinks(ctx context.Context, tenantID, legalEntityID, subjectType, subjectID, cursor string, limit int) (FormSubjectLinkPage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+formSubjectLinkProjection+`
		FROM form_subject_links l
		JOIN monitoring_form_templates f ON f.tenant_id=l.tenant_id
			AND f.legal_entity_id=l.legal_entity_id
			AND f.id=l.form_template_id AND f.version=l.form_template_version
		WHERE l.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND l.legal_entity_id=$2::uuid AND l.subject_type=$3
		  AND l.subject_id=$4::uuid AND l.ended_at IS NULL
		  AND ($5::uuid IS NULL OR l.id < $5::uuid)
		ORDER BY l.id DESC LIMIT $6`,
		tenantID, legalEntityID, subjectType, subjectID, nullUUIDCursor(cursor), limit+1)
	if err != nil {
		return FormSubjectLinkPage{}, mapPostgresError(err)
	}
	defer rows.Close()
	page := FormSubjectLinkPage{Items: []FormSubjectLink{}}
	for rows.Next() {
		link, scanErr := scanFormSubjectLink(rows)
		if scanErr != nil {
			return FormSubjectLinkPage{}, scanErr
		}
		page.Items = append(page.Items, link)
	}
	if err := rows.Err(); err != nil {
		return FormSubjectLinkPage{}, mapPostgresError(err)
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		page.NextCursor = page.Items[limit-1].ID
	}
	return page, nil
}

func nullUUIDCursor(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (r *PostgresRepository) EndFormSubjectLink(ctx context.Context, tenantID, legalEntityID, subjectType, subjectID, linkID string, expectedVersion int64, actorID string, at time.Time, reason string) (FormSubjectLink, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return FormSubjectLink{}, mapPostgresError(err)
	}
	defer tx.Rollback(ctx)
	var endedID string
	err = tx.QueryRow(ctx, `
		UPDATE form_subject_links SET ended_at=$1,ended_by=$2::uuid,end_reason=$3,version=version+1
		WHERE id=$4::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$5 OR slug=$5)
		  AND legal_entity_id=$6::uuid AND subject_type=$7 AND subject_id=$8::uuid
		  AND version=$9 AND ended_at IS NULL
		RETURNING id::text`,
		at, actorID, reason, linkID, tenantID, legalEntityID, subjectType, subjectID, expectedVersion,
	).Scan(&endedID)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS (
			SELECT 1 FROM form_subject_links WHERE id=$1::uuid
			  AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2)
			  AND legal_entity_id=$3::uuid AND subject_type=$4 AND subject_id=$5::uuid
		)`, linkID, tenantID, legalEntityID, subjectType, subjectID).Scan(&exists); err != nil {
			return FormSubjectLink{}, mapPostgresError(err)
		}
		if exists {
			return FormSubjectLink{}, ErrConflict
		}
		return FormSubjectLink{}, ErrNotFound
	}
	if err != nil {
		return FormSubjectLink{}, mapPostgresError(err)
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO form_subject_link_events(link_id,tenant_id,legal_entity_id,event_type,actor_id,payload)
		SELECT id,tenant_id,legal_entity_id,'UNLINKED',$2::uuid,jsonb_build_object('reason',$3::text)
		FROM form_subject_links WHERE id=$1::uuid`, endedID, actorID, reason); err != nil {
		return FormSubjectLink{}, mapPostgresError(err)
	}
	link, err := scanFormSubjectLink(tx.QueryRow(ctx, `
		SELECT `+formSubjectLinkProjection+`
		FROM form_subject_links l
		JOIN monitoring_form_templates f ON f.tenant_id=l.tenant_id
			AND f.legal_entity_id=l.legal_entity_id
			AND f.id=l.form_template_id AND f.version=l.form_template_version
		WHERE l.id=$1::uuid`, endedID))
	if err != nil {
		return FormSubjectLink{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return FormSubjectLink{}, mapPostgresError(err)
	}
	return link, nil
}
