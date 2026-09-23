package ropa_test

import (
	"os"
	"strings"
	"testing"
)

const upFile = "../../migrations/000092_ropa_register.up.sql"
const downFile = "../../migrations/000092_ropa_register.down.sql"

func readMigration(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func normalizeSQL(body string) string {
	sql := strings.ReplaceAll(strings.ToLower(strings.Join(strings.Fields(body), " ")), ",", ", ")
	for strings.Contains(sql, ",  ") {
		sql = strings.ReplaceAll(sql, ",  ", ", ")
	}
	for strings.Contains(sql, "( ") {
		sql = strings.ReplaceAll(sql, "( ", "(")
	}
	for strings.Contains(sql, " )") {
		sql = strings.ReplaceAll(sql, " )", ")")
	}
	return sql
}

func createTableDefinition(t *testing.T, normalizedSQL, table string) string {
	t.Helper()
	start := strings.Index(normalizedSQL, "create table "+table+" ")
	if start < 0 {
		t.Fatalf("missing create table %s", table)
	}
	remainder := normalizedSQL[start:]
	end := strings.Index(remainder, ");")
	if end < 0 {
		t.Fatalf("create table %s has no closing delimiter", table)
	}
	return remainder[:end+2]
}

func triggerFunctionBody(t *testing.T, sql, functionName string) string {
	t.Helper()
	marker := "create function " + functionName + "() returns trigger"
	start := strings.Index(sql, marker)
	if start < 0 {
		t.Fatalf("missing trigger function %s", functionName)
	}
	definitionStart := start + len(marker)
	asStart := strings.Index(sql[definitionStart:], "as $$")
	if asStart < 0 {
		t.Fatalf("trigger function %s has no AS $$ body", functionName)
	}
	bodyStart := definitionStart + asStart + len("as $$")
	bodyEnd := strings.Index(sql[bodyStart:], "$$;")
	if bodyEnd < 0 {
		t.Fatalf("trigger function %s has an unterminated AS $$ body", functionName)
	}
	return sql[bodyStart : bodyStart+bodyEnd]
}

func TestUpMigrationIsASingleTransaction(t *testing.T) {
	body := readMigration(t, upFile)
	if got := strings.Count(body, "BEGIN;"); got != 1 {
		t.Fatalf("expected exactly one BEGIN; got %d", got)
	}
	if got := strings.Count(body, "COMMIT;"); got != 1 {
		t.Fatalf("expected exactly one COMMIT; got %d", got)
	}
	trimmed := strings.TrimSpace(body)
	if !strings.HasPrefix(trimmed, "BEGIN;") {
		t.Fatal("migration must open with BEGIN;")
	}
	if !strings.HasSuffix(trimmed, "COMMIT;") {
		t.Fatal("migration must close with COMMIT;")
	}
}

func TestUpMigrationCreatesEveryOwnedTable(t *testing.T) {
	body := readMigration(t, upFile)
	for _, table := range []string{
		"ropa_processing_activities",
		"ropa_processing_activity_revisions",
		"ropa_processing_activity_data_categories",
		"ropa_processing_activity_recipients",
		"ropa_processing_activity_systems",
		"ropa_processing_activity_reviews",
		"ropa_events",
		"ropa_register_summary",
	} {
		if !strings.Contains(body, "CREATE TABLE "+table) {
			t.Errorf("migration must create %s", table)
		}
	}
}

func TestLegalEntityIsNotNullAndImmutable(t *testing.T) {
	body := readMigration(t, upFile)
	if !strings.Contains(body, "legal_entity_id uuid NOT NULL") {
		t.Error("legal_entity_id must be NOT NULL")
	}
	if !strings.Contains(body, "ropa_legal_entity_immutable") {
		t.Error("migration must install the legal-entity immutability trigger")
	}
}

func TestCurrentReadIndexesCoverTheKeyset(t *testing.T) {
	body := readMigration(t, upFile)
	sql := normalizeSQL(body)
	for _, index := range []string{
		"create index ropa_register_keyset_idx on ropa_processing_activities(tenant_id, legal_entity_id, (case status when 'new' then 1 when 'open' then 2 when 'closed' then 3 end), (coalesce(next_review_date, '0001-01-01'::date)), id) where end_date is null",
		"create index ropa_register_history_keyset_idx on ropa_processing_activities(tenant_id, legal_entity_id, (case status when 'new' then 1 when 'open' then 2 when 'closed' then 3 end), (coalesce(next_review_date, '0001-01-01'::date)), id)",
		"create index ropa_lawful_basis_idx on ropa_processing_activities(tenant_id, legal_entity_id, lawful_basis)",
		"create index ropa_owner_idx on ropa_processing_activities(tenant_id, legal_entity_id, owner_principal_id)",
		"create index ropa_reviews_activity_idx on ropa_processing_activity_reviews(tenant_id, legal_entity_id, activity_id, created_at, id)",
	} {
		if !strings.Contains(sql, index) {
			t.Errorf("migration must create matching index shape %q", index)
		}
	}
	if !strings.Contains(sql, "repository order by and cursor predicates must use this exact expression") {
		t.Error("migration must document the exact keyset expressions for the later repository query")
	}
	if strings.Contains(sql, "ropa_events_replay_idx") {
		t.Error("ropa_events unique constraint already creates the replay lookup index")
	}
}

func TestActivityScopeKeySupportsCompositeChildReferences(t *testing.T) {
	body := readMigration(t, upFile)
	if !strings.Contains(body, "UNIQUE (id, tenant_id, legal_entity_id)") {
		t.Error("activity table must expose the scoped key required by child foreign keys")
	}
}

func TestRecipientCrossBorderColumnsAndCoherenceAreLoadBearing(t *testing.T) {
	body := readMigration(t, upFile)
	recipients := createTableDefinition(t, normalizeSQL(body), "ropa_processing_activity_recipients")

	for _, required := range []string{
		"country_code text,",
		"is_cross_border boolean not null default false,",
		"transfer_basis text not null default 'not_applicable',",
		"constraint ropa_recipients_country_code_ck check (country_code is null or country_code ~ '^[a-z]{2}$')",
		"constraint ropa_recipients_cross_border_coherence_ck check ((is_cross_border and country_code is not null) or (not is_cross_border and country_code is null and transfer_basis = 'not_applicable'))",
	} {
		if !strings.Contains(recipients, required) {
			t.Errorf("recipient table must include %q", required)
		}
	}
	if !strings.Contains(body, "country_code ~ '^[A-Z]{2}$'") {
		t.Error("country-code check must require exactly two uppercase letters")
	}
	if strings.Contains(recipients, "transfer_basis text not null default ''") {
		t.Error("recipient transfer basis must not retain the free-text empty default")
	}
}

func TestRecipientTransferSafeguardVocabularyAndCrossBorderIndex(t *testing.T) {
	body := readMigration(t, upFile)
	sql := normalizeSQL(body)
	recipients := createTableDefinition(t, sql, "ropa_processing_activity_recipients")

	requiredConstraint := "constraint ropa_recipients_transfer_basis_ck check (transfer_basis in ('adequacy', 'approved_instrument', 'recognised_lawful_basis', 'consent', 'standard_contract_clauses', 'binding_corporate_rules', 'certification', 'not_applicable'))"
	if !strings.Contains(recipients, requiredConstraint) {
		t.Errorf("recipient table must constrain transfer_basis to the Article 45 / Schedule 5 vocabulary: %s", requiredConstraint)
	}
	requiredIndex := "create index ropa_recipients_cross_border_idx on ropa_processing_activity_recipients(tenant_id, legal_entity_id, is_cross_border)"
	if !strings.Contains(sql, requiredIndex) {
		t.Errorf("migration must create cross-border filter index %q", requiredIndex)
	}
	if !strings.Contains(body, "NDPA Article 45 and Schedule 5 safeguards") {
		t.Error("migration must document the statutory source for the transfer vocabulary")
	}
}

func TestActivityLegalEntityScopeUsesCompositeForeignKey(t *testing.T) {
	body := readMigration(t, upFile)
	sql := strings.ToLower(strings.Join(strings.Fields(body), " "))
	if !strings.Contains(sql, "foreign key (legal_entity_id, tenant_id) references legal_entities(id, tenant_id)") {
		t.Error("activity legal-entity scope must reference legal_entities with the tenant in the foreign key")
	}
}

func TestActivityChildReferencesUseTheRetainedScopedKey(t *testing.T) {
	body := readMigration(t, upFile)
	sql := strings.ToLower(strings.Join(strings.Fields(body), " "))
	required := "foreign key (activity_id, tenant_id, legal_entity_id) references ropa_processing_activities(id, tenant_id, legal_entity_id)"
	if got := strings.Count(sql, required); got != 5 {
		t.Errorf("expected five activity references to use UNIQUE (id, tenant_id, legal_entity_id); got %d", got)
	}
	if strings.Contains(sql, "references ropa_processing_activities(tenant_id, legal_entity_id, id)") {
		t.Error("activity references must not target an unavailable parent key order")
	}
}

func TestAllLegalEntityReferencesUseTheAvailableCompositeKey(t *testing.T) {
	body := readMigration(t, upFile)
	sql := strings.ToLower(strings.Join(strings.Fields(body), " "))
	required := "foreign key (legal_entity_id, tenant_id) references legal_entities(id, tenant_id)"
	if got := strings.Count(sql, required); got != 3 {
		t.Errorf("expected activity, event and summary legal-entity references to use the available key; got %d", got)
	}
	if strings.Contains(sql, "references legal_entities(tenant_id, id)") {
		t.Error("legal-entity references must not use unavailable reversed parent key order")
	}
}

func TestRopaMigrationDropsRedundantActivityUniqueKeys(t *testing.T) {
	body := readMigration(t, upFile)
	sql := strings.ToLower(strings.Join(strings.Fields(body), " "))
	for _, redundant := range []string{
		"unique (id, tenant_id)",
		"ropa_activities_legal_entity_uk",
	} {
		if strings.Contains(sql, redundant) {
			t.Errorf("migration must not retain redundant activity uniqueness key %q", redundant)
		}
	}
}

func TestActivityPrincipalAndProgramReferencesAreTenantScoped(t *testing.T) {
	body := readMigration(t, upFile)
	sql := normalizeSQL(body)
	for _, required := range []string{
		"constraint ropa_processing_activities_owner_tenant_fk foreign key (owner_principal_id, tenant_id) references principals(id, tenant_id)",
		"constraint ropa_processing_activities_required_authority_tenant_fk foreign key (required_authority_principal_id, tenant_id) references principals(id, tenant_id)",
		"constraint ropa_processing_activities_program_tenant_fk foreign key (program_id, tenant_id) references programs(id, tenant_id)",
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("migration must use tenant-scoped reference %q", required)
		}
	}
	for _, unsafe := range []string{
		"owner_principal_id uuid references principals(id)",
		"required_authority_principal_id uuid references principals(id)",
	} {
		if strings.Contains(sql, unsafe) {
			t.Errorf("migration must not retain unscoped principal reference %q", unsafe)
		}
	}
}

func TestReviewReferenceIsTenantScoped(t *testing.T) {
	body := readMigration(t, upFile)
	sql := normalizeSQL(body)
	required := "constraint ropa_processing_activity_reviews_reviewer_tenant_fk foreign key (reviewer_principal_id, tenant_id) references principals(id, tenant_id)"
	if !strings.Contains(sql, required) {
		t.Errorf("reviewer reference must use the tenant-scoped principal key: %s", required)
	}
	if strings.Contains(sql, "reviewer_principal_id uuid references principals(id)") {
		t.Error("reviewer principal reference must not remain single-column")
	}
}

func TestUserEventActorIsScopedToTheEventTenant(t *testing.T) {
	body := readMigration(t, upFile)
	sql := normalizeSQL(body)
	for _, required := range []string{
		"create function validate_ropa_event_actor_scope() returns trigger",
		"create trigger ropa_event_actor_tenant_scope before insert on ropa_events for each row execute function validate_ropa_event_actor_scope()",
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("event actor scope guard must contain %q", required)
		}
	}
	if strings.Contains(sql, "foreign key (actor_id, tenant_id) references principals(id, tenant_id)") {
		t.Error("service actors must not be forced through an unconditional principal foreign key")
	}
	functionBody := triggerFunctionBody(t, sql, "validate_ropa_event_actor_scope")
	for _, required := range []string{
		"if new.actor_type = 'user' then",
		"if not exists (",
		"from principals p where p.id = new.actor_id and p.tenant_id = new.tenant_id",
		"raise exception",
		"return new;",
	} {
		if !strings.Contains(functionBody, required) {
			t.Errorf("event actor scope function must contain %q", required)
		}
	}
}

func TestProcessingActivityEventAggregateIsScopedToTheEvent(t *testing.T) {
	body := readMigration(t, upFile)
	sql := normalizeSQL(body)
	for _, required := range []string{
		"create function validate_ropa_event_aggregate_scope() returns trigger",
		"create trigger ropa_event_aggregate_tenant_legal_entity_scope before insert on ropa_events for each row execute function validate_ropa_event_aggregate_scope()",
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("event aggregate scope guard must contain %q", required)
		}
	}
	functionBody := triggerFunctionBody(t, sql, "validate_ropa_event_aggregate_scope")
	for _, required := range []string{
		"if new.aggregate_type = 'processing_activity' then",
		"if not exists (",
		"from ropa_processing_activities a where a.id = new.aggregate_id and a.tenant_id = new.tenant_id and a.legal_entity_id = new.legal_entity_id",
		"raise exception",
		"return new;",
	} {
		if !strings.Contains(functionBody, required) {
			t.Errorf("event aggregate scope function must contain %q", required)
		}
	}
}

func TestEventLedgerIsImmutable(t *testing.T) {
	body := readMigration(t, upFile)
	sql := normalizeSQL(body)
	required := "create trigger ropa_event_immutable before update or delete on ropa_events for each row execute function protect_ropa_event()"
	if !strings.Contains(sql, required) {
		t.Errorf("event ledger must install %q", required)
	}
	functionBody := triggerFunctionBody(t, sql, "protect_ropa_event")
	for _, fragment := range []string{"raise exception", "return new;"} {
		if !strings.Contains(functionBody, fragment) {
			t.Errorf("protect_ropa_event must contain %q", fragment)
		}
	}
}

func TestProjectionJSONAndTimestampChecksAreLoadBearing(t *testing.T) {
	body := readMigration(t, upFile)
	sql := normalizeSQL(body)
	for _, required := range []string{
		"constraint ropa_processing_activities_updated_at_order_ck check (updated_at >= created_at)",
		"constraint ropa_processing_activity_revisions_snapshot_object_ck check (jsonb_typeof(snapshot) = 'object')",
		"constraint ropa_events_payload_object_ck check (jsonb_typeof(payload) = 'object')",
		"constraint ropa_register_summary_population_nonnegative_ck check (population >= 0)",
		"constraint ropa_register_summary_excluded_nonnegative_ck check (excluded is null or excluded >= 0)",
		"constraint ropa_register_summary_unknown_nonnegative_ck check (unknown is null or unknown >= 0)",
		"constraint ropa_register_summary_counts_object_ck check (jsonb_typeof(counts) = 'object')",
		"constraint ropa_register_summary_projection_version_nonblank_ck check (btrim(projection_version) <> '')",
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("migration must include integrity check %q", required)
		}
	}
	for _, nullable := range []string{"excluded integer,", "unknown integer,"} {
		if !strings.Contains(sql, nullable) {
			t.Errorf("nullable summary count must remain %q", nullable)
		}
	}
}

func TestDownMigrationRefusesWhenAuthoritativeOrHistoryRowsExist(t *testing.T) {
	body := readMigration(t, downFile)
	sql := normalizeSQL(body)
	for _, table := range []string{
		"ropa_processing_activities",
		"ropa_processing_activity_revisions",
		"ropa_processing_activity_data_categories",
		"ropa_processing_activity_recipients",
		"ropa_processing_activity_systems",
		"ropa_processing_activity_reviews",
		"ropa_events",
	} {
		guard := "exists (select 1 from " + table + ")"
		if !strings.Contains(sql, guard) {
			t.Errorf("down migration must guard authoritative/history table %s", table)
		}
	}
	if !strings.Contains(sql, "raise exception") {
		t.Error("down migration must refuse to destroy authoritative or history rows")
	}
	if strings.Contains(sql, "exists (select 1 from ropa_register_summary)") {
		t.Error("ropa_register_summary is disposable and must not block rollback")
	}
	if !strings.Contains(sql, "ropa_register_summary is a disposable projection and may be discarded") {
		t.Error("down migration must document why the summary projection is discarded")
	}
}
