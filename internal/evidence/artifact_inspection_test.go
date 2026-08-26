package evidence

import (
	"archive/zip"
	"bytes"
	"errors"
	"testing"
)

func TestInspectArtifactRecognizesBoundedPDFAndDOCXContent(t *testing.T) {
	pdf := []byte("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n%%EOF")
	data, mediaType, err := inspectArtifact("control-review.pdf", "application/octet-stream", bytes.NewReader(pdf), 1<<20)
	if err != nil || mediaType != "application/pdf" || !bytes.Equal(data, pdf) {
		t.Fatalf("inspect PDF: media=%q err=%v", mediaType, err)
	}

	docx := testOfficeDocument(t, map[string]string{
		"[Content_Types].xml": "<Types/>",
		"_rels/.rels":         "<Relationships/>",
		"word/document.xml":   "<w:document/>",
	})
	_, mediaType, err = inspectArtifact("assurance.docx", docxMediaType, bytes.NewReader(docx), 1<<20)
	if err != nil || mediaType != docxMediaType {
		t.Fatalf("inspect DOCX: media=%q err=%v", mediaType, err)
	}
}

func TestInspectArtifactRejectsMisleadingOrActiveFiles(t *testing.T) {
	validPDF := []byte("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n%%EOF")
	tests := []struct {
		name      string
		fileName  string
		mediaType string
		data      []byte
		target    error
	}{
		{name: "path filename", fileName: "../control.pdf", mediaType: "application/pdf", data: validPDF, target: ErrFileName},
		{name: "extension mismatch", fileName: "control.docx", mediaType: "application/pdf", data: validPDF, target: ErrContentInvalid},
		{name: "active PDF", fileName: "control.pdf", mediaType: "application/pdf", data: []byte("%PDF-1.7\n1 0 obj\n<< /JavaScript 2 0 R >>\nendobj\n%%EOF"), target: ErrContentInvalid},
		{name: "macro document", fileName: "control.docx", mediaType: docxMediaType, data: testOfficeDocument(t, map[string]string{"[Content_Types].xml": "<Types/>", "word/document.xml": "<w:document/>", "word/vbaProject.bin": "macro"}), target: ErrContentInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := inspectArtifact(test.fileName, test.mediaType, bytes.NewReader(test.data), 1<<20); !errors.Is(err, test.target) {
				t.Fatalf("expected %v, got %v", test.target, err)
			}
		})
	}
}

func testOfficeDocument(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for name, content := range entries {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
