package evidence

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	docxMediaType                 = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	xlsxMediaType                 = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	maxArchiveEntries             = 2000
	maxArchiveExpandedBytes int64 = 200 << 20
)

var mediaTypeExtensions = map[string]map[string]struct{}{
	"application/pdf": {".pdf": {}},
	"image/png":       {".png": {}},
	"image/jpeg":      {".jpg": {}, ".jpeg": {}},
	"text/plain":      {".txt": {}},
	"text/csv":        {".csv": {}},
	docxMediaType:     {".docx": {}},
	xlsxMediaType:     {".xlsx": {}},
}

func inspectArtifact(fileName, claimedMediaType string, reader io.Reader, maximum int64) ([]byte, string, error) {
	if err := validateArtifactFileName(fileName); err != nil {
		return nil, "", err
	}
	limited := &io.LimitedReader{R: reader, N: maximum + 1}
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > maximum {
		return nil, "", ErrArtifactTooLarge
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("%w: the file is empty", ErrContentInvalid)
	}
	mediaType, err := detectArtifactMediaType(fileName, data)
	if err != nil {
		return nil, "", err
	}
	claimed := normalizeMediaType(claimedMediaType)
	if claimed != "" && claimed != "application/octet-stream" && !equivalentMediaType(claimed, mediaType) {
		return nil, "", fmt.Errorf("%w: the browser file type does not match the file contents", ErrContentInvalid)
	}
	return data, mediaType, nil
}

func validateArtifactFileName(fileName string) error {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" || len(fileName) > 255 || filepath.Base(fileName) != fileName || strings.ContainsAny(fileName, "/\\") {
		return ErrFileName
	}
	for _, character := range fileName {
		if character < 32 || character == 127 {
			return ErrFileName
		}
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	for _, blocked := range []string{".exe", ".dll", ".js", ".jar", ".msi", ".bat", ".cmd", ".ps1", ".docm", ".xlsm", ".zip"} {
		if ext == blocked {
			return ErrFileName
		}
	}
	return nil
}

func detectArtifactMediaType(fileName string, data []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	var mediaType string
	switch {
	case bytes.HasPrefix(data, []byte("%PDF-")):
		if !bytes.Contains(data, []byte("%%EOF")) || !bytes.Contains(data, []byte(" obj")) {
			return "", fmt.Errorf("%w: the PDF structure is incomplete", ErrContentInvalid)
		}
		lower := bytes.ToLower(data)
		for _, marker := range [][]byte{[]byte("/javascript"), []byte("/launch"), []byte("/embeddedfile"), []byte("/richmedia")} {
			if bytes.Contains(lower, marker) {
				return "", fmt.Errorf("%w: the PDF contains active or embedded content", ErrContentInvalid)
			}
		}
		mediaType = "application/pdf"
	case bytes.HasPrefix(data, []byte("PK\x03\x04")):
		var err error
		mediaType, err = inspectOfficeArchive(data)
		if err != nil {
			return "", err
		}
	default:
		detected := normalizeMediaType(http.DetectContentType(data))
		switch detected {
		case "image/png", "image/jpeg":
			mediaType = detected
		default:
			if bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
				return "", fmt.Errorf("%w: the file contents are not supported", ErrContentInvalid)
			}
			if ext == ".csv" {
				mediaType = "text/csv"
			} else {
				mediaType = "text/plain"
			}
		}
	}
	allowedExtensions, ok := mediaTypeExtensions[mediaType]
	if !ok {
		return "", ErrMediaType
	}
	if _, ok := allowedExtensions[ext]; !ok {
		return "", fmt.Errorf("%w: the filename extension does not match the file contents", ErrContentInvalid)
	}
	return mediaType, nil
}

func inspectOfficeArchive(data []byte) (string, error) {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(archive.File) == 0 || len(archive.File) > maxArchiveEntries {
		return "", fmt.Errorf("%w: the Office document structure is invalid", ErrContentInvalid)
	}
	entries := make(map[string]struct{}, len(archive.File))
	var expanded uint64
	for _, entry := range archive.File {
		name := strings.ToLower(strings.ReplaceAll(entry.Name, "\\", "/"))
		if strings.HasPrefix(name, "/") || strings.Contains(name, "../") {
			return "", fmt.Errorf("%w: the Office document contains an unsafe path", ErrContentInvalid)
		}
		expanded += entry.UncompressedSize64
		if expanded > uint64(maxArchiveExpandedBytes) {
			return "", fmt.Errorf("%w: the Office document expands beyond the inspection limit", ErrContentInvalid)
		}
		if strings.Contains(name, "vbaproject.bin") || strings.Contains(name, "/embeddings/") || strings.HasSuffix(name, ".exe") {
			return "", fmt.Errorf("%w: the Office document contains macros or embedded content", ErrContentInvalid)
		}
		entries[name] = struct{}{}
	}
	if _, ok := entries["[content_types].xml"]; !ok {
		return "", fmt.Errorf("%w: the Office document is missing its content type manifest", ErrContentInvalid)
	}
	if _, ok := entries["_rels/.rels"]; !ok {
		return "", fmt.Errorf("%w: the Office document is missing its relationship manifest", ErrContentInvalid)
	}
	if _, ok := entries["word/document.xml"]; ok {
		return docxMediaType, nil
	}
	if _, ok := entries["xl/workbook.xml"]; ok {
		return xlsxMediaType, nil
	}
	return "", fmt.Errorf("%w: only DOCX and XLSX Office documents are supported", ErrContentInvalid)
}

func equivalentMediaType(claimed, detected string) bool {
	if claimed == detected {
		return true
	}
	return (claimed == "text/plain" || claimed == "text/csv") && (detected == "text/plain" || detected == "text/csv")
}

func mediaTypeForExtension(value string) string {
	if parsed := normalizeMediaType(value); strings.Contains(parsed, "/") {
		return parsed
	}
	if extension := strings.ToLower(strings.TrimSpace(value)); strings.HasPrefix(extension, ".") {
		return normalizeMediaType(mime.TypeByExtension(extension))
	}
	return ""
}
