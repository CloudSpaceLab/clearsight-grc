package httpapi

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProductionHTTPHandlersDoNotEmbedReservedDemoRecords(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve httpapi package directory")
	}
	dir := filepath.Dir(current)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	reserved := []string{
		`"bank-demo"`,
		`"user-demo"`,
		`"Amaka Okafor"`,
		`"matter-operating-evidence"`,
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		for _, literal := range reserved {
			if strings.Contains(string(content), literal) {
				t.Errorf("%s embeds reserved demo record literal %s; move demo truth to an explicit seed/fixture", entry.Name(), literal)
			}
		}
	}
}
