package database

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

var tableDDL = regexp.MustCompile(`(?i)\b(CREATE|DROP)\s+TABLE\s+(?:IF\s+(?:NOT\s+)?EXISTS\s+)?(?:public\.)?([a-z_][a-z0-9_]*)`)

var allowedOwnershipClassifications = map[string]struct{}{
	"active authoritative state":                   {},
	"active projection":                            {},
	"active infrastructure ledger":                 {},
	"explicitly reserved for a named future phase": {},
	"deprecated/migration-only":                    {},
	"removable":                                    {},
}

func TestLiveTablesHaveExactlyOneOwnershipEntry(t *testing.T) {
	root := repositoryRoot(t)
	live := liveTablesFromMigrations(t, filepath.Join(root, "migrations"))
	registered := ownershipRegisters(
		t,
		filepath.Join(root, "docs", "architecture", "durable-schema-ownership.md"),
		filepath.Join(root, "docs", "architecture", "durable-schema-ownership.d"),
	)

	missing := difference(live, registered)
	extra := difference(registered, live)
	if len(missing) != 0 || len(extra) != 0 {
		t.Fatalf("durable schema ownership mismatch\nmissing register entries: %v\nregister entries not live: %v", missing, extra)
	}
}

func TestEveryUpMigrationHasDownPair(t *testing.T) {
	root := repositoryRoot(t)
	up, err := filepath.Glob(filepath.Join(root, "migrations", "*.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if len(up) == 0 {
		t.Fatal("no up migrations found")
	}
	for _, path := range up {
		down := strings.TrimSuffix(path, ".up.sql") + ".down.sql"
		if _, err := os.Stat(down); err != nil {
			t.Errorf("migration %s has no matching down migration: %v", filepath.Base(path), err)
		}
	}
}

func TestFormResponsePolicySchemaKeepsScopeAndIdempotencyInDatabase(t *testing.T) {
	root := repositoryRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "migrations", "000065_form_scoring_and_response_policies.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(strings.Join(strings.Fields(string(raw)), " "))
	for _, required := range []string{
		"foreign key (legal_entity_id,tenant_id) references legal_entities(id,tenant_id)",
		"foreign key (tenant_id,legal_entity_id,form_template_id,form_template_version) references monitoring_form_templates(tenant_id,legal_entity_id,id,version)",
		"unique(tenant_id,legal_entity_id,policy_id,policy_version,response_revision_id)",
		"on form_response_policy_adverse_episodes(tenant_id,legal_entity_id,policy_code,subject_type,subject_id) where state='open'",
		"where aggregate_type='form_response_policy'",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("form-response policy migration is missing database invariant %q", required)
		}
	}
}

func TestAutomationPolicyTimestampRepairUsesFollowUpMigration(t *testing.T) {
	root := repositoryRoot(t)
	prior, err := os.ReadFile(filepath.Join(root, "migrations", "000065_form_scoring_and_response_policies.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(prior)), "alter table automation_policies") {
		t.Fatal("an applied form-policy migration must not be rewritten to repair automation policy timestamps")
	}
	followUp, err := os.ReadFile(filepath.Join(root, "migrations", "000066_automation_policy_timestamps.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(strings.Join(strings.Fields(string(followUp)), " "))
	for _, required := range []string{"alter table automation_policies", "add column created_at", "add column updated_at"} {
		if !strings.Contains(sql, required) {
			t.Fatalf("automation timestamp repair is missing %q", required)
		}
	}
}

func TestFormResponsePolicyMaintenanceUsesDurableLeasesAndExactLookup(t *testing.T) {
	root := repositoryRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "migrations", "000067_form_response_policy_maintenance.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(strings.Join(strings.Fields(string(raw)), " "))
	for _, required := range []string{
		"create table form_response_policy_maintenance_jobs",
		"create table form_response_policy_compensations",
		"create table form_response_policy_execution_failures",
		"lease_until timestamptz",
		"locked_by text",
		"for update skip locked",
		"on form_response_policy_definitions(tenant_id,legal_entity_id,form_template_id,form_template_version,status,effective_from,effective_until)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("form-response policy maintenance migration is missing %q", required)
		}
	}
	downRaw, err := os.ReadFile(filepath.Join(root, "migrations", "000067_form_response_policy_maintenance.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	down := strings.ToLower(strings.Join(strings.Fields(string(downRaw)), " "))
	if !strings.Contains(down, "if exists (select 1 from form_response_policy_maintenance_jobs)") || !strings.Contains(down, "raise exception") {
		t.Fatal("maintenance downgrade must refuse to discard recovery history")
	}
}

func liveTablesFromMigrations(t *testing.T, migrationDir string) map[string]struct{} {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(migrationDir, "*.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		t.Fatal("no up migrations found")
	}

	live := make(map[string]struct{})
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		sql := stripSQLLineComments(string(raw))
		for _, match := range tableDDL.FindAllStringSubmatch(sql, -1) {
			action := strings.ToUpper(match[1])
			name := strings.ToLower(match[2])
			switch action {
			case "CREATE":
				live[name] = struct{}{}
			case "DROP":
				delete(live, name)
			default:
				t.Fatalf("unsupported table DDL action %q in %s", action, filepath.Base(path))
			}
		}
	}
	return live
}

func ownershipRegisters(t *testing.T, corePath, fragmentDir string) map[string]struct{} {
	t.Helper()
	paths := []string{corePath}
	fragments, err := filepath.Glob(filepath.Join(fragmentDir, "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(fragments)
	paths = append(paths, fragments...)

	registered := make(map[string]struct{})
	for _, path := range paths {
		loadOwnershipRegister(t, path, registered)
	}
	if len(registered) == 0 {
		t.Fatal("durable schema ownership register contains no table rows")
	}
	return registered
}

func loadOwnershipRegister(t *testing.T, path string, registered map[string]struct{}) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	const begin = "<!-- schema-ownership:begin -->"
	const end = "<!-- schema-ownership:end -->"
	start := strings.Index(text, begin)
	finish := strings.Index(text, end)
	if start < 0 || finish < 0 || finish <= start {
		t.Fatalf("ownership markers are missing or out of order in %s", path)
	}

	rows := 0
	body := text[start+len(begin) : finish]
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 10 { // leading/trailing separators plus eight cells
			t.Fatalf("ownership row must contain eight cells in %s: %q", path, line)
		}
		cells := make([]string, 0, 8)
		for _, part := range parts[1 : len(parts)-1] {
			cells = append(cells, strings.TrimSpace(part))
		}
		name := strings.Trim(cells[0], "`")
		if name == "" {
			t.Fatalf("ownership row has an empty table name in %s: %q", path, line)
		}
		if _, exists := registered[name]; exists {
			t.Fatalf("duplicate ownership entry for table %s while loading %s", name, path)
		}
		if _, ok := allowedOwnershipClassifications[cells[1]]; !ok {
			t.Fatalf("table %s has unsupported classification %q in %s", name, cells[1], path)
		}
		for index, cell := range cells[2:] {
			if cell == "" || cell == "—" || strings.EqualFold(cell, "tbd") {
				t.Fatalf("table %s has an incomplete ownership field at column %d in %s", name, index+3, path)
			}
		}
		registered[name] = struct{}{}
		rows++
	}
	if rows == 0 {
		t.Fatalf("ownership register contains no table rows: %s", path)
	}
}

func stripSQLLineComments(sql string) string {
	lines := strings.Split(sql, "\n")
	for index, line := range lines {
		if comment := strings.Index(line, "--"); comment >= 0 {
			lines[index] = line[:comment]
		}
	}
	return strings.Join(lines, "\n")
}

func difference(left, right map[string]struct{}) []string {
	result := make([]string, 0)
	for value := range left {
		if _, ok := right[value]; !ok {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve schema ownership test path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatal(fmt.Errorf("resolve repository root from %s: %w", file, err))
	}
	return root
}
