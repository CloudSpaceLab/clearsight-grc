package documentimport

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestTabularMetadataCoversCSVJSONAndNDJSON(t *testing.T) {
	ctx := context.Background()
	csvMeta, err := InspectTabularArtifact(ctx, "accounts.csv", "text/csv", []byte("id,name\n1,Ada\n2,Grace\n"), DefaultExtractionPolicy())
	if err != nil || csvMeta.ParserVersion != TabularParserVersion || len(csvMeta.Resources) != 1 || csvMeta.Resources[0].RowsTotal != 2 || len(csvMeta.Resources[0].Fields) != 2 || csvMeta.Resources[0].SchemaFingerprint == "" {
		t.Fatalf("unexpected CSV metadata: %#v err=%v", csvMeta, err)
	}
	jsonMeta, err := InspectTabularArtifact(ctx, "models.json", "application/json", []byte(`[{"id":1,"approved":true},{"id":2,"approved":false,"owner":"risk"}]`), DefaultExtractionPolicy())
	if err != nil || jsonMeta.RowsTotal != 2 || len(jsonMeta.Resources) != 1 || len(jsonMeta.Resources[0].Fields) != 3 {
		t.Fatalf("unexpected JSON metadata: %#v err=%v", jsonMeta, err)
	}
	ndjson := []byte("{\"id\":1,\"name\":\"Ada\"}\nnot-json\n{\"id\":2,\"name\":\"Grace\"}\n")
	ndMeta, err := InspectTabularArtifact(ctx, "users.ndjson", "application/x-ndjson", ndjson, DefaultExtractionPolicy())
	if err != nil || ndMeta.RowsTotal != 3 || ndMeta.RowsRejected != 1 || len(ndMeta.RowErrors) != 1 || ndMeta.Resources[0].RowsRejected != 1 {
		t.Fatalf("unexpected NDJSON metadata: %#v err=%v", ndMeta, err)
	}
}

func TestTabularXLSXUsesCachedValuesWithoutFormulaEvaluation(t *testing.T) {
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	write := func(name, body string) {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	write("xl/worksheets/sheet1.xml", `<?xml version="1.0"?><worksheet><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>id</t></is></c><c r="B1" t="inlineStr"><is><t>score</t></is></c></row><row r="2"><c r="A2"><v>1</v></c><c r="B2"><f>1+1</f><v>2</v></c></row></sheetData></worksheet>`)
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	meta, err := InspectTabularArtifact(context.Background(), "scores.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer.Bytes(), DefaultExtractionPolicy())
	if err != nil || len(meta.Resources) != 1 || meta.Resources[0].Name != "Sheet 1" || meta.Resources[0].RowsTotal != 1 || meta.Resources[0].SchemaFingerprint == "" {
		t.Fatalf("unexpected XLSX metadata: %#v err=%v", meta, err)
	}
	var values []tabularRow
	_, err = scanTabularArtifact(context.Background(), TabularXLSX, buffer.Bytes(), DefaultExtractionPolicy(), "Sheet 1", func(row tabularRow) error { values = append(values, row); return nil })
	if err != nil || len(values) != 1 || values[0].Values["score"].text != "2" {
		t.Fatalf("XLSX cached value was not preserved: %#v err=%v", values, err)
	}
}

func TestJSONAndNDJSONProduceReviewSections(t *testing.T) {
	jsonResult := Extract("inventory.json", "application/json", []byte(`[{"id":"a","state":"approved"}]`))
	if jsonResult.Status != ExtractionExtracted || len(jsonResult.Sections) != 1 {
		t.Fatalf("JSON review extraction failed: %#v", jsonResult)
	}
	ndResult := Extract("inventory.ndjson", "application/x-ndjson", []byte("{\"id\":\"a\"}\n{bad}\n{\"id\":\"b\"}\n"))
	if ndResult.Status != ExtractionExtracted || len(ndResult.Sections) != 2 {
		t.Fatalf("NDJSON review extraction failed: %#v", ndResult)
	}
}

func TestTabularMetadataBudgetFailsCompactly(t *testing.T) {
	fields := make([]TabularField, 0, 10000)
	for index := 0; index < 10000; index++ {
		fields = append(fields, TabularField{Name: fmt.Sprintf("field_%05d_%s", index, strings.Repeat("x", 240)), NativeType: "tabular:string"})
	}
	metadata, err := boundTabularMetadata(TabularMetadata{Format: TabularXLSX, ParserVersion: TabularParserVersion, Resources: []TabularResource{{Name: "Sheet 1", Fields: fields}}})
	if !errors.Is(err, ErrResourceLimit) || metadata.FatalError == "" || len(metadata.Resources) != 0 {
		t.Fatalf("oversized tabular metadata was not compactly bounded: metadata=%#v err=%v", metadata, err)
	}
	encoded, marshalErr := json.Marshal(metadata)
	if marshalErr != nil || len(encoded) > HardMaxTabularMetadataBytes {
		t.Fatalf("compact failure receipt exceeded the persistence budget: bytes=%d err=%v", len(encoded), marshalErr)
	}
}

func TestTabularXLSXRejectsNonMonotonicRowPositions(t *testing.T) {
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	entry, err := archive.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, err = entry.Write([]byte(`<?xml version="1.0"?><worksheet><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>id</t></is></c></row><row r="2"><c r="A2"><v>1</v></c></row><row r="2"><c r="A2"><v>2</v></c></row></sheetData></worksheet>`))
	if err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	metadata, err := InspectTabularArtifact(context.Background(), "duplicate-rows.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer.Bytes(), DefaultExtractionPolicy())
	if err == nil || metadata.FatalError == "" || !strings.Contains(metadata.FatalError, "strictly increasing") {
		t.Fatalf("non-monotonic XLSX rows remained resumable: metadata=%#v err=%v", metadata, err)
	}
}
