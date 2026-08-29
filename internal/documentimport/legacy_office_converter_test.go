package documentimport

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLegacyOfficeConverterDisabledAndInputLimit(t *testing.T) {
	if _, err := (LegacyOfficeConverter{}).ConvertXLS(t.Context(), "legacy.xls", []byte("xls")); !errors.Is(err, ErrLegacyOfficeConversion) {
		t.Fatalf("disabled converter error = %v", err)
	}
	converter := LegacyOfficeConverter{Executable: "/does/not/matter", MaxInputBytes: 2, MaxOutputBytes: 10, Timeout: time.Second}
	if _, err := converter.ConvertXLS(t.Context(), "legacy.xls", []byte("too large")); !errors.Is(err, ErrLegacyOfficeConversion) {
		t.Fatalf("input limit error = %v", err)
	}
}

func TestLegacyOfficeConverterTimeout(t *testing.T) {
	executable := writeConverterScript(t, `sleep 1`)
	converter := LegacyOfficeConverter{Executable: executable, MaxInputBytes: 1024, MaxOutputBytes: 1024, Timeout: 10 * time.Millisecond}
	_, err := converter.ConvertXLS(t.Context(), "legacy.xls", []byte("xls"))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout error = %v", err)
	}
}

func TestLegacyOfficeConverterOutputLimitAndSingleArtifact(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "too-large", body: `printf '0123456789abcdef' > "$out/source.xlsx"`},
		{name: "multiple", body: `printf 'one' > "$out/source.xlsx"; printf 'two' > "$out/second.xlsx"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			executable := writeConverterScript(t, tc.body)
			converter := LegacyOfficeConverter{Executable: executable, MaxInputBytes: 1024, MaxOutputBytes: 8, Timeout: time.Second}
			_, err := converter.ConvertXLS(t.Context(), "legacy.xls", []byte("xls"))
			if !errors.Is(err, ErrLegacyOfficeConversion) {
				t.Fatalf("expected bounded conversion failure, got %v", err)
			}
		})
	}
}

func TestLegacyOfficeConverterReturnsOnlyBoundedXLSX(t *testing.T) {
	executable := writeConverterScript(t, `printf 'xlsx-bytes' > "$out/source.xlsx"`)
	converter := LegacyOfficeConverter{Executable: executable, MaxInputBytes: 1024, MaxOutputBytes: 1024, Timeout: time.Second}
	converted, err := converter.ConvertXLS(t.Context(), "legacy.xls", []byte("xls"))
	if err != nil {
		t.Fatal(err)
	}
	if string(converted) != "xlsx-bytes" {
		t.Fatalf("converted bytes = %q", converted)
	}
}

func writeConverterScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-libreoffice")
	script := `#!/bin/sh
set -eu
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--outdir" ]; then
    shift
    out="$1"
    break
  fi
  shift
done
if [ -z "$out" ]; then
  exit 2
fi
` + strings.TrimSpace(body) + "\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}
