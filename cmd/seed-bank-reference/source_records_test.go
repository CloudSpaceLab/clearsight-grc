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
			if group.ResponsePerRecord {
				if len(group.Records) < 2 || len(group.Records[0].Fields) == 0 {
					t.Fatalf("register %s must hold at least two records with fields", group.Key)
				}
				registerSchema := group.Records[0].Fields
				for _, record := range group.Records {
					if len(record.Fields) != len(registerSchema) {
						t.Fatalf("register schema drift: %s", record.Key)
					}
					for f, field := range record.Fields {
						if field.Label != registerSchema[f].Label {
							t.Fatalf("register label drift: %s / %s", record.Key, field.Label)
						}
					}
					if seen[record.Key] {
						t.Fatalf("record repeated: %s", record.Key)
					}
					seen[record.Key] = true
				}
				// One form is built once from record 0's schema and stays within limits.
				form, answers, err := ensureSourceForm(context.Background(), service, seed, group.ProgramCode, group, group.Records[:1], 0, 1)
				if err != nil {
					t.Fatalf("register form %s: %v", group.Key, err)
				}
				if len(form.Fields) > formcontract.MaxFields || len(form.Sections) > formcontract.MaxSections {
					t.Fatalf("register %s exceeds capture limits", group.Key)
				}
				// The single form build from record 0 must expose exactly the
				// source context plus every nonblank record-0 value keyed r0_f%d.
				if len(form.Fields) != 1+len(registerSchema) || answers["source_context"].Text == nil {
					t.Fatalf("register %s must keep the shared source + record schema", group.Key)
				}
				own := registerAnswers(answers["source_context"], group.Records[0])
				if len(own) != len(answers) {
					t.Fatalf("register %s record-0 answer count changed: %d != %d", group.Key, len(own), len(answers))
				}
				for bid, answer := range own {
					if other := answers[bid]; (other.Text == nil) != (answer.Text == nil) || (answer.Text != nil && *answer.Text != *other.Text) {
						t.Fatalf("register %s / %s record-0 answer content changed", group.Key, bid)
					}
				}
				// Every record shares the key space: each nonblank value is
				// preserved keyed r0_f%d and blanks stay absent.
				for _, record := range group.Records {
					perRecord := registerAnswers(answers["source_context"], record)
					nonblank := 0
					for _, field := range record.Fields {
						if strings.TrimSpace(field.Value) != "" {
							nonblank++
						}
					}
					if len(perRecord) != 1+nonblank {
						t.Fatalf("register %s / %s answer count: %d != %d", group.Key, record.Key, len(perRecord), 1+nonblank)
					}
					for f, field := range record.Fields {
						bid := fmt.Sprintf("r0_f%d", f)
						if strings.TrimSpace(field.Value) == "" {
							if _, exists := perRecord[bid]; exists {
								t.Fatalf("register %s / %s must skip blank %s", group.Key, record.Key, bid)
							}
							continue
						}
						if perRecord[bid].Text == nil || *perRecord[bid].Text != field.Value {
							t.Fatalf("register %s / %s lost %s value", group.Key, record.Key, bid)
						}
					}
				}
				continue
			}
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
							expected := field.Value
							if strings.TrimSpace(expected) == "" {
								expected = ""
							}
							if value != expected {
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

func TestSyntheticSourceRegisterFitsLimitsAndPreservesAnswers(t *testing.T) {
	group := sourceRecordGroup{Key: "synthetic-register", ProgramCode: "TEST", Title: "Branch KRI register", SourceFile: "synthetic.xlsx", SourceSheet: "Register", SourceSHA256: strings.Repeat("c", 64), ResponsePerRecord: true}
	// 31 records that share one 55-field schema, mirroring the branch KRI sheet.
	for r := 0; r < 31; r++ {
		record := sourceRecord{Key: fmt.Sprintf("synthetic-register-%d", r), Title: "Branch " + fmt.Sprint(r)}
		for f := 0; f < 55; f++ {
			record.Fields = append(record.Fields, sourceRecordField{Label: fmt.Sprintf("Field %02d", f), Value: fmt.Sprintf("Branch %d value %d", r, f)})
		}
		group.Records = append(group.Records, record)
	}
	service := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	seed := bankverticals.SeedConfig{TenantID: "tenant", LegalEntityID: "entity", ActorID: "maker", ReviewerPrincipalID: "checker", OwnerPrincipalID: "owner"}
	// A register above the row threshold must not be compacted when captured
	// through the register path (the installer builds one form from record 0).
	// Non-compact parts may still split at the 15-row fallback, which the
	// register installer branch intentionally bypasses.
	// The single form build from record 0 must expose exactly the source
	// context plus every nonblank record-0 value keyed r0_f%d, and stay
	// within capture limits.
	form, answers, err := ensureSourceForm(context.Background(), service, seed, group.ProgramCode, group, group.Records[:1], 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(form.Fields) > formcontract.MaxFields || len(form.Sections) > formcontract.MaxSections {
		t.Fatal("synthetic register exceeds capture limits")
	}
	if len(form.Fields) != 1+55 || len(form.Sections) != 2 {
		t.Fatalf("synthetic register schema drift: %d fields / %d sections", len(form.Fields), len(form.Sections))
	}
	own := registerAnswers(answers["source_context"], group.Records[0])
	if len(own) != len(answers) {
		t.Fatalf("synthetic register record-0 answer count changed: %d != %d", len(own), len(answers))
	}
	for bid, answer := range own {
		if other := answers[bid]; (other.Text == nil) != (answer.Text == nil) || (answer.Text != nil && *answer.Text != *other.Text) {
			t.Fatalf("synthetic register / %s record-0 answer content changed", bid)
		}
	}
	// Every record shares the key space: each nonblank value is preserved
	// keyed r0_f%d and blanks stay absent.
	for _, record := range group.Records {
		perRecord := registerAnswers(answers["source_context"], record)
		nonblank := 0
		for _, field := range record.Fields {
			if strings.TrimSpace(field.Value) != "" {
				nonblank++
			}
		}
		if len(perRecord) != 1+nonblank {
			t.Fatalf("synthetic register / %s answer count: %d != %d", record.Key, len(perRecord), 1+nonblank)
		}
		for f, field := range record.Fields {
			bid := fmt.Sprintf("r0_f%d", f)
			if strings.TrimSpace(field.Value) == "" {
				if _, exists := perRecord[bid]; exists {
					t.Fatalf("synthetic register / %s must skip blank %s", record.Key, bid)
				}
				continue
			}
			if perRecord[bid].Text == nil || *perRecord[bid].Text != field.Value {
				t.Fatalf("synthetic register / %s lost %s value", record.Key, bid)
			}
		}
	}
}
