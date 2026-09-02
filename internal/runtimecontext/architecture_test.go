package runtimecontext

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRuntimePackagesDoNotContainDemoFallbacks(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller path is unavailable")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	forbidden := []string{
		"today.DemoItems(",
		"func DemoItems(",
		"func demoActorName(",
		`"id": "bank-demo"`,
	}

	for _, relativeRoot := range []string{"cmd/api", "internal/httpapi", "internal/today"} {
		root := filepath.Join(repositoryRoot, filepath.FromSlash(relativeRoot))
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, marker := range forbidden {
				if strings.Contains(string(contents), marker) {
					t.Errorf("%s contains forbidden runtime fallback %q", filepath.ToSlash(path), marker)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", relativeRoot, err)
		}
	}
}
