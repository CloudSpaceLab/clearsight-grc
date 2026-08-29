package documentimport

import (
	"context"
	"path/filepath"
	"strings"
)

const legacyXLSAdapterVersion = "LIBREOFFICE_XLS_CONVERTER_V1"

func ExtractAdvanced(ctx context.Context, document Document, data []byte, extraction ExtractionPolicy, adapter ParserAdapter, adapterPolicy ParserAdapterPolicy, converter *LegacyOfficeConverter) ExtractionResult {
	if strings.EqualFold(filepath.Ext(document.FileName), ".xls") && converter != nil && converter.Enabled() {
		converted, err := converter.ConvertXLS(ctx, document.FileName, data)
		if err != nil {
			message := "Configured legacy Office conversion failed; the original artifact remains available for governed review."
			return ExtractionResult{
				Status: ExtractionFailed, Method: "LEGACY_OFFICE_CONVERSION", ParserVersion: "LEGACY_OFFICE_CONVERSION",
				AdapterVersion: legacyXLSAdapterVersion, Limitations: []string{message}, Sections: []Section{}, Elements: []ExtractedElement{},
				Degradations: []Degradation{{Code: "LEGACY_OFFICE_CONVERSION_FAILED", Message: message, Recoverable: true}},
			}
		}
		convertedName := strings.TrimSuffix(document.FileName, filepath.Ext(document.FileName)) + ".xlsx"
		result := ExtractWithPolicy(ctx, convertedName, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", converted, extraction)
		result.Method = "LEGACY_OFFICE_CONVERSION+" + result.Method
		result.AdapterVersion = legacyXLSAdapterVersion
		result.Limitations = append([]string{"Legacy .xls content was converted in an isolated bounded worker before normal XLSX extraction."}, result.Limitations...)
		return result
	}

	base := ExtractWithPolicy(ctx, document.FileName, document.MediaType, data, extraction)
	return ApplyParserAdapter(ctx, document.ID, document.FileName, document.MediaType, data, base, extraction, adapter, adapterPolicy)
}
