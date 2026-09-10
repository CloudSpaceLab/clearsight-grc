//go:build postgres

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

// Validate an explicitly supplied private source package through the form service;
// no database, sessions, delivery, or hosted state is involved.
func TestSourceRecordCaptureContractAndITVendorCoverage(t *testing.T) {
	directory := os.Getenv("CLEARSIGHT_SOURCE_MANIFEST_DIR")
	if directory == "" {
		t.Skip("private source manifests are not configured")
	}
	previous := sourceRecordFiles
	sourceRecordFiles = os.DirFS(directory)
	t.Cleanup(func() { sourceRecordFiles = previous })
	service := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	seed := bankverticals.SeedConfig{TenantID: "tenant", LegalEntityID: "entity", ActorID: "maker", ReviewerPrincipalID: "checker", OwnerPrincipalID: "owner"}
	itCounts := map[string]int{}
	seen := map[string]bool{}
	for _, filename := range []string{"source_records_it_vendor.json", "source_records_ops.json"} {
		data, err := fs.ReadFile(sourceRecordFiles, filename)
		if err != nil {
			t.Fatal(err)
		}
		var manifest sourceRecordManifest
		if err = json.Unmarshal(data, &manifest); err != nil {
			t.Fatal(err)
		}
		if manifest.Version != 1 {
			t.Fatalf("unsupported manifest %s", filename)
		}
		for _, group := range manifest.Groups {
			parts := sourceCaptureParts(group)
			consumed := 0
			for index, part := range parts {
				form, answers, err := ensureSourceForm(context.Background(), service, seed, group.ProgramCode, group, part, index, len(parts))
				if err != nil {
					t.Fatalf("%s capture part %d: %v", group.Key, index, err)
				}
				if len(form.Fields) > formcontract.MaxFields || len(form.Sections) > formcontract.MaxSections {
					t.Fatalf("%s exceeds capture limits", group.Key)
				}
				for r, record := range part {
					if consumed >= len(group.Records) || record.Key != group.Records[consumed].Key || seen[record.Key] {
						t.Fatalf("record lost, reordered or repeated: %s", record.Key)
					}
					seen[record.Key] = true
					consumed++
					if len(group.Records) > 20 {
						answer := answers[fmt.Sprintf("row_%d", r)].Text
						if answer == nil || *answer != sourceRecordText(record) {
							t.Fatalf("compact row lost source values: %s", record.Key)
						}
					} else {
						for f, field := range record.Fields {
							answer := answers[fmt.Sprintf("r%d_f%d", r, f)].Text
							value := ""
							if answer != nil {
								value = *answer
							}
							if value != field.Value {
								t.Fatalf("source answer changed: %s / %s", record.Key, field.Label)
							}
						}
					}
				}
			}
			if consumed != len(group.Records) {
				t.Fatalf("%s lost records in capture partition", group.Key)
			}
			if filename == "source_records_it_vendor.json" {
				key := group.Key
				if strings.HasPrefix(key, "it-workplan-") {
					key = "it-workplan"
				}
				itCounts[key] += len(group.Records)
				if key == "third-party-risk-register" {
					for _, record := range group.Records {
						if record.Owner != "Hakeem" || record.DueDate != "2026-03-31" || record.Status != "Open" || !record.CreateMatter {
							t.Fatalf("third-party source owner/deadline/state changed: %+v", record)
						}
						if record.Assessor != "Blessing" && record.Assessor != "Joel" {
							t.Fatalf("missing source assessor: %+v", record)
						}
					}
				}
			}
		}
	}
	for kind, count := range map[string]int{"it-risk-register": 3, "it-risk-exceptions": 2, "it-workplan": 18, "third-party-risk-register": 5} {
		if itCounts[kind] != count {
			t.Fatalf("%s records=%d, want %d", kind, itCounts[kind], count)
		}
	}
}

func TestSyntheticSourceCaptureFitsLimitsWithoutPrivateFiles(t *testing.T) {
	group := sourceRecordGroup{Key: "synthetic-contract", ProgramCode: "TEST", Title: strings.Repeat("é", 150), SourceFile: "synthetic.xlsx", SourceSheet: "Synthetic", SourceSHA256: strings.Repeat("a", 64)}
	for r := 0; r < 3; r++ {
		record := sourceRecord{Key: fmt.Sprintf("synthetic-%d", r), Title: strings.Repeat("é", 150)}
		for f := 0; f < 60; f++ {
			record.Fields = append(record.Fields, sourceRecordField{Label: strings.Repeat("é", 150), Value: fmt.Sprintf("Synthetic value %d/%d", r, f)})
		}
		group.Records = append(group.Records, record)
	}
	parts := sourceCaptureParts(group)
	if len(parts) != 2 || len(parts[0]) != 2 || len(parts[1]) != 1 {
		t.Fatal("source rows must split before exceeding the form field budget")
	}
	service := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	seed := bankverticals.SeedConfig{TenantID: "tenant", LegalEntityID: "entity", ActorID: "maker", ReviewerPrincipalID: "checker", OwnerPrincipalID: "owner"}
	for index, part := range parts {
		form, answers, err := ensureSourceForm(context.Background(), service, seed, group.ProgramCode, group, part, index, len(parts))
		if err != nil {
			t.Fatal(err)
		}
		if len(form.Fields) > formcontract.MaxFields || len(form.Sections) > formcontract.MaxSections || len(answers) != 1+60*len(part) {
			t.Fatal("synthetic capture exceeds limits or loses answers")
		}
	}
}
