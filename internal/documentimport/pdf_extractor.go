package documentimport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const (
	pdfExtractionMethod = "POPPLER_TEXT_V1"
	pdfInfoOutputLimit  = int64(64 << 10)
	pdfDiagnosticLimit  = int64(32 << 10)
)

type pdfToolPaths struct {
	Info string
	Text string
}

type pdfCommandResponse struct {
	Stdout []byte
	Stderr []byte
	Err    error
}

type pdfCommandRunner interface {
	Run(ctx context.Context, path string, arguments []string, stdoutLimit, stderrLimit int64) pdfCommandResponse
}

type systemPDFCommandRunner struct{}

func (systemPDFCommandRunner) Run(ctx context.Context, path string, arguments []string, stdoutLimit, stderrLimit int64) pdfCommandResponse {
	command := exec.CommandContext(ctx, path, arguments...)
	command.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	stdout := newBoundedCommandBuffer(stdoutLimit)
	stderr := newBoundedCommandBuffer(stderrLimit)
	command.Stdout = stdout
	command.Stderr = stderr
	err := command.Run()
	if stdout.exceeded || stderr.exceeded {
		err = ErrResourceLimit
	}
	return pdfCommandResponse{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), Err: err}
}

type boundedCommandBuffer struct {
	buffer   bytes.Buffer
	maximum  int64
	exceeded bool
}

func newBoundedCommandBuffer(maximum int64) *boundedCommandBuffer {
	return &boundedCommandBuffer{maximum: maximum}
}

func (b *boundedCommandBuffer) Write(value []byte) (int, error) {
	if b.maximum <= 0 {
		b.exceeded = len(value) > 0
		return 0, ErrResourceLimit
	}
	remaining := b.maximum - int64(b.buffer.Len())
	if remaining <= 0 {
		b.exceeded = len(value) > 0
		return 0, ErrResourceLimit
	}
	if int64(len(value)) > remaining {
		written, _ := b.buffer.Write(value[:remaining])
		b.exceeded = true
		return written, ErrResourceLimit
	}
	return b.buffer.Write(value)
}

func (b *boundedCommandBuffer) Bytes() []byte {
	return append([]byte(nil), b.buffer.Bytes()...)
}

func extractPDF(ctx context.Context, data []byte, collector *sectionCollector, policy ExtractionPolicy) ExtractionResult {
	tools := discoverPDFTools()
	return extractPDFWithTools(ctx, data, collector, policy, tools, systemPDFCommandRunner{})
}

func discoverPDFTools() pdfToolPaths {
	info, infoErr := exec.LookPath("pdfinfo")
	text, textErr := exec.LookPath("pdftotext")
	if infoErr != nil || textErr != nil || !directlyExecutable(info) || !directlyExecutable(text) {
		return pdfToolPaths{}
	}
	return pdfToolPaths{Info: info, Text: text}
}

func directlyExecutable(path string) bool {
	if runtime.GOOS != "windows" {
		return true
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".exe", ".com":
		return true
	default:
		return false
	}
}

func extractPDFWithTools(ctx context.Context, data []byte, collector *sectionCollector, policy ExtractionPolicy, tools pdfToolPaths, runner pdfCommandRunner) ExtractionResult {
	policy = policy.normalized()
	if strings.TrimSpace(tools.Info) == "" || strings.TrimSpace(tools.Text) == "" {
		return collector.result(ExtractionUnsupported, "NONE", "Automated PDF text extraction is unavailable because the Poppler tools are not installed.", "No compliance proposal was generated from the PDF.")
	}
	if runner == nil {
		return failedExtraction(pdfExtractionMethod, errors.New("PDF command runner is unavailable"))
	}

	extractionContext, cancel := context.WithTimeout(ctx, policy.PDFExtractionTimeout)
	defer cancel()

	directory, err := os.MkdirTemp("", "clearsight-pdf-")
	if err != nil {
		return failedExtraction(pdfExtractionMethod, fmt.Errorf("create private PDF workspace: %w", err))
	}
	defer os.RemoveAll(directory)
	fileName := filepath.Join(directory, "source.pdf")
	if err := os.WriteFile(fileName, data, 0o600); err != nil {
		return failedExtraction(pdfExtractionMethod, fmt.Errorf("write private PDF source: %w", err))
	}

	info := runner.Run(extractionContext, tools.Info, []string{fileName}, pdfInfoOutputLimit, pdfDiagnosticLimit)
	if err := pdfResponseError(extractionContext, "inspect PDF", info); err != nil {
		return failedExtraction(pdfExtractionMethod, err)
	}
	pages, err := parsePDFPageCount(info.Stdout)
	if err != nil {
		return failedExtraction(pdfExtractionMethod, err)
	}
	if pages > policy.MaxPDFPages {
		return failedExtraction(pdfExtractionMethod, limitError("PDF page count exceeds %d", policy.MaxPDFPages))
	}

	outputLimit := pdfTextOutputLimit(policy)
	converted := runner.Run(extractionContext, tools.Text, []string{"-layout", "-enc", "UTF-8", "-eol", "unix", fileName, "-"}, outputLimit, pdfDiagnosticLimit)
	if err := pdfResponseError(extractionContext, "extract PDF text", converted); err != nil {
		return failedExtraction(pdfExtractionMethod, err)
	}

	for index, pageText := range strings.Split(string(converted.Stdout), "\f") {
		text := normalizeText(pageText)
		collector.add(Section{Title: fmt.Sprintf("Page %d", index+1), Text: text, Page: index + 1}, text != "", false)
	}
	if collector.total == 0 {
		return collector.result(ExtractionUnsupported, pdfExtractionMethod, "The PDF contains no machine-readable text. OCR is required for this scanned or image-only document.", "No compliance proposal was generated from the PDF.")
	}
	return collector.result(ExtractionExtracted, pdfExtractionMethod)
}

func parsePDFPageCount(output []byte) (int, error) {
	for _, line := range strings.Split(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToLower(trimmed), "pages:") {
			continue
		}
		value := strings.TrimSpace(trimmed[len("pages:"):])
		pages, err := strconv.Atoi(value)
		if err != nil || pages < 1 {
			return 0, fmt.Errorf("PDF page count is invalid")
		}
		return pages, nil
	}
	return 0, fmt.Errorf("PDF page count is missing")
}

func pdfResponseError(ctx context.Context, operation string, response pdfCommandResponse) error {
	if response.Err == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if errors.Is(response.Err, ErrResourceLimit) {
		return limitError("%s output exceeds its configured byte budget", operation)
	}
	diagnostic := strings.TrimSpace(string(response.Stderr))
	if diagnostic == "" {
		return fmt.Errorf("%s: %w", operation, response.Err)
	}
	return fmt.Errorf("%s: %s", operation, diagnostic)
}

func pdfTextOutputLimit(policy ExtractionPolicy) int64 {
	maximum := policy.MaxExtractedTextBytes
	allowance := maximum / 4
	if allowance > 2<<20 {
		allowance = 2 << 20
	}
	pageBreaks := int64(policy.MaxPDFPages) * 2
	if maximum > int64(^uint64(0)>>1)-allowance-pageBreaks {
		return int64(^uint64(0) >> 1)
	}
	return maximum + allowance + pageBreaks
}

var _ io.Writer = (*boundedCommandBuffer)(nil)
