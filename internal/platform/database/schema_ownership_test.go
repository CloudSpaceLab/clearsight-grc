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

func TestLiveTablesHaveExactlyOneOwnershipEntry(t *testing.T) {
	root := repositoryRoot(t)
	live := liveTablesFromMigrations(t, filepath.Join(root, "migrations"))
	registered := ownershipRegister(t, filepath.Join(root, "docs", "architecture", "durable-schema-ownership.md"))

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

func ownershipRegister(t *testing.T, path string) map[string]struct{} {
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

	allowed := map[string]struct{}{
		"active authoritative state":                   {},
		"active projection":                            {},
		"active infrastructure ledger":                 {},
		"explicitly reserved for a named future phase": {},
		"deprecated/migration-only":                    {},
		"removable":                                    {},
	}
	registered := make(map[string]struct{})
	body := text[start+len(begin) : finish]
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 10 { // leading/trailing separators plus eight cells
			t.Fatalf("ownership row must contain eight cells: %q", line)
		}
		cells := make([]string, 0, 8)
		for _, part := range parts[1 : len(parts)-1] {
			cells = append(cells, strings.TrimSpace(part))
		}
		name := strings.Trim(cells[0], "`")
		if name == "" {
			t.Fatalf("ownership row has an empty table name: %q", line)
		}
		if _, exists := registered[name]; exists {
			t.Fatalf("duplicate ownership entry for table %s", name)
		}
		if _, ok := allowed[cells[1]]; !ok {
			t.Fatalf("table %s has unsupported classification %q", name, cells[1])
		}
		for index, cell := range cells[2:] {
			if cell == "" || cell == "—" || strings.EqualFold(cell, "tbd") {
				t.Fatalf("table %s has an incomplete ownership field at column %d", name, index+3)
			}
		}
		registered[name] = struct{}{}
	}
	if len(registered) == 0 {
		t.Fatalf("ownership register contains no table rows: %s", path)
	}
	return registered
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
