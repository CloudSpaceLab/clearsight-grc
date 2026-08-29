package documentimport

import (
	"fmt"
	"time"
)

// ExtractionPolicy bounds both decompression and semantic materialization. Hard
// structural limits fail extraction; retained-text/section limits truncate the
// review projection while preserving the original artifact for reconstruction.
type ExtractionPolicy struct {
	MaxArchiveEntries     int
	MaxExpandedBytes      int64
	MaxCompressionRatio   float64
	CompressionRatioFloor int64
	MaxSheets             int
	MaxRows               int
	MaxColumns            int
	MaxCells              int
	MaxCellBytes          int
	MaxSharedStrings      int
	MaxSharedStringBytes  int64
	MaxExtractedTextBytes int64
	MaxSections           int
	MaxProposals          int
	MaxRowErrors          int
	MaxPDFPages           int
	PDFExtractionTimeout  time.Duration
}

func DefaultExtractionPolicy() ExtractionPolicy {
	return ExtractionPolicy{
		MaxArchiveEntries:     2048,
		MaxExpandedBytes:      64 << 20,
		MaxCompressionRatio:   200,
		CompressionRatioFloor: 1 << 20,
		MaxSheets:             64,
		MaxRows:               100_000,
		MaxColumns:            256,
		MaxCells:              1_000_000,
		MaxCellBytes:          256 << 10,
		MaxSharedStrings:      200_000,
		MaxSharedStringBytes:  16 << 20,
		MaxExtractedTextBytes: 8 << 20,
		MaxSections:           5000,
		MaxProposals:          500,
		MaxRowErrors:          50,
		MaxPDFPages:           500,
		PDFExtractionTimeout:  30 * time.Second,
	}
}

func (p ExtractionPolicy) normalized() ExtractionPolicy {
	defaults := DefaultExtractionPolicy()
	if p.MaxArchiveEntries <= 0 {
		p.MaxArchiveEntries = defaults.MaxArchiveEntries
	}
	if p.MaxExpandedBytes <= 0 {
		p.MaxExpandedBytes = defaults.MaxExpandedBytes
	}
	if p.MaxCompressionRatio <= 0 {
		p.MaxCompressionRatio = defaults.MaxCompressionRatio
	}
	if p.CompressionRatioFloor <= 0 {
		p.CompressionRatioFloor = defaults.CompressionRatioFloor
	}
	if p.MaxSheets <= 0 {
		p.MaxSheets = defaults.MaxSheets
	}
	if p.MaxRows <= 0 {
		p.MaxRows = defaults.MaxRows
	}
	if p.MaxColumns <= 0 {
		p.MaxColumns = defaults.MaxColumns
	}
	if p.MaxCells <= 0 {
		p.MaxCells = defaults.MaxCells
	}
	if p.MaxCellBytes <= 0 {
		p.MaxCellBytes = defaults.MaxCellBytes
	}
	if p.MaxSharedStrings <= 0 {
		p.MaxSharedStrings = defaults.MaxSharedStrings
	}
	if p.MaxSharedStringBytes <= 0 {
		p.MaxSharedStringBytes = defaults.MaxSharedStringBytes
	}
	if p.MaxExtractedTextBytes <= 0 {
		p.MaxExtractedTextBytes = defaults.MaxExtractedTextBytes
	}
	if p.MaxSections <= 0 {
		p.MaxSections = defaults.MaxSections
	}
	if p.MaxProposals <= 0 {
		p.MaxProposals = defaults.MaxProposals
	}
	if p.MaxRowErrors <= 0 {
		p.MaxRowErrors = defaults.MaxRowErrors
	}
	if p.MaxPDFPages <= 0 {
		p.MaxPDFPages = defaults.MaxPDFPages
	}
	if p.PDFExtractionTimeout <= 0 {
		p.PDFExtractionTimeout = defaults.PDFExtractionTimeout
	}
	return p
}

type resourceLimitError struct{ message string }

func (e resourceLimitError) Error() string { return e.message }
func (e resourceLimitError) Unwrap() error { return ErrResourceLimit }

func limitError(format string, args ...any) error {
	return resourceLimitError{message: fmt.Sprintf(format, args...)}
}

func explicitExtractionStatus(base ExtractionStatus, truncated bool, degradations []Degradation) ExtractionStatus {
	if base != ExtractionExtracted {
		return base
	}
	if truncated {
		return ExtractionTruncated
	}
	for _, degradation := range degradations {
		if degradation.Recoverable {
			return ExtractionPartial
		}
	}
	return ExtractionExtracted
}

// hasUsableContent governs deterministic proposal analysis, not whether an
// operator may review the recovered structure. PARTIAL means a known source
// element class was not preserved; treating its text as a complete source for
// proposal generation would silently erase that limitation. TRUNCATED retains
// the existing bounded-analysis behavior with an explicit incompleteness flag.
func (status ExtractionStatus) hasUsableContent() bool {
	switch status {
	case ExtractionExtracted, ExtractionTruncated:
		return true
	default:
		return false
	}
}
