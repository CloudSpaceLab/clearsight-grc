package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvironmentFileSetsMissingValuesWithoutOverridingProcessEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("CLEARSIGHT_TEST_FILE_VALUE=from-file\nCLEARSIGHT_TEST_PROCESS_VALUE=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLEARSIGHT_TEST_FILE_VALUE", "")
	t.Setenv("CLEARSIGHT_TEST_PROCESS_VALUE", "from-process")
	if err := os.Unsetenv("CLEARSIGHT_TEST_FILE_VALUE"); err != nil {
		t.Fatal(err)
	}
	if err := LoadEnvironmentFile(path); err != nil {
		t.Fatal(err)
	}
	if value := os.Getenv("CLEARSIGHT_TEST_FILE_VALUE"); value != "from-file" {
		t.Fatalf("file value = %q, want from-file", value)
	}
	if value := os.Getenv("CLEARSIGHT_TEST_PROCESS_VALUE"); value != "from-process" {
		t.Fatalf("process value = %q, want from-process", value)
	}
}
