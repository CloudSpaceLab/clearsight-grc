package httpapi

import (
	"os"
	"strings"
	"testing"
)

func TestDocumentImportOpenAPITracksDurableExtractionContract(t *testing.T) {
	content, err := os.ReadFile("../../api/document-imports.openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	contract := string(content)

	required := []string{
		"$ref: '#/components/schemas/DocumentImportSummary'",
		"enum: [PENDING, EXTRACTED, PARTIAL, TRUNCATED, UNSUPPORTED, FAILED]",
		"enum: [PENDING, REVIEW_REQUIRED, NO_PROPOSALS, UNAVAILABLE]",
		"parser_version:",
		"adapter_version:",
		"elements:",
		"degradations:",
		"content_truncated:",
		"DocumentTabularMetadata:",
		"PDF text extraction is available only when the bounded Poppler adapter is installed",
	}
	for _, expected := range required {
		if !strings.Contains(contract, expected) {
			t.Fatalf("document-import OpenAPI is missing %q", expected)
		}
	}

	staleClaims := []string{
		"PDF originals are retained,\n        but this build does not claim PDF text extraction or OCR.",
		"enum: [EXTRACTED, UNSUPPORTED, FAILED]",
	}
	for _, stale := range staleClaims {
		if strings.Contains(contract, stale) {
			t.Fatalf("document-import OpenAPI retained obsolete contract %q", stale)
		}
	}
}
