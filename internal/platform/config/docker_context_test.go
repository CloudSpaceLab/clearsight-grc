package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackendDockerBuildContextIncludesTestContracts(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	dockerignore := readRepositoryFile(t, filepath.Join(root, ".dockerignore"))
	if !strings.Contains(dockerignore, "!.env.example") {
		t.Fatal(".dockerignore must allow .env.example for backend build-time contract tests")
	}

	for _, name := range []string{"Dockerfile.api", "Dockerfile.worker"} {
		contents := readRepositoryFile(t, filepath.Join(root, name))
		for _, required := range []string{
			"COPY .env.example ./.env.example",
			"COPY docs/architecture/durable-schema-ownership.d ./docs/architecture/durable-schema-ownership.d",
		} {
			if !strings.Contains(contents, required) {
				t.Errorf("%s must contain %q so its build-time test suite sees the same contracts as CI", name, required)
			}
		}
	}
}

func readRepositoryFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
