package demodocuments

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestManifestContainsOnlyMatchingImmutableSampleBytes(t *testing.T) {
	files := Files()
	if len(files) != 6 {
		t.Fatalf("want six reviewed sample files, got %d", len(files))
	}
	seen := map[string]bool{}
	expectedMedia := map[string]string{
		"sample-insurance-schedule.pdf":            "application/pdf",
		"sample-office-statement.png":              "image/png",
		"sample-recovery-plan.pdf":                 "application/pdf",
		"sample-security-declaration-previous.pdf": "application/pdf",
		"sample-security-declaration.pdf":          "application/pdf",
		"sample-subprocessor-register.xlsx":        "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}
	for _, file := range files {
		if seen[file.Name] {
			t.Fatalf("duplicate %s", file.Name)
		}
		seen[file.Name] = true
		content, ok := Read(file.Name)
		if !ok || len(content) == 0 {
			t.Fatalf("missing %s", file.Name)
		}
		if file.MediaType != expectedMedia[file.Name] || file.MediaType == "" {
			t.Fatalf("incorrect sample media type: %+v", file)
		}
		switch file.MediaType {
		case "application/pdf":
			if !bytes.HasPrefix(content, []byte("%PDF-")) || !bytes.Contains(content, []byte("%%EOF")) {
				t.Fatal("invalid PDF")
			}
			for _, marker := range []string{"/javascript", "/launch", "/embeddedfile", "/richmedia"} {
				if strings.Contains(strings.ToLower(string(content)), marker) {
					t.Fatal("active sample PDF content")
				}
			}
		case "image/png":
			if !bytes.HasPrefix(content, []byte("\x89PNG\r\n\x1a\n")) {
				t.Fatal("invalid PNG")
			}
		default:
			archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
			if err != nil {
				t.Fatal(err)
			}
			reader, err := archive.Open("xl/workbook.xml")
			if err != nil {
				t.Fatal("missing XLSX workbook")
			}
			_ = reader.Close()
		}
		digest := sha256.Sum256(content)
		if file.SizeBytes != int64(len(content)) || file.SHA256 != hex.EncodeToString(digest[:]) {
			t.Fatalf("manifest differs from shipped bytes: %s", file.Name)
		}
		if !Matches(file.Name, file.MediaType, file.SHA256, file.SizeBytes) {
			t.Fatalf("sample not recognized: %s", file.Name)
		}
		for _, changed := range []struct {
			name, media, digest string
			size                int64
		}{
			{"uploaded-file.pdf", file.MediaType, file.SHA256, file.SizeBytes},
			{file.Name, "application/octet-stream", file.SHA256, file.SizeBytes},
			{file.Name, file.MediaType, "not-the-sample-digest", file.SizeBytes},
			{file.Name, file.MediaType, file.SHA256, file.SizeBytes + 1},
		} {
			if Matches(changed.name, changed.media, changed.digest, changed.size) {
				t.Fatalf("accepted changed sample metadata: %+v", changed)
			}
		}
		content[0] ^= 0xff
		again, _ := Read(file.Name)
		if content[0] == again[0] {
			t.Fatal("caller can change embedded sample bytes")
		}
	}
	entries, err := assets.ReadDir("assets")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(files) {
		t.Fatal("unreviewed or missing embedded assets")
	}
	for _, entry := range entries {
		if !seen[entry.Name()] {
			t.Fatalf("unlisted asset %s", entry.Name())
		}
	}
	if _, ok := Read("../manifest.go"); ok {
		t.Fatal("accepted arbitrary path")
	}
	files[0].Name = "changed"
	if Files()[0].Name == "changed" {
		t.Fatal("caller can change the manifest")
	}
}
