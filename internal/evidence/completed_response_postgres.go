//go:build postgres

package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
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
		query.CompletedFrom, query.CompletedUntil, query.CurrentOnly, query.PrincipalID, time.Now().UTC(),
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
		`+completedResponseRequestJoinsSQL()+`
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
		  AND (`+completedResponseDiscoverySQL(17, 18)+`)
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
		`+completedResponseRequestJoinsSQL()+`
		WHERE (t.id::text=$1 OR t.slug=$1) AND r.legal_entity_id=$2::uuid AND r.id::text=$3
		  AND (`+completedResponseDiscoverySQL(4, 5)+`)`,
		tenantID, legalEntityID, revisionID, principalID, time.Now().UTC()), &formID, &formVersion, &title, &subjectType, &subjectID)
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

func (s *PostgresDistributionStore) GetCompletedResponseForExecution(ctx context.Context, tenantID, revisionID string) (CompletedResponseSummary, error) {
	if s == nil || s.repo == nil || strings.TrimSpace(tenantID) == "" || strings.TrimSpace(revisionID) == "" {
		return CompletedResponseSummary{}, ErrDistributionInvalid
	}
	var formID, title, subjectType, subjectID string
	var formVersion int64
	revision, err := scanPostgresResponseRevisionWithExtra(s.repo.pool.QueryRow(ctx, `
		SELECT `+responseRevisionProjection+`,d.form_template_id::text,d.form_template_version,d.title,d.subject_type,d.subject_id::text
		FROM capture_response_revisions r
		JOIN tenants t ON t.id=r.tenant_id
		JOIN capture_form_distributions d
		  ON d.id=r.distribution_id AND d.tenant_id=r.tenant_id AND d.legal_entity_id=r.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND r.id::text=$2
		  AND r.state IN ('FINAL','PROVISIONAL') AND (r.score_state IN ('FINAL','PROVISIONAL') OR EXISTS(SELECT 1 FROM capture_response_assessments a WHERE a.tenant_id=r.tenant_id AND a.response_revision_id=r.id))`, tenantID, revisionID), &formID, &formVersion, &title, &subjectType, &subjectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return CompletedResponseSummary{}, ErrNotFound
	}
	if err != nil {
		return CompletedResponseSummary{}, err
	}
	return CompletedResponseSummary{
		ID: revision.ID, TenantID: revision.TenantID, LegalEntityID: revision.LegalEntityID, DistributionID: revision.DistributionID,
		FormTemplateID: formID, FormTemplateVersion: formVersion, Title: title, SubjectType: subjectType, SubjectID: subjectID,
		Revision: revision.Revision, Current: revision.Current, State: revision.State, Score: revision.Score, CompletedAt: revision.CreatedAt.UTC(),
	}, nil
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

func completedResponseSubjectVisibilitySQL(principalPlaceholder, atPlaceholder string) string {
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
			  AND (visible_relationship.business_owner_principal_id::text=` + principalPlaceholder + ` OR ` + readAllCapabilitySQL(principalPlaceholder, atPlaceholder) + `)
		)
		ELSE true
	END`
}

func readAllCapabilitySQL(principalPlaceholder, atPlaceholder string) string {
	return `EXISTS (
		SELECT 1
		FROM org_positions read_position
		JOIN position_role_bindings read_binding ON read_binding.tenant_id=read_position.tenant_id AND read_binding.position_id=read_position.id
		JOIN role_templates read_role ON read_role.tenant_id=read_binding.tenant_id AND read_role.id=read_binding.role_template_id
		WHERE read_position.tenant_id=r.tenant_id
		  AND read_position.legal_entity_id=r.legal_entity_id
		  AND read_position.occupant_principal_id::text=` + principalPlaceholder + `
		  AND read_position.valid_from<=` + atPlaceholder + ` AND (read_position.valid_until IS NULL OR ` + atPlaceholder + `<read_position.valid_until)
		  AND read_binding.valid_from<=` + atPlaceholder + ` AND (read_binding.valid_until IS NULL OR ` + atPlaceholder + `<read_binding.valid_until)
		  AND (NOT (read_binding.scope ? 'legal_entity_id') OR read_binding.scope->>'legal_entity_id'=r.legal_entity_id::text)
		  AND read_role.valid_from<=` + atPlaceholder + ` AND (read_role.valid_until IS NULL OR ` + atPlaceholder + `<read_role.valid_until)
		  AND 'read:all'=ANY(read_role.capabilities)
	)`
}

func completedResponseRequestJoinsSQL() string {
	return `LEFT JOIN capture_submissions submission ON submission.id=r.submission_id AND submission.tenant_id=r.tenant_id
	LEFT JOIN capture_requests req ON req.id=submission.request_id AND req.tenant_id=r.tenant_id AND req.legal_entity_id=r.legal_entity_id
	` + documentWorkflowJoinsSQL()
}

func completedResponseVisibilitySQL(principal, at int) string {
	return `submission.id IS NOT NULL AND req.id IS NOT NULL AND CASE WHEN req.origin_type IN ('THIRD_PARTY_ASSESSMENT','THIRD_PARTY_WORK')
	THEN ` + documentRevisionScopeSQL() + ` AND (` + documentReadAuthoritySQLAt(principal, at) + `)
	ELSE (` + completedResponseSubjectVisibilitySQL(fmt.Sprintf("$%d", principal), fmt.Sprintf("$%d", at)) + `) END`
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

func completedResponseDiscoverySQL(principal, at int) string {
	return "(" + completedResponseVisibilitySQL(principal, at) + ") OR (" + completedResponseReviewerSQL(principal, at) + ")"
}
func completedResponseReviewerSQL(principal, at int) string {
	return `submission.id IS NOT NULL AND req.id IS NOT NULL AND r.id IS NOT NULL AND r.state IN ('FINAL','PROVISIONAL') AND d.subject_type='VENDOR_RELATIONSHIP' AND COALESCE(req.origin_type,'') NOT IN ('THIRD_PARTY_WORK','THIRD_PARTY_ASSESSMENT') AND ` + documentRevisionScopeSQL() + ` AND COALESCE(submission.submitted_by::text,'')<>` + fmt.Sprintf("$%d", principal) + ` AND EXISTS(SELECT 1 FROM third_party_relationships vr WHERE vr.tenant_id=r.tenant_id AND vr.legal_entity_id=r.legal_entity_id AND vr.id=d.subject_id) AND EXISTS(SELECT 1 FROM jsonb_array_elements(req.fields) field WHERE field->'assessment'->>'mode' IN ('MANUAL','AUTOMATIC_REVIEW')) AND ` + authority.PostgresReadRouteSQL("r", "id", "FORM_RESPONSE", "FORMS.RESPONSE.ASSESS", "REVIEWER", principal, at)
}
