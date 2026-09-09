package documentimport

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Opt-in local acceptance check. The supplied workbook is never committed or
// printed; synthetic fixtures cover the same structure in the default suite.
func TestRiskRegisterLocalSample(t *testing.T) {
	path := os.Getenv("CLEARSIGHT_REGISTER_SAMPLE")
	if path == "" {
		t.Skip("CLEARSIGHT_REGISTER_SAMPLE is not configured")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	x := Extract(filepath.Base(path), "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
	metadata, err := InspectTabularArtifact(context.Background(), filepath.Base(path), "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data, DefaultExtractionPolicy())
	if err != nil {
		t.Fatal(err)
	}
	d := Document{ID: "local-sample", Version: 1, SHA256: fmt.Sprintf("%x", sha256.Sum256(data)), ExtractionStatus: x.Status, Elements: x.Elements, Sections: x.Sections, SectionsTotal: x.SectionsTotal, SectionsOmitted: x.SectionsOmitted, ContentTruncated: x.ContentTruncated, Degradations: x.Degradations, Tabular: &metadata}
	p, err := ParseRiskRegister(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Groups) != 2 || len(p.Rows) != 5 {
		t.Fatalf("expected 2 assessments / 5 findings, received %d / %d", len(p.Groups), len(p.Rows))
	}
	for _, row := range p.Rows {
		if row.Responsibility == "" || row.SuggestedDueDate != "2026-03-31" {
			t.Fatal("responsibility or timeline not resolved")
		}
	}
	t.Logf("Validated %d assessments and %d findings without changing the workbook", len(p.Groups), len(p.Rows))
}
