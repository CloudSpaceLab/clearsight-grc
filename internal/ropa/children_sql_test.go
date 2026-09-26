package ropa_test

import (
	"os"
	"strings"
	"testing"
)

func readPostgresSource(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(body)
}

func assertOrderedSQL(t *testing.T, source, name string, fragments ...string) {
	t.Helper()
	position := 0
	for _, fragment := range fragments {
		relative := strings.Index(source[position:], fragment)
		if relative < 0 {
			t.Errorf("%s must contain %q after the preceding SQL fragment", name, fragment)
			continue
		}
		position += relative + len(fragment)
	}
}

func TestChildInsertBuildersUseMigrationColumnsAndParentScope(t *testing.T) {
	source := normalizeSQL(readPostgresSource(t, "postgres.go"))
	tests := []struct {
		name     string
		columns  string
		values   string
		builder  string
		tableSQL string
	}{
		{
			name:     "data categories",
			builder:  "func buildinsertactivitydatacategoriessql() string",
			tableSQL: "ropa_processing_activity_data_categories (tenant_id, legal_entity_id, activity_id, category, sensitivity)",
			columns:  "tenant_id, legal_entity_id, activity_id, category, sensitivity",
			values:   "values ($1::uuid, $2::uuid, $3::uuid, $4, $5) on conflict do nothing",
		},
		{
			name:     "recipients",
			builder:  "func buildinsertactivityrecipientssql() string",
			tableSQL: "ropa_processing_activity_recipients (tenant_id, legal_entity_id, activity_id, recipient, recipient_kind, country_code, is_cross_border, transfer_basis)",
			columns:  "tenant_id, legal_entity_id, activity_id, recipient, recipient_kind, country_code, is_cross_border, transfer_basis",
			values:   "values ($1::uuid, $2::uuid, $3::uuid, $4, $5, nullif($6, ''), $7, $8) on conflict do nothing",
		},
		{
			name:     "systems",
			builder:  "func buildinsertactivitysystemssql() string",
			tableSQL: "ropa_processing_activity_systems (tenant_id, legal_entity_id, activity_id, system_name, system_kind)",
			columns:  "tenant_id, legal_entity_id, activity_id, system_name, system_kind",
			values:   "values ($1::uuid, $2::uuid, $3::uuid, $4, $5) on conflict do nothing",
		},
		{
			name:     "reviews",
			builder:  "func buildinsertactivityreviewssql() string",
			tableSQL: "ropa_processing_activity_reviews (id, tenant_id, legal_entity_id, activity_id, due_date, completed_at, outcome, reviewer_principal_id, created_at)",
			columns:  "id, tenant_id, legal_entity_id, activity_id, due_date, completed_at, outcome, reviewer_principal_id, created_at",
			values:   "values ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::date, $6::timestamptz, nullif($7, ''), nullif($8, '')::uuid, $9) on conflict do nothing",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, fragment := range []string{test.builder, test.tableSQL, test.columns, test.values} {
				if !strings.Contains(source, fragment) {
					t.Errorf("child INSERT builder must contain %q", fragment)
				}
			}
			assertOrderedSQL(t, source, test.name+" INSERT", test.builder, test.tableSQL, test.values)
		})
	}
}

func TestChildReadBuildersSelectEveryCollectionWithinParentScope(t *testing.T) {
	source := normalizeSQL(readPostgresSource(t, "current_postgres.go"))
	tests := []struct {
		name    string
		builder string
		columns string
	}{
		{
			name:    "data categories",
			builder: "func activitydatacategoriessql() string",
			columns: "select activity_id::text, category, sensitivity from ropa_processing_activity_data_categories",
		},
		{
			name:    "recipients",
			builder: "func activityrecipientssql() string",
			columns: "select activity_id::text, recipient, recipient_kind, coalesce(country_code, ''), is_cross_border, transfer_basis from ropa_processing_activity_recipients",
		},
		{
			name:    "systems",
			builder: "func activitysystemssql() string",
			columns: "select activity_id::text, system_name, system_kind from ropa_processing_activity_systems",
		},
		{
			name:    "reviews",
			builder: "func activityreviewssql() string",
			columns: "select activity_id::text, id::text, due_date, completed_at, coalesce(outcome, ''), coalesce(reviewer_principal_id::text, ''), created_at from ropa_processing_activity_reviews",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, fragment := range []string{test.builder, test.columns, "where tenant_id = $1::uuid and legal_entity_id = $2::uuid and activity_id=any($3::uuid[])"} {
				if !strings.Contains(source, fragment) {
					t.Errorf("child read builder must contain %q", fragment)
				}
			}
			assertOrderedSQL(t, source, test.name+" read", test.builder, test.columns)
		})
	}
}

func TestReadHydrationResolvesPrincipalDisplayNamesInOneBoundedQuery(t *testing.T) {
	source := normalizeSQL(readPostgresSource(t, "current_postgres.go"))
	for _, fragment := range []string{
		"func activityprincipalnamessql() string",
		"select id::text, display_name from principals where tenant_id = $1::uuid and id=any($2::uuid[])",
		"addprincipal(activity.ownerprincipalid)",
		"addprincipal(activity.requiredauthorityprincipalid)",
		"addprincipal(review.reviewerprincipalid)",
		"activity.ownerdisplayname = names[activity.ownerprincipalid]",
		"activity.requiredauthoritydisplayname = names[activity.requiredauthorityprincipalid]",
		"activity.reviews[index].reviewerdisplayname = names[activity.reviews[index].reviewerprincipalid]",
	} {
		if !strings.Contains(source, fragment) {
			t.Errorf("principal display-name hydration must contain %q", fragment)
		}
	}
}

func TestChildInsertExecutionBuildsOneStatementPerChildRow(t *testing.T) {
	source := normalizeSQL(readPostgresSource(t, "postgres.go"))
	if !strings.Contains(source, "func buildactivitychildinsertstatements(activity processingactivity) []activitychildinsertstatement") {
		t.Fatal("child INSERT execution must build a statement per child row")
	}
	if !strings.Contains(source, "for _, statement := range buildactivitychildinsertstatements(activity)") {
		t.Fatal("child INSERT execution must execute the per-row statement collection")
	}
	if strings.Contains(source, "recipients.values = append(recipients.values,") ||
		strings.Contains(source, "systems.values = append(systems.values,") ||
		strings.Contains(source, "reviews.values = append(reviews.values,") {
		t.Fatal("child INSERT execution must not flatten child rows into one statement")
	}
}

func TestChildInsertExecutionVerifiesEveryPerRowInsert(t *testing.T) {
	source := normalizeSQL(readPostgresSource(t, "postgres.go"))
	if !strings.Contains(source, "func execactivitychildinsert(ctx context.context, tx pgx.tx, name, query string, values []any) error") {
		t.Error("child INSERT execution must use a one-row helper")
	}
	if !strings.Contains(source, "if tag.rowsaffected() != 1") {
		t.Error("each child INSERT must verify exactly one affected row")
	}
}

func TestChildReplaceBuilderDeletesEveryScopedCollection(t *testing.T) {
	source := normalizeSQL(readPostgresSource(t, "postgres.go"))
	assertOrderedSQL(t, source, "child replace builder", "func builddeleteactivitychildrensql() []string")
	for _, table := range []string{
		"ropa_processing_activity_data_categories",
		"ropa_processing_activity_recipients",
		"ropa_processing_activity_systems",
		"ropa_processing_activity_reviews",
	} {
		fragment := "delete from " + table + " where tenant_id = $1::uuid and legal_entity_id = $2::uuid and activity_id = $3::uuid"
		if !strings.Contains(source, fragment) {
			t.Errorf("child replace builder must contain %q", fragment)
		}
	}
}

func TestPostgresCommandPathsPersistAndReplaceChildrenInsideTheTransaction(t *testing.T) {
	source := normalizeSQL(readPostgresSource(t, "postgres.go"))

	assertOrderedSQL(t, source, "CreateActivity child path",
		"func (r *postgresrepository) createactivity",
		"if tag.rowsaffected() != 1",
		"return processingactivity{}, errduplicate",
		"insertactivitychildren(ctx, tx, activity)",
		"appendactivityhistory(ctx, tx, activity, event)",
		"tx.commit(ctx)",
	)
	assertOrderedSQL(t, source, "ApplyActivityEvent child path",
		"func (r *postgresrepository) applyactivityevent",
		"updateactivitysql()",
		"replaceactivitychildren(ctx, tx, next)",
		"appendactivityhistory(ctx, tx, next, event)",
		"tx.commit(ctx)",
	)
	if !strings.Contains(source, "full replacement rather than a diff") {
		t.Error("ApplyActivityEvent must document that child replacement is a full replacement rather than a diff")
	}
}

func TestPostgresReadPathsHydrateExactAndBoundedPageChildren(t *testing.T) {
	commandSource := normalizeSQL(readPostgresSource(t, "postgres.go"))
	assertOrderedSQL(t, commandSource, "GetActivity child path",
		"func (r *postgresrepository) getactivity",
		"scanactivity(r.pool.queryrow",
		"loadactivitychildren(ctx, r.pool, &activity)",
		"return cloneprocessingactivity(activity), nil",
	)

	listSource := normalizeSQL(readPostgresSource(t, "current_postgres.go"))
	assertOrderedSQL(t, listSource, "ListActivities child path",
		"func (l *postgreslister) listactivities",
		"page.rows = page.rows[:filter.limit]",
		"loadactivitychildrenforpage(ctx, l.pool, page.rows)",
		"return page, nil",
	)
	if !strings.Contains(listSource, "bounded by the requested page size") {
		t.Error("ListActivities must document that child hydration is bounded by the requested page size")
	}
}
