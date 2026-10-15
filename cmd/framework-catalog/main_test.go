package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOfflineFrameworkPublicationVerification(t *testing.T) {
	path := filepath.Join("..", "..", "internal", "frameworkcatalog", "testdata")
	var out, errOut bytes.Buffer
	code := run([]string{"-manifest", filepath.Join(path, "synthetic_manifest.json"), "-source", filepath.Join(path, "synthetic_source.txt")}, &out, &errOut)
	if code != 0 || errOut.Len() > 0 || !strings.Contains(out.String(), `"published":false`) ||
		!strings.Contains(out.String(), `"edition_code":"2026"`) {
		t.Fatalf("verification failed: code=%d stdout=%s stderr=%s", code, out.String(), errOut.String())
	}
}

func TestOfflineFrameworkPublicationRejectsMissingOrAlteredSource(t *testing.T) {
	path := filepath.Join("..", "..", "internal", "frameworkcatalog", "testdata")
	var out, errOut bytes.Buffer
	if code := run([]string{"-manifest", filepath.Join(path, "synthetic_manifest.json")}, &out, &errOut); code != 2 {
		t.Fatalf("missing source accepted, code %d", code)
	}
	out.Reset()
	errOut.Reset()
	file := filepath.Join(t.TempDir(), "modified_source.txt")
	if err := os.WriteFile(file, []byte("not the approved source"), 0600); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"-manifest", filepath.Join(path, "synthetic_manifest.json"), "-source", file}, &out, &errOut); code != 1 {
		t.Fatalf("wrong source accepted: code=%d", code)
	}
	if out.Len() != 0 {
		t.Fatalf("failed verification emitted a successful receipt: %s", out.String())
	}
}
