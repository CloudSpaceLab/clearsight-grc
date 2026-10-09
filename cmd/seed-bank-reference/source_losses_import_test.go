//go:build postgres

package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"testing/fstest"
)

func lossCandidateTestFS(t *testing.T, records []sourceRecord) fstest.MapFS {
	t.Helper()
	group := syntheticOpsLossGroup()
	group.Records = records
	manifest := sourceRecordManifest{Version: 1, Groups: []sourceRecordGroup{group}}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return fstest.MapFS{"source_records_ops_loss.json": &fstest.MapFile{Data: data}}
}

func TestSourceLossCandidatesRequireCompleteEightEventCohort(t *testing.T) {
	records := make([]sourceRecord, 0, 96)
	missing := make([]sourceRecord, 0, 84)
	for month := 1; month <= 12; month++ {
		for index := 0; index < 8; index++ {
			record := syntheticOpsLossRecord(index, fmt.Sprintf("%02d", month))
			records = append(records, record)
			if index < 7 {
				missing = append(missing, record)
			}
		}
	}
	if values, err := sourceLossCandidateValues(lossCandidateTestFS(t, records)); err != nil || len(values) != 8 {
		t.Fatalf("96 monthly source views must yield eight canonical events: count=%d err=%v", len(values), err)
	}
	if _, err := sourceLossCandidateValues(lossCandidateTestFS(t, missing)); err == nil {
		t.Fatal("truncated seven-event manifest must fail before the first ledger write")
	}
	if _, err := sourceLossCandidateValues(fstest.MapFS{}); err == nil {
		t.Fatal("missing private source manifest must not be accepted")
	}
}

func TestSourceLossCandidatesRejectConflictsBeforeWrite(t *testing.T) {
	records := make([]sourceRecord, 0, 8)
	for index := 0; index < 8; index++ {
		records = append(records, syntheticOpsLossRecord(index, "Jan"))
	}
	conflicting := syntheticOpsLossRecord(0, "Feb")
	for index := range conflicting.Fields {
		if conflicting.Fields[index].Label == "Amount" {
			conflicting.Fields[index].Value = "999.99"
		}
	}
	records = append(records, conflicting)
	if _, err := sourceLossCandidateValues(lossCandidateTestFS(t, records)); err == nil {
		t.Fatal("conflicting monthly value for the same loss identity must fail in preflight")
	}
	records = records[:8]
	if values, err := sourceLossCandidateValues(lossCandidateTestFS(t, records)); err != nil || len(values) != 8 {
		t.Fatalf("eight deduplicated source events must remain supported: count=%d err=%v", len(values), err)
	}
}
