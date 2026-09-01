//go:build postgres

package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *PostgresDistributionStore) ListCompletedResponses(ctx context.Context, query CompletedResponseQuery) (CompletedResponsePage, error) {
	cursor, err := normalizeCompletedResponseQuery(&query)
	if err != nil {
		return CompletedResponsePage{}, err
	}
	if cursor.ID != "" {
		var cursorID pgtype.UUID
		if err := cursorID.Scan(cursor.ID); err != nil || !cursorID.Valid {
			return CompletedResponsePage{}, fmt.Errorf("completed response cursor is invalid")
		}
	}
	modes := scoringModeStrings(query.Modes)
	bands := concernBandStrings(query.Bands)
	states := scoreStateStrings(query.States)
	args := []any{
		query.TenantID, query.LegalEntityID, strings.TrimSpace(query.FormTemplateID), query.FormTemplateVersion,
		strings.TrimSpace(query.SubjectType), strings.TrimSpace(query.SubjectID), modes, bands, states,
		query.RawMinimum, query.RawMaximum, query.AdverseMinimum, query.AdverseMaximum,
		query.CompletedFrom, query.CompletedUntil, query.CurrentOnly, query.PrincipalID,
	}
	cursorSQL, orderSQL := postgresCompletedResponseOrder(query.Sort, cursor, &args)
	currentIndexSQL := ""
	if query.CurrentOnly {
		currentIndexSQL = " AND r.is_current"
	}
	scoreStateIndexSQL := postgresCompletedResponseScoreStateIndexPredicate(query.States)
	limitPlaceholder := fmt.Sprintf("$%d", len(args)+1)
	args = append(args, query.Limit+1)
	rows, err := s.repo.pool.Query(ctx, `
		SELECT `+responseRevisionProjection+`,d.form_template_id::text,d.form_template_version,d.title,d.subject_type,d.subject_id::text
		FROM capture_response_revisions r
		JOIN tenants t ON t.id=r.tenant_id
		JOIN capture_form_distributions d
		  ON d.id=r.distribution_id AND d.tenant_id=r.tenant_id AND d.legal_entity_id=r.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND r.legal_entity_id=$2::uuid
		  AND ($3='' OR d.form_template_id::text=$3) AND ($4=0 OR d.form_template_version=$4)
		  AND ($5='' OR d.subject_type=$5) AND ($6='' OR d.subject_id::text=$6)
		  AND (cardinality($7::text[])=0 OR r.score_mode=ANY($7::text[]))
		  AND (cardinality($8::text[])=0 OR r.concern_band=ANY($8::text[]))
		  AND (cardinality($9::text[])=0 OR r.score_state=ANY($9::text[]))
		  AND ($10::numeric IS NULL OR r.raw_score >= $10) AND ($11::numeric IS NULL OR r.raw_score <= $11)
		  AND ($12::numeric IS NULL OR r.adverse_score >= $12) AND ($13::numeric IS NULL OR r.adverse_score <= $13)
		  AND ($14::timestamptz IS NULL OR r.created_at >= $14) AND ($15::timestamptz IS NULL OR r.created_at <= $15)
		  AND (NOT $16::boolean OR r.is_current)`+currentIndexSQL+scoreStateIndexSQL+`
		  AND (`+completedResponseSubjectVisibilitySQL("$17")+`)
		  AND (`+cursorSQL+`)
		ORDER BY `+orderSQL+`
		LIMIT `+limitPlaceholder, args...)
	if err != nil {
		return CompletedResponsePage{}, fmt.Errorf("list completed responses: %w", err)
	}
	defer rows.Close()
	values := make([]CompletedResponseSummary, 0, query.Limit+1)
	for rows.Next() {
		var formID, title, subjectType, subjectID string
		var formVersion int64
		revision, scanErr := scanPostgresResponseRevisionWithExtra(rows, &formID, &formVersion, &title, &subjectType, &subjectID)
		if scanErr != nil {
			return CompletedResponsePage{}, fmt.Errorf("scan completed response: %w", scanErr)
		}
		values = append(values, CompletedResponseSummary{
			ID: revision.ID, TenantID: revision.TenantID, LegalEntityID: revision.LegalEntityID, DistributionID: revision.DistributionID,
			FormTemplateID: formID, FormTemplateVersion: formVersion, Title: title, SubjectType: subjectType, SubjectID: subjectID,
			Revision: revision.Revision, Current: revision.Current, State: revision.State, Score: revision.Score, CompletedAt: revision.CreatedAt.UTC(),
		})
	}
	if err := rows.Err(); err != nil {
		return CompletedResponsePage{}, fmt.Errorf("iterate completed responses: %w", err)
	}
	page := CompletedResponsePage{Items: values}
	if len(values) > query.Limit {
		page.Items = values[:query.Limit]
		page.NextCursor = encodeCompletedResponseCursor(page.Items[len(page.Items)-1], query.Sort)
	}
	return page, nil
}

func (s *PostgresDistributionStore) GetCompletedResponse(ctx context.Context, tenantID, legalEntityID, principalID, revisionID string) (CompletedResponseSummary, ResponseRevision, error) {
	var formID, title, subjectType, subjectID string
	var formVersion int64
	revision, err := scanPostgresResponseRevisionWithExtra(s.repo.pool.QueryRow(ctx, `
		SELECT `+responseRevisionProjection+`,d.form_template_id::text,d.form_template_version,d.title,d.subject_type,d.subject_id::text
		FROM capture_response_revisions r
		JOIN tenants t ON t.id=r.tenant_id
		JOIN capture_form_distributions d
		  ON d.id=r.distribution_id AND d.tenant_id=r.tenant_id AND d.legal_entity_id=r.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND r.legal_entity_id=$2::uuid AND r.id::text=$3
		  AND (`+completedResponseSubjectVisibilitySQL("$4")+`)`,
		tenantID, legalEntityID, revisionID, principalID), &formID, &formVersion, &title, &subjectType, &subjectID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CompletedResponseSummary{}, ResponseRevision{}, ErrNotFound
		}
		return CompletedResponseSummary{}, ResponseRevision{}, err
	}
	summary := CompletedResponseSummary{
		ID: revision.ID, TenantID: revision.TenantID, LegalEntityID: revision.LegalEntityID, DistributionID: revision.DistributionID,
		FormTemplateID: formID, FormTemplateVersion: formVersion, Title: title, SubjectType: subjectType, SubjectID: subjectID,
		Revision: revision.Revision, Current: revision.Current, State: revision.State, Score: revision.Score, CompletedAt: revision.CreatedAt.UTC(),
	}
	return summary, revision, nil
}

func postgresCompletedResponseOrder(sortOrder ResponseSort, cursor completedResponseCursor, args *[]any) (string, string) {
	column, direction := "r.adverse_score", "DESC"
	if sortOrder == ResponseSortNewest {
		column = ""
	} else if sortOrder == ResponseSortRawAsc {
		column, direction = "r.raw_score", "ASC"
	} else if sortOrder == ResponseSortRawDesc {
		column = "r.raw_score"
	}
	order := "r.created_at DESC,r.id DESC"
	if column != "" {
		order = column + " " + direction + " NULLS LAST," + order
	}
	if cursor.ID == "" {
		return "TRUE", order
	}
	if column == "" {
		start := len(*args) + 1
		*args = append(*args, cursor.CompletedAt.UTC(), cursor.ID)
		return fmt.Sprintf("(r.created_at,r.id) < ($%d,$%d::uuid)", start, start+1), order
	}
	start := len(*args) + 1
	*args = append(*args, cursor.Score, cursor.CompletedAt.UTC(), cursor.ID)
	if cursor.Score == nil {
		return fmt.Sprintf("%s IS NULL AND (r.created_at,r.id) < ($%d,$%d::uuid)", column, start+1, start+2), order
	}
	operator := "<"
	if direction == "ASC" {
		operator = ">"
	}
	return fmt.Sprintf("(%s %s $%d OR %s IS NULL OR (%s=$%d AND (r.created_at,r.id) < ($%d,$%d::uuid)))", column, operator, start, column, column, start, start+1, start+2), order
}

func postgresCompletedResponseScoreStateIndexPredicate(states []ResponseScoreState) string {
	if len(states) == 0 {
		return ""
	}
	hasFinal, hasProvisional := false, false
	for _, state := range states {
		switch state {
		case ResponseScoreFinal:
			hasFinal = true
		case ResponseScoreProvisional:
			hasProvisional = true
		default:
			return ""
		}
	}
	if hasFinal && hasProvisional {
		return " AND r.score_state IN ('FINAL','PROVISIONAL')"
	}
	if hasFinal {
		return " AND r.score_state='FINAL'"
	}
	return " AND r.score_state='PROVISIONAL'"
}

func completedResponseSubjectVisibilitySQL(principalPlaceholder string) string {
	return `CASE upper(d.subject_type)
		WHEN 'PROGRAM' THEN EXISTS (
			SELECT 1 FROM programs visible_program
			WHERE visible_program.tenant_id=r.tenant_id
			  AND visible_program.legal_entity_id=r.legal_entity_id
			  AND visible_program.id::text=d.subject_id::text
			  AND ` + recipientSubjectVisibilityPredicate("visible_program", principalPlaceholder) + `
		)
		WHEN 'MATTER' THEN EXISTS (
			SELECT 1 FROM matters visible_matter
			WHERE visible_matter.tenant_id=r.tenant_id
			  AND visible_matter.legal_entity_id=r.legal_entity_id
			  AND visible_matter.id::text=d.subject_id::text
			  AND ` + recipientSubjectVisibilityPredicate("visible_matter", principalPlaceholder) + `
		)
		WHEN 'VENDOR_RELATIONSHIP' THEN EXISTS (
			SELECT 1 FROM third_party_relationships visible_relationship
			WHERE visible_relationship.tenant_id=r.tenant_id
			  AND visible_relationship.legal_entity_id=r.legal_entity_id
			  AND visible_relationship.id=d.subject_id
			  AND visible_relationship.business_owner_principal_id::text=` + principalPlaceholder + `
		)
		ELSE true
	END`
}

func scoringModeStrings(values []formcontract.ScoringMode) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = string(value)
	}
	return result
}

func concernBandStrings(values []formcontract.ConcernBand) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = string(value)
	}
	return result
}

func scoreStateStrings(values []ResponseScoreState) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = string(value)
	}
	return result
}
