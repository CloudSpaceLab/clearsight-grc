//go:build postgres

package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func (r *PostgresRepository) ListAdvancedFormLibrary(ctx context.Context, filter FormLibraryFilter, includeStatusFacets bool) (FormTemplatePage, error) {
	if filter.TenantID == "" || filter.LegalEntityID == "" {
		return FormTemplatePage{}, ErrInvalid
	}
	cursor, err := decodeFormLibraryCursor(filter.Cursor)
	if err != nil {
		return FormTemplatePage{}, err
	}
	order, err := normalizedFormLibrarySort(filter.Sort)
	if err != nil {
		return FormTemplatePage{}, err
	}
	cursorOperator, sortDirection := "<", "DESC"
	if order == FormLibraryUpdatedAsc {
		cursorOperator, sortDirection = ">", "ASC"
	}
	expression, err := combinedFormFilterExpression(filter)
	if err != nil {
		return FormTemplatePage{}, err
	}
	whereExpression, expressionArgs, nextArg := formFilterSQL(expression, "f", 4)
	cursorTime := ""
	if !cursor.UpdatedAt.IsZero() {
		cursorTime = cursor.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	limit := boundedFormLibraryLimit(filter.Limit)
	pageArgs := []any{filter.TenantID, filter.LegalEntityID, strings.TrimSpace(filter.Search)}
	pageArgs = append(pageArgs, expressionArgs...)
	pageArgs = append(pageArgs, cursorTime, cursor.ID, limit+1)
	pageSQL := fmt.Sprintf(`
	WITH latest_ids AS (
		SELECT DISTINCT ON (f.tenant_id,f.id) f.revision_id
		FROM monitoring_form_templates f JOIN tenants t ON t.id=f.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND f.legal_entity_id=$2::uuid
		ORDER BY f.tenant_id,f.id,f.version DESC
	)
	SELECT %s,COALESCE(active.version,0),COALESCE(active.status,'')
	FROM latest_ids latest
	JOIN monitoring_form_templates f ON f.revision_id=latest.revision_id
	LEFT JOIN monitoring_form_templates active ON active.tenant_id=f.tenant_id AND active.id=f.id AND active.is_current
	WHERE ($3='' OR lower(f.code) LIKE '%%'||lower($3)||'%%' OR lower(f.name) LIKE '%%'||lower($3)||'%%' OR lower(f.purpose) LIKE '%%'||lower($3)||'%%')
	  AND (%s)
	  AND ($%d='' OR (f.updated_at,f.id) %s (NULLIF($%d,'')::timestamptz,NULLIF($%d,'')::uuid))
	ORDER BY f.updated_at %s,f.id %s
	LIMIT $%d`, formProjection, whereExpression, nextArg, cursorOperator, nextArg, nextArg+1, sortDirection, sortDirection, nextArg+2)

	rows, err := r.pool.Query(ctx, pageSQL, pageArgs...)
	if err != nil {
		return FormTemplatePage{}, mapPostgresError(err)
	}
	defer rows.Close()
	page := FormTemplatePage{Items: make([]FormLibraryItem, 0, limit)}
	for rows.Next() {
		var activeVersion int64
		var activeStatus string
		value, scanErr := scanFormWithExtra(rows, &activeVersion, &activeStatus)
		if scanErr != nil {
			return FormTemplatePage{}, scanErr
		}
		page.Items = append(page.Items, FormLibraryItem{Template: value, ActiveVersion: activeVersion, ActiveStatus: LifecycleStatus(activeStatus)})
	}
	if err := rows.Err(); err != nil {
		return FormTemplatePage{}, mapPostgresError(err)
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[len(page.Items)-1].Template
		page.NextCursor = encodeFormLibraryCursor(formLibraryCursor{UpdatedAt: last.UpdatedAt, ID: last.ID})
	}

	total, facets, err := r.advancedFormLibraryMetadata(ctx, filter, expression, includeStatusFacets)
	if err != nil {
		return FormTemplatePage{}, err
	}
	page.Total = &total
	if includeStatusFacets {
		page.Facets = &facets
	}
	return page, nil
}

func (r *PostgresRepository) advancedFormLibraryMetadata(ctx context.Context, filter FormLibraryFilter, expression *FormFilterExpression, includeStatusFacets bool) (int, FormLibraryFacets, error) {
	fullSQL, fullArgs, nextArg := formFilterSQL(expression, "f", 4)
	facetExpression := expression
	if includeStatusFacets {
		facetExpression = formFilterExpressionWithoutField(expression, FormFilterStatus)
	}
	facetSQL, facetArgs, _ := formFilterSQL(facetExpression, "f", nextArg)
	args := []any{filter.TenantID, filter.LegalEntityID, strings.TrimSpace(filter.Search)}
	args = append(args, fullArgs...)
	args = append(args, facetArgs...)
	metadataSQL := fmt.Sprintf(`
	WITH latest_ids AS (
		SELECT DISTINCT ON (f.tenant_id,f.id) f.revision_id
		FROM monitoring_form_templates f JOIN tenants t ON t.id=f.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND f.legal_entity_id=$2::uuid
		ORDER BY f.tenant_id,f.id,f.version DESC
	), latest AS (
		SELECT f.* FROM latest_ids latest JOIN monitoring_form_templates f ON f.revision_id=latest.revision_id
	), total AS (
		SELECT count(*)::int AS value FROM latest f
		WHERE ($3='' OR lower(f.code) LIKE '%%'||lower($3)||'%%' OR lower(f.name) LIKE '%%'||lower($3)||'%%' OR lower(f.purpose) LIKE '%%'||lower($3)||'%%')
		  AND (%s)
	), status_counts AS (
		SELECT f.status,count(*)::int AS value FROM latest f
		WHERE ($3='' OR lower(f.code) LIKE '%%'||lower($3)||'%%' OR lower(f.name) LIKE '%%'||lower($3)||'%%' OR lower(f.purpose) LIKE '%%'||lower($3)||'%%')
		  AND (%s)
		GROUP BY f.status
	)
	SELECT total.value,COALESCE(jsonb_object_agg(status_counts.status,status_counts.value) FILTER (WHERE status_counts.status IS NOT NULL),'{}'::jsonb)
	FROM total LEFT JOIN status_counts ON true
	GROUP BY total.value`, fullSQL, facetSQL)

	var total int
	var rawFacets []byte
	if err := r.pool.QueryRow(ctx, metadataSQL, args...).Scan(&total, &rawFacets); err != nil {
		return 0, FormLibraryFacets{}, mapPostgresError(err)
	}
	facets := FormLibraryFacets{}
	if includeStatusFacets {
		encoded := map[string]int{}
		if err := json.Unmarshal(rawFacets, &encoded); err != nil {
			return 0, FormLibraryFacets{}, err
		}
		facets.Status = make(map[LifecycleStatus]int, len(encoded))
		for status, count := range encoded {
			facets.Status[LifecycleStatus(status)] = count
		}
	}
	return total, facets, nil
}

func formFilterSQL(expression *FormFilterExpression, alias string, startArg int) (string, []any, int) {
	if expression == nil {
		return "TRUE", nil, startArg
	}
	if expression.Kind == "condition" {
		placeholder := fmt.Sprintf("$%d", startArg)
		switch expression.Field {
		case FormFilterStatus:
			return alias + ".status=" + placeholder, []any{expression.Value}, startArg + 1
		case FormFilterOwner:
			return alias + ".owner_principal_id=NULLIF(" + placeholder + ",'')::uuid", []any{expression.Value}, startArg + 1
		case FormFilterProgram:
			return alias + ".program_id=NULLIF(" + placeholder + ",'')::uuid", []any{expression.Value}, startArg + 1
		case FormFilterUse:
			return alias + ".approved_uses @> ARRAY[" + placeholder + "]::text[]", []any{expression.Value}, startArg + 1
		case FormFilterTag:
			return alias + ".tags @> ARRAY[" + placeholder + "]::text[]", []any{expression.Value}, startArg + 1
		default:
			return "FALSE", nil, startArg
		}
	}
	parts := make([]string, 0, len(expression.Children))
	args := make([]any, 0, len(expression.Children))
	nextArg := startArg
	for index := range expression.Children {
		part, childArgs, next := formFilterSQL(&expression.Children[index], alias, nextArg)
		parts = append(parts, "("+part+")")
		args = append(args, childArgs...)
		nextArg = next
	}
	joiner := " AND "
	if expression.Operator == "or" {
		joiner = " OR "
	}
	return strings.Join(parts, joiner), args, nextArg
}
